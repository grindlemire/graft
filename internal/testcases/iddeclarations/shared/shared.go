package shared

import (
	"context"

	"github.com/grindlemire/graft"
)

// Mix of const and var declarations
const (
	CacheID    graft.ID = "cache-node"
	DatabaseID graft.ID = "database-node"
)

var LoggerID graft.ID = "logger-node"

type Cache struct{}
type Database struct{}
type Logger struct{}

func init() {
	graft.Register(graft.Node[Cache]{
		ID: CacheID,
		Run: func(ctx context.Context) (Cache, error) {
			return Cache{}, nil
		},
	})

	graft.Register(graft.Node[Database]{
		ID: DatabaseID,
		Run: func(ctx context.Context) (Database, error) {
			return Database{}, nil
		},
	})

	graft.Register(graft.Node[Logger]{
		ID: LoggerID,
		Run: func(ctx context.Context) (Logger, error) {
			return Logger{}, nil
		},
	})
}

