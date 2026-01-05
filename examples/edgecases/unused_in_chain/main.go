package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/unused_in_chain/nodes/cache"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_in_chain/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_in_chain/nodes/db"
	_ "github.com/grindlemire/graft/examples/edgecases/unused_in_chain/nodes/middleware"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
