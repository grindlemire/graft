package graft

import (
	"fmt"
	"sort"
	"strings"
)

// graphRenderer handles rendering the dependency graph to ASCII.
type graphRenderer struct {
	nodes  map[ID]node
	levels [][]ID

	// Layout state
	nodePositions map[ID]position // node ID -> (row, col) in grid
	levelRows     map[int][]int   // level index -> list of row numbers
	grid          [][]rune        // 2D grid for rendering
	maxRow        int
	maxCol        int
}

type position struct {
	row int
	col int
}

// connectorInfo tracks connection information at a grid position for junction fixing.
type connectorInfo struct {
	row     int
	col     int
	hasUp   bool // connection coming from above
	hasDown bool // connection going below
}

func newGraphRenderer(nodes map[ID]node, levels [][]ID) *graphRenderer {
	return &graphRenderer{
		nodes:         nodes,
		levels:        levels,
		nodePositions: make(map[ID]position),
		levelRows:     make(map[int][]int),
	}
}

func (gr *graphRenderer) render() string {
	gr.computeLayout()
	gr.drawNodes()
	gr.drawEdges()
	return gr.gridToString()
}

// computeLayout determines where each node should be placed in the grid.
func (gr *graphRenderer) computeLayout() {
	// Calculate node widths (including box borders)
	nodeWidths := make(map[ID]int)
	for id := range gr.nodes {
		width := len(string(id))
		if gr.nodes[id].cacheable {
			width++ // Add space for * marker
		}
		width += 4 // Box borders: "│ " + " │"
		if width < 7 {
			width = 7 // Minimum width for readability
		}
		nodeWidths[id] = width
	}

	// Group levels into rows
	type rowInfo struct {
		nodes    []ID
		levelIdx int
	}
	var rows []rowInfo
	const minSpacing = 2

	for levelIdx, level := range gr.levels {
		// Sort level for deterministic output
		sortedLevel := make([]ID, len(level))
		copy(sortedLevel, level)
		sort.Slice(sortedLevel, func(i, j int) bool {
			return sortedLevel[i] < sortedLevel[j]
		})

		// Put all nodes in a single row for this level
		rows = append(rows, rowInfo{nodes: sortedLevel, levelIdx: levelIdx})
	}

	// Calculate level widths to determine maximum for centering
	levelWidths := make(map[int]int)
	for _, rowInfo := range rows {
		levelIdx := rowInfo.levelIdx
		rowNodes := rowInfo.nodes
		totalWidth := 0
		for i, id := range rowNodes {
			totalWidth += nodeWidths[id]
			if i < len(rowNodes)-1 {
				totalWidth += minSpacing
			}
		}
		if totalWidth > levelWidths[levelIdx] {
			levelWidths[levelIdx] = totalWidth
		}
	}

	// Find maximum level width for centering
	maxLevelWidth := 0
	for _, width := range levelWidths {
		if width > maxLevelWidth {
			maxLevelWidth = width
		}
	}

	// Calculate positions
	gr.maxRow = 0
	gr.maxCol = 0
	rowOffset := 0

	for _, rowInfo := range rows {
		levelIdx := rowInfo.levelIdx
		rowNodes := rowInfo.nodes

		// Track which rows belong to this level
		gr.levelRows[levelIdx] = append(gr.levelRows[levelIdx], rowOffset)

		// Calculate row width
		rowWidth := 0
		for i, id := range rowNodes {
			rowWidth += nodeWidths[id]
			if i < len(rowNodes)-1 {
				rowWidth += minSpacing
			}
		}

		// Center this row relative to the maximum level width
		spacing := minSpacing
		startCol := (maxLevelWidth - rowWidth) / 2
		if startCol < 0 {
			startCol = 0
		}

		// Position nodes in this row
		col := startCol
		for _, id := range rowNodes {
			gr.nodePositions[id] = position{
				row: rowOffset,
				col: col,
			}
			col += nodeWidths[id] + spacing
		}

		// Update grid dimensions
		if col > gr.maxCol {
			gr.maxCol = col
		}
		gr.maxRow = rowOffset + 3 // Each node takes 3 rows (box + space)

		// Add spacing between levels (6 rows: 3 for box + 3 for connector/drop/arrow)
		rowOffset += 6
	}

	// Initialize grid
	gr.grid = make([][]rune, gr.maxRow+1)
	for i := range gr.grid {
		gr.grid[i] = make([]rune, gr.maxCol+1)
		for j := range gr.grid[i] {
			gr.grid[i][j] = ' '
		}
	}
}

// drawNodes draws the node boxes in the grid.
func (gr *graphRenderer) drawNodes() {
	for id, pos := range gr.nodePositions {
		width := len(string(id))
		if gr.nodes[id].cacheable {
			width++
		}
		width += 4 // Box borders

		// Draw box
		// Top border
		gr.setChar(pos.row, pos.col, '┌')
		for i := 1; i < width-1; i++ {
			gr.setChar(pos.row, pos.col+i, '─')
		}
		gr.setChar(pos.row, pos.col+width-1, '┐')

		// Middle with text
		text := string(id)
		if gr.nodes[id].cacheable {
			text += "*"
		}
		gr.setChar(pos.row+1, pos.col, '│')
		gr.setString(pos.row+1, pos.col+2, text)
		gr.setChar(pos.row+1, pos.col+width-1, '│')

		// Bottom border
		gr.setChar(pos.row+2, pos.col, '└')
		for i := 1; i < width-1; i++ {
			gr.setChar(pos.row+2, pos.col+i, '─')
		}
		gr.setChar(pos.row+2, pos.col+width-1, '┘')
	}
}

// drawEdges draws the edges connecting nodes using a two-pass approach.
// Pass 1: Draw all vertical and horizontal lines with simple characters.
// Pass 2: Scan the grid and fix all junctions based on neighboring characters.
func (gr *graphRenderer) drawEdges() {
	children := gr.buildChildMap()
	childrenByLevel := gr.groupChildrenByLevel(children)
	connectors := gr.drawAllEdgeLines(children, childrenByLevel)
	gr.fixJunctions(connectors)
}

// buildChildMap builds a reverse dependency map: child -> parents.
func (gr *graphRenderer) buildChildMap() map[ID][]ID {
	children := make(map[ID][]ID)
	for id, n := range gr.nodes {
		for _, dep := range n.dependsOn {
			children[id] = append(children[id], dep)
		}
	}
	return children
}

// groupChildrenByLevel groups children by their level to handle wrapping.
func (gr *graphRenderer) groupChildrenByLevel(children map[ID][]ID) map[int][]ID {
	childrenByLevel := make(map[int][]ID)
	for childID := range children {
		// Find which level this child belongs to
		for levelIdx, level := range gr.levels {
			for _, id := range level {
				if id == childID {
					childrenByLevel[levelIdx] = append(childrenByLevel[levelIdx], childID)
					break
				}
			}
		}
	}
	return childrenByLevel
}

// drawAllEdgeLines draws all edge lines and returns connector information for junction fixing.
func (gr *graphRenderer) drawAllEdgeLines(children map[ID][]ID, childrenByLevel map[int][]ID) map[string]*connectorInfo {
	connectors := make(map[string]*connectorInfo) // key: "row,col"

	getConnector := func(row, col int) *connectorInfo {
		key := fmt.Sprintf("%d,%d", row, col)
		if c, ok := connectors[key]; ok {
			return c
		}
		c := &connectorInfo{row: row, col: col}
		connectors[key] = c
		return c
	}

	// Draw edges level by level
	for levelIdx, childIDs := range childrenByLevel {
		// Get all rows that contain children from this level
		childRows := gr.levelRows[levelIdx]
		if len(childRows) == 0 {
			continue
		}

		// Collect all parents of children in this level
		allParentIDs := make(map[ID]bool)
		for _, childID := range childIDs {
			for _, parentID := range children[childID] {
				allParentIDs[parentID] = true
			}
		}

		// Draw edges from each parent to all its children in this level
		for parentID := range allParentIDs {
			gr.drawParentToChildrenEdges(parentID, childIDs, children, childRows, getConnector)
		}
	}

	return connectors
}

// drawParentToChildrenEdges draws edges from a parent to its children in a specific level.
func (gr *graphRenderer) drawParentToChildrenEdges(
	parentID ID,
	childIDs []ID,
	children map[ID][]ID,
	childRows []int,
	getConnector func(int, int) *connectorInfo,
) {
	// Find which children in this level depend on this parent
	var targetChildren []ID
	for _, childID := range childIDs {
		for _, dep := range children[childID] {
			if dep == parentID {
				targetChildren = append(targetChildren, childID)
				break
			}
		}
	}

	if len(targetChildren) == 0 {
		return
	}

	parentPos := gr.nodePositions[parentID]
	parentCol := parentPos.col + gr.getNodeCenterOffset(parentID)
	parentBottomRow := parentPos.row + 2

	// Find the connect row (horizontal connector) and arrow row (with vertical drop)
	firstChildRow := childRows[0]
	// Arrow should be one row above the child's top border
	arrowRow := firstChildRow - 1
	// Horizontal connector should be at least 2 rows above the arrow (for vertical drop)
	calculatedConnectRow := arrowRow - 2
	connectRow := calculatedConnectRow
	// Ensure connectRow is below parent's bottom border
	if connectRow <= parentBottomRow {
		// If calculated row is at or above parent bottom, place it one row below
		connectRow = parentBottomRow + 1
	}

	// Draw vertical line from parent down to connect row (inclusive)
	for row := parentBottomRow + 1; row <= connectRow; row++ {
		gr.setChar(row, parentCol, '│')
	}
	// Mark this connector as having an "up" connection
	getConnector(connectRow, parentCol).hasUp = true

	// Collect child center columns
	childCols := make([]int, len(targetChildren))
	for i, childID := range targetChildren {
		childPos := gr.nodePositions[childID]
		childCols[i] = childPos.col + gr.getNodeCenterOffset(childID)
	}

	// Find horizontal span
	minCol := parentCol
	maxCol := parentCol
	for _, col := range childCols {
		if col < minCol {
			minCol = col
		}
		if col > maxCol {
			maxCol = col
		}
	}

	// Draw horizontal connector at connect row
	for col := minCol; col <= maxCol; col++ {
		gr.setChar(connectRow, col, '─')
	}

	// Draw vertical lines from horizontal connector down to arrow row (inclusive of connect row)
	for _, col := range childCols {
		// Mark this connector as having a "down" connection
		getConnector(connectRow, col).hasDown = true

		// Draw vertical line from connect row down to arrow row
		for row := connectRow; row <= arrowRow; row++ {
			gr.setChar(row, col, '│')
		}
		// Place arrow at arrow row
		gr.setChar(arrowRow, col, '▼')
	}

	// Draw vertical lines from arrow down to each child
	for i, childID := range targetChildren {
		childPos := gr.nodePositions[childID]
		childCol := childCols[i]
		childTopRow := childPos.row

		// Draw vertical line from arrow row down to child
		for row := arrowRow + 1; row < childTopRow; row++ {
			gr.setChar(row, childCol, '│')
		}
	}
}

// fixJunctions scans the grid and fixes all junctions based on neighboring characters.
func (gr *graphRenderer) fixJunctions(connectors map[string]*connectorInfo) {
	// Build a set of rows that contain node boxes (to avoid modifying box characters)
	nodeBoxRows := make(map[int]bool)
	for _, pos := range gr.nodePositions {
		nodeBoxRows[pos.row] = true   // top border
		nodeBoxRows[pos.row+1] = true // content
		nodeBoxRows[pos.row+2] = true // bottom border
	}

	// Scan all non-node-box rows for line characters and fix junctions
	for row := 0; row < len(gr.grid); row++ {
		// Skip rows that contain node boxes
		if nodeBoxRows[row] {
			continue
		}

		for col := 0; col < len(gr.grid[row]); col++ {
			current := gr.getChar(row, col)

			// Skip non-line characters
			if current != '│' && current != '─' {
				continue
			}

			// Check neighbors
			up := gr.getChar(row-1, col)
			down := gr.getChar(row+1, col)
			left := gr.getChar(row, col-1)
			right := gr.getChar(row, col+1)

			hasUp := isVerticalConnector(up)
			hasDown := isVerticalConnector(down)
			hasLeft := isHorizontalConnector(left)
			hasRight := isHorizontalConnector(right)

			// Check connector info for logical connections (even if no line was drawn due to tight spacing)
			if c, ok := connectors[fmt.Sprintf("%d,%d", row, col)]; ok {
				hasUp = hasUp || c.hasUp
				hasDown = hasDown || c.hasDown
			}

			// Determine the correct glyph
			glyph := gr.selectJunctionGlyph(hasUp, hasDown, hasLeft, hasRight)
			if glyph != current {
				gr.setChar(row, col, glyph)
			}
		}
	}
}

// isVerticalConnector checks if a rune represents a vertical connection.
func isVerticalConnector(r rune) bool {
	switch r {
	case '│', '┼', '├', '┤', '┬', '┴', '▼':
		return true
	}
	return false
}

// isHorizontalConnector checks if a rune represents a horizontal connection.
func isHorizontalConnector(r rune) bool {
	switch r {
	case '─', '┼', '├', '┤', '┬', '┴', '┌', '┐', '└', '┘':
		return true
	}
	return false
}

// selectJunctionGlyph returns the appropriate junction glyph based on connections.
func (gr *graphRenderer) selectJunctionGlyph(up, down, left, right bool) rune {
	switch {
	case up && down && left && right:
		return '┼' // cross
	case up && down && left && !right:
		return '┤' // T left
	case up && down && !left && right:
		return '├' // T right
	case up && down && !left && !right:
		return '│' // vertical only
	case up && !down && left && right:
		return '┴' // T up
	case up && !down && left && !right:
		return '┘' // corner bottom-right
	case up && !down && !left && right:
		return '└' // corner bottom-left
	case up && !down && !left && !right:
		return '│' // vertical only (stub up)
	case !up && down && left && right:
		return '┬' // T down
	case !up && down && left && !right:
		return '┐' // corner top-right
	case !up && down && !left && right:
		return '┌' // corner top-left
	case !up && down && !left && !right:
		return '│' // vertical only (stub down)
	case !up && !down && left && right:
		return '─' // horizontal only
	case !up && !down && left && !right:
		return '─' // horizontal only (stub left)
	case !up && !down && !left && right:
		return '─' // horizontal only (stub right)
	default:
		return ' '
	}
}

// getNodeCenterOffset returns the column offset to the center of a node.
func (gr *graphRenderer) getNodeCenterOffset(id ID) int {
	width := len(string(id))
	if gr.nodes[id].cacheable {
		width++
	}
	width += 4 // Box borders
	return width / 2
}

// Helper methods for grid manipulation
func (gr *graphRenderer) setChar(row, col int, char rune) {
	if row >= 0 && row < len(gr.grid) && col >= 0 && col < len(gr.grid[row]) {
		gr.grid[row][col] = char
	}
}

func (gr *graphRenderer) getChar(row, col int) rune {
	if row >= 0 && row < len(gr.grid) && col >= 0 && col < len(gr.grid[row]) {
		return gr.grid[row][col]
	}
	return ' '
}

func (gr *graphRenderer) setString(row, col int, s string) {
	for i, r := range s {
		gr.setChar(row, col+i, r)
	}
}

func (gr *graphRenderer) gridToString() string {
	var sb strings.Builder
	for _, row := range gr.grid {
		// Trim trailing spaces
		line := strings.TrimRight(string(row), " ")
		if line != "" {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
