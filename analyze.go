package graft

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// AnalysisResult contains the result of analyzing a node's dependency usage.
//
// It captures both declared dependencies (in DependsOn) and used dependencies
// (via Dep[T] calls), allowing detection of mismatches.
type AnalysisResult struct {
	// NodeID is the ID field value from the analyzed node.
	NodeID string

	// File is the path to the source file containing the node.
	File string

	// DeclaredDeps are the dependency IDs listed in the DependsOn field.
	DeclaredDeps []string

	// UsedDeps are the dependency IDs accessed via Dep[T] calls in Run.
	UsedDeps []string

	// Undeclared are dependencies used but not declared in DependsOn.
	// These will cause runtime errors.
	Undeclared []string

	// Unused are dependencies declared but never used.
	// These indicate dead code or missing implementation.
	Unused []string
}

// HasIssues returns true if there are undeclared or unused dependencies.
func (r AnalysisResult) HasIssues() bool {
	return len(r.Undeclared) > 0 || len(r.Unused) > 0
}

// String returns a human-readable summary of issues.
//
// Returns "NodeID: OK" if there are no issues, otherwise returns
// a summary of undeclared and unused dependencies.
func (r AnalysisResult) String() string {
	if !r.HasIssues() {
		return fmt.Sprintf("%s: OK", r.NodeID)
	}

	var parts []string
	if len(r.Undeclared) > 0 {
		parts = append(parts, fmt.Sprintf("undeclared deps: %v", r.Undeclared))
	}
	if len(r.Unused) > 0 {
		parts = append(parts, fmt.Sprintf("unused deps: %v", r.Unused))
	}
	return fmt.Sprintf("%s (%s): %s", r.NodeID, r.File, strings.Join(parts, "; "))
}

// AnalyzeDirDebug controls whether AnalyzeDir prints debug information.
var AnalyzeDirDebug = false

// Analyzer performs type-aware dependency analysis using go/packages.
type Analyzer struct {
	pkgs     []*packages.Package
	fset     *token.FileSet
	registry *nodeRegistry
	ssaProg  *ssa.Program
}

// nodeRegistry maps output types to node definitions.
type nodeRegistry struct {
	// byType maps types.Type to node definition (using type identity)
	byType map[string]*nodeDef // canonical type string -> node

	// byID maps node ID to definition for quick lookup
	byID map[string]*nodeDef
}

func newNodeRegistry() *nodeRegistry {
	return &nodeRegistry{
		byType: make(map[string]*nodeDef),
		byID:   make(map[string]*nodeDef),
	}
}

func (r *nodeRegistry) register(def *nodeDef) {
	typeKey := types.TypeString(def.outputType, nil)
	r.byType[typeKey] = def
	r.byID[def.id] = def
}

func (r *nodeRegistry) lookupByType(t types.Type) *nodeDef {
	typeKey := types.TypeString(t, nil)
	return r.byType[typeKey]
}

// nodeDef holds information about a graft.Node[T] definition.
type nodeDef struct {
	id           string
	outputType   types.Type
	file         string
	position     token.Position
	declaredDeps []string
	runFunc      ast.Expr // The Run field value (FuncLit or Ident)
	pkg          *packages.Package
}

// AnalyzeDir analyzes all Go files in a directory for dependency correctness.
//
// It uses go/packages for full type resolution, enabling accurate detection of:
//   - Dependencies declared but not used
//   - Dependencies used but not declared
//   - Cross-package type resolution
//   - Import aliases
//   - Type aliases
//
// Returns all nodes found with their analysis results. Use [AnalysisResult.HasIssues]
// to filter for problems.
func AnalyzeDir(dir string) ([]AnalysisResult, error) {
	a := &Analyzer{
		registry: newNodeRegistry(),
	}

	if err := a.loadPackages(dir); err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	if err := a.buildNodeRegistry(); err != nil {
		return nil, fmt.Errorf("building node registry: %w", err)
	}

	if err := a.buildSSA(); err != nil {
		return nil, fmt.Errorf("building SSA: %w", err)
	}

	return a.analyzeAllNodes()
}

// loadPackages loads all packages in the directory with full type information.
func (a *Analyzer) loadPackages(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:        absDir,
		Tests:      false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}

	// Check for errors in loaded packages
	var errs []string
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("package errors: %s", strings.Join(errs, "; "))
	}

	a.pkgs = pkgs
	if len(pkgs) > 0 {
		a.fset = pkgs[0].Fset
	}

	if AnalyzeDirDebug {
		fmt.Printf("[ANALYZE DEBUG] Loaded %d packages\n", len(pkgs))
		for _, pkg := range pkgs {
			fmt.Printf("[ANALYZE DEBUG]   %s (%d files)\n", pkg.PkgPath, len(pkg.Syntax))
		}
	}

	return nil
}

// buildNodeRegistry scans all packages for Node[T] definitions.
func (a *Analyzer) buildNodeRegistry() error {
	for _, pkg := range a.pkgs {
		for i, file := range pkg.Syntax {
			filePath := pkg.CompiledGoFiles[i]

			ast.Inspect(file, func(n ast.Node) bool {
				comp, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}

				def := a.extractNodeDef(comp, pkg, filePath)
				if def != nil {
					a.registry.register(def)
					if AnalyzeDirDebug {
						fmt.Printf("[ANALYZE DEBUG] Registered node %q (type: %s) from %s\n",
							def.id, types.TypeString(def.outputType, nil), def.file)
					}
				}
				return true
			})
		}
	}

	if AnalyzeDirDebug {
		fmt.Printf("[ANALYZE DEBUG] Registry contains %d nodes\n", len(a.registry.byID))
	}

	return nil
}

// extractNodeDef extracts a nodeDef from a Node[T] composite literal.
func (a *Analyzer) extractNodeDef(comp *ast.CompositeLit, pkg *packages.Package, file string) *nodeDef {
	// Check if this is a Node[T] type
	outputType := a.extractNodeOutputType(comp, pkg)
	if outputType == nil {
		return nil
	}

	def := &nodeDef{
		outputType: outputType,
		file:       file,
		position:   pkg.Fset.Position(comp.Pos()),
		pkg:        pkg,
	}

	// Extract fields from the composite literal
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "ID":
			def.id = a.extractStringValue(kv.Value, pkg)
		case "DependsOn":
			def.declaredDeps = a.extractDependsOn(kv.Value, pkg)
		case "Run":
			def.runFunc = kv.Value
		}
	}

	if def.id == "" {
		return nil
	}

	return def
}

// extractNodeOutputType returns the output type T from a Node[T] composite literal.
func (a *Analyzer) extractNodeOutputType(comp *ast.CompositeLit, pkg *packages.Package) types.Type {
	// Get the type of the composite literal
	tv, ok := pkg.TypesInfo.Types[comp]
	if !ok {
		return nil
	}

	// Check if it's a Node type
	named, ok := tv.Type.(*types.Named)
	if !ok {
		return nil
	}

	// Check the type name
	obj := named.Obj()
	if obj.Name() != "Node" {
		return nil
	}

	// Check it's from the graft package
	pkgPath := obj.Pkg().Path()
	if !strings.HasSuffix(pkgPath, "graft") && pkgPath != "github.com/grindlemire/graft" {
		// Also allow local package named "graft"
		if obj.Pkg().Name() != "graft" {
			return nil
		}
	}

	// Extract type argument
	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return nil
	}

	return typeArgs.At(0)
}

// extractStringValue extracts a string value from an AST expression.
func (a *Analyzer) extractStringValue(expr ast.Expr, pkg *packages.Package) string {
	// Try to get constant value from types.Info
	if tv, ok := pkg.TypesInfo.Types[expr]; ok && tv.Value != nil {
		// It's a constant - extract string value
		return strings.Trim(tv.Value.ExactString(), `"`)
	}

	// Fall back to AST analysis for string literals
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, `"`)
	}

	return ""
}

// extractDependsOn extracts dependency IDs from a DependsOn field value.
func (a *Analyzer) extractDependsOn(expr ast.Expr, pkg *packages.Package) []string {
	var deps []string

	comp, ok := expr.(*ast.CompositeLit)
	if !ok {
		return deps
	}

	for _, elt := range comp.Elts {
		if dep := a.extractStringValue(elt, pkg); dep != "" {
			deps = append(deps, dep)
		}
	}

	return deps
}

// buildSSA builds the SSA representation for call graph analysis.
func (a *Analyzer) buildSSA() error {
	prog, _ := ssautil.AllPackages(a.pkgs, ssa.SanityCheckFunctions)
	prog.Build()
	a.ssaProg = prog
	return nil
}

// analyzeAllNodes analyzes each registered node for dependency correctness.
func (a *Analyzer) analyzeAllNodes() ([]AnalysisResult, error) {
	var results []AnalysisResult

	for _, def := range a.registry.byID {
		result, err := a.analyzeNode(def)
		if err != nil {
			return nil, fmt.Errorf("analyzing node %q: %w", def.id, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// analyzeNode analyzes a single node for dependency correctness.
func (a *Analyzer) analyzeNode(def *nodeDef) (AnalysisResult, error) {
	result := AnalysisResult{
		NodeID:       def.id,
		File:         def.file,
		DeclaredDeps: def.declaredDeps,
	}

	// Find all Dep[T] calls in the Run function
	usedDeps := a.findDepCalls(def)
	result.UsedDeps = usedDeps

	// Build sets for comparison
	declaredSet := make(map[string]bool)
	for _, d := range def.declaredDeps {
		declaredSet[d] = true
	}

	usedSet := make(map[string]bool)
	for _, d := range usedDeps {
		usedSet[d] = true
	}

	// Find undeclared (used but not declared)
	for dep := range usedSet {
		if !declaredSet[dep] {
			result.Undeclared = append(result.Undeclared, dep)
		}
	}

	// Find unused (declared but not used)
	for dep := range declaredSet {
		if !usedSet[dep] {
			result.Unused = append(result.Unused, dep)
		}
	}

	return result, nil
}

// findDepCalls finds all Dep[T] calls reachable from the Run function.
func (a *Analyzer) findDepCalls(def *nodeDef) []string {
	var deps []string
	seen := make(map[string]bool)

	// Walk the Run function AST to find Dep[T] calls
	if def.runFunc == nil {
		return deps
	}

	// Get the functions to analyze (Run func + transitive calls)
	funcsToAnalyze := a.getReachableFunctions(def)

	for _, fn := range funcsToAnalyze {
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			depType := a.extractDepTypeArg(call, def.pkg)
			if depType == nil {
				return true
			}

			// Look up the node that outputs this type
			nodeDef := a.registry.lookupByType(depType)
			if nodeDef != nil && !seen[nodeDef.id] {
				seen[nodeDef.id] = true
				deps = append(deps, nodeDef.id)
			} else if nodeDef == nil && AnalyzeDirDebug {
				fmt.Printf("[ANALYZE DEBUG] No node found for Dep[%s] in %s\n",
					types.TypeString(depType, nil), def.id)
			}

			return true
		})
	}

	return deps
}

// getReachableFunctions returns all function bodies reachable from Run using SSA call graph.
func (a *Analyzer) getReachableFunctions(def *nodeDef) []ast.Node {
	var nodes []ast.Node
	seen := make(map[*ssa.Function]bool)

	// Always include the Run expression itself in case SSA fails
	nodes = append(nodes, def.runFunc)

	// Helper to find the SSA package matching the source package
	ssaPkg := a.ssaProg.Package(def.pkg.Types)
	if ssaPkg == nil {
		if AnalyzeDirDebug {
			fmt.Printf("[ANALYZE DEBUG] No SSA package found for %s\n", def.id)
		}
		return nodes
	}

	// Find the SSA function corresponding to Run
	entryFn := a.findSSAFunc(ssaPkg, def.runFunc)
	if entryFn == nil {
		// Fallback: if Run is a function selector (e.g. `Run: myFunc`), 
		// findSSAFunc might fail because def.runFunc is just the Identifier/Selector, 
		// not the FuncDecl. We need to resolve it.
		// For now, let's try to handle the specific case where Run IS the function body (FuncLit)
		// or verify if we can resolve the reference.
		if ident, ok := def.runFunc.(*ast.Ident); ok {
			// Resolve ident to object, then look up in SSA
			obj := def.pkg.TypesInfo.ObjectOf(ident)
			if fn, ok := obj.(*types.Func); ok {
				entryFn = ssaPkg.Prog.FuncValue(fn)
			}
		}
	}

	if entryFn == nil {
		if AnalyzeDirDebug {
			fmt.Printf("[ANALYZE DEBUG] No SSA function found for Run in %s\n", def.id)
		}
		return nodes
	}

	// Worklist algorithm to find all reachable functions
	worklist := []*ssa.Function{entryFn}
	seen[entryFn] = true

	for len(worklist) > 0 {
		fn := worklist[0]
		worklist = worklist[1:]

		// Add syntax node to result if available
		if fn.Syntax() != nil {
			nodes = append(nodes, fn.Syntax())
		}

		// Inspect instructions for calls
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				// Check for Call or Defer
				if call, ok := instr.(ssa.CallInstruction); ok {
					callee := call.Common().StaticCallee()
					if callee != nil && !seen[callee] {
						// Only traverse functions in the same module/workspace 
						// (heuristic: if it has syntax, it's likely source)
						if callee.Syntax() != nil {
							seen[callee] = true
							worklist = append(worklist, callee)
						}
					}
				}
			}
		}
	}

	return nodes
}

// findSSAFunc finds the SSA function corresponding to a given AST node (FuncLit or FuncDecl).
func (a *Analyzer) findSSAFunc(pkg *ssa.Package, node ast.Node) *ssa.Function {
	// Search source members
	for _, member := range pkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			if fn.Syntax() == node {
				return fn
			}
			// Search anonymous functions inside members
			for _, anon := range fn.AnonFuncs {
				if anon.Syntax() == node {
					return anon
				}
				// Deep search if needed (nested anon usually flattened in AnonFuncs list? 
				// SSA documentation says "AnonFuncs includes all anonymous functions directly... or indirectly?")
				// Actually SSA lifts anon funcs. Check all.
				if found := a.findSSAFuncInAnon(anon, node); found != nil {
					return found
				}
			}
		}
	}
	
	// Also check Init function
	if pkg.Func("init") != nil {
		if pkg.Func("init").Syntax() == node {
			return pkg.Func("init")
		}
		for _, anon := range pkg.Func("init").AnonFuncs {
			if anon.Syntax() == node {
				return anon
			}
			if found := a.findSSAFuncInAnon(anon, node); found != nil {
				return found
			}
		}
	}

	return nil
}

func (a *Analyzer) findSSAFuncInAnon(fn *ssa.Function, node ast.Node) *ssa.Function {
	if fn.Syntax() == node {
		return fn
	}
	for _, anon := range fn.AnonFuncs {
		if res := a.findSSAFuncInAnon(anon, node); res != nil {
			return res
		}
	}
	return nil
}

// extractDepTypeArg extracts the type argument T from a Dep[T](...) call.
func (a *Analyzer) extractDepTypeArg(call *ast.CallExpr, pkg *packages.Package) types.Type {
	// Must be an IndexExpr (generic call) - Dep[T](ctx)
	indexExpr, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return nil
	}

	// Check if the function is "Dep" or "graft.Dep"
	if !a.isDepCall(indexExpr.X) {
		return nil
	}

	// Get the type of the type argument
	if tv, ok := pkg.TypesInfo.Types[indexExpr.Index]; ok {
		return tv.Type
	}

	return nil
}

// isDepCall checks if an expression is "Dep" or "graft.Dep".
func (a *Analyzer) isDepCall(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name == "Dep"
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name == "graft" && x.Sel.Name == "Dep"
		}
	}
	return false
}

// AnalyzeFile analyzes a single Go file for graft.Node[T] dependency correctness.
//
// Note: For accurate cross-package type resolution, use AnalyzeDir instead.
// This function is provided for backwards compatibility.
func AnalyzeFile(path string) ([]AnalysisResult, error) {
	dir := filepath.Dir(path)
	allResults, err := AnalyzeDir(dir)
	if err != nil {
		return nil, err
	}

	// Filter to only nodes from this file
	var results []AnalysisResult
	for _, r := range allResults {
		if r.File == path {
			results = append(results, r)
		}
	}

	return results, nil
}

// ValidateDeps is a convenience function that returns an error if any
// dependency issues are found.
//
// Pass "." for the current directory or a specific path. This is useful
// for CI integration or programmatic validation.
//
// Example:
//
//	if err := graft.ValidateDeps("./nodes"); err != nil {
//	    log.Fatal(err)
//	}
func ValidateDeps(dir string) error {
	results, err := AnalyzeDir(dir)
	if err != nil {
		return err
	}

	var issues []string
	for _, r := range results {
		if r.HasIssues() {
			issues = append(issues, r.String())
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("dependency validation failed:\n  %s", strings.Join(issues, "\n  "))
	}

	return nil
}

// AnalyzeFileDebug controls whether file-level debug output is printed.
var AnalyzeFileDebug = false
