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
	typeToID map[string]string     // Canonical type string → node ID
	idToType map[string]types.Type // Node ID → type
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

// dependencyExtractor extracts declared and used dependencies from nodes
type dependencyExtractor struct {
	mapper *typeIDMapper
	prog   *ssa.Program
	fset   *token.FileSet
}

// newDependencyExtractor creates a new dependency extractor
func newDependencyExtractor(mapper *typeIDMapper, prog *ssa.Program, fset *token.FileSet) *dependencyExtractor {
	return &dependencyExtractor{
		mapper: mapper,
		prog:   prog,
		fset:   fset,
	}
}

// ExtractDeclared extracts declared dependencies from a node's DependsOn field
func (e *dependencyExtractor) ExtractDeclared(node NodeDefinition) ([]string, error) {
	if node.DependsOn == nil {
		// No dependencies declared
		return []string{}, nil
	}

	// DependsOn is []graft.ID
	// In SSA, this is typically a slice literal or a reference to one
	// We need to trace it to find the actual ID values

	ids, err := e.extractIDsFromValue(node.DependsOn)
	if err != nil {
		// If we can't extract, return empty list
		return []string{}, nil
	}

	return ids, nil
}

// extractIDsFromValue extracts ID strings from an SSA value representing []graft.ID
func (e *dependencyExtractor) extractIDsFromValue(v ssa.Value) ([]string, error) {
	// Check if this is an Alloc (local variable)
	if alloc, ok := v.(*ssa.Alloc); ok {
		// Find stores to this alloc
		return e.extractIDsFromAlloc(alloc)
	}

	// Check if this is a slice literal
	if slice, ok := v.(*ssa.Slice); ok {
		return e.extractIDsFromSlice(slice)
	}

	// Check if this is a MakeSlice
	if makeSlice, ok := v.(*ssa.MakeSlice); ok {
		return e.extractIDsFromMakeSlice(makeSlice)
	}

	// For now, return empty if we can't handle it
	return []string{}, fmt.Errorf("cannot extract IDs from %T", v)
}

// extractIDsFromAlloc extracts IDs from an allocated slice
func (e *dependencyExtractor) extractIDsFromAlloc(alloc *ssa.Alloc) ([]string, error) {
	var ids []string

	if alloc.Referrers() == nil {
		return ids, nil
	}

	// Look for IndexAddr instructions (accessing slice elements)
	for _, instr := range *alloc.Referrers() {
		if indexAddr, ok := instr.(*ssa.IndexAddr); ok {
			// Find stores to this index
			if indexAddr.Referrers() != nil {
				for _, store := range *indexAddr.Referrers() {
					if s, ok := store.(*ssa.Store); ok {
						// Extract the ID from the stored value
						if id, err := e.extractIDFromValue(s.Val); err == nil {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}

	return ids, nil
}

// extractIDsFromSlice extracts IDs from a slice operation
func (e *dependencyExtractor) extractIDsFromSlice(slice *ssa.Slice) ([]string, error) {
	// Extract from the underlying array
	return e.extractIDsFromValue(slice.X)
}

// extractIDsFromMakeSlice extracts IDs from a MakeSlice
func (e *dependencyExtractor) extractIDsFromMakeSlice(makeSlice *ssa.MakeSlice) ([]string, error) {
	var ids []string

	// Look for stores to the slice
	if makeSlice.Referrers() == nil {
		return ids, nil
	}

	for _, instr := range *makeSlice.Referrers() {
		if indexAddr, ok := instr.(*ssa.IndexAddr); ok {
			if indexAddr.Referrers() != nil {
				for _, store := range *indexAddr.Referrers() {
					if s, ok := store.(*ssa.Store); ok {
						if id, err := e.extractIDFromValue(s.Val); err == nil {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}

	return ids, nil
}

// extractIDFromValue extracts a single ID from an SSA value
func (e *dependencyExtractor) extractIDFromValue(v ssa.Value) (string, error) {
	// Handle constants (string literals)
	if c, ok := v.(*ssa.Const); ok {
		if c.Value != nil {
			return constant.StringVal(c.Value), nil
		}
	}

	// Handle UnOp (dereferencing, address-of, etc.)
	if unop, ok := v.(*ssa.UnOp); ok {
		return e.extractIDFromValue(unop.X)
	}

	// Handle ChangeType
	if ct, ok := v.(*ssa.ChangeType); ok {
		return e.extractIDFromValue(ct.X)
	}

	// Handle Convert
	if conv, ok := v.(*ssa.Convert); ok {
		return e.extractIDFromValue(conv.X)
	}

	return "", fmt.Errorf("cannot extract ID from %T", v)
}

// ExtractUsed extracts used dependencies from a node's Run function
func (e *dependencyExtractor) ExtractUsed(node NodeDefinition) ([]string, error) {
	if node.RunFunc == nil {
		// No Run function - no dependencies can be used
		return []string{}, nil
	}

	var ids []string
	seen := make(map[string]bool)

	// Walk all instructions in the Run function
	for _, block := range node.RunFunc.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				if isGraftDepCall(call) {
					// Extract the type parameter
					depType, err := e.extractDepTypeParameter(call)
					if err != nil {
						continue
					}

					// Resolve type to ID
					id, err := e.mapper.ResolveType(depType)
					if err != nil {
						// Type not in mapping - skip this dependency
						continue
					}

					if !seen[id] {
						ids = append(ids, id)
						seen[id] = true
					}
				}
			}
		}
	}

	return ids, nil
}

// extractDepTypeParameter extracts the type parameter from a Dep[T]() call
func (e *dependencyExtractor) extractDepTypeParameter(call *ssa.Call) (types.Type, error) {
	callee := call.Common().StaticCallee()
	if callee == nil {
		return nil, fmt.Errorf("no static callee")
	}

	// Get the signature
	sig := callee.Signature
	if sig == nil {
		return nil, fmt.Errorf("no signature")
	}

	// For Dep[T], the return type is (T, error)
	// Extract T from the first return value
	results := sig.Results()
	if results.Len() < 2 {
		return nil, fmt.Errorf("unexpected result count: %d", results.Len())
	}

	// First result is T
	return results.At(0).Type(), nil
}

// AnalyzeNode analyzes a single node and produces an AnalysisResult
func (e *dependencyExtractor) AnalyzeNode(node NodeDefinition) (AnalysisResult, error) {
	result := AnalysisResult{
		NodeID: node.ID,
		File:   node.File,
	}

	// Extract declared dependencies
	declared, err := e.ExtractDeclared(node)
	if err != nil {
		// Log but continue
		declared = []string{}
	}
	result.DeclaredDeps = declared

	// Extract used dependencies
	used, err := e.ExtractUsed(node)
	if err != nil {
		// Log but continue
		used = []string{}
	}
	result.UsedDeps = used

	// Build sets for comparison
	declaredSet := make(map[string]bool)
	for _, d := range declared {
		declaredSet[d] = true
	}

	usedSet := make(map[string]bool)
	for _, u := range used {
		usedSet[u] = true
	}

	// Find undeclared (used but not declared)
	for u := range usedSet {
		if !declaredSet[u] {
			result.Undeclared = append(result.Undeclared, u)
		}
	}

	// Find unused (declared but not used)
	for d := range declaredSet {
		if !usedSet[d] {
			result.Unused = append(result.Unused, d)
		}
	}

	return result, nil
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

	// Phase 5: Extract dependencies and analyze
	a.debugf("Extracting and analyzing dependencies...")
	extractor := newDependencyExtractor(mapper, prog, prog.Fset)

	var results []AnalysisResult
	for _, node := range nodes {
		result, err := extractor.AnalyzeNode(node)
		if err != nil {
			// Log error but continue with other nodes
			a.debugf("Error analyzing node %q: %v", node.ID, err)
			continue
		}

		a.debugf("Analyzed node %q: declared=%v, used=%v",
			result.NodeID, result.DeclaredDeps, result.UsedDeps)

		if result.HasIssues() {
			a.debugf("  Issues: undeclared=%v, unused=%v",
				result.Undeclared, result.Unused)
		}

		results = append(results, result)
	}

	// Phase 6: Detect cycles and annotate results
	a.debugf("Detecting cycles...")
	detector := newCycleDetector(results)
	allCycles := detector.detectCycles()
	if len(allCycles) > 0 {
		a.debugf("Found %d cycles:", len(allCycles))
		for _, cycle := range allCycles {
			a.debugf("  - %v", cycle)
		}
	} else {
		a.debugf("No cycles detected")
	}

	nodeCycles := detector.mapCyclesToNodes()

	// Annotate each node's result with its cycles
	for i := range results {
		if cycles, found := nodeCycles[results[i].NodeID]; found {
			results[i].Cycles = cycles
			a.debugf("Node %q participates in %d cycle(s)", results[i].NodeID, len(cycles))
		}
	}

	a.debugf("Analysis complete: %d nodes analyzed", len(results))

	return results, nil
}

// cycleDetector discovers circular dependencies using DFS
type cycleDetector struct {
	adjList map[string][]string // nodeID → dependencies
	state   map[string]int      // DFS visit state: 0=unvisited, 1=visiting, 2=visited
	path    []string            // Current DFS path
	cycles  [][]string          // All discovered cycles
}

// newCycleDetector creates a cycle detector from analysis results
func newCycleDetector(results []AnalysisResult) *cycleDetector {
	adjList := make(map[string][]string)

	// Build adjacency list from declared dependencies
	for _, r := range results {
		adjList[r.NodeID] = r.DeclaredDeps
	}

	return &cycleDetector{
		adjList: adjList,
		state:   make(map[string]int),
		path:    make([]string, 0),
		cycles:  make([][]string, 0),
	}
}

// detectCycles finds all cycles in the dependency graph using DFS
func (d *cycleDetector) detectCycles() [][]string {
	// Run DFS from each unvisited node
	for node := range d.adjList {
		if d.state[node] == 0 {
			d.dfs(node)
		}
	}
	return d.cycles
}

// dfs performs depth-first search to detect cycles
func (d *cycleDetector) dfs(node string) {
	d.state[node] = 1 // Mark as visiting
	d.path = append(d.path, node)

	// Explore dependencies
	for _, dep := range d.adjList[node] {
		if d.state[dep] == 0 {
			// Unvisited: continue DFS
			d.dfs(dep)
		} else if d.state[dep] == 1 {
			// Back edge detected: cycle found
			cycle := d.extractCycle(dep)
			d.cycles = append(d.cycles, cycle)
		}
		// If state[dep] == 2 (visited), no cycle through this edge
	}

	// Backtrack
	d.path = d.path[:len(d.path)-1]
	d.state[node] = 2 // Mark as visited
}

// extractCycle extracts the cycle path from the current DFS path
func (d *cycleDetector) extractCycle(backEdgeTarget string) []string {
	// Find where the cycle starts in the current path
	cycleStart := -1
	for i, node := range d.path {
		if node == backEdgeTarget {
			cycleStart = i
			break
		}
	}

	if cycleStart == -1 {
		// Should not happen if algorithm is correct
		return nil
	}

	// Extract cycle path and append the back edge target to close the loop
	cycle := make([]string, 0, len(d.path)-cycleStart+1)
	cycle = append(cycle, d.path[cycleStart:]...)
	cycle = append(cycle, backEdgeTarget)

	return cycle
}

// mapCyclesToNodes maps each node to all cycles it participates in
func (d *cycleDetector) mapCyclesToNodes() map[string][][]string {
	nodeCycles := make(map[string][][]string)

	for _, cycle := range d.cycles {
		// Add this cycle to all participating nodes (except the last duplicate)
		seen := make(map[string]bool)
		for i := 0; i < len(cycle)-1; i++ {
			node := cycle[i]
			// Only add unique cycles per node
			if !seen[node] {
				nodeCycles[node] = append(nodeCycles[node], cycle)
				seen[node] = true
			}
		}
	}

	return nodeCycles
}
