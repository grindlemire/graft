package pointer_mismatch

import (
	"context"

	"github.com/grindlemire/graft"
)

type Config struct {
	Port int
}

// Producer outputs *Config (pointer type)
func init() {
	graft.Register(graft.Node[*Config]{
		ID: "config",
		Run: func(ctx context.Context) (*Config, error) {
			return &Config{Port: 8080}, nil
		},
	})
}
