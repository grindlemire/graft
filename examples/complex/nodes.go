package main

// if you have many nodes, you can import them all at once with a blank import
// in a dedicated nodes.go file to keep your main.go clean
import (
	// Import all nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/complex/nodes/admin"
	_ "github.com/grindlemire/graft/examples/complex/nodes/auth"
	_ "github.com/grindlemire/graft/examples/complex/nodes/config"
	_ "github.com/grindlemire/graft/examples/complex/nodes/db"
	_ "github.com/grindlemire/graft/examples/complex/nodes/env"
	_ "github.com/grindlemire/graft/examples/complex/nodes/gateway"
	_ "github.com/grindlemire/graft/examples/complex/nodes/logger"
	_ "github.com/grindlemire/graft/examples/complex/nodes/secrets"
	_ "github.com/grindlemire/graft/examples/complex/nodes/user"
)
