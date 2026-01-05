package nodeA

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/10_mixed_cycle_undeclared/nodes/config"
)

const ID graft.ID = "nodeA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// INTENTIONAL ERROR: Cycle with nodeB + undeclared config
		// Declares nodeB (creates cycle) but also uses config without declaring
		DependsOn: []graft.ID{"nodeB"},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Using config without declaring it (undeclared)
	cfg, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}
	// Can't use nodeB due to Go import cycle
	return Output{Value: "A-" + cfg.Value}, nil
}
