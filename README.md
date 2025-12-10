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

Lightweight, type-safe dependency injection for Go.

```text
Example Graph:
                                    ┌───────┐
                              ┌────▶│ cache │────┐
                              │     └───────┘    │
┌────────┐    ┌─────────┐     │                  │     ┌────────┐
│ config │───▶│ secrets │─────┤                  ├────▶│ server │
└────────┘    └─────────┘     │                  │     └────────┘
                              │     ┌────┐       │
                              └────▶│ db │───────┘
                                    └────┘
```

You define independent nodes for your dependency graph. Graft assembles and runs your graph optimally, and gives you type-safe access to dependencies. No reflection or codegen and a minimal but flexible API.

## Why Graft?

- **Simple** - No complex arg routing even in large projects.
- **No Codegen** - No extra build step or opaque code.
- **Type Safe** - Compile-time type safety for inputs and outputs and no reflection.
- **Concurrent** - Independent nodes execute in parallel automatically.
- **Validatable** - Validate your entire graph in [CI and compile time](#validation)

## Install

```bash
go get github.com/grindlemire/graft
```

Full API documentation: [pkg.go.dev/github.com/grindlemire/graft](https://pkg.go.dev/github.com/grindlemire/graft)

## Usage

### 1. Define Nodes

Create a package for your node and register it in `init()`.

```go
// nodes/db/db.go
package db

import (
    "context"
    "github.com/grindlemire/graft"
    "myapp/nodes/config"
)

// the ID for this node
const ID graft.ID = "db"

// what type you want to output from this node for others to consume
type Output struct { Pool *sql.DB }

func init() {
    // register our node with graft for our output type
    graft.Register(graft.Node[Output]{
        ID:        ID,
        // list any dependencies here (any cycles won't let you import it)
        DependsOn: []graft.ID{config.ID},
        Run:       run,
    })
}

// run gets executed by graft to produce our output from our dependencies
func run(ctx context.Context) (Output, error) {
    // Type-safe dependency injection
    cfg, err := graft.Dep[config.Output](ctx)
    if err != nil {
        return Output{}, err
    }

    return Output{Pool: connect(cfg)}, nil
}
```

### 2. Execute

Import your nodes for side-effects and run the engine.

```go
package main

import (
    "github.com/grindlemire/graft"
    // import our nodes to register them
    _ "myapp/nodes/config"
    _ "myapp/nodes/db"
)

func main() {
    // Run a specific subgraph
    // Returns the typed output of the target node
    db, results, err := graft.ExecuteFor[db.Output](ctx)
    if err != nil {
         // handle error
    }

    // OR run the entire graph
    // results, err := graft.Execute(ctx)
}
```

### Caching

Skip expensive calculations by marking nodes as cacheable.

```go
graft.Register(graft.Node[Output]{
    // ...
    Cacheable: true,
})
```

Control cache behavior at runtime:

```go
graft.Execute(ctx, graft.WithCache(customCache))
graft.Execute(ctx, graft.IgnoreCache(config.ID))
```

### Validation

Catch missing or unused dependencies during tests.

```go
func TestDeps(t *testing.T) {
    graft.AssertDepsValid(t, ".")
}
```

#### Compile Time Graph Checking

Place each node in its own package and Go's import rules enforce a valid graph for you:

- **No cycles** - To depend on node `B`, node `A` must import `B`'s package. Circular imports don't compile.
- **No missing deps** - Referencing `config.ID` in your `DependsOn` requires importing the config package. If it doesn't exist, the build fails.

See the [examples](./examples) directory for this pattern.

### Visualization

Generate diagrams directly from your code for better visibility in large code bases

```go
// Print ASCII graph to stdout
graft.PrintGraph(os.Stdout)

// Generate Mermaid syntax
graft.PrintMermaid(os.Stdout)
```

## Why not Wire or Fx?

### [Wire](https://github.com/google/wire)

Wire uses code generation to wire dependencies at compile time. It's powerful but adds complexity:

- No longer maintained (archived as of 2025-08-25)
- Requires running `wire` before each build
- Provider sets and injector functions add conceptual overhead

Graft uses plain Go `init()` functions and generics. No extra build step for CI, no generated files to manage, and nothing new to learn.

### [Fx](https://github.com/uber-go/fx)

Fx wires dependencies at runtime using reflection. It's actively maintained and heavily used at Uber, but has trade-offs:

- Errors surface at runtime, not compile time
- Reflection can obscure stack traces, making debugging harder
- The lifecycle model (`fx.Invoke`, `fx.Lifecycle`) adds a ton of boiler plate for most use cases

Graft resolves the graph at execution time but uses generics for type safety. If your types don't match, or you have a dependency cycle then the compiler tells you.

### When to use Graft

Graft is a good fit when you want dependency injection without the tooling overhead of Wire or the runtime reflection of Fx. It is also a much smaller and less complex library compared to either Wire or Fx with many less opinions on how you structure your code.

Graft is intentionally minimal: define your nodes, declare your dependencies, and run.

## License

MIT
