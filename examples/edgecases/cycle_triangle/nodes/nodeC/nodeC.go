package nodeC

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "nodeC"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL CYCLE: A → B → C → A
		DependsOn: []graft.ID{"nodeA"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "C"}, nil
}
