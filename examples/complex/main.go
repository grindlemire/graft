package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grindlemire/graft"

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

func main() {
	engine := graft.Build()

	start := time.Now()
	if err := engine.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)

	fmt.Println("\n=== Results ===")
	for id, result := range engine.Results() {
		fmt.Printf("%s: %+v\n", id, result)
	}

	fmt.Printf("\n=== Timing ===\n")
	fmt.Printf("Total execution time: %v\n", elapsed)
	fmt.Println("9 nodes executed in 5 parallel levels")
}
