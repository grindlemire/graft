# Fan-Out Example

Demonstrates wide parallelization where a single root node fans out to many parallel workers, then converges to an aggregator.

## Dependency Graph

```text
                   ┌────────┐
                   │ config │
                   └────────┘
    ┌─────────┬─────────┼─────────┬─────────┐
    │         │         │         │         │
    ▼         ▼         ▼         ▼         ▼
┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐
│ svc1 │  │ svc2 │  │ svc3 │  │ svc4 │  │ svc5 │
└──────┘  └──────┘  └──────┘  └──────┘  └──────┘
    └─────────┴─────────┼─────────┴─────────┘
                        │
                        ▼
                 ┌────────────┐
                 │ aggregator │
                 └────────────┘
```

## What It Demonstrates

- **Maximum parallelization**: All 5 service nodes run concurrently after config completes
- **Fan-out pattern**: One node spawning many parallel children
- **Fan-in pattern**: Many nodes converging to a single aggregator
- **Scalability**: This pattern scales well for batch processing, parallel API calls, etc.

## Run It

```bash
cd examples/fanout
go run .
```

## Run Tests

```bash
cd examples/fanout
go test -v
```

## Expected Output

```text
[config] Loading configuration...
[config] Done
[svc1] Processing...  [svc2] Processing...  [svc3] Processing...  [svc4] Processing...  [svc5] Processing...
[svc3] Done (150ms)
[svc1] Done (200ms)
[svc5] Done (180ms)
[svc2] Done (220ms)
[svc4] Done (170ms)
[aggregator] Aggregating 5 service results...
[aggregator] Done

=== Timing ===
Total: ~320ms (config + max(svc1..5) + aggregator)
Sequential would be: ~1020ms
```

## Key Insight

With 5 services each taking ~200ms, sequential execution would take ~1 second.
With parallel execution, all 5 run simultaneously, so the middle tier only takes as long as the slowest service (~220ms).
