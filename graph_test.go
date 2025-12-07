package graft

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestPrintGraph_EmptyRegistry(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No nodes registered") {
		t.Errorf("expected 'No nodes registered', got: %q", output)
	}
}

func TestPrintGraph_Linear(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	// Create a linear dependency chain: a -> b -> c
	Register(Node[string]{
		ID:        "a",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "a", nil
		},
	})

	Register(Node[string]{
		ID:        "b",
		DependsOn: []ID{"a"},
		Run: func(ctx context.Context) (string, error) {
			return "b", nil
		},
	})

	Register(Node[string]{
		ID:        "c",
		DependsOn: []ID{"b"},
		Run: func(ctx context.Context) (string, error) {
			return "c", nil
		},
	})

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Verify all nodes are present
	if !strings.Contains(output, "a") {
		t.Error("output should contain 'a'")
	}
	if !strings.Contains(output, "b") {
		t.Error("output should contain 'b'")
	}
	if !strings.Contains(output, "c") {
		t.Error("output should contain 'c'")
	}
}

func TestPrintGraph_Diamond(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	// Create diamond pattern: root -> (left, right) -> merge
	Register(Node[string]{
		ID:        "root",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "root", nil
		},
	})

	Register(Node[string]{
		ID:        "left",
		DependsOn: []ID{"root"},
		Run: func(ctx context.Context) (string, error) {
			return "left", nil
		},
	})

	Register(Node[string]{
		ID:        "right",
		DependsOn: []ID{"root"},
		Run: func(ctx context.Context) (string, error) {
			return "right", nil
		},
	})

	Register(Node[string]{
		ID:        "merge",
		DependsOn: []ID{"left", "right"},
		Run: func(ctx context.Context) (string, error) {
			return "merge", nil
		},
	})

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Verify all nodes are present
	for _, node := range []string{"root", "left", "right", "merge"} {
		if !strings.Contains(output, node) {
			t.Errorf("output should contain %q", node)
		}
	}
}

func TestPrintGraph_Fanout(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	// Create fanout: root -> (a, b, c, d, e)
	Register(Node[string]{
		ID:        "root",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "root", nil
		},
	})

	for _, id := range []ID{"a", "b", "c", "d", "e"} {
		Register(Node[string]{
			ID:        id,
			DependsOn: []ID{"root"},
			Run: func(ctx context.Context) (string, error) {
				return string(id), nil
			},
		})
	}

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Verify all nodes are present
	for _, node := range []string{"root", "a", "b", "c", "d", "e"} {
		if !strings.Contains(output, node) {
			t.Errorf("output should contain %q", node)
		}
	}
}

func TestPrintGraph_WithCacheable(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	Register(Node[string]{
		ID:        "cached",
		DependsOn: []ID{},
		Cacheable: true,
		Run: func(ctx context.Context) (string, error) {
			return "cached", nil
		},
	})

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Verify cacheable node is marked with *
	if !strings.Contains(output, "cached*") {
		t.Error("cacheable node should be marked with *")
	}
}

func TestPrintGraph_WithRegistryOption(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	// Create a custom registry
	customNodes := map[ID]node{
		"custom": {
			id:        "custom",
			dependsOn: []ID{},
			run: func(ctx context.Context) (any, error) {
				return "custom", nil
			},
		},
	}

	var buf bytes.Buffer
	err := PrintGraph(&buf, WithRegistry(customNodes))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom") {
		t.Error("output should contain 'custom'")
	}
}

func TestTopoSortLevelsForGraph_CycleDetection(t *testing.T) {
	nodes := map[ID]node{
		"a": {
			id:        "a",
			dependsOn: []ID{"b"},
			run:       nil,
		},
		"b": {
			id:        "b",
			dependsOn: []ID{"a"},
			run:       nil,
		},
	}

	_, err := topoSortLevels(nodes)
	if err == nil {
		t.Error("expected error for cycle, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestTopoSortLevelsForGraph_UnknownDependency(t *testing.T) {
	nodes := map[ID]node{
		"a": {
			id:        "a",
			dependsOn: []ID{"missing"},
			run:       nil,
		},
	}

	_, err := topoSortLevels(nodes)
	if err == nil {
		t.Error("expected error for unknown dependency, got nil")
	}
	if !strings.Contains(err.Error(), "unknown node") {
		t.Errorf("error should mention unknown node, got: %v", err)
	}
}

func TestPrintGraph_VisualOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping visual output test in short mode")
	}

	ResetRegistry()
	defer ResetRegistry()

	// Create diamond pattern: config -> (db, cache) -> api
	Register(Node[string]{
		ID:        "config",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "config", nil
		},
	})

	Register(Node[string]{
		ID:        "db",
		DependsOn: []ID{"config"},
		Run: func(ctx context.Context) (string, error) {
			return "db", nil
		},
	})

	Register(Node[string]{
		ID:        "cache",
		DependsOn: []ID{"config"},
		Run: func(ctx context.Context) (string, error) {
			return "cache", nil
		},
	})

	Register(Node[string]{
		ID:        "api",
		DependsOn: []ID{"db", "cache"},
		Run: func(ctx context.Context) (string, error) {
			return "api", nil
		},
	})

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	t.Logf("Graph output:\n%s", output)

	// Verify structure - should have box-drawing characters
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Error("output should contain box-drawing characters")
	}

	// Verify all nodes appear
	for _, node := range []string{"config", "db", "cache", "api"} {
		if !strings.Contains(output, node) {
			t.Errorf("output should contain %q", node)
		}
	}
}

func TestPrintGraph_WideGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wide graph test in short mode")
	}

	ResetRegistry()
	defer ResetRegistry()

	// Create a wide fanout that will wrap: root -> (svc1, svc2, svc3, svc4, svc5)
	Register(Node[string]{
		ID:        "root",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "root", nil
		},
	})

	for i := 1; i <= 5; i++ {
		id := ID(fmt.Sprintf("svc%d", i))
		Register(Node[string]{
			ID:        id,
			DependsOn: []ID{"root"},
			Run: func(ctx context.Context) (string, error) {
				return string(id), nil
			},
		})
	}

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	t.Logf("Wide graph output:\n%s", output)

	// Verify all nodes appear
	for i := 1; i <= 5; i++ {
		node := fmt.Sprintf("svc%d", i)
		if !strings.Contains(output, node) {
			t.Errorf("output should contain %q", node)
		}
	}
}

func TestPrintGraph_SimpleDiamond(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	// Simple diamond: a -> (b, c) -> d
	Register(Node[string]{
		ID:        "a",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "a", nil
		},
	})

	Register(Node[string]{
		ID:        "b",
		DependsOn: []ID{"a"},
		Run: func(ctx context.Context) (string, error) {
			return "b", nil
		},
	})

	Register(Node[string]{
		ID:        "c",
		DependsOn: []ID{"a"},
		Run: func(ctx context.Context) (string, error) {
			return "c", nil
		},
	})

	Register(Node[string]{
		ID:        "d",
		DependsOn: []ID{"b", "c"},
		Run: func(ctx context.Context) (string, error) {
			return "d", nil
		},
	})

	var buf bytes.Buffer
	err := PrintGraph(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	t.Logf("Simple diamond graph output:\n%s", output)
	t.Logf("Output length: %d characters", len(output))

	// Print with visible markers to debug
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		t.Logf("Line %d: %q", i, line)
	}

	// Verify all nodes appear
	for _, node := range []string{"a", "b", "c", "d"} {
		if !strings.Contains(output, node) {
			t.Errorf("output should contain %q", node)
		}
	}
}

func TestPrintGraph_TableDriven(t *testing.T) {
	type tc struct {
		name        string
		setupNodes  func()
		opts        []Option
		wantErr     bool
		errSubstr   string
		wantOutput  []string
		notWant     []string
		checkOutput func(t *testing.T, output string)
	}

	tests := map[string]tc{
		"empty registry": {
			name:       "empty registry",
			setupNodes: func() {},
			wantErr:    false,
			wantOutput: []string{"No nodes registered"},
		},
		"nil registry option": {
			name:       "nil registry option",
			setupNodes: func() {},
			opts: []Option{
				WithRegistry(nil),
			},
			wantErr:    false,
			wantOutput: []string{"No nodes registered"},
		},
		"cycle detection": {
			name: "cycle detection",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "a",
					DependsOn: []ID{"b"},
					Run: func(ctx context.Context) (string, error) {
						return "a", nil
					},
				})
				Register(Node[string]{
					ID:        "b",
					DependsOn: []ID{"a"},
					Run: func(ctx context.Context) (string, error) {
						return "b", nil
					},
				})
			},
			wantErr:   true,
			errSubstr: "cycle",
		},
		"unknown dependency": {
			name: "unknown dependency",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "a",
					DependsOn: []ID{"missing"},
					Run: func(ctx context.Context) (string, error) {
						return "a", nil
					},
				})
			},
			wantErr:   true,
			errSubstr: "unknown node",
		},
		"output contains box drawing": {
			name: "output contains box drawing",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "test",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "test", nil
					},
				})
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
					t.Error("output should contain box-drawing characters")
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ResetRegistry()
			defer ResetRegistry()

			tt.setupNodes()

			var buf bytes.Buffer
			err := PrintGraph(&buf, tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got: %q", want, output)
				}
			}

			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q", notWant)
				}
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestPrintMermaid_TableDriven(t *testing.T) {
	type tc struct {
		name        string
		setupNodes  func()
		opts        []Option
		wantErr     bool
		errSubstr   string
		wantOutput  []string
		notWant     []string
		checkOutput func(t *testing.T, output string)
	}

	tests := map[string]tc{
		"empty registry": {
			name:       "empty registry",
			setupNodes: func() {},
			wantErr:    false,
			wantOutput: []string{"graph TD"},
			notWant:    []string{"-->"},
		},
		"single node": {
			name: "single node",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "single",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "single", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD"},
			notWant:    []string{"single -->"},
		},
		"linear chain": {
			name: "linear chain",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "a",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "a", nil
					},
				})
				Register(Node[string]{
					ID:        "b",
					DependsOn: []ID{"a"},
					Run: func(ctx context.Context) (string, error) {
						return "b", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "a --> b"},
		},
		"diamond pattern": {
			name: "diamond pattern",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "root",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "root", nil
					},
				})
				Register(Node[string]{
					ID:        "left",
					DependsOn: []ID{"root"},
					Run: func(ctx context.Context) (string, error) {
						return "left", nil
					},
				})
				Register(Node[string]{
					ID:        "right",
					DependsOn: []ID{"root"},
					Run: func(ctx context.Context) (string, error) {
						return "right", nil
					},
				})
				Register(Node[string]{
					ID:        "merge",
					DependsOn: []ID{"left", "right"},
					Run: func(ctx context.Context) (string, error) {
						return "merge", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "root --> left", "root --> right", "left --> merge", "right --> merge"},
		},
		"cacheable nodes": {
			name: "cacheable nodes",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "cached",
					DependsOn: []ID{},
					Cacheable: true,
					Run: func(ctx context.Context) (string, error) {
						return "cached", nil
					},
				})
				Register(Node[string]{
					ID:        "normal",
					DependsOn: []ID{"cached"},
					Run: func(ctx context.Context) (string, error) {
						return "normal", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "cached --> normal", "style cached fill:#e1f5fe"},
		},
		"multiple cacheable nodes": {
			name: "multiple cacheable nodes",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "cached1",
					DependsOn: []ID{},
					Cacheable: true,
					Run: func(ctx context.Context) (string, error) {
						return "cached1", nil
					},
				})
				Register(Node[string]{
					ID:        "cached2",
					DependsOn: []ID{},
					Cacheable: true,
					Run: func(ctx context.Context) (string, error) {
						return "cached2", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "style cached1 fill:#e1f5fe", "style cached2 fill:#e1f5fe"},
		},
		"custom registry": {
			name:       "custom registry",
			setupNodes: func() {},
			opts: []Option{
				WithRegistry(map[ID]node{
					"custom": {
						id:        "custom",
						dependsOn: []ID{},
						run: func(ctx context.Context) (any, error) {
							return "custom", nil
						},
					},
				}),
			},
			wantErr:    false,
			wantOutput: []string{"graph TD"},
		},
		"nil registry option": {
			name:       "nil registry option",
			setupNodes: func() {},
			opts: []Option{
				WithRegistry(nil),
			},
			wantErr:    false,
			wantOutput: []string{"graph TD"},
		},
		"fanout pattern": {
			name: "fanout pattern",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "root",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "root", nil
					},
				})
				for _, id := range []ID{"a", "b", "c"} {
					Register(Node[string]{
						ID:        id,
						DependsOn: []ID{"root"},
						Run: func(ctx context.Context) (string, error) {
							return string(id), nil
						},
					})
				}
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "root --> a", "root --> b", "root --> c"},
		},
		"complex graph": {
			name: "complex graph",
			setupNodes: func() {
				Register(Node[string]{
					ID:        "config",
					DependsOn: []ID{},
					Run: func(ctx context.Context) (string, error) {
						return "config", nil
					},
				})
				Register(Node[string]{
					ID:        "db",
					DependsOn: []ID{"config"},
					Cacheable: true,
					Run: func(ctx context.Context) (string, error) {
						return "db", nil
					},
				})
				Register(Node[string]{
					ID:        "cache",
					DependsOn: []ID{"config"},
					Cacheable: true,
					Run: func(ctx context.Context) (string, error) {
						return "cache", nil
					},
				})
				Register(Node[string]{
					ID:        "api",
					DependsOn: []ID{"db", "cache"},
					Run: func(ctx context.Context) (string, error) {
						return "api", nil
					},
				})
			},
			wantErr:    false,
			wantOutput: []string{"graph TD", "config --> db", "config --> cache", "db --> api", "cache --> api", "style db fill:#e1f5fe", "style cache fill:#e1f5fe"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ResetRegistry()
			defer ResetRegistry()

			tt.setupNodes()

			var buf bytes.Buffer
			err := PrintMermaid(&buf, tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got: %q", want, output)
				}
			}

			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q", notWant)
				}
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestTopoSortLevelsForGraph_TableDriven(t *testing.T) {
	type tc struct {
		name        string
		nodes       map[ID]node
		wantErr     bool
		errSubstr   string
		wantLevels  [][]ID
		checkLevels func(t *testing.T, levels [][]ID)
	}

	tests := map[string]tc{
		"empty graph": {
			name:       "empty graph",
			nodes:      map[ID]node{},
			wantErr:    false,
			wantLevels: [][]ID{},
		},
		"single node": {
			name: "single node",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{},
					run:       nil,
				},
			},
			wantErr:    false,
			wantLevels: [][]ID{{"a"}},
		},
		"linear chain": {
			name: "linear chain",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{},
					run:       nil,
				},
				"b": {
					id:        "b",
					dependsOn: []ID{"a"},
					run:       nil,
				},
				"c": {
					id:        "c",
					dependsOn: []ID{"b"},
					run:       nil,
				},
			},
			wantErr: false,
			wantLevels: [][]ID{
				{"a"},
				{"b"},
				{"c"},
			},
		},
		"diamond pattern": {
			name: "diamond pattern",
			nodes: map[ID]node{
				"root": {
					id:        "root",
					dependsOn: []ID{},
					run:       nil,
				},
				"left": {
					id:        "left",
					dependsOn: []ID{"root"},
					run:       nil,
				},
				"right": {
					id:        "right",
					dependsOn: []ID{"root"},
					run:       nil,
				},
				"merge": {
					id:        "merge",
					dependsOn: []ID{"left", "right"},
					run:       nil,
				},
			},
			wantErr: false,
			wantLevels: [][]ID{
				{"root"},
				{"left", "right"},
				{"merge"},
			},
		},
		"fanout pattern": {
			name: "fanout pattern",
			nodes: map[ID]node{
				"root": {
					id:        "root",
					dependsOn: []ID{},
					run:       nil,
				},
				"a": {
					id:        "a",
					dependsOn: []ID{"root"},
					run:       nil,
				},
				"b": {
					id:        "b",
					dependsOn: []ID{"root"},
					run:       nil,
				},
				"c": {
					id:        "c",
					dependsOn: []ID{"root"},
					run:       nil,
				},
			},
			wantErr: false,
			wantLevels: [][]ID{
				{"root"},
				{"a", "b", "c"},
			},
		},
		"multiple independent roots": {
			name: "multiple independent roots",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{},
					run:       nil,
				},
				"b": {
					id:        "b",
					dependsOn: []ID{},
					run:       nil,
				},
				"c": {
					id:        "c",
					dependsOn: []ID{"a"},
					run:       nil,
				},
			},
			wantErr: false,
			wantLevels: [][]ID{
				{"a", "b"},
				{"c"},
			},
		},
		"cycle detection": {
			name: "cycle detection",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{"b"},
					run:       nil,
				},
				"b": {
					id:        "b",
					dependsOn: []ID{"a"},
					run:       nil,
				},
			},
			wantErr:   true,
			errSubstr: "cycle",
		},
		"self cycle": {
			name: "self cycle",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{"a"},
					run:       nil,
				},
			},
			wantErr:   true,
			errSubstr: "cycle",
		},
		"long cycle": {
			name: "long cycle",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{"b"},
					run:       nil,
				},
				"b": {
					id:        "b",
					dependsOn: []ID{"c"},
					run:       nil,
				},
				"c": {
					id:        "c",
					dependsOn: []ID{"a"},
					run:       nil,
				},
			},
			wantErr:   true,
			errSubstr: "cycle",
		},
		"unknown dependency": {
			name: "unknown dependency",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{"missing"},
					run:       nil,
				},
			},
			wantErr:   true,
			errSubstr: "unknown node",
		},
		"multiple unknown dependencies": {
			name: "multiple unknown dependencies",
			nodes: map[ID]node{
				"a": {
					id:        "a",
					dependsOn: []ID{"missing1", "missing2"},
					run:       nil,
				},
			},
			wantErr:   true,
			errSubstr: "unknown node",
		},
		"complex graph": {
			name: "complex graph",
			nodes: map[ID]node{
				"config": {
					id:        "config",
					dependsOn: []ID{},
					run:       nil,
				},
				"db": {
					id:        "db",
					dependsOn: []ID{"config"},
					run:       nil,
				},
				"cache": {
					id:        "cache",
					dependsOn: []ID{"config"},
					run:       nil,
				},
				"api": {
					id:        "api",
					dependsOn: []ID{"db", "cache"},
					run:       nil,
				},
			},
			wantErr: false,
			wantLevels: [][]ID{
				{"config"},
				{"cache", "db"},
				{"api"},
			},
		},
		"deterministic ordering": {
			name: "deterministic ordering",
			nodes: map[ID]node{
				"z": {
					id:        "z",
					dependsOn: []ID{},
					run:       nil,
				},
				"a": {
					id:        "a",
					dependsOn: []ID{},
					run:       nil,
				},
				"m": {
					id:        "m",
					dependsOn: []ID{},
					run:       nil,
				},
			},
			wantErr: false,
			checkLevels: func(t *testing.T, levels [][]ID) {
				if len(levels) != 1 {
					t.Fatalf("expected 1 level, got %d", len(levels))
				}
				if len(levels[0]) != 3 {
					t.Fatalf("expected 3 nodes in level, got %d", len(levels[0]))
				}
				// Should be sorted alphabetically
				if levels[0][0] != "a" || levels[0][1] != "m" || levels[0][2] != "z" {
					t.Errorf("expected sorted order [a, m, z], got %v", levels[0])
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			levels, err := topoSortLevels(tt.nodes)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantLevels != nil {
				if len(levels) != len(tt.wantLevels) {
					t.Fatalf("got %d levels, want %d", len(levels), len(tt.wantLevels))
				}
				for i, wantLevel := range tt.wantLevels {
					if i >= len(levels) {
						t.Fatalf("level %d missing", i)
					}
					gotLevel := levels[i]
					if len(gotLevel) != len(wantLevel) {
						t.Errorf("level %d: got %d nodes, want %d", i, len(gotLevel), len(wantLevel))
					}
					gotSet := make(map[ID]bool)
					for _, id := range gotLevel {
						gotSet[id] = true
					}
					for _, wantID := range wantLevel {
						if !gotSet[wantID] {
							t.Errorf("level %d: missing node %q", i, wantID)
						}
					}
				}
			}

			if tt.checkLevels != nil {
				tt.checkLevels(t, levels)
			}
		})
	}
}
