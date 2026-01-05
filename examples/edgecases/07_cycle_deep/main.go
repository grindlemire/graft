package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeB"
	_ "github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeC"
	_ "github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeD"
	_ "github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeE"
)

func main() {
	// This example has a cycle deep in the chain: A → B → C → D → E → C
	// Only C, D, E participate in the cycle; A and B are clean
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
