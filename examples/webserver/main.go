package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/grindlemire/graft"

	// Import all nodes for side-effect registration into the catalog
	_ "github.com/grindlemire/graft/examples/webserver/nodes/admin"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/cache"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/config"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/db"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/health"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/metrics"
	_ "github.com/grindlemire/graft/examples/webserver/nodes/user"

	"github.com/grindlemire/graft/examples/webserver/nodes/admin"
	"github.com/grindlemire/graft/examples/webserver/nodes/health"
	"github.com/grindlemire/graft/examples/webserver/nodes/user"
)

func main() {
	// Create a builder from the node catalog
	builder := graft.NewBuilder(graft.Registry())

	mux := http.NewServeMux()

	// Health endpoint - minimal subgraph (just health node, no deps)
	mux.HandleFunc("/health", handleHealth(builder))

	// User endpoint - builds user subgraph (config, db, cache, user)
	mux.HandleFunc("/user/", handleUser(builder))

	// Admin stats endpoint - builds admin subgraph (config, metrics, admin)
	mux.HandleFunc("/admin/stats", handleAdminStats(builder))

	fmt.Println("Server starting on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /health       - minimal health check")
	fmt.Println("  GET /user/:id     - user service (builds db+cache subgraph)")
	fmt.Println("  GET /admin/stats  - admin stats (builds metrics subgraph)")
	fmt.Println()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// handleHealth runs the minimal health check - just the health node
func handleHealth(builder *graft.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("\n=== GET /health ===")

		// Build subgraph with only health node (no dependencies)
		engine, err := builder.BuildFor(health.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := engine.Run(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(w, engine.Results())
	}
}

// handleUser builds the user subgraph: config → db, cache → user
func handleUser(builder *graft.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from path
		userID := strings.TrimPrefix(r.URL.Path, "/user/")
		if userID == "" {
			http.Error(w, "missing user ID", http.StatusBadRequest)
			return
		}

		fmt.Printf("\n=== GET /user/%s ===\n", userID)

		// Build subgraph for user node (automatically includes db, cache, config)
		engine, err := builder.BuildFor(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add user ID to context for the user node to access
		ctx := user.SetUserID(r.Context(), userID)

		if err := engine.Run(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(w, engine.Results())
	}
}

// handleAdminStats builds the admin subgraph: config → metrics → admin
func handleAdminStats(builder *graft.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("\n=== GET /admin/stats ===")

		// Build subgraph for admin node (automatically includes metrics, config)
		engine, err := builder.BuildFor(admin.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := engine.Run(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(w, engine.Results())
	}
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}
