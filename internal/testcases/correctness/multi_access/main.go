// Package main tests multiple specific calls to the same dependency.
//
// Purpose:
// Verify that calling 'Dep[T]' multiple times for the same 'T' is valid and counts as a single
// "used" dependency (set semantics).
//
// Failure Case:
// If the analyzer counts this as 2 used dependencies or fails to match against the single declared
// dependency, it indicates a set/list logic error.
package main

import (
    "context"
    "github.com/grindlemire/graft"
)

type DepOut string
var DepNode = graft.Node[DepOut]{
    ID: "dep",
    Run: func(ctx context.Context) (DepOut, error) { return "", nil },
}

// Multi Access usage
var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep"},
    Run: func(ctx context.Context) (string, error) {
        graft.Dep[DepOut](ctx)
        graft.Dep[DepOut](ctx)
        return "val", nil
    },
}
