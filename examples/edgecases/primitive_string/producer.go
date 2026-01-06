package primitive_string

import (
	"context"

	"github.com/grindlemire/graft"
)

// Producer outputs a primitive string type
func init() {
	graft.Register(graft.Node[string]{
		ID: "str",
		Run: func(ctx context.Context) (string, error) {
			return "hello", nil
		},
	})
}
