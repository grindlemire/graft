// Package main tests import aliases.
//
// Purpose:
// Verify that 'Dep[T]' calls work correctly when the package defining 'T' is imported with an alias
// (e.g., import myalias "pkgA").
//
// Failure Case:
// If the analyzer resolves the type based on the alias name ('myalias.Output') instead of the canonical
// type ('pkgA.Output'), it might fail to match the node ID.
package main

import (
	"context"
	"github.com/grindlemire/graft"
	myalias "testmod/pkgA"
)

var Node = graft.Node[string]{
	ID: "nodeB",
	DependsOn: []graft.ID{"nodeA"},
	Run: func(ctx context.Context) (string, error) {
		val, _ := graft.Dep[myalias.Output](ctx)
		return string(val), nil
	},
}
