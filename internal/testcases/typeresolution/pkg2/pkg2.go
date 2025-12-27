package pkg2

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "pkg2-output"

// Output type with same name as pkg1.Output
type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		Run: func(ctx context.Context) (Output, error) {
			return Output{Value: "two"}, nil
		},
	})
}

