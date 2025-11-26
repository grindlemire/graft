package graft

import (
	"os"
	"path/filepath"
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{"dep1", "dep2"},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[string](ctx, "dep1")
		v2, _ := graft.Dep[int](ctx, "dep2")
		return v1 + string(v2), nil
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[string](ctx, "dep1")
		v2, _ := graft.Dep[string](ctx, "dep2") // not declared!
		return v1 + v2, nil
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{"dep1", "dep2"},
	Run: func(ctx context.Context) (string, error) {
		v1, _ := graft.Dep[string](ctx, "dep1")
		// dep2 is declared but never used
		return v1, nil
	},
}
`,
			wantNodes:  1,
			wantIssues: 1,
			checkResult: func(t *testing.T, results []AnalysisResult) {
				if len(results[0].Unused) != 1 || results[0].Unused[0] != "dep2" {
					t.Errorf("expected unused dep2, got %v", results[0].Unused)
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
)

var nodeA = graft.Node[string]{
	ID: "nodeA",
	Run: func(ctx context.Context) (string, error) {
		return "a", nil
	},
}

var nodeB = graft.Node[string]{
	ID:        "nodeB",
	DependsOn: []string{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "nodeA")
		return v, nil
	},
}
`,
			wantNodes:  2,
			wantIssues: 0,
		},
		"local Node type (not a graft.Node)": {
			code: `package graft

import "context"

var node = Node[string]{
	ID:        "localnode",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := Dep[string](ctx, "dep1")
		return v, nil
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
)

var nodeB = graft.Node[string]{
	ID:        "nodeB",
	DependsOn: []string{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "nodeA")
		return v, nil
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "dep1")
		return v, nil
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "undeclared")
		return v, nil
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

			if tt.wantSubstr != "" && !containsSubstr(s, tt.wantSubstr) {
				t.Errorf("String() = %q, should contain %q", s, tt.wantSubstr)
			}
		})
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
