// Package lib simulates a vendored library package.
package lib

import (
	"context"
	"github.com/grindlemire/graft"
)

type Output string

var Node = graft.Node[Output]{
	ID: "libnode",
	Run: func(ctx context.Context) (Output, error) {
		return "lib", nil
	},
}
