package graft

import (
	"fmt"
	"io"
	"sort"
)

// PrintGraph outputs an ASCII representation of the dependency graph to the provided io.Writer.
func PrintGraph(w io.Writer, opts ...Option) error {
	cfg := &config{registry: Registry()}
	for _, opt := range opts {
		opt(cfg)
	}

	if len(cfg.registry) == 0 {
		fmt.Fprintln(w, "No nodes registered")
		return nil
	}

	levels, err := topoSortLevels(cfg.registry)
	if err != nil {
		return err
	}

	renderer := newGraphRenderer(cfg.registry, levels)
	output := renderer.render()
	fmt.Fprint(w, output)

	return nil
}

// PrintMermaid outputs a Mermaid diagram of the dependency graph to the provided io.Writer.
func PrintMermaid(w io.Writer, opts ...Option) error {
	cfg := &config{registry: Registry()}
	for _, opt := range opts {
		opt(cfg)
	}

	fmt.Fprintln(w, "graph TD")

	if len(cfg.registry) == 0 {
		return nil
	}

	for id, n := range cfg.registry {
		for _, dep := range n.dependsOn {
			fmt.Fprintf(w, "    %s --> %s\n", dep, id)
		}
	}

	for id, n := range cfg.registry {
		if n.cacheable {
			fmt.Fprintf(w, "    style %s fill:#e1f5fe\n", id)
		}
	}

	return nil
}

// topoSortLevels computes topological levels using Kahn's algorithm.
// Nodes are grouped into levels where all nodes in a level can execute concurrently.
// Levels are sorted for deterministic output.
func topoSortLevels(nodes map[ID]node) ([][]ID, error) {
	inDegree := make(map[ID]int)
	for id := range nodes {
		inDegree[id] = 0
	}

	for _, n := range nodes {
		for _, dep := range n.dependsOn {
			if _, exists := nodes[dep]; !exists {
				return nil, fmt.Errorf("node %s depends on unknown node %s", n.id, dep)
			}
		}
		inDegree[n.id] = len(n.dependsOn)
	}

	dependents := make(map[ID][]ID)
	for _, n := range nodes {
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
		// Sort for deterministic output
		sort.Slice(currentLevel, func(i, j int) bool {
			return currentLevel[i] < currentLevel[j]
		})
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

	if processed != len(nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return levels, nil
}
