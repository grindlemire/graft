package env

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID = "env"

type Output struct {
	Environment string
	Debug       bool
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	fmt.Println("[env] Loading environment...")
	time.Sleep(50 * time.Millisecond)

	fmt.Println("[env] Done")
	return Output{
		Environment: "production",
		Debug:       false,
	}, nil
}

