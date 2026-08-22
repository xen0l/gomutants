package mutator

import (
	"go/ast"
	"go/token"
)

// literalStep is the single Mutator implementation behind the four
// numeric-literal stepping mutators (INTEGER_INCREMENT / INTEGER_DECREMENT
// / FLOAT_INCREMENT / FLOAT_DECREMENT). Each registered instance carries
// the (MutationType, target literal kind, ±1 delta) tuple it stamps onto
// its candidates; the actual parse/format/skip logic lives in the shared
// numericLiteralCandidates helper.
//
// Collapsing the four near-identical wrappers into one type keeps the
// behaviour 1:1 with separate structs but removes the duplication
// SonarCloud's new-code gate flagged.
type literalStep struct {
	typ  MutationType
	kind token.Token
	// delta is the signed step applied to the literal's parsed value. Must
	// be non-zero: numericLiteralCandidates's no-phantom-collision argument
	// relies on the canonical re-formatted result differing from the source
	// span, which only holds when delta ≠ 0. All registry entries pass ±1.
	delta int
	// description and example are this instance's --list-mutators entry.
	// Per-instance like typ, because one literalStep type backs four
	// distinct mutators that each need their own wording.
	description string
	example     string
}

func (m *literalStep) Type() MutationType  { return m.typ }
func (m *literalStep) Description() string { return m.description }
func (m *literalStep) Example() string     { return m.example }

func (m *literalStep) Discover(fset *token.FileSet, file *ast.File, src []byte) []MutantCandidate {
	return numericLiteralCandidates(fset, file, m.typ, m.kind, m.delta)
}
