package seed

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/unit"
)

// The trial oracle deliberately does not call internal/vocab/tinystack or any
// DSL word. It is a second, small interpreter used to falsify the production
// generator, evaluator, artifact accounting, and simplification comparison.
func oracleStackProgram(input []int, program []string) ([]int, bool) {
	if len(program) < 1 || len(program) > 3 || len(input) > 4 {
		return nil, false
	}
	stack := append([]int(nil), input...)
	for _, value := range stack {
		if value < -100 || value > 100 {
			return nil, false
		}
	}
	for _, opcode := range program {
		switch opcode {
		case "dup":
			if len(stack) < 1 || len(stack) == 7 {
				return nil, false
			}
			stack = append(stack, stack[len(stack)-1])
		case "swap":
			if len(stack) < 2 {
				return nil, false
			}
			stack[len(stack)-1], stack[len(stack)-2] = stack[len(stack)-2], stack[len(stack)-1]
		case "drop":
			if len(stack) < 1 {
				return nil, false
			}
			stack = stack[:len(stack)-1]
		case "over":
			if len(stack) < 2 || len(stack) == 7 {
				return nil, false
			}
			stack = append(stack, stack[len(stack)-2])
		case "add", "mul":
			if len(stack) < 2 {
				return nil, false
			}
			a, b := int64(stack[len(stack)-2]), int64(stack[len(stack)-1])
			value := a + b
			if opcode == "mul" {
				value = a * b
			}
			if value < -100_000_000 || value > 100_000_000 {
				return nil, false
			}
			stack = append(stack[:len(stack)-2], int(value))
		case "double", "neg":
			if len(stack) < 1 {
				return nil, false
			}
			value := int64(stack[len(stack)-1])
			if opcode == "double" {
				value *= 2
			} else {
				value = -value
			}
			if value < -100_000_000 || value > 100_000_000 {
				return nil, false
			}
			stack[len(stack)-1] = int(value)
		default:
			return nil, false
		}
	}
	return stack, true
}

func oracleSequences(opcodes []string, maxLength int) [][]string {
	var sequences [][]string
	var appendLength func([]string, int)
	appendLength = func(prefix []string, remaining int) {
		if remaining == 0 {
			sequences = append(sequences, append([]string(nil), prefix...))
			return
		}
		for _, opcode := range opcodes {
			appendLength(append(prefix, opcode), remaining-1)
		}
	}
	for length := 1; length <= maxLength; length++ {
		appendLength(nil, length)
	}
	return sequences
}

func semanticKey(sequence []string) string { return strings.Join(sequence, "\x00") }

func intsSlot(u *unit.Unit, slot string) ([]int, bool) {
	if u == nil {
		return nil, false
	}
	switch values := u.Get(slot).(type) {
	case []int:
		return values, true
	case []string:
		return []int{}, len(values) == 0
	case []any:
		return []int{}, len(values) == 0
	default:
		return nil, false
	}
}

func indexByPair(store *unit.Store, category, leftSlot, rightSlot string) map[string]*unit.Unit {
	result := map[string]*unit.Unit{}
	for _, member := range tinyStackMembers(store, category) {
		key := member.GetString(leftSlot) + "\x00" + member.GetString(rightSlot)
		result[key] = member
	}
	return result
}

func hasDirectOutput(u *unit.Unit, output string) bool {
	applics, _ := u.Get("applics").([]map[string]any)
	for _, applic := range applics {
		if applic["direct"] == true && applic["output"] == output {
			return true
		}
	}
	return false
}

func assertTinyStackIndependentMatrix(t *testing.T, store *unit.Store, experimentName string, opcodes []string) {
	t.Helper()
	experiment := store.Get(experimentName)
	candidateCategory := experiment.GetString("candidateCategory")
	exampleCategory := experiment.GetString("exampleCategory")
	resultCategory := experiment.GetString("resultCategory")
	observationCategory := experiment.GetString("observationCategory")
	evidenceCategory := experiment.GetString("evidenceCategory")
	probeCategory := experiment.GetString("probeCategory")

	candidates := map[string]*unit.Unit{}
	for _, candidate := range tinyStackMembers(store, candidateCategory) {
		key := semanticKey(candidate.GetStrings("semanticSequence"))
		if candidates[key] != nil {
			t.Fatalf("duplicate semantic candidate %v", candidate.GetStrings("semanticSequence"))
		}
		candidates[key] = candidate
	}
	sequences := oracleSequences(opcodes, 3)
	if len(sequences) != 399 || len(candidates) != len(sequences) {
		t.Fatalf("oracle/candidate cardinality = %d/%d", len(sequences), len(candidates))
	}
	examples := tinyStackMembers(store, exampleCategory)
	observations := indexByPair(store, observationCategory, "program", "example")
	actualResults, undefined := 0, 0
	for _, sequence := range sequences {
		candidate := candidates[semanticKey(sequence)]
		if candidate == nil || candidate.GetInt("programLength") != len(sequence) {
			t.Fatalf("missing/malformed candidate %v", sequence)
		}
		support, failures, invalid := 0, 0, 0
		for _, example := range examples {
			input, inputOK := intsSlot(example, experiment.GetString("inputSlot"))
			expected, expectedOK := intsSlot(example, experiment.GetString("expectedSlot"))
			if !inputOK || !expectedOK {
				t.Fatalf("oracle example %s malformed", example.Name)
			}
			actual, defined := oracleStackProgram(input, sequence)
			observation := observations[candidate.Name+"\x00"+example.Name]
			if observation == nil {
				t.Fatalf("missing observation for %v/%s", sequence, example.Name)
			}
			if defined {
				actualResults++
				matches := reflect.DeepEqual(actual, expected)
				if matches {
					support++
				} else {
					failures++
				}
				observed, ok := intsSlot(observation, "actual")
				if !ok || !reflect.DeepEqual(observed, actual) || observation.GetBool("outcome") != matches {
					t.Fatalf("observation mismatch for %v/%s: %#v", sequence, example.Name, observation.Slots)
				}
				result := store.Get(observation.GetString("resultUnit"))
				resultData, resultOK := intsSlot(result, experiment.GetString("resultValueSlot"))
				if result == nil || !store.IsA(result.Name, resultCategory) || result.GetString("program") != candidate.Name || result.GetString("example") != example.Name || !resultOK || !reflect.DeepEqual(resultData, actual) || !hasDirectOutput(candidate, result.Name) {
					t.Fatalf("result/application mismatch for %v/%s", sequence, example.Name)
				}
			} else {
				invalid++
				undefined++
				if observation.GetString("resultUnit") != "" || observation.Get("actual") != nil || observation.GetString("status") != "semantic-nil" {
					t.Fatalf("undefined observation mismatch for %v/%s: %#v", sequence, example.Name, observation.Slots)
				}
			}
		}
		exact := support == len(examples) && failures == 0 && invalid == 0
		if candidate.GetInt("supportCount") != support || candidate.GetInt("failureCount") != failures || candidate.GetInt("invalidCount") != invalid || candidate.GetBool("exactProgram") != exact {
			t.Fatalf("candidate evidence mismatch for %v: %#v, oracle %d/%d/%d", sequence, candidate.Slots, support, failures, invalid)
		}
		evidence := store.Get(candidate.GetString("evidenceUnit"))
		if evidence == nil || !store.IsA(evidence.Name, evidenceCategory) || evidence.GetString("program") != candidate.Name || len(evidence.GetStrings("observations")) != len(examples) || len(evidence.GetStrings("resultUnits")) != support+failures || evidence.GetInt("supportCount") != support || evidence.GetInt("failureCount") != failures || evidence.GetInt("invalidCount") != invalid {
			t.Fatalf("aggregate evidence mismatch for %v", sequence)
		}
		wantWorth := 300
		if exact {
			wantWorth = 500
			if len(sequence) == experiment.GetInt("minimumLength") {
				wantWorth = 800
			}
		}
		if candidate.Worth() != wantWorth {
			t.Fatalf("candidate %v worth = %d, want %d", sequence, candidate.Worth(), wantWorth)
		}
		if exact && len(sequence) > experiment.GetInt("minimumLength") && !reflect.DeepEqual(candidate.GetStrings("dominatedBy"), experiment.GetStrings("selectedPrograms")) {
			t.Fatalf("longer exact candidate %v domination = %v", sequence, candidate.GetStrings("dominatedBy"))
		}
	}
	if len(tinyStackMembers(store, evidenceCategory)) != 399 || len(tinyStackMembers(store, observationCategory)) != 1596 || len(tinyStackMembers(store, resultCategory)) != 984 || actualResults != 984 || undefined != 612 {
		t.Fatalf("actual synthesis artifact counts: evidence=%d observations=%d results=%d oracle-results=%d undefined=%d", len(tinyStackMembers(store, evidenceCategory)), len(tinyStackMembers(store, observationCategory)), len(tinyStackMembers(store, resultCategory)), actualResults, undefined)
	}

	assertIndependentSimplificationMatrix(t, store, experiment, candidates, opcodes, probeCategory)
	assertExactTinyStackApplications(t, store, experiment, candidates)
}

func assertExactTinyStackApplications(t *testing.T, store *unit.Store, experiment *unit.Unit, candidates map[string]*unit.Unit) {
	t.Helper()
	expected := map[string][]string{}
	synthesisCount, simplificationCount := 0, 0
	for _, result := range tinyStackMembers(store, experiment.GetString("resultCategory")) {
		expected[result.GetString("program")+"\x00"+result.Name] = []string{result.GetString("example")}
		synthesisCount++
	}
	for _, result := range tinyStackMembers(store, experiment.GetString("simplificationExecutionResultCategory")) {
		expected[result.GetString("subjectProgram")+"\x00"+result.Name] = []string{result.GetString("probe")}
		simplificationCount++
	}
	if synthesisCount != 984 || simplificationCount != 213 {
		t.Fatalf("expected application sources = synthesis %d simplification %d", synthesisCount, simplificationCount)
	}
	var subjects []*unit.Unit
	for _, candidate := range candidates {
		subjects = append(subjects, candidate)
	}
	subjects = append(subjects, tinyStackMembers(store, experiment.GetString("primitiveCategory"))...)
	seen := map[string]bool{}
	for _, subject := range subjects {
		applics, _ := subject.Get("applics").([]map[string]any)
		for _, applic := range applics {
			direct, _ := applic["direct"].(bool)
			if !direct {
				continue
			}
			output, outputOK := applic["output"].(string)
			if !outputOK || output == "" {
				t.Fatalf("malformed direct application on %s: %#v", subject.Name, applic)
			}
			key := subject.Name + "\x00" + output
			args, argsOK := applic["args"].([]string)
			wantArgs, expectedApplication := expected[key]
			if !expectedApplication || seen[key] || !argsOK || !reflect.DeepEqual(args, wantArgs) || applic["target"] != subject.Name || applic["result"] != true {
				t.Fatalf("unexpected direct application on %s: %#v", subject.Name, applic)
			}
			seen[key] = true
		}
	}
	if len(seen) != len(expected) {
		for key := range expected {
			if !seen[key] {
				t.Fatalf("missing direct application %q", key)
			}
		}
	}
}

func assertIndependentSimplificationMatrix(t *testing.T, store *unit.Store, experiment *unit.Unit, candidates map[string]*unit.Unit, opcodes []string, probeCategory string) {
	t.Helper()
	primitives := map[string]*unit.Unit{}
	for _, primitive := range tinyStackMembers(store, experiment.GetString("primitiveCategory")) {
		primitives[primitive.GetString(experiment.GetString("primitiveSemanticSlot"))] = primitive
	}
	probes := tinyStackMembers(store, probeCategory)
	execCategory := experiment.GetString("simplificationExecutionObservationCategory")
	execResultCategory := experiment.GetString("simplificationExecutionResultCategory")
	comparisonCategory := experiment.GetString("simplificationComparisonObservationCategory")
	evidenceCategory := experiment.GetString("simplificationEvidenceCategory")
	pairCategory := experiment.GetString("simplificationPairCategory")
	executions := indexByPair(store, execCategory, "subjectProgram", "probe")
	pairEvidence := indexByPair(store, evidenceCategory, "longProgram", "shortProgram")

	var longPrograms []*unit.Unit
	for _, sequence := range oracleSequences(opcodes, 2) {
		if len(sequence) == 2 {
			longPrograms = append(longPrograms, candidates[semanticKey(sequence)])
		}
	}
	validExecutions, undefinedExecutions := 0, 0
	for _, subject := range append(append([]*unit.Unit(nil), longPrograms...), mapValuesSorted(primitives)...) {
		sequence := subject.GetStrings("semanticSequence")
		if len(sequence) == 0 {
			sequence = []string{subject.GetString(experiment.GetString("primitiveSemanticSlot"))}
		}
		for _, probe := range probes {
			input, _ := intsSlot(probe, experiment.GetString("probeInputSlot"))
			actual, defined := oracleStackProgram(input, sequence)
			observation := executions[subject.Name+"\x00"+probe.Name]
			if observation == nil || observation.GetBool("defined") != defined {
				t.Fatalf("execution observation mismatch for %v/%s", sequence, probe.Name)
			}
			if defined {
				validExecutions++
				observed, ok := intsSlot(observation, "actual")
				result := store.Get(observation.GetString("resultUnit"))
				resultData, resultOK := intsSlot(result, experiment.GetString("resultValueSlot"))
				if !ok || !reflect.DeepEqual(observed, actual) || result == nil || !store.IsA(result.Name, execResultCategory) || result.GetString("subjectProgram") != subject.Name || result.GetString("probe") != probe.Name || !resultOK || !reflect.DeepEqual(resultData, actual) || !hasDirectOutput(subject, result.Name) {
					t.Fatalf("execution result mismatch for %v/%s", sequence, probe.Name)
				}
			} else {
				undefinedExecutions++
				if observation.GetString("resultUnit") != "" || observation.Get("actual") != nil {
					t.Fatalf("undefined execution retained result for %v/%s", sequence, probe.Name)
				}
			}
		}
	}

	equivalent := map[string]bool{}
	for _, long := range longPrograms {
		longSequence := long.GetStrings("semanticSequence")
		for _, opcode := range opcodes {
			short := primitives[opcode]
			bothDefined, bothUndefined, oneUndefined, definedMismatch, mismatch, support := 0, 0, 0, 0, 0, 0
			expectedOutcomes := map[string]bool{}
			expectedStatuses := map[string]string{}
			for _, probe := range probes {
				input, _ := intsSlot(probe, experiment.GetString("probeInputSlot"))
				left, leftDefined := oracleStackProgram(input, longSequence)
				right, rightDefined := oracleStackProgram(input, []string{opcode})
				agree := false
				status := "one-undefined"
				switch {
				case leftDefined && rightDefined:
					bothDefined++
					agree = reflect.DeepEqual(left, right)
					if !agree {
						definedMismatch++
						status = "both-defined-mismatch"
					} else {
						status = "both-defined-match"
					}
				case !leftDefined && !rightDefined:
					bothUndefined++
					agree = true
					status = "both-undefined"
				default:
					oneUndefined++
				}
				if agree {
					support++
				} else {
					mismatch++
				}
				expectedOutcomes[probe.Name] = agree
				expectedStatuses[probe.Name] = status
			}
			isEquivalent := mismatch == 0 && oneUndefined == 0 && bothDefined >= 3
			if isEquivalent {
				equivalent[semanticKey(longSequence)+"->"+opcode] = true
			}
			evidence := pairEvidence[long.Name+"\x00"+short.Name]
			if evidence == nil || evidence.GetInt("bothDefinedCount") != bothDefined || evidence.GetInt("bothUndefinedCount") != bothUndefined || evidence.GetInt("oneUndefinedCount") != oneUndefined || evidence.GetInt("definedMismatchCount") != definedMismatch || evidence.GetInt("mismatchCount") != mismatch || evidence.GetInt("supportCount") != support || evidence.GetBool("equivalent") != isEquivalent || len(evidence.GetStrings("comparisonObservations")) != len(probes) {
				t.Fatalf("pair evidence mismatch for %v -> %s: %#v", longSequence, opcode, evidence)
			}
			seenProbes := map[string]bool{}
			for _, observationName := range evidence.GetStrings("comparisonObservations") {
				observation := store.Get(observationName)
				probe := ""
				if observation != nil {
					probe = observation.GetString("probe")
				}
				if observation == nil || seenProbes[probe] || !store.IsA(observationName, comparisonCategory) || observation.GetString("pair") != evidence.GetString("pair") || observation.GetBool("outcome") != expectedOutcomes[probe] || observation.GetString("status") != expectedStatuses[probe] || observation.GetString("longExecutionObservation") != executions[long.Name+"\x00"+probe].Name || observation.GetString("shortExecutionObservation") != executions[short.Name+"\x00"+probe].Name {
					t.Fatalf("comparison linkage mismatch for %v -> %s", longSequence, opcode)
				}
				seenProbes[probe] = true
			}
			if len(seenProbes) != len(probes) {
				t.Fatalf("comparison probe coverage for %v -> %s = %d", longSequence, opcode, len(seenProbes))
			}
		}
	}
	if len(tinyStackMembers(store, execCategory)) != 336 || len(tinyStackMembers(store, execResultCategory)) != 213 || validExecutions != 213 || undefinedExecutions != 123 || len(tinyStackMembers(store, comparisonCategory)) != 2058 || len(tinyStackMembers(store, pairCategory)) != 343 || len(tinyStackMembers(store, evidenceCategory)) != 343 {
		t.Fatalf("actual simplification artifact counts: exec=%d results=%d oracle-results=%d undefined=%d comparisons=%d pairs=%d evidence=%d", len(tinyStackMembers(store, execCategory)), len(tinyStackMembers(store, execResultCategory)), validExecutions, undefinedExecutions, len(tinyStackMembers(store, comparisonCategory)), len(tinyStackMembers(store, pairCategory)), len(tinyStackMembers(store, evidenceCategory)))
	}
	var promoted []string
	for _, schema := range tinyStackMembers(store, experiment.GetString("simplificationSchemaCategory")) {
		long := store.Get(schema.GetString("longProgram"))
		short := store.Get(schema.GetString("shortProgram"))
		promoted = append(promoted, semanticKey(long.GetStrings("semanticSequence"))+"->"+short.GetString(experiment.GetString("primitiveSemanticSlot")))
	}
	sort.Strings(promoted)
	var oraclePromoted []string
	for relation := range equivalent {
		oraclePromoted = append(oraclePromoted, relation)
	}
	sort.Strings(oraclePromoted)
	if !reflect.DeepEqual(promoted, oraclePromoted) {
		t.Fatalf("promoted simplifications differ from oracle: got %v want %v", promoted, oraclePromoted)
	}
}

func mapValuesSorted(values map[string]*unit.Unit) []*unit.Unit {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]*unit.Unit, len(keys))
	for index, key := range keys {
		result[index] = values[key]
	}
	return result
}
