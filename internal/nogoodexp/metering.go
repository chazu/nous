package nogoodexp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/nogoodbaseline"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func chargeMeterOperations(token string, category int, subject string, operations []string) error {
	for _, operation := range operations {
		if err := dsl.ChargeNogoodMeter(token, operation, subject, operation, "ok", category); err != nil {
			return err
		}
	}
	return nil
}

var engineDispatchOperations = []string{
	"task-number-increment",
	"category-store-count-before", "category-agenda-length-before", "category-store-count-after", "category-agenda-length-after",
	"category-applicability-read", "category-applicability-write", "category-result-read", "category-result-write",
	"lane-store-count-before", "lane-agenda-length-before", "lane-store-count-after", "lane-agenda-length-after",
	"lane-applicability-read", "lane-applicability-write", "lane-result-read", "lane-result-write",
	"lane-then-record-heuristic-read", "lane-then-record-record-read", "lane-then-record-write",
	"finished-category-dispatch", "finished-lane-dispatch",
}

func baselineTranscript(taskOrdinal uint32, result nogoodbaseline.Result) ([]TranscriptEvent, error) {
	events := make([]TranscriptEvent, 0, len(result.Events))
	for _, source := range result.Events {
		if source.Category < 1 || source.Category > 12 {
			return nil, fmt.Errorf("invalid baseline category %d", source.Category)
		}
		operation := source.Transition
		outcome := "ok"
		operand := func(index int) int {
			if index < len(source.Operands) {
				return source.Operands[index]
			}
			return 0
		}
		variable, color := fmt.Sprintf("v:%d", operand(0)), fmt.Sprintf("c:%d", operand(1))
		event := TranscriptEvent{Category: uint8(source.Category), TaskOrdinal: taskOrdinal}
		switch source.Category {
		case 1:
			event.Code = 1
			event.Operands = [8]TranscriptOperand{ID("candidate"), OptionalID(""), Number(int32(operand(0))), Number(int32(operand(1))), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 2:
			event.Code = 2
			event.Operands = [8]TranscriptOperand{ID(variable), ID(operation), ID(color), ID(outcome), Omitted(), Omitted(), Omitted(), Omitted()}
		case 3:
			event.Code = 3
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), Number(0), ID(operation), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 4:
			event.Code = 4
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), ID(fmt.Sprintf("v:%d", operand(2))), ID(fmt.Sprintf("c:%d", operand(3))), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 5:
			event.Code = 5
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), ID(operation), OptionalID(""), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 6:
			event.Code = 6
			event.Operands = [8]TranscriptOperand{ID(variable), ID(fmt.Sprintf("v:%d", operand(1))), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 7:
			event.Code = 7
			event.Operands = [8]TranscriptOperand{ID("conflict:" + operation), OptionalID(variable), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 8:
			event.Code = 8
			event.Operands = [8]TranscriptOperand{OptionalID("artifact"), OptionalID("binding"), Number(int32(operand(0))), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 9:
			event.Code = 9
			event.Operands = [8]TranscriptOperand{ID("binding"), ID("completion"), OptionalID(""), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 10:
			event.Code = 10
			event.Operands = [8]TranscriptOperand{ID("certificate"), OptionalID(""), ID(operation), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 11:
			event.Code = 11
			event.Operands = [8]TranscriptOperand{ID(operation), Number(1), Omitted(), Omitted(), Omitted(), Omitted(), Omitted(), Omitted()}
		case 12:
			event.Code = 18
			event.Operands = [8]TranscriptOperand{ID("record:" + operation), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted(), Omitted()}
		}
		events = append(events, event)
	}
	return events, nil
}

func fixedVectorEvents(taskOrdinal uint32, vector [12]int64, prefix string) []TranscriptEvent {
	var events []TranscriptEvent
	for category, count := range vector {
		for index := int64(0); index < count; index++ {
			operation := fmt.Sprintf("%s:c%d", prefix, category+1)
			source := nogoodbaseline.Event{Category: category + 1, Transition: operation, Operands: []int{int(index), category}}
			mapped, _ := baselineTranscript(taskOrdinal, nogoodbaseline.Result{Events: []nogoodbaseline.Event{source}})
			events = append(events, mapped[0])
		}
	}
	return events
}

func acquisitionTranscript(run TrainingRun, preflight []TranscriptEvent) ([]TranscriptEvent, error) {
	if err := auditTrainingMeter(run); err != nil {
		return nil, err
	}
	events := slices.Clone(preflight)
	for index, record := range run.MeterRecords {
		event, err := meterRecordTranscript(0x80000000, index, record)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func transcriptVector(events []TranscriptEvent) (vector [12]int64) {
	for _, event := range events {
		vector[event.Category-1]++
	}
	return vector
}

func transcriptWork(events []TranscriptEvent) int64 { return int64(len(events)) }

func appendEvents(first, second []TranscriptEvent) []TranscriptEvent {
	return append(slices.Clone(first), second...)
}

func bridgeTranscript(taskOrdinal uint32, disposition Disposition) ([]TranscriptEvent, error) {
	if len(disposition.MeterRecords) == 0 {
		return nil, fmt.Errorf("bridge emitted no verifier-owned meter records")
	}
	if err := auditBridgeMeter(disposition); err != nil {
		return nil, err
	}
	events := make([]TranscriptEvent, 0, len(disposition.MeterRecords))
	for index, record := range disposition.MeterRecords {
		event, err := meterRecordTranscript(taskOrdinal, index, record)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func auditTrainingMeter(run TrainingRun) error {
	counts := meterOperationCounts(run.MeterRecords)
	candidates := len(run.Store.Examples("NogoodCandidate")) - 1
	refinements := len(run.Store.Examples("NogoodRefinement")) - 1
	bindings := len(run.Store.Examples("NogoodBinding")) - 1
	results := len(run.Store.Examples("NogoodResult")) - 1
	evidence := len(run.Store.Examples("NogoodEvidence")) - 1
	proofs := len(run.Store.Examples("NogoodPromotionProof")) - 1
	selectionAttempts := run.TasksPopped - 2*candidates - 2
	want := map[string]int{
		"candidate-proposal": 1, "candidate-write": candidates, "semantic-key-read": 1 + refinements,
		"refinement-proposal": refinements, "refinement-record-write": refinements,
		"agenda-enqueue": 2 + 3*candidates, "agenda-dequeue": run.TasksPopped,
		"problem-read": evidence, "decision-variable-read": evidence, "decision-color-read": evidence,
		"binding-proposal": evidence, "binding-write": bindings,
		"artifact-read": bindings, "mask-bit-check": 3 * bindings, "edge-read": 3 * bindings,
		"completion-construct": results, "domain-read-x": results, "domain-read-y": results, "inequality-check": 3 * results,
		"result-write": results, "result-read": results, "evidence-write": evidence,
		"expected-key-check": results, "actual-key-check": results, "key-set-equality": evidence, "completion-count-check": evidence, "barrier-write": evidence,
		"selection-evidence-read": candidates * selectionAttempts, "selection-comparison": candidates * selectionAttempts,
		"expected-mask-check": candidates * selectionAttempts, "actual-mask-check": candidates * selectionAttempts,
		"mask-set-equality": selectionAttempts, "candidate-count-check": selectionAttempts, "complete-count-check": selectionAttempts, "selection-barrier-write": selectionAttempts,
		"tie-record-write": 1, "selection-record-write": 1,
		"promotion-completion": proofs, "promotion-proof-write": proofs, "expected-promotion-check": proofs, "actual-promotion-check": proofs,
		"promotion-count-check": 1, "promotion-conflict-check": 1, "promotion-barrier-write": 1,
		"artifact-freeze-write": 1, "provenance-write": 1, "boundary-write": 1,
	}
	for _, operation := range engineDispatchOperations {
		want[operation] = run.TasksPopped
	}
	if len(counts) != len(want) {
		return fmt.Errorf("training meter has %d operation kinds, reconstructed ledger requires %d", len(counts), len(want))
	}
	for operation, expected := range want {
		if counts[operation] != expected {
			return fmt.Errorf("training meter %s=%d, authoritative store requires %d", operation, counts[operation], expected)
		}
	}
	if proofs != 24 {
		return fmt.Errorf("training promotion meter does not cover the sealed 24-case boundary")
	}
	expectedRecords, err := reconstructTrainingMeter(run)
	if err != nil {
		return err
	}
	if err := compareMeterMultiset(run.MeterRecords, expectedRecords); err != nil {
		return fmt.Errorf("training meter reconstruction: %w", err)
	}
	return auditTrainingMeterTuples(run)
}

func auditTrainingMeterTuples(run TrainingRun) error {
	completionKeys := map[string]bool{}
	for _, resultName := range run.Store.Examples("NogoodResult") {
		if resultName == "NogoodResult" {
			continue
		}
		result := run.Store.Get(resultName)
		completionKeys[fmt.Sprintf("completion:%d:%d", result.GetInt("xColor"), result.GetInt("yColor"))] = true
	}
	for _, record := range run.MeterRecords {
		if record.Outcome != "ok" {
			return fmt.Errorf("training meter has non-ok tuple outcome %q", record.Outcome)
		}
		switch record.Operation {
		case "candidate-proposal":
			if !run.Store.IsA(record.Subject, "NogoodLearningExperiment") || !run.Store.IsA(record.Object, "NogoodCandidate") {
				return fmt.Errorf("training candidate proposal tuple is retargeted")
			}
		case "candidate-write":
			if run.Store.Get(record.Subject) == nil || !run.Store.IsA(record.Object, "NogoodCandidate") {
				return fmt.Errorf("training candidate write tuple is retargeted")
			}
		case "semantic-key-read", "refinement-proposal":
			if run.Store.Get(record.Subject) == nil || !strings.HasPrefix(record.Object, "mask:") {
				return fmt.Errorf("training refinement key tuple is retargeted")
			}
		case "refinement-record-write":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || !run.Store.IsA(record.Object, "NogoodRefinement") {
				return fmt.Errorf("training refinement tuple is retargeted")
			}
		case "agenda-enqueue", "agenda-dequeue":
			if run.Store.Get(record.Subject) == nil || !slices.Contains([]string{"ngStart", "ngRefine", "ngEvaluate", "ngSelect", "ngPromote"}, record.Object) {
				return fmt.Errorf("training agenda tuple is retargeted")
			}
		case "problem-read":
			if !run.Store.IsA(record.Subject, "NogoodTrainingExample") || record.Object != "problem" {
				return fmt.Errorf("training problem-read tuple is retargeted")
			}
		case "decision-variable-read":
			if !run.Store.IsA(record.Subject, "NogoodTrainingExample") || record.Object != "decisionVariable" {
				return fmt.Errorf("training decision-variable tuple is retargeted")
			}
		case "decision-color-read":
			if !run.Store.IsA(record.Subject, "NogoodTrainingExample") || record.Object != "decisionColor" {
				return fmt.Errorf("training decision-color tuple is retargeted")
			}
		case "domain-read-x", "domain-read-y", "completion-construct":
			if !run.Store.IsA(record.Subject, "NogoodBinding") || !completionKeys[record.Object] {
				return fmt.Errorf("training completion tuple is retargeted")
			}
		case "result-read":
			if !run.Store.IsA(record.Subject, "NogoodEvidence") || !run.Store.IsA(record.Object, "NogoodResult") {
				return fmt.Errorf("training result-read tuple is retargeted")
			}
		case "result-write":
			if !run.Store.IsA(record.Subject, "NogoodBinding") || !run.Store.IsA(record.Object, "NogoodResult") {
				return fmt.Errorf("training result-write tuple is retargeted")
			}
		case "binding-proposal", "binding-write":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || !run.Store.IsA(record.Object, "NogoodBinding") {
				return fmt.Errorf("training binding tuple is retargeted")
			}
		case "evidence-write":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || !run.Store.IsA(record.Object, "NogoodEvidence") {
				return fmt.Errorf("training evidence tuple is retargeted")
			}
		case "promotion-proof-write":
			if !run.Store.IsA(record.Subject, "NogoodPromotionCase") || !run.Store.IsA(record.Object, "NogoodPromotionProof") {
				return fmt.Errorf("training promotion tuple is retargeted")
			}
		case "artifact-read":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || record.Object != "mask" {
				return fmt.Errorf("training artifact-read tuple is retargeted")
			}
		case "mask-bit-check":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || !slices.Contains([]string{"bit:0", "bit:1", "bit:2"}, record.Object) {
				return fmt.Errorf("training mask-bit tuple is retargeted")
			}
		case "edge-read":
			if !run.Store.IsA(record.Subject, "NogoodTrainingExample") || !slices.Contains([]string{"a-x", "a-y", "x-y"}, record.Object) {
				return fmt.Errorf("training edge-read tuple is retargeted")
			}
		case "inequality-check":
			if !run.Store.IsA(record.Subject, "NogoodBinding") || !slices.Contains([]string{"a-x", "a-y", "x-y"}, record.Object) {
				return fmt.Errorf("training inequality tuple is retargeted")
			}
		case "expected-key-check", "actual-key-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || !completionKeys[record.Object] {
				return fmt.Errorf("training evidence-key tuple is retargeted")
			}
		case "key-set-equality":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != "expected-actual" {
				return fmt.Errorf("training evidence equality tuple is retargeted")
			}
		case "completion-count-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != "completionCount" {
				return fmt.Errorf("training completion count tuple is retargeted")
			}
		case "barrier-write":
			if !run.Store.IsA(record.Subject, "NogoodCandidate") || !run.Store.IsA(record.Object, "NogoodEvidenceBarrier") {
				return fmt.Errorf("training barrier tuple is retargeted")
			}
		case "selection-evidence-read", "selection-comparison":
			if !run.Store.IsA(record.Subject, "NogoodLearningExperiment") || !run.Store.IsA(record.Object, "NogoodCandidate") {
				return fmt.Errorf("training selection tuple is retargeted")
			}
		case "expected-mask-check", "actual-mask-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || !strings.HasPrefix(record.Object, "mask:") {
				return fmt.Errorf("training selection mask tuple is retargeted")
			}
		case "mask-set-equality":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != "expected-actual" {
				return fmt.Errorf("training mask equality tuple is retargeted")
			}
		case "candidate-count-check", "complete-count-check":
			wantObject := "candidateCount"
			if record.Operation == "complete-count-check" {
				wantObject = "completeCount"
			}
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != wantObject {
				return fmt.Errorf("training selection count tuple is retargeted")
			}
		case "selection-barrier-write":
			if !run.Store.IsA(record.Subject, "NogoodLearningExperiment") || !run.Store.IsA(record.Object, "NogoodEvidenceBarrier") {
				return fmt.Errorf("training selection barrier tuple is retargeted")
			}
		case "tie-record-write":
			if !run.Store.IsA(record.Subject, "NogoodSelection") || record.Object != "exactCandidates" {
				return fmt.Errorf("training tie tuple is retargeted")
			}
		case "selection-record-write":
			if !run.Store.IsA(record.Subject, "NogoodLearningExperiment") || !run.Store.IsA(record.Object, "NogoodSelection") {
				return fmt.Errorf("training selection record tuple is retargeted")
			}
		case "promotion-completion":
			if !run.Store.IsA(record.Subject, "NogoodPromotionCase") || record.Object != "only-only" {
				return fmt.Errorf("training promotion completion tuple is retargeted")
			}
		case "expected-promotion-check", "actual-promotion-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || !run.Store.IsA(record.Object, "NogoodPromotionCase") {
				return fmt.Errorf("training promotion boundary tuple is retargeted")
			}
		case "promotion-count-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != "proofCount" {
				return fmt.Errorf("training promotion count tuple is retargeted")
			}
		case "promotion-conflict-check":
			if !run.Store.IsA(record.Subject, "NogoodEvidenceBarrier") || record.Object != "conflictCount" {
				return fmt.Errorf("training promotion conflict tuple is retargeted")
			}
		case "promotion-barrier-write":
			if !run.Store.IsA(record.Subject, "NogoodLearningExperiment") || !run.Store.IsA(record.Object, "NogoodEvidenceBarrier") {
				return fmt.Errorf("training promotion barrier tuple is retargeted")
			}
		case "artifact-freeze-write":
			if !run.Store.IsA(record.Subject, "NogoodSelection") || !run.Store.IsA(record.Object, "NogoodArtifact") {
				return fmt.Errorf("training artifact freeze tuple is retargeted")
			}
		case "provenance-write":
			if !run.Store.IsA(record.Subject, "NogoodArtifact") || record.Object != nogoodfixtureAuthority {
				return fmt.Errorf("training provenance tuple is retargeted")
			}
		case "boundary-write":
			if !run.Store.IsA(record.Subject, "NogoodArtifact") || !run.Store.IsA(record.Object, "NogoodEvidenceBarrier") {
				return fmt.Errorf("training boundary tuple is retargeted")
			}
		default:
			if slices.Contains(engineDispatchOperations, record.Operation) {
				if record.Object != record.Operation || !strings.Contains(record.Subject, ".") {
					return fmt.Errorf("training engine dispatch tuple is retargeted")
				}
				continue
			}
			return fmt.Errorf("training meter contains unaudited operation %q", record.Operation)
		}
	}
	return nil
}

func auditBridgeMeter(disposition Disposition) error {
	if disposition.Store == nil || disposition.Request == "" {
		return fmt.Errorf("bridge meter has no authoritative occurrence store")
	}
	request := disposition.Store.Get(disposition.Request)
	if request == nil {
		return fmt.Errorf("bridge meter request is missing")
	}
	problem, err := nogoods.ParseProblem([]byte(request.GetString("problem")))
	if err != nil {
		return err
	}
	counts := meterOperationCounts(disposition.MeterRecords)
	expectedRecords, err := reconstructBridgeMeter(disposition)
	if err != nil {
		return err
	}
	if err := compareMeterMultiset(disposition.MeterRecords, expectedRecords); err != nil {
		return fmt.Errorf("bridge meter reconstruction: %w", err)
	}
	for _, operation := range engineDispatchOperations {
		if counts[operation] != disposition.TasksPopped {
			return fmt.Errorf("bridge dispatch meter %s=%d, want %d", operation, counts[operation], disposition.TasksPopped)
		}
	}
	expectedVisits := 0
	anchor, blocked := request.GetInt("decisionVariable"), request.GetInt("decisionColor")
	if len(problem.Variables[anchor].Domain) == 2 && problem.DomainContains(anchor, blocked) {
		expectedVisits = len(problem.Variables) - 1
	}
	if counts["role-visit-record"] != expectedVisits || counts["role-candidate"] != len(disposition.Store.Examples("NogoodRoleCandidate"))-1 || counts["role-candidate-write"] != counts["role-candidate"] || counts["pair-candidate"] != len(disposition.Store.Examples("NogoodPairProposal"))-1 || counts["pair-record-write"] != counts["pair-candidate"] {
		return fmt.Errorf("bridge matcher meter does not reconcile with its authoritative units")
	}
	adapterOperation := "adapter-resume-check-"
	dispositionChecks := 4
	if disposition.Status == "concrete-prune" {
		adapterOperation = "adapter-concrete-check-"
	}
	if disposition.Status == "propose-prune" {
		adapterOperation = "adapter-proposal-check-"
		dispositionChecks = 6
		if disposition.Barrier == "" || disposition.Store.Get(disposition.Barrier) == nil {
			return fmt.Errorf("proposal meter has no barrier")
		}
		barrier := disposition.Store.Get(disposition.Barrier)
		if counts["barrier-predicate-check"] != len(barrier.GetStrings("predicateKeys")) || counts["barrier-predicate-check"] != 18 {
			return fmt.Errorf("proposal meter does not cover the 18 barrier predicates")
		}
		wantVector := [12]int64{3, 23, 2, 3, 2, 0, 0, 10, 1, 25, 0, 51}
		events := make([]TranscriptEvent, 0, len(disposition.MeterRecords))
		for index, record := range disposition.MeterRecords {
			event, eventErr := meterRecordTranscript(0, index, record)
			if eventErr != nil {
				return eventErr
			}
			events = append(events, event)
		}
		if got := transcriptVector(events); got != wantVector {
			return fmt.Errorf("proposal meter vector %v does not match audited transition set %v", got, wantVector)
		}
		expectedOperations := map[string]int{
			"root-domain-read": 1, "root-propose": 1, "root-bind": 1, "root-delete": 1, "root-empty-check": 1,
			"request-write": 1, "agenda-enqueue": 1, "agenda-dequeue": 1, "request-digest-check": 2,
			"domain-read": 1, "domain-size-check": 7, "domain-membership-check": 7, "role-visit-record": 7,
			"role-candidate": 2, "role-candidate-write": 2, "pair-candidate": 1, "pair-only-equality": 1, "pair-escape-inequality": 1, "pair-record-write": 1,
			"artifact-read": 1, "mask-bit-0": 1, "mask-bit-1": 1, "mask-bit-2": 1, "authority-read": 1, "frozen-read": 1, "schema-read": 1, "guard-version-read": 1, "artifact-digest-read": 1, "evidence-digest-read": 1,
			"artifact-edge-read": 3, "artifact-match-read": 1, "artifact-match-record": 1,
			"completion-construct": 1, "completion-domain-read": 2, "completion-inequality": 3, "completion-result-write": 1,
			"certificate-record": 1, "barrier-predicate-check": 18, "certificate-index-write": 1, "agreement-result-read": 1, "agreement-record-write": 1, "expected-key-set-write": 1, "actual-key-set-write": 1, "sealed-barrier-write": 1,
			"disposition-write": 1, "target-digest-check": 1, "decision-digest-check": 1, "assignment-digest-check": 1, "artifact-digest-check": 1,
		}
		for _, operation := range engineDispatchOperations {
			expectedOperations[operation] = 1
		}
		for index := 1; index <= 6; index++ {
			expectedOperations[fmt.Sprintf("adapter-proposal-check-%d", index)] = 1
		}
		if len(counts) != len(expectedOperations) {
			return fmt.Errorf("proposal meter has %d operation kinds, want %d", len(counts), len(expectedOperations))
		}
		for operation, expected := range expectedOperations {
			if counts[operation] != expected {
				return fmt.Errorf("proposal meter %s=%d, want %d", operation, counts[operation], expected)
			}
		}
		predicateSet := map[string]bool{}
		for _, predicate := range barrier.GetStrings("predicateKeys") {
			predicateSet[predicate] = true
		}
		seenPredicates := map[string]bool{}
		for _, record := range disposition.MeterRecords {
			if record.Outcome != "ok" {
				return fmt.Errorf("proposal meter recorded non-ok outcome %q", record.Outcome)
			}
			switch record.Operation {
			case "role-candidate", "role-candidate-write":
				if !disposition.Store.IsA(record.Object, "NogoodRoleCandidate") {
					return fmt.Errorf("proposal meter role object %q is not authoritative", record.Object)
				}
			case "barrier-predicate-check":
				if !predicateSet[record.Object] || seenPredicates[record.Object] {
					return fmt.Errorf("proposal meter predicate %q is absent or duplicated", record.Object)
				}
				seenPredicates[record.Object] = true
			}
		}
	}
	for index := 1; index <= 6; index++ {
		if counts[fmt.Sprintf("%s%d", adapterOperation, index)] != 1 {
			return fmt.Errorf("bridge meter omitted adapter check %d", index)
		}
	}
	if counts["disposition-write"] != 1 || counts["request-digest-check"] != 2 || counts["target-digest-check"] != 1 || counts["decision-digest-check"] != 1 || dispositionChecks == 6 && (counts["assignment-digest-check"] != 1 || counts["artifact-digest-check"] != 1) {
		return fmt.Errorf("bridge meter disposition checks do not reconcile")
	}
	return nil
}

func meterOperationCounts(records []dsl.NogoodMeterRecord) map[string]int {
	counts := make(map[string]int)
	for _, record := range records {
		counts[record.Operation]++
	}
	return counts
}

func meterRecordTranscript(taskOrdinal uint32, index int, record dsl.NogoodMeterRecord) (TranscriptEvent, error) {
	if record.Category < 1 || record.Category > 12 || record.Operation == "" || record.Subject == "" || record.Object == "" || record.Outcome == "" {
		return TranscriptEvent{}, fmt.Errorf("invalid verifier-owned meter record %d", index)
	}
	event := TranscriptEvent{Category: record.Category, TaskOrdinal: taskOrdinal}
	subject, object, operation, outcome := ID(record.Subject), ID(record.Object), ID(record.Operation), ID(record.Outcome)
	switch record.Category {
	case 1:
		event.Code, event.Operands = 1, [8]TranscriptOperand{subject, OptionalID(record.Object), Number(int32(index)), Number(0), operation, outcome, Omitted(), Omitted()}
	case 2:
		event.Code, event.Operands = 2, [8]TranscriptOperand{subject, operation, object, outcome, Omitted(), Omitted(), Omitted(), Omitted()}
	case 3:
		event.Code, event.Operands = 3, [8]TranscriptOperand{subject, object, Number(0), operation, outcome, Omitted(), Omitted(), Omitted()}
	case 4:
		event.Code, event.Operands = 4, [8]TranscriptOperand{subject, object, operation, outcome, outcome, Omitted(), Omitted(), Omitted()}
	case 5:
		event.Code, event.Operands = 5, [8]TranscriptOperand{subject, object, operation, OptionalID(record.Object), outcome, Omitted(), Omitted(), Omitted()}
	case 6:
		event.Code, event.Operands = 6, [8]TranscriptOperand{subject, object, OptionalID(record.Object), Number(int32(index)), operation, outcome, Omitted(), Omitted()}
	case 7:
		event.Code, event.Operands = 7, [8]TranscriptOperand{subject, OptionalID(record.Object), OptionalID(record.Operation), Number(int32(index)), operation, outcome, Omitted(), Omitted()}
	case 8:
		event.Code, event.Operands = 8, [8]TranscriptOperand{OptionalID(record.Subject), OptionalID(record.Object), Number(int32(index)), OptionalID(record.Operation), OptionalID(record.Outcome), operation, outcome, Omitted()}
	case 9:
		event.Code, event.Operands = 9, [8]TranscriptOperand{subject, object, OptionalID(record.Operation), OptionalID(record.Subject), OptionalID(record.Object), operation, outcome, Omitted()}
	case 10:
		event.Code, event.Operands = 10, [8]TranscriptOperand{subject, OptionalID(record.Object), operation, OptionalID(record.Subject), OptionalID(record.Object), operation, outcome, Omitted()}
	case 11:
		event.Code, event.Operands = 11, [8]TranscriptOperand{ID(record.Operation + ":" + record.Subject + ":" + record.Object + ":" + record.Outcome), Number(1), Omitted(), Omitted(), Omitted(), Omitted(), Omitted(), Omitted()}
	case 12:
		event.Code, event.Operands = 18, [8]TranscriptOperand{subject, OptionalID(record.Object), Number(int32(index)), operation, outcome, Omitted(), Omitted(), Omitted()}
	}
	return event, nil
}
