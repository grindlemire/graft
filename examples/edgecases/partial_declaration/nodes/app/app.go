package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/partial_declaration/nodes/cache"
	"github.com/grindlemire/graft/examples/edgecases/partial_declaration/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/partial_declaration/nodes/db"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares config and db, but uses all 3
		// cache is undeclared
		DependsOn: []graft.ID{config.ID, db.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[config.Output](ctx)
	d, _ := graft.Dep[db.Output](ctx)
	cch, _ := graft.Dep[cache.Output](ctx) // undeclared
	return Output{Value: cfg.Value + "-" + d.Value + "-" + cch.Value}, nil
}
