package typeaware

import (
	"fmt"
)

// Config configures the type-aware analyzer
type Config struct {
	// BuildTags specifies build tags to use (e.g., []string{"integration"})
	BuildTags []string

	// IncludeTests includes _test.go files in analysis
	IncludeTests bool

	// WorkDir sets the working directory for package resolution
	WorkDir string

	// Debug enables detailed logging for troubleshooting
	Debug bool
}

// Analyzer orchestrates the entire type-aware analysis pipeline
type Analyzer struct {
	cfg Config
}

// New creates a new type-aware analyzer with the given config
func New(cfg Config) *Analyzer {
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}
	return &Analyzer{cfg: cfg}
}

// debugf prints debug output if Debug is enabled
func (a *Analyzer) debugf(format string, args ...interface{}) {
	if a.cfg.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// Analyze performs type-aware dependency analysis on a directory
func (a *Analyzer) Analyze(dir string) ([]Result, error) {
	a.debugf("Starting type-aware analysis of %s", dir)

	// Phase 1: Load packages
	a.debugf("Loading packages...")
	loader := newPackageLoader(a.cfg)
	pkgs, err := loader.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	a.debugf("Loaded %d packages", len(pkgs))

	// Phase 2: Build SSA
	a.debugf("Building SSA program...")
	builder := newSSABuilder()
	prog, srcPkgs, err := builder.Build(pkgs)
	if err != nil {
		return nil, fmt.Errorf("building SSA: %w", err)
	}

	ssaPkgs := builder.GetPackages()
	a.debugf("Built SSA for %d packages", len(ssaPkgs))

	// Phase 3: Discover nodes
	a.debugf("Discovering nodes...")
	discoverer := newNodeDiscoverer(prog, prog.Fset, srcPkgs)
	nodes, err := discoverer.FindNodes()
	if err != nil {
		return nil, fmt.Errorf("discovering nodes: %w", err)
	}
	a.debugf("Discovered %d nodes", len(nodes))

	for _, node := range nodes {
		a.debugf("  - %s", node.String())
	}

	// Phase 4: Build type-to-ID mapping
	a.debugf("Building type-to-ID mapping...")
	mapper := newTypeIDMapper()
	if err := mapper.BuildMapping(nodes); err != nil {
		return nil, fmt.Errorf("building type mapping: %w", err)
	}
	a.debugf("Built mapping for %d types", mapper.Size())

	// Log the mapping if debug is enabled
	for _, node := range nodes {
		typeKey := mapper.typeKey(node.OutputType)
		a.debugf("  %s → %q", typeKey, node.ID)
	}

	// Phase 5: Extract dependencies and analyze
	a.debugf("Extracting and analyzing dependencies...")
	extractor := newDependencyExtractor(mapper, prog, prog.Fset)

	var results []Result
	for _, node := range nodes {
		result, err := extractor.AnalyzeNode(node)
		if err != nil {
			// Log error but continue with other nodes
			a.debugf("Error analyzing node %q: %v", node.ID, err)
			continue
		}

		a.debugf("Analyzed node %q: declared=%v, used=%v",
			result.NodeID, result.DeclaredDeps, result.UsedDeps)

		if result.HasIssues() {
			a.debugf("  Issues: undeclared=%v, unused=%v",
				result.Undeclared, result.Unused)
		}

		results = append(results, result)
	}

	// Phase 6: Detect cycles and annotate results
	a.debugf("Detecting cycles...")
	detector := newCycleDetector(results)
	allCycles := detector.detectCycles()
	if len(allCycles) > 0 {
		a.debugf("Found %d cycles:", len(allCycles))
		for _, cycle := range allCycles {
			a.debugf("  - %v", cycle)
		}
	} else {
		a.debugf("No cycles detected")
	}

	nodeCycles := detector.mapCyclesToNodes()

	// Annotate each node's result with its cycles
	for i := range results {
		if cycles, found := nodeCycles[results[i].NodeID]; found {
			results[i].Cycles = cycles
			a.debugf("Node %q participates in %d cycle(s)", results[i].NodeID, len(cycles))
		}
	}

	a.debugf("Analysis complete: %d nodes analyzed", len(results))

	return results, nil
}
