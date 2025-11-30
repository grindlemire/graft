package graft

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// makeNode is a test helper that creates a type-erased node for direct testing.
func makeNode(id ID, dependsOn []ID, run func(ctx context.Context) (any, error)) node {
	return node{
		id:        id,
		dependsOn: dependsOn,
		run:       run,
	}
}

func TestExecute(t *testing.T) {
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
					val, _ := depByID[int](ctx, "a")
					return val * 2, nil
				}),
				"c": makeNode("c", []ID{"b"}, func(ctx context.Context) (any, error) {
					val, _ := depByID[int](ctx, "b")
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
					val, _ := depByID[int](ctx, "a")
					return val + 10, nil
				}),
				"c": makeNode("c", []ID{"a"}, func(ctx context.Context) (any, error) {
					val, _ := depByID[int](ctx, "a")
					return val + 100, nil
				}),
				"d": makeNode("d", []ID{"b", "c"}, func(ctx context.Context) (any, error) {
					b, _ := depByID[int](ctx, "b")
					c, _ := depByID[int](ctx, "c")
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
			results, err := Execute(context.Background(), WithRegistry(tt.nodes))

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

func TestExecuteContextCancellation(t *testing.T) {
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

			_, err := Execute(ctx, WithRegistry(nodes))

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

func TestExecuteContextCancelledBetweenLevels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	level1Done := make(chan struct{})
	cancelComplete := make(chan struct{})

	nodes := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) {
			close(level1Done)
			<-cancelComplete
			return "a", nil
		}),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
			t.Error("node b should not have run - context was cancelled between levels")
			return "b", nil
		}),
	}

	go func() {
		<-level1Done
		cancel()
		close(cancelComplete)
	}()

	_, err := Execute(ctx, WithRegistry(nodes))

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestExecuteFor(t *testing.T) {
	type tc struct {
		registry    map[ID]node
		targets     []ID
		wantResults map[ID]any
		wantErr     bool
		errSubstr   string
	}

	baseRegistry := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "a", nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) { return "b", nil }),
		"c": makeNode("c", []ID{"a"}, func(ctx context.Context) (any, error) { return "c", nil }),
		"d": makeNode("d", []ID{"b", "c"}, func(ctx context.Context) (any, error) { return "d", nil }),
		"e": makeNode("e", nil, func(ctx context.Context) (any, error) { return "e", nil }),
	}

	tests := map[string]tc{
		"single target no deps": {
			registry:    baseRegistry,
			targets:     []ID{"a"},
			wantResults: map[ID]any{"a": "a"},
		},
		"single target with one dep": {
			registry:    baseRegistry,
			targets:     []ID{"b"},
			wantResults: map[ID]any{"a": "a", "b": "b"},
		},
		"single target with transitive deps": {
			registry:    baseRegistry,
			targets:     []ID{"d"},
			wantResults: map[ID]any{"a": "a", "b": "b", "c": "c", "d": "d"},
		},
		"multiple targets": {
			registry:    baseRegistry,
			targets:     []ID{"b", "c"},
			wantResults: map[ID]any{"a": "a", "b": "b", "c": "c"},
		},
		"multiple targets with overlap": {
			registry:    baseRegistry,
			targets:     []ID{"d", "e"},
			wantResults: map[ID]any{"a": "a", "b": "b", "c": "c", "d": "d", "e": "e"},
		},
		"unknown target": {
			registry:  baseRegistry,
			targets:   []ID{"unknown"},
			wantErr:   true,
			errSubstr: "unknown node: unknown",
		},
		"mixed known and unknown targets": {
			registry:  baseRegistry,
			targets:   []ID{"a", "unknown"},
			wantErr:   true,
			errSubstr: "unknown node: unknown",
		},
		"empty targets": {
			registry:    baseRegistry,
			targets:     []ID{},
			wantResults: map[ID]any{},
		},
		"duplicate targets": {
			registry:    baseRegistry,
			targets:     []ID{"a", "a", "a"},
			wantResults: map[ID]any{"a": "a"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := executeForIDs(context.Background(), tt.targets, WithRegistry(tt.registry))

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

func TestExecuteForWithDeps(t *testing.T) {
	// Test that ExecuteFor correctly executes nodes that use Dep
	registry := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) {
			v, _ := depByID[int](ctx, "a")
			return v * 2, nil
		}),
		"c": makeNode("c", []ID{"b"}, func(ctx context.Context) (any, error) {
			v, _ := depByID[int](ctx, "b")
			return v * 2, nil
		}),
	}

	results, err := executeForIDs(context.Background(), []ID{"c"}, WithRegistry(registry))
	if err != nil {
		t.Fatalf("ExecuteFor error: %v", err)
	}

	wantResults := map[ID]any{"a": 1, "b": 2, "c": 4}
	if len(results) != len(wantResults) {
		t.Errorf("got %d results, want %d", len(results), len(wantResults))
	}

	for k, want := range wantResults {
		got, ok := results[k]
		if !ok {
			t.Errorf("missing result for %q", k)
			continue
		}
		if got != want {
			t.Errorf("result[%q] = %v, want %v", k, got, want)
		}
	}
}

func TestExecuteForDoesNotMutateRegistry(t *testing.T) {
	reg := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return 1, nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) { return 2, nil }),
	}

	originalLen := len(reg)

	_, err := executeForIDs(context.Background(), []ID{"b"}, WithRegistry(reg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reg) != originalLen {
		t.Errorf("registry was mutated: got %d nodes, want %d", len(reg), originalLen)
	}
}

func TestMergeRegistry(t *testing.T) {
	// Create a base registry
	baseRegistry := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "original_a", nil }),
		"b": makeNode("b", []ID{"a"}, func(ctx context.Context) (any, error) { return "original_b", nil }),
	}

	// Override node "a" with a different implementation
	overrides := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "overridden_a", nil }),
	}

	// Manually merge to simulate what MergeRegistry does
	merged := make(map[ID]node)
	for k, v := range baseRegistry {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}

	results, err := executeForIDs(context.Background(), []ID{"b"}, WithRegistry(merged))
	if err != nil {
		t.Fatalf("ExecuteFor error: %v", err)
	}

	// Node "a" should have overridden value
	if results["a"] != "overridden_a" {
		t.Errorf("expected overridden_a, got %v", results["a"])
	}

	// Node "b" should still work (uses original implementation)
	if results["b"] != "original_b" {
		t.Errorf("expected original_b, got %v", results["b"])
	}
}

func TestTopoSortLevels(t *testing.T) {
	type tc struct {
		nodes     map[ID]node
		wantErr   bool
		errSubstr string
	}

	tests := map[string]tc{
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
			_, err := Execute(context.Background(), WithRegistry(tt.nodes))

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
		})
	}
}

func TestParallelExecution(t *testing.T) {
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

	start := time.Now()
	_, err := Execute(context.Background(), WithRegistry(nodes))
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

func TestResultsAreCopied(t *testing.T) {
	nodes := map[ID]node{
		"a": makeNode("a", nil, func(ctx context.Context) (any, error) { return "value", nil }),
	}

	results1, err := Execute(context.Background(), WithRegistry(nodes))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results2, err := Execute(context.Background(), WithRegistry(nodes))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify results1
	results1["a"] = "modified"

	// results2 should be unaffected
	if results2["a"] != "value" {
		t.Error("Results from separate Execute calls should be independent")
	}
}

// Test types for ExecuteFor[T] tests
type testConfigOutput struct {
	Host string
	Port int
}

type testDBOutput struct {
	Connected bool
	PoolSize  int
}

type testUnregisteredOutput struct {
	Value string
}

func TestExecuteForTyped(t *testing.T) {
	// Reset registry for clean test
	ResetRegistry()

	// Register test nodes
	Register(Node[testConfigOutput]{
		ID:        "test_config",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (testConfigOutput, error) {
			return testConfigOutput{Host: "localhost", Port: 5432}, nil
		},
	})

	Register(Node[testDBOutput]{
		ID:        "test_db",
		DependsOn: []ID{"test_config"},
		Run: func(ctx context.Context) (testDBOutput, error) {
			cfg, err := Dep[testConfigOutput](ctx)
			if err != nil {
				return testDBOutput{}, err
			}
			return testDBOutput{Connected: cfg.Host != "", PoolSize: 10}, nil
		},
	})

	t.Run("returns typed result", func(t *testing.T) {
		cfg, results, err := ExecuteFor[testConfigOutput](context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Host != "localhost" {
			t.Errorf("cfg.Host = %q, want %q", cfg.Host, "localhost")
		}
		if cfg.Port != 5432 {
			t.Errorf("cfg.Port = %d, want %d", cfg.Port, 5432)
		}

		// Results map should also contain the config
		if results == nil {
			t.Fatal("results map is nil")
		}
		if _, ok := results["test_config"]; !ok {
			t.Error("results map missing test_config")
		}
	})

	t.Run("returns results with dependencies", func(t *testing.T) {
		db, results, err := ExecuteFor[testDBOutput](context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check typed result
		if !db.Connected {
			t.Error("db.Connected = false, want true")
		}
		if db.PoolSize != 10 {
			t.Errorf("db.PoolSize = %d, want %d", db.PoolSize, 10)
		}

		// Check results map contains both nodes
		if len(results) != 2 {
			t.Errorf("results has %d entries, want 2", len(results))
		}
		if _, ok := results["test_config"]; !ok {
			t.Error("results map missing test_config")
		}
		if _, ok := results["test_db"]; !ok {
			t.Error("results map missing test_db")
		}

		// Verify we can extract typed results from the map
		cfg, err := Result[testConfigOutput](results)
		if err != nil {
			t.Fatalf("Result[testConfigOutput] error: %v", err)
		}
		if cfg.Host != "localhost" {
			t.Errorf("cfg.Host from results = %q, want %q", cfg.Host, "localhost")
		}
	})

	t.Run("error on unregistered type", func(t *testing.T) {
		_, _, err := ExecuteFor[testUnregisteredOutput](context.Background())
		if err == nil {
			t.Fatal("expected error for unregistered type, got nil")
		}
		if !containsSubstr(err.Error(), "not registered") {
			t.Errorf("error %q should contain 'not registered'", err.Error())
		}
	})
}

func TestWithCacheOption(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": makeNode("counter", nil, func(ctx context.Context) (any, error) {
			execCount.Add(1)
			return "executed", nil
		}),
	}
	nodes["counter"] = node{
		id:        "counter",
		dependsOn: nil,
		cacheable: true,
		run: func(ctx context.Context) (any, error) {
			execCount.Add(1)
			return "executed", nil
		},
	}

	customCache := NewMemoryCache()

	// First execution with custom cache
	_, err := Execute(context.Background(), WithRegistry(nodes), WithCache(customCache))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution - should use cache
	_, err = Execute(context.Background(), WithRegistry(nodes), WithCache(customCache))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected still 1 execution (cached), got %d", execCount.Load())
	}

	// Verify cache contains the value
	val, found, _ := customCache.Get(context.Background(), "counter")
	if !found {
		t.Fatal("expected value in custom cache")
	}
	if val != "executed" {
		t.Fatalf("expected 'executed', got %v", val)
	}
}

func TestIgnoreCacheOption(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": {
			id:        "counter",
			dependsOn: nil,
			cacheable: true,
			run: func(ctx context.Context) (any, error) {
				execCount.Add(1)
				return "executed", nil
			},
		},
	}

	customCache := NewMemoryCache()

	// First execution - populates cache
	_, err := Execute(context.Background(), WithRegistry(nodes), WithCache(customCache))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution with IgnoreCache - should re-execute
	_, err = Execute(context.Background(), WithRegistry(nodes), WithCache(customCache), IgnoreCache("counter"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 2 {
		t.Fatalf("expected 2 executions (ignored cache), got %d", execCount.Load())
	}
}

func TestDisableCacheOption(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": {
			id:        "counter",
			dependsOn: nil,
			cacheable: true,
			run: func(ctx context.Context) (any, error) {
				execCount.Add(1)
				return "executed", nil
			},
		},
	}

	// First execution with DisableCache
	_, err := Execute(context.Background(), WithRegistry(nodes), DisableCache())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution with DisableCache - should run again since cache is disabled
	_, err = Execute(context.Background(), WithRegistry(nodes), DisableCache())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 2 {
		t.Fatalf("expected 2 executions (cache disabled), got %d", execCount.Load())
	}
}

func TestMergeRegistryOption(t *testing.T) {
	// Reset the global registry
	ResetRegistry()

	// Register a node in the global registry
	Register(Node[string]{
		ID:        "global_node",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (string, error) {
			return "global_value", nil
		},
	})

	// Override with MergeRegistry
	overrides := map[ID]node{
		"global_node": makeNode("global_node", nil, func(ctx context.Context) (any, error) {
			return "overridden_value", nil
		}),
	}

	results, err := Execute(context.Background(), MergeRegistry(overrides))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have the overridden value
	if results["global_node"] != "overridden_value" {
		t.Errorf("expected 'overridden_value', got %v", results["global_node"])
	}
}

// Test types for Patch tests
type patchTestConfig struct {
	Host string
	Port int
}

type patchTestDB struct {
	Connected bool
}

func TestPatchValueOption(t *testing.T) {
	ResetRegistry()

	var originalRan atomic.Bool

	// Register original node
	Register(Node[patchTestConfig]{
		ID:        "patch_config",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (patchTestConfig, error) {
			originalRan.Store(true)
			return patchTestConfig{Host: "original", Port: 5432}, nil
		},
	})

	// Patch with static value
	patchedValue := patchTestConfig{Host: "patched", Port: 9999}
	results, err := Execute(context.Background(), PatchValue[patchTestConfig](patchedValue))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify patched value is returned
	cfg, err := Result[patchTestConfig](results)
	if err != nil {
		t.Fatalf("Result error: %v", err)
	}

	if cfg.Host != "patched" {
		t.Errorf("cfg.Host = %q, want %q", cfg.Host, "patched")
	}
	if cfg.Port != 9999 {
		t.Errorf("cfg.Port = %d, want %d", cfg.Port, 9999)
	}

	// Verify original Run was never called
	if originalRan.Load() {
		t.Error("original Run function should not have been called")
	}
}

func TestPatchOption(t *testing.T) {
	ResetRegistry()

	var originalRan atomic.Bool
	var patchedRan atomic.Bool

	// Register original nodes
	Register(Node[patchTestConfig]{
		ID:        "patch_config",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (patchTestConfig, error) {
			return patchTestConfig{Host: "original", Port: 5432}, nil
		},
	})

	Register(Node[patchTestDB]{
		ID:        "patch_db",
		DependsOn: []ID{"patch_config"},
		Run: func(ctx context.Context) (patchTestDB, error) {
			originalRan.Store(true)
			return patchTestDB{Connected: false}, nil
		},
	})

	// Patch db node with custom implementation
	results, err := Execute(context.Background(),
		Patch[patchTestDB](Node[patchTestDB]{
			DependsOn: []ID{"patch_config"},
			Run: func(ctx context.Context) (patchTestDB, error) {
				patchedRan.Store(true)
				// Verify we can still access dependencies
				cfg, err := Dep[patchTestConfig](ctx)
				if err != nil {
					return patchTestDB{}, err
				}
				return patchTestDB{Connected: cfg.Host == "original"}, nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify patched node ran
	if !patchedRan.Load() {
		t.Error("patched Run function should have been called")
	}

	// Verify original didn't run
	if originalRan.Load() {
		t.Error("original Run function should not have been called")
	}

	// Verify result
	db, err := Result[patchTestDB](results)
	if err != nil {
		t.Fatalf("Result error: %v", err)
	}

	if !db.Connected {
		t.Error("db.Connected = false, want true (patched logic should have set it based on config)")
	}
}
