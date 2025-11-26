package config

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "config"

type Output struct {
	DBHost    string
	DBPort    int
	RedisHost string
	RedisPort int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	fmt.Println("[config] Loading configuration...")
	time.Sleep(100 * time.Millisecond)

	fmt.Println("[config] Done")
	return Output{
		DBHost:    "localhost",
		DBPort:    5432,
		RedisHost: "localhost",
		RedisPort: 6379,
	}, nil
}
