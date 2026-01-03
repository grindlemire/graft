// Package main tests handling of duplicate entries in DependsOn.
//
// Purpose:
// Verify how the analyzer handles duplicate dependency IDs in the 'DependsOn' list.
// Ideally, this should not crash and potentially be normalized or reported.
//
// Failure Case:
// If the analyzer crashes or reports inconsistent counts, it fails this test.
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

// Duplicate usage
var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep", "dep"}, // Duplicate
    Run: func(ctx context.Context) (string, error) {
        graft.Dep[DepOut](ctx)
        return "val", nil
    },
}
