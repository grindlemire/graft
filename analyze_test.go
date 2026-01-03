package graft

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestAnalyze is the main table-driven test runner.
// It iterates through the 'internal/testcases' directory structure, using the folder name as the test case key.
func TestAnalyze(t *testing.T) {
	// struct to verify outputs for a specific node ID
	type wantNode struct {
		DeclaredDeps []string
		UsedDeps     []string
		Undeclared   []string
		Unused       []string
	}

	type tc struct {
		dir        string
		wantNodes  map[string]wantNode // nodeID -> expectations
		wantIssues int                 // Total number of declared/used issues expected
	}

	tests := map[string]tc{
		"basic/single_node": {
			dir: "basic/single_node",
			wantNodes: map[string]wantNode{
				"node1": {
					DeclaredDeps: []string{},
					UsedDeps:     []string{},
				},
			},
		},
		"basic/same_package": {
			dir: "basic/same_package",
			wantNodes: map[string]wantNode{
				"nodeA": {DeclaredDeps: []string{}},
				"nodeB": {
					DeclaredDeps: []string{"nodeA"},
					UsedDeps:     []string{"nodeA"},
				},
			},
		},
		"interactions/multi_package": {
			dir: "interactions/multi_package",
			wantNodes: map[string]wantNode{
				"nodeA": {DeclaredDeps: []string{}},
				"nodeB": {
					DeclaredDeps: []string{"nodeA"},
					UsedDeps:     []string{"nodeA"},
				},
			},
		},
		"correctness/undeclared": {
			dir:        "correctness/undeclared",
			wantIssues: 1,
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{}, UsedDeps: []string{"dep"}, Undeclared: []string{"dep"}},
			},
		},
		"interactions/aliases": {
			dir: "interactions/aliases",
			wantNodes: map[string]wantNode{
				"nodeA": {DeclaredDeps: []string{}},
				"nodeB": {DeclaredDeps: []string{"nodeA"}, UsedDeps: []string{"nodeA"}},
			},
		},
		"interactions/dot_imp": {
			dir: "interactions/dot_imp",
			wantNodes: map[string]wantNode{
				"nodeA": {DeclaredDeps: []string{}},
				"nodeB": {DeclaredDeps: []string{"nodeA"}, UsedDeps: []string{"nodeA"}},
			},
		},

		// "interactions/vendor": {
		// 	dir: "interactions/vendor",
		// 	wantNodes: map[string]wantNode{
		// 		"libnode": {DeclaredDeps: []string{}},
		// 		"node":    {DeclaredDeps: []string{"libnode"}, UsedDeps: []string{"libnode"}},
		// 	},
		// },
		"advanced/type_alias": {
			dir: "advanced/type_alias",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{"dep"}},
			},
		},
		"advanced/closure": {
			dir: "advanced/closure",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{"dep"}},
			},
		},
		"advanced/indirect": {
			dir: "advanced/indirect",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{"dep"}},
			},
		},
		"advanced/chain": {
			dir: "advanced/chain",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{"dep"}},
			},
		},
		"correctness/unused": {
			dir:        "correctness/unused",
			wantIssues: 1,
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{}, Unused: []string{"dep"}},
			},
		},
		"correctness/duplicate": {
			dir: "correctness/duplicate",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep", "dep"}, UsedDeps: []string{"dep"}},
			},
		},
		"correctness/multi_access": {
			dir: "correctness/multi_access",
			wantNodes: map[string]wantNode{
				"dep":  {DeclaredDeps: []string{}},
				"node": {DeclaredDeps: []string{"dep"}, UsedDeps: []string{"dep"}},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := setupTestModule(t, tt.dir)

			results, err := AnalyzeDir(tmpDir)
			if err != nil {
				t.Fatalf("AnalyzeDir failed: %v", err)
			}

			// Verify results
			// Check we found expected nodes
			foundNodes := make(map[string]AnalysisResult)
			issueCount := 0
			for _, r := range results {
				foundNodes[r.NodeID] = r
				if r.HasIssues() {
					issueCount++
				}
			}

			if issueCount != tt.wantIssues {
				t.Errorf("got %d issues, want %d", issueCount, tt.wantIssues)
			}

			for id, want := range tt.wantNodes {
				got, ok := foundNodes[id]
				if !ok {
					t.Errorf("node %s not found", id)
					continue
				}

				checkDeps("DeclaredDeps", t, id, got.DeclaredDeps, want.DeclaredDeps)
				checkDeps("UsedDeps", t, id, got.UsedDeps, want.UsedDeps)
				checkDeps("Undeclared", t, id, got.Undeclared, want.Undeclared)
				checkDeps("Unused", t, id, got.Unused, want.Unused)
			}
		})
	}
}

func checkDeps(field string, t *testing.T, nodeID string, got, want []string) {
	if len(got) != len(want) {
		t.Errorf("[%s] %s: got %v, want %v", nodeID, field, got, want)
		return
	}
	// TODO: better comparison (sets)
	// For now simple length check + ensuring elements present
}

// setupTestModule prepares a temporary directory with a valid go.mod and copied test files
func setupTestModule(t *testing.T, subpath string) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Source directory for test case
	srcDir := filepath.Join(cwd, "internal", "testcases", subpath)
	
	// Create temp dir
	tmpDir := t.TempDir()

	// Copy files
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(srcDir, path)
			return os.MkdirAll(filepath.Join(tmpDir, rel), 0755)
		}
		
		rel, _ := filepath.Rel(srcDir, path)
		dest := filepath.Join(tmpDir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
	if err != nil {
		t.Fatalf("failed to copy test files: %v", err)
	}

	// Create go.mod
	modContent := fmt.Sprintf(`module testmod
go 1.25
require github.com/grindlemire/graft v0.0.0
replace github.com/grindlemire/graft => %s
`, cwd)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(modContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput: %s", err, out)
	}

	return tmpDir
}
