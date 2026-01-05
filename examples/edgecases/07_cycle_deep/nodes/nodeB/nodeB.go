package nodeB

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/07_cycle_deep/nodes/nodeC"
)

const ID graft.ID = "nodeB"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{nodeC.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	c, err := graft.Dep[nodeC.Output](ctx)
	if err != nil {
		return Output{}, err
	}
	return Output{Value: "B-" + c.Value}, nil
}
