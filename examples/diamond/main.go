package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	"github.com/grindlemire/graft/examples/diamond/nodes/api"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/api"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/cache"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/config"
	_ "github.com/grindlemire/graft/examples/diamond/nodes/db"
)

func main() {
	start := time.Now()
	// We can also use ExecuteFor to calculate the results for a specific
	// subgraph. In this case the subgraph is the entire graph.
	apiOutput, _, err := graft.ExecuteFor[api.Output](context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %+v\n", api.ID, apiOutput)
	elapsed := time.Since(start)

	fmt.Printf("\n=== Timing ===\n")
	fmt.Printf("Total execution time: %v\n", elapsed)
	fmt.Println("(db and cache ran in parallel, so total is less than sum of all nodes)")

	fmt.Printf("\n=== Graph ===\n")
	err = graft.PrintGraph(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
