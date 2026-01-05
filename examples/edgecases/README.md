# Edge Case Test Suite for Type-Aware Graph Validation

This directory contains 20 comprehensive edge case examples for testing the type-aware graph validation system in graft. Each edge case is a complete Go module that demonstrates different scenarios for dependency analysis.

## Test Categories

### Undeclared Dependencies (6 cases)
Dependencies used via `graft.Dep[T](ctx)` but not listed in `DependsOn`:

- **01_undeclared_single**: Single undeclared dependency (app uses config but doesn't declare it)
- **02_undeclared_multiple**: Multiple undeclared dependencies (app uses 3 deps, declares none)
- **09_mixed_undeclared_unused**: Both undeclared and unused in same node
- **10_mixed_cycle_undeclared**: Cycle + undeclared dependency
- **15_partial_declaration**: Some declared, some undeclared
- **20_conditional_dep_usage**: Dependency used in conditional (SSA catches it)

### Unused Dependencies (5 cases)
Dependencies declared in `DependsOn` but never accessed:

- **03_unused_single**: Single unused dependency (declares but never uses)
- **04_unused_multiple**: Multiple unused dependencies
- **09_mixed_undeclared_unused**: (also tests unused)
- **14_unused_in_chain**: Some deps used, some unused in a chain
- **18_complex_multi_parent**: Unused in diamond structure

### Cycle Detection (6 cases)
Circular dependencies detected via DFS:

- **05_cycle_simple**: 2-node cycle (A→B→A)
- **06_cycle_triangle**: 3-node cycle (A→B→C→A)
- **07_cycle_deep**: Long chain cycle (C→D→E→C, with A→B upstream)
- **08_cycle_self**: Self-referential cycle (A→A)
- **10_mixed_cycle_undeclared**: (also tests cycles)
- **16_multiple_cycles_same_node**: Hub node in multiple distinct cycles

### Mixed Issues (1 case)
Combined validation issues:

- **11_mixed_all_issues**: Single node with undeclared, unused, AND cycle

### Structural/Valid Cases (5 cases)
Various graph structures and valid configurations:

- **12_empty_node**: Minimal node with no logic
- **13_no_deps_node**: Completely independent node
- **17_long_chain**: Deep linear chain (n1→n2→...→n10)
- **18_complex_multi_parent**: Diamond structure with multiple parents
- **19_orphan_nodes**: Disconnected subgraphs

## Structure

Each edge case directory follows this structure:
```
XX_case_name/
├── go.mod              # Module definition with replace directive
├── main.go             # Imports all nodes
├── main_test.go        # Calls graft.AssertDepsValid
└── nodes/              # Node implementations
    ├── node1/
    │   └── node1.go
    ├── node2/
    │   └── node2.go
    └── ...
```

## Running Tests

All edge cases are tested via table-driven tests in `analyze_edgecase_test.go` at the project root:

```bash
# Run all edge case tests
go test -v -run TestAnalyzeDirEdgeCases

# Run specific category
go test -v -run TestAnalyzeDirEdgeCases_Undeclared
go test -v -run TestAnalyzeDirEdgeCases_Cycles
```

## Implementation Notes

### Go Import Cycle Limitation

For cycle test cases (05-08, 10-11, 16), there's a technical constraint: Go doesn't allow package import cycles. To test graft dependency cycles (e.g., nodeA→nodeB→nodeA), we use string literals for dependency IDs:

```go
// In nodeA/nodeA.go
DependsOn: []graft.ID{"nodeB"}  // string literal instead of nodeB.ID
```

This creates the graft dependency cycle without causing a Go package import cycle. However, it means we cannot call `graft.Dep[nodeB.Output](ctx)` due to the import restriction, so these dependencies appear as "unused" in the analysis. Test expectations account for this limitation.

### SSA Analysis

Case 20 demonstrates that the SSA-based analysis correctly detects dependency usage even in conditionals:

```go
// Using feature in conditional - SSA catches this
f, _ := graft.Dep[feature.Output](ctx)  // undeclared
if f.Enabled {
    result += "-feature-enabled"
}
```

## Test Helpers

The test suite includes comprehensive helper functions in `analyze_edgecase_test.go`:

- `findNode(t, results, id)` - Find node by ID
- `assertUndeclared(t, node, want)` - Check undeclared deps
- `assertUnused(t, node, want)` - Check unused deps
- `assertCycles(t, node, wantCount)` - Check cycle count
- `assertCycleContains(t, node, path)` - Verify specific cycle path
- `assertDeps(t, node, wantDeclared, wantUsed)` - Check declared vs used
- Order-independent slice comparison helpers

## Coverage

This test suite provides comprehensive coverage of:
- Single and multiple dependency issues
- Cycle detection in various graph structures
- Mixed issue scenarios
- Edge cases in graph topology
- Valid configurations that should pass
- Scalability with deep chains (10 nodes)
- Disconnected subgraphs
