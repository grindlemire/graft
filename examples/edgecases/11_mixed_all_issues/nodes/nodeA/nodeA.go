package nodeA

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/config"
	"github.com/grindlemire/graft/examples/edgecases/11_mixed_all_issues/nodes/db"
)

const ID graft.ID = "nodeA"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		// ALL ISSUES: cycle (A↔B) + undeclared (config) + unused (db)
		DependsOn: []graft.ID{"nodeB", db.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Using config without declaring it (undeclared)
	// NOT using db (unused)
	// Can't use nodeB due to Go import cycle (also creates unused for nodeB)
	cfg, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}
	return Output{Value: "A-" + cfg.Value}, nil
}
