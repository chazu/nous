// Package mutate implements token-level mutation of DSL programs.
// This is the nous equivalent of EURISKO's generalization/specialization
// engine, operating on stack-based program strings rather than Lisp S-expressions.
package mutate

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

// Op describes what mutation was applied.
type Op struct {
	Kind string // "swap", "delete", "insert", "replace", "widen", "narrow", "duplicate"
	Pos  int
	From string
	To   string
}

// Mutator applies random mutations to DSL program strings.
type Mutator struct {
	rng   *rand.Rand
	store *unit.Store
}

// New creates a mutator with the given random source and unit store
// (used for generalization/specialization lookups).
func New(rng *rand.Rand, store *unit.Store) *Mutator {
	return &Mutator{rng: rng, store: store}
}

// Mutate applies a single random mutation to a program string.
// Returns the mutated program and a description of the mutation.
// Returns "", nil if mutation is not possible.
func (m *Mutator) Mutate(program string) (string, *Op) {
	tokens := dsl.Tokenize(program)
	if len(tokens) < 2 {
		return "", nil
	}

	// Pick a mutation type
	mutations := []func([]dsl.Token) ([]dsl.Token, *Op){
		m.swapAdj,
		m.deleteToken,
		m.insertToken,
		m.widenNumeric,
		m.narrowNumeric,
		m.replaceUnitRef,
		m.duplicateSeq,
	}

	// Try up to 5 times to get a valid mutation
	for attempts := 0; attempts < 5; attempts++ {
		fn := mutations[m.rng.Intn(len(mutations))]
		result, op := fn(tokens)
		if op != nil {
			return tokensToString(result), op
		}
	}
	return "", nil
}

// swapAdj swaps two adjacent tokens.
func (m *Mutator) swapAdj(tokens []dsl.Token) ([]dsl.Token, *Op) {
	if len(tokens) < 2 {
		return nil, nil
	}
	out := cloneTokens(tokens)
	i := m.rng.Intn(len(out) - 1)
	// Don't swap control flow keywords — that breaks structure
	if isControl(out[i]) || isControl(out[i+1]) {
		return nil, nil
	}
	out[i], out[i+1] = out[i+1], out[i]
	return out, &Op{Kind: "swap", Pos: i, From: out[i+1].Text, To: out[i].Text}
}

// deleteToken removes a non-structural token.
func (m *Mutator) deleteToken(tokens []dsl.Token) ([]dsl.Token, *Op) {
	// Find deletable positions (not control flow, not the only token)
	var candidates []int
	for i, t := range tokens {
		if !isControl(t) && !isVarOp(t) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	i := candidates[m.rng.Intn(len(candidates))]
	out := append(cloneTokens(tokens[:i]), tokens[i+1:]...)
	return out, &Op{Kind: "delete", Pos: i, From: tokens[i].Text}
}

// insertToken inserts a random builtin word or small literal.
func (m *Mutator) insertToken(tokens []dsl.Token) ([]dsl.Token, *Op) {
	insertables := []dsl.Token{
		{Kind: dsl.TokWord, Text: "dup"},
		{Kind: dsl.TokWord, Text: "drop"},
		{Kind: dsl.TokWord, Text: "swap"},
		{Kind: dsl.TokWord, Text: "not"},
		{Kind: dsl.TokWord, Text: "and"},
		{Kind: dsl.TokWord, Text: "or"},
		{Kind: dsl.TokInt, Text: "0", Int: 0},
		{Kind: dsl.TokInt, Text: "1", Int: 1},
		{Kind: dsl.TokInt, Text: "2", Int: 2},
		{Kind: dsl.TokWord, Text: "nil"},
		{Kind: dsl.TokWord, Text: "true"},
		{Kind: dsl.TokWord, Text: "false"},
	}

	pos := m.rng.Intn(len(tokens) + 1)
	tok := insertables[m.rng.Intn(len(insertables))]

	out := make([]dsl.Token, 0, len(tokens)+1)
	out = append(out, tokens[:pos]...)
	out = append(out, tok)
	out = append(out, tokens[pos:]...)

	return out, &Op{Kind: "insert", Pos: pos, To: tok.Text}
}

// widenNumeric increases a numeric literal (generalization).
func (m *Mutator) widenNumeric(tokens []dsl.Token) ([]dsl.Token, *Op) {
	var candidates []int
	for i, t := range tokens {
		if t.Kind == dsl.TokInt && t.Int > 0 {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	i := candidates[m.rng.Intn(len(candidates))]
	out := cloneTokens(tokens)
	old := out[i].Int
	delta := m.rng.Intn(old/2+1) + 1
	newVal := old + delta
	out[i] = dsl.Token{Kind: dsl.TokInt, Text: fmt.Sprintf("%d", newVal), Int: newVal}
	return out, &Op{Kind: "widen", Pos: i, From: fmt.Sprintf("%d", old), To: fmt.Sprintf("%d", newVal)}
}

// narrowNumeric decreases a numeric literal (specialization).
func (m *Mutator) narrowNumeric(tokens []dsl.Token) ([]dsl.Token, *Op) {
	var candidates []int
	for i, t := range tokens {
		if t.Kind == dsl.TokInt && t.Int > 1 {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	i := candidates[m.rng.Intn(len(candidates))]
	out := cloneTokens(tokens)
	old := out[i].Int
	delta := m.rng.Intn(old/2) + 1
	newVal := old - delta
	if newVal < 0 {
		newVal = 0
	}
	out[i] = dsl.Token{Kind: dsl.TokInt, Text: fmt.Sprintf("%d", newVal), Int: newVal}
	return out, &Op{Kind: "narrow", Pos: i, From: fmt.Sprintf("%d", old), To: fmt.Sprintf("%d", newVal)}
}

// replaceUnitRef replaces a quoted string that names a unit with one of its
// generalizations or specializations.
func (m *Mutator) replaceUnitRef(tokens []dsl.Token) ([]dsl.Token, *Op) {
	if m.store == nil {
		return nil, nil
	}
	var candidates []int
	for i, t := range tokens {
		if t.Kind == dsl.TokString && m.store.Has(t.Text) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	i := candidates[m.rng.Intn(len(candidates))]
	name := tokens[i].Text

	// Collect alternatives: generalizations + specializations
	var alts []string
	alts = append(alts, m.store.Generalizations(name)...)
	alts = append(alts, m.store.Specializations(name)...)
	// Also try isA parents
	u := m.store.Get(name)
	if u != nil {
		alts = append(alts, u.GetStrings("isA")...)
	}
	if len(alts) == 0 {
		return nil, nil
	}

	out := cloneTokens(tokens)
	newName := alts[m.rng.Intn(len(alts))]
	out[i] = dsl.Token{Kind: dsl.TokString, Text: newName}
	return out, &Op{Kind: "replace", Pos: i, From: name, To: newName}
}

// duplicateSeq copies a short subsequence to a nearby position.
func (m *Mutator) duplicateSeq(tokens []dsl.Token) ([]dsl.Token, *Op) {
	if len(tokens) < 4 {
		return nil, nil
	}
	// Pick 1-3 tokens to duplicate
	seqLen := m.rng.Intn(3) + 1
	if seqLen > len(tokens)-1 {
		seqLen = len(tokens) - 1
	}
	start := m.rng.Intn(len(tokens) - seqLen)

	// Check that we're not duplicating control flow
	for j := start; j < start+seqLen; j++ {
		if isControl(tokens[j]) {
			return nil, nil
		}
	}

	// Insert the copy right after the original
	insertAt := start + seqLen
	out := make([]dsl.Token, 0, len(tokens)+seqLen)
	out = append(out, tokens[:insertAt]...)
	out = append(out, tokens[start:start+seqLen]...)
	out = append(out, tokens[insertAt:]...)

	seq := tokensToString(tokens[start : start+seqLen])
	return out, &Op{Kind: "duplicate", Pos: start, From: seq, To: seq}
}

// Validate checks if a mutated program is syntactically valid by
// trial-executing it on a minimal VM. Returns true if it runs without
// crashing (the result doesn't matter — only structural validity).
func Validate(program string, store *unit.Store) bool {
	vm := dsl.NewVM(store, nil)
	vm.Out = devNull{}

	// Set dummy env vars that heuristic programs expect
	vm.SetEnv("ArgU", dsl.StringVal("_test"))
	vm.SetEnv("CurUnit", dsl.StringVal("_test"))
	vm.SetEnv("CurSlot", dsl.StringVal("_test"))
	vm.SetEnv("CurPri", dsl.IntVal(500))

	_, err := vm.Execute(program)
	// AbortError is valid behavior, not a bug
	if err != nil && !dsl.IsAbort(err) {
		return false
	}
	return true
}

// Helper functions

func isControl(t dsl.Token) bool {
	if t.Kind != dsl.TokWord {
		return false
	}
	switch t.Text {
	case "if", "then", "else", "each", "end":
		return true
	}
	return false
}

func isVarOp(t dsl.Token) bool {
	if t.Kind != dsl.TokWord {
		return false
	}
	return t.Text == "!" || t.Text == "@"
}

func cloneTokens(tokens []dsl.Token) []dsl.Token {
	out := make([]dsl.Token, len(tokens))
	copy(out, tokens)
	return out
}

func tokensToString(tokens []dsl.Token) string {
	var parts []string
	for _, t := range tokens {
		switch t.Kind {
		case dsl.TokString:
			parts = append(parts, fmt.Sprintf("%q", t.Text))
		default:
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}

// devNull discards all writes.
type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
