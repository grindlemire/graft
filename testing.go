package graft

import "testing"

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
// For a specific subdirectory:
//
//	func TestNodeDependencies(t *testing.T) {
//	    graft.AssertDepsValid(t, "./nodes")
//	}
//
// Example failure output:
//
//	graft.AssertDepsValid: db (nodes/db/db.go): undeclared deps: [cache]
//	  → node "db" uses Dep[cache.Output](ctx) but does not declare "cache" in DependsOn
func AssertDepsValid(t testing.TB, dir string) {
	t.Helper()

	results, err := AnalyzeDir(dir)
	if err != nil {
		t.Fatalf("graft.AssertDepsValid: failed to analyze directory %q: %v", dir, err)
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

	if !failed && len(results) > 0 {
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
