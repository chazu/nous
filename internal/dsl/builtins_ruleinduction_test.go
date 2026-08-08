package dsl

import (
	"strconv"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func ruleInductionVM(t *testing.T) *VM {
	t.Helper()
	store := unit.NewStore()
	category := unit.New("Vocabulary")
	category.Set("isA", []string{"Anything"})
	store.Put(category)
	vocabulary := unit.New("RuleInductionVocabulary")
	vocabulary.Set("isA", []string{"Vocabulary", "Anything"})
	vocabulary.Set("dslExtension", "ruleinduction")
	store.Put(vocabulary)
	return NewVM(store, agenda.New(), nil)
}

func TestRuleInductionRefinesExactlyOneField(t *testing.T) {
	vm := ruleInductionVM(t)
	value, err := vm.Execute(`"----" ri-refine-one`)
	if err != nil {
		t.Fatal(err)
	}
	items := value.AsList()
	if len(items) != 2 || items[0].AsString() != "0---" || items[1].AsString() != "1---" {
		t.Fatalf("children = %v", value)
	}
	value, err = vm.Execute(`"0010" ri-complete-code`)
	if err != nil || value.AsString() != "03" {
		t.Fatalf("complete = (%v,%v)", value, err)
	}
}

func TestRuleInductionRefinementLedgerCoversEveryBindValidationAndOrder(t *testing.T) {
	vm := ruleInductionVM(t)
	frontier, work := []string{"----"}, 1
	for len(frontier) > 0 {
		var next []string
		for _, partial := range frontier {
			value, err := vm.Execute(strconv.Quote(partial) + " ri-refine-one")
			if err != nil { t.Fatal(err) }
			for _, child := range value.AsList() {
				text := child.AsString()
				charged, err := vm.Execute(strconv.Quote(text) + " ri-refinement-work")
				if err != nil { t.Fatal(err) }
				work += charged.AsInt()
				if strings.Contains(text, "-") { next = append(next, text) }
			}
		}
		frontier = next
	}
	if work != 149 { t.Fatalf("refinement work = %d, want 149", work) }
}

func TestRuleInductionExecutesOneExplicitDefinition(t *testing.T) {
	vm := ruleInductionVM(t)
	program := `"03" "0:0:1" "0:1:2" "0:1:3" "0:3:4" 4 list-of ri-signature`
	value, err := vm.Execute(program)
	if err != nil || len(value.AsString()) != 64 {
		t.Fatalf("signature = (%v,%v)", value, err)
	}
	member, err := vm.Execute(`"03" "0:0:1" "0:1:2" "0:1:3" "0:3:4" 4 list-of ri-signature 0 4 ri-signature-has?`)
	if err != nil || !member.AsBool() {
		t.Fatalf("membership = (%v,%v)", member, err)
	}
}

func TestRuleInductionWordsRemainScoped(t *testing.T) {
	vm := NewVM(unit.NewStore(), agenda.New(), nil)
	if _, err := vm.Execute(`"----" ri-refine-one`); err == nil {
		t.Fatal("rule-induction word available without vocabulary")
	}
}
