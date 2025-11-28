package main

import (
	"context"
	"fmt"
	"log"

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

	configOutput, err := graft.Result[config.Output](results, config.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %+v\n", config.ID, configOutput)

	dbOutput, err := graft.Result[db.Output](results, db.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %+v\n", db.ID, dbOutput)
}
