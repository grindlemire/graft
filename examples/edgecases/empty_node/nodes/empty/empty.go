package empty

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "empty"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Minimal node with no logic
	return Output{}, nil
}
