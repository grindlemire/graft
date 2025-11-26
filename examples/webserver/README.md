# Web Server Example

Demonstrates using graft to build per-request dependency graphs in an HTTP server. Different endpoints build different subgraphs based on what they need.

## Dependency Tree (Full Catalog)

```
          config
         /   |   \
        ▼    ▼    ▼
       db  cache  metrics
        \   /       |
         ▼ ▼        ▼
         user     admin
           \       /
            ▼     ▼
            health
```

## Endpoints and Their Subgraphs

### GET /health
Minimal graph - just the health node:
```
health (no dependencies)
```

### GET /user/:id  
User service subgraph:
```
config → db → user
       ↘ cache ↗
```

### GET /admin/stats
Admin + metrics subgraph:
```
config → metrics → admin
```

## What It Demonstrates

- **`builder.BuildFor()`**: Building minimal subgraphs per request
- **Transitive dependency resolution**: Request `user`, get `db`, `cache`, and `config` automatically
- **Efficient execution**: Only runs nodes needed for each endpoint
- **Request isolation**: Each request gets a fresh engine instance

## Run It

```bash
cd examples/webserver
go run .
```

Then in another terminal:

```bash
# Health check (minimal graph)
curl http://localhost:8080/health

# User endpoint (builds user subgraph)
curl http://localhost:8080/user/123

# Admin stats (builds admin subgraph)  
curl http://localhost:8080/admin/stats
```

## Expected Output

Server logs show which nodes execute per request:

```
Server starting on :8080

=== GET /health ===
[health] Checking health...
[health] Done

=== GET /user/123 ===
[config] Loading...
[db] Connecting...  [cache] Connecting...
[user] Fetching user 123...

=== GET /admin/stats ===
[config] Loading...
[metrics] Collecting...
[admin] Building stats...
```

## Key Insight

This pattern is ideal for APIs where different endpoints have different dependencies. Instead of initializing everything at startup, each request builds exactly what it needs.

