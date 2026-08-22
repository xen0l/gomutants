package mutator

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

// Mutator discovers mutation candidates in a parsed Go source file.
type Mutator interface {
	Type() MutationType
	// Description is the one-line summary shown by --list-mutators.
	// Part of the interface, not a side table, so a new mutator cannot be
	// registered without one and the listing cannot drift from the code
	// that actually runs.
	Description() string
	// Example is the short before/after shown by --list-mutators.
	Example() string
	Discover(fset *token.FileSet, file *ast.File, src []byte) []MutantCandidate
}

// CatalogEntry is one row of the mutator catalog printed by
// --list-mutators.
type CatalogEntry struct {
	Type        MutationType
	Description string
	Example     string
}

// Registry holds all registered mutators. typeSet mirrors mutators by
// MutationType so name-validation lookups (IsKnown / UnknownNames) and
// the directive parser share a single source of truth populated in
// NewRegistry.
type Registry struct {
	mutators []Mutator
	typeSet  map[string]struct{}
}

// NewRegistry creates a registry with all built-in mutators.
func NewRegistry() *Registry {
	mutators := []Mutator{
		&arithmeticBase{},
		&conditionalsBoundary{},
		&conditionalsNegation{},
		&incrementDecrement{},
		&invertNegatives{},
		&invertAssignments{},
		&invertBitwise{},
		&invertBitwiseAssignments{},
		&invertLogical{},
		&invertLoopCtrl{},
		&removeSelfAssignments{},
		&removeLogicalNot{},
		&errorfWrap{},
		&branchIf{},
		&branchElse{},
		&branchCase{},
		&expressionRemove{},
		&statementRemove{},
		&literalStep{typ: IntegerIncrement, kind: token.INT, delta: 1,
			description: "Increment integer literal", example: "42 -> 43, 0xFF -> 256"},
		&literalStep{typ: IntegerDecrement, kind: token.INT, delta: -1,
			description: "Decrement integer literal", example: "42 -> 41, 0 -> -1"},
		&literalStep{typ: FloatIncrement, kind: token.FLOAT, delta: 1,
			description: "Increment float literal", example: "1.5 -> 2.5, 0.0 -> 1.0"},
		&literalStep{typ: FloatDecrement, kind: token.FLOAT, delta: -1,
			description: "Decrement float literal", example: "1.5 -> 0.5, 1e2 -> 99.0"},
		&loopCondition{},
		&rangeBreak{},
		&returnValue{typ: ReturnErrorNil, owns: ownsError, replacement: fixed("nil"),
			description: "Swallow a propagated error", example: "return nil, err -> return nil, nil"},
		&returnValue{typ: ReturnZero, owns: ownsOther, replacement: zeroValueExpr,
			description: "Return the zero value instead", example: "return count -> return 0"},
		&returnValue{typ: ReturnTrue, owns: ownsBool, replacement: fixed("true"),
			description: "Force a boolean return true", example: "return x > 0 -> return true"},
		&returnValue{typ: ReturnFalse, owns: ownsBool, replacement: fixed("false"),
			description: "Force a boolean return false", example: "return x > 0 -> return false"},
	}
	typeSet := make(map[string]struct{}, len(mutators))
	for _, m := range mutators {
		typeSet[string(m.Type())] = struct{}{}
	}
	return &Registry{mutators: mutators, typeSet: typeSet}
}

// Mutators returns all registered mutators.
func (r *Registry) Mutators() []Mutator {
	return r.mutators
}

// Catalog returns every registered mutator's entry, sorted by type name.
// Registry order groups mutators by how they were added, which shifts over
// time; alphabetical is what makes the listing greppable and two runs
// diffable.
func (r *Registry) Catalog() []CatalogEntry {
	entries := make([]CatalogEntry, 0, len(r.mutators))
	for _, m := range r.mutators {
		entries = append(entries, CatalogEntry{
			Type:        m.Type(),
			Description: m.Description(),
			Example:     m.Example(),
		})
	}
	slices.SortFunc(entries, func(a, b CatalogEntry) int {
		return strings.Compare(string(a.Type), string(b.Type))
	})
	return entries
}

// IsKnown reports whether name matches a registered mutator type.
func (r *Registry) IsKnown(name string) bool {
	_, ok := r.typeSet[name]
	return ok
}

// UnknownNames returns the subset of names that don't match any
// registered mutator type. Used by callers that accept user-supplied
// mutator lists (--only / --disable, config file) to surface typos
// before silently filtering them out.
func (r *Registry) UnknownNames(names []string) []string {
	var unknown []string
	for _, n := range names {
		if !r.IsKnown(n) {
			unknown = append(unknown, n)
		}
	}
	return unknown
}

// EnabledMutators returns mutators filtered by the given only/disable lists.
// If only is non-empty, only those types are included.
// Otherwise, disabled types are excluded.
func (r *Registry) EnabledMutators(only, disable []string) []Mutator {
	if len(only) > 0 {
		set := make(map[string]bool, len(only))
		for _, t := range only {
			set[t] = true
		}
		var out []Mutator
		for _, m := range r.mutators {
			if set[string(m.Type())] {
				out = append(out, m)
			}
		}
		return out
	}

	set := make(map[string]bool, len(disable))
	for _, t := range disable {
		set[t] = true
	}
	var out []Mutator
	for _, m := range r.mutators {
		if !set[string(m.Type())] {
			out = append(out, m)
		}
	}
	return out
}
