package aggregator

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceA"
	"github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceB"
	"github.com/grindlemire/graft/examples/edgecases/complex_multi_parent/nodes/serviceC"
)

const ID graft.ID = "aggregator"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Declares all 3 services but only uses A and B
		DependsOn: []graft.ID{serviceA.ID, serviceB.ID, serviceC.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	a, _ := graft.Dep[serviceA.Output](ctx)
	b, _ := graft.Dep[serviceB.Output](ctx)
	// NOT using serviceC (unused)
	return Output{Value: a.Value + "-" + b.Value}, nil
}
