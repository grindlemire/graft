package slice_type

import (
	"context"

	"github.com/grindlemire/graft"
)

type App struct {
	Tags []string
}

// Consumer depends on the []string output
func init() {
	graft.Register(graft.Node[App]{
		ID:        "app",
		DependsOn: []graft.ID{"tags"},
		Run: func(ctx context.Context) (App, error) {
			tags, _ := graft.Dep[[]string](ctx)
			return App{Tags: tags}, nil
		},
	})
}
