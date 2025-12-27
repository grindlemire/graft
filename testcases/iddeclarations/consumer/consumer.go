package consumer

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/testcases/iddeclarations/producer"
)

const ID graft.ID = "consumer"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{producer.MyNodeIdentifier},
		Run: func(ctx context.Context) (Output, error) {
			_, _ = graft.Dep[producer.Data](ctx)
			return Output{}, nil
		},
	})
}

