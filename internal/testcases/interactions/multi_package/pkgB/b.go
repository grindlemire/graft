// Package pkgb defines a node that depends on pkgA in the multi-package test.
package pkgb

import (
	"context"
	"github.com/grindlemire/graft"
	"testmod/pkgA"
)

var Node = graft.Node[string]{
	ID: "nodeB",
	DependsOn: []graft.ID{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		val, _ := graft.Dep[pkga.Output](ctx)
		return string(val), nil
	},
}
