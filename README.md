# graft

A graph-based dependency execution framework for Go. Nodes declare their dependencies explicitly, and the engine executes them in topological order with automatic parallelization.

## Install

```bash
go get github.com/grindlemire/graft
```

## Usage

### Define Nodes

Each node is typically its own package with an `init()` function that registers it:

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
    graft.Register(graft.Node{
        ID:        ID,
        DependsOn: []string{}, // root node
        Run: func(ctx context.Context) (any, error) {
            return Output{
                DBHost: "localhost",
                Port:   5432,
            }, nil
        },
    })
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
    graft.Register(graft.Node{
        ID:        ID,
        DependsOn: []string{config.ID},
        Run: func(ctx context.Context) (any, error) {
            cfg, err := graft.Dep[config.Output](ctx, config.ID)
            if err != nil {
                return nil, err
            }
            
            pool, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d", cfg.DBHost, cfg.Port))
            if err != nil {
                return nil, err
            }
            
            return Output{Pool: pool}, nil
        },
    })
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

## API Reference

### Core Types

```go
// Node represents a unit of work in the dependency graph.
type Node struct {
    ID        string
    DependsOn []string
    Run       func(ctx context.Context) (any, error)
}

// Dep retrieves a dependency's output from context with type assertion.
func Dep[T any](ctx context.Context, nodeID string) (T, error)
```

### Registration

```go
// Register adds a node to the global registry (call from init).
func Register(node Node)

// Registry returns all registered nodes.
func Registry() map[string]Node
```

### Engine

```go
// New creates an engine from a map of nodes.
func New(nodes map[string]Node) *Engine

// Build creates an engine using all registered nodes.
func Build() *Engine

// Run executes nodes in topological order with parallel execution.
func (e *Engine) Run(ctx context.Context) error

// Results returns all node outputs after execution.
func (e *Engine) Results() map[string]any
```

### Builder

```go
// NewBuilder creates a builder for subgraph extraction.
func NewBuilder(catalog map[string]Node) *Builder

// BuildFor creates an engine with target nodes and their transitive deps.
func (b *Builder) BuildFor(targetNodeIDs ...string) (*Engine, error)
```

## License

MIT
