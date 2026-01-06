package nodeA

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/cycle_deep/nodes/nodeB"
)

const ID graft.ID = "nodeA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{nodeB.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	b, err := graft.Dep[nodeB.Output](ctx)
	if err != nil {
		return Output{}, err
	}
	return Output{Value: "A-" + b.Value}, nil
}
