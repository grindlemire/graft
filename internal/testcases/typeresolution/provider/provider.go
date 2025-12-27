package provider

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "connection-provider"

type Connection struct {
	Host string
}

func init() {
	// Node returns a pointer type
	graft.Register(graft.Node[*Connection]{
		ID: ID,
		Run: func(ctx context.Context) (*Connection, error) {
			return &Connection{Host: "localhost"}, nil
		},
	})
}

