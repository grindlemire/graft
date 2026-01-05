package pointer_mismatch

import (
	"context"

	"github.com/grindlemire/graft"
)

type App struct {
	Config Config
}

// Consumer declares it depends on "config" (which outputs *Config)
// but tries to use Dep[Config] (non-pointer), which won't resolve
// This should show "config" as UNUSED (declared but Dep call doesn't match)
func init() {
	graft.Register(graft.Node[App]{
		ID:        "consumer",
		DependsOn: []graft.ID{"config"}, // Declaring config
		Run: func(ctx context.Context) (App, error) {
			cfg, _ := graft.Dep[Config](ctx) // Tries non-pointer, won't match *Config
			return App{Config: cfg}, nil
		},
	})
}
