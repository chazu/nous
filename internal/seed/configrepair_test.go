package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	configvocab "github.com/chazu/nous/internal/vocab/configrepair"
)

const configurationRepairCycles = 700

func loadConfigurationRepair(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "configrepair"); err != nil {
		t.Fatal(err)
	}
	return store
}

func runConfigurationRepair(t *testing.T, store *unit.Store, mutate bool) (*unit.Store, *engine.Engine) {
	t.Helper()
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = configurationRepairCycles
	eng.MutConfig.Enabled = mutate
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ag.Len() != 0 {
		t.Fatalf("configuration repair agenda did not drain after %d cycles: %d tasks remain", eng.MaxCycles, ag.Len())
	}
	return store, eng
}

func configurationRepairUnits(store *unit.Store, category, requiredSlot string) []string {
	var names []string
	for _, name := range store.Examples(category) {
		if name != category && store.Get(name).Has(requiredSlot) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func TestConfigurationRepairVocabularySynthesizesUniqueIntentPreservingPlan(t *testing.T) {
	store, eng := runConfigurationRepair(t, loadConfigurationRepair(t), false)
	if err := eng.VM.InitError(); err != nil {
		t.Fatal(err)
	}
	for _, foreign := range []string{"MathConcept", "Protocol", "RewriteString", "BuildGraph"} {
		if store.Has(foreign) {
			t.Fatalf("configuration repair vocabulary loaded foreign unit %s", foreign)
		}
	}
	target := assertConfigurationRepairExperiment(t, store, 4)
	if got := oracleAssignments(target.GetStrings("components"), store); !sameAssignments(got, []configAssignment{{"admin_public", "false"}, {"replicas", "2"}, {"service_port", "443"}}) {
		t.Fatalf("promoted assignments = %v", got)
	}
	if target.GetInt("creationWorth") != 500 || target.GetInt("lastRewardedWorth") != 800 || target.Worth() != 800 {
		t.Fatalf("target reward state = creation %d last %d worth %d", target.GetInt("creationWorth"), target.GetInt("lastRewardedWorth"), target.Worth())
	}
	if store.Get("H-ComposeConfigurationRepairPlans").Worth() != 900 {
		t.Fatalf("synthesis heuristic worth = %d, want 900", store.Get("H-ComposeConfigurationRepairPlans").Worth())
	}
	components := target.GetStrings("components")
	for _, primitive := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		want := 600
		if containsString(components, primitive) {
			want = 750
		}
		if got := store.Get(primitive).Worth(); got != want {
			t.Fatalf("%s worth = %d, want %d", primitive, got, want)
		}
	}
	if !credit.ValidDeclaration(target.GetString("creditContext"), target.GetString("creditDecision"), target.GetStrings("creditors"), target.GetStrings("creditRoles")) {
		t.Fatal("target contextual credit declaration is invalid")
	}
	assertContextualCredit(t, store, credit.DecisionTuple(configvocab.CreditContext, target.GetString("creditDecision")), 300)
	assertContextualCredit(t, store, credit.Tuple{Context: configvocab.CreditContext, Subject: "H-ComposeConfigurationRepairPlans", Role: "synthesis"}, 150)
	for _, component := range components {
		assertContextualCredit(t, store, credit.Tuple{Context: configvocab.CreditContext, Subject: component, Role: "repair"}, 150)
	}
	assertConfigurationRepairHeldOut(t, eng.VM, target, seedHeldOutCases())
	heldOut := seedHeldOutCases()
	for _, shortcut := range []struct {
		caseIndex  int
		assignment configAssignment
	}{
		{1, configAssignment{"tls", "false"}},
		{2, configAssignment{"environment", "development"}},
	} {
		outcome := oracleOutcome(heldOut[shortcut.caseIndex].input, heldOut[shortcut.caseIndex].schema, []configAssignment{shortcut.assignment})
		if !outcome.schemaSatisfied || outcome.intentPreserved || outcome.outcome || outcome.changed != 1 {
			t.Fatalf("held-out protected shortcut %v = %#v", shortcut.assignment, outcome)
		}
	}

	protectedShortcut := findProgramByAssignments(store, []configAssignment{{"environment", "development"}, {"tls", "false"}})
	if protectedShortcut == nil || protectedShortcut.GetInt("constraintFailureCount") != 0 || protectedShortcut.GetInt("intentFailureCount") != 4 || protectedShortcut.GetInt("supportCount") != 0 {
		t.Fatalf("protected shortcut evidence = %#v", protectedShortcut)
	}

	before := configurationRepairSnapshot(t, store)
	for _, primitive := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		eng.WorkOnUnit(primitive)
	}
	eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: target.Name, SlotName: "configurationRepairEvaluation"})
	if after := configurationRepairSnapshot(t, store); string(before) != string(after) {
		t.Fatal("repeated focus/task changed guarded configuration repair evidence")
	}
	creditBefore := configurationRepairCreditSnapshot(t, store)
	eng.MaxCycles = 11
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if creditAfter := configurationRepairCreditSnapshot(t, store); string(creditBefore) != string(creditAfter) {
		t.Fatal("later periodic reward interval changed one-shot credit")
	}
}

func TestConfigurationRepairDefinitionsMatchStructuredPlansAndPermutations(t *testing.T) {
	store := loadConfigurationRepair(t)
	vm := dsl.NewVM(store, agenda.New(), nil)
	probes := make([][]string, 0)
	for _, example := range configurationRepairUnits(store, "ConfigurationRepairExample", "configuration") {
		probes = append(probes, store.Get(example).GetStrings("configuration"))
	}
	for _, primitiveName := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		primitive := store.Get(primitiveName)
		assignment := configAssignment{primitive.GetString("repairKey"), primitive.GetString("repairValue")}
		for _, probe := range probes {
			want := oracleOutcome(probe, []string{}, []configAssignment{assignment}).actual
			value, err := vm.Execute(configDSLList(probe) + " " + strconv.Quote(primitiveName) + " apply-op")
			got, ok := dslStringList(value)
			if err != nil || !ok || !equalStringSlices(got, want) {
				t.Fatalf("primitive %s parity = (%v,%v), want %v", primitiveName, got, err, want)
			}
		}
	}
	store, eng := runConfigurationRepair(t, store, false)
	for _, programName := range configurationRepairUnits(store, "CompositeConfigurationRepair", "components") {
		program := store.Get(programName)
		components := program.GetStrings("components")
		assignments := oracleAssignments(components, store)
		repairs := make([]configvocab.Repair, len(assignments))
		for index, assignment := range assignments {
			repairs[index] = configvocab.Repair{Key: assignment.key, Value: assignment.value}
		}
		for _, probe := range probes {
			want := oracleOutcome(probe, []string{}, assignments).actual
			pure, pureErr := configvocab.ApplyPlan(probe, repairs)
			if pureErr != nil || !equalStringSlices(pure, want) {
				t.Fatalf("pure plan %s parity = (%v,%v), want %v", programName, pure, pureErr, want)
			}
			value, err := eng.VM.Execute(configDSLList(probe) + " " + strconv.Quote(programName) + " apply-op")
			got, ok := dslStringList(value)
			if err != nil || !ok || !equalStringSlices(got, want) {
				t.Fatalf("composite %s parity = (%v,%v), want %v", programName, got, err, want)
			}
		}
		for _, permutation := range stringPermutations(components) {
			definition, err := eng.VM.Execute(configDSLList(permutation) + " config-plan-defn")
			if err != nil || definition.Kind() != dsl.VString || definition.AsString() != program.GetString("defn") {
				t.Fatalf("%s permutation %v definition = (%v,%v)", programName, permutation, definition, err)
			}
			decision, err := eng.VM.Execute(configDSLList(permutation) + " config-decision-key")
			if err != nil || decision.AsString() != program.GetString("creditDecision") {
				t.Fatalf("%s permutation %v decision = (%v,%v)", programName, permutation, decision, err)
			}
		}
	}
}

func stringPermutations(values []string) [][]string {
	values = append([]string(nil), values...)
	var result [][]string
	var visit func(int)
	visit = func(index int) {
		if index == len(values) {
			result = append(result, append([]string(nil), values...))
			return
		}
		for candidate := index; candidate < len(values); candidate++ {
			values[index], values[candidate] = values[candidate], values[index]
			visit(index + 1)
			values[index], values[candidate] = values[candidate], values[index]
		}
	}
	visit(0)
	return result
}

type configAssignment struct {
	key, value string
}

type configOracleOutcome struct {
	actual                                      []string
	applicationValid, schemaSatisfied           bool
	intentPreserved, idempotent, outcome        bool
	changed, constraintFailures, intentFailures int
}

func oracleAssignments(components []string, store *unit.Store) []configAssignment {
	assignments := make([]configAssignment, 0, len(components))
	for _, name := range components {
		u := store.Get(name)
		if u != nil {
			assignments = append(assignments, configAssignment{u.GetString("repairKey"), u.GetString("repairValue")})
		}
	}
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].key == assignments[j].key {
			return assignments[i].value < assignments[j].value
		}
		return assignments[i].key < assignments[j].key
	})
	return assignments
}

func sameAssignments(left, right []configAssignment) bool {
	if len(left) != len(right) {
		return false
	}
	right = append([]configAssignment(nil), right...)
	sort.Slice(right, func(i, j int) bool {
		if right[i].key == right[j].key {
			return right[i].value < right[j].value
		}
		return right[i].key < right[j].key
	})
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func oracleConfig(data []string) (map[string]string, bool) {
	values := make(map[string]string, len(data))
	for _, record := range data {
		parts := strings.Split(record, "=")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, false
		}
		if _, exists := values[parts[0]]; exists {
			return nil, false
		}
		values[parts[0]] = parts[1]
	}
	return values, true
}

func oracleCanonical(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, len(keys))
	for index, key := range keys {
		out[index] = key + "=" + values[key]
	}
	return out
}

func oracleSatisfies(values map[string]string, schema []string) bool {
	fields := map[string]string{}
	required := map[string]bool{}
	var constraints []string
	for _, record := range schema {
		parts := strings.Split(record, ":")
		switch parts[0] {
		case "field":
			if len(parts) != 3 && len(parts) != 5 {
				return false
			}
			fields[parts[1]] = strings.Join(parts[2:], ":")
		case "required":
			if len(parts) != 2 {
				return false
			}
			required[parts[1]] = true
		case "protected":
			if len(parts) != 2 {
				return false
			}
		case "eq-if", "min-if":
			if len(parts) != 2 {
				return false
			}
			constraints = append(constraints, record)
		default:
			return false
		}
	}
	for key, value := range values {
		kind, exists := fields[key]
		if !exists || !oracleTypedValue(value, kind) {
			return false
		}
	}
	for key := range required {
		if _, exists := values[key]; !exists {
			return false
		}
	}
	for _, record := range constraints {
		parts := strings.SplitN(record, ":", 2)
		assignments := strings.Split(parts[1], ",")
		guard := strings.Split(assignments[0], "=")
		target := strings.Split(assignments[1], "=")
		if values[guard[0]] != guard[1] {
			continue
		}
		if parts[0] == "eq-if" {
			if values[target[0]] != target[1] {
				return false
			}
		} else {
			actual, actualErr := strconv.Atoi(values[target[0]])
			minimum, minimumErr := strconv.Atoi(target[1])
			if actualErr != nil || minimumErr != nil || actual < minimum {
				return false
			}
		}
	}
	return true
}

func oracleTypedValue(value, kind string) bool {
	switch {
	case kind == "string":
		return value != ""
	case kind == "bool":
		return value == "true" || value == "false"
	case strings.HasPrefix(kind, "int:"):
		parts := strings.Split(kind, ":")
		actual, err1 := strconv.Atoi(value)
		minimum, err2 := strconv.Atoi(parts[1])
		maximum, err3 := strconv.Atoi(parts[2])
		return err1 == nil && err2 == nil && err3 == nil && strconv.Itoa(actual) == value && actual >= minimum && actual <= maximum
	default:
		return false
	}
}

func oracleOutcome(input, schema []string, assignments []configAssignment) configOracleOutcome {
	before, ok := oracleConfig(input)
	if !ok || len(assignments) == 0 || len(assignments) > 3 {
		return configOracleOutcome{}
	}
	after := make(map[string]string, len(before)+len(assignments))
	for key, value := range before {
		after[key] = value
	}
	seen := map[string]bool{}
	for _, assignment := range assignments {
		if assignment.key == "" || assignment.value == "" || seen[assignment.key] {
			return configOracleOutcome{}
		}
		seen[assignment.key] = true
		after[assignment.key] = assignment.value
	}
	second := make(map[string]string, len(after))
	for key, value := range after {
		second[key] = value
	}
	for _, assignment := range assignments {
		second[assignment.key] = assignment.value
	}
	intent := true
	for _, record := range schema {
		if strings.HasPrefix(record, "protected:") {
			key := strings.TrimPrefix(record, "protected:")
			if before[key] != after[key] {
				intent = false
			}
		}
	}
	changed := 0
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	for key := range keys {
		if before[key] != after[key] {
			changed++
		}
	}
	satisfied := oracleSatisfies(after, schema)
	result := configOracleOutcome{
		actual: oracleCanonical(after), applicationValid: true, schemaSatisfied: satisfied,
		intentPreserved: intent, idempotent: equalStringSlices(oracleCanonical(after), oracleCanonical(second)), changed: changed,
	}
	result.outcome = satisfied && intent
	if !satisfied {
		result.constraintFailures = 1
	}
	if !intent {
		result.intentFailures = 1
	}
	return result
}

func assertConfigurationRepairExperiment(t *testing.T, store *unit.Store, corpusSize int) *unit.Unit {
	t.Helper()
	programs := configurationRepairUnits(store, "CompositeConfigurationRepair", "components")
	evidence := configurationRepairUnits(store, "ConfigurationRepairEvidence", "program")
	results := configurationRepairUnits(store, "ConfigurationRepairResult", "program")
	observations := configurationRepairUnits(store, "ConfigurationRepairObservation", "program")
	schemas := configurationRepairUnits(store, "ConfigurationRepairSchema", "program")
	if len(programs) != 41 || len(evidence) != 41 || len(results) != 41*corpusSize || len(observations) != 41*corpusSize || len(schemas) != 1 {
		t.Fatalf("configuration repair matrix = %d programs, %d evidence, %d results, %d observations, %d schemas", len(programs), len(evidence), len(results), len(observations), len(schemas))
	}
	conjectures := 0
	for _, name := range store.All() {
		if strings.HasPrefix(name, "Conjec-ConfigurationRepairPlanSatisfiesCorpus-") {
			conjectures++
		}
	}
	if conjectures != 1 {
		t.Fatalf("configuration repair conjectures = %d, want 1", conjectures)
	}
	examples := configurationRepairUnits(store, "ConfigurationRepairExample", "configuration")
	primitives := configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey")
	wantPlans := independentlyEnumeratedPlans(primitives, store)
	gotPlans := map[string]bool{}
	for _, programName := range programs {
		program := store.Get(programName)
		assignments := oracleAssignments(program.GetStrings("components"), store)
		if len(assignments) < 1 || len(assignments) > 3 {
			t.Fatalf("%s assignments = %v", programName, assignments)
		}
		planKey := assignmentKey(assignments)
		if gotPlans[planKey] {
			t.Fatalf("duplicate semantic plan %s", planKey)
		}
		gotPlans[planKey] = true
		wantSupport, wantConstraints, wantIntent, wantChanged := 0, 0, 0, 0
		for _, exampleName := range examples {
			example := store.Get(exampleName)
			schema := store.Get(example.GetString("schema"))
			if schema == nil {
				t.Fatalf("example %s has missing schema", exampleName)
			}
			want := oracleOutcome(example.GetStrings("configuration"), schema.GetStrings("data"), assignments)
			if want.outcome {
				wantSupport++
			}
			wantConstraints += want.constraintFailures
			wantIntent += want.intentFailures
			wantChanged += want.changed
			observation := findConfigurationRepairUnit(store, "ConfigurationRepairObservation", programName, exampleName)
			result := findConfigurationRepairUnit(store, "ConfigurationRepairResult", programName, exampleName)
			if observation == nil || result == nil {
				t.Fatalf("missing artifacts for %s/%s", programName, exampleName)
			}
			if !equalStringSlices(result.GetStrings("data"), want.actual) || result.GetString("schema") != schema.Name {
				t.Fatalf("result %s disagrees with independent oracle: got %v want %v", result.Name, result.GetStrings("data"), want.actual)
			}
			wantStatus := oracleStatus(want)
			if observation.GetBool("applicationValid") != want.applicationValid || observation.GetBool("schemaSatisfied") != want.schemaSatisfied || observation.GetBool("intentPreserved") != want.intentPreserved || observation.GetBool("idempotent") != want.idempotent || observation.GetBool("outcome") != want.outcome || observation.GetInt("changedCount") != want.changed || observation.GetString("status") != wantStatus || observation.GetString("resultUnit") != result.Name {
				t.Fatalf("observation %s disagrees with independent oracle", observation.Name)
			}
		}
		if program.GetInt("supportCount") != wantSupport || program.GetInt("constraintFailureCount") != wantConstraints || program.GetInt("intentFailureCount") != wantIntent || program.GetInt("invalidApplicationCount") != 0 || program.GetInt("idempotenceFailureCount") != 0 || program.GetInt("changedKeyTotal") != wantChanged || program.GetInt("corpusSize") != corpusSize || program.GetInt("evaluatedCount") != corpusSize {
			t.Fatalf("%s aggregate disagrees with independent oracle", programName)
		}
		wantWorth := 300
		if wantSupport == corpusSize && wantConstraints == 0 && wantIntent == 0 {
			wantWorth = 800
		}
		if program.Worth() != wantWorth {
			t.Fatalf("%s worth = %d, want %d", programName, program.Worth(), wantWorth)
		}
		applications, ok := program.Get("applics").([]map[string]any)
		if !ok || len(applications) != corpusSize {
			t.Fatalf("%s applications = %#v", programName, applications)
		}
		seenApplications := map[string]bool{}
		for _, application := range applications {
			args, _ := application["args"].([]string)
			output, _ := application["output"].(string)
			direct, _ := application["direct"].(bool)
			if len(args) != 1 || !direct || seenApplications[args[0]] {
				t.Fatalf("%s malformed application %#v", programName, application)
			}
			result := store.Get(output)
			if result == nil || result.GetString("program") != programName || result.GetString("example") != args[0] {
				t.Fatalf("%s unlinked application %#v", programName, application)
			}
			seenApplications[args[0]] = true
		}
		evidenceUnit := store.Get(program.GetString("evidenceUnit"))
		if evidenceUnit == nil || evidenceUnit.GetInt("supportCount") != wantSupport || evidenceUnit.GetInt("constraintFailureCount") != wantConstraints || evidenceUnit.GetInt("intentFailureCount") != wantIntent || evidenceUnit.GetInt("changedKeyTotal") != wantChanged || len(evidenceUnit.GetStrings("trainingExamples")) != corpusSize || len(evidenceUnit.GetStrings("resultUnits")) != corpusSize || len(evidenceUnit.GetStrings("observations")) != corpusSize {
			t.Fatalf("%s evidence disagrees with independent oracle", programName)
		}
		creditors := program.GetStrings("creditors")
		roles := program.GetStrings("creditRoles")
		if len(creditors) != len(assignments)+1 || creditors[0] != "H-ComposeConfigurationRepairPlans" {
			t.Fatalf("%s creditors = %v", programName, creditors)
		}
		if len(roles) != len(assignments)+1 || roles[0] != "synthesis" {
			t.Fatalf("%s credit roles = %v", programName, roles)
		}
		for index, component := range program.GetStrings("components") {
			if creditors[index+1] != component || roles[index+1] != "repair" {
				t.Fatalf("%s attribution alignment = creditors %v roles %v", programName, creditors, roles)
			}
		}
		repairs := make([]configvocab.Repair, len(assignments))
		for index, assignment := range assignments {
			repairs[index] = configvocab.Repair{Key: assignment.key, Value: assignment.value}
		}
		wantDecision, err := configvocab.DecisionKey(repairs)
		if err != nil || program.GetString("creditContext") != configvocab.CreditContext || program.GetString("creditDecision") != wantDecision || !credit.ValidDeclaration(program.GetString("creditContext"), program.GetString("creditDecision"), creditors, roles) {
			t.Fatalf("%s has invalid contextual declaration", programName)
		}
	}
	if len(wantPlans) != 41 || len(gotPlans) != len(wantPlans) {
		t.Fatalf("semantic plan set sizes = got %d want %d", len(gotPlans), len(wantPlans))
	}
	for key := range wantPlans {
		if !gotPlans[key] {
			t.Fatalf("missing independently enumerated plan %s", key)
		}
	}
	schema := store.Get(schemas[0])
	target := store.Get(schema.GetString("program"))
	if target == nil || schema.GetString("evidenceUnit") != target.GetString("evidenceUnit") || schema.GetInt("creationWorth") != 800 || schema.GetInt("lastRewardedWorth") != 800 {
		t.Fatalf("promoted schema %s has inconsistent provenance", schema.Name)
	}
	for _, name := range store.All() {
		if strings.HasPrefix(name, "Conjec-ConfigurationRepairPlanSatisfiesCorpus-") && !containsString(store.Get(name).GetStrings("evidence"), target.GetString("evidenceUnit")) {
			t.Fatalf("conjecture %s does not cite target evidence", name)
		}
	}
	return target
}

func independentlyEnumeratedPlans(primitives []string, store *unit.Store) map[string]bool {
	assignments := oracleAssignments(primitives, store)
	plans := map[string]bool{}
	for first := range assignments {
		plans[assignmentKey([]configAssignment{assignments[first]})] = true
		for second := first + 1; second < len(assignments); second++ {
			plans[assignmentKey([]configAssignment{assignments[first], assignments[second]})] = true
			for third := second + 1; third < len(assignments); third++ {
				plans[assignmentKey([]configAssignment{assignments[first], assignments[second], assignments[third]})] = true
			}
		}
	}
	return plans
}

func assignmentKey(assignments []configAssignment) string {
	assignments = append([]configAssignment(nil), assignments...)
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].key == assignments[j].key {
			return assignments[i].value < assignments[j].value
		}
		return assignments[i].key < assignments[j].key
	})
	parts := make([]string, len(assignments))
	for index, assignment := range assignments {
		parts[index] = strconv.Quote(assignment.key) + "=" + strconv.Quote(assignment.value)
	}
	return strings.Join(parts, ";")
}

func oracleStatus(outcome configOracleOutcome) string {
	if !outcome.applicationValid {
		return "invalid-application"
	}
	if !outcome.idempotent {
		return "non-idempotent"
	}
	if !outcome.schemaSatisfied && !outcome.intentPreserved {
		return "constraint-and-intent-failure"
	}
	if !outcome.schemaSatisfied {
		return "constraint-failure"
	}
	if !outcome.intentPreserved {
		return "intent-failure"
	}
	return "success"
}

func findConfigurationRepairUnit(store *unit.Store, category, program, example string) *unit.Unit {
	for _, name := range configurationRepairUnits(store, category, "program") {
		u := store.Get(name)
		if u.GetString("program") == program && u.GetString("example") == example {
			return u
		}
	}
	return nil
}

func findProgramByAssignments(store *unit.Store, assignments []configAssignment) *unit.Unit {
	for _, name := range configurationRepairUnits(store, "CompositeConfigurationRepair", "components") {
		u := store.Get(name)
		if sameAssignments(oracleAssignments(u.GetStrings("components"), store), assignments) {
			return u
		}
	}
	return nil
}

type heldOutConfigCase struct {
	name            string
	input, expected []string
	schema          []string
	changed         int
}

func seedHeldOutCases() []heldOutConfigCase {
	service := []string{
		"field:environment:string", "field:tls:bool", "field:service_port:int:1:65535", "field:replicas:int:0:10", "field:admin_public:bool", "field:redirect_http:bool",
		"required:environment", "required:tls", "required:service_port", "required:replicas", "required:admin_public", "required:redirect_http", "protected:environment", "protected:tls",
		"eq-if:tls=true,service_port=443", "min-if:environment=production,replicas=2", "eq-if:environment=production,admin_public=false",
	}
	gateway := append(append([]string(nil), service[:6]...), "field:route_count:int:0:100")
	gateway = append(gateway, append([]string{"required:environment", "required:tls", "required:service_port", "required:replicas", "required:admin_public", "required:redirect_http", "required:route_count", "protected:environment", "protected:tls"}, service[14:]...)...)
	return []heldOutConfigCase{
		{"valid-service", []string{"environment=production", "tls=true", "service_port=443", "replicas=2", "admin_public=false", "redirect_http=true"}, []string{"admin_public=false", "environment=production", "redirect_http=true", "replicas=2", "service_port=443", "tls=true"}, service, 0},
		{"port-only", []string{"environment=production", "tls=true", "service_port=80", "replicas=4", "admin_public=false", "redirect_http=false"}, []string{"admin_public=false", "environment=production", "redirect_http=false", "replicas=2", "service_port=443", "tls=true"}, service, 2},
		{"production-obligations", []string{"environment=production", "tls=true", "service_port=443", "replicas=1", "admin_public=true", "redirect_http=false"}, []string{"admin_public=false", "environment=production", "redirect_http=false", "replicas=2", "service_port=443", "tls=true"}, service, 2},
		{"gateway-all", []string{"environment=production", "tls=true", "service_port=80", "replicas=0", "admin_public=true", "redirect_http=true", "route_count=99"}, []string{"admin_public=false", "environment=production", "redirect_http=true", "replicas=2", "route_count=99", "service_port=443", "tls=true"}, gateway, 3},
	}
}

func configDSLList(values []string) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Quote(value)
	}
	return strings.Join(parts, " ") + fmt.Sprintf(" %d list-of", len(values))
}

func dslStringList(value dsl.Value) ([]string, bool) {
	if value.Kind() != dsl.VList {
		return nil, false
	}
	items := value.AsList()
	out := make([]string, len(items))
	for index, item := range items {
		if item.Kind() != dsl.VString {
			return nil, false
		}
		out[index] = item.AsString()
	}
	return out, true
}

func assertConfigurationRepairHeldOut(t *testing.T, vm *dsl.VM, program *unit.Unit, cases []heldOutConfigCase) {
	t.Helper()
	beforeCount := vm.Store.Count()
	for _, testCase := range cases {
		value, err := vm.Execute(configDSLList(testCase.input) + " " + strconv.Quote(program.Name) + " apply-op")
		actual, ok := dslStringList(value)
		if err != nil || !ok || !equalStringSlices(actual, testCase.expected) {
			t.Fatalf("held-out %s = (%v,%v), want %v", testCase.name, actual, err, testCase.expected)
		}
		outcome := oracleOutcome(testCase.input, testCase.schema, oracleAssignments(program.GetStrings("components"), vm.Store))
		if !outcome.schemaSatisfied || !outcome.intentPreserved || !outcome.idempotent || outcome.changed != testCase.changed || !equalStringSlices(outcome.actual, testCase.expected) {
			t.Fatalf("held-out oracle %s = %#v", testCase.name, outcome)
		}
	}
	if vm.Store.Count() != beforeCount {
		t.Fatal("held-out execution created store units or evidence")
	}
}

func configurationRepairSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	snapshot := map[string]map[string]any{}
	for _, name := range store.All() {
		if store.IsA(name, "PrimitiveConfigurationRepair") || store.IsA(name, "CompositeConfigurationRepair") || store.IsA(name, "ConfigurationRepairEvidence") || store.IsA(name, "ConfigurationRepairResult") || store.IsA(name, "ConfigurationRepairObservation") || store.IsA(name, "ConfigurationRepairSchema") || strings.HasPrefix(name, "Conjec-ConfigurationRepairPlanSatisfiesCorpus-") {
			snapshot[name] = store.Get(name).Slots
		}
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func configurationRepairCreditSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	var names []string
	for _, name := range store.Examples(credit.Category) {
		if name != credit.Category {
			names = append(names, name)
		}
	}
	if len(names) != 5 {
		t.Fatalf("contextual credit record count = %d, want 5: %v", len(names), names)
	}
	schemas := configurationRepairUnits(store, "ConfigurationRepairSchema", "program")
	if len(schemas) != 1 {
		t.Fatalf("promoted schema count = %d while checking credit", len(schemas))
	}
	target := store.Get(schemas[0]).GetString("program")
	snapshot := map[string]any{}
	for _, name := range names {
		record := store.Get(name)
		if record.GetInt("evidenceCount") != 1 || record.GetString("lastSourceUnit") != target {
			t.Fatalf("credit record %s evidence/source = %d/%q, want 1/%q", name, record.GetInt("evidenceCount"), record.GetString("lastSourceUnit"), target)
		}
		snapshot[name] = record.Slots
	}
	for _, name := range append([]string{"H-ComposeConfigurationRepairPlans"}, configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey")...) {
		snapshot["worth:"+name] = store.Get(name).Worth()
	}
	targetUnit := store.Get(target)
	snapshot["target-worth"] = targetUnit.Worth()
	snapshot["target-last-rewarded-worth"] = targetUnit.GetInt("lastRewardedWorth")
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

type configSchemaSpec struct {
	name string
	data []string
}

type configExampleSpec struct {
	name, schema string
	data         []string
}

type configRepairSpec struct {
	name, key, value string
}

func TestConfigurationRepairAlternateRuntimeCorpus(t *testing.T) {
	store := loadConfigurationRepair(t)
	replaceConfigurationRepairTrial(store,
		[]configSchemaSpec{
			{"WorkerSchema", workerSchema(false)},
			{"WorkerExtendedSchema", workerSchema(true)},
		},
		[]configExampleSpec{
			{"WorkerExampleD", "WorkerExtendedSchema", []string{"stage=live", "encryption=true", "ingress_port=8080", "workers=3", "debug=false", "metrics=false", "queue_depth=80"}},
			{"WorkerExampleB", "WorkerSchema", []string{"stage=live", "encryption=true", "ingress_port=8443", "workers=0", "debug=true", "metrics=false"}},
			{"WorkerExampleA", "WorkerSchema", []string{"stage=live", "encryption=true", "ingress_port=8080", "workers=0", "debug=true", "metrics=false"}},
			{"WorkerExampleC", "WorkerExtendedSchema", []string{"stage=live", "encryption=true", "ingress_port=8080", "workers=3", "debug=true", "metrics=false", "queue_depth=60"}},
		},
		[]configRepairSpec{
			{"WorkerAlias-Z", "metrics", "true"},
			{"WorkerAlias-4", "debug", "false"},
			{"WorkerAlias.2", "stage", "test"},
			{"WorkerAlias:A", "ingress_port", "8443"},
			{"WorkerAlias_Q", "encryption", "false"},
			{"WorkerAlias-1", "workers", "3"},
		})
	store, _ = runConfigurationRepair(t, store, false)
	target := assertConfigurationRepairExperiment(t, store, 4)
	if got := oracleAssignments(target.GetStrings("components"), store); !sameAssignments(got, []configAssignment{{"debug", "false"}, {"ingress_port", "8443"}, {"workers", "3"}}) {
		t.Fatalf("alternate promoted assignments = %v", got)
	}
	shortcut := findProgramByAssignments(store, []configAssignment{{"encryption", "false"}, {"stage", "test"}})
	if shortcut == nil || shortcut.GetInt("constraintFailureCount") != 0 || shortcut.GetInt("intentFailureCount") != 4 || shortcut.GetInt("supportCount") != 0 {
		t.Fatalf("alternate protected shortcut evidence = %#v", shortcut)
	}
}

func TestConfigurationRepairOpaqueAliasesCollisionsAndPrimitiveDeletion(t *testing.T) {
	store := loadConfigurationRepair(t)
	for _, name := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		store.Delete(name)
	}
	aliases := []configRepairSpec{
		{"Alias:gamma", "admin_public", "false"},
		{"Alias.dot", "service_port", "443"},
		{"Alias-alpha", "replicas", "2"},
		{"Alias_ignored", "redirect_http", "true"},
		{"Alias protected z", "tls", "false"},
		{"Alias/protected/y", "environment", "development"},
	}
	for _, spec := range aliases {
		putConfigurationRepair(store, spec)
	}
	temporaryVM := dsl.NewVM(store, agenda.New(), nil)
	targetNames := []string{"Alias.dot", "Alias-alpha", "Alias:gamma"}
	nameValue, err := temporaryVM.Execute(configDSLList(targetNames) + " config-plan-name")
	if err != nil || nameValue.Kind() != dsl.VString {
		t.Fatalf("target base = (%v,%v)", nameValue, err)
	}
	targetBase := nameValue.AsString()
	occupied := unit.New(targetBase)
	occupied.Set("sentinel", "preserve")
	store.Put(occupied)
	artifactValue, err := temporaryVM.Execute(strconv.Quote("Result") + " " + strconv.Quote(targetBase+"-collision-1") + " " + strconv.Quote("ServiceExampleA") + " config-artifact-name")
	if err != nil || artifactValue.Kind() != dsl.VString {
		t.Fatalf("artifact base = (%v,%v)", artifactValue, err)
	}
	artifactBase := artifactValue.AsString()
	artifactOccupied := unit.New(artifactBase)
	artifactOccupied.Set("sentinel", "preserve")
	store.Put(artifactOccupied)

	store, eng := runConfigurationRepair(t, store, false)
	target := assertConfigurationRepairExperiment(t, store, 4)
	if target.Name != targetBase+"-collision-1" {
		t.Fatalf("collision target = %s, want %s-collision-1", target.Name, targetBase)
	}
	if store.Get(targetBase).GetString("sentinel") != "preserve" || store.Get(artifactBase).GetString("sentinel") != "preserve" {
		t.Fatal("occupied candidate or artifact was overwritten")
	}
	wantDecision, err := configvocab.DecisionKey([]configvocab.Repair{{Key: "service_port", Value: "443"}, {Key: "replicas", Value: "2"}, {Key: "admin_public", Value: "false"}})
	if err != nil || target.GetString("creditDecision") != wantDecision {
		t.Fatalf("alias decision = %q, want %q", target.GetString("creditDecision"), wantDecision)
	}
	for _, name := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		store.Delete(name)
	}
	input := []string{"environment=production", "tls=true", "service_port=80", "replicas=0", "admin_public=true", "redirect_http=false"}
	want := []string{"admin_public=false", "environment=production", "redirect_http=false", "replicas=2", "service_port=443", "tls=true"}
	value, err := eng.VM.Execute(configDSLList(input) + " " + strconv.Quote(target.Name) + " apply-op")
	got, ok := dslStringList(value)
	if err != nil || !ok || !equalStringSlices(got, want) {
		t.Fatalf("self-contained plan after primitive deletion = (%v,%v), want %v", got, err, want)
	}
}

func TestConfigurationRepairMalformedAndNoSolutionCorporaDoNotPromote(t *testing.T) {
	tests := map[string]struct {
		alter                          func(*unit.Store)
		programs, results, invalidEach int
	}{
		"malformed-configuration": {alter: func(store *unit.Store) {
			store.Get("ServiceExampleA").Set("configuration", []string{"not-an-assignment"})
		}, programs: 41, results: 123, invalidEach: 1},
		"no-bounded-solution": {alter: func(store *unit.Store) {
			schema := store.Get("ServiceSchemaV1")
			schema.Set("data", append(schema.GetStrings("data"), "eq-if:environment=production,redirect_http=true"))
		}, programs: 41, results: 164},
		"invalid-schema": {alter: func(store *unit.Store) {
			store.Get("ServiceSchemaV1").Set("data", []string{"field:replicas:int:00:10"})
		}, programs: 41, results: 82, invalidEach: 2},
		"conflicting-primitive": {alter: func(store *unit.Store) {
			store.Get("ConfigRepairMu").Set("repairKey", "replicas")
			store.Get("ConfigRepairMu").Set("repairValue", "2")
			store.Get("ConfigRepairMu").Set("defn", `"replicas" "2" config-set`)
		}, programs: 36, results: 144},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			store := loadConfigurationRepair(t)
			testCase.alter(store)
			store, _ = runConfigurationRepair(t, store, false)
			if got := configurationRepairUnits(store, "ConfigurationRepairSchema", "program"); len(got) != 0 {
				t.Fatalf("control promoted schemas %v", got)
			}
			programs := configurationRepairUnits(store, "CompositeConfigurationRepair", "components")
			evidence := configurationRepairUnits(store, "ConfigurationRepairEvidence", "program")
			results := configurationRepairUnits(store, "ConfigurationRepairResult", "program")
			observations := configurationRepairUnits(store, "ConfigurationRepairObservation", "program")
			if len(programs) != testCase.programs || len(evidence) != testCase.programs || len(results) != testCase.results || len(observations) != testCase.programs*4 {
				t.Fatalf("control matrix = programs %d evidence %d results %d observations %d", len(programs), len(evidence), len(results), len(observations))
			}
			for _, programName := range programs {
				program := store.Get(programName)
				if program.GetInt("invalidApplicationCount") != testCase.invalidEach || program.GetInt("evaluatedCount") != 4 || program.GetInt("corpusSize") != 4 {
					t.Fatalf("%s control aggregate invalid/evaluated/corpus = %d/%d/%d", programName, program.GetInt("invalidApplicationCount"), program.GetInt("evaluatedCount"), program.GetInt("corpusSize"))
				}
			}
		})
	}
}

func TestConfigurationRepairStoresAreDeterministicAndMutationInactive(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprintf("mutation=%v", mutate), func(t *testing.T) {
			first, _ := runConfigurationRepair(t, loadConfigurationRepair(t), mutate)
			second, _ := runConfigurationRepair(t, loadConfigurationRepair(t), mutate)
			firstJSON, err := first.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			secondJSON, err := second.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(firstJSON) != string(secondJSON) {
				t.Fatal("configuration repair store snapshots differ")
			}
			if mutate {
				for _, unitName := range first.All() {
					if first.Get(unitName).GetString("mutant_of") != "" {
						t.Fatalf("mutation-enabled inactive control created %s", unitName)
					}
				}
			}
		})
	}
}

func workerSchema(extended bool) []string {
	schema := []string{
		"field:stage:string", "field:encryption:bool", "field:ingress_port:int:1:65535", "field:workers:int:0:20", "field:debug:bool", "field:metrics:bool",
		"required:stage", "required:encryption", "required:ingress_port", "required:workers", "required:debug", "required:metrics",
		"protected:stage", "protected:encryption",
		"eq-if:encryption=true,ingress_port=8443", "min-if:stage=live,workers=3", "eq-if:stage=live,debug=false",
	}
	if !extended {
		return schema
	}
	return append(append([]string(nil), schema[:6]...), append([]string{"field:queue_depth:int:0:100"}, append(schema[6:12], append([]string{"required:queue_depth"}, schema[12:]...)...)...)...)
}

func replaceConfigurationRepairTrial(store *unit.Store, schemas []configSchemaSpec, examples []configExampleSpec, repairs []configRepairSpec) {
	for _, name := range configurationRepairUnits(store, "ConfigurationSchema", "data") {
		store.Delete(name)
	}
	for _, name := range configurationRepairUnits(store, "ConfigurationRepairExample", "configuration") {
		store.Delete(name)
	}
	for _, name := range configurationRepairUnits(store, "PrimitiveConfigurationRepair", "repairKey") {
		store.Delete(name)
	}
	for _, spec := range schemas {
		u := unit.New(spec.name)
		u.SetWorth(700)
		u.Set("isA", []string{"ConfigurationSchema", "Anything"})
		u.Set("data", spec.data)
		store.Put(u)
	}
	for _, spec := range examples {
		u := unit.New(spec.name)
		u.SetWorth(650)
		u.Set("isA", []string{"ConfigurationRepairExample", "Anything"})
		u.Set("schema", spec.schema)
		u.Set("configuration", spec.data)
		store.Put(u)
	}
	for _, spec := range repairs {
		putConfigurationRepair(store, spec)
	}
}

func putConfigurationRepair(store *unit.Store, spec configRepairSpec) {
	u := unit.New(spec.name)
	u.SetWorth(600)
	u.Set("isA", []string{"PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"})
	u.Set("domain", []string{"Configuration"})
	u.Set("range", []string{"Configuration"})
	u.Set("arity", 1)
	u.Set("repairKey", spec.key)
	u.Set("repairValue", spec.value)
	u.Set("defn", strconv.Quote(spec.key)+" "+strconv.Quote(spec.value)+" config-set")
	store.Put(u)
}
