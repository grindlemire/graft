package feature

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "feature"

type Output struct {
	Enabled bool
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Enabled: true}, nil
}
