package graft

import "context"

// registry holds all registered nodes in type-erased form.
// It is populated at init time by calls to Register.
var registry = make(map[ID]node)

// typeToID maps output types to their node IDs.
// This enables type-based ExecuteFor without reflection.
var typeToID = make(map[any]ID)

// Register adds a typed node to the global registry.
//
// The type parameter is erased internally for heterogeneous storage.
// This is typically called from init() functions in node packages, ensuring
// all nodes are registered before main() runs. This pattern allows nodes
// to be self-registering via blank imports.
//
// Panics if a node with the same ID is already registered. This catches
// accidental ID collisions at startup.
//
// Example:
//
//	// nodes/config/config.go
//	package config
//
//	type Output struct {
//	    Host string
//	    Port int
//	}
//
//	func init() {
//	    graft.Register(graft.Node[Output]{
//	        ID:        "config",
//	        DependsOn: []graft.ID{},
//	        Run:       loadConfig,
//	    })
//	}
//
//	func loadConfig(ctx context.Context) (Output, error) {
//	    return Output{Host: "localhost", Port: 5432}, nil
//	}
//
// Then import the package for its side effects:
//
//	import _ "myapp/nodes/config"
func Register[T any](n Node[T]) {
	if _, exists := registry[n.ID]; exists {
		panic("graft: duplicate node registration: " + string(n.ID))
	}

	// Type erasure: convert typed Node[T] to internal node with any
	registry[n.ID] = node{
		id:        n.ID,
		dependsOn: n.DependsOn,
		run: func(ctx context.Context) (any, error) {
			return n.Run(ctx)
		},
		cacheable: n.Cacheable,
	}

	// Record type → ID mapping using nil pointer sentinel
	typeToID[(*T)(nil)] = n.ID
}

// Registry returns a copy of all registered nodes.
//
// The returned map is a copy; modifications do not affect the global registry.
// This is commonly passed to [WithRegistry] for custom execution scenarios.
//
// Example:
//
//	nodes := graft.Registry()
//	fmt.Printf("Registered %d nodes\n", len(nodes))
func Registry() map[ID]node {
	cp := make(map[ID]node, len(registry))
	for k, v := range registry {
		cp[k] = v
	}
	return cp
}

// ResetRegistry clears the global registry.
// This is primarily useful for test isolation.
func ResetRegistry() {
	for k := range registry {
		delete(registry, k)
	}
	for k := range typeToID {
		delete(typeToID, k)
	}
}
