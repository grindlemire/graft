package config

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID = "config"

type Output struct {
	ServiceCount int
	Timeout      time.Duration
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	fmt.Println("[config] Loading configuration...")
	time.Sleep(50 * time.Millisecond)

	fmt.Println("[config] Done")
	return Output{
		ServiceCount: 5,
		Timeout:      30 * time.Second,
	}, nil
}

