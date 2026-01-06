package consumer

import (
	"context"

	"github.com/grindlemire/graft"
	// Import with alias - this tests whether the analyzer correctly resolves
	// types regardless of import aliases
	sharedcfg "github.com/grindlemire/graft/examples/edgecases/aliased_import/shared"
)

type App struct {
	Config sharedcfg.Config
}

// Consumer imports the shared package with an alias "sharedcfg"
// and uses Dep[sharedcfg.Config] to access the config
// This should work because the underlying type is the same
func init() {
	graft.Register(graft.Node[App]{
		ID:        "app",
		DependsOn: []graft.ID{"config"},
		Run: func(ctx context.Context) (App, error) {
			// Using the aliased import name
			cfg, _ := graft.Dep[sharedcfg.Config](ctx)
			return App{Config: cfg}, nil
		},
	})
}
