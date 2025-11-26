package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/webserver/nodes/config"
)

const ID = "cache"

type Output struct {
	Connected bool
	TTL       time.Duration
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

	fmt.Printf("[cache] Connecting to Redis at %s:%d...\n", cfg.CacheHost, cfg.CachePort)
	time.Sleep(40 * time.Millisecond)

	fmt.Println("[cache] Done")
	return Output{
		Connected: true,
		TTL:       5 * time.Minute,
	}, nil
}

