package graft

import (
	"path/filepath"
	"testing"
)

// BenchmarkAnalyzeDir benchmarks the type-aware analyzer on different example projects.
func BenchmarkAnalyzeDir(b *testing.B) {
	benchmarks := []struct {
		name string
		dir  string
	}{
		{"simple", "examples/simple"},
		{"complex", "examples/complex"},
		{"diamond", "examples/diamond"},
		{"fanout", "examples/fanout"},
		{"httpserver", "examples/httpserver"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			absDir, err := filepath.Abs(bm.dir)
			if err != nil {
				b.Fatalf("failed to get absolute path: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results, err := AnalyzeDir(absDir)
				if err != nil {
					b.Fatalf("AnalyzeDir error: %v", err)
				}
				if len(results) == 0 {
					b.Fatal("expected nodes, got 0")
				}
			}
		})
	}
}

// BenchmarkAnalyzeDir_CurrentDirectory benchmarks analyzing the full graft project.
func BenchmarkAnalyzeDir_CurrentDirectory(b *testing.B) {
	absDir, err := filepath.Abs(".")
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := AnalyzeDir(absDir)
		if err != nil {
			b.Fatalf("AnalyzeDir error: %v", err)
		}
		if len(results) == 0 {
			b.Fatal("expected nodes from examples, got 0")
		}
	}
}

// BenchmarkValidateDeps benchmarks the ValidateDeps function.
func BenchmarkValidateDeps(b *testing.B) {
	absDir, err := filepath.Abs("examples/simple")
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateDeps(absDir)
		if err != nil {
			b.Fatalf("ValidateDeps error: %v", err)
		}
	}
}

// BenchmarkTypeAwareAnalyzer_Phases benchmarks individual phases of the analyzer.
func BenchmarkTypeAwareAnalyzer_Phases(b *testing.B) {
	absDir, err := filepath.Abs("examples/simple")
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	b.Run("phase_1_package_loading", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg := AnalyzerConfig{
				WorkDir: absDir,
			}
			loader := newPackageLoader(cfg)
			pkgs, err := loader.Load(absDir)
			if err != nil {
				b.Fatalf("Load error: %v", err)
			}
			if len(pkgs) == 0 {
				b.Fatal("expected packages, got 0")
			}
		}
	})

	b.Run("phase_2_ssa_building", func(b *testing.B) {
		cfg := AnalyzerConfig{
			WorkDir: absDir,
		}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(absDir)
		if err != nil {
			b.Fatalf("Load error: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := newSSABuilder()
			prog, _, err := builder.Build(pkgs)
			if err != nil {
				b.Fatalf("Build error: %v", err)
			}
			if prog == nil {
				b.Fatal("expected SSA program, got nil")
			}
		}
	})

	b.Run("phase_3_node_discovery", func(b *testing.B) {
		cfg := AnalyzerConfig{
			WorkDir: absDir,
		}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(absDir)
		if err != nil {
			b.Fatalf("Load error: %v", err)
		}

		builder := newSSABuilder()
		prog, srcPkgs, err := builder.Build(pkgs)
		if err != nil {
			b.Fatalf("Build error: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
			nodes, err := discoverer.FindNodes()
			if err != nil {
				b.Fatalf("FindNodes error: %v", err)
			}
			if len(nodes) == 0 {
				b.Fatal("expected nodes, got 0")
			}
		}
	})

	b.Run("phase_4_type_mapping", func(b *testing.B) {
		cfg := AnalyzerConfig{
			WorkDir: absDir,
		}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(absDir)
		if err != nil {
			b.Fatalf("Load error: %v", err)
		}

		builder := newSSABuilder()
		prog, srcPkgs, err := builder.Build(pkgs)
		if err != nil {
			b.Fatalf("Build error: %v", err)
		}

		discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
		nodes, err := discoverer.FindNodes()
		if err != nil {
			b.Fatalf("FindNodes error: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mapper := newTypeIDMapper()
			err := mapper.BuildMapping(nodes)
			if err != nil {
				b.Fatalf("BuildMapping error: %v", err)
			}
		}
	})

	b.Run("phase_5_dependency_extraction", func(b *testing.B) {
		cfg := AnalyzerConfig{
			WorkDir: absDir,
		}
		loader := newPackageLoader(cfg)
		pkgs, err := loader.Load(absDir)
		if err != nil {
			b.Fatalf("Load error: %v", err)
		}

		builder := newSSABuilder()
		prog, srcPkgs, err := builder.Build(pkgs)
		if err != nil {
			b.Fatalf("Build error: %v", err)
		}

		discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
		nodes, err := discoverer.FindNodes()
		if err != nil {
			b.Fatalf("FindNodes error: %v", err)
		}

		mapper := newTypeIDMapper()
		err = mapper.BuildMapping(nodes)
		if err != nil {
			b.Fatalf("BuildMapping error: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			extractor := newDependencyExtractor(mapper, prog, prog.Fset)
			for _, node := range nodes {
				_, err := extractor.AnalyzeNode(node)
				if err != nil {
					b.Fatalf("AnalyzeNode error: %v", err)
				}
			}
		}
	})
}

// BenchmarkTypeIDMapper benchmarks the type mapper operations.
func BenchmarkTypeIDMapper(b *testing.B) {
	absDir, err := filepath.Abs("examples/complex")
	if err != nil {
		b.Fatalf("failed to get absolute path: %v", err)
	}

	cfg := AnalyzerConfig{
		WorkDir: absDir,
	}
	loader := newPackageLoader(cfg)
	pkgs, err := loader.Load(absDir)
	if err != nil {
		b.Fatalf("Load error: %v", err)
	}

	builder := newSSABuilder()
	prog, srcPkgs, err := builder.Build(pkgs)
	if err != nil {
		b.Fatalf("Build error: %v", err)
	}

	discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
	nodes, err := discoverer.FindNodes()
	if err != nil {
		b.Fatalf("FindNodes error: %v", err)
	}

	mapper := newTypeIDMapper()
	err = mapper.BuildMapping(nodes)
	if err != nil {
		b.Fatalf("BuildMapping error: %v", err)
	}

	b.Run("ResolveType", func(b *testing.B) {
		if len(nodes) == 0 {
			b.Skip("no nodes to benchmark")
		}
		outputType := nodes[0].OutputType

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mapper.ResolveType(outputType)
			if err != nil {
				b.Fatalf("ResolveType error: %v", err)
			}
		}
	})

	b.Run("GetType", func(b *testing.B) {
		if len(nodes) == 0 {
			b.Skip("no nodes to benchmark")
		}
		nodeID := nodes[0].ID

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mapper.GetType(nodeID)
			if err != nil {
				b.Fatalf("GetType error: %v", err)
			}
		}
	})

	b.Run("HasType", func(b *testing.B) {
		if len(nodes) == 0 {
			b.Skip("no nodes to benchmark")
		}
		outputType := nodes[0].OutputType

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mapper.HasType(outputType)
		}
	})
}
