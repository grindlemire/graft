package app

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/simple/nodes/db"
)

const ID = "app"

type Output struct {
	AppName string
	Version string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{db.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	dbOut, err := graft.Dep[db.Output](ctx, db.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[app] Starting application with db pool (size=%d)...\n", dbOut.PoolSize)
	time.Sleep(100 * time.Millisecond) // simulate startup

	fmt.Println("[app] Done")
	return Output{
		AppName: "MyApp",
		Version: "1.0.0",
	}, nil
}
