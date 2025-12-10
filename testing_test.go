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

func TestAssertDepsValid(t *testing.T) {
	type tc struct {
		code       string
		wantErrors int
		wantFatals int
		wantLogs   int
	}

	tests := map[string]tc{
		"valid deps - logs success": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantErrors: 0,
			wantFatals: 0,
			wantLogs:   1, // Success log
		},
		"undeclared dep - reports error": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/undeclared"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[undeclared.Output](ctx)
		return v.String(), nil
	},
}
`,
			wantErrors: 2, // Main error + detailed error for undeclared
			wantFatals: 0,
			wantLogs:   0,
		},
		"unused dep - reports error": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[any]{
	ID:        "mynode",
	DependsOn: []string{"unused"},
	Run: func(ctx context.Context) (any, error) {
		return nil, nil
	},
}
`,
			wantErrors: 2, // Main error + detailed error for unused
			wantFatals: 0,
			wantLogs:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			mock := &mockT{}
			AssertDepsValid(mock, tmpDir)

			if !mock.helperCalled {
				t.Error("Helper() was not called")
			}

			if len(mock.errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(mock.errors), tt.wantErrors)
			}

			if len(mock.fatals) != tt.wantFatals {
				t.Errorf("got %d fatals, want %d", len(mock.fatals), tt.wantFatals)
			}

			if len(mock.logs) != tt.wantLogs {
				t.Errorf("got %d logs, want %d", len(mock.logs), tt.wantLogs)
			}
		})
	}
}

func TestAssertDepsValidBadDir(t *testing.T) {
	mock := &mockT{}
	AssertDepsValid(mock, "/nonexistent/path/that/should/not/exist")

	if len(mock.fatals) != 1 {
		t.Errorf("expected 1 fatal for bad directory, got %d", len(mock.fatals))
	}
}

func TestAssertDepsValidWithVerbose(t *testing.T) {
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/dep1"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{dep1.ID},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[dep1.Output](ctx)
		return v.String(), nil
	},
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir, WithVerboseTesting())

	if !mock.helperCalled {
		t.Error("Helper() was not called")
	}

	// Verbose mode should produce multiple logs
	if len(mock.logs) < 3 {
		t.Errorf("expected at least 3 logs in verbose mode, got %d", len(mock.logs))
	}

	// Should contain node summary info
	foundNodeLog := false
	for _, log := range mock.logs {
		if strings.Contains(log, "Node:") || strings.Contains(log, "DeclaredDeps") {
			foundNodeLog = true
			break
		}
	}
	if !foundNodeLog {
		t.Error("verbose output should contain node summary info")
	}
}

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
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Save original debug state
	origDirDebug := AnalyzeDirDebug
	origFileDebug := AnalyzeFileDebug

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir, WithDebugTesting())

	// Debug flags should be reset after call
	if AnalyzeDirDebug != origDirDebug {
		t.Error("AnalyzeDirDebug was not reset after AssertDepsValid")
	}
	if AnalyzeFileDebug != origFileDebug {
		t.Error("AnalyzeFileDebug was not reset after AssertDepsValid")
	}

	if !mock.helperCalled {
		t.Error("Helper() was not called")
	}
}

func TestAssertDepsValidVerboseWithIssues(t *testing.T) {
	// Test verbose output when there ARE issues
	code := `package test

import (
	"context"
	"github.com/grindlemire/graft"
	"myapp/nodes/undeclared"
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []graft.ID{"unused_dep"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[undeclared.Output](ctx)
		return v.String(), nil
	},
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir, WithVerboseTesting())

	// Should have errors for undeclared and unused
	if len(mock.errors) == 0 {
		t.Error("expected errors for issues, got none")
	}

	// Verbose logs should contain issue info
	foundUndeclaredLog := false
	foundUnusedLog := false
	for _, log := range mock.logs {
		if strings.Contains(log, "Undeclared") {
			foundUndeclaredLog = true
		}
		if strings.Contains(log, "Unused") {
			foundUnusedLog = true
		}
	}
	if !foundUndeclaredLog {
		t.Error("verbose output should contain Undeclared info when there are undeclared deps")
	}
	if !foundUnusedLog {
		t.Error("verbose output should contain Unused info when there are unused deps")
	}
}

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
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	mock := &mockT{}
	AssertDepsValid(mock, tmpDir)

	// No nodes means no success log (condition: len(results) > 0)
	for _, log := range mock.logs {
		if strings.Contains(log, "validated") {
			t.Error("should not log 'validated' when no nodes found")
		}
	}
}

func TestCheckDepsValid(t *testing.T) {
	type tc struct {
		code      string
		wantNodes int
		wantErr   bool
	}

	tests := map[string]tc{
		"valid code": {
			code: `package test

import (
	"context"
	"github.com/grindlemire/graft"
)

var node = graft.Node[any]{
	ID: "mynode",
	Run: func(ctx context.Context) (any, error) { return nil, nil },
}
`,
			wantNodes: 1,
			wantErr:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			results, err := CheckDepsValid(tmpDir)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tt.wantNodes {
				t.Errorf("got %d results, want %d", len(results), tt.wantNodes)
			}
		})
	}
}
