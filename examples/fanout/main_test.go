package main

import (
	"testing"

	"github.com/grindlemire/graft"
)

// TestMyGraph verifies that the intentional cycle between svc5 and svc5-2 is detected.
func TestMyGraph(t *testing.T) {
	graft.AssertDepsValid(t, ".", graft.WithVerboseTesting())
}
