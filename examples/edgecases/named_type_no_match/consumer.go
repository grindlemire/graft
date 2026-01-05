package named_type_no_match

import (
	"context"

	"github.com/grindlemire/graft"
)

type Server struct {
	Port int
}

// Consumer declares it depends on "port" (which outputs Port)
// but tries to use Dep[int], which won't resolve
// This should show "port" as UNUSED (declared but Dep call doesn't match)
func init() {
	graft.Register(graft.Node[Server]{
		ID:        "consumer",
		DependsOn: []graft.ID{"port"}, // Declaring port
		Run: func(ctx context.Context) (Server, error) {
			port, _ := graft.Dep[int](ctx) // Tries int, won't match Port
			return Server{Port: port}, nil
		},
	})
}
