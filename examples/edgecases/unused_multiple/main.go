package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/unused_multiple/nodes/app"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_multiple/nodes/cache"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_multiple/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_multiple/nodes/db"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
