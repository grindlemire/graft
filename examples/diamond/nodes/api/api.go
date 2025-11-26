package api

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/diamond/nodes/cache"
	"github.com/grindlemire/graft/examples/diamond/nodes/db"
)

const ID graft.ID = "api"

type Output struct {
	Ready   bool
	Version string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{db.ID, cache.ID}, // depends on both
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	dbOut, err := graft.Dep[db.Output](ctx, db.ID)
	if err != nil {
		return Output{}, err
	}

	cacheOut, err := graft.Dep[cache.Output](ctx, cache.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[api] Initializing API (db pool=%d, cache keys=%d)...\n",
		dbOut.PoolSize, cacheOut.MaxKeys)
	time.Sleep(100 * time.Millisecond)

	fmt.Println("[api] Done")
	return Output{
		Ready:   true,
		Version: "2.0.0",
	}, nil
}
