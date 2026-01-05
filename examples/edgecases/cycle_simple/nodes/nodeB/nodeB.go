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
		// INTENTIONAL CYCLE: nodeA → nodeB → nodeA
		// Note: Using string literal to avoid Go import cycle
		DependsOn: []graft.ID{"nodeA"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Cannot call graft.Dep[nodeA.Output] due to Go import cycle
	// This creates an "unused" dependency, but the cycle is still detected
	return Output{Value: "B"}, nil
}
