package primitive_string

import (
	"context"

	"github.com/grindlemire/graft"
)

type App struct {
	Message string
}

// Consumer depends on the string output
func init() {
	graft.Register(graft.Node[App]{
		ID:        "app",
		DependsOn: []graft.ID{"str"},
		Run: func(ctx context.Context) (App, error) {
			str, _ := graft.Dep[string](ctx)
			return App{Message: str}, nil
		},
	})
}
