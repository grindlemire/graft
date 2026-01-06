package typeaware

import (
	"strings"
	"testing"
)

func TestResult_HasIssues(t *testing.T) {
	tests := map[string]struct {
		result   Result
		wantHas  bool
	}{
		"no issues": {
			result: Result{
				NodeID:       "test",
				DeclaredDeps: []string{},
				UsedDeps:     []string{},
				Undeclared:   []string{},
				Unused:       []string{},
				Cycles:       [][]string{},
			},
			wantHas: false,
		},
		"has undeclared": {
			result: Result{
				NodeID:     "test",
				Undeclared: []string{"dep1"},
			},
			wantHas: true,
		},
		"has unused": {
			result: Result{
				NodeID: "test",
				Unused: []string{"dep1"},
			},
			wantHas: true,
		},
		"has cycles": {
			result: Result{
				NodeID: "test",
				Cycles: [][]string{{"a", "b", "a"}},
			},
			wantHas: true,
		},
		"has all issues": {
			result: Result{
				NodeID:     "test",
				Undeclared: []string{"dep1"},
				Unused:     []string{"dep2"},
				Cycles:     [][]string{{"a", "b", "a"}},
			},
			wantHas: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.result.HasIssues()
			if got != tt.wantHas {
				t.Errorf("HasIssues() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestResult_String(t *testing.T) {
	tests := map[string]struct {
		result       Result
		wantContains []string
	}{
		"no issues": {
			result: Result{
				NodeID:       "mynode",
				File:         "myfile.go",
				DeclaredDeps: []string{},
				UsedDeps:     []string{},
				Undeclared:   []string{},
				Unused:       []string{},
				Cycles:       [][]string{},
			},
			wantContains: []string{"mynode", "OK"},
		},
		"undeclared only": {
			result: Result{
				NodeID:     "mynode",
				File:       "myfile.go",
				Undeclared: []string{"dep1", "dep2"},
			},
			wantContains: []string{
				"mynode",
				"myfile.go",
				"undeclared deps",
				"dep1",
				"dep2",
			},
		},
		"unused only": {
			result: Result{
				NodeID: "mynode",
				File:   "myfile.go",
				Unused: []string{"dep1", "dep2"},
			},
			wantContains: []string{
				"mynode",
				"myfile.go",
				"unused deps",
				"dep1",
				"dep2",
			},
		},
		"cycles only": {
			result: Result{
				NodeID: "mynode",
				File:   "myfile.go",
				Cycles: [][]string{
					{"a", "b", "c", "a"},
				},
			},
			wantContains: []string{
				"mynode",
				"myfile.go",
				"cycles",
				"a → b → c → a",
			},
		},
		"multiple cycles": {
			result: Result{
				NodeID: "hub",
				File:   "hub.go",
				Cycles: [][]string{
					{"hub", "spoke1", "hub"},
					{"hub", "spoke2", "hub"},
				},
			},
			wantContains: []string{
				"hub",
				"hub.go",
				"cycles",
				"hub → spoke1 → hub",
				"hub → spoke2 → hub",
			},
		},
		"all issues combined": {
			result: Result{
				NodeID:     "mynode",
				File:       "myfile.go",
				Undeclared: []string{"config"},
				Unused:     []string{"db", "cache"},
				Cycles: [][]string{
					{"mynode", "other", "mynode"},
				},
			},
			wantContains: []string{
				"mynode",
				"myfile.go",
				"undeclared deps",
				"config",
				"unused deps",
				"db",
				"cache",
				"cycles",
				"mynode → other → mynode",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.result.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("String() = %q should contain %q", got, want)
				}
			}
		})
	}
}

func TestResult_String_NoIssuesDoesNotContainFile(t *testing.T) {
	result := Result{
		NodeID: "mynode",
		File:   "myfile.go",
	}

	got := result.String()

	// When there are no issues, file should not be in the output
	if strings.Contains(got, "myfile.go") {
		t.Errorf("String() for no issues should not contain file path, got: %q", got)
	}

	// Should just be "NodeID: OK"
	want := "mynode: OK"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
