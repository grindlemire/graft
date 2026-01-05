package nodeB

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "nodeB"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL CYCLE: nodeB → nodeA
		DependsOn: []graft.ID{"nodeA"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Can't use nodeA due to Go import cycle
	return Output{Value: "B"}, nil
}
