package consumer

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/testcases/typeresolution/pkg1"
	"github.com/grindlemire/graft/testcases/typeresolution/pkg2"
)

const ID graft.ID = "consumer"

type Result struct{}

func init() {
	graft.Register(graft.Node[Result]{
		ID:        ID,
		DependsOn: []graft.ID{pkg1.ID, pkg2.ID},
		Run: func(ctx context.Context) (Result, error) {
			// Both packages have Output type, but they should resolve to different nodes
			_, _ = graft.Dep[pkg1.Output](ctx)
			_, _ = graft.Dep[pkg2.Output](ctx)
			return Result{}, nil
		},
	})
}

