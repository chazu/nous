package dsl

import (
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func rewriteTestStore() *unit.Store {
	store := protocolTestStore("")
	marker := unit.New("RewriteVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "rewrite")
	store.Put(marker)
	return store
}

func TestRewriteWordsAreScopedAndComposeInChildVM(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(`"abc" "ab" "x" rewrite-replace-all`); err == nil || !strings.Contains(err.Error(), "unknown word") {
		t.Fatalf("base VM unexpectedly exposed rewrite word: %v", err)
	}

	store := rewriteTestStore()
	op := unit.New("TwoPass")
	op.Set("isA", []string{"UnaryOp", "Op", "Anything"})
	op.Set("defn", `"ab" "x" rewrite-replace-all "xc" "y" rewrite-replace-all`)
	store.Put(op)
	value, err := NewVM(store, agenda.New(), nil).Execute(`"abcabc" "TwoPass" apply-op`)
	if err != nil || value.AsString() != "yy" {
		t.Fatalf("composed child execution = (%v,%v), want yy", value, err)
	}
}

func TestRewriteValidationAndSemanticNilRarity(t *testing.T) {
	store := rewriteTestStore()
	valid := unit.New("ValidRewriteString")
	valid.Set("isA", []string{"UnaryPred", "Pred", "Op", "Anything"})
	valid.Set("defn", "rewrite-valid?")
	store.Put(valid)
	applies := unit.New("RuleApplies")
	applies.Set("isA", []string{"BinaryPred", "Pred", "Op", "Anything"})
	applies.Set("defn", "rewrite-rule-applies?")
	store.Put(applies)
	vm := NewVM(store, agenda.New(), nil)

	classified, err := vm.Execute(`"INVALID" "ValidRewriteString" apply-op`)
	if err != nil || classified.AsBool() {
		t.Fatalf("invalid classifier = (%v,%v)", classified, err)
	}
	if rarity, ok := valid.Get("rarity").([]any); !ok || rarity[2] != 1 {
		t.Fatalf("validity rarity = %#v", valid.Get("rarity"))
	}

	semantic, err := vm.Execute(`"INVALID" "a" "RuleApplies" apply-op`)
	if err != nil || !semantic.IsNil() {
		t.Fatalf("invalid semantic predicate = (%v,%v)", semantic, err)
	}
	if applies.Has("rarity") {
		t.Fatal("semantic nil changed predicate rarity")
	}
}

func TestRewriteNamesPreserveOrderAndAvoidOccupiedIdentities(t *testing.T) {
	store := rewriteTestStore()
	vm := NewVM(store, agenda.New(), nil)
	forward, err := vm.Execute(`"a-then-b" "c" rewrite-compose-name`)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := vm.Execute(`"c" "a-then-b" rewrite-compose-name`)
	if err != nil {
		t.Fatal(err)
	}
	if forward.AsString() == reverse.AsString() {
		t.Fatal("ordered component identities collided")
	}
	occupied := unit.New(forward.AsString())
	store.Put(occupied)
	fresh, err := vm.Execute(`"a-then-b" "c" rewrite-compose-name`)
	if err != nil || fresh.AsString() == forward.AsString() || !strings.Contains(fresh.AsString(), "collision-1") {
		t.Fatalf("fresh identity = (%q,%v)", fresh.AsString(), err)
	}
	firstKey, err := vm.Execute(`"a-then-b" "c" rewrite-decision-key`)
	if err != nil {
		t.Fatal(err)
	}
	secondKey, err := vm.Execute(`"a-then-b" "c" rewrite-decision-key`)
	if err != nil || firstKey.AsString() != secondKey.AsString() {
		t.Fatalf("semantic decision key changed across occupied names: %q %q", firstKey.AsString(), secondKey.AsString())
	}
}
