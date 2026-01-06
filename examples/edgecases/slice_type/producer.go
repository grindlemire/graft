package slice_type

import (
	"context"

	"github.com/grindlemire/graft"
)

// Producer outputs a slice type
func init() {
	graft.Register(graft.Node[[]string]{
		ID: "tags",
		Run: func(ctx context.Context) ([]string, error) {
			return []string{"a", "b", "c"}, nil
		},
	})
}
