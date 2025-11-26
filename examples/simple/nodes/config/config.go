package config

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "config"

type Output struct {
	Host string
	Port int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{}, // root node
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	fmt.Println("[config] Loading configuration...")
	time.Sleep(100 * time.Millisecond) // simulate work

	fmt.Println("[config] Done")
	return Output{
		Host: "localhost",
		Port: 5432,
	}, nil
}
