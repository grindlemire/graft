package graft

import (
	"fmt"
	"strings"

	"github.com/grindlemire/graft/internal/typeaware"
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

	// Cycles are circular dependency paths this node participates in.
	// Each cycle is represented as a path of node IDs forming a loop.
	// For example: ["svc5", "svc5-2", "svc5"] indicates svc5 → svc5-2 → svc5.
	Cycles [][]string
}

// HasIssues returns true if there are undeclared, unused dependencies, or cycles.
func (r AnalysisResult) HasIssues() bool {
	return len(r.Undeclared) > 0 || len(r.Unused) > 0 || len(r.Cycles) > 0
}

// String returns a human-readable summary of issues.
//
// Returns "NodeID: OK" if there are no issues, otherwise returns
// a summary of undeclared, unused dependencies, and cycles.
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
	if len(r.Cycles) > 0 {
		var cycleStrs []string
		for _, cycle := range r.Cycles {
			cycleStrs = append(cycleStrs, strings.Join(cycle, " → "))
		}
		parts = append(parts, fmt.Sprintf("cycles: [%s]", strings.Join(cycleStrs, ", ")))
	}
	return fmt.Sprintf("%s (%s): %s", r.NodeID, r.File, strings.Join(parts, "; "))
}

// AnalyzeDirDebug controls whether AnalyzeDir prints debug information.
// Set this to true before calling AssertDepsValidVerbose to see file-level tracing.
var AnalyzeDirDebug = false

// AnalyzeDir analyzes all Go files in a directory for dependency correctness.
//
// This function uses type-aware analysis with go/packages and go/ssa to accurately
// detect dependency issues. It discovers all graft.Node[T] registrations in the
// directory and compares declared dependencies (in DependsOn) against actual
// Dep[T] usage in Run functions.
//
// The type-aware approach is more robust than AST-based pattern matching:
//   - Handles type aliases correctly
//   - Resolves package imports accurately
//   - Works with various code structures (dependencies in same package, etc.)
//   - Uses SSA for precise dataflow analysis
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
func AnalyzeDir(dir string) ([]typeaware.Result, error) {
	cfg := typeaware.Config{
		WorkDir: dir,
		Debug:   AnalyzeDirDebug,
	}
	analyzer := typeaware.New(cfg)
	return analyzer.Analyze(dir)
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
