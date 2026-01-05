package n1

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "n1"

type Output struct {
	Value int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: 1}, nil
}
