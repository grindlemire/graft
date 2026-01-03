// Package main acts as the entry point for a multi-package test case.
//
// Purpose:
// Verify that the analyzer allows recursive scanning of sub-packages when the root (this file)
// imports them.
//
// Failure Case:
// If 'AnalyzeDir' only scans the root folder and ignores subdirectories without an explicit './...',
// the nodes in pkgA and pkgB will not be found.
package main
import (
	_ "testmod/pkgA"
	_ "testmod/pkgB"
)
func main() {}
