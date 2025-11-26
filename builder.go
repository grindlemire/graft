package graft

import "fmt"

// Builder constructs engines from a node catalog with automatic dependency resolution.
//
// Use Builder when you need to execute only a subset of nodes. Given target nodes,
// it automatically includes all transitive dependencies.
//
// Example:
//
//	builder := graft.NewBuilder(graft.Registry())
//
//	// Build an engine with only "api" and its dependencies
//	engine, err := builder.BuildFor("api")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Only executes nodes needed by "api"
//	engine.Run(ctx)
type Builder struct {
	catalog map[string]node
}

// NewBuilder creates a builder from a node catalog.
//
// The catalog is typically [Registry], but you can provide any map of nodes.
// The builder does not modify the catalog.
//
// Example:
//
//	// Use global registry
//	builder := graft.NewBuilder(graft.Registry())
func NewBuilder(catalog map[string]node) *Builder {
	return &Builder{catalog: catalog}
}

// BuildFor creates an engine with the specified target nodes and all their
// transitive dependencies.
//
// Just specify the terminal nodes you need — dependencies are resolved
// automatically. Duplicate nodes in the dependency tree are handled correctly.
//
// Returns an error if any target or dependency node is not found in the catalog.
//
// Example:
//
//	// Single target
//	engine, err := builder.BuildFor("api")
//
//	// Multiple targets
//	engine, err := builder.BuildFor("api", "worker", "scheduler")
func (b *Builder) BuildFor(targetNodeIDs ...string) (*Engine, error) {
	needed := make(map[string]node)

	// Recursive resolver with memoization
	var resolve func(id string) error
	resolve = func(id string) error {
		// Already resolved
		if _, already := needed[id]; already {
			return nil
		}

		// Find node in catalog
		n, ok := b.catalog[id]
		if !ok {
			return fmt.Errorf("unknown node: %s", id)
		}

		// Add to needed set
		needed[id] = n

		// Recursively resolve dependencies
		for _, dep := range n.dependsOn {
			if err := resolve(dep); err != nil {
				return err
			}
		}
		return nil
	}

	// Resolve all targets
	for _, id := range targetNodeIDs {
		if err := resolve(id); err != nil {
			return nil, err
		}
	}

	return New(needed), nil
}
