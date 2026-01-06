package standalone

import (
	"context"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "standalone"

type Output struct {
	Value string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	// Independent node with complex logic but no dependencies
	result := "computed-value"
	for i := 0; i < 10; i++ {
		result += "-step"
	}
	return Output{Value: result}, nil
}
