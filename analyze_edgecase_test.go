package graft

import (
	"path/filepath"
	"sort"
	"testing"
)

// TestAnalyzeDirEdgeCases_Undeclared tests detection of undeclared dependencies.
// These cases verify that the type-aware analyzer correctly identifies when a node
// uses dependencies via graft.Dep[T](ctx) but fails to declare them in DependsOn.
func TestAnalyzeDirEdgeCases_Undeclared(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"01_undeclared_single": {
			dir:            "examples/edgecases/01_undeclared_single",
			wantNodes:      2,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUndeclared(t, app, []string{"config"})
				assertUnused(t, app, []string{})
				assertNoCycles(t, app)
				assertDeps(t, app, []string{}, []string{"config"})
			},
		},
		"02_undeclared_multiple": {
			dir:            "examples/edgecases/02_undeclared_multiple",
			wantNodes:      4,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUndeclaredContains(t, app, []string{"config", "db", "cache"})
				assertUnused(t, app, []string{})
				assertNoCycles(t, app)
			},
		},
		"15_partial_declaration": {
			dir:            "examples/edgecases/15_partial_declaration",
			wantNodes:      4,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUndeclared(t, app, []string{"cache"})
				assertUnused(t, app, []string{})
				assertNoCycles(t, app)
				assertDepsContain(t, app, []string{"config", "db"}, []string{"config", "db", "cache"})
			},
		},
		"20_conditional_dep_usage": {
			dir:            "examples/edgecases/20_conditional_dep_usage",
			wantNodes:      3,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				// Type-aware analysis via SSA should catch conditional usage
				assertUndeclared(t, app, []string{"feature"})
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_Unused tests detection of unused dependencies.
// These cases verify that the analyzer identifies dependencies declared in
// DependsOn but never actually accessed via graft.Dep[T](ctx).
func TestAnalyzeDirEdgeCases_Unused(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"03_unused_single": {
			dir:            "examples/edgecases/03_unused_single",
			wantNodes:      2,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUnused(t, app, []string{"config"})
				assertUndeclared(t, app, []string{})
				assertNoCycles(t, app)
				assertDeps(t, app, []string{"config"}, []string{})
			},
		},
		"04_unused_multiple": {
			dir:            "examples/edgecases/04_unused_multiple",
			wantNodes:      4,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUnusedContains(t, app, []string{"config", "db", "cache"})
				assertUndeclared(t, app, []string{})
				assertNoCycles(t, app)
			},
		},
		"14_unused_in_chain": {
			dir:            "examples/edgecases/14_unused_in_chain",
			wantNodes:      4,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				middleware := findNode(t, results, "middleware")
				assertUnused(t, middleware, []string{"db"})
				assertUndeclared(t, middleware, []string{})
				assertNoCycles(t, middleware)
			},
		},
		"18_complex_multi_parent": {
			dir:            "examples/edgecases/18_complex_multi_parent",
			wantNodes:      5,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				aggregator := findNode(t, results, "aggregator")
				assertUnused(t, aggregator, []string{"serviceC"})
				assertNoCycles(t, aggregator)
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_Cycles tests cycle detection via DFS.
// These cases verify that circular dependencies are correctly identified,
// including simple 2-node cycles, longer chains, and self-references.
func TestAnalyzeDirEdgeCases_Cycles(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"05_cycle_simple": {
			dir:            "examples/edgecases/05_cycle_simple",
			wantNodes:      2,
			wantIssueCount: 2,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")
				assertCycles(t, nodeA, 1)
				assertCycles(t, nodeB, 1)
				// Both should have the cycle path
				assertCycleContains(t, nodeA, []string{"nodeA", "nodeB", "nodeA"})
				assertCycleContains(t, nodeB, []string{"nodeA", "nodeB", "nodeA"})
				// Note: Both will also have unused deps due to Go import cycle limitation
				assertUnused(t, nodeA, []string{"nodeB"})
				assertUnused(t, nodeB, []string{"nodeA"})
			},
		},
		"06_cycle_triangle": {
			dir:            "examples/edgecases/06_cycle_triangle",
			wantNodes:      3,
			wantIssueCount: 3,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")
				nodeC := findNode(t, results, "nodeC")
				assertCycles(t, nodeA, 1)
				assertCycles(t, nodeB, 1)
				assertCycles(t, nodeC, 1)
				// All will also have unused deps due to Go import cycle limitation
			},
		},
		"07_cycle_deep": {
			dir:            "examples/edgecases/07_cycle_deep",
			wantNodes:      5,
			wantIssueCount: 3,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				// Only nodeC, nodeD, nodeE should be in the cycle
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")
				nodeC := findNode(t, results, "nodeC")
				nodeD := findNode(t, results, "nodeD")
				nodeE := findNode(t, results, "nodeE")

				assertNoCycles(t, nodeA)
				assertNoCycles(t, nodeB)
				assertCycles(t, nodeC, 1)
				assertCycles(t, nodeD, 1)
				assertCycles(t, nodeE, 1)

				assertCycleContains(t, nodeC, []string{"nodeC", "nodeD", "nodeE", "nodeC"})
			},
		},
		"08_cycle_self": {
			dir:            "examples/edgecases/08_cycle_self",
			wantNodes:      1,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				nodeA := findNode(t, results, "nodeA")
				assertCycles(t, nodeA, 1)
				assertCycleContains(t, nodeA, []string{"nodeA", "nodeA"})
			},
		},
		"16_multiple_cycles_same_node": {
			dir:            "examples/edgecases/16_multiple_cycles_same_node",
			wantNodes:      3,
			wantIssueCount: 3,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				hub := findNode(t, results, "hub")
				// Hub participates in 2 cycles
				assertCycles(t, hub, 2)
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_Mixed tests combinations of multiple issue types.
// These cases verify that the analyzer can detect multiple problems in the
// same node or graph (e.g., undeclared + unused, cycles + undeclared, etc.).
func TestAnalyzeDirEdgeCases_Mixed(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"09_mixed_undeclared_unused": {
			dir:            "examples/edgecases/09_mixed_undeclared_unused",
			wantNodes:      4,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				app := findNode(t, results, "app")
				assertUndeclared(t, app, []string{"cache"})
				assertUnusedContains(t, app, []string{"config", "db"})
				assertNoCycles(t, app)
			},
		},
		"10_mixed_cycle_undeclared": {
			dir:            "examples/edgecases/10_mixed_cycle_undeclared",
			wantNodes:      3,
			wantIssueCount: 2,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")

				// nodeA has undeclared dep AND is in a cycle
				assertUndeclared(t, nodeA, []string{"config"})
				assertCycles(t, nodeA, 1)

				// nodeB is in the cycle
				assertCycles(t, nodeB, 1)
			},
		},
		"11_mixed_all_issues": {
			dir:            "examples/edgecases/11_mixed_all_issues",
			wantNodes:      5,
			wantIssueCount: 2,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")

				// nodeA: undeclared + unused + cycle
				assertUndeclared(t, nodeA, []string{"config"})
				// nodeB is also unused due to Go import cycle limitation
				assertUnused(t, nodeA, []string{"db", "nodeB"})
				assertCycles(t, nodeA, 1)

				// nodeB: unused + cycle
				// nodeA is also unused due to Go import cycle limitation
				assertUnused(t, nodeB, []string{"cache", "nodeA"})
				assertCycles(t, nodeB, 1)
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_Structural tests various graph structures and valid cases.
// These cases verify that the analyzer handles different graph topologies correctly,
// including minimal nodes, deep chains, disconnected graphs, etc.
func TestAnalyzeDirEdgeCases_Structural(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"12_empty_node": {
			dir:            "examples/edgecases/12_empty_node",
			wantNodes:      1,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				empty := findNode(t, results, "empty")
				assertDeps(t, empty, []string{}, []string{})
				assertNoCycles(t, empty)
			},
		},
		"13_no_deps_node": {
			dir:            "examples/edgecases/13_no_deps_node",
			wantNodes:      1,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				standalone := findNode(t, results, "standalone")
				assertDeps(t, standalone, []string{}, []string{})
				assertNoCycles(t, standalone)
			},
		},
		"17_long_chain": {
			dir:            "examples/edgecases/17_long_chain",
			wantNodes:      10,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				// Verify no cycles in any node
				for _, r := range results {
					assertNoCycles(t, r)
				}
				// Verify linear chain structure
				n1 := findNode(t, results, "n1")
				assertDeps(t, n1, []string{}, []string{})

				n10 := findNode(t, results, "n10")
				assertDeps(t, n10, []string{"n9"}, []string{"n9"})
			},
		},
		"19_orphan_nodes": {
			dir:            "examples/edgecases/19_orphan_nodes",
			wantNodes:      4,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []AnalysisResult) {
				// Two independent subgraphs, both should be valid
				for _, r := range results {
					if r.HasIssues() {
						t.Errorf("node %q should not have issues, got %s", r.NodeID, r.String())
					}
				}
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// edgeCaseTest defines a single edge case test.
type edgeCaseTest struct {
	dir            string
	wantNodes      int
	wantIssueCount int
	checkSpecific  func(t *testing.T, results []AnalysisResult)
}

// runEdgeCaseTests is the common test runner for all edge case tests.
// It handles the boilerplate of running AnalyzeDir, checking node counts,
// issue counts, and invoking custom validation functions.
func runEdgeCaseTests(t *testing.T, tests map[string]edgeCaseTest) {
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			absDir, err := filepath.Abs(tt.dir)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			results, err := AnalyzeDir(absDir)
			if err != nil {
				t.Fatalf("AnalyzeDir(%q) error: %v", tt.dir, err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", len(results), tt.wantNodes)
				for _, r := range results {
					t.Logf("  - %s", r.NodeID)
				}
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssueCount {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssueCount)
				for _, r := range results {
					if r.HasIssues() {
						t.Logf("  %s", r.String())
					}
				}
			}

			if tt.checkSpecific != nil {
				tt.checkSpecific(t, results)
			}
		})
	}
}

// findNode finds a node by ID in the results, failing the test if not found.
func findNode(t *testing.T, results []AnalysisResult, id string) AnalysisResult {
	t.Helper()
	for _, r := range results {
		if r.NodeID == id {
			return r
		}
	}
	t.Fatalf("node %q not found in results", id)
	return AnalysisResult{}
}

// assertUndeclared checks that undeclared dependencies exactly match the expected list.
func assertUndeclared(t *testing.T, r AnalysisResult, want []string) {
	t.Helper()
	if !equalStringSlices(r.Undeclared, want) {
		t.Errorf("node %q: undeclared = %v, want %v", r.NodeID, r.Undeclared, want)
	}
}

// assertUnused checks that unused dependencies exactly match the expected list.
func assertUnused(t *testing.T, r AnalysisResult, want []string) {
	t.Helper()
	if !equalStringSlices(r.Unused, want) {
		t.Errorf("node %q: unused = %v, want %v", r.NodeID, r.Unused, want)
	}
}

// assertNoCycles verifies that a node has no cycles.
func assertNoCycles(t *testing.T, r AnalysisResult) {
	t.Helper()
	if len(r.Cycles) > 0 {
		t.Errorf("node %q: expected no cycles, got %v", r.NodeID, r.Cycles)
	}
}

// assertCycles checks that a node has the expected number of cycles.
func assertCycles(t *testing.T, r AnalysisResult, wantCount int) {
	t.Helper()
	if len(r.Cycles) != wantCount {
		t.Errorf("node %q: got %d cycles, want %d; cycles: %v", r.NodeID, len(r.Cycles), wantCount, r.Cycles)
	}
}

// assertCycleContains verifies that a specific cycle path exists in the node's cycles.
func assertCycleContains(t *testing.T, r AnalysisResult, expectedPath []string) {
	t.Helper()
	for _, cycle := range r.Cycles {
		if equalStringSlices(cycle, expectedPath) {
			return
		}
	}
	t.Errorf("node %q: cycle %v not found in %v", r.NodeID, expectedPath, r.Cycles)
}

// assertDeps checks both declared and used dependencies.
func assertDeps(t *testing.T, r AnalysisResult, wantDeclared, wantUsed []string) {
	t.Helper()
	if !equalStringSlices(r.DeclaredDeps, wantDeclared) {
		t.Errorf("node %q: declared = %v, want %v", r.NodeID, r.DeclaredDeps, wantDeclared)
	}
	if !equalStringSlices(r.UsedDeps, wantUsed) {
		t.Errorf("node %q: used = %v, want %v", r.NodeID, r.UsedDeps, wantUsed)
	}
}

// assertDepsContain checks that dependencies contain at least the specified items.
func assertDepsContain(t *testing.T, r AnalysisResult, wantDeclared, wantUsed []string) {
	t.Helper()
	if !containsAll(r.DeclaredDeps, wantDeclared) {
		t.Errorf("node %q: declared %v does not contain all of %v", r.NodeID, r.DeclaredDeps, wantDeclared)
	}
	if !containsAll(r.UsedDeps, wantUsed) {
		t.Errorf("node %q: used %v does not contain all of %v", r.NodeID, r.UsedDeps, wantUsed)
	}
}

// assertUndeclaredContains checks that undeclared contains all specified items.
func assertUndeclaredContains(t *testing.T, r AnalysisResult, items []string) {
	t.Helper()
	if len(r.Undeclared) != len(items) {
		t.Errorf("node %q: undeclared has %d items, want %d; got %v", r.NodeID, len(r.Undeclared), len(items), r.Undeclared)
	}
	if !containsAll(r.Undeclared, items) {
		t.Errorf("node %q: undeclared %v does not contain all of %v", r.NodeID, r.Undeclared, items)
	}
}

// assertUnusedContains checks that unused contains all specified items.
func assertUnusedContains(t *testing.T, r AnalysisResult, items []string) {
	t.Helper()
	if len(r.Unused) != len(items) {
		t.Errorf("node %q: unused has %d items, want %d; got %v", r.NodeID, len(r.Unused), len(items), r.Unused)
	}
	if !containsAll(r.Unused, items) {
		t.Errorf("node %q: unused %v does not contain all of %v", r.NodeID, r.Unused, items)
	}
}

// equalStringSlices checks if two string slices contain the same elements,
// regardless of order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort copies to avoid modifying originals
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

// containsAll checks if slice contains all items.
func containsAll(slice, items []string) bool {
	sliceMap := make(map[string]bool)
	for _, s := range slice {
		sliceMap[s] = true
	}
	for _, item := range items {
		if !sliceMap[item] {
			return false
		}
	}
	return true
}
