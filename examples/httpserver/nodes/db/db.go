package db

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/httpserver/nodes/config"
)

const ID graft.ID = "db"

var executionCount atomic.Int32

// Output represents a database connection pool
// In real code, this would be *sql.DB or similar
type Output struct {
	ConnectionString string
	PoolSize         int
	Connected        bool
	ExecutionNum     int32
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{config.ID},
		Cacheable: true, // Startup node - execute once and cache
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	num := executionCount.Add(1)

	cfg, err := graft.Dep[config.Output](ctx, config.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[db] Executing db node (execution #%d) - connecting to %s\n", num, cfg.DatabaseURL)

	// Simulate establishing connection pool - this is expensive!
	time.Sleep(100 * time.Millisecond)

	return Output{
		ConnectionString: cfg.DatabaseURL,
		PoolSize:         10,
		Connected:        true,
		ExecutionNum:     num,
	}, nil
}

func ExecutionCount() int32 {
	return executionCount.Load()
}
