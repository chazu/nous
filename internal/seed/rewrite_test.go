package seed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	rewritevocab "github.com/chazu/nous/internal/vocab/rewrite"
)

const rewriteCycles = 220

type rewriteRuleSpec struct {
	name        string
	left, right string
}

type rewriteExampleSpec struct {
	name            string
	input, expected string
}

func loadRewrite(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "rewrite"); err != nil {
		t.Fatal(err)
	}
	return store
}

func runRewriteStore(t *testing.T, store *unit.Store, mutate bool) (*unit.Store, *engine.Engine) {
	t.Helper()
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = rewriteCycles
	eng.MutConfig.Enabled = mutate
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store, eng
}

func TestRewriteVocabularySynthesizesAndCreditsUniqueProgram(t *testing.T) {
	store, eng := runRewriteStore(t, loadRewrite(t), false)
	for _, foreign := range []string{"MathConcept", "Protocol", "BuildGraph"} {
		if store.Has(foreign) {
			t.Fatalf("rewrite vocabulary loaded foreign unit %s", foreign)
		}
	}
	if err := eng.VM.InitError(); err != nil {
		t.Fatal(err)
	}
	target := assertRewriteExperiment(t, store, 4)
	components := target.GetStrings("components")
	if len(components) != 2 || store.Get(components[0]).GetString("rewriteLeft") != "ab" || store.Get(components[1]).GetString("rewriteLeft") != "xc" {
		t.Fatalf("promoted components = %v", components)
	}
	if target.GetInt("creationWorth") != 500 || target.GetInt("lastRewardedWorth") != 800 || target.Worth() != 800 {
		t.Fatalf("target reward state = creation %d last %d worth %d", target.GetInt("creationWorth"), target.GetInt("lastRewardedWorth"), target.Worth())
	}
	if store.Get("H-ComposeRewritePrograms").Worth() != 900 {
		t.Fatalf("synthesis heuristic worth = %d, want 900", store.Get("H-ComposeRewritePrograms").Worth())
	}
	for _, primitive := range rewritePrimitives(store) {
		want := 600
		if primitive == components[0] || primitive == components[1] {
			want = 750
		}
		if got := store.Get(primitive).Worth(); got != want {
			t.Fatalf("%s worth = %d, want %d", primitive, got, want)
		}
	}
	decision := rewritevocab.DecisionKey(components[0], components[1])
	if target.GetString("creditContext") != rewritevocab.CreditContext || target.GetString("creditDecision") != decision {
		t.Fatalf("contextual declaration = context %q decision %q", target.GetString("creditContext"), target.GetString("creditDecision"))
	}
	if got := target.GetStrings("creditRoles"); !equalStringSlices(got, []string{"synthesis", "first", "second"}) {
		t.Fatalf("credit roles = %v", got)
	}
	assertContextualCredit(t, store, credit.DecisionTuple(rewritevocab.CreditContext, decision), 300)
	assertContextualCredit(t, store, credit.Tuple{Context: rewritevocab.CreditContext, Subject: "H-ComposeRewritePrograms", Role: "synthesis"}, 150)
	assertContextualCredit(t, store, credit.Tuple{Context: rewritevocab.CreditContext, Subject: components[0], Role: "first"}, 150)
	assertContextualCredit(t, store, credit.Tuple{Context: rewritevocab.CreditContext, Subject: components[1], Role: "second"}, 150)
	assertRewriteHeldOut(t, store, target, map[string]string{
		"qabcw":     "qyw",
		"cabcd":     "cyd",
		"abcabcc":   "yyc",
		"zzabcabcq": "zzyyq",
	})

	result := firstRewriteResult(store, target.Name)
	if result == nil {
		t.Fatal("target has no result unit")
	}
	want, ok := referenceProgram(result.GetString("data"), target.GetStrings("components"), store)
	if !ok {
		t.Fatal("reference reapplication failed")
	}
	value, err := eng.VM.Execute(fmt.Sprintf("%q %q apply-op", result.GetString("data"), target.Name))
	if err != nil || value.AsString() != want {
		t.Fatalf("result reapplication = (%q,%v), want %q", value.AsString(), err, want)
	}

	before := rewriteExperimentSnapshot(t, store)
	for _, primitive := range rewritePrimitives(store) {
		eng.WorkOnUnit(primitive)
	}
	eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: target.Name, SlotName: "rewriteEvaluation"})
	if after := rewriteExperimentSnapshot(t, store); string(before) != string(after) {
		t.Fatal("repeated focus/task changed guarded rewrite evidence")
	}
}

func TestRewriteDefinitionsMatchStructuredRulesOnGeneratedProbes(t *testing.T) {
	store := loadRewrite(t)
	vm := dsl.NewVM(store, agenda.New(), nil)
	vm.Out = io.Discard
	probes := rewriteProbes([]byte("abcxyz"), 3)
	for _, primitive := range rewritePrimitives(store) {
		op := store.Get(primitive)
		for _, probe := range probes {
			want, ok := referenceReplace(probe, op.GetString("rewriteLeft"), op.GetString("rewriteRight"))
			if !ok {
				t.Fatalf("structured primitive %s rejected probe %q", primitive, probe)
			}
			got, err := vm.Execute(fmt.Sprintf("%q %q apply-op", probe, primitive))
			if err != nil || got.AsString() != want {
				t.Fatalf("%s(%q) = (%q,%v), want %q", primitive, probe, got.AsString(), err, want)
			}
		}
	}

	store, eng := runRewriteStore(t, store, false)
	for _, program := range rewriteUnits(store, "CompositeRewriteOp", "components") {
		for _, probe := range probes {
			want, ok := referenceProgram(probe, store.Get(program).GetStrings("components"), store)
			if !ok {
				t.Fatalf("reference composite %s rejected probe %q", program, probe)
			}
			got, err := eng.VM.Execute(fmt.Sprintf("%q %q apply-op", probe, program))
			if err != nil || got.AsString() != want {
				t.Fatalf("%s(%q) = (%q,%v), want %q", program, probe, got.AsString(), err, want)
			}
		}
	}
}

func TestRewriteOpaqueAliasesCollisionsAndPrimitiveDeletion(t *testing.T) {
	store := loadRewrite(t)
	replaceRewriteTrial(t, store,
		[]rewriteRuleSpec{
			{"Alias-then-Z", "xc", "y"},
			{"Alias_collision_1", "bc", "z"},
			{"Alias.dot", "ab", "x"},
			{"Alias:three", "x", "q"},
		},
		[]rewriteExampleSpec{
			{"OpaqueExampleD", "abcabc", "yy"},
			{"OpaqueExampleB", "zabc", "zy"},
			{"OpaqueExampleA", "abc", "y"},
			{"OpaqueExampleC", "abcc", "yc"},
		})

	targetBase := composeBase("Alias.dot", "Alias-then-Z")
	collision := unit.New(targetBase)
	collision.Set("sentinel", "preserve")
	store.Put(collision)
	decoyProgram := composeBase("Alias.dot", "Alias_collision_1")
	resultCollisionName := artifactBase("Result", decoyProgram, "OpaqueExampleA")
	resultCollision := unit.New(resultCollisionName)
	resultCollision.Set("sentinel", "preserve")
	store.Put(resultCollision)

	store, eng := runRewriteStore(t, store, false)
	target := assertRewriteExperiment(t, store, 4)
	if !strings.Contains(target.Name, "collision-1") {
		t.Fatalf("occupied target name did not receive deterministic suffix: %s", target.Name)
	}
	if got, want := target.GetString("creditDecision"), rewritevocab.DecisionKey("Alias.dot", "Alias-then-Z"); got != want {
		t.Fatalf("collision changed semantic credit key: got %q want %q", got, want)
	}
	for _, occupied := range []string{targetBase, resultCollisionName} {
		if got := store.Get(occupied).GetString("sentinel"); got != "preserve" {
			t.Fatalf("occupied unit %s was overwritten", occupied)
		}
	}
	assertRewriteHeldOut(t, store, target, map[string]string{"qabcw": "qyw", "abcabcabc": "yyy"})

	for _, primitive := range append([]string(nil), rewritePrimitives(store)...) {
		store.Delete(primitive)
	}
	value, err := eng.VM.Execute(fmt.Sprintf("%q %q apply-op", "qabcw", target.Name))
	if err != nil || value.AsString() != "qyw" {
		t.Fatalf("inlined composite after primitive deletion = (%q,%v)", value.AsString(), err)
	}
}

func assertContextualCredit(t *testing.T, store *unit.Store, tuple credit.Tuple, want int) {
	t.Helper()
	record := credit.Lookup(store, tuple)
	if record == nil || record.GetInt("rewardTotal") != want || record.GetInt("evidenceCount") != 1 {
		t.Fatalf("contextual credit %v = %#v, want %d", tuple, record, want)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestRewriteAlternateRuntimeCorpusDefeatsSeedHardcoding(t *testing.T) {
	store := loadRewrite(t)
	replaceRewriteTrial(t, store,
		[]rewriteRuleSpec{
			{"Alt0", "nq", "s"},
			{"Alt1", "pq", "r"},
			{"Alt2", "p", "t"},
			{"Alt3", "mn", "p"},
		},
		[]rewriteExampleSpec{
			{"AltExample3", "mnqq", "rq"},
			{"AltExample1", "mnq", "r"},
			{"AltExample4", "mnqmnq", "rr"},
			{"AltExample2", "zmnq", "zr"},
		})
	store, _ = runRewriteStore(t, store, false)
	target := assertRewriteExperiment(t, store, 4)
	if got := target.GetStrings("components"); len(got) != 2 || got[0] != "Alt3" || got[1] != "Alt1" {
		t.Fatalf("alternate trial promoted %v, want [Alt3 Alt1]", got)
	}
	assertRewriteHeldOut(t, store, target, map[string]string{"amnqb": "arb", "mnqmnqq": "rrq"})
}

func TestRewriteInvalidCorpusAndRulesCannotPromote(t *testing.T) {
	tests := map[string]func(*unit.Store){
		"input": func(store *unit.Store) {
			store.Get("RewriteExampleOne").Set("input", "INVALID")
		},
		"expected": func(store *unit.Store) {
			store.Get("RewriteExampleOne").Set("expected", "INVALID")
		},
		"rule": func(store *unit.Store) {
			op := store.Get("RewriteOpKappa")
			op.Set("rewriteLeft", "")
			op.Set("defn", `"" "x" rewrite-replace-all`)
		},
		"overflow": func(store *unit.Store) {
			op := store.Get("RewriteOpKappa")
			op.Set("rewriteLeft", "a")
			op.Set("rewriteRight", "aaaaaaaa")
			op.Set("defn", `"a" "aaaaaaaa" rewrite-replace-all`)
			example := store.Get("RewriteExampleOne")
			example.Set("input", strings.Repeat("a", 64))
			example.Set("expected", "a")
		},
	}
	for name, alter := range tests {
		t.Run(name, func(t *testing.T) {
			store := loadRewrite(t)
			alter(store)
			store, _ = runRewriteStore(t, store, false)
			if got := rewriteUnits(store, "RewriteProgramSchema", "program"); len(got) != 0 {
				t.Fatalf("invalid trial promoted schemas %v", got)
			}
			foundDiagnostic := false
			for _, observation := range rewriteUnits(store, "RewriteObservation", "program") {
				status := store.Get(observation).GetString("status")
				if strings.HasPrefix(status, "invalid-") || status == "semantic-nil" {
					foundDiagnostic = true
					if store.Get(observation).GetBool("outcome") {
						t.Fatalf("diagnostic observation %s counted as support", observation)
					}
				}
			}
			if !foundDiagnostic {
				t.Fatal("invalid trial produced no diagnostic observation")
			}
		})
	}
}

func TestRewriteStoreSnapshotsAreDeterministicAndMutationInactive(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprintf("mutation=%v", mutate), func(t *testing.T) {
			first, _ := runRewriteStore(t, loadRewrite(t), mutate)
			second, _ := runRewriteStore(t, loadRewrite(t), mutate)
			firstJSON, err := first.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			secondJSON, err := second.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(firstJSON) != string(secondJSON) {
				t.Fatal("rewrite store snapshots differ")
			}
			if mutate {
				for _, name := range first.All() {
					if first.Get(name).GetString("mutant_of") != "" {
						t.Fatalf("mutation-enabled inactive control created %s", name)
					}
				}
			}
		})
	}
}

func assertRewriteExperiment(t *testing.T, store *unit.Store, corpusSize int) *unit.Unit {
	t.Helper()
	primitives := rewritePrimitives(store)
	programs := rewriteUnits(store, "CompositeRewriteOp", "components")
	evidence := rewriteUnits(store, "RewriteProgramEvidence", "program")
	results := rewriteUnits(store, "RewriteStringResult", "program")
	observations := rewriteUnits(store, "RewriteObservation", "program")
	schemas := rewriteUnits(store, "RewriteProgramSchema", "program")
	if len(primitives) != 4 || len(programs) != 12 || len(evidence) != 12 || len(results) != 12*corpusSize || len(observations) != 12*corpusSize {
		t.Fatalf("rewrite matrix = %d primitives, %d programs, %d evidence, %d results, %d observations", len(primitives), len(programs), len(evidence), len(results), len(observations))
	}
	if len(schemas) != 1 {
		t.Fatalf("promoted schemas = %v, want exactly one", schemas)
	}
	conjectures := 0
	for _, name := range store.All() {
		if strings.HasPrefix(name, "Conjec-RewriteProgramSatisfiesCorpus-") {
			conjectures++
		}
	}
	if conjectures != 1 {
		t.Fatalf("rewrite conjectures = %d, want 1", conjectures)
	}

	for _, programName := range programs {
		program := store.Get(programName)
		components := program.GetStrings("components")
		if len(components) != 2 || components[0] == components[1] {
			t.Fatalf("%s components = %v", programName, components)
		}
		wantSupport, wantFailures := 0, 0
		for _, exampleName := range rewriteExamples(store) {
			example := store.Get(exampleName)
			actual, ok := referenceProgram(example.GetString("input"), components, store)
			if !ok {
				t.Fatalf("reference rejected valid pair %s", programName)
			}
			outcome := actual == example.GetString("expected")
			if outcome {
				wantSupport++
			} else {
				wantFailures++
			}
			observation := findRewriteUnit(store, "RewriteObservation", programName, exampleName)
			result := findRewriteUnit(store, "RewriteStringResult", programName, exampleName)
			if observation == nil || result == nil {
				t.Fatalf("missing linked artifacts for %s/%s", programName, exampleName)
			}
			wantStatus := "mismatch"
			if outcome {
				wantStatus = "match"
			}
			if observation.GetString("input") != example.GetString("input") || observation.GetString("expected") != example.GetString("expected") || observation.GetString("actual") != actual || observation.GetBool("outcome") != outcome || observation.GetString("status") != wantStatus || observation.GetString("resultUnit") != result.Name {
				t.Fatalf("observation %s disagrees with independent oracle", observation.Name)
			}
			if result.GetString("data") != actual || result.GetString("program") != programName || result.GetString("example") != exampleName {
				t.Fatalf("result %s has inconsistent provenance", result.Name)
			}
		}
		if program.GetInt("supportCount") != wantSupport || program.GetInt("failureCount") != wantFailures || program.GetInt("invalidCount") != 0 || program.GetInt("evaluatedCount") != corpusSize || program.GetInt("corpusSize") != corpusSize {
			t.Fatalf("%s counts disagree with oracle", programName)
		}
		wantWorth := 300
		if wantSupport == corpusSize && wantFailures == 0 {
			wantWorth = 800
		}
		if program.Worth() != wantWorth {
			t.Fatalf("%s worth = %d, want %d", programName, program.Worth(), wantWorth)
		}
		applications, ok := program.Get("applics").([]map[string]any)
		if !ok || len(applications) != corpusSize {
			t.Fatalf("%s applications = %d, want %d", programName, len(applications), corpusSize)
		}
		seenApplications := map[string]bool{}
		for _, application := range applications {
			args, _ := application["args"].([]string)
			output, _ := application["output"].(string)
			if len(args) != 1 || store.Get(output) == nil || store.Get(output).GetString("program") != programName || store.Get(output).GetString("example") != args[0] {
				t.Fatalf("%s has invalid application record %#v", programName, application)
			}
			if seenApplications[args[0]] {
				t.Fatalf("%s has duplicate application for %s", programName, args[0])
			}
			seenApplications[args[0]] = true
		}
		evidenceUnit := store.Get(program.GetString("evidenceUnit"))
		if evidenceUnit == nil || len(evidenceUnit.GetStrings("trainingExamples")) != corpusSize || len(evidenceUnit.GetStrings("resultUnits")) != corpusSize || len(evidenceUnit.GetStrings("observations")) != corpusSize {
			t.Fatalf("%s evidence is incomplete", programName)
		}
		if evidenceUnit.GetInt("supportCount") != wantSupport || evidenceUnit.GetInt("failureCount") != wantFailures || evidenceUnit.GetInt("invalidCount") != 0 || evidenceUnit.GetInt("corpusSize") != corpusSize {
			t.Fatalf("%s evidence counts disagree with oracle", programName)
		}
		for _, creditor := range []string{"H-ComposeRewritePrograms", components[0], components[1]} {
			if !containsString(program.GetStrings("creditors"), creditor) {
				t.Fatalf("%s missing creditor %s", programName, creditor)
			}
		}
	}
	target := store.Get(store.Get(schemas[0]).GetString("program"))
	if target == nil {
		t.Fatal("schema does not resolve to a program")
	}
	schema := store.Get(schemas[0])
	if schema.GetString("evidenceUnit") != target.GetString("evidenceUnit") || schema.GetInt("creationWorth") != 800 || schema.GetInt("lastRewardedWorth") != 800 {
		t.Fatalf("schema %s has inconsistent evidence or reward baseline", schema.Name)
	}
	for _, name := range store.All() {
		if strings.HasPrefix(name, "Conjec-RewriteProgramSatisfiesCorpus-") && !containsString(store.Get(name).GetStrings("evidence"), target.GetString("evidenceUnit")) {
			t.Fatalf("conjecture %s does not cite target evidence", name)
		}
	}
	return target
}

func referenceProgram(input string, components []string, store *unit.Store) (string, bool) {
	result := input
	for _, component := range components {
		op := store.Get(component)
		if op == nil {
			return "", false
		}
		var ok bool
		result, ok = referenceReplace(result, op.GetString("rewriteLeft"), op.GetString("rewriteRight"))
		if !ok {
			return "", false
		}
	}
	return result, true
}

func referenceReplace(input, left, right string) (string, bool) {
	valid := func(text string) bool {
		if len(text) > 256 {
			return false
		}
		for i := range len(text) {
			if text[i] < 'a' || text[i] > 'z' {
				return false
			}
		}
		return true
	}
	if !valid(input) || left == "" || len(left) > 8 || len(right) > 8 || !valid(left) || !valid(right) {
		return "", false
	}
	var out strings.Builder
	for position := 0; position < len(input); {
		if strings.HasPrefix(input[position:], left) {
			out.WriteString(right)
			position += len(left)
		} else {
			out.WriteByte(input[position])
			position++
		}
		if out.Len() > 256 {
			return "", false
		}
	}
	return out.String(), true
}

func rewritePrimitives(store *unit.Store) []string {
	return rewriteUnits(store, "PrimitiveRewriteOp", "rewriteLeft")
}

func rewriteExamples(store *unit.Store) []string {
	return rewriteUnits(store, "RewriteTrainingExample", "input")
}

func rewriteUnits(store *unit.Store, category, requiredSlot string) []string {
	var names []string
	for _, name := range store.Examples(category) {
		if name != category && store.Get(name).Has(requiredSlot) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func findRewriteUnit(store *unit.Store, category, program, example string) *unit.Unit {
	for _, name := range rewriteUnits(store, category, "program") {
		u := store.Get(name)
		if u.GetString("program") == program && u.GetString("example") == example {
			return u
		}
	}
	return nil
}

func firstRewriteResult(store *unit.Store, program string) *unit.Unit {
	for _, name := range rewriteUnits(store, "RewriteStringResult", "program") {
		if store.Get(name).GetString("program") == program {
			return store.Get(name)
		}
	}
	return nil
}

func assertRewriteHeldOut(t *testing.T, store *unit.Store, program *unit.Unit, cases map[string]string) {
	t.Helper()
	before := store.Count()
	vm := dsl.NewVM(store, agenda.New(), nil)
	vm.Out = io.Discard
	for input, expected := range cases {
		actual, ok := referenceProgram(input, program.GetStrings("components"), store)
		if !ok || actual != expected {
			t.Fatalf("independent held-out oracle for %q = (%q,%v), want %q", input, actual, ok, expected)
		}
		value, err := vm.Execute(fmt.Sprintf("%q %q apply-op", input, program.Name))
		if err != nil || value.AsString() != expected {
			t.Fatalf("promoted program on %q = (%q,%v), want %q", input, value.AsString(), err, expected)
		}
	}
	if store.Count() != before {
		t.Fatal("held-out corpus entered the discovery store")
	}
}

func replaceRewriteTrial(t *testing.T, store *unit.Store, rules []rewriteRuleSpec, examples []rewriteExampleSpec) {
	t.Helper()
	for _, name := range append(rewritePrimitives(store), rewriteExamples(store)...) {
		store.Delete(name)
	}
	for _, spec := range rules {
		u := unit.New(spec.name)
		u.SetWorth(600)
		u.Set("isA", []string{"PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"})
		u.Set("domain", []string{"RewriteString"})
		u.Set("range", []string{"RewriteString"})
		u.Set("arity", 1)
		u.Set("rewriteLeft", spec.left)
		u.Set("rewriteRight", spec.right)
		u.Set("defn", fmt.Sprintf("%q %q rewrite-replace-all", spec.left, spec.right))
		store.Put(u)
	}
	for _, spec := range examples {
		u := unit.New(spec.name)
		u.SetWorth(600)
		u.Set("isA", []string{"RewriteTrainingExample", "Anything"})
		u.Set("input", spec.input)
		u.Set("expected", spec.expected)
		store.Put(u)
	}
}

func composeBase(first, second string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return "Compose." + encode([]byte(first)) + "." + encode([]byte(second))
}

func artifactBase(kind, program, example string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return "RewriteArtifact." + encode([]byte(kind)) + "." + encode([]byte(program)) + "." + encode([]byte(example))
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func rewriteProbes(alphabet []byte, maxLength int) []string {
	probes := []string{""}
	frontier := []string{""}
	for depth := 0; depth < maxLength; depth++ {
		var next []string
		for _, prefix := range frontier {
			for _, symbol := range alphabet {
				value := prefix + string(symbol)
				probes = append(probes, value)
				next = append(next, value)
			}
		}
		frontier = next
	}
	return probes
}

func rewriteExperimentSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	snapshot := map[string]map[string]any{}
	for _, name := range store.All() {
		if !(store.IsA(name, "PrimitiveRewriteOp") ||
			store.IsA(name, "CompositeRewriteOp") ||
			store.IsA(name, "RewriteProgramEvidence") ||
			store.IsA(name, "RewriteStringResult") ||
			store.IsA(name, "RewriteObservation") ||
			store.IsA(name, "RewriteProgramSchema") ||
			strings.HasPrefix(name, "Conjec-RewriteProgramSatisfiesCorpus-")) {
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
