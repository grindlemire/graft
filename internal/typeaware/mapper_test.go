package typeaware

import (
	"go/types"
	"strings"
	"testing"
)

func TestTypeIDMapper_BuildMapping(t *testing.T) {
	tests := map[string]struct {
		nodes   []NodeDefinition
		wantErr bool
		errMsg  string
	}{
		"empty nodes": {
			nodes:   []NodeDefinition{},
			wantErr: false,
		},
		"single node": {
			nodes: []NodeDefinition{
				{
					ID:         "config",
					OutputType: types.Typ[types.String],
				},
			},
			wantErr: false,
		},
		"multiple nodes with different types": {
			nodes: []NodeDefinition{
				{
					ID:         "config",
					OutputType: types.Typ[types.String],
				},
				{
					ID:         "port",
					OutputType: types.Typ[types.Int],
				},
			},
			wantErr: false,
		},
		"nil output type": {
			nodes: []NodeDefinition{
				{
					ID:         "bad",
					OutputType: nil,
				},
			},
			wantErr: true,
			errMsg:  "nil output type",
		},
		"type conflict - same type different nodes": {
			nodes: []NodeDefinition{
				{
					ID:         "node1",
					OutputType: types.Typ[types.String],
				},
				{
					ID:         "node2",
					OutputType: types.Typ[types.String],
				},
			},
			wantErr: true,
			errMsg:  "type conflict",
		},
		"duplicate registration same node - OK": {
			nodes: []NodeDefinition{
				{
					ID:         "config",
					OutputType: types.Typ[types.String],
				},
				{
					ID:         "config",
					OutputType: types.Typ[types.String],
				},
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mapper := newTypeIDMapper()
			err := mapper.BuildMapping(tt.nodes)

			if tt.wantErr {
				if err == nil {
					t.Errorf("BuildMapping() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("BuildMapping() error = %q, should contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("BuildMapping() unexpected error: %v", err)
			}
		})
	}
}

func TestTypeIDMapper_ResolveType(t *testing.T) {
	mapper := newTypeIDMapper()

	// Build mapping
	nodes := []NodeDefinition{
		{ID: "str", OutputType: types.Typ[types.String]},
		{ID: "num", OutputType: types.Typ[types.Int]},
	}

	err := mapper.BuildMapping(nodes)
	if err != nil {
		t.Fatalf("BuildMapping() failed: %v", err)
	}

	tests := map[string]struct {
		typ     types.Type
		wantID  string
		wantErr bool
	}{
		"string type": {
			typ:     types.Typ[types.String],
			wantID:  "str",
			wantErr: false,
		},
		"int type": {
			typ:     types.Typ[types.Int],
			wantID:  "num",
			wantErr: false,
		},
		"unregistered type": {
			typ:     types.Typ[types.Bool],
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := mapper.ResolveType(tt.typ)

			if tt.wantErr {
				if err == nil {
					t.Error("ResolveType() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveType() unexpected error: %v", err)
				return
			}

			if got != tt.wantID {
				t.Errorf("ResolveType() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestTypeIDMapper_GetType(t *testing.T) {
	mapper := newTypeIDMapper()

	// Build mapping
	nodes := []NodeDefinition{
		{ID: "str", OutputType: types.Typ[types.String]},
		{ID: "num", OutputType: types.Typ[types.Int]},
	}

	err := mapper.BuildMapping(nodes)
	if err != nil {
		t.Fatalf("BuildMapping() failed: %v", err)
	}

	tests := map[string]struct {
		nodeID   string
		wantType types.Type
		wantErr  bool
	}{
		"str node": {
			nodeID:   "str",
			wantType: types.Typ[types.String],
			wantErr:  false,
		},
		"num node": {
			nodeID:   "num",
			wantType: types.Typ[types.Int],
			wantErr:  false,
		},
		"unknown node": {
			nodeID:  "unknown",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := mapper.GetType(tt.nodeID)

			if tt.wantErr {
				if err == nil {
					t.Error("GetType() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetType() unexpected error: %v", err)
				return
			}

			if got != tt.wantType {
				t.Errorf("GetType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestTypeIDMapper_HasType(t *testing.T) {
	mapper := newTypeIDMapper()

	// Build mapping
	nodes := []NodeDefinition{
		{ID: "str", OutputType: types.Typ[types.String]},
		{ID: "num", OutputType: types.Typ[types.Int]},
	}

	err := mapper.BuildMapping(nodes)
	if err != nil {
		t.Fatalf("BuildMapping() failed: %v", err)
	}

	tests := map[string]struct {
		typ  types.Type
		want bool
	}{
		"registered string": {
			typ:  types.Typ[types.String],
			want: true,
		},
		"registered int": {
			typ:  types.Typ[types.Int],
			want: true,
		},
		"unregistered bool": {
			typ:  types.Typ[types.Bool],
			want: false,
		},
		"unregistered float64": {
			typ:  types.Typ[types.Float64],
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := mapper.HasType(tt.typ)
			if got != tt.want {
				t.Errorf("HasType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeIDMapper_Size(t *testing.T) {
	tests := map[string]struct {
		nodes    []NodeDefinition
		wantSize int
	}{
		"empty": {
			nodes:    []NodeDefinition{},
			wantSize: 0,
		},
		"single node": {
			nodes: []NodeDefinition{
				{ID: "str", OutputType: types.Typ[types.String]},
			},
			wantSize: 1,
		},
		"multiple nodes": {
			nodes: []NodeDefinition{
				{ID: "str", OutputType: types.Typ[types.String]},
				{ID: "num", OutputType: types.Typ[types.Int]},
				{ID: "flag", OutputType: types.Typ[types.Bool]},
			},
			wantSize: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mapper := newTypeIDMapper()
			err := mapper.BuildMapping(tt.nodes)
			if err != nil {
				t.Fatalf("BuildMapping() failed: %v", err)
			}

			got := mapper.Size()
			if got != tt.wantSize {
				t.Errorf("Size() = %d, want %d", got, tt.wantSize)
			}
		})
	}
}

func TestTypeIDMapper_NormalizeType(t *testing.T) {
	mapper := newTypeIDMapper()

	tests := map[string]struct {
		typ         types.Type
		wantContain string
	}{
		"basic string": {
			typ:         types.Typ[types.String],
			wantContain: "string",
		},
		"basic int": {
			typ:         types.Typ[types.Int],
			wantContain: "int",
		},
		"pointer to string": {
			typ:         types.NewPointer(types.Typ[types.String]),
			wantContain: "*string",
		},
		"slice of int": {
			typ:         types.NewSlice(types.Typ[types.Int]),
			wantContain: "[]int",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := mapper.normalizeType(tt.typ)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("normalizeType() = %q, should contain %q", got, tt.wantContain)
			}
		})
	}
}

func TestTypeIDMapper_TypeKey(t *testing.T) {
	mapper := newTypeIDMapper()

	// TypeKey should produce the same result for the same type
	typ := types.Typ[types.String]
	key1 := mapper.typeKey(typ)
	key2 := mapper.typeKey(typ)

	if key1 != key2 {
		t.Errorf("typeKey() should be consistent: %q != %q", key1, key2)
	}

	// Different types should produce different keys
	typ2 := types.Typ[types.Int]
	key3 := mapper.typeKey(typ2)

	if key1 == key3 {
		t.Errorf("typeKey() should be different for different types: %q == %q", key1, key3)
	}
}
