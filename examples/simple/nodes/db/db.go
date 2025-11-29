package db

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/simple/nodes/config"
)

const ID graft.ID = "db"

type Output struct {
	Connected bool
	PoolSize  int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[db] Connecting to database at %s:%d...\n", cfg.Host, cfg.Port)
	time.Sleep(150 * time.Millisecond) // simulate connection

	fmt.Println("[db] Done")
	return Output{
		Connected: true,
		PoolSize:  10,
	}, nil
}
