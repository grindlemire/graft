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
		// INTENTIONAL SELF-CYCLE: nodeA depends on itself
		DependsOn: []graft.ID{ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Trying to get own output (self-cycle)
	a, err := graft.Dep[Output](ctx)
	if err != nil {
		return Output{}, err
	}
	return Output{Value: "A-" + a.Value}, nil
}
