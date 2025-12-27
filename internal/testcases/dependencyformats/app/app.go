package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/internal/testcases/dependencyformats/dep1"
	"github.com/grindlemire/graft/internal/testcases/dependencyformats/dep2"
)

const ID graft.ID = "app"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{dep1.ID, dep2.ID},
		Run: func(ctx context.Context) (Output, error) {
			_, _ = graft.Dep[dep1.Output](ctx)
			_, _ = graft.Dep[dep2.Output](ctx)
			return Output{}, nil
		},
	})
}

