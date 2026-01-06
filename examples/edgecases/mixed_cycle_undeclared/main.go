package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/mixed_cycle_undeclared/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/mixed_cycle_undeclared/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/mixed_cycle_undeclared/nodes/nodeB"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
