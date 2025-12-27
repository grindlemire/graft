package graft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeFile(t *testing.T) {
	type tc struct {
		code        string
		wantNodes   int
		wantIssues  int
		checkResult func(t *testing.T, results []AnalysisResult)
	}

	tests := map[string]tc{
		"valid node with matching deps": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
	"myapp/nodes/dep2"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID, dep2.ID},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[dep1.Output](ctx)
		v2, _ := graft.Dep[dep2.Output](ctx)
		return v1.String() + v2.String(), nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 0,
			checkResult: func(t *testing.T, results []AnalysisResult) {
				if results[0].NodeID != "mynode" {
					t.Errorf("got NodeID %q, want %q", results[0].NodeID, "mynode")
				}
				if len(results[0].DeclaredDeps) != 2 {
					t.Errorf("got %d declared deps, want 2", len(results[0].DeclaredDeps))
				}
				if len(results[0].UsedDeps) != 2 {
					t.Errorf("got %d used deps, want 2", len(results[0].UsedDeps))
				}
			},
		},
		"undeclared dependency": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
	"myapp/nodes/dep2"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[dep1.Output](ctx)
		v2, _ := graft.Dep[dep2.Output](ctx) // not declared!
		return v1.String() + v2.String(), nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 1,
			checkResult: func(t *testing.T, results []AnalysisResult) {
				if len(results[0].Undeclared) != 1 || results[0].Undeclared[0] != "dep2" {
					t.Errorf("expected undeclared dep2, got %v", results[0].Undeclared)
				}
			},
		},
		"unused dependency": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
	"myapp/nodes/dep2"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID, dep2.ID},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[dep1.Output](ctx)
		// dep2 is declared but never used
		return v1.String(), nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 1,
			checkResult: func(t *testing.T, results []AnalysisResult) {
				// With canonical forms, dep2.ID becomes "myapp/nodes/dep2.ID"
				if len(results[0].Unused) != 1 {
					t.Errorf("expected 1 unused dep, got %d: %v", len(results[0].Unused), results[0].Unused)
				}
				// Check that dep2 is reported (either as "dep2" or canonical form)
				unused := results[0].Unused[0]
				if unused != "dep2" && unused != "myapp/nodes/dep2.ID" {
					t.Errorf("expected unused dep2 (or myapp/nodes/dep2.ID), got %v", results[0].Unused)
				}
			},
		},
		"node without deps - no issues": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID: "standalone",
	Run: func(ctx context.Context) (string, error) {
		return "hello", nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 0,
		},
		"multiple nodes": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/nodeA"
)

var nodeADef = graft.Node[string]{
	ID: "nodeA",
	Run: func(ctx context.Context) (string, error) {
		return "a", nil
	},
}

var nodeB = graft.Node[string]{
	ID:        "nodeB",
	DependsOn: []graft.ID{nodeA.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[nodeA.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantNodes:  2,
			wantIssues: 0,
		},
		"local Node type (not a graft.Node)": {
			code: `package graft

import (
	"context"
	"myapp/nodes/dep1"
)

var node = Node[string]{
	ID:        "localnode",
	DependsOn: []ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 0,
		},
		"no nodes in file": {
			code: `package test

func helper() string {
	return "hello"
}
`,
			wantNodes:  0,
			wantIssues: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Write temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := AnalyzeFile(tmpFile)
			if err != nil {
				t.Fatalf("AnalyzeFile error: %v", err)
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

			if tt.checkResult != nil && len(results) > 0 {
				tt.checkResult(t, results)
			}
		})
	}
}

func TestAnalyzeDir(t *testing.T) {
	type tc struct {
		files      map[string]string
		wantNodes  int
		wantIssues int
	}

	tests := map[string]tc{
		"multiple files": {
			files: map[string]string{
				"node_a.go": `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var nodeA = graft.Node[string]{
	ID: "nodeA",
	Run: func(ctx context.Context) (string, error) { return "a", nil },
}
`,
				"node_b.go": `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/nodeA"
)

var nodeB = graft.Node[string]{
	ID:        "nodeB",
	DependsOn: []graft.ID{nodeA.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[nodeA.Output](ctx)
		return v.String(), nil
	},
}
`,
			},
			wantNodes:  2,
			wantIssues: 0,
		},
		"skips test files": {
			files: map[string]string{
				"node.go": `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[any]{
	ID: "realnode",
	Run: func(ctx context.Context) (any, error) { return nil, nil },
}
`,
				"node_test.go": `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var testNode = graft.Node[any]{
	ID: "testnode",
	Run: func(ctx context.Context) (any, error) { return nil, nil },
}
`,
			},
			wantNodes:  1, // Only the non-test file
			wantIssues: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.files {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write %s: %v", filename, err)
				}
			}

			results, err := AnalyzeDir(tmpDir)
			if err != nil {
				t.Fatalf("AnalyzeDir error: %v", err)
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
		})
	}
}

func TestValidateDeps(t *testing.T) {
	type tc struct {
		code    string
		wantErr bool
	}

	tests := map[string]tc{
		"valid deps - no error": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantErr: false,
		},
		"invalid deps - returns error": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/undeclared"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[undeclared.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			err := ValidateDeps(tmpDir)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAnalysisResultString(t *testing.T) {
	type tc struct {
		result     AnalysisResult
		wantSubstr string
		shouldBeOK bool
	}

	tests := map[string]tc{
		"no issues": {
			result: AnalysisResult{
				NodeID:       "mynode",
				File:         "test.go",
				DeclaredDeps: []string{"a"},
				UsedDeps:     []string{"a"},
			},
			wantSubstr: "OK",
			shouldBeOK: true,
		},
		"undeclared deps": {
			result: AnalysisResult{
				NodeID:     "mynode",
				File:       "test.go",
				Undeclared: []string{"missing"},
			},
			wantSubstr: "undeclared",
			shouldBeOK: false,
		},
		"unused deps": {
			result: AnalysisResult{
				NodeID: "mynode",
				File:   "test.go",
				Unused: []string{"extra"},
			},
			wantSubstr: "unused",
			shouldBeOK: false,
		},
		"both issues": {
			result: AnalysisResult{
				NodeID:     "mynode",
				File:       "test.go",
				Undeclared: []string{"missing"},
				Unused:     []string{"extra"},
			},
			shouldBeOK: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := tt.result.String()

			if tt.shouldBeOK != !tt.result.HasIssues() {
				t.Errorf("HasIssues() = %v, want %v", tt.result.HasIssues(), !tt.shouldBeOK)
			}

			if tt.wantSubstr != "" && !strings.Contains(s, tt.wantSubstr) {
				t.Errorf("String() = %q, should contain %q", s, tt.wantSubstr)
			}
		})
	}
}

func TestNodeWithIDConstant(t *testing.T) {
	// Test getPackageNameAsID - when ID field references a constant like ID: ID
	code := `package mypackage

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

const ID = "mypackage"

var node = graft.Node[string]{
	ID:        ID,
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`
	tmpDir := t.TempDir()
	// Create subdirectory to test package name extraction
	subDir := filepath.Join(tmpDir, "mypackage")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	tmpFile := filepath.Join(subDir, "node.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	results, err := AnalyzeFile(tmpFile)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d nodes, want 1", len(results))
	}

	// When ID is a constant reference, we use the package name (directory name)
	if results[0].NodeID != "mypackage" {
		t.Errorf("got NodeID %q, want %q", results[0].NodeID, "mypackage")
	}
}

func TestExtractFromCall(t *testing.T) {
	// Test extractFromCall - graft.ID("foo") and ID("foo") patterns
	tests := map[string]struct {
		code      string
		wantNodes int
		wantDeps  []string
	}{
		"graft.ID call in DependsOn": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{graft.ID("dep1"), graft.ID("dep2")},
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 1,
			wantDeps:  []string{"dep1", "dep2"},
		},
		"local ID call in DependsOn": {
			code: `package graft

import (
	"context"
)

var node = Node[string]{
	ID:        "mynode",
	DependsOn: []ID{ID("dep1"), ID("dep2")},
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 1,
			wantDeps:  []string{"dep1", "dep2"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := AnalyzeFile(tmpFile)
			if err != nil {
				t.Fatalf("AnalyzeFile error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Fatalf("got %d nodes, want %d", len(results), tt.wantNodes)
			}

			if tt.wantDeps != nil {
				for _, wantDep := range tt.wantDeps {
					found := false
					for _, d := range results[0].DeclaredDeps {
						if d == wantDep {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected declared dep %q not found in %v", wantDep, results[0].DeclaredDeps)
					}
				}
			}
		})
	}
}

func TestIsNodeTypeEdgeCases(t *testing.T) {
	// Test isNodeType - non-generic Node, graft.Node, and IndexListExpr cases
	tests := map[string]struct {
		code      string
		wantNodes int
	}{
		"non-generic Node (Ident)": {
			// This tests the *ast.Ident case for backwards compat
			code: `package graft

import "context"

type Node struct {
	ID  string
	Run func(ctx context.Context) (any, error)
}

var node = Node{
	ID: "simplenode",
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes: 1,
		},
		"non-generic graft.Node (SelectorExpr)": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

type Output = any

// This is a hypothetical non-generic graft.Node usage
var _ = struct {
	graft.Node
}{}

// But the actual pattern we need to test
var node = graft.Node[Output]{
	ID: "testnode",
	Run: func(ctx context.Context) (Output, error) {
		return nil, nil
	},
}
`,
			wantNodes: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := AnalyzeFile(tmpFile)
			if err != nil {
				t.Fatalf("AnalyzeFile error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", len(results), tt.wantNodes)
			}
		})
	}
}

func TestCheckDepCallIndexListExpr(t *testing.T) {
	// Test checkDepCall with IndexListExpr (multiple type params)
	// This is a future-proofing test for Dep[T, U] style calls
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

// Simulate a hypothetical multi-param generic call
// In real code this doesn't exist, but the analyzer supports it
var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	results, err := AnalyzeFile(tmpFile)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d nodes, want 1", len(results))
	}
}

func TestExtractDependsOnEdgeCases(t *testing.T) {
	// Test extractDependsOn with identifiers (local constants)
	tests := map[string]struct {
		code      string
		wantNodes int
		wantDeps  int
	}{
		"identifier deps (local constants)": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

const (
	Dep1ID = "dep1"
	Dep2ID = "dep2"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{Dep1ID, Dep2ID},
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 1,
			wantDeps:  2,
		},
		"string literal deps": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{"dep1", "dep2", "dep3"},
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 1,
			wantDeps:  3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := AnalyzeFile(tmpFile)
			if err != nil {
				t.Fatalf("AnalyzeFile error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Fatalf("got %d nodes, want %d", len(results), tt.wantNodes)
			}

			if len(results[0].DeclaredDeps) != tt.wantDeps {
				t.Errorf("got %d declared deps, want %d", len(results[0].DeclaredDeps), tt.wantDeps)
			}
		})
	}
}

func TestRunFunctionReference(t *testing.T) {
	// Test analyzeNodeLiteral with Run: funcName (function reference)
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

func run(ctx context.Context) (string, error) {
	v, _ := graft.Dep[dep1.Output](ctx)
	return v.String(), nil
}

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run:       run,
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	results, err := AnalyzeFile(tmpFile)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d nodes, want 1", len(results))
	}

	// Should detect dep1 usage even though Run is a function reference
	if len(results[0].UsedDeps) != 1 {
		t.Errorf("got %d used deps, want 1", len(results[0].UsedDeps))
	}

	found := false
	for _, d := range results[0].UsedDeps {
		if d == "dep1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected used dep 'dep1' not found in %v", results[0].UsedDeps)
	}
}

func TestAnalyzeNodeLiteralEdgeCases(t *testing.T) {
	// Test analyzeNodeLiteral edge cases
	tests := map[string]struct {
		code      string
		wantNodes int
	}{
		"node without ID - skipped": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 0, // No ID means it's skipped
		},
		"non-KeyValueExpr elements - skipped gracefully": {
			// This tests the path where elt is not *ast.KeyValueExpr
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID: "mynode",
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`,
			wantNodes: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := AnalyzeFile(tmpFile)
			if err != nil {
				t.Fatalf("AnalyzeFile error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", len(results), tt.wantNodes)
			}
		})
	}
}

func TestLocalDepCall(t *testing.T) {
	// Test checkDepCall with local Dep[T] (not graft.Dep[T])
	code := `package graft

import "context"

var node = Node[string]{
	ID:        "localnode",
	DependsOn: []ID{"dep1"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := Dep[dep1Output](ctx)
		return v.String(), nil
	},
}

type dep1Output struct{}
func (d dep1Output) String() string { return "" }
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	results, err := AnalyzeFile(tmpFile)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d nodes, want 1", len(results))
	}

	// Should detect dep1Output usage with local Dep call
	if len(results[0].UsedDeps) != 1 {
		t.Errorf("got %d used deps, want 1: %v", len(results[0].UsedDeps), results[0].UsedDeps)
	}
}

// TestSharedInterfacePackage tests that Dep[T] calls resolve to the correct node IDs
// when multiple nodes in one package output different types, and another node depends on them.
// This tests the bug where the analyzer incorrectly infers dependency IDs from the package name
// of the type argument (e.g., Dep[ports.Executor] -> "ports") instead of resolving to the
// actual node ID that outputs that type (e.g., "executor").
func TestSharedInterfacePackage(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/sharedinterface")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	// Find the app node
	var appResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "app" {
			appResult = &results[i]
			break
		}
	}
	if appResult == nil {
		t.Fatalf("app node not found in results. Got node IDs: %v", func() []string {
			ids := make([]string, len(results))
			for i := range results {
				ids[i] = results[i].NodeID
			}
			return ids
		}())
	}

	// Check that used deps are "executor" and "logger", NOT "ports" (the package name)
	usedDepsMap := make(map[string]bool)
	for _, dep := range appResult.UsedDeps {
		usedDepsMap[dep] = true
	}

	if !usedDepsMap["executor"] {
		t.Errorf("expected 'executor' in used deps, got %v", appResult.UsedDeps)
	}
	if !usedDepsMap["logger"] {
		t.Errorf("expected 'logger' in used deps, got %v", appResult.UsedDeps)
	}

	// "ports" (the package name of the type argument) should NOT be detected as a used dep
	if usedDepsMap["ports"] {
		t.Errorf("'ports' should not be in used deps (it's the interface package, not a node ID), got %v", appResult.UsedDeps)
	}

	// The node should have no issues - all deps are properly declared
	if appResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			appResult.Undeclared, appResult.Unused)
	}
}

// TestSharedPackageWithImportAlias tests that Dep[T] calls resolve correctly
// when using a custom import alias for the dependency package.
func TestSharedPackageWithImportAlias(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/importaliases")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var appResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "app" {
			appResult = &results[i]
			break
		}
	}
	if appResult == nil {
		t.Fatalf("app node not found")
	}

	// Should resolve svc.DBConnection to "database-service", not "svc" (the alias)
	usedDepsMap := make(map[string]bool)
	for _, dep := range appResult.UsedDeps {
		usedDepsMap[dep] = true
	}

	if !usedDepsMap["database-service"] {
		t.Errorf("expected 'database-service' in used deps, got %v", appResult.UsedDeps)
	}

	// "svc" (the alias) should NOT be detected as a used dep
	if usedDepsMap["svc"] {
		t.Errorf("'svc' should not be in used deps (it's an alias), got %v", appResult.UsedDeps)
	}

	if appResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			appResult.Undeclared, appResult.Unused)
	}
}

// TestNodeIDVariableNameDoesNotMatter tests that the variable name used
// to store the node ID constant doesn't affect dependency resolution.
func TestNodeIDVariableNameDoesNotMatter(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/iddeclarations")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var consumerResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "consumer" {
			consumerResult = &results[i]
			break
		}
	}
	if consumerResult == nil {
		t.Fatalf("consumer node not found")
	}

	if consumerResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			consumerResult.Undeclared, consumerResult.Unused)
	}
}

// TestMultipleNodesInPackageWithDifferentIDPatterns tests resolution when
// multiple nodes in the same package use different ID declaration patterns.
func TestMultipleNodesInPackageWithDifferentIDPatterns(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/iddeclarations")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var appResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "app" {
			appResult = &results[i]
			break
		}
	}
	if appResult == nil {
		t.Fatalf("app node not found")
	}

	usedDepsMap := make(map[string]bool)
	for _, dep := range appResult.UsedDeps {
		usedDepsMap[dep] = true
	}

	// All three dependencies should be resolved to their actual IDs
	expectedDeps := []string{"cache-node", "database-node", "logger-node"}
	for _, expected := range expectedDeps {
		if !usedDepsMap[expected] {
			t.Errorf("expected '%s' in used deps, got %v", expected, appResult.UsedDeps)
		}
	}

	// "shared" (the package name) should NOT appear
	if usedDepsMap["shared"] {
		t.Errorf("'shared' should not be in used deps, got %v", appResult.UsedDeps)
	}

	if appResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			appResult.Undeclared, appResult.Unused)
	}
}

// TestDeepNestedImportPath tests that deeply nested import paths resolve correctly.
func TestDeepNestedImportPath(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/importpaths")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var handlerResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "handler" {
			handlerResult = &results[i]
			break
		}
	}
	if handlerResult == nil {
		t.Fatalf("handler node not found")
	}

	usedDepsMap := make(map[string]bool)
	for _, dep := range handlerResult.UsedDeps {
		usedDepsMap[dep] = true
	}

	if !usedDepsMap["oauth-provider"] {
		t.Errorf("expected 'oauth-provider' in used deps, got %v", handlerResult.UsedDeps)
	}

	// "providers" (the package name) should NOT appear
	if usedDepsMap["providers"] {
		t.Errorf("'providers' should not be in used deps, got %v", handlerResult.UsedDeps)
	}

	if handlerResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			handlerResult.Undeclared, handlerResult.Unused)
	}
}

// TestMultipleImportAliasesSamePackage tests that the analyzer handles
// when the same package is (hypothetically) imported with different aliases in different files.
// This simulates the case where node resolution must work across files with different import styles.
func TestMultipleImportAliasesSamePackage(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/importaliases")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	// Check both consumers
	for _, consumerID := range []string{"consumer1", "consumer2"} {
		var result *AnalysisResult
		for i := range results {
			if results[i].NodeID == consumerID {
				result = &results[i]
				break
			}
		}
		if result == nil {
			t.Fatalf("%s node not found", consumerID)
		}

		usedDepsMap := make(map[string]bool)
		for _, dep := range result.UsedDeps {
			usedDepsMap[dep] = true
		}

		if !usedDepsMap["config-node"] {
			t.Errorf("%s: expected 'config-node' in used deps, got %v", consumerID, result.UsedDeps)
		}

		// Neither alias should appear
		if usedDepsMap["t"] || usedDepsMap["apptypes"] || usedDepsMap["types"] {
			t.Errorf("%s: aliases/package name should not be in used deps, got %v", consumerID, result.UsedDeps)
		}

		if result.HasIssues() {
			t.Errorf("%s: expected no issues, but got: undeclared=%v, unused=%v",
				consumerID, result.Undeclared, result.Unused)
		}
	}
}

// TestDependsOnWithDifferentIDFormats tests that DependsOn declarations work
// with various formats: string literals, pkg.ID constants, and graft.ID() calls.
func TestDependsOnWithDifferentIDFormats(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/dependencyformats")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var appResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "app" {
			appResult = &results[i]
			break
		}
	}
	if appResult == nil {
		t.Fatalf("app node not found")
	}

	if appResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			appResult.Undeclared, appResult.Unused)
	}

	// Verify both declared deps are found (in some canonical form)
	if len(appResult.DeclaredDeps) != 2 {
		t.Errorf("expected 2 declared deps, got %d: %v", len(appResult.DeclaredDeps), appResult.DeclaredDeps)
	}

	if len(appResult.UsedDeps) != 2 {
		t.Errorf("expected 2 used deps, got %d: %v", len(appResult.UsedDeps), appResult.UsedDeps)
	}
}

// TestSameTypeNameDifferentPackages tests that the analyzer correctly distinguishes
// between types with the same name but from different packages.
func TestSameTypeNameDifferentPackages(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/typeresolution")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	var consumerResult *AnalysisResult
	for i := range results {
		if results[i].NodeID == "consumer" {
			consumerResult = &results[i]
			break
		}
	}
	if consumerResult == nil {
		t.Fatalf("consumer node not found")
	}

	usedDepsMap := make(map[string]bool)
	for _, dep := range consumerResult.UsedDeps {
		usedDepsMap[dep] = true
	}

	// Should resolve to the correct node IDs, not package names
	if !usedDepsMap["pkg1-output"] {
		t.Errorf("expected 'pkg1-output' in used deps, got %v", consumerResult.UsedDeps)
	}
	if !usedDepsMap["pkg2-output"] {
		t.Errorf("expected 'pkg2-output' in used deps, got %v", consumerResult.UsedDeps)
	}

	// Package names should NOT appear
	if usedDepsMap["pkg1"] || usedDepsMap["pkg2"] {
		t.Errorf("package names should not be in used deps, got %v", consumerResult.UsedDeps)
	}

	if consumerResult.HasIssues() {
		t.Errorf("expected no issues, but got: undeclared=%v, unused=%v",
			consumerResult.Undeclared, consumerResult.Unused)
	}
}

// TestPointerTypeOutputResolution tests that pointer types in Node output are handled.
func TestPointerTypeOutputResolution(t *testing.T) {
	results, err := AnalyzeDir("internal/testcases/typeresolution")
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	// Just verify no panics and basic structure works
	foundProvider := false
	foundPointerConsumer := false
	for _, r := range results {
		if r.NodeID == "connection-provider" {
			foundProvider = true
		}
		if r.NodeID == "pointer-consumer" {
			foundPointerConsumer = true
			// Verify it has no issues
			if r.HasIssues() {
				t.Errorf("pointer-consumer: expected no issues, but got: undeclared=%v, unused=%v",
					r.Undeclared, r.Unused)
			}
		}
	}

	if !foundProvider {
		t.Error("connection-provider node not found")
	}
	if !foundPointerConsumer {
		t.Error("pointer-consumer node not found")
	}
}

func TestAnalysisResultHasIssues(t *testing.T) {
	type tc struct {
		result  AnalysisResult
		wantHas bool
	}

	tests := map[string]tc{
		"empty - no issues": {
			result:  AnalysisResult{},
			wantHas: false,
		},
		"declared and used match - no issues": {
			result: AnalysisResult{
				DeclaredDeps: []string{"a", "b"},
				UsedDeps:     []string{"a", "b"},
			},
			wantHas: false,
		},
		"has undeclared": {
			result: AnalysisResult{
				Undeclared: []string{"x"},
			},
			wantHas: true,
		},
		"has unused": {
			result: AnalysisResult{
				Unused: []string{"y"},
			},
			wantHas: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.result.HasIssues()
			if got != tt.wantHas {
				t.Errorf("HasIssues() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}
