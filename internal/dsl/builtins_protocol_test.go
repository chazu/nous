package dsl

import (
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func protocolTestStore(extension string) *unit.Store {
	store := unit.NewStore()
	vocabulary := unit.New("Vocabulary")
	vocabulary.Set("isA", []string{"Anything"})
	store.Put(vocabulary)
	if extension != "" {
		marker := unit.New("TestVocabulary")
		marker.Set("isA", []string{"Vocabulary", "Anything"})
		marker.Set("dslExtension", extension)
		store.Put(marker)
	}
	return store
}

func protocolLiteral() string {
	return `"state:idle" "state:done" "event:go" "start:idle" "accept:done" "trans:idle,go>done" 6 list-of`
}

func TestProtocolWordsAreVocabularyScopedAndInherited(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(protocolLiteral() + ` protocol-valid?`); err == nil || !strings.Contains(err.Error(), "unknown word") {
		t.Fatalf("base VM unexpectedly exposed protocol word: %v", err)
	}

	store := protocolTestStore("protocol")
	op := unit.New("Canonicalize")
	op.Set("isA", []string{"UnaryOp", "Op", "Anything"})
	op.Set("defn", "protocol-canonicalize")
	store.Put(op)
	vm := NewVM(store, agenda.New(), nil)
	result, err := vm.Execute(protocolLiteral() + ` "Canonicalize" apply-op`)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsNil() || len(result.AsList()) != 6 {
		t.Fatalf("sub-VM did not inherit protocol words: %v", result)
	}
}

func TestVocabularyMarkerFailures(t *testing.T) {
	emptyStore := protocolTestStore("")
	empty := unit.New("EmptyVocabulary")
	empty.Set("isA", []string{"Vocabulary", "Anything"})
	emptyStore.Put(empty)
	if err := NewVM(emptyStore, agenda.New(), nil).InitError(); err == nil || !strings.Contains(err.Error(), "has no dslExtension") {
		t.Fatalf("empty extension error = %v", err)
	}

	unknown := NewVM(protocolTestStore("missing-extension"), agenda.New(), nil)
	if err := unknown.InitError(); err == nil || !strings.Contains(err.Error(), "unknown DSL extension") {
		t.Fatalf("unknown extension error = %v", err)
	}

	store := protocolTestStore("protocol")
	duplicate := unit.New("DuplicateVocabulary")
	duplicate.Set("isA", []string{"Vocabulary", "Anything"})
	duplicate.Set("dslExtension", "protocol")
	store.Put(duplicate)
	vm := NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err == nil || !strings.Contains(err.Error(), "duplicate DSL extension") {
		t.Fatalf("duplicate extension error = %v", err)
	}
}

func TestMultipleVocabularyExtensionsResolveDeterministically(t *testing.T) {
	registerVocabularyWords("protocol-test-z", map[string]builtinFn{
		"protocol-test-z-word": func(vm *VM) error { vm.push(StringVal("z")); return nil },
	})
	registerVocabularyWords("protocol-test-a", map[string]builtinFn{
		"protocol-test-a-word": func(vm *VM) error { vm.push(StringVal("a")); return nil },
	})

	store := protocolTestStore("protocol-test-z")
	marker := unit.New("EarlierVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "protocol-test-a")
	store.Put(marker)
	vm := NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
	value, err := vm.Execute(`protocol-test-z-word protocol-test-a-word`)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.AsString(); got != "a" {
		t.Fatalf("last selected word = %q, want a", got)
	}
}

func TestInvalidSemanticPredicateDoesNotUpdateRarity(t *testing.T) {
	store := protocolTestStore("protocol")
	pred := unit.New("EquivalentProtocols")
	pred.Set("isA", []string{"BinaryPred", "Pred", "Op", "Anything"})
	pred.Set("defn", "protocol-equivalent?")
	store.Put(pred)
	vm := NewVM(store, agenda.New(), nil)
	value, err := vm.Execute(`"not-a-protocol" 1 list-of "also-bad" 1 list-of "EquivalentProtocols" apply-op`)
	if err != nil {
		t.Fatal(err)
	}
	if !value.IsNil() {
		t.Fatalf("invalid equivalence = %v, want nil", value)
	}
	if store.Get("EquivalentProtocols").Has("rarity") {
		t.Fatal("invalid predicate application changed rarity")
	}
}

func TestValidProtocolClassifiesMalformedInputAndUpdatesRarity(t *testing.T) {
	store := protocolTestStore("protocol")
	pred := unit.New("ValidProtocol")
	pred.Set("isA", []string{"UnaryPred", "Pred", "Op", "Anything"})
	pred.Set("defn", "protocol-valid?")
	store.Put(pred)
	vm := NewVM(store, agenda.New(), nil)
	value, err := vm.Execute(`"not-a-protocol" 1 list-of "ValidProtocol" apply-op`)
	if err != nil {
		t.Fatal(err)
	}
	if value.AsBool() {
		t.Fatalf("malformed protocol classified as valid: %v", value)
	}
	rarity, ok := store.Get("ValidProtocol").Get("rarity").([]any)
	if !ok || len(rarity) != 3 {
		t.Fatalf("rarity = %#v, want one classification", store.Get("ValidProtocol").Get("rarity"))
	}
	if truths, falses := rarity[1], rarity[2]; truths != 0 || falses != 1 {
		t.Fatalf("rarity counts = (%v,%v), want (0,1)", truths, falses)
	}
}
