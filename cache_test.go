package graft

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestResetDefaultCache(t *testing.T) {
	ctx := context.Background()

	// Add something to the default cache
	if err := defaultCache.Set(ctx, "test-key", "test-value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's there
	val, found, _ := defaultCache.Get(ctx, "test-key")
	if !found || val != "test-value" {
		t.Fatal("expected value in default cache before reset")
	}

	// Reset
	ResetDefaultCache()

	// Verify it's gone
	_, found, _ = defaultCache.Get(ctx, "test-key")
	if found {
		t.Fatal("expected empty default cache after reset")
	}
}

func TestMemoryCache(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()

	// Test miss
	val, found, err := cache.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected cache miss")
	}
	if val != nil {
		t.Fatalf("expected nil value, got %v", val)
	}

	// Test set and hit
	if err := cache.Set(ctx, "foo", "bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, found, err = cache.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if val != "bar" {
		t.Fatalf("expected 'bar', got %v", val)
	}

	// Test delete
	cache.Delete("foo")
	_, found, _ = cache.Get(ctx, "foo")
	if found {
		t.Fatal("expected cache miss after delete")
	}

	// Test clear
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)
	cache.Clear()

	if len(cache.Snapshot()) != 0 {
		t.Fatal("expected empty cache after clear")
	}
}

func TestCacheableNodeIsCached(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": {
			id:        "counter",
			dependsOn: nil,
			cacheable: true, // Mark as cacheable
			run: func(ctx context.Context) (any, error) {
				execCount.Add(1)
				return "executed", nil
			},
		},
	}

	cache := NewMemoryCache()
	cfg := &config{
		registry: nodes,
		cache:    cache,
	}

	ctx := context.Background()

	// First execution - should run the node
	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution - should use cache
	engine = newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected still 1 execution (cached), got %d", execCount.Load())
	}

	// Verify result is correct from cache
	if engine.results["counter"] != "executed" {
		t.Fatalf("expected 'executed', got %v", engine.results["counter"])
	}
}

func TestNonCacheableNodeAlwaysRuns(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": {
			id:        "counter",
			dependsOn: nil,
			cacheable: false, // Not cacheable (default)
			run: func(ctx context.Context) (any, error) {
				execCount.Add(1)
				return "executed", nil
			},
		},
	}

	cache := NewMemoryCache()
	cfg := &config{
		registry: nodes,
		cache:    cache,
	}

	ctx := context.Background()

	// First execution
	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution - should run again (not cacheable)
	engine = newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 2 {
		t.Fatalf("expected 2 executions (not cacheable), got %d", execCount.Load())
	}
}

func TestIgnoreCacheForCacheableNode(t *testing.T) {
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

	cache := NewMemoryCache()
	ctx := context.Background()

	// First execution - populates cache
	cfg := &config{
		registry: nodes,
		cache:    cache,
	}
	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution with IgnoreCache - should re-execute
	cfg = &config{
		registry:       nodes,
		cache:          cache,
		ignoreCacheFor: map[ID]bool{"counter": true},
	}
	engine = newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 2 {
		t.Fatalf("expected 2 executions (ignored cache), got %d", execCount.Load())
	}
}

func TestMixedCacheableNodes(t *testing.T) {
	var configExec, dbExec, handlerExec atomic.Int32

	nodes := map[ID]node{
		"config": {
			id:        "config",
			dependsOn: nil,
			cacheable: true, // Startup node - cacheable
			run: func(ctx context.Context) (any, error) {
				configExec.Add(1)
				return "config-value", nil
			},
		},
		"db": {
			id:        "db",
			dependsOn: []ID{"config"},
			cacheable: true, // Startup node - cacheable
			run: func(ctx context.Context) (any, error) {
				dbExec.Add(1)
				cfg, _ := Dep[string](ctx, "config")
				return "db-with-" + cfg, nil
			},
		},
		"handler": {
			id:        "handler",
			dependsOn: []ID{"db"},
			cacheable: false, // Request-scoped - not cacheable
			run: func(ctx context.Context) (any, error) {
				handlerExec.Add(1)
				db, _ := Dep[string](ctx, "db")
				return "handled-" + db, nil
			},
		},
	}

	cache := NewMemoryCache()
	ctx := context.Background()

	// First execution - all nodes run
	cfg := &config{registry: nodes, cache: cache}
	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if configExec.Load() != 1 || dbExec.Load() != 1 || handlerExec.Load() != 1 {
		t.Fatalf("expected 1 execution each, got config=%d db=%d handler=%d",
			configExec.Load(), dbExec.Load(), handlerExec.Load())
	}

	// Second execution - only handler runs (config and db are cached)
	engine = newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if configExec.Load() != 1 {
		t.Fatalf("config should still be 1 (cached), got %d", configExec.Load())
	}
	if dbExec.Load() != 1 {
		t.Fatalf("db should still be 1 (cached), got %d", dbExec.Load())
	}
	if handlerExec.Load() != 2 {
		t.Fatalf("handler should be 2 (not cacheable), got %d", handlerExec.Load())
	}

	// Verify handler got the cached db value
	if engine.results["handler"] != "handled-db-with-config-value" {
		t.Fatalf("expected 'handled-db-with-config-value', got %v", engine.results["handler"])
	}
}

func TestNoCacheProvided(t *testing.T) {
	var execCount atomic.Int32

	nodes := map[ID]node{
		"counter": {
			id:        "counter",
			dependsOn: nil,
			cacheable: true, // Cacheable but no cache provided
			run: func(ctx context.Context) (any, error) {
				execCount.Add(1)
				return "executed", nil
			},
		},
	}

	ctx := context.Background()

	// No cache configured - cacheable nodes still run every time
	cfg := &config{registry: nodes}
	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", execCount.Load())
	}

	// Second execution without cache - should run again
	engine = newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execCount.Load() != 2 {
		t.Fatalf("expected 2 executions (no cache provided), got %d", execCount.Load())
	}
}
