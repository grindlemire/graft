package svc5

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/fanout/nodes/config"
)

const ID graft.ID = "svc5"

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
	_, err := graft.Dep[config.Output](ctx, config.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Println("[svc5] Processing...")
	time.Sleep(180 * time.Millisecond)

	fmt.Println("[svc5] Done")
	return Output{Name: "service-5", Result: 500}, nil
}
