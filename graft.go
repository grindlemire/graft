// Package graft provides a graph-based dependency execution framework.
//
// Graft allows you to define nodes that declare their dependencies explicitly,
// and executes them in topological order with automatic parallelization.
// Nodes at the same level (no interdependencies) run concurrently.
//
// # Quick Start
//
// Define nodes with typed Run functions:
//
//	graft.Register(graft.Node[Config]{
//	    ID:        "config",
//	    DependsOn: []graft.ID{},
//	    Run: func(ctx context.Context) (Config, error) {
//	        return Config{Host: "localhost"}, nil
//	    },
//	})
//
//	graft.Register(graft.Node[*sql.DB]{
//	    ID:        "db",
//	    DependsOn: []graft.ID{"config"},
//	    Run: func(ctx context.Context) (*sql.DB, error) {
//	        cfg, err := graft.Dep[Config](ctx)
//	        if err != nil {
//	            return nil, err
//	        }
//	        return connectDB(cfg), nil
//	    },
//	})
//
// Execute the graph:
//
//	results, err := graft.Execute(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	db := results["db"].(*sql.DB)
//
// # Type Safety
//
// The generic Node[T] type ensures compile-time type checking on Run return values.
// The type parameter T specifies what the node produces.
//
// # Dependency Access
//
// Use [Dep] to retrieve dependency outputs with type safety:
//
//	cfg, err := graft.Dep[config.Output](ctx)
//
// # Subgraph Execution
//
// Use [ExecuteFor] to execute a specific node and its transitive dependencies:
//
//	appOut, results, err := graft.ExecuteFor[app.Output](ctx)
//	// appOut is typed; results map available for accessing other outputs
//
// # Static Analysis
//
// Use [AssertDepsValid] in tests to verify dependency declarations:
//
//	func TestDeps(t *testing.T) {
//	    graft.AssertDepsValid(t, ".")
//	}
package graft

import (
	"context"
	"fmt"
)

// contextKey is the type for context keys used by graft.
// Using an unexported struct type ensures no collisions with other packages.
type contextKey struct{}

// resultsKey is the context key for storing dependency results.
var resultsKey = contextKey{}

type ID string

// Node represents a single node in the dependency graph with a typed output.
//
// The type parameter T specifies the output type of the Run function,
// providing compile-time type safety. Each node has a unique ID, declares
// its dependencies, and provides a Run function that executes its business logic.
//
// Example:
//
//	graft.Node[MyOutput]{
//	    ID:        "mynode",
//	    DependsOn: []graft.ID{config.ID, db.ID},
//	    Run: func(ctx context.Context) (MyOutput, error) {
//	        cfg, _ := graft.Dep[config.Output](ctx)
//	        db, _ := graft.Dep[db.Output](ctx)
//	        return doWork(cfg, db.Pool), nil
//	    },
//	}
type Node[T any] struct {
	// ID is the unique identifier for this node.
	// This is used to reference the node in DependsOn lists and Dep calls.
	ID ID

	// DependsOn lists the IDs of nodes that must complete before this node runs.
	// The engine ensures all dependencies have completed and their outputs
	// are available via Dep before calling Run.
	DependsOn []ID

	// Run executes the node's business logic and returns a typed output.
	// Dependencies are accessed via Dep[T](ctx).
	Run func(ctx context.Context) (T, error)

	// Cacheable indicates whether this node's output should be cached.
	// When true and a cache is provided via WithCache, the node's output
	// is stored after first execution and reused on subsequent runs.
	// Default is false (not cached).
	Cacheable bool
}

// node is the internal type-erased representation used for storage.
// Type erasure happens at registration time, allowing heterogeneous storage.
type node struct {
	id        ID
	dependsOn []ID
	run       func(ctx context.Context) (any, error)
	cacheable bool
}

// results is the internal type for storing node outputs in context.
type results map[ID]any

// withResults adds results to a context for downstream node access.
func withResults(ctx context.Context, r results) context.Context {
	return context.WithValue(ctx, resultsKey, r)
}

// getResults retrieves results from context.
func getResults(ctx context.Context) (results, bool) {
	r, ok := ctx.Value(resultsKey).(results)
	return r, ok
}

// Dep retrieves a dependency's output from the context with type assertion.
//
// This is the primary way for nodes to access their dependencies' outputs.
// The type parameter T specifies the expected output type, and the node ID
// is derived from T using the type-to-ID mapping established at registration.
//
// Returns an error if:
//   - The type T is not registered as a node output
//   - The context has no results (called outside of a node's Run function)
//   - The dependency is not found (not declared in DependsOn)
//   - The dependency's output cannot be asserted to type T
//
// Example:
//
//	func(ctx context.Context) (MyOutput, error) {
//	    cfg, err := graft.Dep[config.Output](ctx)
//	    if err != nil {
//	        return MyOutput{}, err
//	    }
//	    // use cfg...
//	}
func Dep[T any](ctx context.Context) (T, error) {
	var zero T

	id, ok := typeToID[(*T)(nil)]
	if !ok {
		return zero, fmt.Errorf("graft: type %T not registered as node output", zero)
	}

	r, ok := getResults(ctx)
	if !ok {
		return zero, fmt.Errorf("graft: no results in context")
	}

	val, ok := r[id]
	if !ok {
		return zero, fmt.Errorf("graft: dependency %q not found", id)
	}

	typed, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("graft: dependency %q has wrong type (got %T, want %T)", id, val, zero)
	}

	return typed, nil
}

// Result retrieves a node's output from a results map with type assertion.
//
// This is used after Execute/ExecuteFor to access node outputs from the
// returned results map. The node ID is derived from T using the type-to-ID
// mapping established at registration.
//
// Returns an error if:
//   - The type T is not registered as a node output
//   - The node is not found in the results
//   - The output cannot be asserted to type T
//
// Example:
//
//	results, _ := graft.Execute(ctx)
//	cfg, err := graft.Result[config.Output](results)
func Result[T any](r results) (T, error) {
	var zero T

	id, ok := typeToID[(*T)(nil)]
	if !ok {
		return zero, fmt.Errorf("graft: type %T not registered as node output", zero)
	}

	val, ok := r[id]
	if !ok {
		return zero, fmt.Errorf("graft: result %q not found", id)
	}

	typed, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("graft: result %q has wrong type (got %T, want %T)", id, val, zero)
	}

	return typed, nil
}

// depByID is an internal helper for testing that retrieves a dependency by explicit ID.
// This is needed for tests that use makeNode() to create nodes without going through Register.
func depByID[T any](ctx context.Context, nodeID ID) (T, error) {
	var zero T

	r, ok := getResults(ctx)
	if !ok {
		return zero, fmt.Errorf("graft: no results in context")
	}

	val, ok := r[nodeID]
	if !ok {
		return zero, fmt.Errorf("graft: dependency %q not found", nodeID)
	}

	typed, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("graft: dependency %q has wrong type (got %T, want %T)", nodeID, val, zero)
	}

	return typed, nil
}

// resultByID is an internal helper for testing that retrieves a result by explicit ID.
func resultByID[T any](r results, nodeID ID) (T, error) {
	var zero T

	val, ok := r[nodeID]
	if !ok {
		return zero, fmt.Errorf("graft: result %q not found", nodeID)
	}

	typed, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("graft: result %q has wrong type (got %T, want %T)", nodeID, val, zero)
	}

	return typed, nil
}
