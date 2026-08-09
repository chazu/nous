package dsl

import (
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func nogoodTestStore() *unit.Store {
	store := protocolTestStore("")
	marker := unit.New("ConstraintNogoodVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "nogoods")
	store.Put(marker)
	return store
}

func nogoodTestProblem(t *testing.T) string {
	t.Helper()
	problem := nogoods.Problem{
		Version:      nogoods.ProblemVersion,
		ColorAliases: []string{"blocked", "escape", "only"},
		Variables: []nogoods.Variable{
			{Alias: "a", Domain: []int{0, 1}},
			{Alias: "x", Domain: []int{0, 2}},
			{Alias: "y", Domain: []int{0, 2}},
		},
		Edges:      []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}},
		Assignment: []nogoods.Literal{},
	}
	data, err := problem.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestNogoodWordsAreVocabularyScoped(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(`0 0 ng-refine-mask`); err == nil || !strings.Contains(err.Error(), "unknown word") {
		t.Fatalf("base VM exposed nogood word: %v", err)
	}
	vm := NewVM(nogoodTestStore(), agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
	value, err := vm.Execute(`0 2 ng-refine-mask`)
	if err != nil || value.AsInt() != 4 {
		t.Fatalf("refine = %v, %v", value, err)
	}
}

func TestNogoodBoundedSemanticWords(t *testing.T) {
	vm := NewVM(nogoodTestStore(), agenda.New(), nil)
	input := unit.New("InputProblem")
	input.Set("data", nogoodTestProblem(t))
	vm.Store.Put(input)
	problem := `"InputProblem" "data" get-slot`
	checks := []struct {
		program string
		want    bool
	}{
		{problem + ` ng-problem-valid?`, true},
		{problem + ` 1 2 ng-domain-has?`, true},
		{problem + ` 0 2 ng-edge-has?`, true},
		{problem + ` 0 0 0 1 2 0 1 2 ng-guard-matches?`, true},
		{problem + ` 7 0 1 2 0 1 2 ng-mask-matches?`, true},
		{problem + ` 7 0 1 2 0 1 2 2 2 ng-completion-conflicts?`, true},
		{problem + ` 7 0 0 0 1 2 0 1 2 2 2 true ng-certificate-valid?`, true},
	}
	for _, check := range checks {
		value, err := vm.Execute(check.program)
		if err != nil || value.Kind() != VBool || value.AsBool() != check.want {
			t.Fatalf("%s = %v, %v; want %v", check.program, value, err, check.want)
		}
	}
	key, err := vm.Execute(problem + ` ng-semantic-key`)
	if err != nil || len(key.AsString()) != 64 {
		t.Fatalf("semantic key = %v, %v", key, err)
	}
	name, err := vm.Execute(`"candidate" "mask:7" ng-artifact-name`)
	if err != nil || !strings.HasPrefix(name.AsString(), "NG.candidate.") {
		t.Fatalf("artifact name = %v, %v", name, err)
	}
}

func TestNogoodWordsRejectMalformedTypesAndObjects(t *testing.T) {
	vm := NewVM(nogoodTestStore(), agenda.New(), nil)
	for _, program := range []string{`"{}" ng-problem-valid? not`, `"{}" 0 0 ng-domain-has? not`, `0 "bit" ng-refine-mask nil =`, `"kind" 3 ng-artifact-name nil =`} {
		value, err := vm.Execute(program)
		if err != nil || !value.AsBool() {
			t.Fatalf("%s = %v, %v", program, value, err)
		}
	}
}
