package nogoods

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

func fullProblem() Problem {
	return Problem{
		Version:      ProblemVersion,
		ColorAliases: []string{"red", "green", "blue", "gold"},
		Variables: []Variable{
			{Alias: "anchor", Domain: []int{0, 1}},
			{Alias: "left", Domain: []int{0, 2}},
			{Alias: "right", Domain: []int{0, 2}},
		},
		Edges: []Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}},
	}
}

func fullBinding() Binding {
	return Binding{Anchor: 0, X: 1, Y: 2, Blocked: 0, Escape: 1, Only: 2}
}

func TestCanonicalProblemRoundTripAndSemanticAliases(t *testing.T) {
	p := fullProblem()
	encoded, err := p.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseProblem(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed.Edges, p.Edges) {
		t.Fatalf("round trip edges = %#v", parsed.Edges)
	}
	key, err := p.SemanticKey()
	if err != nil {
		t.Fatal(err)
	}
	renamed := p
	renamed.ColorAliases = []string{"c0", "c1", "c2", "c3"}
	renamed.Variables = slices.Clone(p.Variables)
	for index := range renamed.Variables {
		renamed.Variables[index].Alias = "v" + string(rune('a'+index))
	}
	renamedKey, err := renamed.SemanticKey()
	if err != nil {
		t.Fatal(err)
	}
	if key != renamedKey {
		t.Fatalf("semantic key changed under aliases: %s != %s", key, renamedKey)
	}
	var pretty any
	if err := json.Unmarshal(encoded, &pretty); err != nil {
		t.Fatal(err)
	}
	prettyBytes, _ := json.MarshalIndent(pretty, "", "  ")
	if _, err := ParseProblem(prettyBytes); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-canonical JSON error = %v", err)
	}
}

func TestProblemRejectsNonCanonicalAndInvalidForms(t *testing.T) {
	tests := map[string]func(*Problem){
		"version":           func(p *Problem) { p.Version = "v2" },
		"duplicate color":   func(p *Problem) { p.ColorAliases[1] = p.ColorAliases[0] },
		"duplicate alias":   func(p *Problem) { p.Variables[0].Alias = p.ColorAliases[0] },
		"empty domain":      func(p *Problem) { p.Variables[0].Domain = nil },
		"domain order":      func(p *Problem) { p.Variables[0].Domain = []int{1, 0} },
		"domain duplicate":  func(p *Problem) { p.Variables[0].Domain = []int{0, 0} },
		"self edge":         func(p *Problem) { p.Edges[0] = Edge{Left: 0, Right: 0} },
		"reversed edge":     func(p *Problem) { p.Edges[0] = Edge{Left: 1, Right: 0} },
		"duplicate edge":    func(p *Problem) { p.Edges[1] = p.Edges[0] },
		"assignment order":  func(p *Problem) { p.Assignment = []Literal{{Variable: 2, Color: 2}, {Variable: 1, Color: 0}} },
		"assignment domain": func(p *Problem) { p.Assignment = []Literal{{Variable: 0, Color: 3}} },
		"assignment edge":   func(p *Problem) { p.Assignment = []Literal{{Variable: 0, Color: 0}, {Variable: 1, Color: 0}} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			p := fullProblem()
			mutate(&p)
			if err := p.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate error = %v", err)
			}
		})
	}
}

func TestBoundedFactsExtensionAndEvaluation(t *testing.T) {
	p := fullProblem()
	if !p.DomainContains(1, 2) || p.DomainContains(1, 3) || !p.EdgePresent(2, 0) || p.EdgePresent(0, 0) {
		t.Fatal("bounded fact query mismatch")
	}
	extended, err := p.Extend(Literal{Variable: 0, Color: 0})
	if err != nil || len(extended.Assignment) != 1 {
		t.Fatalf("Extend = %#v, %v", extended, err)
	}
	if _, err := extended.Extend(Literal{Variable: 1, Color: 0}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("inconsistent extension error = %v", err)
	}
	violations, err := p.Evaluate([]Literal{{Variable: 0, Color: 0}, {Variable: 1, Color: 0}})
	if err != nil || len(violations) != 1 || violations[0].Kind != "inequality" {
		t.Fatalf("Evaluate = %#v, %v", violations, err)
	}
}

func TestAllMasksAndUniqueExactMask(t *testing.T) {
	p := fullProblem()
	binding := fullBinding()
	decision := Literal{Variable: 0, Color: 0}
	if !GuardMatches(p, decision, binding) {
		t.Fatal("full binding did not satisfy guard")
	}
	completion := Completion{XColor: 2, YColor: 2}
	for mask := Mask(0); mask <= FullMask; mask++ {
		if !MaskMatches(p, mask, binding) {
			t.Fatalf("mask %d did not match full graph", mask)
		}
		conflict, err := EvaluateCompletion(p, mask, binding, completion)
		if err != nil {
			t.Fatal(err)
		}
		if conflict != (mask&(1<<2) != 0) {
			t.Fatalf("mask %d completion conflict = %v", mask, conflict)
		}
	}
	for bit := 0; bit < 3; bit++ {
		near := p
		near.Edges = slices.DeleteFunc(slices.Clone(p.Edges), func(edge Edge) bool {
			pairs := [3]Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}}
			return edge == pairs[bit]
		})
		if MaskMatches(near, FullMask, binding) {
			t.Fatalf("full mask matched missing bit %d", bit)
		}
	}
}

func TestRefinementAndCompleteGuard(t *testing.T) {
	seen := map[Mask]bool{0: true}
	frontier := []Mask{0}
	for len(frontier) > 0 {
		mask := frontier[0]
		frontier = frontier[1:]
		for bit := 0; bit < 3; bit++ {
			next, err := RefineMask(mask, bit)
			if err != nil {
				continue
			}
			if !seen[next] {
				seen[next] = true
				frontier = append(frontier, next)
			}
		}
	}
	if len(seen) != 8 {
		t.Fatalf("refinement reached %d masks", len(seen))
	}
	p := fullProblem()
	b := fullBinding()
	decision := Literal{Variable: 0, Color: 0}
	bad := []Binding{
		{Anchor: 0, X: 2, Y: 1, Blocked: 0, Escape: 1, Only: 2},
		{Anchor: 0, X: 1, Y: 2, Blocked: 0, Escape: 1, Only: 1},
		{Anchor: 0, X: 1, Y: 2, Blocked: 0, Escape: 0, Only: 2},
	}
	for _, candidate := range bad {
		if GuardMatches(p, decision, candidate) {
			t.Fatalf("bad guard matched: %#v", candidate)
		}
	}
	if !GuardMatches(p, decision, b) {
		t.Fatal("valid guard rejected")
	}
}

func TestCertificateRecordIsOneRecordNotSetValidation(t *testing.T) {
	p := fullProblem()
	record := CertificateRecord{
		SchemaVersion: SchemaVersion,
		Mask:          FullMask,
		Binding:       fullBinding(),
		Decision:      Literal{Variable: 0, Color: 0},
		Completion:    Completion{XColor: 2, YColor: 2},
		Conflict:      true,
	}
	if err := ValidateCertificateRecord(p, record); err != nil {
		t.Fatal(err)
	}
	record.Conflict = false
	if err := ValidateCertificateRecord(p, record); !errors.Is(err, ErrInvalid) {
		t.Fatalf("corrupt record error = %v", err)
	}
}
