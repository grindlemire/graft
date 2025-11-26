package graft

import (
	"context"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	type tc struct {
		catalog   map[ID]node
		wantCount int
	}

	tests := map[string]tc{
		"empty catalog": {
			catalog:   map[ID]node{},
			wantCount: 0,
		},
		"single node catalog": {
			catalog: map[ID]node{
				"a": makeNode("a", nil, nil),
			},
			wantCount: 1,
		},
		"multiple node catalog": {
			catalog: map[ID]node{
				"a": makeNode("a", nil, nil),
				"b": makeNode("b", nil, nil),
				"c": makeNode("c", nil, nil),
			},
			wantCount: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := NewBuilder(tt.catalog)
			if b == nil {
				t.Fatal("NewBuilder returned nil")
			}
			if len(b.catalog) != tt.wantCount {
				t.Errorf("got %d nodes in catalog, want %d", len(b.catalog), tt.wantCount)
			}
		})
	}
}

func TestBuilderBuildFor(t *testing.T) {
	type tc struct {
		catalog      map[ID]node
		targets      []ID
		wantNodes    []ID
		wantErr      bool
		errSubstr    string
		verifyResult func(t *testing.T, e *Engine)
	}

	baseCatalog := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "a", nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) { return "b", nil }),
		"c": makeNode("c", []ID{"a"}, func(ctx context.Context) (any, error) { return "c", nil }),
		"d": makeNode("d", []ID{"b", "c"}, func(ctx context.Context) (any, error) { return "d", nil }),
		"e": makeNode("e", nil, func(ctx context.Context) (any, error) { return "e", nil }),
	}

	tests := map[string]tc{
		"single target no deps": {
			catalog:   baseCatalog,
			targets:   []ID{"a"},
			wantNodes: []ID{"a"},
			wantErr:   false,
		},
		"single target with one dep": {
			catalog:   baseCatalog,
			targets:   []ID{"b"},
			wantNodes: []ID{"a", "b"},
			wantErr:   false,
		},
		"single target with transitive deps": {
			catalog:   baseCatalog,
			targets:   []ID{"d"},
			wantNodes: []ID{"a", "b", "c", "d"},
			wantErr:   false,
		},
		"multiple targets": {
			catalog:   baseCatalog,
			targets:   []ID{"b", "c"},
			wantNodes: []ID{"a", "b", "c"},
			wantErr:   false,
		},
		"multiple targets with overlap": {
			catalog:   baseCatalog,
			targets:   []ID{"d", "e"},
			wantNodes: []ID{"a", "b", "c", "d", "e"},
			wantErr:   false,
		},
		"unknown target": {
			catalog:   baseCatalog,
			targets:   []ID{"unknown"},
			wantErr:   true,
			errSubstr: "unknown node: unknown",
		},
		"mixed known and unknown targets": {
			catalog:   baseCatalog,
			targets:   []ID{"a", "unknown"},
			wantErr:   true,
			errSubstr: "unknown node: unknown",
		},
		"empty targets": {
			catalog:   baseCatalog,
			targets:   []ID{},
			wantNodes: []ID{},
			wantErr:   false,
		},
		"duplicate targets": {
			catalog:   baseCatalog,
			targets:   []ID{"a", "a", "a"},
			wantNodes: []ID{"a"},
			wantErr:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := NewBuilder(tt.catalog)
			e, err := b.BuildFor(tt.targets...)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !containsSubstr(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if e == nil {
				t.Fatal("BuildFor returned nil engine")
			}

			// Verify correct nodes are present
			if len(e.nodes) != len(tt.wantNodes) {
				t.Errorf("got %d nodes, want %d", len(e.nodes), len(tt.wantNodes))
			}

			for _, nodeID := range tt.wantNodes {
				if _, exists := e.nodes[nodeID]; !exists {
					t.Errorf("missing expected node %q", nodeID)
				}
			}

			if tt.verifyResult != nil {
				tt.verifyResult(t, e)
			}
		})
	}
}

func TestBuilderBuildForExecution(t *testing.T) {
	// Test that built engines actually execute correctly
	type tc struct {
		catalog     map[ID]node
		targets     []ID
		wantResults map[ID]any
	}

	tests := map[string]tc{
		"execute linear chain": {
			catalog: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
				"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
					v, _ := Dep[int](ctx, "a")
					return v * 2, nil
				}),
				"c": makeNode("c", []ID{"b"}, func(ctx context.Context) (any, error) {
					v, _ := Dep[int](ctx, "b")
					return v * 2, nil
				}),
			},
			targets:     []ID{"c"},
			wantResults: map[ID]any{"a": 1, "b": 2, "c": 4},
		},
		"execute subset of catalog": {
			catalog: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "included", nil }),
				"b": makeNode("b", nil, func(ctx context.Context) (any, error) { return "excluded", nil }),
			},
			targets:     []ID{"a"},
			wantResults: map[ID]any{"a": "included"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := NewBuilder(tt.catalog)
			e, err := b.BuildFor(tt.targets...)
			if err != nil {
				t.Fatalf("BuildFor error: %v", err)
			}

			if err := e.Run(context.Background()); err != nil {
				t.Fatalf("Run error: %v", err)
			}

			results := e.Results()
			if len(results) != len(tt.wantResults) {
				t.Errorf("got %d results, want %d", len(results), len(tt.wantResults))
			}

			for k, want := range tt.wantResults {
				got, ok := results[k]
				if !ok {
					t.Errorf("missing result for %q", k)
					continue
				}
				if got != want {
					t.Errorf("result[%q] = %v, want %v", k, got, want)
				}
			}
		})
	}
}

func TestBuilderDoesNotMutateCatalog(t *testing.T) {
	catalog := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) { return 2, nil }),
	}

	originalLen := len(catalog)

	b := NewBuilder(catalog)
	_, err := b.BuildFor("b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Catalog should be unchanged
	if len(catalog) != originalLen {
		t.Errorf("catalog was mutated: got %d nodes, want %d", len(catalog), originalLen)
	}
}
