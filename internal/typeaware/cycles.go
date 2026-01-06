package typeaware

// cycleDetector discovers circular dependencies using DFS
type cycleDetector struct {
	adjList map[string][]string // nodeID → dependencies
	state   map[string]int      // DFS visit state: 0=unvisited, 1=visiting, 2=visited
	path    []string            // Current DFS path
	cycles  [][]string          // All discovered cycles
}

// newCycleDetector creates a cycle detector from analysis results
func newCycleDetector(results []Result) *cycleDetector {
	adjList := make(map[string][]string)

	// Build adjacency list from declared dependencies
	for _, r := range results {
		adjList[r.NodeID] = r.DeclaredDeps
	}

	return &cycleDetector{
		adjList: adjList,
		state:   make(map[string]int),
		path:    make([]string, 0),
		cycles:  make([][]string, 0),
	}
}

// detectCycles finds all cycles in the dependency graph using DFS
func (d *cycleDetector) detectCycles() [][]string {
	// Run DFS from each unvisited node
	for node := range d.adjList {
		if d.state[node] == 0 {
			d.dfs(node)
		}
	}
	return d.cycles
}

// dfs performs depth-first search to detect cycles
func (d *cycleDetector) dfs(node string) {
	d.state[node] = 1 // Mark as visiting
	d.path = append(d.path, node)

	// Explore dependencies
	for _, dep := range d.adjList[node] {
		if d.state[dep] == 0 {
			// Unvisited: continue DFS
			d.dfs(dep)
			continue
		}

		if d.state[dep] == 1 {
			// Back edge detected: cycle found
			cycle := d.extractCycle(dep)
			if cycle != nil {
				d.cycles = append(d.cycles, cycle)
			}
			continue
		}

		// If state[dep] == 2 (visited), no cycle through this edge
	}

	// Backtrack
	d.path = d.path[:len(d.path)-1]
	d.state[node] = 2 // Mark as visited
}

// extractCycle extracts the cycle path from the current DFS path
func (d *cycleDetector) extractCycle(backEdgeTarget string) []string {
	// Find where the cycle starts in the current path
	cycleStart := -1
	for i, node := range d.path {
		if node == backEdgeTarget {
			cycleStart = i
			break
		}
	}

	if cycleStart == -1 {
		// Should not happen if algorithm is correct
		return nil
	}

	// Extract cycle path and append the back edge target to close the loop
	cycle := make([]string, 0, len(d.path)-cycleStart+1)
	cycle = append(cycle, d.path[cycleStart:]...)
	cycle = append(cycle, backEdgeTarget)

	return cycle
}

// mapCyclesToNodes maps each node to all cycles it participates in
func (d *cycleDetector) mapCyclesToNodes() map[string][][]string {
	nodeCycles := make(map[string][][]string)

	for _, cycle := range d.cycles {
		// Add this cycle to all participating nodes (except the last duplicate)
		seen := make(map[string]bool)
		for i := 0; i < len(cycle)-1; i++ {
			node := cycle[i]
			// Only add unique cycles per node
			if !seen[node] {
				nodeCycles[node] = append(nodeCycles[node], cycle)
				seen[node] = true
			}
		}
	}

	return nodeCycles
}
