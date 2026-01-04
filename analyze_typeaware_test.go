package graft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageLoader_Load(t *testing.T) {
	t.Run("load current package", func(t *testing.T) {
		cfg := AnalyzerConfig{
			WorkDir: ".",
		}
		loader := newPackageLoader(cfg)

		pkgs, err := loader.Load(".")
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		if len(pkgs) == 0 {
			t.Fatal("expected at least one package")
		}

		// Should find the graft package itself
		foundGraft := false
		for _, pkg := range pkgs {
			if pkg.Name == "graft" {
				foundGraft = true
				break
			}
		}

		if !foundGraft {
			t.Error("expected to find graft package")
		}
	})

	t.Run("load examples/simple", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple")

		// Check if examples directory exists
		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple directory not found")
		}

		cfg := AnalyzerConfig{
			WorkDir: exampleDir,
		}
		loader := newPackageLoader(cfg)

		pkgs, err := loader.Load(exampleDir)
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		if len(pkgs) == 0 {
			t.Fatal("expected at least one package")
		}

		t.Logf("Loaded %d packages", len(pkgs))
	})

	t.Run("load with test files", func(t *testing.T) {
		cfg := AnalyzerConfig{
			WorkDir:      ".",
			IncludeTests: true,
		}
		loader := newPackageLoader(cfg)

		pkgs, err := loader.Load(".")
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		if len(pkgs) == 0 {
			t.Fatal("expected at least one package")
		}

		// With tests included, should have test files
		hasTestFiles := false
		for _, pkg := range pkgs {
			if len(pkg.GoFiles) > 0 {
				for _, f := range pkg.GoFiles {
					if strings.HasSuffix(f, "_test.go") {
						hasTestFiles = true
						break
					}
				}
			}
			if hasTestFiles {
				break
			}
		}

		if !hasTestFiles {
			t.Log("Warning: No test files found even with IncludeTests=true")
		}
	})

	t.Run("load nonexistent directory", func(t *testing.T) {
		cfg := AnalyzerConfig{
			WorkDir: "/nonexistent/path/that/should/not/exist",
		}
		loader := newPackageLoader(cfg)

		_, err := loader.Load("/nonexistent/path/that/should/not/exist")
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}

		t.Logf("Got expected error: %v", err)
	})
}

func TestPackageLoader_filterPackages(t *testing.T) {
	cfg := AnalyzerConfig{
		WorkDir:      ".",
		IncludeTests: false,
	}
	loader := newPackageLoader(cfg)

	// Load packages first
	pkgs, err := loader.Load(".")
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	// Filter them
	filtered := loader.filterPackages(pkgs)

	// Filtered should be <= original
	if len(filtered) > len(pkgs) {
		t.Errorf("filtered packages (%d) > original (%d)", len(filtered), len(pkgs))
	}

	t.Logf("Original: %d packages, Filtered: %d packages", len(pkgs), len(filtered))
}

func TestAnalyzerConfig_defaults(t *testing.T) {
	cfg := AnalyzerConfig{}
	analyzer := newTypeAwareAnalyzer(cfg)

	if analyzer.cfg.WorkDir != "." {
		t.Errorf("expected default WorkDir to be '.', got %q", analyzer.cfg.WorkDir)
	}

	if analyzer.cfg.IncludeTests {
		t.Error("expected default IncludeTests to be false")
	}

	if analyzer.cfg.Debug {
		t.Error("expected default Debug to be false")
	}

	if len(analyzer.cfg.BuildTags) != 0 {
		t.Error("expected default BuildTags to be empty")
	}
}
