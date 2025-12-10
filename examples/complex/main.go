package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grindlemire/graft"
	// if you can have a dedicated import file for your nodes for simpler
	// main.go files (see nodes.go in this directory for the imports)
)

func main() {
	start := time.Now()
	results, err := graft.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)

	fmt.Println("\n=== Results ===")
	for id, result := range results {
		fmt.Printf("%s: %+v\n", id, result)
	}

	fmt.Printf("\n=== Timing ===\n")
	fmt.Printf("Total execution time: %v\n", elapsed)
	fmt.Println("9 nodes executed in 5 parallel levels")

	fmt.Printf("\n=== Graph ===\n")
	err = graft.PrintGraph(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
