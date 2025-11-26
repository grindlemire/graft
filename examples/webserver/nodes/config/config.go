package config

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID = "config"

type Output struct {
	DBHost      string
	DBPort      int
	CacheHost   string
	CachePort   int
	MetricsHost string
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
	time.Sleep(30 * time.Millisecond)

	fmt.Println("[config] Done")
	return Output{
		DBHost:      "localhost",
		DBPort:      5432,
		CacheHost:   "localhost",
		CachePort:   6379,
		MetricsHost: "localhost:9090",
	}, nil
}

