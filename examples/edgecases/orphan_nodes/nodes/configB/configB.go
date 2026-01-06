package configB

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "configB"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "configB"}, nil
}
