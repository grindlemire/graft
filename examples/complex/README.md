# Complex Example

A realistic multi-level dependency graph with shared dependencies, multiple convergence points, and varied parallelization.

## Dependency Graph

```text
            ┌─────┐
            │ env │
            └─────┘
   ┌──────────┬┴──────────┐
   │          │           │
   ▼          ▼           ▼
┌─────┐  ┌────────┐  ┌─────────┐
│ cfg │  │ logger │  │ secrets │
└─────┘  └────────┘  └─────────┘
   └───────┬──┴─────┬─────┘
           │        │
           ▼        ▼
       ┌──────┐  ┌────┐
       │ auth │  │ db │
       └──────┘  └────┘
          ┌┴────────┘┐
          │          │
          ▼          ▼
      ┌───────┐  ┌──────┐
      │ admin │  │ user │
      └───────┘  └──────┘
          └────┬─────┘
               │
               ▼
          ┌─────────┐
          │ gateway │
          └─────────┘
```

## Execution Levels

1. **Level 0**: `env` (root)
2. **Level 1**: `config`, `secrets`, `logger` (run in parallel)
3. **Level 2**: `db`, `auth` (run in parallel - auth waits for secrets+logger, db waits for config)
4. **Level 3**: `user`, `admin` (run in parallel)
5. **Level 4**: `gateway` (final convergence)

## What It Demonstrates

- **Shared dependencies**: Both `auth` and `db` depend on different subsets of level 1
- **Multiple convergence points**: `user` depends on db+auth, `admin` depends only on auth
- **Mixed fan-out/fan-in**: Several parallel branches with different convergence patterns
- **Realistic architecture**: Mirrors a real application with config, secrets, database, auth, and API layers

## Run It

```bash
cd examples/complex
go run .
```

## Run Tests

```bash
cd examples/complex
go test -v
```

## Expected Output

The output shows the layered execution:

```text
[env] Loading environment...
[env] Done
[config] Loading...  [secrets] Loading...  [logger] Initializing...
[logger] Done
[secrets] Done  
[config] Done
[db] Connecting...  [auth] Initializing auth with secrets...
[db] Done
[auth] Done
[user] Loading user service...  [admin] Loading admin service...
[user] Done
[admin] Done
[gateway] Starting gateway...
[gateway] Done

=== Timing ===
Total: ~500ms (with parallel execution)
Sequential would be: ~950ms
```

## Key Insight

Even though there are 9 nodes, the execution only takes 5 "levels" of time because nodes at the same level run concurrently.
