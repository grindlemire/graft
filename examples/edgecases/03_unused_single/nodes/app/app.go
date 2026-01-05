package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/03_unused_single/nodes/config"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares config but never uses it
		DependsOn: []graft.ID{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// NOT using the config dependency
	return Output{Value: "app"}, nil
}
