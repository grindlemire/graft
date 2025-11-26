package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/simple/nodes/app"
	_ "github.com/grindlemire/graft/examples/simple/nodes/config"
	_ "github.com/grindlemire/graft/examples/simple/nodes/db"
)

func main() {
	// Build engine from all registered nodes
	engine := graft.Build()

	// Run the graph
	if err := engine.Run(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Print results
	fmt.Println("\n=== Results ===")
	for id, result := range engine.Results() {
		fmt.Printf("%s: %+v\n", id, result)
	}
}
