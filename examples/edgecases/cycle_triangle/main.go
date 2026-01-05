package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/cycle_triangle/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/cycle_triangle/nodes/nodeB"
	_ "github.com/grindlemire/graft/examples/edgecases/cycle_triangle/nodes/nodeC"
)

func main() {
	// This example has a 3-node cycle: A → B → C → A
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
