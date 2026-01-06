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

func TestAssertDepsValidWithVerbose(t *testing.T) {
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

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir, WithVerboseTesting())

	// Should have verbose logs (analyzing directory message)
	foundVerboseLog := false
	for _, log := range mock.logs {
		if strings.Contains(log, "analyzing") {
			foundVerboseLog = true
			break
		}
	}

	if !foundVerboseLog {
		t.Errorf("WithVerboseTesting should produce verbose logs, got logs: %v", mock.logs)
	}
}

func TestAssertDepsValidSuccess(t *testing.T) {
	// Use a real example directory that we know has valid nodes
	mock := &mockT{}
	AssertDepsValid(mock, "examples/simple")

	// Should have no errors
	if len(mock.errors) > 0 {
		t.Errorf("expected no errors, got %d: %v", len(mock.errors), mock.errors)
	}

	if len(mock.fatals) > 0 {
		t.Errorf("expected no fatals, got %d: %v", len(mock.fatals), mock.fatals)
	}

	// Should log success message when nodes are found
	foundSuccess := false
	for _, log := range mock.logs {
		if strings.Contains(log, "validated") {
			foundSuccess = true
			break
		}
	}

	if !foundSuccess {
		t.Errorf("should log success message, got logs: %v", mock.logs)
	}
}

func TestAssertDepsValidWithIssues(t *testing.T) {
	// Use an example directory that has known issues
	mock := &mockT{}
	AssertDepsValid(mock, "examples/edgecases/undeclared_multiple")

	// Should have errors for the issues found
	if len(mock.errors) == 0 {
		t.Error("expected errors for directory with issues, got none")
	}

	// Errors should mention the specific issues
	foundUndeclared := false
	for _, err := range mock.errors {
		if strings.Contains(err, "undeclared") || strings.Contains(err, "uses Dep") {
			foundUndeclared = true
			break
		}
	}

	if !foundUndeclared {
		t.Errorf("expected error messages about undeclared deps, got: %v", mock.errors)
	}
}

func TestAssertDepsValidVerboseWithIssues(t *testing.T) {
	// Test verbose mode with issues
	mock := &mockT{}
	AssertDepsValid(mock, "examples/edgecases/unused_multiple", WithVerboseTesting())

	// Should have errors
	if len(mock.errors) == 0 {
		t.Error("expected errors for directory with issues, got none")
	}

	// Should have verbose logs even with errors
	foundVerboseLog := false
	for _, log := range mock.logs {
		if strings.Contains(log, "analyzing") || strings.Contains(log, "Node:") {
			foundVerboseLog = true
			break
		}
	}

	if !foundVerboseLog {
		t.Error("verbose mode should produce logs even when there are errors")
	}

	// Should have detailed error breakdown
	foundUnused := false
	for _, err := range mock.errors {
		if strings.Contains(err, "declares") && strings.Contains(err, "never uses") {
			foundUnused = true
			break
		}
	}

	if !foundUnused {
		t.Errorf("expected detailed error messages about unused deps, got: %v", mock.errors)
	}
}
