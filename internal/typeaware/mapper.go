package typeaware

import (
	"fmt"
	"go/types"
)

// typeIDMapper builds and maintains a bidirectional mapping between
// types.Type and node IDs
type typeIDMapper struct {
	typeToID map[string]string     // Canonical type string → node ID
	idToType map[string]types.Type // Node ID → type
}

// newTypeIDMapper creates a new type-to-ID mapper
func newTypeIDMapper() *typeIDMapper {
	return &typeIDMapper{
		typeToID: make(map[string]string),
		idToType: make(map[string]types.Type),
	}
}

// normalizeType converts a type to its canonical string representation
// This handles type aliases, named types, pointers, etc.
func (m *typeIDMapper) normalizeType(t types.Type) string {
	// Use types.TypeString with full package paths for canonical representation
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Path()
	})
}

// typeKey returns a unique key for a type
func (m *typeIDMapper) typeKey(t types.Type) string {
	return m.normalizeType(t)
}

// BuildMapping constructs the type-to-ID mapping from node definitions
func (m *typeIDMapper) BuildMapping(nodes []NodeDefinition) error {
	for _, node := range nodes {
		if node.OutputType == nil {
			return fmt.Errorf("node %q has nil output type", node.ID)
		}

		key := m.typeKey(node.OutputType)

		// Check for conflicts: same type registered by multiple nodes
		if existingID, exists := m.typeToID[key]; exists {
			if existingID != node.ID {
				return fmt.Errorf(
					"type conflict: type %s is registered by both node %q and node %q",
					key, existingID, node.ID,
				)
			}
			// Same node ID - OK, might be duplicate discovery
			continue
		}

		// Add mapping
		m.typeToID[key] = node.ID
		m.idToType[node.ID] = node.OutputType
	}

	return nil
}

// ResolveType resolves a type to its node ID
func (m *typeIDMapper) ResolveType(t types.Type) (string, error) {
	key := m.typeKey(t)

	id, ok := m.typeToID[key]
	if !ok {
		return "", fmt.Errorf("type %s is not registered", key)
	}

	return id, nil
}

// GetType returns the output type for a given node ID
func (m *typeIDMapper) GetType(nodeID string) (types.Type, error) {
	t, ok := m.idToType[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %q not found in type mapping", nodeID)
	}

	return t, nil
}

// HasType checks if a type is registered
func (m *typeIDMapper) HasType(t types.Type) bool {
	key := m.typeKey(t)
	_, ok := m.typeToID[key]
	return ok
}

// Size returns the number of types in the mapping
func (m *typeIDMapper) Size() int {
	return len(m.typeToID)
}
