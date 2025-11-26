package graft

import (
	"context"
	"fmt"
	"sync"
)

// Engine manages the dependency graph and execution.
type Engine struct {
	nodes   map[string]Node
	results results
	mu      sync.RWMutex
}

// New creates an engine from a map of nodes.
func New(nodes map[string]Node) *Engine {
	return &Engine{
		nodes:   nodes,
		results: make(results),
	}
}

// Build creates an engine using all nodes from the global registry.
func Build() *Engine {
	return New(Registry())
}

// Run executes all nodes in topological order with automatic parallelization.
// Nodes at the same level (no interdependencies) run concurrently.
// Respects context cancellation.
func (e *Engine) Run(ctx context.Context) error {
	levels, err := e.topoSortLevels()
	if err != nil {
		return err
	}

	for _, level := range levels {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := e.runLevel(ctx, level); err != nil {
			return err
		}
	}

	return nil
}

// runLevel executes all nodes in a level concurrently.
func (e *Engine) runLevel(ctx context.Context, level []string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(level))

	for _, id := range level {
		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()

			node := e.nodes[nodeID]

			// Build context with current results
			e.mu.RLock()
			nodeCtx := withResults(ctx, e.copyResults())
			e.mu.RUnlock()

			// Execute node
			output, err := node.Run(nodeCtx)
			if err != nil {
				errCh <- fmt.Errorf("node %s: %w", nodeID, err)
				return
			}

			e.mu.Lock()
			e.results[nodeID] = output
			e.mu.Unlock()
		}(id)
	}

	wg.Wait()
	close(errCh)

	// Return first error encountered
	if err := <-errCh; err != nil {
		return err
	}

	return nil
}

// copyResults returns a copy of current results (caller must hold at least RLock).
func (e *Engine) copyResults() results {
	cp := make(results, len(e.results))
	for k, v := range e.results {
		cp[k] = v
	}
	return cp
}

// Results returns all collected results after execution.
func (e *Engine) Results() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.copyResults()
}

// topoSortLevels groups nodes into execution levels using Kahn's algorithm.
// Nodes in the same level have no dependencies on each other and can run in parallel.
func (e *Engine) topoSortLevels() ([][]string, error) {
	// Build in-degree map
	inDegree := make(map[string]int)
	for id := range e.nodes {
		inDegree[id] = 0
	}

	// Validate dependencies and compute in-degrees
	for _, node := range e.nodes {
		for _, dep := range node.DependsOn {
			if _, exists := e.nodes[dep]; !exists {
				return nil, fmt.Errorf("node %s depends on unknown node %s", node.ID, dep)
			}
		}
		inDegree[node.ID] = len(node.DependsOn)
	}

	// Build reverse adjacency (who depends on me)
	dependents := make(map[string][]string)
	for _, node := range e.nodes {
		for _, dep := range node.DependsOn {
			dependents[dep] = append(dependents[dep], node.ID)
		}
	}

	// Find nodes with no dependencies (first level)
	var currentLevel []string
	for id, degree := range inDegree {
		if degree == 0 {
			currentLevel = append(currentLevel, id)
		}
	}

	// Process level by level
	var levels [][]string
	processed := 0

	for len(currentLevel) > 0 {
		levels = append(levels, currentLevel)
		processed += len(currentLevel)

		var nextLevel []string
		for _, id := range currentLevel {
			for _, dependent := range dependents[id] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					nextLevel = append(nextLevel, dependent)
				}
			}
		}
		currentLevel = nextLevel
	}

	if processed != len(e.nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return levels, nil
}

