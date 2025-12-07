package graft

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// resetGlobalState clears both the global registry and cache for test isolation.
func resetGlobalState() {
	ResetRegistry()
	ResetDefaultCache()
}

func TestRegister(t *testing.T) {
	type tc struct {
		registerNodes func()
		wantCount     int
		wantIDs       []string
	}

	tests := map[string]tc{
		"register single node": {
			registerNodes: func() {
				Register(Node[string]{
					ID:  "test1",
					Run: func(ctx context.Context) (string, error) { return "result", nil },
				})
			},
			wantCount: 1,
			wantIDs:   []string{"test1"},
		},
		"register multiple nodes with different types": {
			registerNodes: func() {
				Register(Node[string]{
					ID:  "test1",
					Run: func(ctx context.Context) (string, error) { return "str", nil },
				})
				Register(Node[int]{
					ID:  "test2",
					Run: func(ctx context.Context) (int, error) { return 42, nil },
				})
				Register(Node[bool]{
					ID:  "test3",
					Run: func(ctx context.Context) (bool, error) { return true, nil },
				})
			},
			wantCount: 3,
			wantIDs:   []string{"test1", "test2", "test3"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resetGlobalState()

			tt.registerNodes()

			if len(registry) != tt.wantCount {
				t.Errorf("got %d registered nodes, want %d", len(registry), tt.wantCount)
			}

			// Verify each node is present
			for _, id := range tt.wantIDs {
				if _, exists := registry[ID(id)]; !exists {
					t.Errorf("node %q not found in registry", id)
				}
			}
		})
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	resetGlobalState()

	Register(Node[string]{
		ID:  "duplicate",
		Run: func(ctx context.Context) (string, error) { return "", nil },
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, got none")
		} else {
			msg, ok := r.(string)
			if !ok {
				t.Errorf("panic value is not a string: %T", r)
				return
			}
			if !strings.Contains(msg, "duplicate node registration") {
				t.Errorf("panic message %q should contain 'duplicate node registration'", msg)
			}
		}
	}()

	// This should panic
	Register(Node[int]{
		ID:  "duplicate",
		Run: func(ctx context.Context) (int, error) { return 0, nil },
	})
}

func TestRegistry(t *testing.T) {
	type tc struct {
		setup     func()
		wantCount int
	}

	tests := map[string]tc{
		"empty registry": {
			setup:     func() { resetGlobalState() },
			wantCount: 0,
		},
		"registry with nodes": {
			setup: func() {
				resetGlobalState()
				Register(Node[string]{ID: "a", Run: func(ctx context.Context) (string, error) { return "", nil }})
				Register(Node[int]{ID: "b", Run: func(ctx context.Context) (int, error) { return 0, nil }})
			},
			wantCount: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.setup()

			got := Registry()

			if len(got) != tt.wantCount {
				t.Errorf("got %d nodes, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestRegistryReturnsCopy(t *testing.T) {
	resetGlobalState()
	Register(Node[string]{ID: "original", Run: func(ctx context.Context) (string, error) { return "", nil }})

	copy1 := Registry()
	copy2 := Registry()

	// Modify copy1
	copy1[ID("modified")] = node{id: "modified"}

	// copy2 should be unaffected
	if _, exists := copy2[ID("modified")]; exists {
		t.Error("Registry() did not return a copy; modification affected other copy")
	}

	// Original registry should be unaffected
	if _, exists := registry[ID("modified")]; exists {
		t.Error("Registry() did not return a copy; modification affected original registry")
	}
}

func TestExecuteFromRegistry(t *testing.T) {
	type tc struct {
		setup       func()
		wantCount   int
		wantResults map[ID]any
	}

	tests := map[string]tc{
		"execute from empty registry": {
			setup:       func() { resetGlobalState() },
			wantCount:   0,
			wantResults: map[ID]any{},
		},
		"execute from populated registry": {
			setup: func() {
				resetGlobalState()
				Register(Node[int]{ID: "a", Run: func(ctx context.Context) (int, error) { return 1, nil }})
				Register(Node[int]{ID: "b", Run: func(ctx context.Context) (int, error) { return 2, nil }})
			},
			wantCount:   2,
			wantResults: map[ID]any{"a": 1, "b": 2},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.setup()

			results, err := Execute(context.Background())
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
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

func TestRegistryConcurrentAccess(t *testing.T) {
	resetGlobalState()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent reads via Registry()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = Registry()
			}
		}()
	}

	wg.Wait()
	// If we get here without a race condition, the test passes
}

func TestTypedNodeExecution(t *testing.T) {
	// Test that typed nodes execute correctly and return proper types
	resetGlobalState()

	type ConfigOutput struct {
		Host string
		Port int
	}

	Register(Node[ConfigOutput]{
		ID:        "config",
		DependsOn: []ID{},
		Run: func(ctx context.Context) (ConfigOutput, error) {
			return ConfigOutput{Host: "localhost", Port: 5432}, nil
		},
	})

	Register(Node[string]{
		ID:        "db",
		DependsOn: []ID{"config"},
		Run: func(ctx context.Context) (string, error) {
			cfg, err := Dep[ConfigOutput](ctx)
			if err != nil {
				return "", err
			}
			return cfg.Host + ":5432", nil
		},
	})

	results, err := Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Verify config output
	configResult, ok := results["config"]
	if !ok {
		t.Fatal("missing config result")
	}
	cfg, ok := configResult.(ConfigOutput)
	if !ok {
		t.Fatalf("config result has wrong type: %T", configResult)
	}
	if cfg.Host != "localhost" || cfg.Port != 5432 {
		t.Errorf("config result = %+v, want {localhost 5432}", cfg)
	}

	// Verify db output
	dbResult, ok := results["db"]
	if !ok {
		t.Fatal("missing db result")
	}
	if dbResult != "localhost:5432" {
		t.Errorf("db result = %v, want localhost:5432", dbResult)
	}
}

func TestResetRegistry(t *testing.T) {
	// Start with clean state
	ResetRegistry()

	// Register a node
	Register(Node[string]{
		ID:  "test-node",
		Run: func(ctx context.Context) (string, error) { return "value", nil },
	})

	// Verify it's there
	if _, exists := registry["test-node"]; !exists {
		t.Fatal("expected node in registry before reset")
	}

	// Reset
	ResetRegistry()

	// Verify it's gone
	if _, exists := registry["test-node"]; exists {
		t.Fatal("expected empty registry after reset")
	}
	if len(registry) != 0 {
		t.Errorf("expected 0 nodes after reset, got %d", len(registry))
	}
}

func TestTestIsolation(t *testing.T) {
	// This test verifies that resetGlobalState clears both registry and cache
	ctx := context.Background()

	// Set up some state
	Register(Node[string]{
		ID:        "isolation-test",
		Cacheable: true,
		Run:       func(ctx context.Context) (string, error) { return "cached-value", nil },
	})
	defaultCache.Set(ctx, "isolation-test", "cached-value")

	// Verify state exists
	if _, exists := registry["isolation-test"]; !exists {
		t.Fatal("expected node in registry")
	}
	if val, found, _ := defaultCache.Get(ctx, "isolation-test"); !found || val != "cached-value" {
		t.Fatal("expected value in cache")
	}

	// Reset
	resetGlobalState()

	// Verify both are cleared
	if _, exists := registry["isolation-test"]; exists {
		t.Fatal("expected empty registry after resetGlobalState")
	}
	if _, found, _ := defaultCache.Get(ctx, "isolation-test"); found {
		t.Fatal("expected empty cache after resetGlobalState")
	}
}
