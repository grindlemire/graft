package primitive_int

import (
	"context"

	"github.com/grindlemire/graft"
)

type Server struct {
	Port int
}

// Consumer depends on the int output
func init() {
	graft.Register(graft.Node[Server]{
		ID:        "server",
		DependsOn: []graft.ID{"port"},
		Run: func(ctx context.Context) (Server, error) {
			port, _ := graft.Dep[int](ctx)
			return Server{Port: port}, nil
		},
	})
}
