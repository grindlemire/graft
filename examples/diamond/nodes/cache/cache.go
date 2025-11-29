package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/diamond/nodes/config"
)

const ID graft.ID = "cache"

type Output struct {
	Connected bool
	MaxKeys   int
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

	fmt.Printf("[cache] Connecting to Redis at %s:%d...\n", cfg.RedisHost, cfg.RedisPort)
	time.Sleep(150 * time.Millisecond) // simulate cache connection

	fmt.Println("[cache] Done")
	return Output{
		Connected: true,
		MaxKeys:   10000,
	}, nil
}
