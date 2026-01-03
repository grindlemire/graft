// Package main tests detection of undeclared dependencies.
//
// Purpose:
// Verify that the analyzer reports an error (or Undeclared warning) when 'Run' calls 'Dep[T]'
// but the 'DependsOn' list does not include the ID for 'T'.
//
// Failure Case:
// If 'Undeclared' is empty, the analyzer failed to compare the used dependency against the
// declared list or failed to find the usage entirely.
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

// Undeclared usage
var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{},
    Run: func(ctx context.Context) (string, error) {
        graft.Dep[DepOut](ctx)
        return "val", nil
    },
}
