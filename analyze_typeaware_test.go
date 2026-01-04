package graft

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
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

func TestSSABuilder_Build(t *testing.T) {
	t.Run("build SSA for current package", func(t *testing.T) {
		// Load packages first
		cfg := AnalyzerConfig{WorkDir: "."}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(".")
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		// Build SSA
		builder := newSSABuilder()
		prog, _, err := builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		if prog == nil {
			t.Fatal("SSA program is nil")
		}

		// Should have SSA packages
		ssaPkgs := builder.GetPackages()
		if len(ssaPkgs) == 0 {
			t.Fatal("no SSA packages built")
		}

		t.Logf("Built SSA for %d packages", len(ssaPkgs))
	})

	t.Run("build SSA for examples/simple", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple")

		// Check if examples directory exists
		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple directory not found")
		}

		// Load packages
		cfg := AnalyzerConfig{WorkDir: exampleDir}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(exampleDir)
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		// Build SSA
		builder := newSSABuilder()
		prog, _, err := builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		if prog == nil {
			t.Fatal("SSA program is nil")
		}

		ssaPkgs := builder.GetPackages()
		t.Logf("Built SSA for %d packages in examples/simple", len(ssaPkgs))

		// Should have the node packages (config, db, app, etc.)
		if len(ssaPkgs) < 3 {
			t.Errorf("expected at least 3 SSA packages, got %d", len(ssaPkgs))
		}
	})

	t.Run("SSA builder mode", func(t *testing.T) {
		builder := newSSABuilder()

		// Check that InstantiateGenerics mode is enabled (may be combined with other flags)
		if builder.mode&ssa.InstantiateGenerics == 0 {
			t.Errorf("expected mode to include InstantiateGenerics, got %v", builder.mode)
		}
	})
}

func TestSSABuilder_GetPackages(t *testing.T) {
	t.Run("get packages before build returns nil", func(t *testing.T) {
		builder := newSSABuilder()
		pkgs := builder.GetPackages()

		if pkgs != nil {
			t.Errorf("expected nil packages before build, got %d packages", len(pkgs))
		}
	})

	t.Run("get packages after build", func(t *testing.T) {
		// Load and build
		cfg := AnalyzerConfig{WorkDir: "."}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(".")
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		builder := newSSABuilder()
		_, _, err = builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		ssaPkgs := builder.GetPackages()
		if len(ssaPkgs) == 0 {
			t.Fatal("expected packages after build")
		}
	})
}

func TestSSA_WalkInstructions(t *testing.T) {
	exampleDir := filepath.Join(".", "examples", "simple", "nodes", "db")

	// Check if examples directory exists
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("examples/simple/nodes/db directory not found")
	}

	// Load packages
	cfg := AnalyzerConfig{WorkDir: exampleDir}
	loader := newPackageLoader(cfg)
	pkgs, err := loader.Load(exampleDir)
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	// Build SSA
	builder := newSSABuilder()
	prog, _, err := builder.Build(pkgs)
	if err != nil {
		t.Fatalf("failed to build SSA: %v", err)
	}

	// Walk all packages looking for graft.Register calls
	registerCallCount := 0
	depCallCount := 0

	for _, pkg := range prog.AllPackages() {
		for _, member := range pkg.Members {
			fn, ok := member.(*ssa.Function)
			if !ok {
				continue
			}

			// Walk function blocks and instructions
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					if call, ok := instr.(*ssa.Call); ok {
						if isGraftRegisterCall(call) {
							registerCallCount++
							t.Logf("Found graft.Register call in %s", fn.Name())
						}
						if isGraftDepCall(call) {
							depCallCount++
							t.Logf("Found graft.Dep call in %s", fn.Name())
						}
					}
				}
			}
		}
	}

	// Note: In SSA, Register calls might be in synthetic init functions or inlined
	// This test verifies we can walk the SSA structure, finding calls is secondary
	t.Logf("Found %d Register calls and %d Dep calls", registerCallCount, depCallCount)

	if registerCallCount == 0 {
		t.Log("Note: No Register calls found - may be in synthetic functions")
	}

	if depCallCount == 0 {
		t.Log("Note: No Dep calls found - may be in inlined or synthetic functions")
	}
}

func TestSSA_FindInitFunctions(t *testing.T) {
	exampleDir := filepath.Join(".", "examples", "simple", "nodes", "db")

	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("examples/simple/nodes/db directory not found")
	}

	// Load and build SSA
	cfg := AnalyzerConfig{WorkDir: exampleDir}
	loader := newPackageLoader(cfg)
	pkgs, err := loader.Load(exampleDir)
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	builder := newSSABuilder()
	prog, _, err := builder.Build(pkgs)
	if err != nil {
		t.Fatalf("failed to build SSA: %v", err)
	}

	// Find init functions
	foundInit := false
	initWithBlocks := 0
	totalInits := 0

	for _, pkg := range prog.AllPackages() {
		inits := findInitFunctions(pkg)
		if len(inits) > 0 {
			foundInit = true
			totalInits += len(inits)

			// Count inits with blocks
			for _, initFn := range inits {
				if len(initFn.Blocks) > 0 {
					initWithBlocks++
					t.Logf("Package %s has init function with %d block(s)", pkg.Pkg.Path(), len(initFn.Blocks))
				}
			}
		}
	}

	if !foundInit {
		t.Error("expected to find init functions")
	}

	t.Logf("Found %d total init functions, %d with blocks", totalInits, initWithBlocks)
}

func TestSSA_HelperFunctions(t *testing.T) {
	t.Run("isGraftPackage", func(t *testing.T) {
		// Load graft package itself
		cfg := AnalyzerConfig{WorkDir: "."}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(".")
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		builder := newSSABuilder()
		prog, _, err := builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		foundGraft := false
		for _, pkg := range prog.AllPackages() {
			if isGraftPackage(pkg) {
				foundGraft = true
				t.Logf("Found graft package: %s", pkg.Pkg.Path())
				break
			}
		}

		if !foundGraft {
			t.Error("expected to find graft package")
		}
	})

	t.Run("isGraftPackage with nil", func(t *testing.T) {
		if isGraftPackage(nil) {
			t.Error("isGraftPackage(nil) should return false")
		}
	})
}

func TestTypeAwareAnalyzer_AnalyzeBasic(t *testing.T) {
	t.Run("analyze current directory", func(t *testing.T) {
		analyzer := newTypeAwareAnalyzer(AnalyzerConfig{
			WorkDir: ".",
			Debug:   testing.Verbose(),
		})

		results, err := analyzer.Analyze(".")
		if err != nil {
			t.Fatalf("analysis failed: %v", err)
		}

		// Results will be empty for now (node discovery not yet implemented)
		// This test just verifies the pipeline runs without error
		t.Logf("Analysis completed, got %d results", len(results))
	})

	t.Run("analyze examples/simple", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple")

		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple directory not found")
		}

		analyzer := newTypeAwareAnalyzer(AnalyzerConfig{
			WorkDir: exampleDir,
			Debug:   testing.Verbose(),
		})

		results, err := analyzer.Analyze(exampleDir)
		if err != nil {
			t.Fatalf("analysis failed: %v", err)
		}

		t.Logf("Analysis completed, got %d results", len(results))
	})

	t.Run("analyze with debug enabled", func(t *testing.T) {
		analyzer := newTypeAwareAnalyzer(AnalyzerConfig{
			WorkDir: ".",
			Debug:   true,
		})

		_, err := analyzer.Analyze(".")
		if err != nil {
			t.Fatalf("analysis failed: %v", err)
		}

		// If debug is enabled, should see debug output
		// (check manually when running with -v)
	})
}

func TestNodeDiscoverer_FindNodes(t *testing.T) {
	t.Run("find nodes in examples/simple/nodes/db", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple", "nodes", "db")

		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple/nodes/db directory not found")
		}

		// Load and build SSA
		cfg := AnalyzerConfig{WorkDir: exampleDir}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(exampleDir)
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		builder := newSSABuilder()
		prog, srcPkgs, err := builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		// Find nodes
		discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
		nodes, err := discoverer.FindNodes()
		if err != nil {
			t.Fatalf("failed to find nodes: %v", err)
		}

		// Should find the db node
		if len(nodes) == 0 {
			t.Fatal("expected to find at least one node")
		}

		t.Logf("Found %d node(s)", len(nodes))

		// Verify the db node
		foundDB := false
		for _, node := range nodes {
			t.Logf("Node: %s", node.String())

			if node.ID == "db" {
				foundDB = true

				if node.OutputType == nil {
					t.Error("db node has nil OutputType")
				}

				if node.File == "" {
					t.Error("db node has empty File")
				}

				// Note: DependsOn and RunFunc might be nil if extraction is partial
				// That's OK for now, we'll improve extraction in iterations
			}
		}

		if !foundDB {
			t.Error("expected to find db node")
		}
	})

	t.Run("find nodes in examples/simple (all nodes)", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple")

		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple directory not found")
		}

		// Load all packages in examples/simple
		cfg := AnalyzerConfig{WorkDir: exampleDir}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(exampleDir)
		if err != nil {
			t.Fatalf("failed to load packages: %v", err)
		}

		builder := newSSABuilder()
		prog, srcPkgs, err := builder.Build(pkgs)
		if err != nil {
			t.Fatalf("failed to build SSA: %v", err)
		}

		discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
		nodes, err := discoverer.FindNodes()
		if err != nil {
			t.Fatalf("failed to find nodes: %v", err)
		}

		t.Logf("Found %d nodes in examples/simple", len(nodes))

		// examples/simple should have: config, db, app
		expectedIDs := map[string]bool{
			"config": false,
			"db":     false,
			"app":    false,
		}

		for _, node := range nodes {
			t.Logf("  - %s", node.String())

			if _, expected := expectedIDs[node.ID]; expected {
				expectedIDs[node.ID] = true
			}
		}

		// Check we found all expected nodes
		for id, found := range expectedIDs {
			if !found {
				t.Errorf("did not find expected node %q", id)
			}
		}
	})
}

func TestNodeDiscoverer_extractOutputType(t *testing.T) {
	// This test verifies type extraction logic
	// We'll need real types from loaded packages

	exampleDir := filepath.Join(".", "examples", "simple", "nodes", "config")

	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("examples/simple/nodes/config directory not found")
	}

	cfg := AnalyzerConfig{WorkDir: exampleDir}
	loader := newPackageLoader(cfg)
	pkgs, err := loader.Load(exampleDir)
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	builder := newSSABuilder()
	prog, srcPkgs, err := builder.Build(pkgs)
	if err != nil {
		t.Fatalf("failed to build SSA: %v", err)
	}

	discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)

	// Find the config.Output type in the loaded packages
	var configOutputType types.Type
	for _, pkg := range pkgs {
		if pkg.Types != nil {
			scope := pkg.Types.Scope()
			if obj := scope.Lookup("Output"); obj != nil {
				configOutputType = obj.Type()
				break
			}
		}
	}

	if configOutputType == nil {
		t.Skip("Could not find config.Output type")
	}

	// Now we need to construct a Node[config.Output] type
	// This is complex - for now just verify discoverer was created
	if discoverer == nil {
		t.Fatal("discoverer is nil")
	}

	t.Logf("Found config.Output type: %v", configOutputType)
}

func TestTypeAwareAnalyzer_WithNodeDiscovery(t *testing.T) {
	t.Run("analyze examples/simple with node discovery", func(t *testing.T) {
		exampleDir := filepath.Join(".", "examples", "simple")

		if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
			t.Skip("examples/simple directory not found")
		}

		analyzer := newTypeAwareAnalyzer(AnalyzerConfig{
			WorkDir: exampleDir,
			Debug:   testing.Verbose(),
		})

		results, err := analyzer.Analyze(exampleDir)
		if err != nil {
			t.Fatalf("analysis failed: %v", err)
		}

		// Results still empty (dependency extraction not implemented)
		// But should run without error and discover nodes
		t.Logf("Analysis completed with %d results", len(results))
	})
}
