package graft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockT implements testing.TB for testing AssertDepsValid
type mockT struct {
	testing.TB
	errors       []string
	fatals       []string
	logs         []string
	helperCalled bool
}

func (m *mockT) Helper() {
	m.helperCalled = true
}

func (m *mockT) Errorf(format string, args ...any) {
	m.errors = append(m.errors, format)
}

func (m *mockT) Fatalf(format string, args ...any) {
	m.fatals = append(m.fatals, format)
}

func (m *mockT) Logf(format string, args ...any) {
	m.logs = append(m.logs, format)
}

// setupTestModule creates a temporary Go module with the given Go files.
// This is required for the type-aware analyzer which needs valid Go modules.
// Returns the module directory path.
func setupTestModule(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Get the absolute path to the current graft package
	graftPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	// Create go.mod file with replace directive to use local graft package
	goMod := `module testmodule

go 1.21

require github.com/grindlemire/graft v0.0.0-00010101000000-000000000000

replace github.com/grindlemire/graft => ` + graftPath + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write all provided files
	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	return tmpDir
}

// TestAssertDepsValid is covered by integration tests in analyze_integration_test.go.
// The type-aware analyzer requires valid Go modules with real dependencies,
// making synthetic test cases complex to maintain. See TestAnalyzeDirIntegration
// and TestValidateDepsIntegration for comprehensive testing of AssertDepsValid.

func TestAssertDepsValidBadDir(t *testing.T) {
	mock := &mockT{}
	AssertDepsValid(mock, "/nonexistent/path/that/should/not/exist")

	if len(mock.fatals) != 1 {
		t.Errorf("expected 1 fatal for bad directory, got %d", len(mock.fatals))
	}
}

// TestAssertDepsValidWithVerbose is covered by integration tests.
// See TestAnalyzeDirIntegration for verbose output testing.

func TestAssertDepsValidWithDebug(t *testing.T) {
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[string]{
	ID: "mynode",
	Run: func(ctx context.Context) (string, error) {
		return "ok", nil
	},
}
`
	tmpDir := setupTestModule(t, map[string]string{
		"test.go": code,
	})

	// Save original debug state
	origDirDebug := AnalyzeDirDebug

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir, WithDebugTesting())

	// Debug flags should be reset after call
	if AnalyzeDirDebug != origDirDebug {
		t.Error("AnalyzeDirDebug was not reset after AssertDepsValid")
	}

	if !mock.helperCalled {
		t.Error("Helper() was not called")
	}
}

// TestAssertDepsValidVerboseWithIssues is covered by integration tests.
// See TestAnalyzeDirIntegration for error reporting testing.

func TestWithVerboseTestingOption(t *testing.T) {
	opts := &AssertOpts{}
	opt := WithVerboseTesting()
	opt(opts)

	if !opts.Verbose {
		t.Error("WithVerboseTesting should set Verbose to true")
	}
}

func TestWithDebugTestingOption(t *testing.T) {
	opts := &AssertOpts{}
	opt := WithDebugTesting()
	opt(opts)

	if !opts.Debug {
		t.Error("WithDebugTesting should set Debug to true")
	}
}

func TestAssertDepsValidNoNodes(t *testing.T) {
	// Test case where directory has no nodes - should not log success
	code := `package test

func helper() string {
	return "hello"
}
`
	tmpDir := setupTestModule(t, map[string]string{
		"test.go": code,
	})

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir)

	// No nodes means no success log (condition: len(results) > 0)
	for _, log := range mock.logs {
		if strings.Contains(log, "validated") {
			t.Error("should not log 'validated' when no nodes found")
		}
	}
}

// TestCheckDepsValid is covered by integration tests.
// See TestValidateDepsIntegration for testing of the CheckDepsValid function.
