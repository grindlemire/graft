package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/20_conditional_dep_usage/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/20_conditional_dep_usage/nodes/feature"
)

const ID graft.ID = "app"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares config but uses feature in conditional
		// Type-aware SSA analysis should detect feature usage
		DependsOn: []graft.ID{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[config.Output](ctx)
	result := cfg.Value

	// Using feature in conditional - SSA should catch this
	f, _ := graft.Dep[feature.Output](ctx) // undeclared
	if f.Enabled {
		result += "-feature-enabled"
	}

	return Output{Value: result}, nil
}
