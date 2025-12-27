package multiapp

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/testcases/iddeclarations/shared"
)

const ID graft.ID = "app"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{shared.CacheID, shared.DatabaseID, shared.LoggerID},
		Run: func(ctx context.Context) (Output, error) {
			_, _ = graft.Dep[shared.Cache](ctx)
			_, _ = graft.Dep[shared.Database](ctx)
			_, _ = graft.Dep[shared.Logger](ctx)
			return Output{}, nil
		},
	})
}

