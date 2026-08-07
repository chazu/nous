package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	protocolvocab "github.com/chazu/nous/internal/vocab/protocol"
)

const protocolCycles = 120

type protocolTrial struct {
	store *unit.Store
	eng   *engine.Engine
}

func loadProtocols(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "protocols"); err != nil {
		t.Fatal(err)
	}
	return store
}

func runProtocols(t *testing.T, aliases map[string]string, mutate bool) protocolTrial {
	t.Helper()
	store := loadProtocols(t)
	for oldName, newName := range aliases {
		original := store.Delete(oldName)
		if original == nil {
			t.Fatalf("cannot alias missing unit %s", oldName)
		}
		alias := unit.New(newName)
		for slot, value := range original.Slots {
			alias.Set(slot, value)
		}
		store.Put(alias)
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = protocolCycles
	eng.MutConfig.Enabled = mutate
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	return protocolTrial{store: store, eng: eng}
}

func TestProtocolsVocabularyIsIndependentAndDiscoversRelations(t *testing.T) {
	trial := runProtocols(t, nil, false)
	store := trial.store
	for _, name := range []string{"ProtocolVocabulary", "Protocol", "H-DiscoverProtocolRelations"} {
		if !store.Has(name) {
			t.Fatalf("protocols vocabulary missing %s", name)
		}
	}
	for _, mathOnly := range []string{"MathConcept", "Set", "H1"} {
		if store.Has(mathOnly) {
			t.Fatalf("protocols unexpectedly loaded math-only unit %s", mathOnly)
		}
	}
	if err := trial.eng.VM.InitError(); err != nil {
		t.Fatalf("protocol DSL selection failed: %v", err)
	}

	assertProtocolMatrix(t, store)

	if !store.Has("Conjec-ProtocolRejectingTrap-TrainingMachineGamma") {
		t.Fatal("control lane did not report Gamma's rejecting trap")
	}
	for _, name := range []string{
		"Conjec-ProtocolRejectingTrap-TrainingMachineAlpha",
		"Conjec-ProtocolRejectingTrap-TrainingMachineBeta",
	} {
		if store.Has(name) {
			t.Fatalf("control lane reported a nonexistent rejecting trap: %s", name)
		}
	}
	control := store.Get("Conjec-ProtocolRejectingTrap-TrainingMachineGamma")
	if got := control.GetStrings("evidence"); !reflect.DeepEqual(got, []string{"Evidence-RejectingTraps-TrainingMachineGamma"}) {
		t.Fatalf("control evidence = %v", got)
	}

	beforeCount := store.Count()
	before := protocolExperimentSnapshot(t, store)
	for _, transform := range protocolOperations(store, "ProtocolTransform") {
		trial.eng.WorkOnUnit(transform)
	}
	trial.eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: "TrainingMachineGamma", SlotName: "protocolAnalysis"})
	after := protocolExperimentSnapshot(t, store)
	if string(before) != string(after) {
		t.Fatal("repeated focus changed guarded protocol evidence or support")
	}
	if store.Count() != beforeCount {
		t.Fatalf("repeated focus changed unit count from %d to %d", beforeCount, store.Count())
	}

	assertHeldOutSchema(t, store)
}

func TestProtocolDiscoverySurvivesOpaqueOperationAliases(t *testing.T) {
	aliases := map[string]string{
		"CanonicalizeProtocol":            "T0",
		"RemoveUnreachableProtocolStates": "T1",
		"DropFirstProtocolTransition":     "T2",
		"EquivalentProtocols":             "R0",
		"SameProtocolEncoding":            "R1",
	}
	trial := runProtocols(t, aliases, false)
	for canonical := range aliases {
		if trial.store.Has(canonical) {
			t.Fatalf("canonical operation identity survived aliasing: %s", canonical)
		}
	}
	assertProtocolMatrix(t, trial.store)
	assertHeldOutSchema(t, trial.store)
	if !trial.store.Has("Schema-T1-R0") {
		t.Fatal("opaque aliases did not produce the trim/equivalence schema")
	}
}

func TestProtocolRunsHaveDeterministicStoreSnapshots(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprintf("mutation=%v", mutate), func(t *testing.T) {
			first := canonicalStoreSnapshot(t, runProtocols(t, nil, mutate).store)
			second := canonicalStoreSnapshot(t, runProtocols(t, nil, mutate).store)
			if string(first) != string(second) {
				t.Fatal("fixed-seed protocol runs produced different stores")
			}
		})
	}
}

func TestMalformedProtocolDoesNotBecomeDiscoveryEvidence(t *testing.T) {
	store := loadProtocols(t)
	bad := unit.New("MalformedTrainingMachine")
	bad.SetWorth(700)
	bad.Set("isA", []string{"ProtocolTrainingExample", "Protocol", "Anything"})
	bad.Set("data", []string{"state:only", "start:missing"})
	store.Put(bad)

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.MaxCycles = protocolCycles
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: bad.Name, SlotName: "protocolAnalysis"})

	for _, name := range store.All() {
		if (strings.HasPrefix(name, "Result-") || strings.HasPrefix(name, "Observation-")) && strings.HasSuffix(name, "-"+bad.Name) {
			t.Fatalf("malformed semantic input created %s", name)
		}
		if strings.HasPrefix(name, "Evidence-") {
			for _, subject := range store.Get(name).GetStrings("trainingSubjects") {
				if subject == bad.Name {
					t.Fatalf("malformed semantic input entered evidence %s", name)
				}
			}
		}
	}
	if store.Has("Evidence-RejectingTraps-"+bad.Name) || store.Has("Conjec-ProtocolRejectingTrap-"+bad.Name) {
		t.Fatal("malformed semantic input created control evidence or conjecture")
	}
}

func assertProtocolMatrix(t *testing.T, store *unit.Store) {
	t.Helper()
	transforms := protocolOperations(store, "ProtocolTransform")
	relations := protocolOperations(store, "ProtocolRelation")
	training := protocolTraining(store)
	if len(transforms) != 3 || len(relations) != 2 || len(training) != 3 {
		t.Fatalf("experiment dimensions = %d transforms x %d relations x %d training", len(transforms), len(relations), len(training))
	}

	candidateCount, evidenceCount, resultCount, observationCount := 0, 0, 0, 0
	for _, name := range store.All() {
		switch {
		case strings.HasPrefix(name, "Candidate-"):
			candidateCount++
		case strings.HasPrefix(name, "Evidence-") && !strings.HasPrefix(name, "Evidence-RejectingTraps-"):
			evidenceCount++
		case strings.HasPrefix(name, "Result-"):
			resultCount++
		case strings.HasPrefix(name, "Observation-"):
			observationCount++
		}
	}
	if candidateCount != len(transforms)*len(relations) || evidenceCount != candidateCount {
		t.Fatalf("matrix materialization = %d candidates, %d evidence; want %d each", candidateCount, evidenceCount, len(transforms)*len(relations))
	}
	if resultCount != candidateCount*len(training) || observationCount != resultCount {
		t.Fatalf("matrix observations = %d results, %d observations; want %d each", resultCount, observationCount, candidateCount*len(training))
	}

	for _, transform := range transforms {
		applications, ok := store.Get(transform).Get("applics").([]map[string]any)
		if !ok || len(applications) != len(relations)*len(training) {
			t.Fatalf("%s applications = %d, want %d", transform, len(applications), len(relations)*len(training))
		}
		for _, relation := range relations {
			support, failures := independentlyEvaluatePair(t, store, transform, relation, training)
			candidateName := "Candidate-" + transform + "-" + relation
			evidenceName := "Evidence-" + transform + "-" + relation
			candidate := store.Get(candidateName)
			evidence := store.Get(evidenceName)
			if candidate == nil || evidence == nil {
				t.Fatalf("missing pair artifacts for (%s,%s)", transform, relation)
			}
			wantWorth := 500 + 50*support - 100*failures
			if got := [3]int{candidate.GetInt("supportCount"), candidate.GetInt("failureCount"), candidate.Worth()}; got != [3]int{support, failures, wantWorth} {
				t.Fatalf("%s = %v, want support/failure/worth %v", candidateName, got, [3]int{support, failures, wantWorth})
			}
			if candidate.GetString("evidenceUnit") != evidenceName {
				t.Fatalf("%s evidenceUnit = %q", candidateName, candidate.GetString("evidenceUnit"))
			}
			if evidence.GetInt("supportCount") != support || evidence.GetInt("failureCount") != failures {
				t.Fatalf("%s counts disagree with independent control", evidenceName)
			}
			for _, slot := range []string{"trainingSubjects", "resultUnits", "relationObservations"} {
				if got := len(evidence.GetStrings(slot)); got != len(training) {
					t.Fatalf("%s.%s length = %d, want %d", evidenceName, slot, got, len(training))
				}
			}
			for _, artifact := range []*unit.Unit{candidate, evidence} {
				if !reflect.DeepEqual(artifact.GetStrings("creditors"), []string{"H-DiscoverProtocolRelations"}) {
					t.Fatalf("%s creditors = %v", artifact.Name, artifact.GetStrings("creditors"))
				}
			}
			promoted := store.Has("Schema-" + transform + "-" + relation)
			wantPromoted := support == len(training) && failures == 0
			if promoted != wantPromoted {
				t.Fatalf("pair (%s,%s) promotion = %v, want %v", transform, relation, promoted, wantPromoted)
			}
		}
	}
}

func protocolOperations(store *unit.Store, category string) []string {
	var names []string
	for _, name := range store.Examples(category) {
		if name != category && strings.TrimSpace(store.Get(name).GetString("defn")) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func protocolTraining(store *unit.Store) []string {
	var names []string
	for _, name := range store.Examples("ProtocolTrainingExample") {
		if name != "ProtocolTrainingExample" && len(store.Get(name).GetStrings("data")) != 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func independentlyEvaluatePair(t *testing.T, store *unit.Store, transform, relation string, training []string) (int, int) {
	t.Helper()
	var support, failures int
	for _, subject := range training {
		machine := mustProtocol(t, store.Get(subject).GetStrings("data"))
		transformed := applyProtocolTransform(t, strings.TrimSpace(store.Get(transform).GetString("defn")), machine)
		if applyProtocolRelation(t, strings.TrimSpace(store.Get(relation).GetString("defn")), machine, transformed) {
			support++
		} else {
			failures++
		}
	}
	return support, failures
}

func mustProtocol(t *testing.T, records []string) protocolvocab.Machine {
	t.Helper()
	machine, err := protocolvocab.Parse(records)
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func applyProtocolTransform(t *testing.T, defn string, machine protocolvocab.Machine) protocolvocab.Machine {
	t.Helper()
	switch defn {
	case "protocol-canonicalize":
		return mustProtocol(t, machine.Records())
	case "protocol-remove-unreachable":
		return machine.TrimUnreachable()
	case "protocol-drop-first-transition":
		return machine.DropFirstTransition()
	default:
		t.Fatalf("unknown transform definition %q", defn)
		return protocolvocab.Machine{}
	}
}

func applyProtocolRelation(t *testing.T, defn string, a, b protocolvocab.Machine) bool {
	t.Helper()
	switch defn {
	case "protocol-equivalent?":
		equivalent, _ := protocolvocab.Compare(a, b)
		return equivalent
	case "protocol-same-encoding?":
		return protocolvocab.SameEncoding(a, b)
	default:
		t.Fatalf("unknown relation definition %q", defn)
		return false
	}
}

func assertHeldOutSchema(t *testing.T, store *unit.Store) {
	t.Helper()
	var target *unit.Unit
	for _, name := range store.Examples("ProtocolRelationSchema") {
		schema := store.Get(name)
		if schema == nil {
			continue
		}
		transform := store.Get(schema.GetString("transform"))
		relation := store.Get(schema.GetString("relation"))
		if transform != nil && relation != nil &&
			strings.TrimSpace(transform.GetString("defn")) == "protocol-remove-unreachable" &&
			strings.TrimSpace(relation.GetString("defn")) == "protocol-equivalent?" {
			target = schema
			break
		}
	}
	if target == nil {
		t.Fatal("no promoted trim/equivalence schema")
	}

	hidden := [][]string{
		{"state:x", "state:y", "event:go", "start:x", "accept:y", "trans:x,go>y"},
		{"state:p", "state:q", "state:u", "state:v", "event:a", "event:z", "start:p", "accept:q", "accept:v", "trans:p,a>q", "trans:u,z>v", "trans:v,z>u"},
		{"state:s", "state:lost", "event:retry", "start:s", "trans:lost,retry>lost"},
		{"state:r0", "state:r1", "state:r2", "event:close", "event:open", "start:r0", "accept:r2", "trans:r0,open>r1", "trans:r1,close>r2"},
	}
	before := store.Count()
	transformDefn := strings.TrimSpace(store.Get(target.GetString("transform")).GetString("defn"))
	relationDefn := strings.TrimSpace(store.Get(target.GetString("relation")).GetString("defn"))
	for i, records := range hidden {
		machine := mustProtocol(t, records)
		transformed := applyProtocolTransform(t, transformDefn, machine)
		if !applyProtocolRelation(t, relationDefn, machine, transformed) {
			t.Fatalf("promoted schema failed held-out machine %d", i)
		}
	}
	if store.Count() != before {
		t.Fatal("held-out evaluation leaked examples into the discovery store")
	}
}

func canonicalStoreSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	snapshot := make(map[string]map[string]any, store.Count())
	for _, name := range store.All() {
		slots := make(map[string]any, len(store.Get(name).Slots))
		for slot, value := range store.Get(name).Slots {
			slots[slot] = value
		}
		snapshot[name] = slots
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func protocolExperimentSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	snapshot := make(map[string]map[string]any)
	for _, name := range store.All() {
		if !(strings.HasPrefix(name, "Candidate-") ||
			strings.HasPrefix(name, "Evidence-") ||
			strings.HasPrefix(name, "Result-") ||
			strings.HasPrefix(name, "Observation-") ||
			strings.HasPrefix(name, "Schema-") ||
			store.IsA(name, "ProtocolTransform")) {
			continue
		}
		snapshot[name] = store.Get(name).Slots
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
