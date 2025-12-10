# Diamond Example

Demonstrates the classic diamond dependency pattern where two nodes run in parallel and then converge.

## Dependency Graph

```text
         ┌────────┐
         │ config │
         └───┬────┘
        ┌────┴────┐
        ▼         ▼
     ┌────┐   ┌───────┐
     │ db │   │ cache │
     └──┬─┘   └───┬───┘
        └────┬────┘
             ▼
         ┌─────┐
         │ api │
         └─────┘
```

## What It Demonstrates

- **Parallel execution**: `db` and `cache` both depend only on `config`, so they run concurrently
- **Dependency convergence**: `api` waits for both `db` and `cache` to complete before running
- **Multiple dependencies**: A node can depend on multiple other nodes
- **Automatic parallelization**: Graft detects that `db` and `cache` are independent and runs them in parallel

## Run It

```bash
cd examples/diamond
go run .
```

## Run Tests

```bash
cd examples/diamond
go test -v
```

## Expected Output

```text
[config] Loading configuration...
[config] Done (100ms)
[db] Connecting to database...       [cache] Connecting to Redis...
[db] Done (200ms)                    [cache] Done (150ms)
[api] Initializing API with db and cache...
[api] Done (100ms)

=== Execution Time ===
Total: ~550ms (not 650ms, because db and cache ran in parallel)
```

The key observation is that total execution time is approximately:

- config (100ms) + max(db, cache) (200ms) + api (100ms) = ~400ms

Not the sequential sum of all nodes (~550ms), proving parallel execution.
