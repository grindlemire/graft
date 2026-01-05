package graft

import (
	"path/filepath"
	"testing"

	"github.com/grindlemire/graft/internal/typeaware"
)

// TestAnalyzeDirIntegration tests the type-aware analyzer on real example projects.
func TestAnalyzeDirIntegration(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		// Expected node-specific checks (nodeID -> expectations)
		nodeChecks map[string]struct {
			declaredDeps []string
			usedDeps     []string
		}
		// Expected node IDs to verify presence
		wantNodeIDs []string
	}

	tests := map[string]tc{
		"examples/simple": {
			dir:        "examples/simple",
			wantNodes:  3,
			wantIssues: 0,
			nodeChecks: map[string]struct {
				declaredDeps []string
				usedDeps     []string
			}{
				"config": {declaredDeps: []string{}, usedDeps: []string{}},
				"db":     {declaredDeps: []string{"config"}, usedDeps: []string{"config"}},
				"app":    {declaredDeps: []string{"db"}, usedDeps: []string{"db"}},
			},
		},
		"examples/complex": {
			dir:        "examples/complex",
			wantNodes:  9,
			wantIssues: 0,
			wantNodeIDs: []string{
				"env", "logger", "secrets", "auth", "admin",
				"cfg", "db", "user", "gateway",
			},
		},
		"examples/diamond": {
			dir:        "examples/diamond",
			wantNodes:  4,
			wantIssues: 0,
			nodeChecks: map[string]struct {
				declaredDeps []string
				usedDeps     []string
			}{
				"config": {declaredDeps: []string{}, usedDeps: []string{}},
				"cache":  {declaredDeps: []string{"config"}, usedDeps: []string{"config"}},
				"db":     {declaredDeps: []string{"config"}, usedDeps: []string{"config"}},
				// api depends on both cache and db (diamond pattern)
			},
		},
		"examples/fanout": {
			dir:        "examples/fanout",
			wantNodes:  7,
			wantIssues: 0,
			wantNodeIDs: []string{
				"config", "svc1", "svc2", "svc3", "svc4", "svc5", "aggregator",
			},
		},
		"examples/httpserver": {
			dir:        "examples/httpserver",
			wantNodes:  5,
			wantIssues: 0,
			wantNodeIDs: []string{
				"config", "request_logger", "admin", "db", "user",
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

			// Check node count
			if len(results) != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", len(results), tt.wantNodes)
				for _, r := range results {
					t.Logf("  - %s", r.NodeID)
				}
			}

			// Check issues count
			issueCount := 0
			for _, r := range results {
				if r.HasIssues() {
					issueCount++
					t.Logf("Node %q has issues: %s", r.NodeID, r.String())
				}
			}
			if issueCount != tt.wantIssues {
				t.Errorf("got %d nodes with issues, want %d", issueCount, tt.wantIssues)
			}

			// Map results by node ID for checking
			nodeMap := make(map[string]typeaware.Result)
			for _, r := range results {
				nodeMap[r.NodeID] = r
			}

			// Check specific node dependencies if specified
			for nodeID, expected := range tt.nodeChecks {
				node, ok := nodeMap[nodeID]
				if !ok {
					t.Errorf("node %q not found in results", nodeID)
					continue
				}

				if !equalStringSlices(node.DeclaredDeps, expected.declaredDeps) {
					t.Errorf("%s: declared deps = %v, want %v",
						nodeID, node.DeclaredDeps, expected.declaredDeps)
				}

				if !equalStringSlices(node.UsedDeps, expected.usedDeps) {
					t.Errorf("%s: used deps = %v, want %v",
						nodeID, node.UsedDeps, expected.usedDeps)
				}
			}

			// Check expected node IDs if specified
			if len(tt.wantNodeIDs) > 0 {
				for _, id := range tt.wantNodeIDs {
					if _, ok := nodeMap[id]; !ok {
						t.Errorf("expected node %q not found", id)
					}
				}
			}
		})
	}
}

// TestValidateDepsIntegration tests the ValidateDeps function on real examples.
func TestValidateDepsIntegration(t *testing.T) {
	tests := map[string]struct {
		dir string
	}{
		"examples/simple":     {dir: "examples/simple"},
		"examples/complex":    {dir: "examples/complex"},
		"examples/diamond":    {dir: "examples/diamond"},
		"examples/httpserver": {dir: "examples/httpserver"},
		// Note: examples/fanout excluded (intentional cycle with svc5 ↔ svc5-2)
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			absDir, err := filepath.Abs(tt.dir)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			err = ValidateDeps(absDir)
			if err != nil {
				t.Errorf("ValidateDeps(%q) = %v, want nil", tt.dir, err)
			}
		})
	}
}

// TestAnalyzeDirErrors tests error handling for edge cases.
func TestAnalyzeDirErrors(t *testing.T) {
	tests := map[string]struct {
		dir         string
		wantErr     bool
		wantResults int
	}{
		"nonexistent_directory": {
			dir:         "/nonexistent/path/that/does/not/exist",
			wantErr:     true,
			wantResults: 0,
		},
		"empty_directory": {
			dir:         t.TempDir(),
			wantErr:     true, // Type-aware analysis requires valid Go packages
			wantResults: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := AnalyzeDir(tt.dir)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tt.wantResults {
				t.Errorf("got %d results, want %d", len(results), tt.wantResults)
			}
		})
	}
}

// TestAnalyzeDirDebug tests the debug flag functionality.
func TestAnalyzeDirDebug(t *testing.T) {
	// Save original value
	originalDebug := AnalyzeDirDebug
	defer func() { AnalyzeDirDebug = originalDebug }()

	tests := map[string]struct {
		debugFlag bool
		wantNodes int
	}{
		"debug_enabled":  {debugFlag: true, wantNodes: 3},
		"debug_disabled": {debugFlag: false, wantNodes: 3},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			AnalyzeDirDebug = tt.debugFlag

			absDir, err := filepath.Abs("examples/simple")
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			results, err := AnalyzeDir(absDir)
			if err != nil {
				t.Fatalf("AnalyzeDir error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("expected %d nodes, got %d", tt.wantNodes, len(results))
			}
		})
	}
}

// TestTypeAwareAnalyzerAccuracy tests specific scenarios where type-aware
// analysis is more accurate than AST-based analysis.
func TestTypeAwareAnalyzerAccuracy(t *testing.T) {
	tests := map[string]struct {
		dir          string
		wantNodes    int
		checkTypeRes func(t *testing.T, results []typeaware.Result)
	}{
		"handles_type_aliases": {
			dir:       "examples/simple",
			wantNodes: 3,
			checkTypeRes: func(t *testing.T, results []typeaware.Result) {
				// Type-aware analysis should resolve type aliases correctly
				for _, r := range results {
					if r.HasIssues() {
						t.Errorf("node %q has issues (type alias problem?): %s",
							r.NodeID, r.String())
					}
				}
			},
		},
		"resolves_package_imports": {
			dir:       "examples/simple",
			wantNodes: 3,
			checkTypeRes: func(t *testing.T, results []typeaware.Result) {
				// Find db node and verify it correctly resolves config.Output to "config"
				var dbNode *typeaware.Result
				for i := range results {
					if results[i].NodeID == "db" {
						dbNode = &results[i]
						break
					}
				}

				if dbNode == nil {
					t.Fatal("db node not found in results")
				}

				if len(dbNode.DeclaredDeps) != 1 || dbNode.DeclaredDeps[0] != "config" {
					t.Errorf("expected declared dep 'config', got %v", dbNode.DeclaredDeps)
				}

				if len(dbNode.UsedDeps) != 1 || dbNode.UsedDeps[0] != "config" {
					t.Errorf("expected used dep 'config', got %v", dbNode.UsedDeps)
				}
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
				t.Fatalf("AnalyzeDir error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", len(results), tt.wantNodes)
			}

			if tt.checkTypeRes != nil {
				tt.checkTypeRes(t, results)
			}
		})
	}
}
