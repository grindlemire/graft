package nodeE

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "nodeE"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// Part of cycle: C → D → E → C
		DependsOn: []graft.ID{"nodeC"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	return Output{Value: "E"}, nil
}
