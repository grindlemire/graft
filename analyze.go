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

// AnalyzeDirDebug controls whether AnalyzeDir prints debug information.
// Set this to true before calling AssertDepsValidVerbose to see file-level tracing.
var AnalyzeDirDebug = false

// AnalyzeDir analyzes all Go files in a directory for dependency correctness.
//
// It recursively walks the directory, parsing each .go file (excluding _test.go)
// and extracting graft.Node[T] definitions. For each node found, it compares
// declared dependencies against actual Dep[T] usage.
//
// Uses two passes: first collects all node registrations to build a type registry,
// then analyzes each node's dependencies using that registry.
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
	if AnalyzeDirDebug {
		absDir, _ := filepath.Abs(dir)
		fmt.Printf("[ANALYZE DEBUG] Walking directory: %s (abs: %s)\n", dir, absDir)
	}

	// Pass 1: Collect all node registrations to build type registry
	typeReg, idRefReg, nodeInfos, err := buildTypeRegistry(dir)
	if err != nil {
		return nil, fmt.Errorf("building type registry: %w", err)
	}

	if AnalyzeDirDebug {
		fmt.Printf("[ANALYZE DEBUG] Type registry: %v\n", typeReg)
		fmt.Printf("[ANALYZE DEBUG] ID Ref registry: %v\n", idRefReg)
		fmt.Printf("[ANALYZE DEBUG] Collected %d node(s)\n", len(nodeInfos))
	}

	// Pass 2: Analyze each node using the registries
	var results []AnalysisResult
	for _, info := range nodeInfos {
		result, err := analyzeNodeWithRegistry(info, typeReg, idRefReg)
		if err != nil {
			return nil, fmt.Errorf("analyzing node %s in %s: %w", info.CanonicalID, info.File, err)
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// buildTypeRegistry walks all files and collects node registrations.
// Returns the type registry, ID reference registry, and a list of node info for pass 2.
func buildTypeRegistry(dir string) (typeRegistry, idRefRegistry, []*nodeInfo, error) {
	typeReg := make(typeRegistry)
	idRefReg := make(idRefRegistry)
	var nodeInfos []*nodeInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if AnalyzeDirDebug {
				fmt.Printf("[ANALYZE DEBUG] Walk error at %s: %v\n", path, err)
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		importRes := buildImportResolver(f)
		constRes := buildConstResolver(f)
		pkgName := f.Name.Name

		// Find all Node[T] composite literals
		ast.Inspect(f, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			nodeInfo := collectNodeInfo(comp, importRes, constRes, path, pkgName)
			if nodeInfo == nil {
				return true
			}

			// Register: canonical type -> canonical ID
			typeReg[nodeInfo.OutputType] = nodeInfo.CanonicalID

			// Register ID reference if we have one (e.g., "myapp/dep.ID" -> "dep-node")
			if nodeInfo.CanonicalIDRef != "" {
				idRefReg[nodeInfo.CanonicalIDRef] = nodeInfo.CanonicalID
			}

			nodeInfos = append(nodeInfos, nodeInfo)

			if AnalyzeDirDebug {
				fmt.Printf("[ANALYZE DEBUG] Registered: %s -> %s (from %s)\n",
					nodeInfo.OutputType, nodeInfo.CanonicalID, path)
				if nodeInfo.CanonicalIDRef != "" {
					fmt.Printf("[ANALYZE DEBUG] ID Ref: %s -> %s\n",
						nodeInfo.CanonicalIDRef, nodeInfo.CanonicalID)
				}
			}

			return true
		})

		return nil
	})

	return typeReg, idRefReg, nodeInfos, err
}

// analyzeNodeWithRegistry analyzes a single node using the type and ID ref registries.
func analyzeNodeWithRegistry(info *nodeInfo, typeReg typeRegistry, idRefReg idRefRegistry) (*AnalysisResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, info.File, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	importRes := buildImportResolver(f)
	constRes := buildConstResolver(f)

	// Build a map of function declarations for Run function reference resolution
	funcDecls := make(map[string]*ast.FuncDecl)
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcDecls[fn.Name.Name] = fn
		}
	}

	pkgName := f.Name.Name

	// Find the Node composite literal for this node
	var nodeComp *ast.CompositeLit
	ast.Inspect(f, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if this is the node we're looking for
		compInfo := collectNodeInfo(comp, importRes, constRes, info.File, pkgName)
		if compInfo != nil && compInfo.CanonicalID == info.CanonicalID && compInfo.OutputType == info.OutputType {
			nodeComp = comp
			return false // found it, stop searching
		}
		return true
	})

	if nodeComp == nil {
		// Node not found in file (shouldn't happen, but handle gracefully)
		return nil, nil
	}

	// Resolve declared deps using ID ref registry
	// This converts canonical ID refs (like "myapp/dep2.ID") to actual ID values (like "dep2-node")
	resolvedDeclaredDeps := make(map[string]bool)
	for canonicalRef := range info.DeclaredDeps {
		if resolvedID, ok := idRefReg[canonicalRef]; ok {
			resolvedDeclaredDeps[resolvedID] = true
		} else {
			// Try short form lookup (pkgName.ID) for full path refs (myapp/dep2.ID)
			if idx := strings.LastIndex(canonicalRef, "/"); idx != -1 {
				shortRef := canonicalRef[idx+1:]
				if resolvedID, ok := idRefReg[shortRef]; ok {
					resolvedDeclaredDeps[resolvedID] = true
					continue
				}
			}
			// Keep the original if not resolvable (e.g., string literals)
			resolvedDeclaredDeps[canonicalRef] = true
		}
	}

	// Analyze the Run function to find Dep[T] calls
	na := &nodeAnalyzer{
		declaredDeps:   resolvedDeclaredDeps,
		usedDeps:       make(map[string]bool),
		typeRegistry:   typeReg,
		importResolver: importRes,
	}

	// Extract the Run function and walk it
	for _, elt := range nodeComp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Run" {
			continue
		}

		// Handle inline function literals
		if _, ok := kv.Value.(*ast.FuncLit); ok {
			ast.Walk(na, kv.Value)
		} else if ident, ok := kv.Value.(*ast.Ident); ok {
			// Handle function reference: Run: run
			if fn, found := funcDecls[ident.Name]; found {
				ast.Walk(na, fn.Body)
			}
		}
		break
	}

	// Build result
	result := &AnalysisResult{
		NodeID: info.CanonicalID,
		File:   info.File,
	}

	for dep := range na.declaredDeps {
		result.DeclaredDeps = append(result.DeclaredDeps, dep)
	}
	for dep := range na.usedDeps {
		result.UsedDeps = append(result.UsedDeps, dep)
	}

	// Find undeclared (used but not declared)
	// Use a helper to match canonical forms with fallback package names
	for usedDep := range na.usedDeps {
		if !matchesDeclaredDep(usedDep, na.declaredDeps) {
			result.Undeclared = append(result.Undeclared, usedDep)
		}
	}

	// Find unused (declared but not used)
	// Use a helper to match canonical forms with fallback package names
	for declaredDep := range na.declaredDeps {
		if !matchesUsedDep(declaredDep, na.usedDeps) {
			result.Unused = append(result.Unused, declaredDep)
		}
	}

	return result, nil
}

// matchesDeclaredDep checks if a used dependency matches any declared dependency.
// Handles both exact matches and fallback package name matching.
func matchesDeclaredDep(usedDep string, declaredDeps map[string]bool) bool {
	if declaredDeps[usedDep] {
		return true
	}
	// If usedDep is a package name (fallback), check if any declared dep
	// is a string literal matching it, or ends with it
	for declaredDep := range declaredDeps {
		// Exact string literal match
		if declaredDep == usedDep {
			return true
		}
		// Check if declared dep ends with the package name (e.g., "myapp/nodes/dep1.ID" ends with "dep1")
		if strings.HasSuffix(declaredDep, "."+usedDep) || strings.HasSuffix(declaredDep, "/"+usedDep) {
			return true
		}
		// Check if declared dep is a canonical form that ends with the package name
		// e.g., "myapp/nodes/dep1" matches "dep1"
		parts := strings.Split(declaredDep, "/")
		if len(parts) > 0 && parts[len(parts)-1] == usedDep {
			return true
		}
		// Check if declared dep ends with ".ID" and the base matches
		if strings.HasSuffix(declaredDep, ".ID") {
			base := strings.TrimSuffix(declaredDep, ".ID")
			if base == usedDep || strings.HasSuffix(base, "/"+usedDep) {
				return true
			}
		}
	}
	return false
}

// matchesUsedDep checks if a declared dependency matches any used dependency.
// Handles both exact matches and fallback package name matching.
func matchesUsedDep(declaredDep string, usedDeps map[string]bool) bool {
	if usedDeps[declaredDep] {
		return true
	}
	// Extract package name from canonical form and check if it matches any used dep
	// e.g., "myapp/nodes/dep1.ID" -> check if "dep1" is in usedDeps
	parts := strings.Split(declaredDep, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove ".ID" suffix if present
		lastPart = strings.TrimSuffix(lastPart, ".ID")
		if usedDeps[lastPart] {
			return true
		}
		// Extract identifier after the last dot (e.g., "shared.ID1" -> "ID1")
		if dotIdx := strings.LastIndex(lastPart, "."); dotIdx != -1 {
			identName := lastPart[dotIdx+1:]
			if usedDeps[identName] {
				return true
			}
		}
	}
	// Also check if declaredDep is a string literal that matches a used dep
	for usedDep := range usedDeps {
		if declaredDep == usedDep {
			return true
		}
	}
	return false
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
// Note: This builds a registry only from nodes in this file, so cross-file
// type resolution may fall back to package name heuristics.
func AnalyzeFile(path string) ([]AnalysisResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	importRes := buildImportResolver(f)
	constRes := buildConstResolver(f)
	pkgName := f.Name.Name

	// Pass 1: Collect node info from this file
	typeReg := make(typeRegistry)
	idRefReg := make(idRefRegistry)
	var nodeInfos []*nodeInfo
	var nonGenericNodes []*ast.CompositeLit

	ast.Inspect(f, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if it's a Node type
		if !isNodeTypeForCollection(comp) {
			return true
		}

		nodeInfo := collectNodeInfo(comp, importRes, constRes, path, pkgName)
		if nodeInfo != nil {
			typeReg[nodeInfo.OutputType] = nodeInfo.CanonicalID
			if nodeInfo.CanonicalIDRef != "" {
				idRefReg[nodeInfo.CanonicalIDRef] = nodeInfo.CanonicalID
			}
			nodeInfos = append(nodeInfos, nodeInfo)
		} else {
			// Non-generic node - handle separately
			nonGenericNodes = append(nonGenericNodes, comp)
		}
		return true
	})

	// Pass 2: Analyze generic nodes with registry
	var results []AnalysisResult
	for _, info := range nodeInfos {
		result, err := analyzeNodeWithRegistry(info, typeReg, idRefReg)
		if err != nil {
			return nil, fmt.Errorf("analyzing node %s: %w", info.CanonicalID, err)
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	// Handle non-generic nodes with old logic (no type registry)
	funcDecls := make(map[string]*ast.FuncDecl)
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcDecls[fn.Name.Name] = fn
		}
	}

	oldAnalyzer := &fileAnalyzer{
		fset:      token.NewFileSet(),
		file:      path,
		funcDecls: funcDecls,
		results:   results,
	}

	for _, comp := range nonGenericNodes {
		result := oldAnalyzer.analyzeNodeLiteral(comp)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// AnalyzeFileDebug controls whether fileAnalyzer prints debug info about AST traversal.
var AnalyzeFileDebug = false

// importResolver maps import aliases to their full paths for a single file.
type importResolver map[string]string // alias -> full import path

// constResolver maps constant names to their string literal values.
type constResolver map[string]string // constName -> "literal value"

// buildImportResolver parses a file's imports and returns alias -> path mapping.
func buildImportResolver(f *ast.File) importResolver {
	resolver := make(importResolver)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name // explicit alias
		} else {
			// default alias is last path component
			alias = filepath.Base(path)
		}
		resolver[alias] = path
	}
	return resolver
}

// buildConstResolver scans a file's declarations and extracts constant string values.
// This enables resolving constants like `const ID = "mynode"` to their actual values.
func buildConstResolver(f *ast.File) constResolver {
	resolver := make(constResolver)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || (genDecl.Tok != token.CONST && genDecl.Tok != token.VAR) {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// Handle each name with its corresponding value
			for i, name := range valueSpec.Names {
				if i < len(valueSpec.Values) {
					if lit, ok := valueSpec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						resolver[name.Name] = strings.Trim(lit.Value, `"`)
					}
				}
			}
		}
	}
	return resolver
}

// canonicalizeExpr converts a type/ID expression to canonical form.
// e.g., ports.Executor with resolver["ports"]="myapp/ports" → "myapp/ports.Executor"
// String literals are returned as-is.
func canonicalizeExpr(expr ast.Expr, resolver importResolver) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		if t.Kind == token.STRING {
			return strings.Trim(t.Value, `"`)
		}
	case *ast.Ident:
		return t.Name // local identifier, use as-is
	case *ast.SelectorExpr:
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			if fullPath, ok := resolver[pkgIdent.Name]; ok {
				return fullPath + "." + t.Sel.Name
			}
			return pkgIdent.Name + "." + t.Sel.Name // fallback
		}
	case *ast.StarExpr:
		// Handle pointer types: *T -> "*T" canonical form
		inner := canonicalizeExpr(t.X, resolver)
		if inner != "" {
			return "*" + inner
		}
	}
	return ""
}

// canonicalizeExprWithPkg is like canonicalizeExpr but prefixes local types with the package name.
// This is used when registering types to ensure they can be looked up from other packages.
func canonicalizeExprWithPkg(expr ast.Expr, resolver importResolver, pkgName string) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		if t.Kind == token.STRING {
			return strings.Trim(t.Value, `"`)
		}
	case *ast.Ident:
		// Local type - prefix with package name so it can be found by other packages
		return pkgName + "." + t.Name
	case *ast.SelectorExpr:
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			if fullPath, ok := resolver[pkgIdent.Name]; ok {
				return fullPath + "." + t.Sel.Name
			}
			return pkgIdent.Name + "." + t.Sel.Name // fallback
		}
	case *ast.StarExpr:
		// Handle pointer types: *T -> "*T" canonical form
		inner := canonicalizeExprWithPkg(t.X, resolver, pkgName)
		if inner != "" {
			return "*" + inner
		}
	}
	return ""
}

// typeRegistry maps canonical output types to node ID string values.
type typeRegistry map[string]string // "myapp/ports.Executor" -> "executor"

// idRefRegistry maps canonical ID references to node ID string values.
// This is used to resolve selector expressions like dep2.ID to their actual values.
type idRefRegistry map[string]string // "myapp/dep2.ID" -> "dep2-node"

// nodeInfo holds collected information about a node during pass 1.
type nodeInfo struct {
	CanonicalID    string          // resolved string value of the node ID
	CanonicalIDRef string          // canonical reference (e.g., "myapp/dep.ID") if ID was a selector
	OutputType     string          // canonical output type
	DeclaredDeps   map[string]bool // canonical dep references
	File           string
}

// extractNodeTypeParam returns the type expression T from Node[T].
// Returns nil if this isn't a Node[T] literal.
func extractNodeTypeParam(comp *ast.CompositeLit) ast.Expr {
	switch t := comp.Type.(type) {
	case *ast.IndexExpr:
		// Node[T] or graft.Node[T]
		if isNodeTypeIndex(t.X) {
			return t.Index
		}
	case *ast.IndexListExpr:
		// Node[T, U, ...] - use first type param
		if isNodeTypeIndex(t.X) && len(t.Indices) > 0 {
			return t.Indices[0]
		}
	}
	return nil
}

// isNodeTypeIndex checks if an expression is "Node" or "graft.Node" for IndexExpr.
func isNodeTypeIndex(expr ast.Expr) bool {
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

// isNodeTypeForCollection checks if a composite literal is a Node type (for collection phase).
// This includes both generic Node[T] and non-generic Node.
func isNodeTypeForCollection(comp *ast.CompositeLit) bool {
	switch t := comp.Type.(type) {
	case *ast.Ident:
		// Node (non-generic)
		return t.Name == "Node"
	case *ast.SelectorExpr:
		// graft.Node (non-generic)
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "graft" && t.Sel.Name == "Node"
		}
	case *ast.IndexExpr:
		// Node[T] or graft.Node[T] - generic form
		return isNodeTypeIndex(t.X)
	case *ast.IndexListExpr:
		// Node[T, U, ...] - multiple type params
		return isNodeTypeIndex(t.X)
	}
	return false
}

// collectNodeInfo extracts node information from a Node composite literal during pass 1.
// Returns nil if this isn't a Node literal or if it's a non-generic Node (which is handled separately).
func collectNodeInfo(comp *ast.CompositeLit, importRes importResolver, constRes constResolver, file string, pkgName string) *nodeInfo {
	// Check if this is a Node type (generic or non-generic)
	if !isNodeTypeForCollection(comp) {
		return nil
	}

	typeParam := extractNodeTypeParam(comp)
	// Non-generic nodes don't have type parameters - they're handled by old code path
	if typeParam == nil {
		return nil
	}

	// Use package-qualified canonicalization for output types so they can be found by other packages
	canonicalType := canonicalizeExprWithPkg(typeParam, importRes, pkgName)
	if canonicalType == "" {
		return nil
	}

	info := &nodeInfo{
		OutputType:   canonicalType,
		DeclaredDeps: make(map[string]bool),
		File:         file,
	}

	// Extract ID and DependsOn from the composite literal
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
			// First try to get the raw expression value
			rawID := canonicalizeExpr(kv.Value, importRes)
			if rawID == "" {
				continue
			}

			// If it's a local identifier, try to resolve it as a constant
			if ident, ok := kv.Value.(*ast.Ident); ok {
				// Try to resolve the constant to its string value
				if resolvedValue, ok := constRes[ident.Name]; ok {
					info.CanonicalID = resolvedValue
					// Store the canonical reference for ID registry (e.g., "pkgname.ID")
					info.CanonicalIDRef = pkgName + "." + ident.Name
				} else if ident.Name == "ID" {
					// Special case: if the identifier is literally "ID" and not resolvable,
					// use package name as fallback
					dir := filepath.Dir(file)
					info.CanonicalID = filepath.Base(dir)
					info.CanonicalIDRef = pkgName + ".ID"
				} else {
					// Fallback to the identifier name if not resolvable
					info.CanonicalID = rawID
					info.CanonicalIDRef = pkgName + "." + ident.Name
				}
			} else if _, ok := kv.Value.(*ast.BasicLit); ok {
				// String literal - no canonical ref needed
				info.CanonicalID = rawID
			} else {
				// For selector expressions, use as-is
				info.CanonicalID = rawID
			}
		case "DependsOn":
			extractDependsOnCanonical(kv.Value, importRes, constRes, info.DeclaredDeps)
		}
	}

	if info.CanonicalID == "" {
		return nil
	}

	return info
}

// extractDependsOnCanonical extracts declared dependencies and canonicalizes them.
func extractDependsOnCanonical(n ast.Node, importRes importResolver, constRes constResolver, deps map[string]bool) {
	switch v := n.(type) {
	case *ast.CompositeLit:
		// []graft.ID{config.ID, db.ID} or []string{"dep1", "dep2"}
		for _, elt := range v.Elts {
			// First try to resolve as a constant if it's a simple identifier
			if ident, ok := elt.(*ast.Ident); ok {
				if resolvedValue, ok := constRes[ident.Name]; ok {
					deps[resolvedValue] = true
					continue
				}
			}

			canonical := canonicalizeExpr(elt, importRes)
			if canonical != "" {
				deps[canonical] = true
			}
			// Handle call expressions like graft.ID("foo")
			if call, ok := elt.(*ast.CallExpr); ok {
				extractFromCallCanonical(call, importRes, deps)
			}
		}
	case *ast.CallExpr:
		// Handle cases like []graft.ID{nodeA.ID, nodeB.ID} or function calls
		for _, arg := range v.Args {
			extractDependsOnCanonical(arg, importRes, constRes, deps)
		}
	}
}

// extractFromCallCanonical extracts dependency ID from a call expression like graft.ID("foo").
func extractFromCallCanonical(call *ast.CallExpr, importRes importResolver, deps map[string]bool) {
	// Check if this is graft.ID(...) or ID(...)
	isIDCall := false

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		isIDCall = fn.Name == "ID"
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			isIDCall = ident.Name == "graft" && fn.Sel.Name == "ID"
		}
	}

	if isIDCall && len(call.Args) > 0 {
		canonical := canonicalizeExpr(call.Args[0], importRes)
		if canonical != "" {
			deps[canonical] = true
		}
	}
}

// fileAnalyzer is an ast.Visitor that finds graft.Node[T] definitions in a file.
type fileAnalyzer struct {
	fset      *token.FileSet
	file      string
	funcDecls map[string]*ast.FuncDecl // function declarations in this file
	results   []AnalysisResult
}

// getPackageNameAsID returns the package name (directory name) as the node ID.
// This is used when the ID field references a constant like "ID: ID".
func (a *fileAnalyzer) getPackageNameAsID(identName string) string {
	// Use the directory name as the package name
	dir := filepath.Dir(a.file)
	return filepath.Base(dir)
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

	if AnalyzeFileDebug {
		fmt.Printf("[AST DEBUG] Found CompositeLit in %s, type: %T\n", a.file, comp.Type)
		if comp.Type != nil {
			fmt.Printf("[AST DEBUG]   Type details: %#v\n", comp.Type)
		}
	}

	if !a.isNodeType(comp) {
		if AnalyzeFileDebug {
			fmt.Printf("[AST DEBUG]   -> Not a Node type, skipping\n")
		}
		return a
	}

	if AnalyzeFileDebug {
		fmt.Printf("[AST DEBUG]   -> IS a Node type, analyzing...\n")
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

	if AnalyzeFileDebug {
		fmt.Printf("[AST DEBUG]   analyzeNodeLiteral: %d elements in composite literal\n", len(comp.Elts))
	}

	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]     element is not KeyValueExpr: %T\n", elt)
			}
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]     key is not Ident: %T\n", kv.Key)
			}
			continue
		}

		if AnalyzeFileDebug {
			fmt.Printf("[AST DEBUG]     field %q: value type %T\n", key.Name, kv.Value)
		}

		switch key.Name {
		case "ID":
			// Handle string literal: ID: "mynode"
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				nodeID = strings.Trim(lit.Value, `"`)
				if AnalyzeFileDebug {
					fmt.Printf("[AST DEBUG]       ID = %q (from string literal)\n", nodeID)
				}
			} else if ident, ok := kv.Value.(*ast.Ident); ok {
				// Handle identifier: ID: ID (where ID is a const)
				// Use the package name as the node ID since the const is typically
				// named "ID" and the package name is more meaningful
				nodeID = a.getPackageNameAsID(ident.Name)
				if AnalyzeFileDebug {
					fmt.Printf("[AST DEBUG]       ID = %q (from identifier %q, using package name)\n", nodeID, ident.Name)
				}
			} else if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]       ID value is not a string literal or identifier: %T\n", kv.Value)
			}
		case "DependsOn":
			na.extractDependsOn(kv.Value)
			if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]       DependsOn extracted: %v\n", na.declaredDeps)
			}
		case "Run":
			// Handle inline function literals
			if _, ok := kv.Value.(*ast.FuncLit); ok {
				ast.Walk(na, kv.Value)
				if AnalyzeFileDebug {
					fmt.Printf("[AST DEBUG]       Run (inline func): usedDeps: %v\n", na.usedDeps)
				}
			} else if ident, ok := kv.Value.(*ast.Ident); ok {
				// Handle function reference: Run: run
				// Look up the function declaration in this file
				if fn, found := a.funcDecls[ident.Name]; found {
					if AnalyzeFileDebug {
						fmt.Printf("[AST DEBUG]       Run references func %q, walking its body\n", ident.Name)
					}
					ast.Walk(na, fn.Body)
					if AnalyzeFileDebug {
						fmt.Printf("[AST DEBUG]       Run usedDeps: %v\n", na.usedDeps)
					}
				} else if AnalyzeFileDebug {
					fmt.Printf("[AST DEBUG]       Run references func %q but not found in file\n", ident.Name)
				}
			} else if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]       Run value type %T not handled\n", kv.Value)
			}
		}
	}

	if nodeID == "" {
		if AnalyzeFileDebug {
			fmt.Printf("[AST DEBUG]   -> nodeID is empty, returning nil\n")
		}
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
	declaredDeps   map[string]bool
	usedDeps       map[string]bool
	typeRegistry   typeRegistry
	importResolver importResolver
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
// Handles string literals, identifiers, selector expressions, and call expressions.
// For selector expressions like pkg.ID, the package name is used as the dependency.
// For call expressions like graft.ID("foo"), the string argument is used.
func (a *nodeAnalyzer) extractDependsOn(n ast.Node) {
	switch v := n.(type) {
	case *ast.CompositeLit:
		// []graft.ID{config.ID, db.ID} or []string{"dep1", "dep2"}
		for _, elt := range v.Elts {
			// Handle string literals
			if lit, ok := elt.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				dep := strings.Trim(lit.Value, `"`)
				a.declaredDeps[dep] = true
				continue
			}
			// Handle selector expressions like pkg.ID - use package name
			if sel, ok := elt.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := sel.X.(*ast.Ident); ok {
					a.declaredDeps[pkgIdent.Name] = true
				}
				continue
			}
			// Handle identifiers (local constants)
			if ident, ok := elt.(*ast.Ident); ok {
				a.declaredDeps[ident.Name] = true
				continue
			}
			// Handle call expressions like graft.ID("foo") or ID("foo")
			if call, ok := elt.(*ast.CallExpr); ok {
				a.extractFromCall(call)
			}
		}
	case *ast.CallExpr:
		// Handle cases like []graft.ID{nodeA.ID, nodeB.ID} or function calls
		for _, arg := range v.Args {
			a.extractDependsOn(arg)
		}
	}
}

// extractFromCall extracts dependency ID from a call expression like graft.ID("foo").
func (a *nodeAnalyzer) extractFromCall(call *ast.CallExpr) {
	// Check if this is graft.ID(...) or ID(...)
	isIDCall := false

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		// ID("foo")
		isIDCall = fn.Name == "ID"
	case *ast.SelectorExpr:
		// graft.ID("foo")
		if ident, ok := fn.X.(*ast.Ident); ok {
			isIDCall = ident.Name == "graft" && fn.Sel.Name == "ID"
		}
	}

	if isIDCall && len(call.Args) > 0 {
		// Extract the string argument
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			dep := strings.Trim(lit.Value, `"`)
			a.declaredDeps[dep] = true
			if AnalyzeFileDebug {
				fmt.Printf("[AST DEBUG]         extracted %q from call expression\n", dep)
			}
		}
	}
}

// checkDepCall looks for Dep[T](ctx) calls and resolves the dependency ID
// using the type registry. Falls back to package name heuristic if not found.
func (a *nodeAnalyzer) checkDepCall(n ast.Node) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}

	// Check for graft.Dep[T](...) or Dep[T](...)
	var typeExpr ast.Expr

	switch fn := call.Fun.(type) {
	case *ast.IndexExpr:
		// Dep[T] - generic function call with single type param
		switch x := fn.X.(type) {
		case *ast.Ident:
			if x.Name == "Dep" {
				typeExpr = fn.Index
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "Dep" {
				typeExpr = fn.Index
			}
		}
	case *ast.IndexListExpr:
		// Dep[T, U] - generic function with multiple type params (future-proofing)
		switch x := fn.X.(type) {
		case *ast.Ident:
			if x.Name == "Dep" && len(fn.Indices) > 0 {
				typeExpr = fn.Indices[0]
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "Dep" && len(fn.Indices) > 0 {
				typeExpr = fn.Indices[0]
			}
		}
	}

	if typeExpr == nil {
		return
	}

	// Canonicalize the type expression
	canonicalType := canonicalizeExpr(typeExpr, a.importResolver)
	if canonicalType == "" {
		return
	}

	// Try to resolve via registry first (exact match)
	if canonicalID, ok := a.typeRegistry[canonicalType]; ok {
		a.usedDeps[canonicalID] = true
		return
	}

	// Try short form (pkgName.Type) extracted from full path
	// e.g., "myapp/ports.Executor" -> "ports.Executor"
	// e.g., "*myapp/ports.Executor" -> "*ports.Executor"
	prefix := ""
	typeForShortForm := canonicalType
	if strings.HasPrefix(canonicalType, "*") {
		prefix = "*"
		typeForShortForm = canonicalType[1:]
	}
	if idx := strings.LastIndex(typeForShortForm, "/"); idx != -1 {
		shortForm := prefix + typeForShortForm[idx+1:]
		if canonicalID, ok := a.typeRegistry[shortForm]; ok {
			a.usedDeps[canonicalID] = true
			return
		}
	}

	// FALLBACK: Use package name (original behavior for unregistered types)
	if sel, ok := typeExpr.(*ast.SelectorExpr); ok {
		if pkgIdent, ok := sel.X.(*ast.Ident); ok {
			a.usedDeps[pkgIdent.Name] = true
		}
	}
	// For simple identifiers (same package), use the type name
	if ident, ok := typeExpr.(*ast.Ident); ok {
		a.usedDeps[ident.Name] = true
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
