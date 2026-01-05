package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n1"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n10"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n2"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n3"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n4"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n5"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n6"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n7"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n8"
	_ "github.com/grindlemire/graft/examples/edgecases/17_long_chain/nodes/n9"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
