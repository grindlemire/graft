package appA

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/19_orphan_nodes/nodes/configA"
)

const ID graft.ID = "appA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{configA.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, _ := graft.Dep[configA.Output](ctx)
	return Output{Value: "appA-" + cfg.Value}, nil
}
