// Package graft provides a graph-based dependency execution framework.
// Nodes declare their dependencies explicitly, and the engine executes them
// in topological order with automatic parallelization.
package graft

import (
	"context"
	"fmt"
)

// contextKey is the type for context keys used by graft.
type contextKey struct{}

// resultsKey is the context key for storing dependency results.
var resultsKey = contextKey{}

// RunFunc is the signature for a node's execution function.
// Dependencies are accessed via the Dep[T] helper using the context.
type RunFunc func(ctx context.Context) (any, error)

// Node represents a single node in the dependency graph.
type Node struct {
	// ID is the unique identifier for this node.
	ID string
	// DependsOn lists the IDs of nodes that must complete before this node runs.
	DependsOn []string
	// Run executes the node's business logic.
	Run RunFunc
}

// results is the internal type for storing node outputs in context.
type results map[string]any

// withResults adds results to a context.
func withResults(ctx context.Context, r results) context.Context {
	return context.WithValue(ctx, resultsKey, r)
}

// getResults retrieves results from context.
func getResults(ctx context.Context) (results, bool) {
	r, ok := ctx.Value(resultsKey).(results)
	return r, ok
}

// Dep retrieves a dependency's output from the context with type assertion.
// Returns an error if the dependency is not found or has the wrong type.
func Dep[T any](ctx context.Context, nodeID string) (T, error) {
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

