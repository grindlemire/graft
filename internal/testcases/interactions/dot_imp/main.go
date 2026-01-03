// Package main tests dot imports.
//
// Purpose:
// Verify that 'Dep[T]' calls work correctly when the package defining 'T' is dot-imported
// (e.g., import . "pkgA"), meaning 'T' is referenced without a package qualifier.
//
// Failure Case:
// If the analyzer relies on the selector expression (pkg.Type) to find the type, it will fail
// here because 'Output' appears as a simplified identifier, not a selector.
package main

import (
	"context"
	"github.com/grindlemire/graft"
	. "testmod/pkgA"
)

var NodeB = graft.Node[string]{
	ID: "nodeB",
	DependsOn: []graft.ID{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		val, _ := graft.Dep[Output](ctx)
		return string(val), nil
	},
}
