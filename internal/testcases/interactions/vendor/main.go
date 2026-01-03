// Package main tests vendored dependencies.
//
// Purpose:
// Verify that dependencies located in a 'vendor/' directory are correctly resolved and that
// imports of vendored packages use the correct canonical path.
//
// Failure Case:
// If the analyzer (or 'go/packages' configuration) ignores the vendor directory or fails
// to map the vendored location to the canonical import path, the dependency type will not resolve.
package main

import (
	"context"
	"github.com/grindlemire/graft"
	"example.com/lib"
)

var Node = graft.Node[string]{
	ID: "node",
	DependsOn: []graft.ID{"libnode"},
	Run: func(ctx context.Context) (string, error) {
		val, _ := graft.Dep[lib.Output](ctx)
		return string(val), nil
	},
}
