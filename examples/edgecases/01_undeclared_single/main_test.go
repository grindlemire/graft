package main

import (
	"testing"

	"github.com/grindlemire/graft"
)

func TestEdgeCase(t *testing.T) {
	// This test should fail because app has an undeclared dependency
	graft.AssertDepsValid(t, ".")
}
