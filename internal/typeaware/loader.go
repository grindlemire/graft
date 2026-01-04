package typeaware

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

// packageLoader handles go/packages loading
type packageLoader struct {
	cfg *packages.Config
}

// newPackageLoader creates a new package loader
func newPackageLoader(cfg Config) *packageLoader {
	pkgCfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir:   cfg.WorkDir,
		Tests: cfg.IncludeTests,
	}

	if len(cfg.BuildTags) > 0 {
		pkgCfg.BuildFlags = []string{"-tags", strings.Join(cfg.BuildTags, ",")}
	}

	return &packageLoader{cfg: pkgCfg}
}

// Load loads all packages in the given directory
func (l *packageLoader) Load(dir string) ([]*packages.Package, error) {
	// Update Dir to the target directory
	l.cfg.Dir = dir

	// Load all packages recursively
	pkgs, err := packages.Load(l.cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}

	// Check for package errors
	var errs []error
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			errs = append(errs, e)
		}
	})

	if len(errs) > 0 {
		// Return first error for simplicity
		return nil, fmt.Errorf("package errors: %v", errs[0])
	}

	return pkgs, nil
}
