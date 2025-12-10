package main

import (
	"testing"

	"github.com/grindlemire/graft"
)

func TestMyGraph(t *testing.T) {
	graft.AssertDepsValid(t, ".", graft.WithVerboseTesting())
}
