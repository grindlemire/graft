package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/diamond/nodes/api"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/cache"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/config"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/db"
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
	fmt.Println("(db and cache ran in parallel, so total is less than sum of all nodes)")
}

