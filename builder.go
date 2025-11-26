package graft

import "fmt"

// Builder constructs engines from a node catalog with automatic dependency resolution.
type Builder struct {
	catalog map[string]Node
}

// NewBuilder creates a builder from a node catalog.
func NewBuilder(catalog map[string]Node) *Builder {
	return &Builder{catalog: catalog}
}

// BuildFor creates an engine with the specified target nodes and all their transitive dependencies.
// Just specify the terminal nodes you need — dependencies are resolved automatically.
func (b *Builder) BuildFor(targetNodeIDs ...string) (*Engine, error) {
	needed := make(map[string]Node)

	var resolve func(id string) error
	resolve = func(id string) error {
		if _, already := needed[id]; already {
			return nil
		}
		node, ok := b.catalog[id]
		if !ok {
			return fmt.Errorf("unknown node: %s", id)
		}
		needed[id] = node
		for _, dep := range node.DependsOn {
			if err := resolve(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, id := range targetNodeIDs {
		if err := resolve(id); err != nil {
			return nil, err
		}
	}

	return New(needed), nil
}

