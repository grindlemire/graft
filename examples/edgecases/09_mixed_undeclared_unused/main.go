package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/app"
	_ "github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/cache"
	_ "github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/db"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
