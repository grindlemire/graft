package graft

import (
	"context"
	"testing"
)

func TestDep(t *testing.T) {
	type tc struct {
		ctx       context.Context
		nodeID    string
		wantVal   string
		wantErr   bool
		errSubstr string
	}

	// Setup contexts for tests
	ctxWithResults := withResults(context.Background(), results{
		"stringNode": "hello",
		"intNode":    42,
		"nilNode":    nil,
	})

	tests := map[string]tc{
		"success - string value": {
			ctx:     ctxWithResults,
			nodeID:  "stringNode",
			wantVal: "hello",
			wantErr: false,
		},
		"no results in context": {
			ctx:       context.Background(),
			nodeID:    "anyNode",
			wantErr:   true,
			errSubstr: "no results in context",
		},
		"dependency not found": {
			ctx:       ctxWithResults,
			nodeID:    "missingNode",
			wantErr:   true,
			errSubstr: `dependency "missingNode" not found`,
		},
		"wrong type - want string got int": {
			ctx:       ctxWithResults,
			nodeID:    "intNode",
			wantErr:   true,
			errSubstr: "wrong type",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Dep[string](tt.ctx, tt.nodeID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !containsSubstr(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantVal {
				t.Errorf("got %q, want %q", got, tt.wantVal)
			}
		})
	}
}

func TestDepGenericTypes(t *testing.T) {
	type customStruct struct {
		Name  string
		Value int
	}

	type tc struct {
		results   results
		nodeID    string
		checkFunc func(t *testing.T, got any)
	}

	tests := map[string]tc{
		"int type": {
			results: results{"node": 42},
			nodeID:  "node",
			checkFunc: func(t *testing.T, got any) {
				if got.(int) != 42 {
					t.Errorf("got %v, want 42", got)
				}
			},
		},
		"slice type": {
			results: results{"node": []string{"a", "b", "c"}},
			nodeID:  "node",
			checkFunc: func(t *testing.T, got any) {
				slice := got.([]string)
				if len(slice) != 3 || slice[0] != "a" {
					t.Errorf("got %v, want [a b c]", got)
				}
			},
		},
		"struct type": {
			results: results{"node": customStruct{Name: "test", Value: 100}},
			nodeID:  "node",
			checkFunc: func(t *testing.T, got any) {
				s := got.(customStruct)
				if s.Name != "test" || s.Value != 100 {
					t.Errorf("got %v, want {test 100}", got)
				}
			},
		},
		"pointer type": {
			results: results{"node": &customStruct{Name: "ptr", Value: 200}},
			nodeID:  "node",
			checkFunc: func(t *testing.T, got any) {
				s := got.(*customStruct)
				if s.Name != "ptr" || s.Value != 200 {
					t.Errorf("got %v, want &{ptr 200}", got)
				}
			},
		},
		"map type": {
			results: results{"node": map[string]int{"key": 1}},
			nodeID:  "node",
			checkFunc: func(t *testing.T, got any) {
				m := got.(map[string]int)
				if m["key"] != 1 {
					t.Errorf("got %v, want map[key:1]", got)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := withResults(context.Background(), tt.results)

			switch tt.nodeID {
			case "node":
				// Type-specific retrieval based on test case
				switch tt.results["node"].(type) {
				case int:
					got, err := Dep[int](ctx, tt.nodeID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					tt.checkFunc(t, got)
				case []string:
					got, err := Dep[[]string](ctx, tt.nodeID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					tt.checkFunc(t, got)
				case customStruct:
					got, err := Dep[customStruct](ctx, tt.nodeID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					tt.checkFunc(t, got)
				case *customStruct:
					got, err := Dep[*customStruct](ctx, tt.nodeID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					tt.checkFunc(t, got)
				case map[string]int:
					got, err := Dep[map[string]int](ctx, tt.nodeID)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					tt.checkFunc(t, got)
				}
			}
		})
	}
}

func TestWithResultsAndGetResults(t *testing.T) {
	type tc struct {
		inputResults results
		wantOK       bool
	}

	tests := map[string]tc{
		"valid results": {
			inputResults: results{"a": 1, "b": "two"},
			wantOK:       true,
		},
		"empty results": {
			inputResults: results{},
			wantOK:       true,
		},
		"nil results": {
			inputResults: nil,
			wantOK:       true, // nil map is still a valid results type
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := withResults(context.Background(), tt.inputResults)
			got, ok := getResults(ctx)
			if ok != tt.wantOK {
				t.Errorf("getResults ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && len(got) != len(tt.inputResults) {
				t.Errorf("got %d results, want %d", len(got), len(tt.inputResults))
			}
		})
	}
}

func TestGetResultsWithoutContext(t *testing.T) {
	_, ok := getResults(context.Background())
	if ok {
		t.Error("expected ok=false for context without results")
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
