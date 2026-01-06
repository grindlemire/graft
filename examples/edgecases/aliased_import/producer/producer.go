package producer

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/edgecases/aliased_import/shared"
)

// Producer imports the shared package normally (no alias)
// and outputs shared.Config
func init() {
	graft.Register(graft.Node[shared.Config]{
		ID: "config",
		Run: func(ctx context.Context) (shared.Config, error) {
			return shared.Config{
				Port: 8080,
				Host: "localhost",
			}, nil
		},
	})
}
