package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/cycle_simple/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/cycle_simple/nodes/nodeB"
)

func main() {
	// This example has a simple 2-node cycle: A → B → A
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
