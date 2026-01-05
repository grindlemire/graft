package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/edgecases/01_undeclared_single/nodes/app"
	_ "github.com/grindlemire/graft/examples/edgecases/01_undeclared_single/nodes/config"
)

func main() {
	// This example intentionally has a dependency error:
	// app uses config but doesn't declare it in DependsOn
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Executed %d nodes\n", len(results))
}
