package main

import (
	"testing"

	"github.com/grindlemire/graft"
)

func TestEdgeCase(t *testing.T) {
	graft.AssertDepsValid(t, ".")
}
