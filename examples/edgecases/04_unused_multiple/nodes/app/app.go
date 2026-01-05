package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/04_unused_multiple/nodes/cache"
	"github.com/grindlemire/graft/examples/edgecases/04_unused_multiple/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/04_unused_multiple/nodes/db"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares 3 deps but uses none
		DependsOn: []graft.ID{config.ID, db.ID, cache.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// NOT using any of the declared dependencies
	return Output{Value: "app"}, nil
}
