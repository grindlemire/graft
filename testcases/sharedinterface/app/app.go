package app

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/testcases/sharedinterface/ports"
)

const ID graft.ID = "app"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{ports.ExecutorID, ports.LoggerID},
		Run: func(ctx context.Context) (Output, error) {
			// Both types come from the same "ports" package, but they should
			// resolve to different node IDs: "executor" and "logger"
			// NOT to "ports" (the package name)
			_, _ = graft.Dep[ports.Executor](ctx)
			_, _ = graft.Dep[ports.Logger](ctx)
			return Output{}, nil
		},
	})
}
