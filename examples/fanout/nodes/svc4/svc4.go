package svc4

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/fanout/nodes/config"
)

const ID = "svc4"

type Output struct {
	Name   string
	Result int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	_, err := graft.Dep[config.Output](ctx, config.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Println("[svc4] Processing...")
	time.Sleep(170 * time.Millisecond)

	fmt.Println("[svc4] Done")
	return Output{Name: "service-4", Result: 400}, nil
}

