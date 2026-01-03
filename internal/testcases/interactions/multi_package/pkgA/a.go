// Package pkga defines a dependency node for the multi-package test.
package pkga

import (
	"context"
	"github.com/grindlemire/graft"
)

type Output string

var Node = graft.Node[Output]{
	ID: "nodeA",
	Run: func(ctx context.Context) (Output, error) {
		return "from A", nil
	},
}
