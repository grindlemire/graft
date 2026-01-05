package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/01_undeclared_single/nodes/config"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: DependsOn is empty but we use config
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Using config dependency without declaring it
	cfg, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}

	return Output{Value: "app-" + cfg.Value}, nil
}
