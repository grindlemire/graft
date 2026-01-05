package serviceA

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/18_complex_multi_parent/nodes/config"
)

const ID graft.ID = "serviceA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[config.Output](ctx)
	return Output{Value: "A-" + cfg.Value}, nil
}
