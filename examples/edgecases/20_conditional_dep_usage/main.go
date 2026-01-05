package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/20_conditional_dep_usage/nodes/app"
	_ "github.com/grindlemire/graft/examples/edgecases/20_conditional_dep_usage/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/20_conditional_dep_usage/nodes/feature"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
