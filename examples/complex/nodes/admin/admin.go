package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/auth"
)

const ID graft.ID = "admin"

type Output struct {
	Ready    bool
	Endpoint string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{auth.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	authOut, err := graft.Dep[auth.Output](ctx, auth.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[admin] Starting admin service (auth=%v)...\n", authOut.Ready)
	time.Sleep(60 * time.Millisecond)

	fmt.Println("[admin] Done")
	return Output{
		Ready:    true,
		Endpoint: "/api/admin",
	}, nil
}
