package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/cache"
	_ "github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/db"
	_ "github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/nodeA"
	_ "github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/nodeB"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
