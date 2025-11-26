package db

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/diamond/nodes/config"
)

const ID = "db"

type Output struct {
	Connected bool
	PoolSize  int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, err := graft.Dep[config.Output](ctx, config.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[db] Connecting to database at %s:%d...\n", cfg.DBHost, cfg.DBPort)
	time.Sleep(200 * time.Millisecond) // simulate slower db connection

	fmt.Println("[db] Done")
	return Output{
		Connected: true,
		PoolSize:  10,
	}, nil
}

