package main

import (
	"testing"

	"github.com/grindlemire/graft"
)

func TestMyGraph(t *testing.T) {
	graft.AssertDepsValid(t, ".")
	// You can also use WithVerboseTesting() to get more detailed output
	// graft.AssertDepsValid(t, ".", graft.WithVerboseTesting())
}
