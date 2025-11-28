# Simple Example

A minimal example demonstrating the basic graft pattern with a linear dependency chain.

## Dependency Tree

```bash
config
   │
   ▼
   db
   │
   ▼
  app
```

Each node depends on exactly one other node, forming a simple sequential chain. This is the simplest possible graft setup.

## What It Demonstrates

- Self-registering nodes via `init()` and blank imports
- The `graft.Dep[T]()` pattern for accessing dependency outputs
- Sequential execution when nodes have linear dependencies
- Type-safe node outputs with `Node[T]`

## Run It

```bash
cd examples/simple
go run .
```

## Expected Output

```bash
[config] Loading configuration...
[config] Done
[db] Connecting to database at localhost:5432...
[db] Done
[app] Starting application with db pool...
[app] Done

=== Results ===
app: {AppName:MyApp Version:1.0.0}
config: {Host:localhost Port:5432}
db: {Connected:true PoolSize:10}
```

Nodes execute in order: config → db → app, since each depends on the previous.
