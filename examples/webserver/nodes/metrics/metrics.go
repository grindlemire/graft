package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/webserver/nodes/config"
)

const ID = "metrics"

type Output struct {
	Connected    bool
	RequestCount int
	ErrorCount   int
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

	fmt.Printf("[metrics] Connecting to %s...\n", cfg.MetricsHost)
	time.Sleep(35 * time.Millisecond)

	fmt.Println("[metrics] Done")
	return Output{
		Connected:    true,
		RequestCount: 15234,
		ErrorCount:   12,
	}, nil
}

