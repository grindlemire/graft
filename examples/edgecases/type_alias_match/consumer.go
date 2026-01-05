package type_alias_match

import (
	"context"

	"github.com/grindlemire/graft"
)

type App struct {
	UserID UserID
}

// Consumer depends on UserID
// Type alias should resolve correctly, so Dep[UserID] should work
func init() {
	graft.Register(graft.Node[App]{
		ID:        "app",
		DependsOn: []graft.ID{"uid"},
		Run: func(ctx context.Context) (App, error) {
			uid, _ := graft.Dep[UserID](ctx)
			return App{UserID: uid}, nil
		},
	})
}
