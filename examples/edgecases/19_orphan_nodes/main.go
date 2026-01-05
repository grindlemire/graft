package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/19_orphan_nodes/nodes/appA"
	_ "github.com/grindlemire/graft/examples/edgecases/19_orphan_nodes/nodes/appB"
	_ "github.com/grindlemire/graft/examples/edgecases/19_orphan_nodes/nodes/configA"
	_ "github.com/grindlemire/graft/examples/edgecases/19_orphan_nodes/nodes/configB"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
