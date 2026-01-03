// Package main tests method chaining on Dep[T] results.
//
// Purpose:
// Verify that code structured like 'graft.Dep[T](ctx).Method()' is correctly parsed.
// Although the Go parser handles this, custom AST walkers must ensure they visit the CallExpr
// inside the SelectorExpr chain.
//
// Failure Case:
// If the walker misses the inner 'Dep[T]' call because it's wrapped in a selector expression,
// the dependency will be missed.
package main

import (
    "context"
    "github.com/grindlemire/graft"
)

type Output string
func (o Output) String() string { return string(o) }

var DepNode = graft.Node[Output]{
    ID: "dep",
    Run: func(ctx context.Context) (Output, error) { return "val", nil },
}

var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep"},
    Run: func(ctx context.Context) (string, error) {
		// Method chaining
        v, _ := graft.Dep[Output](ctx)
        return v.String(), nil
    },
}
