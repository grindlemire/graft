package typeaware

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"

	"golang.org/x/tools/go/ssa"
)

// NodeDefinition represents a discovered node registration
type NodeDefinition struct {
	ID         string         // Node ID (e.g., "db")
	File       string         // Source file path
	OutputType types.Type     // The T in Node[T]
	DependsOn  ssa.Value      // The DependsOn field value (for dataflow analysis)
	RunFunc    *ssa.Function  // The Run function body
	Position   token.Position // Source location for error reporting
}

// String returns a human-readable summary of the node definition
func (n NodeDefinition) String() string {
	return fmt.Sprintf("Node{ID:%q, Type:%v, File:%s:%d}",
		n.ID,
		n.OutputType,
		filepath.Base(n.File),
		n.Position.Line,
	)
}

var nameRegex = regexp.MustCompile(`^Register\[.*\]$`)

// isGraftRegisterCall checks if a call instruction calls graft.Register
func isGraftRegisterCall(call *ssa.Call) bool {
	callee := call.Common().StaticCallee()
	if callee == nil {
		return false
	}

	if callee.Origin() == nil {
		return false
	}

	return callee.Origin().String() == "github.com/grindlemire/graft.Register" &&
		nameRegex.MatchString(callee.Name())
}

var depNameRegex = regexp.MustCompile(`^Dep\[.*\]$`)

// isGraftDepCall checks if a call instruction calls graft.Dep
func isGraftDepCall(call *ssa.Call) bool {
	callee := call.Common().StaticCallee()
	if callee == nil {
		return false
	}

	if callee.Origin() == nil {
		return false
	}

	// For generic functions, check the Origin
	return callee.Origin().String() == "github.com/grindlemire/graft.Dep" &&
		depNameRegex.MatchString(callee.Name())
}

// nodeDiscoverer finds graft.Register() calls in SSA and extracts NodeDefinitions
type nodeDiscoverer struct {
	prog    *ssa.Program
	fset    *token.FileSet
	srcPkgs *[]*ssa.Package // Source packages we're analyzing
}

// newNodeDiscoverer creates a new node discoverer
func newNodeDiscoverer(prog *ssa.Program, fset *token.FileSet, srcPkgs *[]*ssa.Package) *nodeDiscoverer {
	return &nodeDiscoverer{
		prog:    prog,
		fset:    fset,
		srcPkgs: srcPkgs,
	}
}

// FindNodes finds all graft.Register() calls and extracts node definitions
func (d *nodeDiscoverer) FindNodes() ([]NodeDefinition, error) {
	var nodes []NodeDefinition

	// Walk only the source packages we're analyzing
	// This is more targeted than walking all packages in the program
	for _, pkg := range *d.srcPkgs {
		// Look in all functions (not just init)
		// Register calls can be in init, package-level functions, or methods
		for _, member := range pkg.Members {
			fn, ok := member.(*ssa.Function)
			if !ok {
				continue
			}

			// Walk instructions in function looking for Register calls
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					if call, ok := instr.(*ssa.Call); ok {
						if isGraftRegisterCall(call) {
							node, err := d.extractNodeDefinition(call)
							if err != nil {
								// Log warning but continue analyzing other nodes
								continue
							}
							nodes = append(nodes, node)
						}
					}
				}
			}
		}
	}

	return nodes, nil
}

// extractNodeDefinition extracts a NodeDefinition from a graft.Register() call
func (d *nodeDiscoverer) extractNodeDefinition(call *ssa.Call) (NodeDefinition, error) {
	// Register takes one argument: Node[T]{...}
	if len(call.Common().Args) == 0 {
		return NodeDefinition{}, fmt.Errorf("Register call has no arguments")
	}

	nodeArg := call.Common().Args[0]

	// Get source position
	pos := d.fset.Position(call.Pos())

	// The argument is typically MakeInterface wrapping the Node struct
	// Extract the underlying value
	nodeValue := nodeArg
	if mi, ok := nodeArg.(*ssa.MakeInterface); ok {
		nodeValue = mi.X
	}

	// Extract the type - should be Node[T] where T is the output type
	nodeType := nodeValue.Type()

	// Get the output type (T in Node[T])
	outputType, err := d.extractOutputType(nodeType)
	if err != nil {
		return NodeDefinition{}, fmt.Errorf("extracting output type: %w", err)
	}

	// Try to extract node fields
	// This is complex because the Node might be:
	// 1. Inline literal: Register(Node[T]{ID: "x", ...})
	// 2. Variable: n := Node[T]{...}; Register(n)
	// 3. Returned from function: Register(makeNode())

	nodeDef := NodeDefinition{
		OutputType: outputType,
		Position:   pos,
		File:       pos.Filename,
	}

	// Try to extract ID, DependsOn, and Run from the node value
	// We need to look at the instructions in the function to find field assignments
	fn := call.Parent()
	_ = d.extractNodeFieldsFromFunction(fn, nodeValue, &nodeDef)
	// Don't fail if field extraction doesn't work - we still have the type

	// For now, if we can't extract ID, try to use the package name as a fallback
	if nodeDef.ID == "" {
		// Use "unknown" as placeholder - we'll improve extraction later
		nodeDef.ID = fmt.Sprintf("unknown_%s", outputType.String())
	}

	return nodeDef, nil
}

// extractOutputType extracts T from Node[T] type
func (d *nodeDiscoverer) extractOutputType(t types.Type) (types.Type, error) {
	// Handle pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Should be a named type: graft.Node[T]
	named, ok := t.(*types.Named)
	if !ok {
		return nil, fmt.Errorf("node type is not named: %v", t)
	}

	// Get type arguments (T in Node[T])
	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return nil, fmt.Errorf("node type has no type arguments")
	}

	// Return the first type argument (T)
	return typeArgs.At(0), nil
}

// extractNodeFieldsFromFunction walks all instructions in a function to find field assignments
func (d *nodeDiscoverer) extractNodeFieldsFromFunction(fn *ssa.Function, nodeValue ssa.Value, nodeDef *NodeDefinition) error {
	if fn == nil {
		return fmt.Errorf("nil function")
	}

	// Trace back from the nodeValue to find what it references
	// If it's a UnOp (address-of), get the operand
	var baseValue ssa.Value = nodeValue
	if unop, ok := nodeValue.(*ssa.UnOp); ok {
		baseValue = unop.X
	}

	// Walk all instructions in all blocks to find field assignments
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Look for FieldAddr instructions that reference our base value
			if fa, ok := instr.(*ssa.FieldAddr); ok {
				// Check if this FieldAddr operates on our value
				if fa.X == baseValue {
					fieldName := d.getFieldName(fa)

					// Look for Store that writes to this field
					_ = d.extractFieldValue(fieldName, fa, nodeDef)
					// Continue even if extraction fails
				}
			}
		}
	}

	return nil
}

// getFieldName gets the name of a field from a FieldAddr instruction
func (d *nodeDiscoverer) getFieldName(fa *ssa.FieldAddr) string {
	// Get the struct type
	structType, ok := fa.X.Type().Underlying().(*types.Struct)
	if !ok {
		if ptr, ok := fa.X.Type().Underlying().(*types.Pointer); ok {
			structType, _ = ptr.Elem().Underlying().(*types.Struct)
		}
	}

	if structType == nil {
		return ""
	}

	// Get the field by index
	if fa.Field >= 0 && fa.Field < structType.NumFields() {
		return structType.Field(fa.Field).Name()
	}

	return ""
}

// extractFieldValue extracts the value assigned to a field
func (d *nodeDiscoverer) extractFieldValue(fieldName string, fa *ssa.FieldAddr, nodeDef *NodeDefinition) error {
	// Look for Store instructions that write to this field
	if fa.Referrers() == nil {
		return fmt.Errorf("field has no referrers")
	}

	for _, instr := range *fa.Referrers() {
		if store, ok := instr.(*ssa.Store); ok {
			switch fieldName {
			case "ID":
				// Extract ID - should be a string constant or graft.ID value
				if c, ok := store.Val.(*ssa.Const); ok {
					nodeDef.ID = constant.StringVal(c.Value)
				}
				// Could also be a variable - trace it if needed

			case "DependsOn":
				// Store the SSA value for later analysis
				nodeDef.DependsOn = store.Val

			case "Run":
				// Extract the Run function
				if fn, ok := store.Val.(*ssa.Function); ok {
					nodeDef.RunFunc = fn
				} else if mi, ok := store.Val.(*ssa.MakeInterface); ok {
					// Might be wrapped in MakeInterface
					if fn, ok := mi.X.(*ssa.Function); ok {
						nodeDef.RunFunc = fn
					}
				}
			}
		}
	}

	return nil
}
