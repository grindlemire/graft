package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/cache"
	"github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/09_mixed_undeclared_unused/nodes/db"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares config and db but uses cache
		// This creates both undeclared (cache) and unused (config, db)
		DependsOn: []graft.ID{config.ID, db.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Using cache without declaring it (undeclared)
	// NOT using config or db (unused)
	c, err := graft.Dep[cache.Output](ctx)
	if err != nil {
		return Output{}, err
	}
	return Output{Value: "app-" + c.Value}, nil
}
