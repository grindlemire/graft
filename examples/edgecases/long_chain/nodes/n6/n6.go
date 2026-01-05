package n6

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/long_chain/nodes/n5"
)

const ID graft.ID = "n6"

type Output struct {
	Value int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{n5.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	prev, _ := graft.Dep[n5.Output](ctx)
	return Output{Value: prev.Value + 1}, nil
}
