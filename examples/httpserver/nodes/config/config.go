package config

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "config"

// executionCount tracks how many times this node has been executed
var executionCount atomic.Int32

type Output struct {
	Host         string
	Port         int
	DatabaseURL  string
	StartupTime  time.Time
	ExecutionNum int32 // Which execution this was
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Cacheable: true, // Startup node - execute once and cache
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	num := executionCount.Add(1)
	fmt.Printf("[config] Executing config node (execution #%d)\n", num)

	// Simulate loading config from file/env
	time.Sleep(50 * time.Millisecond)

	return Output{
		Host:         "localhost",
		Port:         8080,
		DatabaseURL:  "postgres://localhost:5432/myapp",
		StartupTime:  time.Now(),
		ExecutionNum: num,
	}, nil
}

// ExecutionCount returns how many times this node has been executed
func ExecutionCount() int32 {
	return executionCount.Load()
}

