package dsl

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func TestTransformSchemaWordsAreScoped(t *testing.T) {
	empty := protocolTestStore("")
	if err := NewVM(empty, agenda.New(), nil).InitError(); err != nil {
		t.Fatal(err)
	}
	if _, ok := NewVM(empty, agenda.New(), nil).words["ts-schema-apply"]; ok {
		t.Fatal("transformation word leaked into base VM")
	}
	store := protocolTestStore("")
	marker := unit.New("TransformSchemaVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "transformschema")
	store.Put(marker)
	vm := NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
	if _, ok := vm.words["ts-schema-apply"]; !ok {
		t.Fatal("scoped word absent")
	}
}

func TestTransformSchemaApplyWord(t *testing.T) {
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "service", Value: "old", Target: -1},
		{ID: 2, Kind: "request", Parent: 0, Key: "change", From: "old", To: "new", Target: 1},
	}}
	fb, _ := f.CanonicalJSON()
	sb, _ := (transformschema.Schema{"request-target", "definition", "local", "any", "required"}).CanonicalJSON()
	vm := &VM{stack: []Value{StringVal(string(fb)), StringVal(string(sb))}}
	if err := bTSSchemaApply(vm); err != nil {
		t.Fatal(err)
	}
	result := vm.pop()
	if result.Kind() != VList || result.AsList()[0].AsString() != "applied" {
		t.Fatalf("result=%v", result)
	}
}

func TestTransformLocalFactsAndMeter(t *testing.T) {
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "service", Value: "old", Target: -1},
	}}
	b, _ := f.CanonicalJSON()
	vm := &VM{stack: []Value{StringVal(string(b)), IntVal(1)}}
	if err := bTSNodeFacts(vm); err != nil {
		t.Fatal(err)
	}
	facts := vm.pop()
	if facts.Kind() != VList || facts.AsList()[0].AsString() != "definition" || facts.AsList()[1].AsString() != "old" {
		t.Fatalf("facts=%v", facts)
	}
	if err := RegisterTransformMeter("test-meter"); err != nil {
		t.Fatal(err)
	}
	defer UnregisterTransformMeter("test-meter")
	if err := ChargeTransformMeter("test-meter", "node", "subject", "object", "ok", 0); err != nil {
		t.Fatal(err)
	}
	records, err := TransformMeterSnapshot("test-meter")
	if err != nil || len(records) != 1 || records[0].Category != 0 {
		t.Fatalf("records=%v err=%v", records, err)
	}
}
