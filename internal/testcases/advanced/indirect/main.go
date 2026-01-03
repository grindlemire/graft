// Package main tests indirect (transitive) dependency usage.
//
// Purpose:
// Verify that the analyzer finds 'Dep[T]' calls inside helper functions called by 'Run'.
// This requires SSA call graph traversal or similar recursive analysis.
//
// Failure Case:
// If the analyzer only looks at the explicit source code of 'Run' and does not follow function calls,
// dependencies used in helpers will be missed (false unused/undeclared).
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
        return helper(ctx), nil
    },
}

func helper(ctx context.Context) string {
    val, _ := graft.Dep[Output](ctx)
    return string(val)
}
