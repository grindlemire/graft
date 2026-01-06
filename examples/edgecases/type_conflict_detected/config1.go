package type_conflict_detected

import (
	"context"

	"github.com/grindlemire/graft"
)

// First node outputs Config
func init() {
	graft.Register(graft.Node[Config]{
		ID: "config1",
		Run: func(ctx context.Context) (Config, error) {
			return Config{Port: 8080}, nil
		},
	})
}
