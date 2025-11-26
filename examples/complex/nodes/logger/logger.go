package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/env"
)

const ID = "logger"

type Output struct {
	Level  string
	Format string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{env.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	envOut, err := graft.Dep[env.Output](ctx, env.ID)
	if err != nil {
		return Output{}, err
	}

	level := "info"
	if envOut.Debug {
		level = "debug"
	}

	fmt.Println("[logger] Initializing logger...")
	time.Sleep(60 * time.Millisecond)

	fmt.Println("[logger] Done")
	return Output{
		Level:  level,
		Format: "json",
	}, nil
}

