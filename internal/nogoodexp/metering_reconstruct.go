package nogoodexp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func reconstructTrainingMeter(run TrainingRun) ([]dsl.NogoodMeterRecord, error) {
	store := run.Store
	if store == nil {
		return nil, fmt.Errorf("missing training occurrence store")
	}
	experiment, err := soleExample(store, "NogoodLearningExperiment")
	if err != nil {
		return nil, err
	}
	var records []dsl.NogoodMeterRecord
	add := func(category int, operation, subject, object string) {
		records = append(records, dsl.NogoodMeterRecord{Category: uint8(category), Operation: operation, Subject: subject, Object: object, Outcome: "ok"})
	}
	addDispatch := func(unitName, slotName string) {
		add(12, "agenda-dequeue", unitName, slotName)
		for _, operation := range engineDispatchOperations {
			add(12, operation, unitName+"."+slotName, operation)
		}
	}

	candidates := map[int]*unit.Unit{}
	for _, name := range store.Examples("NogoodCandidate") {
		if name != "NogoodCandidate" {
			candidate := store.Get(name)
			candidates[candidate.GetInt("mask")] = candidate
		}
	}
	if len(candidates) != 8 {
		return nil, fmt.Errorf("training occurrence has %d candidates", len(candidates))
	}
	root := candidates[0]
	add(12, "agenda-enqueue", experiment.Name, "ngStart")
	add(1, "candidate-proposal", experiment.Name, root.Name)
	add(12, "semantic-key-read", experiment.Name, "mask:0")
	add(12, "candidate-write", experiment.Name, root.Name)
	add(12, "agenda-enqueue", root.Name, "ngRefine")

	refinementsByParent := map[string][]*unit.Unit{}
	for _, name := range store.Examples("NogoodRefinement") {
		if name == "NogoodRefinement" {
			continue
		}
		refinement := store.Get(name)
		refinementsByParent[refinement.GetString("parent")] = append(refinementsByParent[refinement.GetString("parent")], refinement)
	}
	for mask := 0; mask < 8; mask++ {
		candidate := candidates[mask]
		refinements := refinementsByParent[candidate.Name]
		sort.Slice(refinements, func(i, j int) bool { return refinements[i].GetInt("addedBit") < refinements[j].GetInt("addedBit") })
		for _, refinement := range refinements {
			child := store.Get(refinement.GetString("child"))
			if child == nil {
				return nil, fmt.Errorf("training refinement %s has no child", refinement.Name)
			}
			semantic := fmt.Sprintf("mask:%d", child.GetInt("mask"))
			add(1, "refinement-proposal", candidate.Name, semantic)
			add(12, "semantic-key-read", candidate.Name, semantic)
			add(12, "refinement-record-write", candidate.Name, refinement.Name)
			parents := child.GetStrings("refinedFrom")
			if len(parents) > 0 && parents[0] == candidate.Name {
				add(12, "candidate-write", candidate.Name, child.Name)
				add(12, "agenda-enqueue", child.Name, "ngRefine")
			}
		}
		add(12, "agenda-enqueue", candidate.Name, "ngEvaluate")
	}

	for mask := 0; mask < 8; mask++ {
		candidate := candidates[mask]
		for _, exampleName := range experiment.GetStrings("trainingExamples") {
			example := store.Get(exampleName)
			binding := findTrainingBinding(store, candidate.Name, exampleName)
			evidence := findTrainingEvidence(store, candidate.Name, exampleName)
			if example == nil || binding == nil || evidence == nil {
				return nil, fmt.Errorf("missing training occurrence for candidate %d example %s", mask, exampleName)
			}
			add(2, "problem-read", example.Name, "problem")
			add(2, "decision-variable-read", example.Name, "decisionVariable")
			add(2, "decision-color-read", example.Name, "decisionColor")
			add(1, "binding-proposal", candidate.Name, binding.Name)
			add(12, "binding-write", candidate.Name, binding.Name)
			if binding.GetBool("guardMatched") {
				add(8, "artifact-read", candidate.Name, "mask")
				for bit := 0; bit < 3; bit++ {
					add(8, "mask-bit-check", candidate.Name, fmt.Sprintf("bit:%d", bit))
				}
				for _, edge := range []string{"a-x", "a-y", "x-y"} {
					add(2, "edge-read", example.Name, edge)
				}
			}
			for _, resultName := range evidence.GetStrings("results") {
				result := store.Get(resultName)
				if result == nil || result.GetString("binding") != binding.Name {
					return nil, fmt.Errorf("invalid training result %s", resultName)
				}
				completionKey := fmt.Sprintf("completion:%d:%d", result.GetInt("xColor"), result.GetInt("yColor"))
				add(9, "completion-construct", binding.Name, completionKey)
				add(2, "domain-read-x", binding.Name, completionKey)
				add(2, "domain-read-y", binding.Name, completionKey)
				for _, edge := range []string{"a-x", "a-y", "x-y"} {
					add(4, "inequality-check", binding.Name, edge)
				}
				add(12, "result-write", binding.Name, result.Name)
			}
			add(12, "evidence-write", candidate.Name, evidence.Name)
			for _, resultName := range evidence.GetStrings("results") {
				add(12, "result-read", evidence.Name, resultName)
			}
			barrier := store.Get(evidence.GetString("barrier"))
			if barrier == nil || barrier.GetString("candidate") != candidate.Name || barrier.GetString("example") != example.Name {
				return nil, fmt.Errorf("missing training evidence barrier")
			}
			for _, key := range barrier.GetStrings("expectedKeys") {
				add(10, "expected-key-check", barrier.Name, key)
			}
			for _, key := range barrier.GetStrings("actualKeys") {
				add(10, "actual-key-check", barrier.Name, key)
			}
			add(10, "key-set-equality", barrier.Name, "expected-actual")
			add(10, "completion-count-check", barrier.Name, "completionCount")
			add(12, "barrier-write", candidate.Name, barrier.Name)
		}
		add(12, "agenda-enqueue", experiment.Name, "ngSelect")
	}

	selectionBarrier := findSelectionBarrier(store)
	if selectionBarrier == nil {
		return nil, fmt.Errorf("missing training selection barrier")
	}
	for attempt := 0; attempt < 2; attempt++ {
		for mask := 0; mask < 8; mask++ {
			candidate := candidates[mask]
			add(8, "selection-evidence-read", experiment.Name, candidate.Name)
			add(12, "selection-comparison", experiment.Name, candidate.Name)
		}
		for _, key := range unitIntList(selectionBarrier, "expectedKeys") {
			add(10, "expected-mask-check", selectionBarrier.Name, fmt.Sprintf("mask:%d", key))
		}
		for _, key := range unitIntList(selectionBarrier, "actualKeys") {
			add(10, "actual-mask-check", selectionBarrier.Name, fmt.Sprintf("mask:%d", key))
		}
		add(10, "mask-set-equality", selectionBarrier.Name, "expected-actual")
		add(10, "candidate-count-check", selectionBarrier.Name, "candidateCount")
		add(10, "complete-count-check", selectionBarrier.Name, "completeCount")
		add(12, "selection-barrier-write", experiment.Name, selectionBarrier.Name)
	}
	selection := store.Get(experiment.GetString("selectionUnit"))
	if selection == nil || selection.GetString("barrier") != selectionBarrier.Name {
		return nil, fmt.Errorf("missing training selection")
	}
	add(12, "tie-record-write", selection.Name, "exactCandidates")
	add(12, "selection-record-write", experiment.Name, selection.Name)
	add(12, "agenda-enqueue", experiment.Name, "ngPromote")

	artifact := store.Get(experiment.GetString("artifactUnit"))
	if artifact == nil {
		return nil, fmt.Errorf("missing training artifact")
	}
	promotionBarrier := store.Get(artifact.GetString("promotionBarrier"))
	if promotionBarrier == nil {
		return nil, fmt.Errorf("missing promotion barrier")
	}
	for _, caseName := range experiment.GetStrings("promotionCases") {
		proof := findPromotionProof(store, caseName)
		if proof == nil {
			return nil, fmt.Errorf("missing promotion proof for %s", caseName)
		}
		add(9, "promotion-completion", caseName, "only-only")
		add(12, "promotion-proof-write", caseName, proof.Name)
	}
	for _, caseName := range promotionBarrier.GetStrings("expectedKeys") {
		add(10, "expected-promotion-check", promotionBarrier.Name, caseName)
	}
	for _, caseName := range promotionBarrier.GetStrings("actualKeys") {
		add(10, "actual-promotion-check", promotionBarrier.Name, caseName)
	}
	add(10, "promotion-count-check", promotionBarrier.Name, "proofCount")
	add(10, "promotion-conflict-check", promotionBarrier.Name, "conflictCount")
	add(12, "promotion-barrier-write", experiment.Name, promotionBarrier.Name)
	add(8, "artifact-freeze-write", selection.Name, artifact.Name)
	add(12, "provenance-write", artifact.Name, nogoodfixtureAuthority)
	add(12, "boundary-write", artifact.Name, promotionBarrier.Name)

	addDispatch(experiment.Name, "ngStart")
	for mask := 0; mask < 8; mask++ {
		addDispatch(candidates[mask].Name, "ngRefine")
		addDispatch(candidates[mask].Name, "ngEvaluate")
	}
	addDispatch(experiment.Name, "ngSelect")
	addDispatch(experiment.Name, "ngSelect")
	addDispatch(experiment.Name, "ngPromote")
	if len(records) != len(run.MeterRecords) {
		return nil, fmt.Errorf("training reconstructed %d meter records, observed %d", len(records), len(run.MeterRecords))
	}
	return records, nil
}

func soleExample(store *unit.Store, kind string) (*unit.Unit, error) {
	var found *unit.Unit
	for _, name := range store.Examples(kind) {
		if name == kind {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple %s occurrences", kind)
		}
		found = store.Get(name)
	}
	if found == nil {
		return nil, fmt.Errorf("missing %s occurrence", kind)
	}
	return found, nil
}

func findTrainingBinding(store *unit.Store, candidate, example string) *unit.Unit {
	for _, name := range store.Examples("NogoodBinding") {
		unit := store.Get(name)
		if unit != nil && unit.GetString("candidate") == candidate && unit.GetString("example") == example {
			return unit
		}
	}
	return nil
}

func findTrainingEvidence(store *unit.Store, candidate, example string) *unit.Unit {
	for _, name := range store.Examples("NogoodEvidence") {
		unit := store.Get(name)
		if unit != nil && unit.GetString("candidate") == candidate && unit.GetString("example") == example {
			return unit
		}
	}
	return nil
}

func findSelectionBarrier(store *unit.Store) *unit.Unit {
	for _, name := range store.Examples("NogoodEvidenceBarrier") {
		barrier := store.Get(name)
		if barrier != nil && len(unitIntList(barrier, "expectedKeys")) == 8 {
			return barrier
		}
	}
	return nil
}

func unitIntList(source *unit.Unit, slot string) []int {
	switch values := source.Get(slot).(type) {
	case []int:
		return values
	case []any:
		result := make([]int, 0, len(values))
		for _, value := range values {
			if number, ok := value.(int); ok {
				result = append(result, number)
			}
		}
		return result
	default:
		return nil
	}
}

func findPromotionProof(store *unit.Store, caseName string) *unit.Unit {
	for _, name := range store.Examples("NogoodPromotionProof") {
		proof := store.Get(name)
		if proof != nil && proof.GetString("case") == caseName {
			return proof
		}
	}
	return nil
}

func reconstructBridgeMeter(disposition Disposition) ([]dsl.NogoodMeterRecord, error) {
	store, request := disposition.Store, disposition.Store.Get(disposition.Request)
	if store == nil || request == nil {
		return nil, fmt.Errorf("missing bridge occurrence store/request")
	}
	problem, err := nogoods.ParseProblem([]byte(request.GetString("problem")))
	if err != nil {
		return nil, err
	}
	var records []dsl.NogoodMeterRecord
	add := func(category int, operation, subject, object, outcome string) {
		records = append(records, dsl.NogoodMeterRecord{Category: uint8(category), Operation: operation, Subject: subject, Object: object, Outcome: outcome})
	}
	addMany := func(category int, subject string, operations []string) {
		for _, operation := range operations {
			add(category, operation, subject, operation, "ok")
		}
	}
	requestName := request.Name
	addMany(2, requestName, []string{"root-domain-read"})
	addMany(3, requestName, []string{"root-propose", "root-bind"})
	addMany(5, requestName, []string{"root-delete", "root-empty-check"})
	add(12, "request-write", requestName, request.GetString("requestDigest"), "ok")
	add(12, "agenda-enqueue", requestName, "ngConsiderPrune", "ok")
	add(12, "agenda-dequeue", requestName, "ngConsiderPrune", "ok")
	add(12, "request-digest-check", requestName, request.GetString("requestDigest"), "ok")
	addMany(12, requestName+".ngConsiderPrune", engineDispatchOperations)

	for _, memoName := range store.Examples("NogoodConcreteMemo") {
		if memoName == "NogoodConcreteMemo" {
			continue
		}
		memo := store.Get(memoName)
		outcome := "miss"
		if memo.GetString("exactKey") == request.GetString("concreteMemoKey") {
			outcome = "hit"
		}
		add(12, "concrete-memo-lookup", memoName, requestName, outcome)
	}
	anchor, blocked := request.GetInt("decisionVariable"), request.GetInt("decisionColor")
	add(2, "domain-read", requestName, fmt.Sprintf("domain%d", anchor), "ok")
	rolesByVariable := map[int]*unit.Unit{}
	for _, roleName := range store.Examples("NogoodRoleCandidate") {
		if roleName != "NogoodRoleCandidate" {
			role := store.Get(roleName)
			rolesByVariable[role.GetInt("variable")] = role
		}
	}
	var orderedRoles []*unit.Unit
	if len(problem.Variables[anchor].Domain) == 2 && problem.DomainContains(anchor, blocked) {
		for variable := range problem.Variables {
			if variable == anchor {
				continue
			}
			domainSlot := fmt.Sprintf("domain%d", variable)
			add(2, "domain-size-check", requestName, domainSlot, "ok")
			add(2, "domain-membership-check", requestName, domainSlot, "ok")
			add(12, "role-visit-record", requestName, fmt.Sprintf("variable:%d", variable), "ok")
			if role := rolesByVariable[variable]; role != nil {
				add(1, "role-candidate", requestName, role.Name, "ok")
				add(12, "role-candidate-write", requestName, role.Name, "ok")
				orderedRoles = append(orderedRoles, role)
			}
		}
	}
	artifacts := store.Examples("NogoodArtifact")
	for leftIndex, leftRole := range orderedRoles {
		for rightIndex, rightRole := range orderedRoles {
			if leftIndex >= rightIndex {
				continue
			}
			pair := findPair(store, requestName, leftRole.Name, rightRole.Name)
			if pair == nil {
				return nil, fmt.Errorf("missing pair for %s/%s", leftRole.Name, rightRole.Name)
			}
			add(1, "pair-candidate", requestName, pair.Name, "ok")
			add(2, "pair-only-equality", leftRole.Name, rightRole.Name, "ok")
			add(2, "pair-escape-inequality", leftRole.Name, "escape", "ok")
			add(12, "pair-record-write", requestName, pair.Name, "ok")
			if !pair.GetBool("guardMatched") {
				continue
			}
			binding := findBinding(store, requestName, leftRole.GetInt("variable"), rightRole.GetInt("variable"))
			if binding == nil {
				return nil, fmt.Errorf("missing binding for pair %s", pair.Name)
			}
			for _, artifactName := range artifacts {
				if artifactName == "NogoodArtifact" {
					continue
				}
				artifact := store.Get(artifactName)
				for _, operation := range []string{"artifact-read", "mask-bit-0", "mask-bit-1", "mask-bit-2", "authority-read", "frozen-read", "schema-read", "guard-version-read", "artifact-digest-read", "evidence-digest-read"} {
					add(8, operation, artifactName, binding.Name, "ok")
				}
				for _, edge := range []string{"edge-a-x", "edge-a-y", "edge-x-y"} {
					add(2, "artifact-edge-read", binding.Name, edge, "ok")
				}
				add(12, "artifact-match-read", binding.Name, artifactName, "ok")
				add(12, "artifact-match-record", binding.Name, artifactName, "ok")
				completion := findCompletion(store, binding.Name)
				if completion == nil || !artifact.GetBool("authoritative") {
					continue
				}
				add(9, "completion-construct", binding.Name, completion.Name, "ok")
				add(2, "completion-domain-read", binding.Name, "x", "ok")
				add(2, "completion-domain-read", binding.Name, "y", "ok")
				for _, comparison := range []string{"a-x", "a-y", "x-y"} {
					add(4, "completion-inequality", completion.Name, comparison, "ok")
				}
				add(12, "completion-result-write", binding.Name, completion.Name, "ok")
				certificate := store.Get(disposition.Certificate)
				barrier := store.Get(disposition.Barrier)
				if certificate == nil || barrier == nil || certificate.GetString("completion") != completion.Name {
					return nil, fmt.Errorf("missing completion certificate/barrier")
				}
				add(10, "certificate-record", completion.Name, certificate.Name, "ok")
				for _, predicate := range barrier.GetStrings("predicateKeys") {
					add(10, "barrier-predicate-check", certificate.Name, predicate, "ok")
				}
				for _, operation := range []string{"certificate-index-write", "agreement-result-read", "agreement-record-write", "expected-key-set-write", "actual-key-set-write", "sealed-barrier-write"} {
					add(12, operation, certificate.Name, barrier.Name, "ok")
				}
			}
		}
	}
	dispositionUnit := store.Get(request.GetString("dispositionUnit"))
	if dispositionUnit == nil {
		return nil, fmt.Errorf("missing disposition unit")
	}
	dispositionOperations := []string{"disposition-write", "request-digest-check", "target-digest-check", "decision-digest-check"}
	if disposition.Status == "propose-prune" {
		dispositionOperations = append(dispositionOperations, "assignment-digest-check", "artifact-digest-check")
	}
	for _, operation := range dispositionOperations {
		add(12, operation, requestName, dispositionUnit.Name, "ok")
	}
	adapterPrefix, adapterObject := "adapter-resume-check-", disposition.Status
	if disposition.Status == "propose-prune" {
		adapterPrefix, adapterObject = "adapter-proposal-check-", disposition.Proposal
	} else if disposition.Status == "concrete-prune" {
		adapterPrefix, adapterObject = "adapter-concrete-check-", disposition.Memo
	}
	for index := 1; index <= 6; index++ {
		add(10, fmt.Sprintf("%s%d", adapterPrefix, index), requestName, adapterObject, "ok")
	}
	return records, nil
}

func findPair(store *unit.Store, request, leftRole, rightRole string) *unit.Unit {
	for _, name := range store.Examples("NogoodPairProposal") {
		candidate := store.Get(name)
		if candidate != nil && candidate.GetString("request") == request && candidate.GetString("leftRole") == leftRole && candidate.GetString("rightRole") == rightRole {
			return candidate
		}
	}
	return nil
}

func findBinding(store *unit.Store, request string, x, y int) *unit.Unit {
	for _, name := range store.Examples("NogoodBinding") {
		candidate := store.Get(name)
		if candidate != nil && candidate.GetString("request") == request && candidate.GetInt("x") == x && candidate.GetInt("y") == y {
			return candidate
		}
	}
	return nil
}

func findCompletion(store *unit.Store, binding string) *unit.Unit {
	for _, name := range store.Examples("NogoodCompletion") {
		candidate := store.Get(name)
		if candidate != nil && candidate.GetString("binding") == binding {
			return candidate
		}
	}
	return nil
}

func compareMeterMultiset(actual, expected []dsl.NogoodMeterRecord) error {
	encode := func(records []dsl.NogoodMeterRecord) []string {
		encoded := make([]string, len(records))
		for index, record := range records {
			data, _ := json.Marshal(record)
			encoded[index] = string(data)
		}
		sort.Strings(encoded)
		return encoded
	}
	actualEncoded, expectedEncoded := encode(actual), encode(expected)
	if !slices.Equal(actualEncoded, expectedEncoded) {
		for index := 0; index < min(len(actualEncoded), len(expectedEncoded)); index++ {
			if actualEncoded[index] != expectedEncoded[index] {
				return fmt.Errorf("meter tuple mismatch actual=%s expected=%s", actualEncoded[index], expectedEncoded[index])
			}
		}
		return fmt.Errorf("meter tuple count %d, expected %d", len(actualEncoded), len(expectedEncoded))
	}
	return nil
}
