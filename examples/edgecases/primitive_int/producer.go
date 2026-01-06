package primitive_int

import (
	"context"

	"github.com/grindlemire/graft"
)

// Producer outputs a primitive int type
func init() {
	graft.Register(graft.Node[int]{
		ID: "port",
		Run: func(ctx context.Context) (int, error) {
			return 8080, nil
		},
	})
}
