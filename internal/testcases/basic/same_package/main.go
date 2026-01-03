// Package main defines a test case with two nodes in the same package.
//
// Purpose:
// Verify that the analyzer correctly resolves dependencies between nodes defined in the same package/file.
//
// Failure Case:
// If the analyzer fails to link 'NodeB' to 'NodeA' or reports 'NodeA' as undeclared/unused incorrectly,
// it indicates issues with intra-package dependency resolution.
package main

import (
	"context"
	"github.com/grindlemire/graft"
)

// NodeA definition
type AOut string
var NodeA = graft.Node[AOut]{
	ID: "nodeA",
	Run: func(ctx context.Context) (AOut, error) {
		return "a", nil
	},
}

// NodeB depends on NodeA
var NodeB = graft.Node[string]{
	ID: "nodeB",
	DependsOn: []graft.ID{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		val, _ := graft.Dep[AOut](ctx)
		return string(val), nil
	},
}
