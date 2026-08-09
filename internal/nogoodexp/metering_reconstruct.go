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
