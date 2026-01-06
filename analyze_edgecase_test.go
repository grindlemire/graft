package graft

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/grindlemire/graft/internal/typeaware"
)

// TestAnalyzeDirUndeclared tests detection of undeclared dependencies.
func TestAnalyzeDirUndeclared(t *testing.T) {
	type tc struct {
		dir            string
		wantNodes      int
		wantIssues     int
		wantUndeclared map[string][]string // nodeID -> expected undeclared deps
		wantUnused     map[string][]string // nodeID -> expected unused deps (should be empty)
	}

	tests := map[string]tc{
		"undeclared_multiple": {
			dir:        "examples/edgecases/undeclared_multiple",
			wantNodes:  4,
			wantIssues: 1,
			wantUndeclared: map[string][]string{
				"app": {"config", "db", "cache"},
			},
			wantUnused: map[string][]string{
				"app": {},
			},
		},
		"partial_declaration": {
			dir:        "examples/edgecases/partial_declaration",
			wantNodes:  4,
			wantIssues: 1,
			wantUndeclared: map[string][]string{
				"app": {"cache"},
			},
			wantUnused: map[string][]string{
				"app": {},
			},
		},
		"conditional_dep_usage": {
			dir:        "examples/edgecases/conditional_dep_usage",
			wantNodes:  3,
			wantIssues: 1,
			wantUndeclared: map[string][]string{
				"app": {"feature"},
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Check undeclared dependencies
			for nodeID, want := range tt.wantUndeclared {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Undeclared, want) {
					t.Errorf("%s: undeclared = %v, want %v", nodeID, node.Undeclared, want)
				}
			}

			// Check unused dependencies
			for nodeID, want := range tt.wantUnused {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Unused, want) {
					t.Errorf("%s: unused = %v, want %v", nodeID, node.Unused, want)
				}
			}
		})
	}
}

// TestAnalyzeDirUnused tests detection of unused dependencies.
func TestAnalyzeDirUnused(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		wantUnused map[string][]string // nodeID -> expected unused deps
	}

	tests := map[string]tc{
		"unused_multiple": {
			dir:        "examples/edgecases/unused_multiple",
			wantNodes:  4,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"app": {"config", "db", "cache"},
			},
		},
		"unused_in_chain": {
			dir:        "examples/edgecases/unused_in_chain",
			wantNodes:  4,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"middleware": {"db"},
			},
		},
		"complex_multi_parent": {
			dir:        "examples/edgecases/complex_multi_parent",
			wantNodes:  5,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"aggregator": {"serviceC"},
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Check unused dependencies
			for nodeID, want := range tt.wantUnused {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Unused, want) {
					t.Errorf("%s: unused = %v, want %v", nodeID, node.Unused, want)
				}
			}
		})
	}
}

// TestAnalyzeDirCycles tests cycle detection via DFS.
func TestAnalyzeDirCycles(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		wantCycles map[string]int      // nodeID -> expected cycle count
		cyclePaths map[string][]string // nodeID -> expected cycle path
	}

	tests := map[string]tc{
		"cycle_simple": {
			dir:        "examples/edgecases/cycle_simple",
			wantNodes:  2,
			wantIssues: 2,
			wantCycles: map[string]int{
				"nodeA": 1,
				"nodeB": 1,
			},
			cyclePaths: map[string][]string{
				"nodeA": {"nodeA", "nodeB", "nodeA"},
				"nodeB": {"nodeA", "nodeB", "nodeA"},
			},
		},
		"cycle_triangle": {
			dir:        "examples/edgecases/cycle_triangle",
			wantNodes:  3,
			wantIssues: 3,
			wantCycles: map[string]int{
				"nodeA": 1,
				"nodeB": 1,
				"nodeC": 1,
			},
		},
		"cycle_deep": {
			dir:        "examples/edgecases/cycle_deep",
			wantNodes:  5,
			wantIssues: 3,
			wantCycles: map[string]int{
				"nodeA": 0,
				"nodeB": 0,
				"nodeC": 1,
				"nodeD": 1,
				"nodeE": 1,
			},
			cyclePaths: map[string][]string{
				"nodeC": {"nodeC", "nodeD", "nodeE", "nodeC"},
			},
		},
		"cycle_self": {
			dir:        "examples/edgecases/cycle_self",
			wantNodes:  1,
			wantIssues: 1,
			wantCycles: map[string]int{
				"nodeA": 1,
			},
			cyclePaths: map[string][]string{
				"nodeA": {"nodeA", "nodeA"},
			},
		},
		"multiple_cycles_same_node": {
			dir:        "examples/edgecases/multiple_cycles_same_node",
			wantNodes:  3,
			wantIssues: 3,
			wantCycles: map[string]int{
				"hub": 2,
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Check cycle counts
			for nodeID, wantCount := range tt.wantCycles {
				node := findNode(results, nodeID)
				if len(node.Cycles) != wantCount {
					t.Errorf("%s: got %d cycles, want %d; cycles: %v",
						nodeID, len(node.Cycles), wantCount, node.Cycles)
				}
			}

			// Check specific cycle paths
			for nodeID, wantPath := range tt.cyclePaths {
				node := findNode(results, nodeID)
				found := false
				normalizedWant := normalizeCyclePath(wantPath)
				for _, cycle := range node.Cycles {
					normalizedActual := normalizeCyclePath(cycle)
					if isCycleRotation(normalizedWant, normalizedActual) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: cycle path %v not found in %v", nodeID, wantPath, node.Cycles)
				}
			}
		})
	}
}

// TestAnalyzeDirMixed tests combinations of multiple issue types.
func TestAnalyzeDirMixed(t *testing.T) {
	type tc struct {
		dir            string
		wantNodes      int
		wantIssues     int
		wantUndeclared map[string][]string
		wantUnused     map[string][]string
		wantCycles     map[string]int
	}

	tests := map[string]tc{
		"mixed_undeclared_unused": {
			dir:        "examples/edgecases/mixed_undeclared_unused",
			wantNodes:  4,
			wantIssues: 1,
			wantUndeclared: map[string][]string{
				"app": {"cache"},
			},
			wantUnused: map[string][]string{
				"app": {"config", "db"},
			},
			wantCycles: map[string]int{
				"app": 0,
			},
		},
		"mixed_cycle_undeclared": {
			dir:        "examples/edgecases/mixed_cycle_undeclared",
			wantNodes:  3,
			wantIssues: 2,
			wantUndeclared: map[string][]string{
				"nodeA": {"config"},
			},
			wantCycles: map[string]int{
				"nodeA": 1,
				"nodeB": 1,
			},
		},
		"mixed_all_issues": {
			dir:        "examples/edgecases/mixed_all_issues",
			wantNodes:  5,
			wantIssues: 2,
			wantUndeclared: map[string][]string{
				"nodeA": {"config"},
			},
			wantUnused: map[string][]string{
				"nodeA": {"db", "nodeB"},
				"nodeB": {"cache", "nodeA"},
			},
			wantCycles: map[string]int{
				"nodeA": 1,
				"nodeB": 1,
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Check undeclared
			for nodeID, want := range tt.wantUndeclared {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Undeclared, want) {
					t.Errorf("%s: undeclared = %v, want %v", nodeID, node.Undeclared, want)
				}
			}

			// Check unused
			for nodeID, want := range tt.wantUnused {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Unused, want) {
					t.Errorf("%s: unused = %v, want %v", nodeID, node.Unused, want)
				}
			}

			// Check cycles
			for nodeID, wantCount := range tt.wantCycles {
				node := findNode(results, nodeID)
				if len(node.Cycles) != wantCount {
					t.Errorf("%s: got %d cycles, want %d", nodeID, len(node.Cycles), wantCount)
				}
			}
		})
	}
}

// TestAnalyzeDirStructural tests various graph structures and valid cases.
func TestAnalyzeDirStructural(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		// For validation: map of nodeID -> expected deps
		wantDeps map[string]struct {
			declared []string
			used     []string
		}
	}

	tests := map[string]tc{
		"empty_node": {
			dir:        "examples/edgecases/empty_node",
			wantNodes:  1,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"empty": {declared: []string{}, used: []string{}},
			},
		},
		"no_deps_node": {
			dir:        "examples/edgecases/no_deps_node",
			wantNodes:  1,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"standalone": {declared: []string{}, used: []string{}},
			},
		},
		"long_chain": {
			dir:        "examples/edgecases/long_chain",
			wantNodes:  10,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"n1":  {declared: []string{}, used: []string{}},
				"n10": {declared: []string{"n9"}, used: []string{"n9"}},
			},
		},
		"orphan_nodes": {
			dir:        "examples/edgecases/orphan_nodes",
			wantNodes:  4,
			wantIssues: 0,
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Verify no cycles in all nodes
			for _, r := range results {
				if len(r.Cycles) > 0 {
					t.Errorf("%s: expected no cycles, got %v", r.NodeID, r.Cycles)
				}
			}

			// Check specific dependencies if provided
			for nodeID, expected := range tt.wantDeps {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.DeclaredDeps, expected.declared) {
					t.Errorf("%s: declared = %v, want %v",
						nodeID, node.DeclaredDeps, expected.declared)
				}
				if !equalStringSlices(node.UsedDeps, expected.used) {
					t.Errorf("%s: used = %v, want %v",
						nodeID, node.UsedDeps, expected.used)
				}
			}
		})
	}
}

// TestAnalyzeDirTypeVariations tests various output types beyond structs.
func TestAnalyzeDirTypeVariations(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		wantDeps   map[string]struct {
			declared []string
			used     []string
		}
	}

	tests := map[string]tc{
		"primitive_string": {
			dir:        "examples/edgecases/primitive_string",
			wantNodes:  2,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"str": {declared: []string{}, used: []string{}},
				"app": {declared: []string{"str"}, used: []string{"str"}},
			},
		},
		"primitive_int": {
			dir:        "examples/edgecases/primitive_int",
			wantNodes:  2,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"port":   {declared: []string{}, used: []string{}},
				"server": {declared: []string{"port"}, used: []string{"port"}},
			},
		},
		"slice_type": {
			dir:        "examples/edgecases/slice_type",
			wantNodes:  2,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"tags": {declared: []string{}, used: []string{}},
				"app":  {declared: []string{"tags"}, used: []string{"tags"}},
			},
		},
		"type_alias_match": {
			dir:        "examples/edgecases/type_alias_match",
			wantNodes:  2,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"uid": {declared: []string{}, used: []string{}},
				"app": {declared: []string{"uid"}, used: []string{"uid"}},
			},
		},
		"aliased_import": {
			dir:        "examples/edgecases/aliased_import",
			wantNodes:  2,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"config": {declared: []string{}, used: []string{}},
				"app":    {declared: []string{"config"}, used: []string{"config"}},
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Verify no cycles
			for _, r := range results {
				if len(r.Cycles) > 0 {
					t.Errorf("%s: expected no cycles, got %v", r.NodeID, r.Cycles)
				}
			}

			// Check dependencies
			for nodeID, expected := range tt.wantDeps {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.DeclaredDeps, expected.declared) {
					t.Errorf("%s: declared = %v, want %v",
						nodeID, node.DeclaredDeps, expected.declared)
				}
				if !equalStringSlices(node.UsedDeps, expected.used) {
					t.Errorf("%s: used = %v, want %v",
						nodeID, node.UsedDeps, expected.used)
				}
			}
		})
	}
}

// TestAnalyzeDirSamePackage tests same-package dependencies.
func TestAnalyzeDirSamePackage(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		wantDeps   map[string]struct {
			declared []string
			used     []string
		}
	}

	tests := map[string]tc{
		"same_package_deps": {
			dir:        "examples/edgecases/same_package_deps",
			wantNodes:  3,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"config": {declared: []string{}, used: []string{}},
				"db":     {declared: []string{"config"}, used: []string{"config"}},
				"app":    {declared: []string{"db"}, used: []string{"db"}},
			},
		},
		"multiple_nodes_same_file": {
			dir:        "examples/edgecases/multiple_nodes_same_file",
			wantNodes:  3,
			wantIssues: 0,
			wantDeps: map[string]struct {
				declared []string
				used     []string
			}{
				"nodeA": {declared: []string{}, used: []string{}},
				"nodeB": {declared: []string{"nodeA"}, used: []string{"nodeA"}},
				"nodeC": {declared: []string{"nodeB"}, used: []string{"nodeB"}},
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Verify no cycles
			for _, r := range results {
				if len(r.Cycles) > 0 {
					t.Errorf("%s: expected no cycles, got %v", r.NodeID, r.Cycles)
				}
			}

			// Check dependencies
			for nodeID, expected := range tt.wantDeps {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.DeclaredDeps, expected.declared) {
					t.Errorf("%s: declared = %v, want %v",
						nodeID, node.DeclaredDeps, expected.declared)
				}
				if !equalStringSlices(node.UsedDeps, expected.used) {
					t.Errorf("%s: used = %v, want %v",
						nodeID, node.UsedDeps, expected.used)
				}
			}
		})
	}
}

// TestAnalyzeDirTypeMismatches tests pointer and type mismatch detection.
func TestAnalyzeDirTypeMismatches(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		wantUnused map[string][]string
	}

	tests := map[string]tc{
		"pointer_mismatch": {
			dir:        "examples/edgecases/pointer_mismatch",
			wantNodes:  2,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"consumer": {"config"},
			},
		},
		"named_type_no_match": {
			dir:        "examples/edgecases/named_type_no_match",
			wantNodes:  2,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"consumer": {"port"},
			},
		},
		"pointer_direction_mismatch": {
			dir:        "examples/edgecases/pointer_direction_mismatch",
			wantNodes:  2,
			wantIssues: 1,
			wantUnused: map[string][]string{
				"consumer": {"config"},
			},
		},
	}

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
			}

			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Verify no cycles
			for _, r := range results {
				if len(r.Cycles) > 0 {
					t.Errorf("%s: expected no cycles, got %v", r.NodeID, r.Cycles)
				}
			}

			// Check unused dependencies
			for nodeID, want := range tt.wantUnused {
				node := findNode(results, nodeID)
				if !equalStringSlices(node.Unused, want) {
					t.Errorf("%s: unused = %v, want %v", nodeID, node.Unused, want)
				}
			}
		})
	}
}

// TestAnalyzeDirTypeConflicts tests type conflict detection.
func TestAnalyzeDirTypeConflicts(t *testing.T) {
	absDir, err := filepath.Abs("examples/edgecases/type_conflict_detected")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	_, err = AnalyzeDir(absDir)

	if err == nil {
		t.Fatal("expected type conflict error, got nil")
	}

	if !strings.Contains(err.Error(), "type conflict") {
		t.Errorf("expected 'type conflict' in error, got: %v", err)
	}
}

// Helper functions

// findNode finds a node by ID in the results.
func findNode(results []typeaware.Result, id string) typeaware.Result {
	for _, r := range results {
		if r.NodeID == id {
			return r
		}
	}
	// Return empty result if not found - test will fail on checks
	return typeaware.Result{NodeID: id}
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

// normalizeCyclePath removes the duplicate last element from a cycle path.
// [A B C A] -> [A B C]
func normalizeCyclePath(path []string) []string {
	if len(path) <= 1 {
		return path
	}
	// If first and last are the same, remove last
	if path[0] == path[len(path)-1] {
		return path[:len(path)-1]
	}
	return path
}

// isCycleRotation checks if two cycles are rotations of each other.
// [A B C] and [B C A] are rotations of each other.
func isCycleRotation(cycle1, cycle2 []string) bool {
	if len(cycle1) != len(cycle2) {
		return false
	}
	if len(cycle1) == 0 {
		return true
	}

	// Try all rotations of cycle1 to see if any match cycle2
	for offset := 0; offset < len(cycle1); offset++ {
		match := true
		for i := 0; i < len(cycle1); i++ {
			if cycle1[(i+offset)%len(cycle1)] != cycle2[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
