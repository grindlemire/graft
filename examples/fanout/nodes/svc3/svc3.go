package svc3

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/fanout/nodes/config"
)

const ID = "svc3"

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

	fmt.Println("[svc3] Processing...")
	time.Sleep(150 * time.Millisecond)

	fmt.Println("[svc3] Done")
	return Output{Name: "service-3", Result: 300}, nil
}

