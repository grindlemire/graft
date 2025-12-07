package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/fanout/nodes/aggregator"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/config"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/svc1"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/svc2"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/svc3"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/svc4"
	_ "github.com/grindlemire/graft/examples/fanout/nodes/svc5"
)

func main() {
	start := time.Now()
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)

	fmt.Println("\n=== Results ===")
	for id, result := range results {
		fmt.Printf("%s: %+v\n", id, result)
	}

	fmt.Printf("\n=== Timing ===\n")
	fmt.Printf("Total execution time: %v\n", elapsed)
	fmt.Println("All 5 services ran in parallel!")
	fmt.Println("Sequential would be: ~1020ms")

	fmt.Printf("\n=== Graph ===\n")
	err = graft.PrintGraph(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
