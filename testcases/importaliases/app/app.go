package app

import (
	"context"

	"github.com/grindlemire/graft"
	svc "github.com/grindlemire/graft/testcases/importaliases/services"
)

const ID graft.ID = "app"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{svc.ID},
		Run: func(ctx context.Context) (Output, error) {
			// Using aliased import: svc instead of services
			_, _ = graft.Dep[svc.DBConnection](ctx)
			return Output{}, nil
		},
	})
}
