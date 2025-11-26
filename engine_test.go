package graft

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// makeNode is a test helper that creates a type-erased node for direct engine testing.
func makeNode(id ID, dependsOn []ID, run func(ctx context.Context) (any, error)) node {
	return node{
		id:        id,
		dependsOn: dependsOn,
		run:       run,
	}
}

func TestNew(t *testing.T) {
	type tc struct {
		nodes     map[ID]node
		wantCount int
	}

	tests := map[string]tc{
		"empty nodes": {
			nodes:     map[ID]node{},
			wantCount: 0,
		},
		"single node": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return nil, nil }),
			},
			wantCount: 1,
		},
		"multiple nodes": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return nil, nil }),
				"b": makeNode("b", nil, func(ctx context.Context) (any, error) { return nil, nil }),
				"c": makeNode("c", nil, func(ctx context.Context) (any, error) { return nil, nil }),
			},
			wantCount: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := New(tt.nodes)
			if e == nil {
				t.Fatal("New returned nil")
			}
			if len(e.nodes) != tt.wantCount {
				t.Errorf("got %d nodes, want %d", len(e.nodes), tt.wantCount)
			}
		})
	}
}

func TestEngineRun(t *testing.T) {
	type tc struct {
		nodes       map[ID]node
		wantErr     bool
		errSubstr   string
		wantResults map[ID]any
	}

	tests := map[string]tc{
		"empty graph": {
			nodes:       map[ID]node{},
			wantErr:     false,
			wantResults: map[ID]any{},
		},
		"single node": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "resultA", nil }),
			},
			wantErr:     false,
			wantResults: map[ID]any{"a": "resultA"},
		},
		"two independent nodes": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
				"b": makeNode("b", nil, func(ctx context.Context) (any, error) { return 2, nil }),
			},
			wantErr:     false,
			wantResults: map[ID]any{"a": 1, "b": 2},
		},
		"linear dependency chain": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 10, nil }),
				"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
					val, _ := Dep[int](ctx, "a")
					return val * 2, nil
				}),
				"c": makeNode("c", []ID{"b"}, func(ctx context.Context) (any, error) {
					val, _ := Dep[int](ctx, "b")
					return val * 2, nil
				}),
			},
			wantErr:     false,
			wantResults: map[ID]any{"a": 10, "b": 20, "c": 40},
		},
		"diamond dependency": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
				"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
					val, _ := Dep[int](ctx, "a")
					return val + 10, nil
				}),
				"c": makeNode("c", []ID{"a"}, func(ctx context.Context) (any, error) {
					val, _ := Dep[int](ctx, "a")
					return val + 100, nil
				}),
				"d": makeNode("d", []ID{"b", "c"}, func(ctx context.Context) (any, error) {
					b, _ := Dep[int](ctx, "b")
					c, _ := Dep[int](ctx, "c")
					return b + c, nil
				}),
			},
			wantErr:     false,
			wantResults: map[ID]any{"a": 1, "b": 11, "c": 101, "d": 112},
		},
		"node returns error": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) {
					return nil, errors.New("intentional failure")
				}),
			},
			wantErr:   true,
			errSubstr: "node a: intentional failure",
		},
		"dependency on unknown node": {
			nodes: map[ID]node{
				"a": makeNode("a", []ID{"unknown"}, func(ctx context.Context) (any, error) { return nil, nil }),
			},
			wantErr:   true,
			errSubstr: "depends on unknown node",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := New(tt.nodes)
			err := e.Run(context.Background())

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

			results := e.Results()
			if len(results) != len(tt.wantResults) {
				t.Errorf("got %d results, want %d", len(results), len(tt.wantResults))
			}
			for k, want := range tt.wantResults {
				got, ok := results[k]
				if !ok {
					t.Errorf("missing result for node %q", k)
					continue
				}
				if got != want {
					t.Errorf("result[%q] = %v, want %v", k, got, want)
				}
			}
		})
	}
}

func TestEngineRunContextCancellation(t *testing.T) {
	type tc struct {
		cancelBefore bool
		wantErr      bool
	}

	tests := map[string]tc{
		"context already cancelled": {
			cancelBefore: true,
			wantErr:      true,
		},
		"context not cancelled": {
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			nodes := map[ID]node{
				"a": makeNode("a", nil, func(ctx context.Context) (any, error) {
					return "done", nil
				}),
			}

			if tt.cancelBefore {
				cancel()
			}

			e := New(nodes)
			err := e.Run(ctx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEngineRunContextCancelledBetweenLevels(t *testing.T) {
	// Test that context cancellation is checked between levels
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	level1Done := make(chan struct{})
	cancelComplete := make(chan struct{})

	nodes := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) {
			close(level1Done)
			// Wait for cancel to complete before returning, ensuring the context
			// is cancelled before runLevel returns and the loop checks ctx.Err()
			<-cancelComplete
			return "a", nil
		}),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
			// This should not run if context is cancelled between levels
			t.Error("node b should not have run - context was cancelled between levels")
			return "b", nil
		}),
	}

	e := New(nodes)

	// Cancel after first level signals but before node "a" returns
	go func() {
		<-level1Done
		cancel()
		close(cancelComplete)
	}()

	err := e.Run(ctx)

	// Must get a context error since we cancelled between levels
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestTopoSortLevels(t *testing.T) {
	type tc struct {
		nodes      map[ID]node
		wantLevels int
		wantErr    bool
		errSubstr  string
	}

	tests := map[string]tc{
		"empty graph": {
			nodes:      map[ID]node{},
			wantLevels: 0,
			wantErr:    false,
		},
		"single node": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, nil),
			},
			wantLevels: 1,
			wantErr:    false,
		},
		"two independent nodes - same level": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, nil),
				"b": makeNode("b", nil, nil),
			},
			wantLevels: 1,
			wantErr:    false,
		},
		"linear chain - three levels": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, nil),
				"b": makeNode("b", []ID{"a"}, nil),
				"c": makeNode("c", []ID{"b"}, nil),
			},
			wantLevels: 3,
			wantErr:    false,
		},
		"diamond - three levels": {
			nodes: map[ID]node{
				"a": makeNode("a", nil, nil),
				"b": makeNode("b", []ID{"a"}, nil),
				"c": makeNode("c", []ID{"a"}, nil),
				"d": makeNode("d", []ID{"b", "c"}, nil),
			},
			wantLevels: 3,
			wantErr:    false,
		},
		"cycle detection - self loop": {
			nodes: map[ID]node{
				"a": makeNode("a", []ID{"a"}, nil),
			},
			wantErr:   true,
			errSubstr: "cycle detected",
		},
		"cycle detection - two node cycle": {
			nodes: map[ID]node{
				"a": makeNode("a", []ID{"b"}, nil),
				"b": makeNode("b", []ID{"a"}, nil),
			},
			wantErr:   true,
			errSubstr: "cycle detected",
		},
		"cycle detection - three node cycle": {
			nodes: map[ID]node{
				"a": makeNode("a", []ID{"c"}, nil),
				"b": makeNode("b", []ID{"a"}, nil),
				"c": makeNode("c", []ID{"b"}, nil),
			},
			wantErr:   true,
			errSubstr: "cycle detected",
		},
		"unknown dependency": {
			nodes: map[ID]node{
				"a": makeNode("a", []ID{"missing"}, nil),
			},
			wantErr:   true,
			errSubstr: "depends on unknown node",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := New(tt.nodes)
			levels, err := e.topoSortLevels()

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

			if len(levels) != tt.wantLevels {
				t.Errorf("got %d levels, want %d", len(levels), tt.wantLevels)
			}

			// Verify all nodes are present exactly once
			seen := make(map[ID]bool)
			for _, level := range levels {
				for _, id := range level {
					if seen[id] {
						t.Errorf("node %q appears multiple times", id)
					}
					seen[id] = true
				}
			}
			if len(seen) != len(tt.nodes) {
				t.Errorf("got %d unique nodes, want %d", len(seen), len(tt.nodes))
			}
		})
	}
}

func TestParallelExecution(t *testing.T) {
	// Test that nodes in the same level actually run in parallel
	var concurrentCount atomic.Int32
	var maxConcurrent atomic.Int32

	trackConcurrency := func() func() {
		current := concurrentCount.Add(1)
		for {
			old := maxConcurrent.Load()
			if current <= old || maxConcurrent.CompareAndSwap(old, current) {
				break
			}
		}
		return func() {
			concurrentCount.Add(-1)
		}
	}

	nodes := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) {
			done := trackConcurrency()
			defer done()
			time.Sleep(50 * time.Millisecond)
			return "a", nil
		}),
		"b": makeNode("b", nil, func(ctx context.Context) (any, error) {
			done := trackConcurrency()
			defer done()
			time.Sleep(50 * time.Millisecond)
			return "b", nil
		}),
		"c": makeNode("c", nil, func(ctx context.Context) (any, error) {
			done := trackConcurrency()
			defer done()
			time.Sleep(50 * time.Millisecond)
			return "c", nil
		}),
	}

	e := New(nodes)
	start := time.Now()
	err := e.Run(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If running in parallel, should complete in ~50ms, not ~150ms
	if elapsed > 120*time.Millisecond {
		t.Errorf("execution took %v, expected parallel execution under 120ms", elapsed)
	}

	if maxConcurrent.Load() < 3 {
		t.Errorf("max concurrent was %d, expected 3 for parallel execution", maxConcurrent.Load())
	}
}

func TestResultsThreadSafety(t *testing.T) {
	// Ensure Results() returns a copy, not the internal map
	nodes := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "value", nil }),
	}

	e := New(nodes)
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results1 := e.Results()
	results2 := e.Results()

	// Modify results1
	results1["a"] = "modified"

	// results2 should be unaffected
	if results2["a"] != "value" {
		t.Error("Results() did not return a copy; modification affected other copy")
	}
}

func TestCopyResults(t *testing.T) {
	type tc struct {
		initial map[ID]any
	}

	tests := map[string]tc{
		"empty results": {
			initial: map[ID]any{},
		},
		"single result": {
			initial: map[ID]any{"a": 1},
		},
		"multiple results": {
			initial: map[ID]any{"a": 1, "b": "two", "c": 3.0},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := &Engine{
				nodes:   make(map[ID]node),
				results: make(results),
			}
			for k, v := range tt.initial {
				e.results[k] = v
			}

			cp := e.copyResults()

			// Verify it's a copy with same values
			if len(cp) != len(tt.initial) {
				t.Errorf("copy has %d items, want %d", len(cp), len(tt.initial))
			}

			// Modify copy, ensure original unchanged
			cp["newKey"] = "newValue"
			if _, exists := e.results["newKey"]; exists {
				t.Error("modifying copy affected original")
			}
		})
	}
}
