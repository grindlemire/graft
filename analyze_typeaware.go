package graft

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// AnalyzerConfig configures the type-aware analyzer
type AnalyzerConfig struct {
	// BuildTags specifies build tags to use (e.g., []string{"integration"})
	BuildTags []string

	// IncludeTests includes _test.go files in analysis
	IncludeTests bool

	// WorkDir sets the working directory for package resolution
	WorkDir string

	// Debug enables detailed logging for troubleshooting
	Debug bool
}

// typeAwareAnalyzer orchestrates the entire analysis pipeline
type typeAwareAnalyzer struct {
	cfg AnalyzerConfig
}

// newTypeAwareAnalyzer creates a new type-aware analyzer with the given config
func newTypeAwareAnalyzer(cfg AnalyzerConfig) *typeAwareAnalyzer {
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}
	return &typeAwareAnalyzer{cfg: cfg}
}

// debugf prints debug output if Debug is enabled
func (a *typeAwareAnalyzer) debugf(format string, args ...interface{}) {
	if a.cfg.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

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

// packageLoader handles go/packages loading
type packageLoader struct {
	cfg *packages.Config
}

// newPackageLoader creates a new package loader
func newPackageLoader(cfg AnalyzerConfig) *packageLoader {
	pkgCfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir:   cfg.WorkDir,
		Tests: cfg.IncludeTests,
	}

	if len(cfg.BuildTags) > 0 {
		pkgCfg.BuildFlags = []string{"-tags", strings.Join(cfg.BuildTags, ",")}
	}

	return &packageLoader{cfg: pkgCfg}
}

// Load loads all packages in the given directory
func (l *packageLoader) Load(dir string) ([]*packages.Package, error) {
	// Update Dir to the target directory
	l.cfg.Dir = dir

	// Load all packages recursively
	pkgs, err := packages.Load(l.cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}

	// Check for package errors
	var errs []error
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			errs = append(errs, e)
		}
	})

	if len(errs) > 0 {
		// Return first error for simplicity
		return nil, fmt.Errorf("package errors: %v", errs[0])
	}

	return pkgs, nil
}

// filterPackages filters out packages we don't want to analyze
func (l *packageLoader) filterPackages(pkgs []*packages.Package) []*packages.Package {
	var filtered []*packages.Package

	for _, pkg := range pkgs {
		// Skip packages without Go files
		if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 {
			continue
		}

		// Skip test packages unless explicitly requested
		if !l.cfg.Tests && strings.HasSuffix(pkg.ID, ".test") {
			continue
		}

		filtered = append(filtered, pkg)
	}

	return filtered
}

// ssaBuilder builds SSA representation from loaded packages
type ssaBuilder struct {
	mode ssa.BuilderMode
	prog *ssa.Program
}

// newSSABuilder creates a new SSA builder
func newSSABuilder() *ssaBuilder {
	return &ssaBuilder{
		// InstantiateGenerics is required to analyze generic functions like Dep[T]
		mode: ssa.InstantiateGenerics | ssa.GlobalDebug,
	}
}

// Build constructs an SSA program from loaded packages
func (b *ssaBuilder) Build(pkgs []*packages.Package) (*ssa.Program, *[]*ssa.Package, error) {
	// Create SSA program
	prog, ssaPkgs := ssautil.AllPackages(pkgs, b.mode)

	// Build SSA for all packages
	prog.Build()

	b.prog = prog

	// Verify we built successfully
	if len(ssaPkgs) == 0 {
		return nil, nil, fmt.Errorf("no SSA packages built")
	}

	return prog, &ssaPkgs, nil
}

// GetPackages returns all SSA packages in the program
func (b *ssaBuilder) GetPackages() []*ssa.Package {
	if b.prog == nil {
		return nil
	}

	var pkgs []*ssa.Package
	for _, pkg := range b.prog.AllPackages() {
		pkgs = append(pkgs, pkg)
	}

	return pkgs
}

// findInitFunctions finds all init() functions in an SSA package
func findInitFunctions(pkg *ssa.Package) []*ssa.Function {
	var inits []*ssa.Function

	// Look for init functions in package members
	for _, member := range pkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			if fn.Name() == "init" {
				inits = append(inits, fn)
			}
		}
	}

	return inits
}

// isGraftPackage checks if an SSA package is the graft package
func isGraftPackage(pkg *ssa.Package) bool {
	if pkg == nil || pkg.Pkg == nil {
		return false
	}
	return pkg.Pkg.Path() == "github.com/grindlemire/graft"
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

// isGraftDepCall checks if a call instruction calls graft.Dep
func isGraftDepCall(call *ssa.Call) bool {
	callee := call.Common().StaticCallee()
	if callee == nil {
		return false
	}

	pkg := callee.Pkg
	if pkg == nil || pkg.Pkg == nil {
		return false
	}

	return pkg.Pkg.Path() == "github.com/grindlemire/graft" &&
		callee.Name() == "Dep"
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
		// Skip the graft package itself
		// if isGraftPackage(pkg) {
		// 	continue
		// }

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

// typeIDMapper builds and maintains a bidirectional mapping between
// types.Type and node IDs
type typeIDMapper struct {
	typeToID map[string]string      // Canonical type string → node ID
	idToType map[string]types.Type  // Node ID → type
}

// newTypeIDMapper creates a new type-to-ID mapper
func newTypeIDMapper() *typeIDMapper {
	return &typeIDMapper{
		typeToID: make(map[string]string),
		idToType: make(map[string]types.Type),
	}
}

// normalizeType converts a type to its canonical string representation
// This handles type aliases, named types, pointers, etc.
func (m *typeIDMapper) normalizeType(t types.Type) string {
	// Use types.TypeString with full package paths for canonical representation
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Path()
	})
}

// typeKey returns a unique key for a type
func (m *typeIDMapper) typeKey(t types.Type) string {
	return m.normalizeType(t)
}

// BuildMapping constructs the type-to-ID mapping from node definitions
func (m *typeIDMapper) BuildMapping(nodes []NodeDefinition) error {
	for _, node := range nodes {
		if node.OutputType == nil {
			return fmt.Errorf("node %q has nil output type", node.ID)
		}

		key := m.typeKey(node.OutputType)

		// Check for conflicts: same type registered by multiple nodes
		if existingID, exists := m.typeToID[key]; exists {
			if existingID != node.ID {
				return fmt.Errorf(
					"type conflict: type %s is registered by both node %q and node %q",
					key, existingID, node.ID,
				)
			}
			// Same node ID - OK, might be duplicate discovery
			continue
		}

		// Add mapping
		m.typeToID[key] = node.ID
		m.idToType[node.ID] = node.OutputType
	}

	return nil
}

// ResolveType resolves a type to its node ID
func (m *typeIDMapper) ResolveType(t types.Type) (string, error) {
	key := m.typeKey(t)

	id, ok := m.typeToID[key]
	if !ok {
		return "", fmt.Errorf("type %s is not registered", key)
	}

	return id, nil
}

// GetType returns the output type for a given node ID
func (m *typeIDMapper) GetType(nodeID string) (types.Type, error) {
	t, ok := m.idToType[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %q not found in type mapping", nodeID)
	}

	return t, nil
}

// HasType checks if a type is registered
func (m *typeIDMapper) HasType(t types.Type) bool {
	key := m.typeKey(t)
	_, ok := m.typeToID[key]
	return ok
}

// Size returns the number of types in the mapping
func (m *typeIDMapper) Size() int {
	return len(m.typeToID)
}

// Analyze performs type-aware dependency analysis on a directory
func (a *typeAwareAnalyzer) Analyze(dir string) ([]AnalysisResult, error) {
	a.debugf("Starting type-aware analysis of %s", dir)

	// Phase 1: Load packages
	a.debugf("Loading packages...")
	loader := newPackageLoader(a.cfg)
	pkgs, err := loader.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	a.debugf("Loaded %d packages", len(pkgs))

	// Phase 2: Build SSA
	a.debugf("Building SSA program...")
	builder := newSSABuilder()
	prog, srcPkgs, err := builder.Build(pkgs)
	if err != nil {
		return nil, fmt.Errorf("building SSA: %w", err)
	}

	ssaPkgs := builder.GetPackages()
	a.debugf("Built SSA for %d packages", len(ssaPkgs))

	// Phase 3: Discover nodes
	a.debugf("Discovering nodes...")
	discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
	nodes, err := discoverer.FindNodes()
	if err != nil {
		return nil, fmt.Errorf("discovering nodes: %w", err)
	}
	a.debugf("Discovered %d nodes", len(nodes))

	for _, node := range nodes {
		a.debugf("  - %s", node.String())
	}

	// Phase 4: Build type-to-ID mapping
	a.debugf("Building type-to-ID mapping...")
	mapper := newTypeIDMapper()
	if err := mapper.BuildMapping(nodes); err != nil {
		return nil, fmt.Errorf("building type mapping: %w", err)
	}
	a.debugf("Built mapping for %d types", mapper.Size())

	// Log the mapping if debug is enabled
	for _, node := range nodes {
		typeKey := mapper.typeKey(node.OutputType)
		a.debugf("  %s → %q", typeKey, node.ID)
	}

	// TODO: Phase 5 - Dependency extraction (next phase)

	// Placeholder return
	return []AnalysisResult{}, nil
}
