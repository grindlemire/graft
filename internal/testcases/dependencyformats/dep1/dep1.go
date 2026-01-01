package dep1

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "dep1-node"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		Run: func(ctx context.Context) (Output, error) {
			return Output{}, nil
		},
	})
}

