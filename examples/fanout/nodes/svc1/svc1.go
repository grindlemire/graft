package svc1

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/fanout/nodes/config"
)

const ID graft.ID = "svc1"

type Output struct {
	Name   string
	Result int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{config.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	_, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}

	fmt.Println("[svc1] Processing...")
	time.Sleep(200 * time.Millisecond)

	fmt.Println("[svc1] Done")
	return Output{Name: "service-1", Result: 100}, nil
}
