package producer

import (
	"context"

	"github.com/grindlemire/graft"
)

// Using a var instead of const for the ID - the analyzer should still resolve it
var MyNodeIdentifier graft.ID = "data-producer"

type Data struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Data]{
		ID: MyNodeIdentifier,
		Run: func(ctx context.Context) (Data, error) {
			return Data{Value: "test"}, nil
		},
	})
}
