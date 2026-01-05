package graft

import (
	"path/filepath"
	"testing"
)

// TestAnalyzeDirIntegration tests the type-aware analyzer on real example projects.
// These tests verify that the type-aware analysis correctly analyzes actual graft code.
func TestAnalyzeDirIntegration(t *testing.T) {
	type tc struct {
		dir        string
		wantNodes  int
		wantIssues int
		checkNodes func(t *testing.T, results []AnalysisResult)
	}

	tests := map[string]tc{
		"examples/simple": {
			dir:        "examples/simple",
			wantNodes:  3, // config, db, app
			wantIssues: 0,
			checkNodes: func(t *testing.T, results []AnalysisResult) {
				// Map results by node ID for easier checking
				nodeMap := make(map[string]AnalysisResult)
				for _, r := range results {
					nodeMap[r.NodeID] = r
				}

				// Verify config node
				if config, ok := nodeMap["config"]; ok {
					if len(config.DeclaredDeps) != 0 {
						t.Errorf("config: expected 0 declared deps, got %d", len(config.DeclaredDeps))
					}
					if len(config.UsedDeps) != 0 {
						t.Errorf("config: expected 0 used deps, got %d", len(config.UsedDeps))
					}
				} else {
					t.Error("config node not found")
				}

				// Verify db node
				if db, ok := nodeMap["db"]; ok {
					if len(db.DeclaredDeps) != 1 {
						t.Errorf("db: expected 1 declared dep, got %d", len(db.DeclaredDeps))
					}
					if len(db.UsedDeps) != 1 {
						t.Errorf("db: expected 1 used dep, got %d", len(db.UsedDeps))
					}
					if len(db.DeclaredDeps) > 0 && db.DeclaredDeps[0] != "config" {
						t.Errorf("db: expected declared dep 'config', got %q", db.DeclaredDeps[0])
					}
				} else {
					t.Error("db node not found")
				}

				// Verify app node
				if app, ok := nodeMap["app"]; ok {
					if len(app.DeclaredDeps) != 1 {
						t.Errorf("app: expected 1 declared dep, got %d", len(app.DeclaredDeps))
					}
					if len(app.UsedDeps) != 1 {
						t.Errorf("app: expected 1 used dep, got %d", len(app.UsedDeps))
					}
					if len(app.DeclaredDeps) > 0 && app.DeclaredDeps[0] != "db" {
						t.Errorf("app: expected declared dep 'db', got %q", app.DeclaredDeps[0])
					}
				} else {
					t.Error("app node not found")
				}
			},
		},
		"examples/complex": {
			dir:        "examples/complex",
			wantNodes:  9, // env, logger, secrets, auth, admin, cfg, db, user, gateway
			wantIssues: 0,
			checkNodes: func(t *testing.T, results []AnalysisResult) {
				// Verify all nodes have no issues
				for _, r := range results {
					if r.HasIssues() {
						t.Errorf("node %q has unexpected issues: %s", r.NodeID, r.String())
					}
				}

				// Verify we have the expected node IDs
				expectedIDs := map[string]bool{
					"env": true, "logger": true, "secrets": true,
					"auth": true, "admin": true, "cfg": true,
					"db": true, "user": true, "gateway": true,
				}
				for _, r := range results {
					if !expectedIDs[r.NodeID] {
						t.Errorf("unexpected node ID: %q", r.NodeID)
					}
					delete(expectedIDs, r.NodeID)
				}
				for id := range expectedIDs {
					t.Errorf("expected node ID %q not found", id)
				}
			},
		},
		"examples/diamond": {
			dir:        "examples/diamond",
			wantNodes:  4, // config, cache, db, api
			wantIssues: 0,
			checkNodes: func(t *testing.T, results []AnalysisResult) {
				nodeMap := make(map[string]AnalysisResult)
				for _, r := range results {
					nodeMap[r.NodeID] = r
					if r.HasIssues() {
						t.Errorf("node %q has unexpected issues: %s", r.NodeID, r.String())
					}
				}

				// Verify diamond structure: api depends on [cache, db], both depend on config
				if api, ok := nodeMap["api"]; ok {
					if len(api.DeclaredDeps) != 2 {
						t.Errorf("api: expected 2 declared deps, got %d", len(api.DeclaredDeps))
					}
					// Should declare cache and db
					depMap := make(map[string]bool)
					for _, d := range api.DeclaredDeps {
						depMap[d] = true
					}
					if !depMap["cache"] || !depMap["db"] {
						t.Errorf("api: expected deps [cache, db], got %v", api.DeclaredDeps)
					}
				}
			},
		},
		"examples/fanout": {
			dir:        "examples/fanout",
			wantNodes:  9, // config, shared1-2, svc1-5, aggregator
			wantIssues: 0,
			checkNodes: func(t *testing.T, results []AnalysisResult) {
				nodeMap := make(map[string]AnalysisResult)
				for _, r := range results {
					nodeMap[r.NodeID] = r
				}

				// Verify other nodes don't have cycle issues
				for id, r := range nodeMap {
					if len(r.Cycles) > 0 {
						t.Errorf("node %q should not have cycles, got %v", id, r.Cycles)
					}
				}

				// Verify fanout structure: aggregator depends on all services
				if aggregator, ok := nodeMap["aggregator"]; ok {
					if len(aggregator.DeclaredDeps) != 7 {
						t.Errorf("aggregator: expected 7 declared deps (svc1-5, shared1-2), got %d", len(aggregator.DeclaredDeps))
					}
				}

				// Verify each service depends on config (except svc5 which also depends on svc5-2)
				for i := 1; i <= 5; i++ {
					svcID := "svc" + string(rune('0'+i))
					if svc, ok := nodeMap[svcID]; ok {
						// Other services only depend on config
						if len(svc.DeclaredDeps) != 1 {
							t.Errorf("%s: expected 1 declared dep (config), got %d", svcID, len(svc.DeclaredDeps))
						}
					}
				}
			},
		},
		"examples/httpserver": {
			dir:        "examples/httpserver",
			wantNodes:  5, // config, request_logger, admin, db, user
			wantIssues: 0,
			checkNodes: func(t *testing.T, results []AnalysisResult) {
				nodeMap := make(map[string]AnalysisResult)
				for _, r := range results {
					nodeMap[r.NodeID] = r
					if r.HasIssues() {
						t.Errorf("node %q has unexpected issues: %s", r.NodeID, r.String())
					}
				}

				// Verify expected nodes exist
				expectedIDs := []string{"config", "request_logger", "admin", "db", "user"}
				for _, id := range expectedIDs {
					if _, ok := nodeMap[id]; !ok {
						t.Errorf("expected node %q not found", id)
					}
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Convert relative path to absolute
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

			// Run custom checks if provided
			if tt.checkNodes != nil {
				tt.checkNodes(t, results)
			}
		})
	}
}

// TestValidateDepsIntegration tests the ValidateDeps function on real examples.
func TestValidateDepsIntegration(t *testing.T) {
	examples := []string{
		"examples/simple",
		"examples/complex",
		"examples/diamond",
		// Note: examples/fanout is excluded as it contains an intentional cycle (svc5 ↔ svc5-2)
		"examples/httpserver",
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			absDir, err := filepath.Abs(example)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}

			err = ValidateDeps(absDir)
			if err != nil {
				t.Errorf("ValidateDeps(%q) = %v, want nil (no errors)", example, err)
			}
		})
	}
}

// TestAnalyzeDirEdgeCases tests edge cases with the type-aware analyzer.
func TestAnalyzeDirEdgeCases(t *testing.T) {
	t.Run("nonexistent_directory", func(t *testing.T) {
		results, err := AnalyzeDir("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("expected error for nonexistent directory, got nil")
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for nonexistent directory, got %d", len(results))
		}
	})

	t.Run("empty_directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		results, err := AnalyzeDir(tmpDir)
		// Empty directory without Go module will return an error from go/packages
		// This is expected behavior - type-aware analysis requires valid Go packages
		if err == nil {
			// If no error, should have 0 results
			if len(results) != 0 {
				t.Errorf("expected 0 nodes for empty directory, got %d", len(results))
			}
		}
		// Either error or 0 results is acceptable for empty directory
	})

	t.Run("current_directory", func(t *testing.T) {
		// Should analyze the graft package itself
		results, err := AnalyzeDir(".")
		if err != nil {
			t.Fatalf("AnalyzeDir(\".\") error: %v", err)
		}
		// The graft package has test nodes in test files, which should be excluded
		// We should get results from examples
		if len(results) == 0 {
			t.Error("expected some nodes from examples, got 0")
		}
	})
}

// TestAnalyzeDirDebug tests the debug flag functionality.
func TestAnalyzeDirDebug(t *testing.T) {
	// Save original value
	originalDebug := AnalyzeDirDebug
	defer func() { AnalyzeDirDebug = originalDebug }()

	t.Run("debug_enabled", func(t *testing.T) {
		AnalyzeDirDebug = true
		absDir, err := filepath.Abs("examples/simple")
		if err != nil {
			t.Fatalf("failed to get absolute path: %v", err)
		}

		results, err := AnalyzeDir(absDir)
		if err != nil {
			t.Fatalf("AnalyzeDir error: %v", err)
		}

		// Should still work with debug enabled
		if len(results) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(results))
		}
	})

	t.Run("debug_disabled", func(t *testing.T) {
		AnalyzeDirDebug = false
		absDir, err := filepath.Abs("examples/simple")
		if err != nil {
			t.Fatalf("failed to get absolute path: %v", err)
		}

		results, err := AnalyzeDir(absDir)
		if err != nil {
			t.Fatalf("AnalyzeDir error: %v", err)
		}

		// Should work the same with debug disabled
		if len(results) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(results))
		}
	})
}

// TestTypeAwareAnalyzerAccuracy tests specific scenarios where type-aware
// analysis is more accurate than AST-based analysis.
func TestTypeAwareAnalyzerAccuracy(t *testing.T) {
	t.Run("handles_type_aliases", func(t *testing.T) {
		// The examples use type aliases for Output types
		// Type-aware analysis should resolve these correctly
		absDir, err := filepath.Abs("examples/simple")
		if err != nil {
			t.Fatalf("failed to get absolute path: %v", err)
		}

		results, err := AnalyzeDir(absDir)
		if err != nil {
			t.Fatalf("AnalyzeDir error: %v", err)
		}

		// Verify no false positives from type alias confusion
		for _, r := range results {
			if r.HasIssues() {
				t.Errorf("node %q has issues (type alias problem?): %s", r.NodeID, r.String())
			}
		}
	})

	t.Run("resolves_package_imports", func(t *testing.T) {
		// Examples use qualified type references like config.Output
		// Type-aware analysis should resolve these to the correct node IDs
		absDir, err := filepath.Abs("examples/simple")
		if err != nil {
			t.Fatalf("failed to get absolute path: %v", err)
		}

		results, err := AnalyzeDir(absDir)
		if err != nil {
			t.Fatalf("AnalyzeDir error: %v", err)
		}

		// Find the db node
		var dbNode *AnalysisResult
		for i := range results {
			if results[i].NodeID == "db" {
				dbNode = &results[i]
				break
			}
		}

		if dbNode == nil {
			t.Fatal("db node not found in results")
		}

		// db node should correctly resolve config.Output to "config"
		if len(dbNode.DeclaredDeps) != 1 || dbNode.DeclaredDeps[0] != "config" {
			t.Errorf("expected declared dep 'config', got %v", dbNode.DeclaredDeps)
		}
		if len(dbNode.UsedDeps) != 1 || dbNode.UsedDeps[0] != "config" {
			t.Errorf("expected used dep 'config', got %v", dbNode.UsedDeps)
		}
	})
}
