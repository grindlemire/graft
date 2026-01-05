package graft

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/grindlemire/graft/internal/typeaware"
)

// TestAnalyzeDirEdgeCases_TypeVariations tests that the analyzer correctly handles
// various output types beyond structs, including primitives, slices, and type aliases.
func TestAnalyzeDirEdgeCases_TypeVariations(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"primitive_string": {
			dir:            "examples/edgecases/primitive_string",
			wantNodes:      2,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				producer := findNode(t, results, "str")
				consumer := findNode(t, results, "app")
				assertNoCycles(t, producer)
				assertNoCycles(t, consumer)
				assertDeps(t, producer, []string{}, []string{})
				assertDeps(t, consumer, []string{"str"}, []string{"str"})
			},
		},
		"primitive_int": {
			dir:            "examples/edgecases/primitive_int",
			wantNodes:      2,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				producer := findNode(t, results, "port")
				consumer := findNode(t, results, "server")
				assertNoCycles(t, producer)
				assertNoCycles(t, consumer)
				assertDeps(t, producer, []string{}, []string{})
				assertDeps(t, consumer, []string{"port"}, []string{"port"})
			},
		},
		"slice_type": {
			dir:            "examples/edgecases/slice_type",
			wantNodes:      2,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				producer := findNode(t, results, "tags")
				consumer := findNode(t, results, "app")
				assertNoCycles(t, producer)
				assertNoCycles(t, consumer)
				assertDeps(t, producer, []string{}, []string{})
				assertDeps(t, consumer, []string{"tags"}, []string{"tags"})
			},
		},
		"type_alias_match": {
			dir:            "examples/edgecases/type_alias_match",
			wantNodes:      2,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				producer := findNode(t, results, "uid")
				consumer := findNode(t, results, "app")
				// Type alias should resolve correctly
				assertNoCycles(t, producer)
				assertNoCycles(t, consumer)
				assertDeps(t, consumer, []string{"uid"}, []string{"uid"})
			},
		},
		"aliased_import": {
			dir:            "examples/edgecases/aliased_import",
			wantNodes:      2,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				producer := findNode(t, results, "config")
				consumer := findNode(t, results, "app")
				// Import alias should not affect type resolution
				// Producer imports shared normally, consumer uses aliased import
				assertNoCycles(t, producer)
				assertNoCycles(t, consumer)
				assertDeps(t, consumer, []string{"config"}, []string{"config"})
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_SamePackage tests that the analyzer works correctly
// when nodes, output types, and dependencies are all in the same package.
func TestAnalyzeDirEdgeCases_SamePackage(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"same_package_deps": {
			dir:            "examples/edgecases/same_package_deps",
			wantNodes:      3,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				config := findNode(t, results, "config")
				db := findNode(t, results, "db")
				app := findNode(t, results, "app")

				// Verify dependency chain: config -> db -> app
				assertDeps(t, config, []string{}, []string{})
				assertDeps(t, db, []string{"config"}, []string{"config"})
				assertDeps(t, app, []string{"db"}, []string{"db"})

				assertNoCycles(t, config)
				assertNoCycles(t, db)
				assertNoCycles(t, app)
			},
		},
		"multiple_nodes_same_file": {
			dir:            "examples/edgecases/multiple_nodes_same_file",
			wantNodes:      3,
			wantIssueCount: 0,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				// All 3 nodes in single file should be discovered
				nodeA := findNode(t, results, "nodeA")
				nodeB := findNode(t, results, "nodeB")
				nodeC := findNode(t, results, "nodeC")

				assertDeps(t, nodeA, []string{}, []string{})
				assertDeps(t, nodeB, []string{"nodeA"}, []string{"nodeA"})
				assertDeps(t, nodeC, []string{"nodeB"}, []string{"nodeB"})

				assertNoCycles(t, nodeA)
				assertNoCycles(t, nodeB)
				assertNoCycles(t, nodeC)
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_TypeMismatches tests that the analyzer correctly
// detects when types don't match, particularly pointer vs non-pointer and
// named types vs their underlying types.
func TestAnalyzeDirEdgeCases_TypeMismatches(t *testing.T) {
	tests := map[string]edgeCaseTest{
		"pointer_mismatch": {
			dir:            "examples/edgecases/pointer_mismatch",
			wantNodes:      2,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				consumer := findNode(t, results, "consumer")
				// Producer outputs *Config, consumer declares "config" but tries Dep[Config]
				// Dep[Config] won't match *Config, so "config" is unused
				assertUnused(t, consumer, []string{"config"})
				assertNoCycles(t, consumer)
			},
		},
		"named_type_no_match": {
			dir:            "examples/edgecases/named_type_no_match",
			wantNodes:      2,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				consumer := findNode(t, results, "consumer")
				// Producer outputs Port, consumer declares "port" but tries Dep[int]
				// Dep[int] won't match Port (distinct types), so "port" is unused
				assertUnused(t, consumer, []string{"port"})
				assertNoCycles(t, consumer)
			},
		},
		"pointer_direction_mismatch": {
			dir:            "examples/edgecases/pointer_direction_mismatch",
			wantNodes:      2,
			wantIssueCount: 1,
			checkSpecific: func(t *testing.T, results []typeaware.Result) {
				consumer := findNode(t, results, "consumer")
				// Producer outputs Config, consumer declares "config" but tries Dep[*Config]
				// Dep[*Config] won't match Config, so "config" is unused
				assertUnused(t, consumer, []string{"config"})
				assertNoCycles(t, consumer)
			},
		},
	}

	runEdgeCaseTests(t, tests)
}

// TestAnalyzeDirEdgeCases_TypeConflicts tests that the analyzer detects when
// two nodes register the same output type, which creates ambiguity.
func TestAnalyzeDirEdgeCases_TypeConflicts(t *testing.T) {
	absDir, err := filepath.Abs("examples/edgecases/type_conflict_detected")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	results, err := AnalyzeDir(absDir)

	// Should get an error about type conflict
	if err == nil {
		t.Fatalf("expected type conflict error, got nil")
	}

	if !strings.Contains(err.Error(), "type conflict") {
		t.Errorf("expected 'type conflict' in error, got: %v", err)
	}

	// Results may be partial or nil when error occurs
	_ = results
}
