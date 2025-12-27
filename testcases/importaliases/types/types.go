package types

import (
	"context"

	"github.com/grindlemire/graft"
)

const ConfigID graft.ID = "config-node"

type Config struct {
	Debug bool
}

func init() {
	graft.Register(graft.Node[Config]{
		ID: ConfigID,
		Run: func(ctx context.Context) (Config, error) {
			return Config{Debug: true}, nil
		},
	})
}
