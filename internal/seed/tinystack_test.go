package seed

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	programvocab "github.com/chazu/nous/internal/vocab/programsynth"
)

func loadTinyStack(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "tinystack"); err != nil {
		t.Fatal(err)
	}
	return store
}

func runTinyStack(t *testing.T, store *unit.Store) (*unit.Store, *engine.Engine) {
	return runTinyStackLanes(t, store, true)
}

func runTinyStackLanes(t *testing.T, store *unit.Store, simplify bool) (*unit.Store, *engine.Engine) {
	return runTinyStackDescriptor(t, store, "StackSynthesisExperiment", simplify)
}

func runTinyStackDescriptor(t *testing.T, store *unit.Store, experimentName string, simplify bool) (*unit.Store, *engine.Engine) {
	t.Helper()
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		t.Fatal(err)
	}
	experiment := store.Get(experimentName)
	eng.WorkOnTask(&agenda.Task{Priority: experiment.GetInt("synthesisPriority"), UnitName: experimentName, SlotName: experiment.GetString("synthesisTaskSlot")})
	for _, candidate := range experiment.GetStrings("candidateUnits") {
		eng.WorkOnTask(&agenda.Task{Priority: experiment.GetInt("evaluationPriority"), UnitName: candidate, SlotName: experiment.GetString("evaluationTaskSlot")})
	}
	eng.WorkOnTask(&agenda.Task{Priority: experiment.GetInt("finalizationPriority"), UnitName: experimentName, SlotName: experiment.GetString("finalizationTaskSlot")})
	if simplify {
		eng.WorkOnTask(&agenda.Task{Priority: experiment.GetInt("simplificationPriority"), UnitName: experimentName, SlotName: experiment.GetString("simplificationTaskSlot")})
	}
	return store, eng
}

func tinyStackMembers(store *unit.Store, category string) []*unit.Unit {
	var members []*unit.Unit
	for _, name := range store.Examples(category) {
		if name != category {
			members = append(members, store.Get(name))
		}
	}
	return members
}

func stackProgramBySemantics(store *unit.Store, semantics ...string) *unit.Unit {
	return stackProgramBySemanticsIn(store, "StackProgramCandidate", semantics...)
}

func stackProgramBySemanticsIn(store *unit.Store, category string, semantics ...string) *unit.Unit {
	for _, program := range tinyStackMembers(store, category) {
		if reflect.DeepEqual(program.GetStrings("semanticSequence"), semantics) {
			return program
		}
	}
	return nil
}

func primitiveBySemantic(store *unit.Store, semantic string) *unit.Unit {
	return primitiveBySemanticIn(store, "StackInstruction", semantic)
}

func primitiveBySemanticIn(store *unit.Store, category, semantic string) *unit.Unit {
	for _, primitive := range tinyStackMembers(store, category) {
		if primitive.GetString("semanticOpcode") == semantic {
			return primitive
		}
	}
	return nil
}

func simplificationPairs(store *unit.Store) []string {
	return simplificationPairsFor(store, store.Get("StackSynthesisExperiment"))
}

func simplificationPairsFor(store *unit.Store, experiment *unit.Unit) []string {
	var got []string
	for _, schema := range tinyStackMembers(store, experiment.GetString("simplificationSchemaCategory")) {
		long := store.Get(schema.GetString("longProgram"))
		short := store.Get(schema.GetString("shortProgram"))
		got = append(got, fmt.Sprint(long.GetStrings("semanticSequence"))+"->"+short.GetString("semanticOpcode"))
	}
	sort.Strings(got)
	return got
}

func TestTinyStackSeedSynthesisAndSimplification(t *testing.T) {
	store, eng := runTinyStack(t, loadTinyStack(t))
	for _, foreign := range []string{"MathConcept", "Protocol", "RewriteString", "Configuration"} {
		if store.Has(foreign) {
			t.Fatalf("tiny-stack vocabulary loaded foreign unit %s", foreign)
		}
	}
	experiment := store.Get("StackSynthesisExperiment")
	if !experiment.GetBool("generationComplete") || !experiment.GetBool("finalizationComplete") || !experiment.GetBool("simplificationComplete") {
		first := store.Get(experiment.GetStrings("candidateUnits")[0])
		t.Fatalf("experiment incomplete: generation=%v finalizationScheduled=%v finalization=%v simplification=%v evaluated=%d expected=%d first=(evaluated=%v evidence=%q)", experiment.GetBool("generationComplete"), experiment.GetBool("finalizationScheduled"), experiment.GetBool("finalizationComplete"), experiment.GetBool("simplificationComplete"), experiment.GetInt("evaluatedCandidateCount"), experiment.GetInt("expectedCandidateCount"), first.GetBool("evaluatedProgramCorpus"), first.GetString("evidenceUnit"))
	}
	if got := len(experiment.GetStrings("candidateUnits")); got != 399 {
		t.Fatalf("candidate count = %d, want 399", got)
	}
	histogram := map[int]int{}
	for _, program := range tinyStackMembers(store, "StackProgramCandidate") {
		histogram[program.GetInt("supportCount")]++
	}
	if !reflect.DeepEqual(histogram, map[int]int{0: 379, 1: 18, 4: 2}) {
		t.Fatalf("support histogram = %v", histogram)
	}
	selected := experiment.GetStrings("selectedPrograms")
	if len(selected) != 1 || !reflect.DeepEqual(store.Get(selected[0]).GetStrings("semanticSequence"), []string{"over", "add"}) {
		t.Fatalf("selected programs = %v", selected)
	}
	if experiment.GetString("selectionStatus") != "selected" || experiment.GetInt("minimumLength") != 2 {
		t.Fatalf("selection = %q length %d", experiment.GetString("selectionStatus"), experiment.GetInt("minimumLength"))
	}
	if got := len(tinyStackMembers(store, "StackProgramSchema")); got != 1 {
		t.Fatalf("program schemas = %d, want 1", got)
	}
	if experiment.GetInt("simplificationExecutionObservationCount") != 336 || experiment.GetInt("simplificationExecutionResultCount") != 213 || experiment.GetInt("simplificationUndefinedExecutionCount") != 123 || experiment.GetInt("simplificationPairCount") != 343 || experiment.GetInt("simplificationComparisonObservationCount") != 2058 {
		t.Fatalf("simplification counts: observations=%d results=%d undefined=%d pairs=%d comparisons=%d", experiment.GetInt("simplificationExecutionObservationCount"), experiment.GetInt("simplificationExecutionResultCount"), experiment.GetInt("simplificationUndefinedExecutionCount"), experiment.GetInt("simplificationPairCount"), experiment.GetInt("simplificationComparisonObservationCount"))
	}
	wantSimplifications := []string{
		"[double drop]->drop",
		"[dup add]->double",
		"[dup swap]->dup",
		"[swap add]->add",
		"[swap mul]->mul",
	}
	if got := simplificationPairs(store); !reflect.DeepEqual(got, wantSimplifications) {
		t.Fatalf("simplifications = %v", got)
	}
	assertTinyStackIndependentMatrix(t, store, "StackSynthesisExperiment", []string{"dup", "swap", "drop", "over", "add", "mul", "double"})
	for _, schema := range tinyStackMembers(store, "StackSimplificationSchema") {
		long := store.Get(schema.GetString("longProgram"))
		short := store.Get(schema.GetString("shortProgram"))
		for _, input := range exhaustiveSmallStacks() {
			left, leftDefined := oracleStackProgram(input, long.GetStrings("semanticSequence"))
			right, rightDefined := oracleStackProgram(input, []string{short.GetString("semanticOpcode")})
			if leftDefined != rightDefined || (leftDefined && !reflect.DeepEqual(left, right)) {
				t.Fatalf("held-out simplification %v -> %s failed on %v: (%v,%v) vs (%v,%v)", long.GetStrings("semanticSequence"), short.GetString("semanticOpcode"), input, left, leftDefined, right, rightDefined)
			}
		}
	}
	seedHeldOut := []stackCase{
		{[]int{10, 20}, []int{10, 30}},
		{[]int{-7, -8}, []int{-7, -15}},
		{[]int{100, 0}, []int{100, 100}},
		{[]int{3, 4, 5}, []int{3, 4, 9}},
	}
	beforeHeldOut := discoveryTinyStackSnapshot(store)
	for _, testCase := range seedHeldOut {
		for _, semantics := range [][]string{{"over", "add"}, {"over", "swap", "add"}} {
			program := stackProgramBySemantics(store, semantics...)
			value, err := eng.VM.Execute(stackDSLInts(testCase.input) + " " + strconv.Quote(program.Name) + " apply-op")
			if err != nil || !reflect.DeepEqual(dslInts(value), testCase.expected) {
				t.Fatalf("seed generated %v on %v = (%v,%v), want %v", semantics, testCase.input, value, err, testCase.expected)
			}
		}
	}
	if afterHeldOut := discoveryTinyStackSnapshot(store); beforeHeldOut != afterHeldOut {
		t.Fatal("seed held-out applications changed discovery artifacts")
	}
	winner := store.Get(selected[0])
	beforeRepeat := discoveryTinyStackSnapshot(store)
	eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: winner.Name, SlotName: "boundedProgramEvaluation"})
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramFinalization"})
	eng.WorkOnTask(&agenda.Task{Priority: 550, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramSimplification"})
	if afterRepeat := discoveryTinyStackSnapshot(store); beforeRepeat != afterRepeat {
		t.Fatal("repeated guarded evaluation/finalization/simplification changed the store")
	}
	assertTinyStackIndependentMatrix(t, store, "StackSynthesisExperiment", []string{"dup", "swap", "drop", "over", "add", "mul", "double"})
	creditEngine := engine.New(store, agenda.New())
	creditEngine.Out = io.Discard
	creditEngine.VM.Out = io.Discard
	creditEngine.MaxCycles = 11
	creditEngine.MutConfig.Enabled = false
	if err := creditEngine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.Get("H-EnumerateBoundedPrograms").Worth() != 900 || primitiveBySemantic(store, "over").Worth() != 650 || primitiveBySemantic(store, "add").Worth() != 650 {
		t.Fatalf("credited worths: heuristic=%d over=%d add=%d", store.Get("H-EnumerateBoundedPrograms").Worth(), primitiveBySemantic(store, "over").Worth(), primitiveBySemantic(store, "add").Worth())
	}
	assertContextualCredit(t, store, credit.DecisionTuple(winner.GetString("creditContext"), winner.GetString("creditDecision")), 300)
	assertContextualCredit(t, store, credit.Tuple{Context: winner.GetString("creditContext"), Subject: "H-EnumerateBoundedPrograms", Role: "synthesis"}, 150)
	assertContextualCredit(t, store, credit.Tuple{Context: winner.GetString("creditContext"), Subject: primitiveBySemantic(store, "over").Name, Role: "step-1"}, 150)
	assertContextualCredit(t, store, credit.Tuple{Context: winner.GetString("creditContext"), Subject: primitiveBySemantic(store, "add").Name, Role: "step-2"}, 150)
	if records := tinyStackMembers(store, "ContextualCredit"); len(records) != 4 {
		t.Fatalf("contextual credit record count = %d, want 4", len(records))
	} else {
		for _, record := range records {
			if record.GetString("lastSourceUnit") != winner.Name || record.GetInt("evidenceCount") != 1 {
				t.Fatalf("contextual credit provenance = %#v", record.Slots)
			}
		}
	}
	repeated := stackProgramBySemantics(store, "dup", "dup")
	if repeated == nil || !credit.ValidDeclaration(repeated.GetString("creditContext"), repeated.GetString("creditDecision"), repeated.GetStrings("creditors"), repeated.GetStrings("creditRoles")) || !reflect.DeepEqual(repeated.GetStrings("creditors")[1:], []string{primitiveBySemantic(store, "dup").Name, primitiveBySemantic(store, "dup").Name}) || !reflect.DeepEqual(repeated.GetStrings("creditRoles"), []string{"synthesis", "step-1", "step-2"}) {
		t.Fatalf("repeated occurrence declaration = %#v", repeated)
	}
	creditBefore := tinyStackCreditSnapshot(store, winner)
	secondCreditEngine := engine.New(store, agenda.New())
	secondCreditEngine.Out = io.Discard
	secondCreditEngine.VM.Out = io.Discard
	secondCreditEngine.MaxCycles = 11
	secondCreditEngine.MutConfig.Enabled = false
	if err := secondCreditEngine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if creditAfter := tinyStackCreditSnapshot(store, winner); creditBefore != creditAfter {
		t.Fatal("second eligible credit interval changed the store")
	}
}

func TestTinyStackOpaqueAliasesCollisionsAndSelfContainedWinner(t *testing.T) {
	store := loadTinyStack(t)
	var aliases []string
	for index, primitive := range tinyStackMembers(store, "StackInstruction") {
		alias := unit.New(fmt.Sprintf("OpaqueInstruction%02d", index))
		for slot, value := range primitive.Slots {
			alias.Set(slot, value)
		}
		alias.Name = fmt.Sprintf("OpaqueInstruction%02d", index)
		store.Put(alias)
		store.Delete(primitive.Name)
		aliases = append(aliases, alias.Name)
	}
	over := primitiveBySemantic(store, "over")
	add := primitiveBySemantic(store, "add")
	vmEngine := engine.New(store, agenda.New())
	definition := fmt.Sprintf("\"StackSynthesisExperiment\" %s synth-program-name", dslNameList([]string{over.Name, add.Name}))
	baseValue, err := vmEngine.VM.Execute(definition)
	if err != nil || baseValue.AsString() == "" {
		t.Fatalf("alias target name = (%v,%v)", baseValue, err)
	}
	occupied := unit.New(baseValue.AsString())
	occupied.Set("isA", []string{"Anything"})
	occupied.Set("sentinel", true)
	store.Put(occupied)
	targetName := baseValue.AsString() + "-collision-1"
	artifactSentinels := map[string]string{}
	for _, artifact := range []struct{ kind, program, example string }{
		{"Result", targetName, "StackExampleA"},
		{"Observation", targetName, "StackExampleA"},
		{"Evidence", targetName, ""},
		{"SelectedSchema", targetName, ""},
		{"Selection", "StackSynthesisExperiment", ""},
	} {
		name := synthArtifactBase(t, vmEngine, "StackSynthesisExperiment", artifact.kind, artifact.program, artifact.example)
		sentinel := unit.New(name)
		sentinel.Set("isA", []string{"Anything"})
		sentinel.Set("sentinel", true)
		store.Put(sentinel)
		artifactSentinels[artifact.kind] = name
	}
	store, eng := runTinyStackLanes(t, store, false)
	experiment := store.Get("StackSynthesisExperiment")
	selected := experiment.GetStrings("selectedPrograms")
	if len(selected) != 1 || selected[0] != baseValue.AsString()+"-collision-1" {
		t.Fatalf("collision-safe selected identity = %v, base %q", selected, baseValue.AsString())
	}
	if sentinel := store.Get(baseValue.AsString()); sentinel == nil || !sentinel.GetBool("sentinel") {
		t.Fatalf("candidate sentinel overwritten: %#v", sentinel)
	}
	winner := store.Get(selected[0])
	if !reflect.DeepEqual(winner.GetStrings("semanticSequence"), []string{"over", "add"}) {
		t.Fatalf("aliased winner semantics = %v", winner.GetStrings("semanticSequence"))
	}
	wantDecision, err := programvocab.DecisionKey(winner.GetString("synthesisMethod"), []string{"over", "add"})
	if err != nil || winner.GetString("creditDecision") != wantDecision {
		t.Fatalf("semantic decision = %q, want %q (%v)", winner.GetString("creditDecision"), wantDecision, err)
	}
	for kind, name := range artifactSentinels {
		if sentinel := store.Get(name); sentinel == nil || !sentinel.GetBool("sentinel") || len(sentinel.Slots) != 2 {
			t.Fatalf("%s sentinel overwritten: %#v", kind, sentinel)
		}
	}
	if winner.GetString("evidenceUnit") == artifactSentinels["Evidence"] || store.Get("StackSynthesisExperiment").GetString("selectionEvidenceUnit") == artifactSentinels["Selection"] {
		t.Fatal("collision-safe evidence linkage reused sentinel")
	}
	programSchemas := tinyStackMembers(store, "StackProgramSchema")
	if len(programSchemas) != 1 || programSchemas[0].Name == artifactSentinels["SelectedSchema"] {
		t.Fatalf("collision-safe schema linkage = %v", programSchemas)
	}
	foundObservation := false
	for _, observation := range tinyStackMembers(store, "StackProgramObservation") {
		if observation.GetString("program") == winner.Name && observation.GetString("example") == "StackExampleA" {
			foundObservation = true
			if observation.Name == artifactSentinels["Observation"] || observation.GetString("resultUnit") == artifactSentinels["Result"] {
				t.Fatal("collision-safe result/observation linkage reused sentinel")
			}
		}
	}
	if !foundObservation {
		t.Fatal("collision-safe target observation missing")
	}
	for _, alias := range aliases {
		store.Delete(alias)
	}
	value, err := eng.VM.Execute(`10 20 2 list-of "` + winner.Name + `" apply-op`)
	if err != nil || !reflect.DeepEqual(dslInts(value), []int{10, 30}) {
		t.Fatalf("winner after primitive deletion = (%v,%v)", value, err)
	}
}

func TestTinyStackNoSolutionControls(t *testing.T) {
	t.Run("missing-over", func(t *testing.T) {
		store := loadTinyStack(t)
		store.Delete(primitiveBySemantic(store, "over").Name)
		assertTinyStackNoSolution(t, store)
	})
	t.Run("missing-add", func(t *testing.T) {
		store := loadTinyStack(t)
		store.Delete(primitiveBySemantic(store, "add").Name)
		assertTinyStackNoSolution(t, store)
	})
	t.Run("depth-four-target", func(t *testing.T) {
		store := loadTinyStack(t)
		setStackExamples(store, []stackCase{
			{[]int{1}, []int{1, 1, 1, 1, 1}},
			{[]int{2}, []int{2, 2, 2, 2, 2}},
			{[]int{-1}, []int{-1, -1, -1, -1, -1}},
			{[]int{0}, []int{0, 0, 0, 0, 0}},
		})
		assertTinyStackNoSolution(t, store)
	})
}

func TestTinyStackDescriptorPreflightRejectsMalformedExperiments(t *testing.T) {
	tests := map[string]func(*unit.Store){
		"overlapping-categories": func(store *unit.Store) {
			store.Get("StackSynthesisExperiment").Set("candidateCategory", "StackInstruction")
		},
		"priority-order":     func(store *unit.Store) { store.Get("StackSynthesisExperiment").Set("finalizationPriority", 750) },
		"candidate-cap":      func(store *unit.Store) { store.Get("StackSynthesisExperiment").Set("candidateCap", 100) },
		"duplicate-semantic": func(store *unit.Store) { primitiveBySemantic(store, "mul").Set("semanticOpcode", "add") },
		"non-defn-primitive": func(store *unit.Store) { primitiveBySemantic(store, "mul").Set("fastAlg", "host") },
		"malformed-example":  func(store *unit.Store) { store.Get("StackExampleA").Set("input", []any{2, "3"}) },
		"comparison-cap":     func(store *unit.Store) { store.Get("StackSynthesisExperiment").Set("simplificationComparisonCap", 100) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			store := loadTinyStack(t)
			mutate(store)
			eng := engine.New(store, agenda.New())
			eng.Out = io.Discard
			eng.VM.Out = io.Discard
			eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramSynthesis"})
			if len(tinyStackMembers(store, "StackProgramCandidate")) != 0 || store.Get("StackSynthesisExperiment").GetBool("generationComplete") {
				t.Fatal("malformed descriptor allocated candidates")
			}
		})
	}
}

func TestTinyStackCompleteStoresAreDeterministic(t *testing.T) {
	first, _ := runTinyStack(t, loadTinyStack(t))
	second, _ := runTinyStack(t, loadTinyStack(t))
	firstJSON, err := first.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := second.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("complete no-mutation stores differ")
	}
}

func TestTinyStackPrematureAndForgedFinalizationCannotSelect(t *testing.T) {
	store := loadTinyStack(t)
	eng := engine.New(store, agenda.New())
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramFinalization"})
	if store.Get("StackSynthesisExperiment").GetBool("finalizationComplete") {
		t.Fatal("pre-generation finalizer selected")
	}
	eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramSynthesis"})
	experiment := store.Get("StackSynthesisExperiment")
	experiment.Set("evaluatedCandidateCount", experiment.GetInt("expectedCandidateCount"))
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: "StackSynthesisExperiment", SlotName: "boundedProgramFinalization"})
	if experiment.GetBool("finalizationComplete") || len(experiment.GetStrings("selectedPrograms")) != 0 {
		t.Fatal("count-forged finalizer selected")
	}
}

func TestTinyStackFinalizationRejectsForgedCandidateAndEvidence(t *testing.T) {
	store := loadTinyStack(t)
	eng := engine.New(store, agenda.New())
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	experiment := store.Get("StackSynthesisExperiment")
	eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: experiment.Name, SlotName: experiment.GetString("synthesisTaskSlot")})
	for _, candidate := range experiment.GetStrings("candidateUnits") {
		eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: candidate, SlotName: experiment.GetString("evaluationTaskSlot")})
	}
	assertSynthReadiness(t, eng, experiment.Name, true)

	loser := stackProgramBySemantics(store, "dup")
	loser.Set("exactProgram", true)
	assertSynthReadiness(t, eng, experiment.Name, false)
	loser.Set("exactProgram", false)

	winner := stackProgramBySemantics(store, "over", "add")
	winner.Set("semanticSequence", []string{"add", "over"})
	assertSynthReadiness(t, eng, experiment.Name, false)
	winner.Set("semanticSequence", []string{"over", "add"})

	evidence := store.Get(winner.GetString("evidenceUnit"))
	evidence.Set("supportCount", 3)
	assertSynthReadiness(t, eng, experiment.Name, false)
	evidence.Set("supportCount", 4)

	duplicate := unit.New("ForgedDuplicateStackEvidence")
	for slot, value := range evidence.Slots {
		duplicate.Set(slot, value)
	}
	store.Put(duplicate)
	assertSynthReadiness(t, eng, experiment.Name, false)
	store.Delete(duplicate.Name)

	winner.Set("creditDecision", "sha256:v1:forged")
	assertSynthReadiness(t, eng, experiment.Name, false)
	winner.Set("creditDecision", programDecision(t, winner))

	coherentLoser := stackProgramBySemantics(store, "add")
	coherentEvidence := store.Get(coherentLoser.GetString("evidenceUnit"))
	coherentLoser.Set("supportCount", 4)
	coherentLoser.Set("failureCount", 0)
	coherentLoser.Set("exactProgram", true)
	coherentEvidence.Set("supportCount", 4)
	coherentEvidence.Set("failureCount", 0)
	for _, observationName := range coherentEvidence.GetStrings("observations") {
		observation := store.Get(observationName)
		observation.Set("status", "match")
		observation.Set("outcome", true)
	}
	assertSynthReadiness(t, eng, experiment.Name, false)
	coherentLoser.Set("supportCount", 0)
	coherentLoser.Set("failureCount", 4)
	coherentLoser.Set("exactProgram", false)
	coherentEvidence.Set("supportCount", 0)
	coherentEvidence.Set("failureCount", 4)
	for _, observationName := range coherentEvidence.GetStrings("observations") {
		observation := store.Get(observationName)
		observation.Set("status", "mismatch")
		observation.Set("outcome", false)
	}
	assertSynthReadiness(t, eng, experiment.Name, true)

	longer := stackProgramBySemantics(store, "over", "swap", "add")
	longer.Set("components", winner.GetStrings("components"))
	longer.Set("semanticSequence", winner.GetStrings("semanticSequence"))
	longer.Set("defn", winner.GetString("defn"))
	longer.Set("programLength", winner.GetInt("programLength"))
	longer.Set("creditDecision", winner.GetString("creditDecision"))
	longer.Set("creditors", winner.GetStrings("creditors"))
	longer.Set("creditRoles", winner.GetStrings("creditRoles"))
	assertSynthReadiness(t, eng, experiment.Name, false)
}

func programDecision(t *testing.T, program *unit.Unit) string {
	t.Helper()
	decision, err := programvocab.DecisionKey(program.GetString("synthesisMethod"), program.GetStrings("semanticSequence"))
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func assertSynthReadiness(t *testing.T, eng *engine.Engine, experiment string, want bool) {
	t.Helper()
	value, err := eng.VM.Execute(strconv.Quote(experiment) + " synth-ready-to-finalize?")
	if err != nil || value.Kind() != dsl.VBool || value.AsBool() != want {
		t.Fatalf("synthesis readiness = (%v,%v), want %v", value, err, want)
	}
}

func TestTinyStackAlternateCorpusChangesSolutionWithoutHeuristicChanges(t *testing.T) {
	store := loadTinyStack(t)
	descriptor := store.Get("StackSynthesisExperiment")
	categoryFields := []string{
		"primitiveCategory", "candidateCategory", "exampleCategory", "valueCategory", "resultCategory",
		"observationCategory", "evidenceCategory", "selectionEvidenceCategory", "promotedSchemaCategory",
		"probeCategory", "simplificationPairCategory", "simplificationExecutionObservationCategory",
		"simplificationExecutionResultCategory", "simplificationComparisonObservationCategory",
		"simplificationEvidenceCategory", "simplificationSchemaCategory",
	}
	for index, field := range categoryFields {
		renameTinyStackCategory(store, descriptor, field, fmt.Sprintf("AlternateCategory%02d", index))
	}
	primitiveCategory := descriptor.GetString("primitiveCategory")
	for index, primitive := range tinyStackMembers(store, primitiveCategory) {
		renameTinyStackUnit(store, primitive.Name, fmt.Sprintf("AlternatePrimitive%02d", index))
	}
	double := primitiveBySemanticIn(store, primitiveCategory, "double")
	double.Set("semanticOpcode", "neg")
	double.Set("defn", `"neg" stack-exec-op`)
	examples := tinyStackMembers(store, descriptor.GetString("exampleCategory"))
	for index, example := range examples {
		renameTinyStackUnit(store, example.Name, fmt.Sprintf("AlternateExample%02d", index))
	}
	setCategoryStackExamples(store, descriptor.GetString("exampleCategory"), []stackCase{
		{[]int{5, 3}, []int{2}},
		{[]int{-2, 4}, []int{-6}},
		{[]int{0, -3}, []int{3}},
		{[]int{7, 0}, []int{7}},
	})
	probeValues := [][]int{{}, {7}, {4, 5}, {-2, 6, 3}, {1, -4}, {9, 2, -1}}
	probes := tinyStackMembers(store, descriptor.GetString("probeCategory"))
	for index, probe := range probes {
		name := fmt.Sprintf("AlternateProbe%02d", index)
		renameTinyStackUnit(store, probe.Name, name)
		store.Get(name).Set(descriptor.GetString("probeInputSlot"), probeValues[index])
	}
	descriptor.Set("experimentKey", "alternate/stack/search/v2")
	descriptor.Set("synthesisMethod", "alternate-ordered-programs/v2")
	descriptor.Set("creditContext", "alternate/stack/corpus-b/v2")
	renameTinyStackUnit(store, descriptor.Name, "AlternateProgramSearch")
	store, eng := runTinyStackDescriptor(t, store, "AlternateProgramSearch", true)
	experiment := store.Get("AlternateProgramSearch")
	if experiment.GetString("selectionStatus") != "selected" || experiment.GetInt("minimumLength") != 2 {
		t.Fatalf("alternate selection = %q length %d", experiment.GetString("selectionStatus"), experiment.GetInt("minimumLength"))
	}
	selected := experiment.GetStrings("selectedPrograms")
	if len(selected) != 1 || !reflect.DeepEqual(store.Get(selected[0]).GetStrings("semanticSequence"), []string{"neg", "add"}) {
		t.Fatalf("alternate selected programs = %v", selected)
	}
	for _, semantic := range [][]string{{"neg", "add"}, {"neg", "swap", "add"}} {
		program := stackProgramBySemanticsIn(store, experiment.GetString("candidateCategory"), semantic...)
		if program == nil || !program.GetBool("exactProgram") {
			t.Fatalf("alternate exact program %v = %#v", semantic, program)
		}
	}
	for _, testCase := range []stackCase{
		{[]int{9, -2}, []int{11}},
		{[]int{-5, -7}, []int{2}},
		{[]int{0, 0}, []int{0}},
		{[]int{8, 5, 3}, []int{8, 2}},
	} {
		beforeHeldOut := discoveryTinyStackSnapshotFor(store, experiment)
		for _, semantic := range [][]string{{"neg", "add"}, {"neg", "swap", "add"}} {
			program := stackProgramBySemanticsIn(store, experiment.GetString("candidateCategory"), semantic...)
			value, err := eng.VM.Execute(stackDSLInts(testCase.input) + " " + strconv.Quote(program.Name) + " apply-op")
			if err != nil || !reflect.DeepEqual(dslInts(value), testCase.expected) {
				t.Fatalf("alternate generated %v on %v = (%v,%v), want %v", semantic, testCase.input, value, err, testCase.expected)
			}
		}
		if afterHeldOut := discoveryTinyStackSnapshotFor(store, experiment); beforeHeldOut != afterHeldOut {
			t.Fatal("alternate held-out case entered the discovery store")
		}
	}
	wantSimplifications := []string{
		"[dup swap]->dup",
		"[neg drop]->drop",
		"[swap add]->add",
		"[swap mul]->mul",
	}
	if got := simplificationPairsFor(store, experiment); !reflect.DeepEqual(got, wantSimplifications) {
		t.Fatalf("alternate simplifications = %v", got)
	}
	if len(tinyStackMembers(store, experiment.GetString("promotedSchemaCategory"))) != 1 || len(tinyStackMembers(store, experiment.GetString("simplificationSchemaCategory"))) != 4 {
		t.Fatalf("alternate schemas: programs=%d simplifications=%d", len(tinyStackMembers(store, experiment.GetString("promotedSchemaCategory"))), len(tinyStackMembers(store, experiment.GetString("simplificationSchemaCategory"))))
	}
	assertTinyStackIndependentMatrix(t, store, experiment.Name, []string{"dup", "swap", "drop", "over", "add", "mul", "neg"})
	for _, schema := range tinyStackMembers(store, experiment.GetString("simplificationSchemaCategory")) {
		long := store.Get(schema.GetString("longProgram"))
		short := store.Get(schema.GetString("shortProgram"))
		for _, input := range exhaustiveSmallStacks() {
			left, leftDefined := oracleStackProgram(input, long.GetStrings("semanticSequence"))
			right, rightDefined := oracleStackProgram(input, []string{short.GetString("semanticOpcode")})
			if leftDefined != rightDefined || (leftDefined && !reflect.DeepEqual(left, right)) {
				t.Fatalf("alternate held-out simplification %v -> %s failed on %v", long.GetStrings("semanticSequence"), short.GetString("semanticOpcode"), input)
			}
		}
	}
}

func TestTinyStackAmbiguousCorpusPromotesAllCoMinimalPrograms(t *testing.T) {
	store := loadTinyStack(t)
	setStackExamples(store, []stackCase{
		{[]int{9, 2, 2}, []int{9, 4}},
		{[]int{-3, 0, 0}, []int{-3, 0}},
		{[]int{2, 2}, []int{4}},
		{[]int{0, 0}, []int{0}},
	})
	store, _ = runTinyStackLanes(t, store, false)
	experiment := store.Get("StackSynthesisExperiment")
	if experiment.GetString("selectionStatus") != "co-minimal" || experiment.GetInt("minimumLength") != 1 {
		t.Fatalf("ambiguous selection = %q length %d", experiment.GetString("selectionStatus"), experiment.GetInt("minimumLength"))
	}
	var selectedSemantics []string
	for _, name := range experiment.GetStrings("selectedPrograms") {
		selectedSemantics = append(selectedSemantics, store.Get(name).GetStrings("semanticSequence")[0])
	}
	sort.Strings(selectedSemantics)
	if !reflect.DeepEqual(selectedSemantics, []string{"add", "mul"}) {
		t.Fatalf("ambiguous minima = %v", selectedSemantics)
	}
	if got := len(tinyStackMembers(store, "StackProgramSchema")); got != 2 {
		t.Fatalf("ambiguous program schemas = %d, want 2", got)
	}
	conjectures := 0
	for _, candidate := range tinyStackMembers(store, "ProtoConjec") {
		if candidate.GetString("conjecKind") == "BoundedProgramSatisfiesCorpus" && candidate.GetString("synthesisExperiment") == "StackSynthesisExperiment" {
			conjectures++
		}
	}
	if conjectures != 2 {
		t.Fatalf("ambiguous selection conjectures = %d, want 2", conjectures)
	}
}

func TestTinyStackUndefinedOnlyAgreementIsNotPromoted(t *testing.T) {
	store := loadTinyStack(t)
	store.Get("StackProbeEmpty").Set("data", []int{})
	store.Get("StackProbeSingle").Set("data", []int{1})
	store.Get("StackProbePair").Set("data", []int{2})
	for _, name := range []string{"StackProbeTripleA", "StackProbePairZero", "StackProbeTripleB"} {
		store.Delete(name)
	}
	store, _ = runTinyStack(t, store)
	long := stackProgramBySemantics(store, "swap", "add")
	short := primitiveBySemantic(store, "add")
	if long == nil || short == nil {
		t.Fatal("vacuity-control programs missing")
	}
	var evidence *unit.Unit
	for _, candidate := range tinyStackMembers(store, "StackSimplificationEvidence") {
		if candidate.GetString("longProgram") == long.Name && candidate.GetString("shortProgram") == short.Name {
			evidence = candidate
			break
		}
	}
	if evidence == nil || evidence.GetInt("bothUndefinedCount") != 3 || evidence.GetInt("bothDefinedCount") != 0 || evidence.GetBool("equivalent") {
		t.Fatalf("vacuity evidence = %#v", evidence)
	}
	for _, schema := range tinyStackMembers(store, "StackSimplificationSchema") {
		if schema.GetString("longProgram") == long.Name && schema.GetString("shortProgram") == short.Name {
			t.Fatalf("undefined-only pair was promoted as %s", schema.Name)
		}
	}
}

type stackCase struct {
	input, expected []int
}

func setStackExamples(store *unit.Store, cases []stackCase) {
	setCategoryStackExamples(store, "StackProgramExample", cases)
}

func setCategoryStackExamples(store *unit.Store, category string, cases []stackCase) {
	for index, example := range tinyStackMembers(store, category) {
		example.Set("input", append([]int(nil), cases[index].input...))
		example.Set("expected", append([]int(nil), cases[index].expected...))
	}
}

func renameTinyStackUnit(store *unit.Store, oldName, newName string) *unit.Unit {
	old := store.Get(oldName)
	replacement := unit.New(newName)
	for slot, value := range old.Slots {
		replacement.Set(slot, value)
	}
	store.Put(replacement)
	store.Delete(oldName)
	return replacement
}

func renameTinyStackCategory(store *unit.Store, descriptor *unit.Unit, field, newName string) {
	oldName := descriptor.GetString(field)
	renameTinyStackUnit(store, oldName, newName)
	for _, name := range store.All() {
		u := store.Get(name)
		for _, slot := range []string{"isA", "domain", "range"} {
			values := u.GetStrings(slot)
			changed := false
			for index, value := range values {
				if value == oldName {
					values[index] = newName
					changed = true
				}
			}
			if changed {
				u.Set(slot, values)
			}
		}
	}
	descriptor.Set(field, newName)
}

func stackDSLInts(values []int) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, " ") + fmt.Sprintf(" %d list-of", len(values))
}

func assertTinyStackNoSolution(t *testing.T, store *unit.Store) {
	t.Helper()
	store, _ = runTinyStackLanes(t, store, false)
	experiment := store.Get("StackSynthesisExperiment")
	if !experiment.GetBool("finalizationComplete") || experiment.GetString("selectionStatus") != "no-solution" || len(experiment.GetStrings("selectedPrograms")) != 0 || len(tinyStackMembers(store, "StackProgramSchema")) != 0 {
		t.Fatalf("no-solution control = complete %v status %q selected %v schemas %d", experiment.GetBool("finalizationComplete"), experiment.GetString("selectionStatus"), experiment.GetStrings("selectedPrograms"), len(tinyStackMembers(store, "StackProgramSchema")))
	}
}

func dslNameList(names []string) string {
	quoted := make([]string, len(names))
	for index, name := range names {
		quoted[index] = strconv.Quote(name)
	}
	return strings.Join(quoted, " ") + fmt.Sprintf(" %d list-of", len(names))
}

func synthArtifactBase(t *testing.T, eng *engine.Engine, experiment, kind, program, example string) string {
	t.Helper()
	value, err := eng.VM.Execute(strconv.Quote(experiment) + " " + strconv.Quote(kind) + " " + strconv.Quote(program) + " " + strconv.Quote(example) + " synth-artifact-name")
	if err != nil || value.AsString() == "" {
		t.Fatalf("artifact base %s = (%v,%v)", kind, value, err)
	}
	return value.AsString()
}

func dslInts(value dsl.Value) []int {
	if value.Kind() != dsl.VList {
		return nil
	}
	result := make([]int, len(value.AsList()))
	for index, item := range value.AsList() {
		if item.Kind() != dsl.VInt {
			return nil
		}
		result[index] = item.AsInt()
	}
	return result
}

func canonicalTinyStack(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	data, err := store.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func discoveryTinyStackSnapshot(store *unit.Store) string {
	return discoveryTinyStackSnapshotFor(store, store.Get("StackSynthesisExperiment"))
}

func discoveryTinyStackSnapshotFor(store *unit.Store, experiment *unit.Unit) string {
	categories := []string{
		experiment.GetString("candidateCategory"), experiment.GetString("resultCategory"), experiment.GetString("observationCategory"), experiment.GetString("evidenceCategory"),
		experiment.GetString("selectionEvidenceCategory"), experiment.GetString("promotedSchemaCategory"), experiment.GetString("simplificationPairCategory"),
		experiment.GetString("simplificationExecutionObservationCategory"), experiment.GetString("simplificationExecutionResultCategory"),
		experiment.GetString("simplificationComparisonObservationCategory"), experiment.GetString("simplificationEvidenceCategory"), experiment.GetString("simplificationSchemaCategory"),
	}
	var parts []string
	for _, category := range categories {
		var names []string
		for _, member := range tinyStackMembers(store, category) {
			names = append(names, member.Name)
		}
		parts = append(parts, category+":"+strings.Join(names, ","))
	}
	parts = append(parts, fmt.Sprintf("experiment:%v:%v:%s:%d:%d:%d", experiment.GetStrings("exactCandidates"), experiment.GetStrings("selectedPrograms"), experiment.GetString("selectionStatus"), experiment.GetInt("evaluatedCandidateCount"), experiment.GetInt("simplificationPairCount"), experiment.GetInt("simplificationEquivalentPairCount")))
	for _, candidate := range tinyStackMembers(store, experiment.GetString("candidateCategory")) {
		parts = append(parts, fmt.Sprintf("candidate:%s:%s:%d:%d:%d:%d:%v", candidate.Name, candidate.GetString("evidenceUnit"), candidate.GetInt("supportCount"), candidate.GetInt("failureCount"), candidate.GetInt("invalidCount"), candidate.Worth(), candidate.GetBool("evaluatedProgramCorpus")))
	}
	return strings.Join(parts, "\n")
}

func tinyStackCreditSnapshot(store *unit.Store, winner *unit.Unit) string {
	parts := []string{
		fmt.Sprintf("winner:%d:%d", winner.Worth(), winner.GetInt("lastRewardedWorth")),
		fmt.Sprintf("heuristic:%d", store.Get("H-EnumerateBoundedPrograms").Worth()),
	}
	for _, primitive := range tinyStackMembers(store, "StackInstruction") {
		parts = append(parts, fmt.Sprintf("primitive:%s:%d", primitive.Name, primitive.Worth()))
	}
	for _, record := range tinyStackMembers(store, "ContextualCredit") {
		parts = append(parts, fmt.Sprintf("credit:%s:%s:%s:%d:%d:%s:%d", record.GetString("creditContext"), record.GetString("creditSubject"), record.GetString("creditRole"), record.GetInt("rewardTotal"), record.GetInt("evidenceCount"), record.GetString("lastSourceUnit"), record.GetInt("lastRewardTaskNum")))
	}
	return strings.Join(parts, "\n")
}

func exhaustiveSmallStacks() [][]int {
	values := []int{-2, -1, 0, 1, 2}
	result := [][]int{{}}
	for depth := 1; depth <= 4; depth++ {
		var build func([]int)
		build = func(prefix []int) {
			if len(prefix) == depth {
				result = append(result, append([]int(nil), prefix...))
				return
			}
			for _, value := range values {
				build(append(prefix, value))
			}
		}
		build(nil)
	}
	return result
}
