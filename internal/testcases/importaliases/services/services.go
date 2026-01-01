package services

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "database-service"

type DBConnection struct {
	Host string
}

func init() {
	graft.Register(graft.Node[DBConnection]{
		ID: ID,
		Run: func(ctx context.Context) (DBConnection, error) {
			return DBConnection{Host: "localhost"}, nil
		},
	})
}
