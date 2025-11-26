package user

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/auth"
	"github.com/grindlemire/graft/examples/complex/nodes/db"
)

const ID graft.ID = "user"

type Output struct {
	Ready    bool
	Endpoint string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{db.ID, auth.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	dbOut, err := graft.Dep[db.Output](ctx, db.ID)
	if err != nil {
		return Output{}, err
	}

	authOut, err := graft.Dep[auth.Output](ctx, auth.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[user] Starting user service (db pool=%d, auth=%v)...\n",
		dbOut.PoolSize, authOut.Ready)
	time.Sleep(70 * time.Millisecond)

	fmt.Println("[user] Done")
	return Output{
		Ready:    true,
		Endpoint: "/api/users",
	}, nil
}
