package graft

import (
	"context"
	"fmt"
	"sync"
)

// Engine manages the dependency graph and orchestrates execution.
//
// The engine holds a set of nodes and executes them in topological order,
// running nodes at the same level concurrently. Use [New] to create an
// engine from a custom node map, or [Build] to create one from the
// global registry.
//
// The engine is safe for concurrent use after creation, but Run should
// only be called once per engine instance.
type Engine struct {
	nodes   map[string]node
	results results
	mu      sync.RWMutex
}

// New creates an engine from a map of nodes.
//
// The nodes map keys should match the node id values. This function
// does not validate the dependency graph; validation occurs during Run.
//
// For most use cases, prefer [Build] which uses the global registry,
// or [Builder.BuildFor] for subgraph execution.
func New(nodes map[string]node) *Engine {
	return &Engine{
		nodes:   nodes,
		results: make(results),
	}
}

// Build creates an engine using all nodes from the global registry.
//
// This is the most common way to create an engine. It uses all nodes
// that have been registered via [Register], typically from init() functions.
//
// Example:
//
//	import (
//	    _ "myapp/nodes/config"
//	    _ "myapp/nodes/db"
//	)
//
//	func main() {
//	    engine := graft.Build()
//	    engine.Run(context.Background())
//	}
func Build() *Engine {
	return New(Registry())
}

// Run executes all nodes in topological order with automatic parallelization.
//
// Nodes are grouped into levels based on their dependencies. Within each level,
// all nodes run concurrently. The engine waits for all nodes in a level to
// complete before starting the next level.
//
// Run respects context cancellation. If the context is cancelled between levels,
// execution stops and the context error is returned.
//
// Returns an error if:
//   - A cycle is detected in the dependency graph
//   - A node depends on an unknown node
//   - Any node's Run function returns an error
//   - The context is cancelled
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := engine.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
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

			n := e.nodes[nodeID]

			// Build context with current results snapshot
			e.mu.RLock()
			nodeCtx := withResults(ctx, e.copyResults())
			e.mu.RUnlock()

			// Execute node
			output, err := n.run(nodeCtx)
			if err != nil {
				errCh <- fmt.Errorf("node %s: %w", nodeID, err)
				return
			}

			// Store result
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

// copyResults returns a copy of current results.
// Caller must hold at least RLock.
func (e *Engine) copyResults() results {
	cp := make(results, len(e.results))
	for k, v := range e.results {
		cp[k] = v
	}
	return cp
}

// Results returns all collected results after execution.
//
// The returned map is a copy; modifications do not affect the engine's
// internal state. Keys are node IDs and values are the outputs returned
// by each node's Run function.
//
// Call this after Run completes to access node outputs.
//
// Example:
//
//	engine.Run(ctx)
//	results := engine.Results()
//	dbPool := results["db"].(*sql.DB)
func (e *Engine) Results() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.copyResults()
}

// topoSortLevels groups nodes into execution levels using Kahn's algorithm.
//
// Nodes in the same level have no dependencies on each other and can run
// in parallel. Each level contains only nodes whose dependencies are all
// in previous levels.
//
// Returns an error if a cycle is detected or if a node depends on an
// unknown node.
func (e *Engine) topoSortLevels() ([][]string, error) {
	// Build in-degree map (count of dependencies for each node)
	inDegree := make(map[string]int)
	for id := range e.nodes {
		inDegree[id] = 0
	}

	// Validate dependencies exist and compute in-degrees
	for _, n := range e.nodes {
		for _, dep := range n.dependsOn {
			if _, exists := e.nodes[dep]; !exists {
				return nil, fmt.Errorf("node %s depends on unknown node %s", n.id, dep)
			}
		}
		inDegree[n.id] = len(n.dependsOn)
	}

	// Build reverse adjacency list (who depends on me)
	dependents := make(map[string][]string)
	for _, n := range e.nodes {
		for _, dep := range n.dependsOn {
			dependents[dep] = append(dependents[dep], n.id)
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
