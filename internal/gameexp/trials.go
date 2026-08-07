// Package gameexp runs the preregistered iterated-game trial and a deliberately
// independent exhaustive oracle. This package must not import vocab/game.
package gameexp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
)

const (
	gameCycles     = 500
	experimentName = "MemoryOnePDProfileA"
)

type Objectives struct {
	TrainingTotal int `json:"training_total"`
	TrainingWorst int `json:"training_worst"`
	SelfScore     int `json:"self_score"`
	PerturbScore  int `json:"perturbation_score"`
}

type PolicyReport struct {
	Code                  int        `json:"code"`
	Actions               string     `json:"actions"`
	Objectives            Objectives `json:"objectives"`
	ObjectiveClass        string     `json:"objective_class"`
	TrainingBehaviorClass string     `json:"training_behavior_class"`
	ScalarLeader          bool       `json:"scalar_leader"`
	ParetoFrontier        bool       `json:"pareto_frontier"`
	HeldOutScores         []int      `json:"held_out_scores"`
	HeldOutTotal          int        `json:"held_out_total"`
	HeldOutWorst          int        `json:"held_out_worst"`
	HeldOutBehaviorClass  string     `json:"held_out_behavior_class"`
}

type OracleReport struct {
	EnumerationAgreements       int  `json:"enumeration_agreements"`
	MatchAgreements             int  `json:"match_agreements"`
	ObjectiveAgreements         int  `json:"objective_agreements"`
	DominanceAgreements         int  `json:"dominance_agreements"`
	FrontierAgreement           bool `json:"frontier_agreement"`
	ScalarLeaderAgreement       bool `json:"scalar_leader_agreement"`
	BehaviorPartitionAgreements int  `json:"behavior_partition_agreements"`
}

type Report struct {
	ExperimentKey           string         `json:"experiment_key"`
	ProfileKey              string         `json:"profile_key"`
	ExperimentComplete      bool           `json:"experiment_complete"`
	AgendaDrained           bool           `json:"agenda_drained"`
	Candidates              int            `json:"candidates"`
	EvaluationCases         int            `json:"evaluation_cases"`
	Results                 int            `json:"results"`
	Observations            int            `json:"observations"`
	Evidence                int            `json:"evidence"`
	Schemas                 int            `json:"schemas"`
	Conjectures             int            `json:"conjectures"`
	Frontier                []string       `json:"frontier"`
	ScalarLeaders           []string       `json:"scalar_leaders"`
	SelectionIntersection   []string       `json:"selection_intersection"`
	FrontierOnly            []string       `json:"frontier_only"`
	ScalarOnly              []string       `json:"scalar_only"`
	ObjectiveClasses        int            `json:"objective_classes"`
	TrainingBehaviorClasses int            `json:"training_behavior_classes"`
	FrontierBehaviorClasses int            `json:"frontier_behavior_classes"`
	HeldOutBehaviorClasses  int            `json:"held_out_behavior_classes"`
	SplitTrainingClasses    int            `json:"split_training_classes"`
	SplitCandidatePairs     int            `json:"split_candidate_pairs"`
	TrainingStoreUnchanged  bool           `json:"training_store_unchanged"`
	Oracle                  OracleReport   `json:"oracle"`
	Policies                []PolicyReport `json:"policies"`
	Limitations             []string       `json:"limitations"`
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

type indCase struct {
	axis                        string
	opponent                    string
	self                        bool
	candidateFlip, opponentFlip map[int]bool
}

type indMatch struct {
	candidateScore, opponentScore int
	candidateTrace, opponentTrace string
}

var trainingCases = []indCase{
	{axis: "training", opponent: "CCCCC", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{axis: "training", opponent: "DDDDD", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{axis: "training", opponent: "CCDCD", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{axis: "training", opponent: "CDDCC", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{axis: "self", self: true, candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{axis: "perturbation", opponent: "CCDCD", candidateFlip: map[int]bool{10: true}, opponentFlip: map[int]bool{20: true}},
}

var heldOutCases = []indCase{
	{opponent: "CCDDC", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{opponent: "CCCDD", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{opponent: "DCDCD", candidateFlip: map[int]bool{}, opponentFlip: map[int]bool{}},
	{opponent: "CCDCD", candidateFlip: map[int]bool{1: true}, opponentFlip: map[int]bool{}},
}

func Run(domainsDir string) (Report, error) {
	store, eng, ag, err := runVocabulary(domainsDir)
	if err != nil {
		return Report{}, err
	}
	value, err := eng.VM.Execute(fmt.Sprintf("%q game-experiment-complete?", experimentName))
	if err != nil || !value.AsBool() {
		return Report{}, fmt.Errorf("post-selection verifier = (%v,%v)", value, err)
	}
	experiment := store.Get(experimentName)
	if experiment == nil {
		return Report{}, fmt.Errorf("missing experiment")
	}
	before, err := store.CanonicalJSON()
	if err != nil {
		return Report{}, err
	}

	objectiveByCode := map[int]Objectives{}
	trainingSignature := map[int]string{}
	heldOutSignature := map[int]string{}
	heldOutScores := map[int][]int{}
	rows := make([]PolicyReport, 0, 32)
	oracle := OracleReport{}
	candidateByCode := map[int]*unit.Unit{}
	for code, candidateName := range experiment.GetStrings("candidateUnits") {
		candidate := store.Get(candidateName)
		actions := actionsForCode(code)
		if candidate != nil && candidate.GetInt("semanticCode") == code && candidate.GetString("actions") == actions {
			oracle.EnumerationAgreements++
		}
		candidateByCode[code] = candidate
		objectives, signature, matches := independentTraining(actions)
		objectiveByCode[code] = objectives
		trainingSignature[code] = signature
		if candidate != nil && objectives == storedObjectives(candidate) {
			oracle.ObjectiveAgreements++
		}
		evidence := store.Get(candidate.GetString("evidenceUnit"))
		if evidence == nil || len(evidence.GetStrings("resultUnits")) != len(trainingCases) {
			return Report{}, fmt.Errorf("candidate %02d incomplete evidence", code)
		}
		for index, resultName := range evidence.GetStrings("resultUnits") {
			result := store.Get(resultName)
			match := matches[index]
			if result != nil && result.GetInt("candidateScore") == match.candidateScore && result.GetInt("opponentScore") == match.opponentScore && result.GetString("candidateTrace") == match.candidateTrace && result.GetString("opponentTrace") == match.opponentTrace {
				oracle.MatchAgreements++
			}
		}
		scores, heldSignature := independentHeldOut(actions)
		heldOutScores[code] = scores
		heldOutSignature[code] = heldSignature
	}

	expectedFrontier := independentFrontier(objectiveByCode)
	expectedScalar := independentScalar(objectiveByCode)
	storedFrontier := intSlice(experiment.Get("frontierCodes"))
	storedScalar := intSlice(experiment.Get("scalarLeaderCodes"))
	oracle.FrontierAgreement = equalInts(expectedFrontier, storedFrontier)
	oracle.ScalarLeaderAgreement = equalInts(expectedScalar, storedScalar)
	for first := 0; first < 32; first++ {
		for second := 0; second < 32; second++ {
			want := independentDominates(objectiveByCode[first], objectiveByCode[second])
			selection := store.Get(experiment.GetString("selectionEvidence"))
			records := selection.GetStrings("dominanceDecisions")
			index := first*32 + second
			if index < len(records) && records[index] == fmt.Sprintf("%02d>%02d:%t", first, second, want) {
				oracle.DominanceAgreements++
			}
		}
	}

	trainingClasses, trainingClassByCode := independentClasses(trainingSignature, "B")
	objectiveKeys := map[int]string{}
	for code, objective := range objectiveByCode {
		objectiveKeys[code] = fmt.Sprintf("%d/%d/%d/%d", objective.TrainingTotal, objective.TrainingWorst, objective.SelfScore, objective.PerturbScore)
	}
	objectiveClasses, objectiveClassByCode := independentClasses(objectiveKeys, "O")
	heldClasses, heldClassByCode := independentClasses(heldOutSignature, "H")
	for first := 0; first < 32; first++ {
		for second := 0; second < 32; second++ {
			storedSame := candidateByCode[first].GetString("behaviorClass") == candidateByCode[second].GetString("behaviorClass")
			oracleSame := trainingClassByCode[first] == trainingClassByCode[second]
			if storedSame == oracleSame {
				oracle.BehaviorPartitionAgreements++
			}
		}
	}

	frontierSet, scalarSet := intSet(expectedFrontier), intSet(expectedScalar)
	frontierBehavior := map[string]bool{}
	for code := range frontierSet {
		frontierBehavior[trainingClassByCode[code]] = true
	}
	splitClasses, splitPairs := classSplits(trainingClassByCode, heldClassByCode)
	for code := 0; code < 32; code++ {
		scores := heldOutScores[code]
		total, worst := 0, 20000
		for _, score := range scores {
			total += score
			if score < worst {
				worst = score
			}
		}
		rows = append(rows, PolicyReport{
			Code: code, Actions: actionsForCode(code), Objectives: objectiveByCode[code], ObjectiveClass: objectiveClassByCode[code], TrainingBehaviorClass: trainingClassByCode[code],
			ScalarLeader: scalarSet[code], ParetoFrontier: frontierSet[code], HeldOutScores: scores, HeldOutTotal: total, HeldOutWorst: worst, HeldOutBehaviorClass: heldClassByCode[code],
		})
	}
	after, err := store.CanonicalJSON()
	if err != nil {
		return Report{}, err
	}

	counts := artifactCounts(store, experimentName, experiment.GetString("profileKey"))
	report := Report{
		ExperimentKey: experiment.GetString("experimentKey"), ProfileKey: experiment.GetString("profileKey"), ExperimentComplete: true, AgendaDrained: ag.Len() == 0,
		Candidates: counts["candidate"], EvaluationCases: len(trainingCases), Results: counts["result"], Observations: counts["observation"], Evidence: counts["evidence"], Schemas: counts["schema"], Conjectures: counts["conjecture"],
		Frontier: actionsForCodes(expectedFrontier), ScalarLeaders: actionsForCodes(expectedScalar),
		SelectionIntersection: setIntersection(expectedFrontier, expectedScalar), FrontierOnly: setDifference(expectedFrontier, expectedScalar), ScalarOnly: setDifference(expectedScalar, expectedFrontier),
		ObjectiveClasses: len(objectiveClasses), TrainingBehaviorClasses: len(trainingClasses), FrontierBehaviorClasses: len(frontierBehavior), HeldOutBehaviorClasses: len(heldClasses),
		SplitTrainingClasses: splitClasses, SplitCandidatePairs: splitPairs, TrainingStoreUnchanged: string(before) == string(after), Oracle: oracle, Policies: rows,
		Limitations: []string{
			"The 32-policy space is exhaustively enumerated; this is correctness evidence, not a search advantage.",
			"The opponent profile and four objectives are fixed and preregistered, not an evolving population.",
			"Pareto membership means nondominated under this profile, not universally useful.",
			"No probabilistic policy, longer memory, learning, diversity reward, or component-credit transfer is tested.",
		},
	}
	return report, nil
}

func runVocabulary(domainsDir string) (*unit.Store, *engine.Engine, *agenda.Agenda, error) {
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "games"); err != nil {
		return nil, nil, nil, err
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = gameCycles
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		return nil, nil, nil, err
	}
	return store, eng, ag, nil
}

func actionsForCode(code int) string {
	result := make([]byte, 5)
	for index := 4; index >= 0; index-- {
		result[index] = 'D'
		if code&1 == 1 {
			result[index] = 'C'
		}
		code >>= 1
	}
	return string(result)
}

func independentPlay(candidate, opponent string, c indCase) indMatch {
	if c.self {
		opponent = candidate
	}
	result := indMatch{}
	var previousCandidate, previousOpponent byte
	for round := 0; round < 60; round++ {
		candidateAction, opponentAction := candidate[0], opponent[0]
		if round > 0 {
			candidateAction = candidate[indHistory(previousCandidate, previousOpponent)]
			opponentAction = opponent[indHistory(previousOpponent, previousCandidate)]
		}
		if c.candidateFlip[round] {
			candidateAction = indFlip(candidateAction)
		}
		if c.opponentFlip[round] {
			opponentAction = indFlip(opponentAction)
		}
		candidateScore, opponentScore := indScore(candidateAction, opponentAction)
		result.candidateScore += candidateScore
		result.opponentScore += opponentScore
		result.candidateTrace += string(candidateAction)
		result.opponentTrace += string(opponentAction)
		previousCandidate, previousOpponent = candidateAction, opponentAction
	}
	return result
}

func indHistory(own, other byte) int {
	switch string([]byte{own, other}) {
	case "CC":
		return 1
	case "CD":
		return 2
	case "DC":
		return 3
	default:
		return 4
	}
}

func indFlip(action byte) byte {
	if action == 'C' {
		return 'D'
	}
	return 'C'
}

func indScore(candidate, opponent byte) (int, int) {
	switch string([]byte{candidate, opponent}) {
	case "CC":
		return 3, 3
	case "CD":
		return 0, 5
	case "DC":
		return 5, 0
	default:
		return 1, 1
	}
}

func independentTraining(actions string) (Objectives, string, []indMatch) {
	objective := Objectives{TrainingWorst: 20000}
	var signature []string
	var matches []indMatch
	for index, c := range trainingCases {
		match := independentPlay(actions, c.opponent, c)
		matches = append(matches, match)
		switch c.axis {
		case "training":
			objective.TrainingTotal += match.candidateScore
			if match.candidateScore < objective.TrainingWorst {
				objective.TrainingWorst = match.candidateScore
			}
			signature = append(signature, fmt.Sprintf("%d:%s", index, match.candidateTrace))
		case "self":
			objective.SelfScore = match.candidateScore
		case "perturbation":
			objective.PerturbScore = match.candidateScore
		}
	}
	return objective, strings.Join(signature, "|"), matches
}

func independentHeldOut(actions string) ([]int, string) {
	scores := make([]int, len(heldOutCases))
	traces := make([]string, len(heldOutCases))
	for index, c := range heldOutCases {
		match := independentPlay(actions, c.opponent, c)
		scores[index] = match.candidateScore
		traces[index] = fmt.Sprintf("%d:%s", index, match.candidateTrace)
	}
	return scores, strings.Join(traces, "|")
}

func independentDominates(a, b Objectives) bool {
	atLeast := a.TrainingTotal >= b.TrainingTotal && a.TrainingWorst >= b.TrainingWorst && a.SelfScore >= b.SelfScore && a.PerturbScore >= b.PerturbScore
	strict := a.TrainingTotal > b.TrainingTotal || a.TrainingWorst > b.TrainingWorst || a.SelfScore > b.SelfScore || a.PerturbScore > b.PerturbScore
	return atLeast && strict
}

func independentFrontier(objectives map[int]Objectives) []int {
	var result []int
	for code := 0; code < 32; code++ {
		dominated := false
		for other := 0; other < 32; other++ {
			if other != code && independentDominates(objectives[other], objectives[code]) {
				dominated = true
				break
			}
		}
		if !dominated {
			result = append(result, code)
		}
	}
	return result
}

func independentScalar(objectives map[int]Objectives) []int {
	maximum := -1
	for _, objective := range objectives {
		if objective.TrainingTotal > maximum {
			maximum = objective.TrainingTotal
		}
	}
	var result []int
	for code := 0; code < 32; code++ {
		if objectives[code].TrainingTotal == maximum {
			result = append(result, code)
		}
	}
	return result
}

func independentClasses(keys map[int]string, prefix string) ([]string, map[int]string) {
	groups := map[string][]int{}
	for code := 0; code < 32; code++ {
		groups[keys[code]] = append(groups[keys[code]], code)
	}
	type group []int
	ordered := make([]group, 0, len(groups))
	for _, members := range groups {
		ordered = append(ordered, members)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i][0] < ordered[j][0] })
	records := make([]string, len(ordered))
	byCode := map[int]string{}
	for index, members := range ordered {
		name := fmt.Sprintf("%s%03d", prefix, index+1)
		parts := make([]string, len(members))
		for memberIndex, code := range members {
			parts[memberIndex] = fmt.Sprintf("%02d", code)
			byCode[code] = name
		}
		records[index] = name + ":" + strings.Join(parts, ",")
	}
	return records, byCode
}

func classSplits(training, held map[int]string) (int, int) {
	groups := map[string][]int{}
	for code := 0; code < 32; code++ {
		groups[training[code]] = append(groups[training[code]], code)
	}
	splitClasses, splitPairs := 0, 0
	for _, members := range groups {
		heldClasses := map[string]bool{}
		for _, code := range members {
			heldClasses[held[code]] = true
		}
		if len(heldClasses) > 1 {
			splitClasses++
		}
		for first := 0; first < len(members); first++ {
			for second := first + 1; second < len(members); second++ {
				if held[members[first]] != held[members[second]] {
					splitPairs++
				}
			}
		}
	}
	return splitClasses, splitPairs
}

func storedObjectives(u *unit.Unit) Objectives {
	return Objectives{TrainingTotal: u.GetInt("trainingTotal"), TrainingWorst: u.GetInt("trainingWorst"), SelfScore: u.GetInt("selfScore"), PerturbScore: u.GetInt("perturbationScore")}
}

func intSlice(value any) []int {
	values, _ := value.([]int)
	return append([]int(nil), values...)
}

func equalInts(a, b []int) bool {
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

func intSet(values []int) map[int]bool {
	result := map[int]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func actionsForCodes(codes []int) []string {
	result := make([]string, len(codes))
	for index, code := range codes {
		result[index] = actionsForCode(code)
	}
	return result
}

func setIntersection(a, b []int) []string {
	bSet := intSet(b)
	var result []string
	for _, code := range a {
		if bSet[code] {
			result = append(result, actionsForCode(code))
		}
	}
	return result
}

func setDifference(a, b []int) []string {
	bSet := intSet(b)
	var result []string
	for _, code := range a {
		if !bSet[code] {
			result = append(result, actionsForCode(code))
		}
	}
	return result
}

func artifactCounts(store *unit.Store, experiment, profile string) map[string]int {
	counts := map[string]int{}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("gameExperiment") == experiment && u.GetString("profileKey") == profile {
			counts[u.GetString("artifactKind")]++
		}
	}
	return counts
}
