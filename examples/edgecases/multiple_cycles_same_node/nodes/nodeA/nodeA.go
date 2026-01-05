package nodeA

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "nodeA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{"hub"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "A"}, nil
}
