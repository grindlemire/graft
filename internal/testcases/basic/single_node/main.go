// Package main defines a basic single-node test case.
//
// Purpose:
// Verify that the analyzer can correctly identify a single node with no dependencies.
//
// Failure Case:
// If the analyzer fails to find the node or reports phantom dependencies, this test will fail.
package main

import (
	"context"
	"github.com/grindlemire/graft"
)

var Node = graft.Node[string]{
	ID: "node1",
	Run: func(ctx context.Context) (string, error) {
		return "hello", nil
	},
}
