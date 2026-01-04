package typeaware

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// dependencyExtractor extracts declared and used dependencies from nodes
type dependencyExtractor struct {
	mapper *typeIDMapper
	prog   *ssa.Program
	fset   *token.FileSet
}

// newDependencyExtractor creates a new dependency extractor
func newDependencyExtractor(mapper *typeIDMapper, prog *ssa.Program, fset *token.FileSet) *dependencyExtractor {
	return &dependencyExtractor{
		mapper: mapper,
		prog:   prog,
		fset:   fset,
	}
}

// ExtractDeclared extracts declared dependencies from a node's DependsOn field
func (e *dependencyExtractor) ExtractDeclared(node NodeDefinition) ([]string, error) {
	if node.DependsOn == nil {
		// No dependencies declared
		return []string{}, nil
	}

	// DependsOn is []graft.ID
	// In SSA, this is typically a slice literal or a reference to one
	// We need to trace it to find the actual ID values

	ids, err := e.extractIDsFromValue(node.DependsOn)
	if err != nil {
		// If we can't extract, return empty list
		return []string{}, nil
	}

	return ids, nil
}

// extractIDsFromValue extracts ID strings from an SSA value representing []graft.ID
func (e *dependencyExtractor) extractIDsFromValue(v ssa.Value) ([]string, error) {
	// Check if this is an Alloc (local variable)
	if alloc, ok := v.(*ssa.Alloc); ok {
		// Find stores to this alloc
		return e.extractIDsFromAlloc(alloc)
	}

	// Check if this is a slice literal
	if slice, ok := v.(*ssa.Slice); ok {
		return e.extractIDsFromSlice(slice)
	}

	// Check if this is a MakeSlice
	if makeSlice, ok := v.(*ssa.MakeSlice); ok {
		return e.extractIDsFromMakeSlice(makeSlice)
	}

	// For now, return empty if we can't handle it
	return []string{}, fmt.Errorf("cannot extract IDs from %T", v)
}

// extractIDsFromAlloc extracts IDs from an allocated slice
func (e *dependencyExtractor) extractIDsFromAlloc(alloc *ssa.Alloc) ([]string, error) {
	var ids []string

	if alloc.Referrers() == nil {
		return ids, nil
	}

	// Look for IndexAddr instructions (accessing slice elements)
	for _, instr := range *alloc.Referrers() {
		if indexAddr, ok := instr.(*ssa.IndexAddr); ok {
			// Find stores to this index
			if indexAddr.Referrers() != nil {
				for _, store := range *indexAddr.Referrers() {
					if s, ok := store.(*ssa.Store); ok {
						// Extract the ID from the stored value
						if id, err := e.extractIDFromValue(s.Val); err == nil {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}

	return ids, nil
}

// extractIDsFromSlice extracts IDs from a slice operation
func (e *dependencyExtractor) extractIDsFromSlice(slice *ssa.Slice) ([]string, error) {
	// Extract from the underlying array
	return e.extractIDsFromValue(slice.X)
}

// extractIDsFromMakeSlice extracts IDs from a MakeSlice
func (e *dependencyExtractor) extractIDsFromMakeSlice(makeSlice *ssa.MakeSlice) ([]string, error) {
	var ids []string

	// Look for stores to the slice
	if makeSlice.Referrers() == nil {
		return ids, nil
	}

	for _, instr := range *makeSlice.Referrers() {
		if indexAddr, ok := instr.(*ssa.IndexAddr); ok {
			if indexAddr.Referrers() != nil {
				for _, store := range *indexAddr.Referrers() {
					if s, ok := store.(*ssa.Store); ok {
						if id, err := e.extractIDFromValue(s.Val); err == nil {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}

	return ids, nil
}

// extractIDFromValue extracts a single ID from an SSA value
func (e *dependencyExtractor) extractIDFromValue(v ssa.Value) (string, error) {
	// Handle constants (string literals)
	if c, ok := v.(*ssa.Const); ok {
		if c.Value != nil {
			return constant.StringVal(c.Value), nil
		}
	}

	// Handle UnOp (dereferencing, address-of, etc.)
	if unop, ok := v.(*ssa.UnOp); ok {
		return e.extractIDFromValue(unop.X)
	}

	// Handle ChangeType
	if ct, ok := v.(*ssa.ChangeType); ok {
		return e.extractIDFromValue(ct.X)
	}

	// Handle Convert
	if conv, ok := v.(*ssa.Convert); ok {
		return e.extractIDFromValue(conv.X)
	}

	return "", fmt.Errorf("cannot extract ID from %T", v)
}

// ExtractUsed extracts used dependencies from a node's Run function
func (e *dependencyExtractor) ExtractUsed(node NodeDefinition) ([]string, error) {
	if node.RunFunc == nil {
		// No Run function - no dependencies can be used
		return []string{}, nil
	}

	var ids []string
	seen := make(map[string]bool)

	// Walk all instructions in the Run function
	for _, block := range node.RunFunc.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				if isGraftDepCall(call) {
					// Extract the type parameter
					depType, err := e.extractDepTypeParameter(call)
					if err != nil {
						continue
					}

					// Resolve type to ID
					id, err := e.mapper.ResolveType(depType)
					if err != nil {
						// Type not in mapping - skip this dependency
						continue
					}

					if !seen[id] {
						ids = append(ids, id)
						seen[id] = true
					}
				}
			}
		}
	}

	return ids, nil
}

// extractDepTypeParameter extracts the type parameter from a Dep[T]() call
func (e *dependencyExtractor) extractDepTypeParameter(call *ssa.Call) (types.Type, error) {
	callee := call.Common().StaticCallee()
	if callee == nil {
		return nil, fmt.Errorf("no static callee")
	}

	// Get the signature
	sig := callee.Signature
	if sig == nil {
		return nil, fmt.Errorf("no signature")
	}

	// For Dep[T], the return type is (T, error)
	// Extract T from the first return value
	results := sig.Results()
	if results.Len() < 2 {
		return nil, fmt.Errorf("unexpected result count: %d", results.Len())
	}

	// First result is T
	return results.At(0).Type(), nil
}

// AnalyzeNode analyzes a single node and produces a Result
func (e *dependencyExtractor) AnalyzeNode(node NodeDefinition) (Result, error) {
	result := Result{
		NodeID: node.ID,
		File:   node.File,
	}

	// Extract declared dependencies
	declared, err := e.ExtractDeclared(node)
	if err != nil {
		// Log but continue
		declared = []string{}
	}
	result.DeclaredDeps = declared

	// Extract used dependencies
	used, err := e.ExtractUsed(node)
	if err != nil {
		// Log but continue
		used = []string{}
	}
	result.UsedDeps = used

	// Build sets for comparison
	declaredSet := make(map[string]bool)
	for _, d := range declared {
		declaredSet[d] = true
	}

	usedSet := make(map[string]bool)
	for _, u := range used {
		usedSet[u] = true
	}

	// Find undeclared (used but not declared)
	for u := range usedSet {
		if !declaredSet[u] {
			result.Undeclared = append(result.Undeclared, u)
		}
	}

	// Find unused (declared but not used)
	for d := range declaredSet {
		if !usedSet[d] {
			result.Unused = append(result.Unused, d)
		}
	}

	return result, nil
}
