package n7

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/long_chain/nodes/n6"
)

const ID graft.ID = "n7"

type Output struct {
	Value int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{n6.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	prev, _ := graft.Dep[n6.Output](ctx)
	return Output{Value: prev.Value + 1}, nil
}
