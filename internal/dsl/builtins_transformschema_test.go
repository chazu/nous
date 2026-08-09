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

func TestTransformRefineReturnsCanonicalChildBytes(t *testing.T) {
	root, _ := (transformschema.Partial{}).CanonicalJSON()
	vm := &VM{stack: []Value{StringVal(string(root)), StringVal("definition")}}
	if err := bTSRefine(vm); err != nil {
		t.Fatal(err)
	}
	child := vm.pop()
	if child.Kind() != VString {
		t.Fatalf("child kind=%v", child.Kind())
	}
	partial, err := transformschema.ParsePartial([]byte(child.AsString()))
	if err != nil || partial.Stage != 1 || partial.Targets != "definition" {
		t.Fatalf("partial=%+v err=%v", partial, err)
	}
}

func TestTransformSchemaApplicationReservesCreditsBeforeExecution(t *testing.T) {
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "service", Value: "old", Target: -1},
		{ID: 2, Kind: "request", Parent: 0, Key: "change", From: "old", To: "new", Target: 1},
	}}
	forestBytes, _ := f.CanonicalJSON()
	schemaBytes, _ := (transformschema.Schema{"request-target", "definition", "local", "any", "required"}).CanonicalJSON()
	preloaded := make([]TransformMeterRecord, 39)
	for i := range preloaded {
		preloaded[i] = TransformMeterRecord{Category: 11, Operation: "schema-application", Phase: "training-validate", Outcome: "applied"}
	}
	if err := RegisterTransformMeterWithRecords("live-credit-reservation", preloaded); err != nil {
		t.Fatal(err)
	}
	defer UnregisterTransformMeter("live-credit-reservation")
	store := unit.NewStore()
	experiment := unit.New("TransformExperiment")
	experiment.Set("meterToken", "live-credit-reservation")
	store.Put(experiment)
	vm := NewVM(store, agenda.New(), nil)
	vm.CurrentTask = &agenda.Task{UnitName: experiment.Name, SlotName: "tsClose"}
	terminal, _, err := ExecuteTransformSchemaApplication(vm, forestBytes, schemaBytes)
	if err != nil || terminal != "applied" {
		t.Fatalf("first terminal=%q err=%v", terminal, err)
	}
	afterFirst, _ := TransformMeterSnapshot("live-credit-reservation")
	terminal, _, err = ExecuteTransformSchemaApplication(vm, forestBytes, schemaBytes)
	if err != nil || terminal != "budget-exhausted" {
		t.Fatalf("second terminal=%q err=%v", terminal, err)
	}
	afterSecond, _ := TransformMeterSnapshot("live-credit-reservation")
	if len(afterFirst) <= len(preloaded) || len(afterSecond) != len(afterFirst) {
		t.Fatalf("partial attempt escaped reservation: before=%d first=%d second=%d", len(preloaded), len(afterFirst), len(afterSecond))
	}
}
