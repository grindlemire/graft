package ports

import (
	"context"

	"github.com/grindlemire/graft"
)

const (
	ExecutorID graft.ID = "executor"
	LoggerID   graft.ID = "logger"
)

// Output types
type Executor interface {
	Execute() error
}

type Logger interface {
	Log(msg string)
}

func init() {
	// Two different nodes in the same package, each outputting a different type
	graft.Register(graft.Node[Executor]{
		ID: ExecutorID,
		Run: func(ctx context.Context) (Executor, error) {
			return nil, nil
		},
	})

	graft.Register(graft.Node[Logger]{
		ID: LoggerID,
		Run: func(ctx context.Context) (Logger, error) {
			return nil, nil
		},
	})
}
