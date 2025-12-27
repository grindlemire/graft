package consumer2

import (
	"context"

	"github.com/grindlemire/graft"
	apptypes "github.com/grindlemire/graft/testcases/importaliases/types"
)

const ID graft.ID = "consumer2"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{apptypes.ConfigID},
		Run: func(ctx context.Context) (Output, error) {
			_, _ = graft.Dep[apptypes.Config](ctx)
			return Output{}, nil
		},
	})
}
