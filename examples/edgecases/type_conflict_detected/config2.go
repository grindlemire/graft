package type_conflict_detected

import (
	"context"

	"github.com/grindlemire/graft"
)

// Second node also outputs Config - this creates a type conflict!
// The analyzer should detect that both config1 and config2 produce the same type
func init() {
	graft.Register(graft.Node[Config]{
		ID: "config2",
		Run: func(ctx context.Context) (Config, error) {
			return Config{Port: 9090}, nil
		},
	})
}
