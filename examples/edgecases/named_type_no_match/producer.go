package named_type_no_match

import (
	"context"

	"github.com/grindlemire/graft"
)

// Port is a named type based on int (not a type alias)
type Port int

// Producer outputs Port (a named type distinct from int)
func init() {
	graft.Register(graft.Node[Port]{
		ID: "port",
		Run: func(ctx context.Context) (Port, error) {
			return Port(8080), nil
		},
	})
}
