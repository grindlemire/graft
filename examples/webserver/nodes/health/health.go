package health

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID = "health"

type Output struct {
	Status    string
	Timestamp time.Time
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{}, // no dependencies - standalone health check
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	fmt.Println("[health] Checking health...")
	time.Sleep(10 * time.Millisecond)

	fmt.Println("[health] Done")
	return Output{
		Status:    "healthy",
		Timestamp: time.Now(),
	}, nil
}
