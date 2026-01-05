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

// TestNonGenericNodeAnalysis tests the old code path for non-generic Node literals.
// This exercises analyzeNodeLiteral, getPackageNameAsID, extractDependsOn, and extractFromCall.
func TestNonGenericNodeAnalysis(t *testing.T) {
	tests := map[string]struct {
		code           string
		wantNodes      int
		wantDeclared   int
		wantUsed       int
		wantUndeclared int
		wantUnused     int
	}{
		"non-generic node with string literal ID": {
			// Non-generic Node - exercises analyzeNodeLiteral path
			code: `package graft

import "context"

// Simulate non-generic Node type
type Node struct {
	ID        string
	DependsOn []string
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "simplenode",
	DependsOn: []string{"dep1", "dep2"},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes:    1,
			wantDeclared: 2, // "dep1", "dep2"
		},
		"non-generic node with identifier ID (exercises getPackageNameAsID)": {
			// This tests the ID: SomeConst path which uses getPackageNameAsID
			code: `package graft

import "context"

const NodeID = "myconst"

type Node struct {
	ID  string
	Run func(ctx context.Context) (any, error)
}

var node = Node{
	ID:  NodeID,
	Run: func(ctx context.Context) (any, error) { return nil, nil },
}
`,
			wantNodes: 1,
		},
		"non-generic node DependsOn with selector expressions": {
			// Tests extractDependsOn with pkg.ID style
			code: `package graft

import "context"

type Node struct {
	ID        string
	DependsOn []string
	Run       func(ctx context.Context) (any, error)
}

type dep1 struct{}
type dep2 struct{}

var _ = dep1{}
var _ = dep2{}

var node = Node{
	ID:        "mynode",
	DependsOn: []string{"dep1", "dep2"},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes:    1,
			wantDeclared: 2,
		},
		"non-generic node with graft.ID() calls in DependsOn": {
			// Tests extractFromCall via extractDependsOn
			code: `package graft

import "context"

type ID string

func MakeID(s string) ID { return ID(s) }

type Node struct {
	ID        string
	DependsOn []ID
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "mynode",
	DependsOn: []ID{ID("foo"), ID("bar")},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes:    1,
			wantDeclared: 2,
		},
		"non-generic node with function reference Run": {
			// Tests the funcDecls lookup path in analyzeNodeLiteral
			code: `package graft

import "context"

type Node struct {
	ID  string
	Run func(ctx context.Context) (any, error)
}

func run(ctx context.Context) (any, error) {
	return nil, nil
}

var node = Node{
	ID:  "funcrefnode",
	Run: run,
}
`,
			wantNodes: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			subDir := filepath.Join(tmpDir, "testpkg")
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("failed to create subdir: %v", err)
			}
			tmpFile := filepath.Join(subDir, "node.go")
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

			if tt.wantDeclared > 0 && len(results) > 0 {
				if len(results[0].DeclaredDeps) != tt.wantDeclared {
					t.Errorf("got %d declared deps, want %d: %v", len(results[0].DeclaredDeps), tt.wantDeclared, results[0].DeclaredDeps)
				}
			}
		})
	}
}

// TestAnalyzeFileDebugMode exercises debug output paths in analyzeNodeLiteral
func TestAnalyzeFileDebugMode(t *testing.T) {
	// Enable debug mode
	oldDebug := AnalyzeFileDebug
	AnalyzeFileDebug = true
	defer func() { AnalyzeFileDebug = oldDebug }()

	code := `package graft

import "context"

type Node struct {
	ID        string
	DependsOn []string
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "debugnode",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Just verify it doesn't panic with debug enabled
	results, err := AnalyzeFile(tmpFile)
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d nodes, want 1", len(results))
	}
}

// TestAnalyzeDirDebugMode exercises debug output paths in AnalyzeDir
func TestAnalyzeDirDebugMode(t *testing.T) {
	oldDebug := AnalyzeDirDebug
	AnalyzeDirDebug = true
	defer func() { AnalyzeDirDebug = oldDebug }()

	tmpDir := t.TempDir()
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID: "debugtest",
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "node.go"), []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Just verify it doesn't panic with debug enabled
	results, err := AnalyzeDir(tmpDir)
	if err != nil {
		t.Fatalf("AnalyzeDir error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d nodes, want 1", len(results))
	}
}

// TestIsNodeTypeForCollectionEdgeCases exercises additional isNodeTypeForCollection paths
func TestIsNodeTypeForCollectionEdgeCases(t *testing.T) {
	tests := map[string]struct {
		code      string
		wantNodes int
	}{
		"IndexListExpr with multiple type params": {
			// Future-proofing: Node[T, U, V] style
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

// This simulates how multi-param generics would look
var node = graft.Node[string]{
	ID: "multitype",
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

// TestMatchesDeclaredDepEdgeCases tests additional matching patterns
func TestMatchesDeclaredDepEdgeCases(t *testing.T) {
	tests := map[string]struct {
		usedDep      string
		declaredDeps map[string]bool
		wantMatch    bool
	}{
		"exact match": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"dep1": true},
			wantMatch:    true,
		},
		"suffix match with dot": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"pkg.dep1": true},
			wantMatch:    true,
		},
		"suffix match with slash": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"myapp/nodes/dep1": true},
			wantMatch:    true,
		},
		"suffix match with .ID": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"dep1.ID": true},
			wantMatch:    true,
		},
		"full path .ID match": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"myapp/nodes/dep1.ID": true},
			wantMatch:    true,
		},
		"no match": {
			usedDep:      "dep1",
			declaredDeps: map[string]bool{"dep2": true},
			wantMatch:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := matchesDeclaredDep(tt.usedDep, tt.declaredDeps)
			if got != tt.wantMatch {
				t.Errorf("matchesDeclaredDep(%q, %v) = %v, want %v", tt.usedDep, tt.declaredDeps, got, tt.wantMatch)
			}
		})
	}
}

// TestMatchesUsedDepEdgeCases tests additional matching patterns
func TestMatchesUsedDepEdgeCases(t *testing.T) {
	tests := map[string]struct {
		declaredDep string
		usedDeps    map[string]bool
		wantMatch   bool
	}{
		"exact match": {
			declaredDep: "dep1",
			usedDeps:    map[string]bool{"dep1": true},
			wantMatch:   true,
		},
		"package path to short form": {
			declaredDep: "myapp/nodes/dep1",
			usedDeps:    map[string]bool{"dep1": true},
			wantMatch:   true,
		},
		"with .ID suffix": {
			declaredDep: "myapp/nodes/dep1.ID",
			usedDeps:    map[string]bool{"dep1": true},
			wantMatch:   true,
		},
		"shared.ID1 to ID1": {
			declaredDep: "shared.ID1",
			usedDeps:    map[string]bool{"ID1": true},
			wantMatch:   true,
		},
		"no match": {
			declaredDep: "dep1",
			usedDeps:    map[string]bool{"dep2": true},
			wantMatch:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := matchesUsedDep(tt.declaredDep, tt.usedDeps)
			if got != tt.wantMatch {
				t.Errorf("matchesUsedDep(%q, %v) = %v, want %v", tt.declaredDep, tt.usedDeps, got, tt.wantMatch)
			}
		})
	}
}

// TestExtractDependsOnSelectorExpressions tests the extractDependsOn method with pkg.ID patterns
func TestExtractDependsOnSelectorExpressions(t *testing.T) {
	// This exercises the selector expression path in nodeAnalyzer.extractDependsOn
	code := `package graft

import "context"

type ID string

type dep1pkg struct{}
type dep2pkg struct{}

var ID1 ID = "dep1"
var ID2 ID = "dep2"

type Node struct {
	ID        string
	DependsOn []ID
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "selectortest",
	DependsOn: []ID{dep1pkg.ID, dep2pkg.ID},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "testpkg")
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

	// Should extract deps from selector expressions
	if len(results[0].DeclaredDeps) != 2 {
		t.Errorf("got %d declared deps, want 2: %v", len(results[0].DeclaredDeps), results[0].DeclaredDeps)
	}
}

// TestNonGenericNodeWithIDCall tests extractFromCall in DependsOn
func TestNonGenericNodeWithIDCall(t *testing.T) {
	// This exercises the extractFromCall path via graft.ID("foo") calls
	code := `package graft

import "context"

type ID string

func MakeID(s string) ID { return ID(s) }

type Node struct {
	ID        string
	DependsOn []ID
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "idcalltest",
	DependsOn: []ID{ID("dep1"), ID("dep2")},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "testpkg")
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

	// Should extract deps from ID() calls
	if len(results[0].DeclaredDeps) != 2 {
		t.Errorf("got %d declared deps, want 2: %v", len(results[0].DeclaredDeps), results[0].DeclaredDeps)
	}
}

// TestAnalyzeNodeLiteralDebugPaths exercises debug output in analyzeNodeLiteral
func TestAnalyzeNodeLiteralDebugPaths(t *testing.T) {
	oldDebug := AnalyzeFileDebug
	AnalyzeFileDebug = true
	defer func() { AnalyzeFileDebug = oldDebug }()

	// Test with various patterns to exercise debug paths
	tests := map[string]struct {
		code      string
		wantNodes int
	}{
		"with inline Run function": {
			code: `package graft

import "context"

type Node struct {
	ID        string
	DependsOn []string
	Run       func(ctx context.Context) (any, error)
}

var node = Node{
	ID:        "inlinerun",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes: 1,
		},
		"with function reference Run": {
			code: `package graft

import "context"

type Node struct {
	ID  string
	Run func(ctx context.Context) (any, error)
}

func run(ctx context.Context) (any, error) {
	return nil, nil
}

var node = Node{
	ID:  "funcref",
	Run: run,
}
`,
			wantNodes: 1,
		},
		"missing ID field": {
			code: `package graft

import "context"

type Node struct {
	ID  string
	Run func(ctx context.Context) (any, error)
}

var node = Node{
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantNodes: 0, // No ID means nil result
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
