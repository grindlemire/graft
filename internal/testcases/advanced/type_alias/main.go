// Package main tests type alias resolution.
//
// Purpose:
// Verify that 'Dep[T]' works when 'T' is a type alias (e.g., 'type MyString string').
//
// Failure Case:
// If the analyzer strictly matches the underlying type ('string') or fails to resolve the alias
// to the node registry, it will report the dependency as unused or undeclared.
package main

import (
    "context"
    "github.com/grindlemire/graft"
)

type MyString string

var DepNode = graft.Node[MyString]{
    ID: "dep",
    Run: func(ctx context.Context) (MyString, error) { return "val", nil },
}

var Node = graft.Node[string]{
    ID: "node",
    DependsOn: []graft.ID{"dep"},
    Run: func(ctx context.Context) (string, error) {
        val, _ := graft.Dep[MyString](ctx)
        return string(val), nil
    },
}
