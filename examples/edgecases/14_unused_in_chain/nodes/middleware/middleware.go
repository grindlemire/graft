package middleware

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/14_unused_in_chain/nodes/cache"
	"github.com/grindlemire/graft/examples/edgecases/14_unused_in_chain/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/14_unused_in_chain/nodes/db"
)

const ID graft.ID = "middleware"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares all 3 but only uses config and cache
		DependsOn: []graft.ID{config.ID, db.ID, cache.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[config.Output](ctx)
	cch, _ := graft.Dep[cache.Output](ctx)
	// NOT using db (unused)
	return Output{Value: cfg.Value + "-" + cch.Value}, nil
}
