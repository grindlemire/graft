# HTTP Server Example

Demonstrates using graft's node-level caching for HTTP servers where global resources (config, DB) should initialize once while request-scoped nodes run fresh per request.

## Dependency Graph

```text
              ┌────────┐
              │ config │ (cached)
              └───┬────┘
          ┌───────┴───────┐
          ▼               ▼
       ┌────┐         ┌───────┐
       │ db │         │ admin │
       └──┬─┘         └───┬───┘
          │  (cached)     │ (per-request)
          ▼               │
       ┌──────┐           │
       │ user │◀──────────┘
       └──┬───┘ (per-request)
          │
          ▼
   ┌────────────────┐
   │ request_logger │ (per-request)
   └────────────────┘
```

## How It Works

Nodes declare `Cacheable: true` to be cached after first execution:

```go
graft.Register(graft.Node[Output]{
    ID:        "config",
    DependsOn: []graft.ID{},
    Run:       run,
    Cacheable: true, // executed once, then served from cache
})
```

Request-scoped nodes omit `Cacheable` and run fresh every time.

## Run It

```bash
cd examples/httpserver
go run .
```

Make requests:

```bash
curl http://localhost:8080/user/123
curl http://localhost:8080/user/456
curl http://localhost:8080/stats
```

## Run Tests

```bash
cd examples/httpserver
go test -v
```

## Expected Behavior

```text
=== Request req-1: GET /user/123 ===
[config] Executing (execution #1)
[db] Connecting (execution #1)
[request_logger] Logging req-1
[user] Fetching user 123

=== Request req-2: GET /user/456 ===
[request_logger] Logging req-2      ← config/db served from cache
[user] Fetching user 456

=== Stats after 5 requests ===
{
  "total_requests": 5,
  "config_executions": 1,    ← cached!
  "db_executions": 1         ← cached!
}
```

Config and DB execute once; request-scoped nodes run every request.
