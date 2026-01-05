package n3

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/long_chain/nodes/n2"
)

const ID graft.ID = "n3"

type Output struct {
	Value int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{n2.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	prev, _ := graft.Dep[n2.Output](ctx)
	return Output{Value: prev.Value + 1}, nil
}
