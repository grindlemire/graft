// This package is used to demonstrate that the analyzer correctly infers dependency IDs
// even when the types are shared between multiple nodes in the same package.
package shared

import (
	"context"

	"github.com/grindlemire/graft"
)

const (
	ID1 graft.ID = "shared1"
	ID2 graft.ID = "shared2"
)

type Output1 struct {
	Name   string
	Result int
}

type Output2 struct {
	Name   string
	Result int
}

func init() {
	graft.Register(graft.Node[Output1]{
		ID:        ID1,
		DependsOn: []graft.ID{},
		Run:       run1,
	})

	graft.Register(graft.Node[Output2]{
		ID:        ID2,
		DependsOn: []graft.ID{},
		Run:       run2,
	})
}

func run1(ctx context.Context) (Output1, error) {
	return Output1{Name: "shared1", Result: 100}, nil
}

func run2(ctx context.Context) (Output2, error) {
	return Output2{Name: "shared2", Result: 200}, nil
}
