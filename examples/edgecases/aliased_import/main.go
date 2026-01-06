package main

import (
	_ "github.com/grindlemire/graft/examples/edgecases/aliased_import/consumer"
	_ "github.com/grindlemire/graft/examples/edgecases/aliased_import/producer"
)

func main() {
	// This main package exists to ensure both producer and consumer
	// are loaded during analysis
}
