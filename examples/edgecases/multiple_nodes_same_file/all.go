package multiple_nodes_same_file

import (
	"context"

	"github.com/grindlemire/graft"
)

// All nodes in a single file

type A struct {
	Value int
}

type B struct {
	AValue int
}

type C struct {
	BValue int
}

// Node A - no dependencies
func init() {
	graft.Register(graft.Node[A]{
		ID: "nodeA",
		Run: func(ctx context.Context) (A, error) {
			return A{Value: 1}, nil
		},
	})
}

// Node B - depends on A
func init() {
	graft.Register(graft.Node[B]{
		ID:        "nodeB",
		DependsOn: []graft.ID{"nodeA"},
		Run: func(ctx context.Context) (B, error) {
			a, _ := graft.Dep[A](ctx)
			return B{AValue: a.Value}, nil
		},
	})
}

// Node C - depends on B
func init() {
	graft.Register(graft.Node[C]{
		ID:        "nodeC",
		DependsOn: []graft.ID{"nodeB"},
		Run: func(ctx context.Context) (C, error) {
			b, _ := graft.Dep[B](ctx)
			return C{BValue: b.AValue}, nil
		},
	})
}
