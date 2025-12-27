package pointerconsumer

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/internal/testcases/typeresolution/provider"
)

const ID graft.ID = "pointer-consumer"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{provider.ID},
		Run: func(ctx context.Context) (Output, error) {
			// Using pointer type as dependency
			_, _ = graft.Dep[*provider.Connection](ctx)
			return Output{}, nil
		},
	})
}
