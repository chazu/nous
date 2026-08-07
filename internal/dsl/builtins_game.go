package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
	gamevocab "github.com/chazu/nous/internal/vocab/game"
)

const gameExperimentCategory = "IteratedGameExperiment"

func init() {
	registerVocabularyWords("game", map[string]builtinFn{
		"game-strategy-valid?":      bGameStrategyValid,
		"game-experiment-valid?":    bGameExperimentValid,
		"game-generate-experiment":  bGameGenerateExperiment,
		"game-evaluate-candidate":   bGameEvaluateCandidate,
		"game-ready-to-finalize?":   bGameReadyToFinalize,
		"game-finalize-experiment":  bGameFinalizeExperiment,
		"game-experiment-complete?": bGameExperimentComplete,
	})
}

type gameCaseBinding struct {
	Name   string
	Digest string
	Case   gamevocab.Case
}

type gameDescriptor struct {
	name, profileKey, candidateCategory, opponentCategory, caseCategory string
	resultCategory, observationCategory, evidenceCategory               string
	selectionCategory, schemaCategory, conjectureCategory               string
	generationSlot, evaluationSlot, finalizationSlot                    string
	generationPriority, evaluationPriority, finalizationPriority        int
	profile                                                             gamevocab.Profile
	cases                                                               []gameCaseBinding
}

func bGameStrategyValid(vm *VM) error {
	_, ok := gameStrategyValue(vm.pop())
	vm.push(BoolVal(ok))
	return nil
}

func gameStrategyValue(value Value) (gamevocab.Strategy, bool) {
	items := value.AsList()
	records := make([]string, len(items))
	for index, item := range items {
		if item.Kind() != VString {
			return gamevocab.Strategy{}, false
		}
		records[index] = item.AsString()
	}
	strategy, err := gamevocab.ParseStrategy(records)
	return strategy, err == nil
}

func gameIntSlice(value any) ([]int, bool) {
	switch values := value.(type) {
	case []int:
		return append([]int(nil), values...), true
	case []any:
		if len(values) == 0 {
			return []int{}, true
		}
		return nil, false
	case nil:
		return []int{}, true
	default:
		return nil, false
	}
}

func directGameMember(u *unit.Unit, category string) bool {
	if u == nil || u.Name == category {
		return false
	}
	for _, parent := range u.GetStrings("isA") {
		if parent == category {
			return true
		}
	}
	return false
}

func readGameDescriptor(vm *VM, name string) (gameDescriptor, bool) {
	u := vm.Store.Get(name)
	if u == nil || name == gameExperimentCategory || !vm.Store.IsA(name, gameExperimentCategory) {
		return gameDescriptor{}, false
	}
	d := gameDescriptor{
		name: name, profileKey: u.GetString("profileKey"),
		candidateCategory: u.GetString("candidateCategory"), opponentCategory: u.GetString("opponentCategory"), caseCategory: u.GetString("caseCategory"),
		resultCategory: u.GetString("resultCategory"), observationCategory: u.GetString("observationCategory"), evidenceCategory: u.GetString("evidenceCategory"),
		selectionCategory: u.GetString("selectionCategory"), schemaCategory: u.GetString("schemaCategory"), conjectureCategory: u.GetString("conjectureCategory"),
		generationSlot: u.GetString("generationTaskSlot"), evaluationSlot: u.GetString("evaluationTaskSlot"), finalizationSlot: u.GetString("finalizationTaskSlot"),
		generationPriority: u.GetInt("generationPriority"), evaluationPriority: u.GetInt("evaluationPriority"), finalizationPriority: u.GetInt("finalizationPriority"),
	}
	categoryNames := []string{d.candidateCategory, d.opponentCategory, d.caseCategory, d.resultCategory, d.observationCategory, d.evidenceCategory, d.selectionCategory, d.schemaCategory, d.conjectureCategory}
	seenCategories := map[string]bool{}
	for _, category := range categoryNames {
		if category == "" || category == "Anything" || vm.Store.Get(category) == nil || seenCategories[category] {
			return gameDescriptor{}, false
		}
		seenCategories[category] = true
	}
	for first, a := range categoryNames {
		for second, b := range categoryNames {
			if first != second && vm.Store.IsA(a, b) {
				return gameDescriptor{}, false
			}
		}
	}
	if d.generationSlot == "" || d.evaluationSlot == "" || d.finalizationSlot == "" || d.generationSlot == d.evaluationSlot || d.generationSlot == d.finalizationSlot || d.evaluationSlot == d.finalizationSlot ||
		!(d.generationPriority > d.evaluationPriority && d.evaluationPriority > d.finalizationPriority) || u.GetInt("candidateCap") != gamevocab.StrategyCount || u.GetInt("caseCap") < len(u.GetStrings("evaluationCases")) || u.GetInt("caseCap") > gamevocab.MaxCases {
		return gameDescriptor{}, false
	}
	d.profile = gamevocab.Profile{
		ExperimentKey: u.GetString("experimentKey"), ComparisonMethod: u.GetString("comparisonMethod"),
		Payoffs: gamevocab.Payoffs{Temptation: u.GetInt("temptation"), Reward: u.GetInt("reward"), Punishment: u.GetInt("punishment"), Sucker: u.GetInt("sucker")},
		Rounds:  u.GetInt("rounds"),
	}
	caseNames := u.GetStrings("evaluationCases")
	seenNames, seenDigests := map[string]bool{}, map[string]bool{}
	strategySlot := u.GetString("opponentStrategySlot")
	if strategySlot == "" {
		return gameDescriptor{}, false
	}
	for _, caseName := range caseNames {
		caseUnit := vm.Store.Get(caseName)
		if seenNames[caseName] || !directGameMember(caseUnit, d.caseCategory) {
			return gameDescriptor{}, false
		}
		seenNames[caseName] = true
		candidateFlips, candidateOK := gameIntSlice(caseUnit.Get("candidateFlips"))
		opponentFlips, opponentOK := gameIntSlice(caseUnit.Get("opponentFlips"))
		if !candidateOK || !opponentOK {
			return gameDescriptor{}, false
		}
		binding := gameCaseBinding{Name: caseName}
		binding.Case = gamevocab.Case{Axis: gamevocab.Axis(caseUnit.GetString("axis")), Self: caseUnit.GetBool("self"), CandidateFlip: candidateFlips, OpponentFlip: opponentFlips}
		if !binding.Case.Self {
			opponent := vm.Store.Get(caseUnit.GetString("opponent"))
			if !directGameMember(opponent, d.opponentCategory) {
				return gameDescriptor{}, false
			}
			strategy, err := gamevocab.ParseStrategy(opponent.GetStrings(strategySlot))
			if err != nil {
				return gameDescriptor{}, false
			}
			binding.Case.Opponent = strategy
		}
		d.profile.Cases = append(d.profile.Cases, binding.Case)
		d.cases = append(d.cases, binding)
	}
	if !d.profile.Valid() {
		return gameDescriptor{}, false
	}
	for index := range d.cases {
		digest, err := d.cases[index].Case.Digest(d.profile.Rounds)
		if err != nil || seenDigests[digest] {
			return gameDescriptor{}, false
		}
		seenDigests[digest] = true
		d.cases[index].Digest = digest
	}
	key, err := d.profile.Key()
	if err != nil || key != d.profileKey {
		return gameDescriptor{}, false
	}
	return d, true
}

func bGameExperimentValid(vm *VM) error {
	_, ok := readGameDescriptor(vm, vm.pop().AsString())
	vm.push(BoolVal(ok))
	return nil
}

func gameHash(values ...any) string {
	encoded, _ := json.Marshal(values)
	sum := sha256.Sum256(encoded)
	return "sha256:v1:" + hex.EncodeToString(sum[:])
}

func gameFreshName(store *unit.Store, base string) string {
	if !store.Has(base) {
		return base
	}
	for suffix := 1; ; suffix++ {
		candidate := fmt.Sprintf("%s-collision-%d", base, suffix)
		if !store.Has(candidate) {
			return candidate
		}
	}
}

func gameAttributed(store *unit.Store, d gameDescriptor, kind, semanticKey string) []string {
	var names []string
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("gameExperiment") == d.name && u.GetString("profileKey") == d.profileKey && u.GetString("artifactKind") == kind && u.GetString("semanticKey") == semanticKey {
			names = append(names, name)
		}
	}
	return names
}

func gamePutArtifact(vm *VM, d gameDescriptor, kind, semanticKey, base, category string, slots map[string]any) (*unit.Unit, bool) {
	existing := gameAttributed(vm.Store, d, kind, semanticKey)
	if len(existing) > 1 {
		return nil, false
	}
	common := map[string]any{
		"gameExperiment": d.name, "profileKey": d.profileKey, "artifactKind": kind, "semanticKey": semanticKey,
	}
	for key, value := range slots {
		common[key] = value
	}
	if len(existing) == 1 {
		u := vm.Store.Get(existing[0])
		for key, value := range common {
			if !reflect.DeepEqual(u.Get(key), value) {
				return nil, false
			}
		}
		return u, true
	}
	name := gameFreshName(vm.Store, base)
	u := unit.New(name)
	u.Set("isA", []string{category, "Anything"})
	for key, value := range common {
		u.Set(key, value)
	}
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, name)
	return u, true
}

func bGameGenerateExperiment(vm *VM) error {
	name := vm.pop().AsString()
	d, ok := readGameDescriptor(vm, name)
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	experiment := vm.Store.Get(name)
	if experiment.GetBool("generationComplete") {
		vm.push(BoolVal(gameGeneratedState(vm, d)))
		return nil
	}
	var candidates []string
	for _, strategy := range gamevocab.AllStrategies() {
		code := strategy.Code()
		semanticKey := fmt.Sprintf("candidate:%02d", code)
		u, allocated := gamePutArtifact(vm, d, "candidate", semanticKey, "GameStrategy-"+strategy.Actions(), d.candidateCategory, map[string]any{
			"data": strategy.Records(), "semanticCode": code, "actions": strategy.Actions(),
			"decisionKey": gameHash(d.profileKey, "memory-one-policy", code), "creationWorth": 500,
			"generationHeuristic": "H-EnumerateMemoryOneStrategies",
		})
		if !allocated {
			vm.push(BoolVal(false))
			return nil
		}
		u.SetWorth(500)
		candidates = append(candidates, u.Name)
		vm.Ag.Push(&agenda.Task{Priority: d.evaluationPriority, UnitName: u.Name, SlotName: d.evaluationSlot, Reasons: []string{"Evaluate generated memory-one strategy"}})
	}
	experiment.Set("candidateUnits", candidates)
	experiment.Set("expectedCandidateCount", gamevocab.StrategyCount)
	experiment.Set("evaluatedCandidateCount", 0)
	experiment.Set("generationComplete", true)
	vm.push(BoolVal(true))
	return nil
}

func gameGeneratedState(vm *VM, d gameDescriptor) bool {
	experiment := vm.Store.Get(d.name)
	candidates := experiment.GetStrings("candidateUnits")
	if !experiment.GetBool("generationComplete") || experiment.GetInt("expectedCandidateCount") != gamevocab.StrategyCount || len(candidates) != gamevocab.StrategyCount {
		return false
	}
	for code, name := range candidates {
		u := vm.Store.Get(name)
		if u == nil || !directGameMember(u, d.candidateCategory) || u.GetString("gameExperiment") != d.name || u.GetString("profileKey") != d.profileKey || u.GetString("artifactKind") != "candidate" || u.GetString("semanticKey") != fmt.Sprintf("candidate:%02d", code) || u.GetInt("semanticCode") != code || u.GetInt("creationWorth") != 500 || u.GetString("generationHeuristic") != "H-EnumerateMemoryOneStrategies" || u.GetString("decisionKey") != gameHash(d.profileKey, "memory-one-policy", code) {
			return false
		}
		strategy, err := gamevocab.ParseStrategy(u.GetStrings("data"))
		if err != nil || strategy.Code() != code || u.GetString("actions") != strategy.Actions() || len(gameAttributed(vm.Store, d, "candidate", fmt.Sprintf("candidate:%02d", code))) != 1 {
			return false
		}
	}
	return true
}

type gameEvaluation struct {
	objectives                    gamevocab.Objectives
	resultNames, observationNames []string
	caseNames                     []string
	trainingCoop, mutualCoop      int
	behaviorSignature             string
	applications                  []map[string]any
}

type gameSignatureEntry struct {
	CaseDigest string `json:"case"`
	Length     int    `json:"length"`
	Trace      string `json:"trace"`
}

func evaluateGameCandidate(vm *VM, d gameDescriptor, candidate *unit.Unit, create bool) (gameEvaluation, bool) {
	strategy, err := gamevocab.ParseStrategy(candidate.GetStrings("data"))
	if err != nil || strategy.Code() != candidate.GetInt("semanticCode") {
		return gameEvaluation{}, false
	}
	evaluation := gameEvaluation{objectives: gamevocab.Objectives{TrainingWorst: gamevocab.MaxRounds * gamevocab.MaxPayoff}}
	var signature []gameSignatureEntry
	for _, binding := range d.cases {
		opponent := binding.Case.Opponent
		if binding.Case.Self {
			opponent = strategy
		}
		match, playErr := gamevocab.Play(strategy, opponent, d.profile.Payoffs, gamevocab.MatchSpec{Rounds: d.profile.Rounds, CandidateFlip: binding.Case.CandidateFlip, OpponentFlip: binding.Case.OpponentFlip})
		if playErr != nil {
			return gameEvaluation{}, false
		}
		evaluation.caseNames = append(evaluation.caseNames, binding.Name)
		switch binding.Case.Axis {
		case gamevocab.Training:
			evaluation.objectives.TrainingTotal += match.CandidateScore
			if match.CandidateScore < evaluation.objectives.TrainingWorst {
				evaluation.objectives.TrainingWorst = match.CandidateScore
			}
			evaluation.trainingCoop += match.CandidateCooperations
			evaluation.mutualCoop += match.MutualCooperations
			signature = append(signature, gameSignatureEntry{CaseDigest: binding.Digest, Length: len(match.CandidateTrace), Trace: match.CandidateTrace})
		case gamevocab.Self:
			evaluation.objectives.SelfScore = match.CandidateScore
		case gamevocab.Perturbation:
			evaluation.objectives.PerturbScore = match.CandidateScore
		}
		if !create {
			continue
		}
		code := strategy.Code()
		resultKey := fmt.Sprintf("match:%02d:%s", code, binding.Digest)
		result, resultOK := gamePutArtifact(vm, d, "result", resultKey, fmt.Sprintf("GameMatch-%s-%s", strategy.Actions(), binding.Digest[10:22]), d.resultCategory, map[string]any{
			"candidate": candidate.Name, "candidateCode": code, "caseUnit": binding.Name, "caseDigest": binding.Digest,
			"rounds": match.Rounds, "candidateScore": match.CandidateScore, "opponentScore": match.OpponentScore,
			"candidateCooperations": match.CandidateCooperations, "opponentCooperations": match.OpponentCooperations, "mutualCooperations": match.MutualCooperations,
			"candidateTrace": match.CandidateTrace, "opponentTrace": match.OpponentTrace,
		})
		if !resultOK {
			return gameEvaluation{}, false
		}
		result.SetWorth(500)
		observationKey := fmt.Sprintf("observation:%02d:%s", code, binding.Digest)
		observation, observationOK := gamePutArtifact(vm, d, "observation", observationKey, fmt.Sprintf("GameObservation-%s-%s", strategy.Actions(), binding.Digest[10:22]), d.observationCategory, map[string]any{
			"candidate": candidate.Name, "candidateCode": code, "caseUnit": binding.Name, "caseDigest": binding.Digest, "resultUnit": result.Name,
			"candidateTrace": match.CandidateTrace, "status": "complete",
		})
		if !observationOK {
			return gameEvaluation{}, false
		}
		observation.SetWorth(450)
		evaluation.resultNames = append(evaluation.resultNames, result.Name)
		evaluation.observationNames = append(evaluation.observationNames, observation.Name)
		evaluation.applications = append(evaluation.applications, map[string]any{"target": candidate.Name, "result": true, "args": []string{binding.Name}, "output": result.Name, "direct": true})
	}
	encoded, _ := json.Marshal(signature)
	evaluation.behaviorSignature = string(encoded)
	return evaluation, true
}

func bGameEvaluateCandidate(vm *VM) error {
	candidateName := vm.pop().AsString()
	candidate := vm.Store.Get(candidateName)
	if candidate == nil {
		vm.push(BoolVal(false))
		return nil
	}
	d, ok := readGameDescriptor(vm, candidate.GetString("gameExperiment"))
	if !ok || !gameGeneratedState(vm, d) || !directGameMember(candidate, d.candidateCategory) {
		vm.push(BoolVal(false))
		return nil
	}
	if candidate.GetBool("evaluatedGameProfile") {
		vm.push(BoolVal(gameCandidateEvidenceValid(vm, d, candidate)))
		return nil
	}
	evaluation, evaluated := evaluateGameCandidate(vm, d, candidate, true)
	if !evaluated {
		candidate.SetWorth(300)
		vm.push(BoolVal(false))
		return nil
	}
	code := candidate.GetInt("semanticCode")
	evidenceKey := fmt.Sprintf("evidence:%02d", code)
	evidence, evidenceOK := gamePutArtifact(vm, d, "evidence", evidenceKey, "GameEvidence-"+candidate.GetString("actions"), d.evidenceCategory, map[string]any{
		"candidate": candidate.Name, "candidateCode": code, "caseUnits": evaluation.caseNames, "resultUnits": evaluation.resultNames, "observationUnits": evaluation.observationNames,
		"evaluatedCount": len(d.cases), "invalidCount": 0,
		"trainingTotal": evaluation.objectives.TrainingTotal, "trainingWorst": evaluation.objectives.TrainingWorst, "selfScore": evaluation.objectives.SelfScore, "perturbationScore": evaluation.objectives.PerturbScore,
		"trainingCooperations": evaluation.trainingCoop, "trainingMutualCooperations": evaluation.mutualCoop, "behaviorSignature": evaluation.behaviorSignature,
		"comparisonMethod": d.profile.ComparisonMethod,
	})
	if !evidenceOK {
		candidate.SetWorth(300)
		vm.push(BoolVal(false))
		return nil
	}
	evidence.SetWorth(500)
	candidate.Set("evidenceUnit", evidence.Name)
	candidate.Set("trainingTotal", evaluation.objectives.TrainingTotal)
	candidate.Set("trainingWorst", evaluation.objectives.TrainingWorst)
	candidate.Set("selfScore", evaluation.objectives.SelfScore)
	candidate.Set("perturbationScore", evaluation.objectives.PerturbScore)
	candidate.Set("trainingCooperations", evaluation.trainingCoop)
	candidate.Set("trainingMutualCooperations", evaluation.mutualCoop)
	candidate.Set("behaviorSignature", evaluation.behaviorSignature)
	candidate.Set("applics", evaluation.applications)
	candidate.Set("evaluatedGameProfile", true)
	candidate.SetWorth(500)
	experiment := vm.Store.Get(d.name)
	count := experiment.GetInt("evaluatedCandidateCount") + 1
	experiment.Set("evaluatedCandidateCount", count)
	if count == gamevocab.StrategyCount && !experiment.GetBool("finalizationScheduled") {
		experiment.Set("finalizationScheduled", true)
		vm.Ag.Push(&agenda.Task{Priority: d.finalizationPriority, UnitName: d.name, SlotName: d.finalizationSlot, Reasons: []string{"Select game strategy Pareto frontier"}})
	}
	vm.push(BoolVal(true))
	return nil
}

func gameCandidateObjectives(candidate *unit.Unit) gamevocab.Objectives {
	return gamevocab.Objectives{TrainingTotal: candidate.GetInt("trainingTotal"), TrainingWorst: candidate.GetInt("trainingWorst"), SelfScore: candidate.GetInt("selfScore"), PerturbScore: candidate.GetInt("perturbationScore")}
}

func gameCandidateEvidenceValid(vm *VM, d gameDescriptor, candidate *unit.Unit) bool {
	if !candidate.GetBool("evaluatedGameProfile") || candidate.GetString("profileKey") != d.profileKey {
		return false
	}
	expected, ok := evaluateGameCandidate(vm, d, candidate, false)
	if !ok || gameCandidateObjectives(candidate) != expected.objectives || candidate.GetInt("trainingCooperations") != expected.trainingCoop || candidate.GetInt("trainingMutualCooperations") != expected.mutualCoop || candidate.GetString("behaviorSignature") != expected.behaviorSignature {
		return false
	}
	code := candidate.GetInt("semanticCode")
	evidenceNames := gameAttributed(vm.Store, d, "evidence", fmt.Sprintf("evidence:%02d", code))
	if len(evidenceNames) != 1 || candidate.GetString("evidenceUnit") != evidenceNames[0] {
		return false
	}
	evidence := vm.Store.Get(evidenceNames[0])
	if !directGameMember(evidence, d.evidenceCategory) || evidence.Worth() != 500 || evidence.GetString("candidate") != candidate.Name || evidence.GetInt("candidateCode") != code || evidence.GetInt("evaluatedCount") != len(d.cases) || evidence.GetInt("invalidCount") != 0 || gameCandidateObjectives(evidence) != expected.objectives || evidence.GetString("behaviorSignature") != expected.behaviorSignature || evidence.GetInt("trainingCooperations") != expected.trainingCoop || evidence.GetInt("trainingMutualCooperations") != expected.mutualCoop || evidence.GetString("comparisonMethod") != d.profile.ComparisonMethod {
		return false
	}
	resultNames, observationNames := evidence.GetStrings("resultUnits"), evidence.GetStrings("observationUnits")
	if len(resultNames) != len(d.cases) || len(observationNames) != len(d.cases) || !reflect.DeepEqual(evidence.GetStrings("caseUnits"), caseBindingNames(d.cases)) {
		return false
	}
	strategy, _ := gamevocab.ParseStrategy(candidate.GetStrings("data"))
	for index, binding := range d.cases {
		opponent := binding.Case.Opponent
		if binding.Case.Self {
			opponent = strategy
		}
		match, err := gamevocab.Play(strategy, opponent, d.profile.Payoffs, gamevocab.MatchSpec{Rounds: d.profile.Rounds, CandidateFlip: binding.Case.CandidateFlip, OpponentFlip: binding.Case.OpponentFlip})
		if err != nil {
			return false
		}
		resultKey := fmt.Sprintf("match:%02d:%s", code, binding.Digest)
		observationKey := fmt.Sprintf("observation:%02d:%s", code, binding.Digest)
		if len(gameAttributed(vm.Store, d, "result", resultKey)) != 1 || len(gameAttributed(vm.Store, d, "observation", observationKey)) != 1 || resultNames[index] != gameAttributed(vm.Store, d, "result", resultKey)[0] || observationNames[index] != gameAttributed(vm.Store, d, "observation", observationKey)[0] {
			return false
		}
		result := vm.Store.Get(resultNames[index])
		observation := vm.Store.Get(observationNames[index])
		if !directGameMember(result, d.resultCategory) || result.Worth() != 500 || result.GetString("candidate") != candidate.Name || result.GetInt("candidateCode") != code || result.GetString("caseUnit") != binding.Name || result.GetString("caseDigest") != binding.Digest || result.GetInt("rounds") != match.Rounds || result.GetInt("candidateScore") != match.CandidateScore || result.GetInt("opponentScore") != match.OpponentScore || result.GetInt("candidateCooperations") != match.CandidateCooperations || result.GetInt("opponentCooperations") != match.OpponentCooperations || result.GetInt("mutualCooperations") != match.MutualCooperations || result.GetString("candidateTrace") != match.CandidateTrace || result.GetString("opponentTrace") != match.OpponentTrace ||
			!directGameMember(observation, d.observationCategory) || observation.Worth() != 450 || observation.GetString("candidate") != candidate.Name || observation.GetInt("candidateCode") != code || observation.GetString("caseUnit") != binding.Name || observation.GetString("caseDigest") != binding.Digest || observation.GetString("resultUnit") != result.Name || observation.GetString("candidateTrace") != match.CandidateTrace || observation.GetString("status") != "complete" {
			return false
		}
	}
	applications, ok := candidate.Get("applics").([]map[string]any)
	if !ok || len(applications) != len(d.cases) {
		return false
	}
	for index, application := range applications {
		if application["target"] != candidate.Name || application["result"] != true || application["output"] != resultNames[index] || application["direct"] != true || !reflect.DeepEqual(application["args"], []string{d.cases[index].Name}) {
			return false
		}
	}
	return true
}

func caseBindingNames(bindings []gameCaseBinding) []string {
	names := make([]string, len(bindings))
	for index, binding := range bindings {
		names[index] = binding.Name
	}
	return names
}

func gamePreselectionValid(vm *VM, d gameDescriptor) bool {
	if !gameGeneratedState(vm, d) {
		return false
	}
	experiment := vm.Store.Get(d.name)
	if experiment.GetInt("evaluatedCandidateCount") != gamevocab.StrategyCount {
		return false
	}
	kindCounts := map[string]int{}
	knownKinds := map[string]bool{"candidate": true, "result": true, "observation": true, "evidence": true, "selection": true, "schema": true, "conjecture": true}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("gameExperiment") == d.name && u.GetString("profileKey") == d.profileKey {
			kind := u.GetString("artifactKind")
			if !knownKinds[kind] {
				return false
			}
			kindCounts[kind]++
		}
	}
	if kindCounts["candidate"] != gamevocab.StrategyCount || kindCounts["evidence"] != gamevocab.StrategyCount || kindCounts["result"] != gamevocab.StrategyCount*len(d.cases) || kindCounts["observation"] != gamevocab.StrategyCount*len(d.cases) {
		return false
	}
	for _, name := range experiment.GetStrings("candidateUnits") {
		if !gameCandidateEvidenceValid(vm, d, vm.Store.Get(name)) {
			return false
		}
	}
	return true
}

func bGameReadyToFinalize(vm *VM) error {
	name := vm.pop().AsString()
	d, ok := readGameDescriptor(vm, name)
	vm.push(BoolVal(ok && !vm.Store.Get(name).GetBool("finalizationComplete") && gamePreselectionValid(vm, d)))
	return nil
}

func gameClassRecords(candidates []*unit.Unit, key func(*unit.Unit) string, prefix string) ([]string, map[int]string) {
	groups := map[string][]int{}
	for _, candidate := range candidates {
		groups[key(candidate)] = append(groups[key(candidate)], candidate.GetInt("semanticCode"))
	}
	type group struct {
		key     string
		members []int
	}
	var ordered []group
	for groupKey, members := range groups {
		sort.Ints(members)
		ordered = append(ordered, group{groupKey, members})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].members[0] < ordered[j].members[0] })
	records := make([]string, 0, len(ordered))
	classes := map[int]string{}
	for index, item := range ordered {
		name := fmt.Sprintf("%s%03d", prefix, index+1)
		parts := make([]string, len(item.members))
		for memberIndex, code := range item.members {
			parts[memberIndex] = fmt.Sprintf("%02d", code)
			classes[code] = name
		}
		records = append(records, name+":"+strings.Join(parts, ","))
	}
	return records, classes
}

func gameSelection(vm *VM, d gameDescriptor) ([]*unit.Unit, map[int]gamevocab.Objectives, []int, []int, []string, map[int]string, []string, map[int]string, []string, bool) {
	if !gamePreselectionValid(vm, d) {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, false
	}
	experiment := vm.Store.Get(d.name)
	candidates := make([]*unit.Unit, 0, gamevocab.StrategyCount)
	objectives := map[int]gamevocab.Objectives{}
	for _, name := range experiment.GetStrings("candidateUnits") {
		candidate := vm.Store.Get(name)
		candidates = append(candidates, candidate)
		objectives[candidate.GetInt("semanticCode")] = gameCandidateObjectives(candidate)
	}
	frontier := gamevocab.Frontier(objectives)
	scalar := gamevocab.ScalarLeaders(objectives)
	behaviorRecords, behaviorClasses := gameClassRecords(candidates, func(u *unit.Unit) string { return u.GetString("behaviorSignature") }, "B")
	objectiveRecords, objectiveClasses := gameClassRecords(candidates, func(u *unit.Unit) string {
		o := gameCandidateObjectives(u)
		return fmt.Sprintf("%d/%d/%d/%d", o.TrainingTotal, o.TrainingWorst, o.SelfScore, o.PerturbScore)
	}, "O")
	var dominance []string
	for _, first := range candidates {
		for _, second := range candidates {
			dominates := objectives[first.GetInt("semanticCode")].Dominates(objectives[second.GetInt("semanticCode")])
			dominance = append(dominance, fmt.Sprintf("%02d>%02d:%t", first.GetInt("semanticCode"), second.GetInt("semanticCode"), dominates))
		}
	}
	return candidates, objectives, frontier, scalar, behaviorRecords, behaviorClasses, objectiveRecords, objectiveClasses, dominance, true
}

func namesForCodes(candidates []*unit.Unit, codes []int) []string {
	names := make([]string, 0, len(codes))
	byCode := map[int]string{}
	for _, candidate := range candidates {
		byCode[candidate.GetInt("semanticCode")] = candidate.Name
	}
	for _, code := range codes {
		names = append(names, byCode[code])
	}
	return names
}

func bGameFinalizeExperiment(vm *VM) error {
	name := vm.pop().AsString()
	d, ok := readGameDescriptor(vm, name)
	if !ok || vm.Store.Get(name).GetBool("finalizationComplete") {
		vm.push(BoolVal(false))
		return nil
	}
	candidates, _, frontier, scalar, behaviorRecords, behaviorClasses, objectiveRecords, objectiveClasses, dominance, selected := gameSelection(vm, d)
	if !selected {
		vm.push(BoolVal(false))
		return nil
	}
	frontierNames := namesForCodes(candidates, frontier)
	scalarNames := namesForCodes(candidates, scalar)
	selectionKey := "selection:" + d.profileKey
	selection, selectionOK := gamePutArtifact(vm, d, "selection", selectionKey, "GameSelection-"+d.profileKey[10:22], d.selectionCategory, map[string]any{
		"candidateUnits": namesForCodes(candidates, allGameCodes()), "frontierUnits": frontierNames, "frontierCodes": frontier, "scalarLeaderUnits": scalarNames, "scalarLeaderCodes": scalar,
		"objectiveNames":  []string{"training-total", "training-worst", "self-score", "perturbation-score"},
		"behaviorClasses": behaviorRecords, "behaviorClassCount": len(behaviorRecords), "objectiveClasses": objectiveRecords, "objectiveClassCount": len(objectiveRecords), "dominanceDecisions": dominance,
		"comparisonMethod": d.profile.ComparisonMethod,
	})
	if !selectionOK {
		vm.push(BoolVal(false))
		return nil
	}
	selection.SetWorth(500)
	frontierSet := map[int]bool{}
	for _, code := range frontier {
		frontierSet[code] = true
	}
	for _, candidate := range candidates {
		code := candidate.GetInt("semanticCode")
		candidate.Set("behaviorClass", behaviorClasses[code])
		candidate.Set("objectiveClass", objectiveClasses[code])
		candidate.Set("paretoFrontier", frontierSet[code])
		if !frontierSet[code] {
			candidate.SetWorth(500)
			continue
		}
		candidate.SetWorth(800)
		schemaKey := fmt.Sprintf("schema:%02d", code)
		schema, schemaOK := gamePutArtifact(vm, d, "schema", schemaKey, "GameSchema-"+candidate.GetString("actions"), d.schemaCategory, map[string]any{
			"candidate": candidate.Name, "candidateCode": code, "evidenceUnit": candidate.GetString("evidenceUnit"), "selectionEvidence": selection.Name, "worth": 800,
		})
		if !schemaOK {
			vm.push(BoolVal(false))
			return nil
		}
		schema.SetWorth(800)
		conjectureKey := fmt.Sprintf("conjecture:%02d", code)
		conjecture, conjectureOK := gamePutArtifact(vm, d, "conjecture", conjectureKey, "GameConjecture-"+candidate.GetString("actions"), d.conjectureCategory, map[string]any{
			"candidate": candidate.Name, "candidateCode": code, "evidence": []string{candidate.GetString("evidenceUnit"), selection.Name}, "status": "proposed",
			"statement": "Policy is nondominated under the declared iterated-game profile", "conjecKind": "GameStrategyNondominated",
		})
		if !conjectureOK {
			vm.push(BoolVal(false))
			return nil
		}
		conjecture.Set("isA", []string{d.conjectureCategory, "ProtoConjec", "Anything"})
		conjecture.SetWorth(400)
	}
	experiment := vm.Store.Get(d.name)
	experiment.Set("selectionEvidence", selection.Name)
	experiment.Set("frontierUnits", frontierNames)
	experiment.Set("frontierCodes", frontier)
	experiment.Set("scalarLeaderUnits", scalarNames)
	experiment.Set("scalarLeaderCodes", scalar)
	experiment.Set("finalizationComplete", true)
	vm.push(BoolVal(gameExperimentComplete(vm, d)))
	return nil
}

func allGameCodes() []int {
	codes := make([]int, gamevocab.StrategyCount)
	for code := range codes {
		codes[code] = code
	}
	return codes
}

func gameExperimentComplete(vm *VM, d gameDescriptor) bool {
	experiment := vm.Store.Get(d.name)
	if !experiment.GetBool("finalizationComplete") {
		return false
	}
	candidates, _, frontier, scalar, behaviorRecords, behaviorClasses, objectiveRecords, objectiveClasses, dominance, ok := gameSelection(vm, d)
	if !ok || !reflect.DeepEqual(experiment.GetStrings("frontierUnits"), namesForCodes(candidates, frontier)) || !reflect.DeepEqual(experiment.Get("frontierCodes"), frontier) || !reflect.DeepEqual(experiment.GetStrings("scalarLeaderUnits"), namesForCodes(candidates, scalar)) || !reflect.DeepEqual(experiment.Get("scalarLeaderCodes"), scalar) {
		return false
	}
	selectionNames := gameAttributed(vm.Store, d, "selection", "selection:"+d.profileKey)
	if len(selectionNames) != 1 || experiment.GetString("selectionEvidence") != selectionNames[0] {
		return false
	}
	selection := vm.Store.Get(selectionNames[0])
	if !directGameMember(selection, d.selectionCategory) || !reflect.DeepEqual(selection.GetStrings("candidateUnits"), namesForCodes(candidates, allGameCodes())) || !reflect.DeepEqual(selection.GetStrings("frontierUnits"), namesForCodes(candidates, frontier)) || !reflect.DeepEqual(selection.Get("frontierCodes"), frontier) || !reflect.DeepEqual(selection.GetStrings("scalarLeaderUnits"), namesForCodes(candidates, scalar)) || !reflect.DeepEqual(selection.Get("scalarLeaderCodes"), scalar) || !reflect.DeepEqual(selection.GetStrings("behaviorClasses"), behaviorRecords) || !reflect.DeepEqual(selection.GetStrings("objectiveClasses"), objectiveRecords) || !reflect.DeepEqual(selection.GetStrings("dominanceDecisions"), dominance) {
		return false
	}
	if selection.Worth() != 500 || selection.GetInt("behaviorClassCount") != len(behaviorRecords) || selection.GetInt("objectiveClassCount") != len(objectiveRecords) || selection.GetString("comparisonMethod") != d.profile.ComparisonMethod || !reflect.DeepEqual(selection.GetStrings("objectiveNames"), []string{"training-total", "training-worst", "self-score", "perturbation-score"}) {
		return false
	}
	frontierSet := map[int]bool{}
	for _, code := range frontier {
		frontierSet[code] = true
	}
	kindCounts := map[string]int{}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("gameExperiment") == d.name && u.GetString("profileKey") == d.profileKey {
			kindCounts[u.GetString("artifactKind")]++
		}
	}
	if kindCounts["selection"] != 1 || kindCounts["schema"] != len(frontier) || kindCounts["conjecture"] != len(frontier) {
		return false
	}
	for _, candidate := range candidates {
		code := candidate.GetInt("semanticCode")
		if candidate.GetBool("paretoFrontier") != frontierSet[code] || candidate.GetString("behaviorClass") != behaviorClasses[code] || candidate.GetString("objectiveClass") != objectiveClasses[code] {
			return false
		}
		wantWorth := 500
		if frontierSet[code] {
			wantWorth = 800
			schemaNames := gameAttributed(vm.Store, d, "schema", fmt.Sprintf("schema:%02d", code))
			conjectureNames := gameAttributed(vm.Store, d, "conjecture", fmt.Sprintf("conjecture:%02d", code))
			if len(schemaNames) != 1 || len(conjectureNames) != 1 {
				return false
			}
			schema := vm.Store.Get(schemaNames[0])
			conjecture := vm.Store.Get(conjectureNames[0])
			if !directGameMember(schema, d.schemaCategory) || schema.Worth() != 800 || schema.GetInt("worth") != 800 || schema.GetString("candidate") != candidate.Name || schema.GetInt("candidateCode") != code || schema.GetString("evidenceUnit") != candidate.GetString("evidenceUnit") || schema.GetString("selectionEvidence") != selection.Name ||
				!directGameMember(conjecture, d.conjectureCategory) || !vm.Store.IsA(conjecture.Name, "ProtoConjec") || conjecture.Worth() != 400 || conjecture.GetString("candidate") != candidate.Name || conjecture.GetInt("candidateCode") != code || conjecture.GetString("status") != "proposed" || conjecture.GetString("statement") != "Policy is nondominated under the declared iterated-game profile" || conjecture.GetString("conjecKind") != "GameStrategyNondominated" || !reflect.DeepEqual(conjecture.GetStrings("evidence"), []string{candidate.GetString("evidenceUnit"), selection.Name}) {
				return false
			}
		} else if len(gameAttributed(vm.Store, d, "schema", fmt.Sprintf("schema:%02d", code))) != 0 || len(gameAttributed(vm.Store, d, "conjecture", fmt.Sprintf("conjecture:%02d", code))) != 0 {
			return false
		}
		if candidate.Worth() != wantWorth {
			return false
		}
	}
	return true
}

func bGameExperimentComplete(vm *VM) error {
	name := vm.pop().AsString()
	d, ok := readGameDescriptor(vm, name)
	vm.push(BoolVal(ok && gameExperimentComplete(vm, d)))
	return nil
}
