// Package main tests dependency usage within closures and anonymous functions.
//
// Purpose:
// Verify that the analyzer finds 'Dep[T]' calls embedded inside anonymous functions or closures
// defined within 'Run'.
//
// Failure Case:
// If the AST traversal is shallow (only inspecting the top-level block of 'Run'), duplicate code blocks,
// or fails to recurse into FuncLit nodes, these dependencies will be missed (false unused/undeclared).
package main

import (
    "context"
    "github.com/grindlemire/graft"
)

type Output string

var DepNode = graft.Node[Output]{
    ID: "dep",
    Run: func(ctx context.Context) (Output, error) { return "val", nil },
}

var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep"},
    Run: func(ctx context.Context) (string, error) {
        helper := func() {
            graft.Dep[Output](ctx)
        }
        helper()
        return "ok", nil
    },
}
