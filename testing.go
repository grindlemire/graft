package graft

import (
	"sort"
	"testing"
)

// AssertOpts configures the behavior of AssertDepsValid.
type AssertOpts struct {
	Verbose bool // prints node summaries (DeclaredDeps, UsedDeps, Status)
	Debug   bool // prints AST-level tracing (file walking, composite literals, etc.)
}

// AssertOption is a functional option for configuring AssertDepsValid.
type AssertOption func(*AssertOpts)

// WithVerboseTesting enables verbose output showing each node's declared and used
// dependencies along with their validation status.
func WithVerboseTesting() AssertOption {
	return func(o *AssertOpts) { o.Verbose = true }
}

// WithDebugTesting enables AST-level debug output showing file walking,
// composite literal detection, and dependency extraction details.
func WithDebugTesting() AssertOption {
	return func(o *AssertOpts) { o.Debug = true }
}

// AssertDepsValid is a test helper that validates all graft.Node dependency
// declarations in the specified directory match their actual usage.
//
// Add this to your test suite to catch dependency mismatches at test time
// rather than runtime. It uses AST analysis to compare DependsOn declarations
// against actual Dep[T] calls in Run functions.
//
// This will fail the test if:
//   - Any node uses Dep[T](ctx) without declaring the corresponding dependency in DependsOn
//   - Any node declares a dependency in DependsOn but never uses it
//
// Basic usage in your test file:
//
//	func TestNodeDependencies(t *testing.T) {
//	    graft.AssertDepsValid(t, ".")
//	}
//
// With verbose output (shows each node's deps):
//
//	func TestNodeDependencies(t *testing.T) {
//	    graft.AssertDepsValid(t, ".", graft.WithVerboseTesting())
//	}
//
// With AST debug output:
//
//	func TestNodeDependencies(t *testing.T) {
//	    graft.AssertDepsValid(t, ".", graft.WithDebugTesting())
//	}
//
// Example failure output:
//
//	graft.AssertDepsValid: db (nodes/db/db.go): undeclared deps: [cache]
//	  → node "db" uses Dep[cache.Output](ctx) but does not declare "cache" in DependsOn
func AssertDepsValid(t testing.TB, dir string, opts ...AssertOption) {
	t.Helper()

	cfg := &AssertOpts{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Enable AST-level debug flags if requested
	if cfg.Debug {
		AnalyzeDirDebug = true
		AnalyzeFileDebug = true
		defer func() {
			AnalyzeDirDebug = false
			AnalyzeFileDebug = false
		}()
	}

	results, err := AnalyzeDir(dir)
	if err != nil {
		t.Fatalf("graft.AssertDepsValid: failed to analyze directory %q: %v", dir, err)
	}

	// Verbose output: show each node's dependency summary
	if cfg.Verbose {
		t.Logf("graft.AssertDepsValid: analyzing %q - found %d node(s)", dir, len(results))
		for _, r := range results {
			sortedDeclared := make([]string, len(r.DeclaredDeps))
			copy(sortedDeclared, r.DeclaredDeps)
			sort.Strings(sortedDeclared)

			sortedUsed := make([]string, len(r.UsedDeps))
			copy(sortedUsed, r.UsedDeps)
			sort.Strings(sortedUsed)

			t.Logf("─────────────────────────────────────────")
			t.Logf("Node: %q (%s)", r.NodeID, r.File)
			t.Logf("  DeclaredDeps (from DependsOn): %v", sortedDeclared)
			t.Logf("  UsedDeps (from Dep[T] calls):  %v", sortedUsed)
			if r.HasIssues() {
				t.Logf("  Undeclared (used but not declared): %v", r.Undeclared)
				t.Logf("  Unused (declared but not used):     %v", r.Unused)
			} else {
				t.Logf("  Status: OK")
			}
		}
		t.Logf("─────────────────────────────────────────")
	}

	var failed bool
	for _, r := range results {
		if !r.HasIssues() {
			continue
		}

		failed = true
		t.Errorf("graft.AssertDepsValid: %s", r.String())

		// Provide detailed breakdown
		if len(r.Undeclared) > 0 {
			for _, dep := range r.Undeclared {
				t.Errorf("  → node %q uses Dep[%s.Output](ctx) but does not declare %q in DependsOn", r.NodeID, dep, dep)
			}
		}
		if len(r.Unused) > 0 {
			for _, dep := range r.Unused {
				t.Errorf("  → node %q declares %q in DependsOn but never uses it", r.NodeID, dep)
			}
		}
	}

	if !failed && len(results) > 0 && !cfg.Verbose {
		t.Logf("graft.AssertDepsValid: validated %d node(s) - all dependencies correct", len(results))
	}
}

// CheckDepsValid is like [AssertDepsValid] but returns results instead of failing.
//
// This is useful for custom validation logic, reporting, or CI integration
// where you need programmatic access to the results.
//
// Example:
//
//	results, err := graft.CheckDepsValid("./nodes")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range results {
//	    if r.HasIssues() {
//	        notify(r.NodeID, r.Undeclared, r.Unused)
//	    }
//	}
func CheckDepsValid(dir string) ([]AnalysisResult, error) {
	return AnalyzeDir(dir)
}
