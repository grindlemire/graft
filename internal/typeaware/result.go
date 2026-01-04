package typeaware

import (
	"fmt"
	"strings"
)

// Result contains the result of analyzing a node's dependency usage.
//
// It captures both declared dependencies (in DependsOn) and used dependencies
// (via Dep[T] calls), allowing detection of mismatches.
type Result struct {
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
func (r Result) HasIssues() bool {
	return len(r.Undeclared) > 0 || len(r.Unused) > 0 || len(r.Cycles) > 0
}

// String returns a human-readable summary of issues.
//
// Returns "NodeID: OK" if there are no issues, otherwise returns
// a summary of undeclared, unused dependencies, and cycles.
func (r Result) String() string {
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
