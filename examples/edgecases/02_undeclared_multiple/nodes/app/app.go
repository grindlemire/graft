package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/02_undeclared_multiple/nodes/cache"
	"github.com/grindlemire/graft/examples/edgecases/02_undeclared_multiple/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/02_undeclared_multiple/nodes/db"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Uses 3 deps but declares none
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Using multiple dependencies without declaring them
	cfg, _ := graft.Dep[config.Output](ctx)
	database, _ := graft.Dep[db.Output](ctx)
	cch, _ := graft.Dep[cache.Output](ctx)

	return Output{Value: cfg.Value + "-" + database.Value + "-" + cch.Value}, nil
}
