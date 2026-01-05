package hub

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "hub"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL: hub participates in 2 cycles: hub↔nodeA and hub↔nodeB
		DependsOn: []graft.ID{"nodeA", "nodeB"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "hub"}, nil
}
