package graft

// registry holds all registered nodes.
var registry = make(map[string]Node)

// Register adds a node to the global registry.
// Typically called from init() functions in node packages.
// Panics if a node with the same ID is already registered.
func Register(node Node) {
	if _, exists := registry[node.ID]; exists {
		panic("graft: duplicate node registration: " + node.ID)
	}
	registry[node.ID] = node
}

// Registry returns a copy of all registered nodes.
func Registry() map[string]Node {
	cp := make(map[string]Node, len(registry))
	for k, v := range registry {
		cp[k] = v
	}
	return cp
}

