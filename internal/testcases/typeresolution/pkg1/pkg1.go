package pkg1

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "pkg1-output"

// Output type with same name as pkg2.Output
type Output struct {
	Value int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID: ID,
		Run: func(ctx context.Context) (Output, error) {
			return Output{Value: 1}, nil
		},
	})
}

