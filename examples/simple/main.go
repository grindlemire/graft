package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/grindlemire/graft"

	// Import nodes for side-effect registration
	_ "github.com/grindlemire/graft/examples/simple/nodes/app"
	"github.com/grindlemire/graft/examples/simple/nodes/config"
	"github.com/grindlemire/graft/examples/simple/nodes/db"
)

func main() {
	// Execute all registered nodes
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	configOutput, err := graft.Result[config.Output](results)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %+v\n", config.ID, configOutput)

	dbOutput, err := graft.Result[db.Output](results)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %+v\n", db.ID, dbOutput)

	fmt.Printf("\n=== Graph ===\n")
	err = graft.PrintGraph(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\n")
}
