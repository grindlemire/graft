package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/multiple_cycles_same_node/nodes/hub"
	_ "github.com/grindlemire/graft/examples/edgecases/multiple_cycles_same_node/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/multiple_cycles_same_node/nodes/nodeB"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
