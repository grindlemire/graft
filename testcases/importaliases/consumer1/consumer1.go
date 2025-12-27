package consumer1

import (
	"context"

	"github.com/grindlemire/graft"
	t "github.com/grindlemire/graft/testcases/importaliases/types"
)

const ID graft.ID = "consumer1"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{t.ConfigID},
		Run: func(ctx context.Context) (Output, error) {
			_, _ = graft.Dep[t.Config](ctx)
			return Output{}, nil
		},
	})
}
