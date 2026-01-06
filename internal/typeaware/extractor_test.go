package typeaware

import (
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestDependencyExtractor_ExtractDeclared_NilDependsOn(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	extractor := newDependencyExtractor(mapper, prog, &token.FileSet{})

	node := NodeDefinition{
		ID:         "test",
		DependsOn:  nil, // No dependencies
		OutputType: types.Typ[types.String],
	}

	deps, err := extractor.ExtractDeclared(node)
	if err != nil {
		t.Errorf("ExtractDeclared() unexpected error: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("ExtractDeclared() with nil DependsOn should return empty slice, got %v", deps)
	}
}

func TestDependencyExtractor_ExtractUsed_NilRunFunc(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	extractor := newDependencyExtractor(mapper, prog, &token.FileSet{})

	node := NodeDefinition{
		ID:         "test",
		RunFunc:    nil, // No Run function
		OutputType: types.Typ[types.String],
	}

	deps, err := extractor.ExtractUsed(node)
	if err != nil {
		t.Errorf("ExtractUsed() unexpected error: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("ExtractUsed() with nil RunFunc should return empty slice, got %v", deps)
	}
}

func TestDependencyExtractor_AnalyzeNode_Basic(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	extractor := newDependencyExtractor(mapper, prog, &token.FileSet{})

	tests := map[string]struct {
		node       NodeDefinition
		wantResult Result
	}{
		"node with no dependencies": {
			node: NodeDefinition{
				ID:         "standalone",
				File:       "standalone.go",
				DependsOn:  nil,
				RunFunc:    nil,
				OutputType: types.Typ[types.String],
			},
			wantResult: Result{
				NodeID:       "standalone",
				File:         "standalone.go",
				DeclaredDeps: []string{},
				UsedDeps:     []string{},
				Undeclared:   nil,
				Unused:       nil,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := extractor.AnalyzeNode(tt.node)
			if err != nil {
				t.Errorf("AnalyzeNode() unexpected error: %v", err)
			}

			if result.NodeID != tt.wantResult.NodeID {
				t.Errorf("NodeID = %q, want %q", result.NodeID, tt.wantResult.NodeID)
			}

			if result.File != tt.wantResult.File {
				t.Errorf("File = %q, want %q", result.File, tt.wantResult.File)
			}

			if len(result.DeclaredDeps) != len(tt.wantResult.DeclaredDeps) {
				t.Errorf("DeclaredDeps length = %d, want %d", len(result.DeclaredDeps), len(tt.wantResult.DeclaredDeps))
			}

			if len(result.UsedDeps) != len(tt.wantResult.UsedDeps) {
				t.Errorf("UsedDeps length = %d, want %d", len(result.UsedDeps), len(tt.wantResult.UsedDeps))
			}
		})
	}
}

func TestDependencyExtractor_ExtractIDFromValue_Errors(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	extractor := newDependencyExtractor(mapper, prog, &token.FileSet{})

	tests := map[string]struct {
		value   ssa.Value
		wantErr bool
	}{
		"nil value": {
			value:   nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := extractor.extractIDFromValue(tt.value)

			if tt.wantErr && err == nil {
				t.Error("extractIDFromValue() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("extractIDFromValue() unexpected error: %v", err)
			}
		})
	}
}

func TestDependencyExtractor_ExtractIDsFromValue_Errors(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	extractor := newDependencyExtractor(mapper, prog, &token.FileSet{})

	tests := map[string]struct {
		value   ssa.Value
		wantErr bool
	}{
		"nil value": {
			value:   nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := extractor.extractIDsFromValue(tt.value)

			if tt.wantErr && err == nil {
				t.Error("extractIDsFromValue() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("extractIDsFromValue() unexpected error: %v", err)
			}
		})
	}
}

func TestNewDependencyExtractor(t *testing.T) {
	mapper := newTypeIDMapper()
	prog := ssa.NewProgram(&token.FileSet{}, ssa.SanityCheckFunctions)
	fset := &token.FileSet{}

	extractor := newDependencyExtractor(mapper, prog, fset)

	if extractor == nil {
		t.Error("newDependencyExtractor() returned nil")
	}

	if extractor.mapper != mapper {
		t.Error("extractor.mapper not set correctly")
	}

	if extractor.prog != prog {
		t.Error("extractor.prog not set correctly")
	}

	if extractor.fset != fset {
		t.Error("extractor.fset not set correctly")
	}
}
