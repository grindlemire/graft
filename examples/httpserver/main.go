// Package main demonstrates using graft with node-level caching for HTTP servers.
//
// Nodes declare themselves as cacheable via Cacheable: true. Only cacheable
// nodes are stored in the cache - request-scoped nodes run fresh every time.
//
// Run this example and make a few requests:
//
//	curl http://localhost:8080/user/123
//	curl http://localhost:8080/user/456
//	curl http://localhost:8080/admin
//	curl http://localhost:8080/stats
//
// You'll see that config and db (Cacheable: true) execute only ONCE,
// while request-scoped nodes execute on every request.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/grindlemire/graft"

	// Import all nodes for registration

	"github.com/grindlemire/graft/examples/httpserver/nodes/admin"
	"github.com/grindlemire/graft/examples/httpserver/nodes/config"
	"github.com/grindlemire/graft/examples/httpserver/nodes/db"
	"github.com/grindlemire/graft/examples/httpserver/nodes/requestlogger"
	"github.com/grindlemire/graft/examples/httpserver/nodes/user"
)

var requestCounter atomic.Int32

func main() {
	fmt.Println("=== HTTP Server with Graft Caching ===")
	fmt.Println()
	fmt.Println("BEHAVIOR:")
	fmt.Println("  - Config and DB execute ONCE and are cached")
	fmt.Println("  - Request-scoped nodes (request_logger, handlers) execute every request")
	fmt.Println("  - Watch /stats to confirm config/db stay at 1 execution")
	fmt.Println()

	fmt.Printf("\n=== Graph ===\n")
	err := graft.PrintGraph(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\n")

	mux := http.NewServeMux()

	// User endpoint - depends on db (which depends on config) + request_logger
	mux.HandleFunc("/user/", handleUser())

	// Admin endpoint - depends on config + request_logger
	mux.HandleFunc("/admin", handleAdmin())

	// Stats endpoint - shows execution counts
	mux.HandleFunc("/stats", handleStats())

	fmt.Println("Endpoints:")
	fmt.Println("  GET /user/:id  - fetch user (config/db cached, handler runs fresh)")
	fmt.Println("  GET /admin     - admin info (config cached, handler runs fresh)")
	fmt.Println("  GET /stats     - show execution counts")
	fmt.Println()
	fmt.Println("Server starting on :8080")
	fmt.Println()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqNum := requestCounter.Add(1)
		reqID := fmt.Sprintf("req-%d", reqNum)

		userID := strings.TrimPrefix(r.URL.Path, "/user/")
		if userID == "" {
			http.Error(w, "missing user ID", http.StatusBadRequest)
			return
		}

		fmt.Printf("\n=== Request %s: GET /user/%s ===\n", reqID, userID)

		// Set up request context
		ctx := r.Context()
		ctx = requestlogger.SetRequestID(ctx, reqID)
		ctx = user.SetUserID(ctx, userID)

		// Execute graph with caching:
		// - config, db: Cacheable: true → served from cache
		// - request_logger, user: not cacheable → run fresh
		user, results, err := graft.ExecuteFor[user.Output](ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config, err := graft.Result[config.Output](results)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		db, err := graft.Result[db.Output](results)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("=== Request %s complete ===\n", reqID)

		respondJSON(w, map[string]any{
			"request_id":      reqID,
			"user":            user,
			"config_exec_num": config.ExecutionNum,
			"db_exec_num":     db.ExecutionNum,
		})
	}
}

func handleAdmin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqNum := requestCounter.Add(1)
		reqID := fmt.Sprintf("req-%d", reqNum)

		fmt.Printf("\n=== Request %s: GET /admin ===\n", reqID)

		ctx := requestlogger.SetRequestID(r.Context(), reqID)

		// Execute graph with caching:
		// - config: Cacheable: true → served from cache
		// - request_logger, admin: not cacheable → run fresh
		admin, results, err := graft.ExecuteFor[admin.Output](ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config, err := graft.Result[config.Output](results)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("=== Request %s complete ===\n", reqID)

		respondJSON(w, map[string]any{
			"request_id":      reqID,
			"admin":           admin,
			"config_exec_num": config.ExecutionNum,
		})
	}
}

func handleStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]any{
			"total_requests":    requestCounter.Load(),
			"config_executions": config.ExecutionCount(),
			"db_executions":     db.ExecutionCount(),
			"note":              "config and db should stay at 1 (cached), even after many requests",
		})
	}
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}
