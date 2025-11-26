package config

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/env"
)

const ID graft.ID = "config"

type Output struct {
	DBHost string
	DBPort int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{env.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	envOut, err := graft.Dep[env.Output](ctx, env.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[config] Loading config for %s...\n", envOut.Environment)
	time.Sleep(80 * time.Millisecond)

	fmt.Println("[config] Done")
	return Output{
		DBHost: "db.example.com",
		DBPort: 5432,
	}, nil
}
