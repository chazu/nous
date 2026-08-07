package dsl

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func configRepairTestStore() *unit.Store {
	store := protocolTestStore("")
	marker := unit.New("ConfigurationRepairVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "configrepair")
	store.Put(marker)
	return store
}

func configDSLList(values []string) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Quote(value)
	}
	return strings.Join(parts, " ") + fmt.Sprintf(" %d list-of", len(values))
}

func addConfigRepair(store *unit.Store, name, key, value string) {
	repair := unit.New(name)
	repair.Set("isA", []string{"PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"})
	repair.Set("repairKey", key)
	repair.Set("repairValue", value)
	repair.Set("defn", strconv.Quote(key)+" "+strconv.Quote(value)+" config-set")
	store.Put(repair)
}

func TestConfigRepairWordsAreScopedAndExecuteInChildVM(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(`0 list-of config-valid?`); err == nil || !strings.Contains(err.Error(), "unknown word") {
		t.Fatalf("base VM exposed config word: %v", err)
	}

	store := configRepairTestStore()
	addConfigRepair(store, "SetPort", "service_port", "443")
	op := unit.New("SetPortComposite")
	op.Set("isA", []string{"UnaryOp", "Op", "Anything"})
	op.Set("defn", `"service_port" "443" config-set`)
	store.Put(op)
	input := []string{"tls=true", "service_port=80"}
	value, err := NewVM(store, agenda.New(), nil).Execute(configDSLList(input) + ` "SetPortComposite" apply-op`)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := strictStringList(value)
	if !ok || strings.Join(got, ",") != "service_port=443,tls=true" {
		t.Fatalf("composite output = %v", value)
	}
}

func TestConfigRepairSemanticWords(t *testing.T) {
	vm := NewVM(configRepairTestStore(), agenda.New(), nil)
	schema := []string{
		"field:environment:string", "field:tls:bool", "field:service_port:int:1:65535",
		"required:environment", "required:tls", "required:service_port",
		"protected:environment", "protected:tls", "eq-if:tls=true,service_port=443",
	}
	input := []string{"environment=production", "tls=true", "service_port=80"}
	repaired := []string{"environment=production", "tls=true", "service_port=443"}
	value, err := vm.Execute(configDSLList(input) + " " + configDSLList(schema) + ` config-satisfies?`)
	if err != nil || value.Kind() != VBool || value.AsBool() {
		t.Fatalf("nonconforming config = (%v,%v)", value, err)
	}
	value, err = vm.Execute(configDSLList(repaired) + " " + configDSLList(schema) + ` config-satisfies?`)
	if err != nil || !value.AsBool() {
		t.Fatalf("repaired config = (%v,%v)", value, err)
	}
	value, err = vm.Execute(configDSLList(input) + " " + configDSLList(repaired) + " " + configDSLList(schema) + ` config-preserves-protected?`)
	if err != nil || !value.AsBool() {
		t.Fatalf("preservation = (%v,%v)", value, err)
	}
	value, err = vm.Execute(configDSLList(input) + " " + configDSLList(repaired) + ` config-changed-count`)
	if err != nil || value.AsInt() != 1 {
		t.Fatalf("changed count = (%v,%v)", value, err)
	}
	invalidTyped := []string{"environment=production", "tls=yes", "service_port=443"}
	value, err = vm.Execute(configDSLList(invalidTyped) + " " + configDSLList(invalidTyped) + " " + configDSLList(schema) + ` config-preserves-protected?`)
	if err != nil || !value.IsNil() {
		t.Fatalf("type-invalid protected comparison = (%v,%v), want nil", value, err)
	}
}

func TestConfigPlanIdentityIsAliasInvariantAndCollisionSafe(t *testing.T) {
	store := configRepairTestStore()
	addConfigRepair(store, "A", "service_port", "443")
	addConfigRepair(store, "B", "replicas", "2")
	addConfigRepair(store, "C", "admin_public", "false")
	addConfigRepair(store, "Opaque-X", "admin_public", "false")
	addConfigRepair(store, "Opaque-Y", "service_port", "443")
	addConfigRepair(store, "Opaque-Z", "replicas", "2")
	vm := NewVM(store, agenda.New(), nil)
	first, err := vm.Execute(`"A" "B" "C" 3 list-of config-decision-key`)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := vm.Execute(`"Opaque-X" "Opaque-Y" "Opaque-Z" 3 list-of config-decision-key`)
	if err != nil || first.AsString() != alias.AsString() {
		t.Fatalf("alias decisions = %q %q err=%v", first.AsString(), alias.AsString(), err)
	}
	definition, err := vm.Execute(`"C" "A" "B" 3 list-of config-plan-defn`)
	if err != nil || definition.AsString() != `"service_port" "443" config-set "replicas" "2" config-set "admin_public" "false" config-set` {
		t.Fatalf("structured definition = (%q,%v)", definition.AsString(), err)
	}
	name, err := vm.Execute(`"A" "B" "C" 3 list-of config-plan-name`)
	if err != nil || name.Kind() != VString {
		t.Fatalf("plan name = (%v,%v)", name, err)
	}
	occupied := unit.New(name.AsString())
	occupied.Set("sentinel", "keep")
	store.Put(occupied)
	fresh, err := vm.Execute(`"A" "B" "C" 3 list-of config-plan-name`)
	if err != nil || fresh.AsString() == name.AsString() || !strings.HasSuffix(fresh.AsString(), "collision-1") {
		t.Fatalf("fresh plan name = (%v,%v)", fresh, err)
	}
}

func TestConfigWordsRejectCoerciveAndConflictingInputs(t *testing.T) {
	store := configRepairTestStore()
	addConfigRepair(store, "A", "same", "x")
	addConfigRepair(store, "B", "same", "y")
	impostor := unit.New("Impostor")
	impostor.Set("isA", []string{"Anything"})
	impostor.Set("repairKey", "other")
	impostor.Set("repairValue", "z")
	store.Put(impostor)
	vm := NewVM(store, agenda.New(), nil)
	for _, program := range []string{
		`1 "x" config-repair-valid?`,
		`"x" 1 config-repair-valid?`,
	} {
		value, err := vm.Execute(program)
		if err != nil || value.Kind() != VBool || value.AsBool() {
			t.Fatalf("classifier %q = (%v,%v)", program, value, err)
		}
	}
	for _, program := range []string{
		`1 config-plan-valid?`,
		`"A" 1 2 list-of config-plan-valid?`,
		`"missing" 1 list-of config-plan-valid?`,
		`"Impostor" 1 list-of config-plan-valid?`,
	} {
		value, err := vm.Execute(program)
		if err != nil || !value.IsNil() {
			t.Fatalf("semantic rejection %q = (%v,%v)", program, value, err)
		}
	}
	value, err := vm.Execute(`"A" "B" 2 list-of config-plan-valid?`)
	if err != nil || value.Kind() != VBool || value.AsBool() {
		t.Fatalf("same-key plan = (%v,%v)", value, err)
	}
}
