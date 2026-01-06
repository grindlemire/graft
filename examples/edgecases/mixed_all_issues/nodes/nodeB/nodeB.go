package nodeB

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/mixed_all_issues/nodes/cache"
)

const ID graft.ID = "nodeB"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// Cycle (B→A) + unused (cache)
		DependsOn: []graft.ID{"nodeA", cache.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// NOT using cache (unused)
	// Can't use nodeA due to Go import cycle (also creates unused for nodeA)
	return Output{Value: "B"}, nil
}
