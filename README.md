# graft

A graph-based dependency execution framework for Go. Nodes declare their dependencies explicitly, and the engine executes them in topological order with automatic parallelization.

## Features

- **Type-safe nodes** — Generic `Node[T]` ensures compile-time type checking on return values
- **Declarative dependencies** — Nodes specify what they depend on, not how to get it
- **Automatic parallelization** — Nodes at the same level run concurrently
- **Type-safe dependency access** — Generic `Dep[T]` function with compile-time type checking
- **Subgraph execution** — Build engines for specific targets with automatic transitive dependency resolution
- **Static analysis** — Validate dependency declarations match actual usage at test time

## Install

```bash
go get github.com/grindlemire/graft
```

## Usage

### Define Nodes

Each node is typically its own package with an `init()` function that registers it. The `Node[T]` type parameter specifies the output type:

```go
// nodes/config/config.go
package config

import (
    "context"
    "github.com/grindlemire/graft"
)

const ID = "config"

type Output struct {
    DBHost string
    Port   int
}

func init() {
    graft.Register(graft.Node[Output]{
        ID:        ID,
        DependsOn: []string{}, // root node
        Run:       run,
    })
}

// Compiler enforces this returns Output
func run(ctx context.Context) (Output, error) {
    return Output{
        DBHost: "localhost",
        Port:   5432,
    }, nil
}
```

Nodes that depend on others use `graft.Dep[T]` to retrieve dependency outputs:

```go
// nodes/db/db.go
package db

import (
    "context"
    "github.com/grindlemire/graft"
    "myapp/nodes/config"
)

const ID = "db"

type Output struct {
    Pool *sql.DB
}

func init() {
    graft.Register(graft.Node[Output]{
        ID:        ID,
        DependsOn: []string{config.ID},
        Run:       run,
    })
}

// Compiler enforces this returns Output
func run(ctx context.Context) (Output, error) {
    cfg, err := graft.Dep[config.Output](ctx, config.ID)
    if err != nil {
        return Output{}, err
    }
    
    pool, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d", cfg.DBHost, cfg.Port))
    if err != nil {
        return Output{}, err
    }
    
    return Output{Pool: pool}, nil
}
```

### Run the Graph

Import all node packages (side-effect imports trigger registration), then build and run:

```go
package main

import (
    "context"
    "log"

    "github.com/grindlemire/graft"
    _ "myapp/nodes/config"
    _ "myapp/nodes/db"
    _ "myapp/nodes/cache"
    _ "myapp/nodes/api"
)

func main() {
    engine := graft.Build()
    
    if err := engine.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
    
    results := engine.Results()
    // Use results as needed
}
```

### Subgraph Execution

For cases where you only need a subset of nodes (e.g., different HTTP endpoints needing different dependencies):

```go
builder := graft.NewBuilder(graft.Registry())

// Only builds "api" and its transitive dependencies
engine, err := builder.BuildFor("api")
if err != nil {
    log.Fatal(err)
}

if err := engine.Run(ctx); err != nil {
    log.Fatal(err)
}
```

## Dependency Validation

Graft includes static analysis tools to verify that your dependency declarations are correct at test time. This catches common errors like:

- Using `Dep[T](ctx, "x")` without declaring `"x"` in `DependsOn`
- Declaring a dependency in `DependsOn` that is never used

### Quick Setup

Add a single test to validate all nodes in your project:

```go
// nodes/deps_test.go
package nodes_test

import (
    "testing"
    "github.com/grindlemire/graft"
)

func TestNodeDependencies(t *testing.T) {
    graft.AssertDepsValid(t, "./")
}
```

### Example Output

When validation fails, you get detailed error messages:

```text
=== RUN   TestNodeDependencies
    deps_test.go:10: graft.AssertDepsValid: db (nodes/db/db.go): undeclared deps: [cache]
    deps_test.go:10:   → node "db" uses Dep[T](ctx, "cache") but does not declare it in DependsOn
--- FAIL: TestNodeDependencies (0.01s)
```

### Programmatic Access

For custom validation logic or CI integration:

```go
// Check without failing
results, err := graft.CheckDepsValid("./nodes")
if err != nil {
    log.Fatal(err)
}

for _, r := range results {
    if r.HasIssues() {
        fmt.Printf("Node %s has issues:\n", r.NodeID)
        fmt.Printf("  Undeclared: %v\n", r.Undeclared)
        fmt.Printf("  Unused: %v\n", r.Unused)
    }
}

// Or use ValidateDeps for a simple pass/fail
if err := graft.ValidateDeps("./nodes"); err != nil {
    log.Fatal(err)
}
```

## License

MIT
