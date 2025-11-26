package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/webserver/nodes/metrics"
)

const ID graft.ID = "admin"

type Output struct {
	TotalRequests int
	TotalErrors   int
	ErrorRate     float64
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{metrics.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	met, err := graft.Dep[metrics.Output](ctx, metrics.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[admin] Building stats from metrics (requests=%d)...\n", met.RequestCount)
	time.Sleep(20 * time.Millisecond)

	errorRate := float64(met.ErrorCount) / float64(met.RequestCount) * 100

	fmt.Println("[admin] Done")
	return Output{
		TotalRequests: met.RequestCount,
		TotalErrors:   met.ErrorCount,
		ErrorRate:     errorRate,
	}, nil
}
