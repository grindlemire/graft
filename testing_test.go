package graft

import (
	"os"
	"path/filepath"
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{"dep1"},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "dep1")
		return v, nil
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
)

var node = graft.Node[string]{
	ID:        "mynode",
	DependsOn: []string{},
	Run: func(ctx context.Context) (string, error) {
		v, _ := graft.Dep[string](ctx, "undeclared")
		return v, nil
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
