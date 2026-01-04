package typeaware

import (
	"fmt"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// ssaBuilder builds SSA representation from loaded packages
type ssaBuilder struct {
	mode ssa.BuilderMode
	prog *ssa.Program
}

// newSSABuilder creates a new SSA builder
func newSSABuilder() *ssaBuilder {
	return &ssaBuilder{
		// InstantiateGenerics is required to analyze generic functions like Dep[T]
		mode: ssa.InstantiateGenerics | ssa.GlobalDebug,
	}
}

// Build constructs an SSA program from loaded packages
func (b *ssaBuilder) Build(pkgs []*packages.Package) (*ssa.Program, *[]*ssa.Package, error) {
	// Create SSA program
	prog, ssaPkgs := ssautil.AllPackages(pkgs, b.mode)

	// Build SSA for all packages
	prog.Build()

	b.prog = prog

	// Verify we built successfully
	if len(ssaPkgs) == 0 {
		return nil, nil, fmt.Errorf("no SSA packages built")
	}

	return prog, &ssaPkgs, nil
}

// GetPackages returns all SSA packages in the program
func (b *ssaBuilder) GetPackages() []*ssa.Package {
	if b.prog == nil {
		return nil
	}

	var pkgs []*ssa.Package
	for _, pkg := range b.prog.AllPackages() {
		pkgs = append(pkgs, pkg)
	}

	return pkgs
}
