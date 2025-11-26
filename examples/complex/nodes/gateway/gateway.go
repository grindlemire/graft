package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/admin"
	"github.com/grindlemire/graft/examples/complex/nodes/user"
)

const ID graft.ID = "gateway"

type Output struct {
	Ready     bool
	Port      int
	Endpoints []string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{user.ID, admin.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	userOut, err := graft.Dep[user.Output](ctx, user.ID)
	if err != nil {
		return Output{}, err
	}

	adminOut, err := graft.Dep[admin.Output](ctx, admin.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Println("[gateway] Starting API gateway...")
	time.Sleep(50 * time.Millisecond)

	fmt.Println("[gateway] Done")
	return Output{
		Ready: true,
		Port:  8080,
		Endpoints: []string{
			userOut.Endpoint,
			adminOut.Endpoint,
		},
	}, nil
}
