// Package main tests detection of unused dependencies.
//
// Purpose:
// Verify that the analyzer reports an error (or Unused warning) when 'DependsOn' declares a dependency
// that is never actually called via 'Dep[T]' in the 'Run' function.
//
// Failure Case:
// If 'Unused' is empty, the analyzer incorrectly thinks the dependency was used (false positive usage)
// or failed to check the declared list.
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

// Unused usage
var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep"},
    Run: func(ctx context.Context) (string, error) {
        return "val", nil
    },
}
