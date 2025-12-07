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

	_, err := topoSortLevelsForGraph(nodes)
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

	_, err := topoSortLevelsForGraph(nodes)
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
