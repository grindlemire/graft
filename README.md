<p align="center">
  <img src="logo.png" alt="graft" width="400">
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/grindlemire/graft"><img src="https://pkg.go.dev/badge/github.com/grindlemire/graft.svg?cacheSeconds=3600" alt="Go Reference"></a>
  <a href="https://github.com/grindlemire/graft/actions/workflows/ci.yml"><img src="https://github.com/grindlemire/graft/actions/workflows/ci.yml/badge.svg?branch=main&cacheSeconds=3600" alt="CI"></a>
  <a href="https://coveralls.io/github/grindlemire/graft?branch=main"><img src="https://coveralls.io/repos/github/grindlemire/graft/badge.svg?branch=main&cacheSeconds=3600" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/grindlemire/graft"><img src="https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat&cacheSeconds=3600" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg?cacheSeconds=86400" alt="License: MIT"></a>
  <a href="./examples"><img src="https://img.shields.io/badge/Examples-📂-green.svg?cacheSeconds=86400" alt="Examples"></a>
</p>

Graph-based dependency execution for Go. Nodes declare dependencies explicitly; the engine executes them in topological order with automatic parallelization.

## Features

- **Type-safe nodes** — Generic `Node[T]` with compile-time type checking
- **Declarative dependencies** — Nodes specify what they need, not how to get it
- **Automatic parallelization** — Independent nodes run concurrently
- **Subgraph execution** — Run only specific nodes and their transitive dependencies
- **Node-level caching** — Cache expensive nodes across executions
- **Static analysis** — Validate dependency declarations at test time

## Install

```bash
go get github.com/grindlemire/graft
```

## Usage

### Define Nodes

Each node is typically its own package with an `init()` that registers it.

Suppose we need to load configuration to access our database:

```go
// nodes/config/config.go
package config

import (
    "context"
    "github.com/grindlemire/graft"
)

// The ID of the node in the engine
const ID graft.ID = "config"

// The output type that other nodes will access
type Output struct {
    DBHost string
    Port   int
}

// init registers the node automatically on startup
func init() {
    graft.Register(graft.Node[Output]{
        ID:        ID,
        DependsOn: []graft.ID{}, // root node
        Run:       run,
    })
}

// run is executed by the engine 
func run(ctx context.Context) (Output, error) {
    return Output{DBHost: "localhost", Port: 8080}, nil
}
```

Now the database can specify the config node as a dependency and the engine
will make sure it is run after the config node is executed. Nodes access dependencies via `graft.Dep[T]`:

```go
// nodes/db/db.go
package db

import (
    "context"
    "github.com/grindlemire/graft"
    "myapp/nodes/config"
)

const ID graft.ID = "db"

// Every node has an output type so other nodes can use it
type Output struct {
    Pool *sql.DB
}

func init() {
    graft.Register(graft.Node[Output]{
        ID:        ID,
        // This node depends on the config node
        DependsOn: []graft.ID{config.ID},
        Run:       run,
        // Nodes can choose to be cached so dependencies don't re-run the code every time
        Cacheable: true
    })
}

func run(ctx context.Context) (Output, error) {
    // We can get the config from the graph using the Dep function.
    cfg, err := graft.Dep[config.Output](ctx)
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

### Execute the Graph

Import node packages for side-effect registration, then execute:

```go
package main

import (
    "context"
    "log"

    "github.com/grindlemire/graft"
    _ "myapp/nodes/config"
    _ "myapp/nodes/db"
    _ "myapp/nodes/api"
)

func main() {
    // We could run the entire graph
    results, err := graft.Execute(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    db := results["db"].(*sql.DB)
    // use db...


    // Or we could just run what we need to for a chosen dependency
    // db, _, err := graft.ExecuteFor[db.Output](context.Background())
}
```

### Subgraph Execution

You can choose to only run a specific node and its transitive dependencies with type-safe results:

```go
// Only executes "api" and whatever it depends on
// Returns typed result directly, plus full results map for accessing dependencies
api, results, err := graft.ExecuteFor[api.Output](ctx)
if err != nil {
    log.Fatal(err)
}
// api is already typed as api.Output
// the results map is available for accessing other node outputs if needed
config, err := graft.Result[config.Output](results)
```

### Caching

Mark nodes as cacheable to avoid re-execution across calls:

```go
graft.Register(graft.Node[Output]{
    ID:        ID,
    DependsOn: []graft.ID{},
    Run:       run,
    Cacheable: true, // output cached after first execution
})
```

By default, a global in-memory cache is used. Options for control:

```go
// Use a custom cache
results, _ := graft.Execute(ctx, graft.WithCache(myCache))

// Force re-execution of specific nodes
results, _ := graft.Execute(ctx, graft.IgnoreCache("config"))

// Disable caching entirely
results, _ := graft.Execute(ctx, graft.DisableCache())
```

## Dependency Validation

This library provides static analysis to catch dependency mismatches at test time:

```go
func TestNodeDependencies(t *testing.T) {
    graft.AssertDepsValid(t, ".")
}
```

Catches:

- Using `Dep[T](ctx)` without declaring the corresponding dependency in `DependsOn`
- Declaring a dependency that's never used

## Why use this library?

As teams or projects scale it can be challenging to efficiently route dependencies through the application while still writing idiomatic Go. Dependency injection frameworks attempt to solve this by allowing packages to just specify their dependencies but not how they are executed. However the larger dependency injection frameworks in Go either rely on reflection or large amounts of codegen.

This library attempts to be a lighter weight and more straightforward alternative without relying either on reflection or code generation. Simply specify nodes in their own package and register them in an init function, then the engine takes care of the rest.

## License

MIT
