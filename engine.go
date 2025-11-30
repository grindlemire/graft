package graft

import (
	"context"
	"fmt"
	"sync"
)

// Option configures execution behavior.
type Option func(*config)

type config struct {
	registry       map[ID]node
	cache          Cache       // optional cache for node outputs
	ignoreCacheFor map[ID]bool // nodes to skip cache lookup
}

// WithRegistry uses a custom node registry instead of the global registry.
//
// Example:
//
//	results, err := graft.Execute(ctx, graft.WithRegistry(customNodes))
func WithRegistry(registry map[ID]node) Option {
	return func(c *config) {
		c.registry = registry
	}
}

// MergeRegistry merges the provided registry with the global registry.
// On conflicts, the provided registry takes precedence.
//
// This is useful for testing where you want to override specific nodes
// while keeping the rest of the registered graph.
//
// Example:
//
//	// Override just the "db" node for testing
//	results, err := graft.Execute(ctx, graft.MergeRegistry(mockNodes))
func MergeRegistry(registry map[ID]node) Option {
	return func(c *config) {
		merged := Registry()
		for id, n := range registry {
			merged[id] = n
		}
		c.registry = merged
	}
}

// WithCache overrides the default global cache with a custom cache.
//
// By default, Execute/ExecuteFor use a global in-memory cache (similar to
// the global registry), so caching works automatically across calls.
// Use WithCache to provide a custom cache implementation (e.g., Redis)
// or an isolated cache for testing.
//
// Example:
//
//	// Use a custom cache instead of the global default
//	customCache := graft.NewMemoryCache()
//	results, _ := graft.Execute(ctx, graft.WithCache(customCache))
func WithCache(cache Cache) Option {
	return func(c *config) {
		c.cache = cache
	}
}

// IgnoreCache forces re-execution of the specified cacheable nodes,
// bypassing any cached values. The fresh results are still written
// back to the cache.
//
// This is useful for invalidating specific nodes without clearing the
// entire cache (e.g., to refresh config after a change).
//
// Example:
//
//	// Re-fetch config even though it's cacheable
//	results, err := graft.Execute(ctx,
//	    graft.WithCache(cache),
//	    graft.IgnoreCache("config"),
//	)
func IgnoreCache(ids ...ID) Option {
	return func(cfg *config) {
		if cfg.ignoreCacheFor == nil {
			cfg.ignoreCacheFor = make(map[ID]bool)
		}
		for _, id := range ids {
			cfg.ignoreCacheFor[id] = true
		}
	}
}

// DisableCache disables the use of the default global cache.
//
// Example:
//
//	// Disable the use of the default global cache
//	results, err := graft.Execute(ctx, graft.DisableCache())
func DisableCache() Option {
	return func(cfg *config) {
		cfg.cache = nil
	}
}

// PatchValue replaces a node's output with a fixed value for testing.
//
// The node is identified by the type T, which must match a registered node's
// output type. The patched node has no dependencies and simply returns the
// provided value.
//
// This is a no-op if type T is not registered.
//
// Example:
//
//	// Replace config node with a test value
//	results, err := graft.Execute(ctx,
//	    graft.PatchValue[config.Output](config.Output{Host: "test", Port: 9999}),
//	)
func PatchValue[T any](value T) Option {
	return func(c *config) {
		id, ok := typeToID[(*T)(nil)]
		if !ok {
			return
		}
		if c.registry == nil {
			c.registry = Registry()
		}
		c.registry[id] = node{
			id:        id,
			dependsOn: []ID{},
			run:       func(ctx context.Context) (any, error) { return value, nil },
		}
	}
}

// Patch replaces a node with a custom node for testing.
//
// The node is identified by the type T, which must match a registered node's
// output type. The patched node inherits DependsOn, Run, and Cacheable from
// the provided Node[T].
//
// This is a no-op if type T is not registered.
//
// Example:
//
//	// Replace db node with a mock that uses config
//	results, err := graft.Execute(ctx,
//	    graft.Patch[db.Output](graft.Node[db.Output]{
//	        DependsOn: []graft.ID{"config"},
//	        Run: func(ctx context.Context) (db.Output, error) {
//	            cfg, _ := graft.Dep[config.Output](ctx)
//	            return db.Output{Pool: mockPool(cfg)}, nil
//	        },
//	    }),
//	)
func Patch[T any](n Node[T]) Option {
	return func(c *config) {
		id, ok := typeToID[(*T)(nil)]
		if !ok {
			return
		}
		if c.registry == nil {
			c.registry = Registry()
		}
		c.registry[id] = node{
			id:        id,
			dependsOn: n.DependsOn,
			run:       func(ctx context.Context) (any, error) { return n.Run(ctx) },
			cacheable: n.Cacheable,
		}
	}
}

// Execute runs all registered nodes and returns their results.
//
// Nodes are executed in topological order with automatic parallelization.
// Nodes at the same dependency level run concurrently.
//
// By default, uses the global registry. Use [WithRegistry] for a custom registry.
//
// Example:
//
//	import (
//	    _ "myapp/nodes/config"
//	    _ "myapp/nodes/db"
//	)
//
//	func main() {
//	    results, err := graft.Execute(ctx)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    db := results["db"].(*sql.DB)
//	}
func Execute(ctx context.Context, opts ...Option) (map[ID]any, error) {
	cfg := &config{registry: Registry(), cache: defaultCache}
	for _, opt := range opts {
		opt(cfg)
	}

	engine := newEngine(cfg.registry, cfg)
	if err := engine.run(ctx); err != nil {
		return nil, err
	}
	return engine.results, nil
}

// ExecuteFor runs the node that produces type T and its transitive dependencies.
//
// The target node is determined by the type parameter T, which must match
// the output type used when registering the node. Returns the typed result
// and the full results map for accessing other node outputs if needed.
//
// By default, uses the global registry. Use [WithRegistry] for a custom registry.
//
// Returns an error if the type is not registered or execution fails.
//
// Example:
//
//	appOut, results, err := graft.ExecuteFor[app.Output](ctx)
//	// appOut is typed as app.Output
//	// results map available for accessing dependencies:
//	config, _ := graft.Result[config.Output](results)
func ExecuteFor[T any](ctx context.Context, opts ...Option) (T, results, error) {
	var zero T

	id, ok := typeToID[(*T)(nil)]
	if !ok {
		return zero, nil, fmt.Errorf("graft: type %T not registered as node output", zero)
	}

	results, err := executeForIDs(ctx, []ID{id}, opts...)
	if err != nil {
		return zero, nil, err
	}

	result, err := Result[T](results)
	if err != nil {
		return zero, nil, err
	}

	return result, results, nil
}

// executeForIDs runs the specified target nodes and their transitive dependencies.
// This is an internal helper used by ExecuteFor.
func executeForIDs(ctx context.Context, targets []ID, opts ...Option) (map[ID]any, error) {
	cfg := &config{registry: Registry(), cache: defaultCache}
	for _, opt := range opts {
		opt(cfg)
	}

	nodes, err := resolveSubgraph(cfg.registry, targets)
	if err != nil {
		return nil, err
	}

	engine := newEngine(nodes, cfg)
	if err := engine.run(ctx); err != nil {
		return nil, err
	}
	return engine.results, nil
}

// resolveSubgraph extracts target nodes and their transitive dependencies from a registry.
func resolveSubgraph(registry map[ID]node, targets []ID) (map[ID]node, error) {
	needed := make(map[ID]node)

	var resolve func(id ID) error
	resolve = func(id ID) error {
		if _, already := needed[id]; already {
			return nil
		}

		n, ok := registry[id]
		if !ok {
			return fmt.Errorf("unknown node: %s", id)
		}

		needed[id] = n

		for _, dep := range n.dependsOn {
			if err := resolve(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, id := range targets {
		if err := resolve(id); err != nil {
			return nil, err
		}
	}

	return needed, nil
}

// engine manages the dependency graph and orchestrates execution.
type engine struct {
	nodes          map[ID]node
	results        results
	mu             sync.RWMutex
	cache          Cache
	ignoreCacheFor map[ID]bool
}

func newEngine(nodes map[ID]node, cfg *config) *engine {
	return &engine{
		nodes:          nodes,
		results:        make(results),
		cache:          cfg.cache,
		ignoreCacheFor: cfg.ignoreCacheFor,
	}
}

func (e *engine) run(ctx context.Context) error {
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

func (e *engine) runLevel(ctx context.Context, level []ID) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(level))

	for _, id := range level {
		wg.Add(1)
		go func(nodeID ID) {
			defer wg.Done()

			n := e.nodes[nodeID]

			// Check cache for cacheable nodes (unless explicitly ignored)
			useCache := e.cache != nil && n.cacheable && !e.ignoreCacheFor[nodeID]
			if useCache {
				if val, found, err := e.cache.Get(ctx, nodeID); err != nil {
					errCh <- fmt.Errorf("node %s: cache get: %w", nodeID, err)
					return
				} else if found {
					e.mu.Lock()
					e.results[nodeID] = val
					e.mu.Unlock()
					return // Cache hit - skip execution
				}
			}

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

			// Write to cache for cacheable nodes
			if useCache {
				if err := e.cache.Set(ctx, nodeID, output); err != nil {
					errCh <- fmt.Errorf("node %s: cache set: %w", nodeID, err)
					return
				}
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

func (e *engine) copyResults() results {
	cp := make(results, len(e.results))
	for k, v := range e.results {
		cp[k] = v
	}
	return cp
}

// topoSortLevels groups nodes into execution levels using Kahn's algorithm.
func (e *engine) topoSortLevels() ([][]ID, error) {
	inDegree := make(map[ID]int)
	for id := range e.nodes {
		inDegree[id] = 0
	}

	for _, n := range e.nodes {
		for _, dep := range n.dependsOn {
			if _, exists := e.nodes[dep]; !exists {
				return nil, fmt.Errorf("node %s depends on unknown node %s", n.id, dep)
			}
		}
		inDegree[n.id] = len(n.dependsOn)
	}

	dependents := make(map[ID][]ID)
	for _, n := range e.nodes {
		for _, dep := range n.dependsOn {
			dependents[dep] = append(dependents[dep], n.id)
		}
	}

	var currentLevel []ID
	for id, degree := range inDegree {
		if degree == 0 {
			currentLevel = append(currentLevel, id)
		}
	}

	var levels [][]ID
	processed := 0

	for len(currentLevel) > 0 {
		levels = append(levels, currentLevel)
		processed += len(currentLevel)

		var nextLevel []ID
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
