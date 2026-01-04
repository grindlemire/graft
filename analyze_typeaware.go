package graft

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

// AnalyzerConfig configures the type-aware analyzer
type AnalyzerConfig struct {
	// BuildTags specifies build tags to use (e.g., []string{"integration"})
	BuildTags []string

	// IncludeTests includes _test.go files in analysis
	IncludeTests bool

	// WorkDir sets the working directory for package resolution
	WorkDir string

	// Debug enables detailed logging for troubleshooting
	Debug bool
}

// typeAwareAnalyzer orchestrates the entire analysis pipeline
type typeAwareAnalyzer struct {
	cfg AnalyzerConfig
}

// newTypeAwareAnalyzer creates a new type-aware analyzer with the given config
func newTypeAwareAnalyzer(cfg AnalyzerConfig) *typeAwareAnalyzer {
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}
	return &typeAwareAnalyzer{cfg: cfg}
}

// debugf prints debug output if Debug is enabled
func (a *typeAwareAnalyzer) debugf(format string, args ...interface{}) {
	if a.cfg.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// packageLoader handles go/packages loading
type packageLoader struct {
	cfg *packages.Config
}

// newPackageLoader creates a new package loader
func newPackageLoader(cfg AnalyzerConfig) *packageLoader {
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

// filterPackages filters out packages we don't want to analyze
func (l *packageLoader) filterPackages(pkgs []*packages.Package) []*packages.Package {
	var filtered []*packages.Package

	for _, pkg := range pkgs {
		// Skip packages without Go files
		if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 {
			continue
		}

		// Skip test packages unless explicitly requested
		if !l.cfg.Tests && strings.HasSuffix(pkg.ID, ".test") {
			continue
		}

		filtered = append(filtered, pkg)
	}

	return filtered
}
