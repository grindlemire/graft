package appB

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/orphan_nodes/nodes/configB"
)

const ID graft.ID = "appB"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{configB.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[configB.Output](ctx)
	return Output{Value: "appB-" + cfg.Value}, nil
}
