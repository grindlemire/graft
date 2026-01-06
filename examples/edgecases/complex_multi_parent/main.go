package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grindlemire/graft"

	_ "github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/aggregator"
	_ "github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/config"
	_ "github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceA"
	_ "github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceB"
	_ "github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceC"
)

func main() {
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Executed %d nodes\n", len(results))
}
