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
		ID: ID,
		// INTENTIONAL CYCLE: A → B → C → A
		DependsOn: []graft.ID{"nodeB"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "A"}, nil
}
