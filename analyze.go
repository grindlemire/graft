package graft

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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

// AnalyzeDir analyzes all Go files in a directory for dependency correctness.
//
// It recursively walks the directory, parsing each .go file (excluding _test.go)
// and extracting graft.Node[T] definitions. For each node found, it compares
// declared dependencies against actual Dep[T] usage.
//
// Returns all nodes found with their analysis results. Use [AnalysisResult.HasIssues]
// to filter for problems.
//
// Example:
//
//	results, err := graft.AnalyzeDir("./nodes")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range results {
//	    if r.HasIssues() {
//	        fmt.Println(r.String())
//	    }
//	}
func AnalyzeDir(dir string) ([]AnalysisResult, error) {
	var results []AnalysisResult

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileResults, err := AnalyzeFile(path)
		if err != nil {
			return fmt.Errorf("analyzing %s: %w", path, err)
		}
		results = append(results, fileResults...)
		return nil
	})

	return results, err
}

// AnalyzeFile analyzes a single Go file for graft.Node[T] dependency correctness.
//
// Parses the file's AST and finds all graft.Node[T] composite literals.
// For each node, it extracts:
//   - The ID field value
//   - Dependencies from DependsOn
//   - Dependencies used via Dep[T] calls in the Run function
//
// Returns one AnalysisResult per node found in the file.
func AnalyzeFile(path string) ([]AnalysisResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	analyzer := &fileAnalyzer{
		fset:    fset,
		file:    path,
		results: make([]AnalysisResult, 0),
	}

	ast.Walk(analyzer, f)

	return analyzer.results, nil
}

// fileAnalyzer is an ast.Visitor that finds graft.Node[T] definitions in a file.
type fileAnalyzer struct {
	fset    *token.FileSet
	file    string
	results []AnalysisResult
}

// Visit implements ast.Visitor. It looks for composite literals of type
// graft.Node[T] or Node[T] and analyzes them.
func (a *fileAnalyzer) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	// Look for composite literals that might be graft.Node[T]
	comp, ok := n.(*ast.CompositeLit)
	if !ok {
		return a
	}

	if !a.isNodeType(comp) {
		return a
	}

	result := a.analyzeNodeLiteral(comp)
	if result != nil {
		a.results = append(a.results, *result)
	}

	return a
}

// isNodeType checks if a composite literal is a graft.Node[T] or Node[T] type.
// Handles both generic (Node[T]) and non-generic (Node) forms for compatibility.
func (a *fileAnalyzer) isNodeType(comp *ast.CompositeLit) bool {
	switch t := comp.Type.(type) {
	case *ast.Ident:
		// Node (non-generic, for backwards compat in analysis)
		return t.Name == "Node"
	case *ast.SelectorExpr:
		// graft.Node (non-generic)
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "graft" && t.Sel.Name == "Node"
		}
	case *ast.IndexExpr:
		// Node[T] or graft.Node[T] - generic form
		return a.isNodeIdent(t.X)
	case *ast.IndexListExpr:
		// Node[T, U, ...] - multiple type params (future-proofing)
		return a.isNodeIdent(t.X)
	}
	return false
}

// isNodeIdent checks if an expression is "Node" or "graft.Node"
func (a *fileAnalyzer) isNodeIdent(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name == "Node"
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name == "graft" && x.Sel.Name == "Node"
		}
	}
	return false
}

// analyzeNodeLiteral extracts dependency info from a Node composite literal.
func (a *fileAnalyzer) analyzeNodeLiteral(comp *ast.CompositeLit) *AnalysisResult {
	na := &nodeAnalyzer{
		declaredDeps: make(map[string]bool),
		usedDeps:     make(map[string]bool),
	}

	var nodeID string

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
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				nodeID = strings.Trim(lit.Value, `"`)
			}
		case "DependsOn":
			na.extractDependsOn(kv.Value)
		case "Run":
			ast.Walk(na, kv.Value)
		}
	}

	if nodeID == "" {
		return nil
	}

	// Build result
	result := &AnalysisResult{
		NodeID: nodeID,
		File:   a.file,
	}

	for dep := range na.declaredDeps {
		result.DeclaredDeps = append(result.DeclaredDeps, dep)
	}
	for dep := range na.usedDeps {
		result.UsedDeps = append(result.UsedDeps, dep)
	}

	// Find undeclared (used but not declared)
	for dep := range na.usedDeps {
		if !na.declaredDeps[dep] {
			result.Undeclared = append(result.Undeclared, dep)
		}
	}

	// Find unused (declared but not used)
	for dep := range na.declaredDeps {
		if !na.usedDeps[dep] {
			result.Unused = append(result.Unused, dep)
		}
	}

	return result
}

// nodeAnalyzer is an ast.Visitor that extracts dependency information
// from a Node's DependsOn field and Run function body.
type nodeAnalyzer struct {
	declaredDeps map[string]bool
	usedDeps     map[string]bool
}

// Visit implements ast.Visitor. It looks for Dep[T] calls within the
// Run function to find used dependencies.
func (a *nodeAnalyzer) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	a.checkDepCall(n)
	return a
}

// extractDependsOn extracts declared dependencies from a DependsOn field value.
// Handles string literals, identifiers, and selector expressions.
func (a *nodeAnalyzer) extractDependsOn(n ast.Node) {
	switch v := n.(type) {
	case *ast.CompositeLit:
		// []string{"dep1", "dep2"}
		for _, elt := range v.Elts {
			if lit, ok := elt.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				dep := strings.Trim(lit.Value, `"`)
				a.declaredDeps[dep] = true
			}
		}
	case *ast.CallExpr:
		// Handle cases like []string{nodeA.ID, nodeB.ID} or function calls
		for _, arg := range v.Args {
			a.extractDependsOn(arg)
		}
	}

	// Also handle inline composite literals with expressions
	if comp, ok := n.(*ast.CompositeLit); ok {
		for _, elt := range comp.Elts {
			// Handle selector expressions like pkg.NodeID or consts
			if sel, ok := elt.(*ast.SelectorExpr); ok {
				// Use the selector name as a hint (e.g., pkg.NodeID -> "NodeID" might be the ID)
				a.declaredDeps[sel.Sel.Name] = true
			}
			// Handle identifiers (local constants)
			if ident, ok := elt.(*ast.Ident); ok {
				a.declaredDeps[ident.Name] = true
			}
		}
	}
}

// checkDepCall looks for Dep[T](ctx, "nodeID") calls and extracts the nodeID.
func (a *nodeAnalyzer) checkDepCall(n ast.Node) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}

	// Check for graft.Dep[T](...) or Dep[T](...)
	var funcName string

	switch fn := call.Fun.(type) {
	case *ast.IndexExpr:
		// Dep[T] - generic function call with single type param
		switch x := fn.X.(type) {
		case *ast.Ident:
			funcName = x.Name
		case *ast.SelectorExpr:
			if x.Sel.Name == "Dep" {
				funcName = "Dep"
			}
		}
	case *ast.IndexListExpr:
		// Dep[T, U] - generic function with multiple type params (future-proofing)
		switch x := fn.X.(type) {
		case *ast.Ident:
			funcName = x.Name
		case *ast.SelectorExpr:
			if x.Sel.Name == "Dep" {
				funcName = "Dep"
			}
		}
	}

	if funcName != "Dep" {
		return
	}

	// Extract the nodeID from the second argument (first is ctx)
	if len(call.Args) >= 2 {
		if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			dep := strings.Trim(lit.Value, `"`)
			a.usedDeps[dep] = true
		}
		// Handle constants/identifiers
		if ident, ok := call.Args[1].(*ast.Ident); ok {
			a.usedDeps[ident.Name] = true
		}
		// Handle selector expressions (pkg.Const)
		if sel, ok := call.Args[1].(*ast.SelectorExpr); ok {
			a.usedDeps[sel.Sel.Name] = true
		}
	}
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
