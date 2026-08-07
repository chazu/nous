// Package game implements bounded iterated-game semantics without depending on
// Nous units, the DSL, or the engine.
package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	StrategyRecords = 5
	StrategyCount   = 32
	MaxRounds       = 200
	MaxPayoff       = 100
	MaxCases        = 16
	MaxTraining     = 14
	MaxNameBytes    = 256
	ProfileVersion  = "game-profile-json/v1"
	CaseVersion     = "game-case-json/v1"
)

type Action byte

const (
	Defect    Action = 'D'
	Cooperate Action = 'C'
)

type Strategy [StrategyRecords]Action

var recordNames = [...]string{"initial", "after-CC", "after-CD", "after-DC", "after-DD"}

func ParseStrategy(records []string) (Strategy, error) {
	if len(records) != StrategyRecords {
		return Strategy{}, fmt.Errorf("strategy has %d records, want %d", len(records), StrategyRecords)
	}
	var strategy Strategy
	for index, name := range recordNames {
		prefix := name + ":"
		if !strings.HasPrefix(records[index], prefix) || len(records[index]) != len(prefix)+1 {
			return Strategy{}, fmt.Errorf("invalid strategy record %q at %d", records[index], index)
		}
		action := Action(records[index][len(prefix)])
		if !action.Valid() {
			return Strategy{}, fmt.Errorf("invalid action in %q", records[index])
		}
		strategy[index] = action
	}
	return strategy, nil
}

func (a Action) Valid() bool { return a == Cooperate || a == Defect }

func (s Strategy) Valid() bool {
	for _, action := range s {
		if !action.Valid() {
			return false
		}
	}
	return true
}

func (s Strategy) Records() []string {
	if !s.Valid() {
		return nil
	}
	records := make([]string, StrategyRecords)
	for index, name := range recordNames {
		records[index] = name + ":" + string(s[index])
	}
	return records
}

func (s Strategy) Actions() string {
	if !s.Valid() {
		return ""
	}
	return string(s[:])
}

func (s Strategy) Code() int {
	if !s.Valid() {
		return -1
	}
	code := 0
	for _, action := range s {
		code <<= 1
		if action == Cooperate {
			code++
		}
	}
	return code
}

func StrategyFromCode(code int) (Strategy, error) {
	if code < 0 || code >= StrategyCount {
		return Strategy{}, fmt.Errorf("strategy code %d out of range", code)
	}
	var strategy Strategy
	for index := StrategyRecords - 1; index >= 0; index-- {
		strategy[index] = Defect
		if code&1 == 1 {
			strategy[index] = Cooperate
		}
		code >>= 1
	}
	return strategy, nil
}

func AllStrategies() []Strategy {
	strategies := make([]Strategy, StrategyCount)
	for code := 0; code < StrategyCount; code++ {
		strategies[code], _ = StrategyFromCode(code)
	}
	return strategies
}

type Payoffs struct {
	Temptation int `json:"temptation"`
	Reward     int `json:"reward"`
	Punishment int `json:"punishment"`
	Sucker     int `json:"sucker"`
}

func (p Payoffs) Valid() bool {
	values := []int{p.Temptation, p.Reward, p.Punishment, p.Sucker}
	for _, value := range values {
		if value < 0 || value > MaxPayoff {
			return false
		}
	}
	return p.Temptation > p.Reward && p.Reward > p.Punishment && p.Punishment > p.Sucker &&
		2*p.Reward > p.Temptation+p.Sucker
}

type MatchSpec struct {
	Rounds        int   `json:"rounds"`
	CandidateFlip []int `json:"candidate_flips"`
	OpponentFlip  []int `json:"opponent_flips"`
}

func (s MatchSpec) Valid() bool {
	return s.Rounds >= 1 && s.Rounds <= MaxRounds && validFlips(s.CandidateFlip, s.Rounds) && validFlips(s.OpponentFlip, s.Rounds)
}

func validFlips(flips []int, rounds int) bool {
	if len(flips) > MaxRounds {
		return false
	}
	previous := -1
	for _, round := range flips {
		if round < 0 || round >= rounds || round <= previous {
			return false
		}
		previous = round
	}
	return true
}

type MatchResult struct {
	Rounds                int
	CandidateScore        int
	OpponentScore         int
	CandidateCooperations int
	OpponentCooperations  int
	MutualCooperations    int
	CandidateTrace        string
	OpponentTrace         string
}

func Play(candidate, opponent Strategy, payoffs Payoffs, spec MatchSpec) (MatchResult, error) {
	if !candidate.Valid() || !opponent.Valid() || !payoffs.Valid() || !spec.Valid() {
		return MatchResult{}, fmt.Errorf("invalid match input")
	}
	candidateFlips := flipSet(spec.CandidateFlip)
	opponentFlips := flipSet(spec.OpponentFlip)
	result := MatchResult{Rounds: spec.Rounds}
	var previousCandidate, previousOpponent Action
	for round := 0; round < spec.Rounds; round++ {
		candidateAction := candidate[0]
		opponentAction := opponent[0]
		if round > 0 {
			candidateAction = candidate[historyIndex(previousCandidate, previousOpponent)]
			opponentAction = opponent[historyIndex(previousOpponent, previousCandidate)]
		}
		if candidateFlips[round] {
			candidateAction = flip(candidateAction)
		}
		if opponentFlips[round] {
			opponentAction = flip(opponentAction)
		}
		candidateScore, opponentScore := score(candidateAction, opponentAction, payoffs)
		result.CandidateScore += candidateScore
		result.OpponentScore += opponentScore
		if candidateAction == Cooperate {
			result.CandidateCooperations++
		}
		if opponentAction == Cooperate {
			result.OpponentCooperations++
		}
		if candidateAction == Cooperate && opponentAction == Cooperate {
			result.MutualCooperations++
		}
		result.CandidateTrace += string(candidateAction)
		result.OpponentTrace += string(opponentAction)
		previousCandidate, previousOpponent = candidateAction, opponentAction
	}
	return result, nil
}

func historyIndex(own, other Action) int {
	switch {
	case own == Cooperate && other == Cooperate:
		return 1
	case own == Cooperate && other == Defect:
		return 2
	case own == Defect && other == Cooperate:
		return 3
	default:
		return 4
	}
}

func flip(action Action) Action {
	if action == Cooperate {
		return Defect
	}
	return Cooperate
}

func flipSet(flips []int) map[int]bool {
	result := make(map[int]bool, len(flips))
	for _, round := range flips {
		result[round] = true
	}
	return result
}

func score(candidate, opponent Action, payoffs Payoffs) (int, int) {
	switch {
	case candidate == Cooperate && opponent == Cooperate:
		return payoffs.Reward, payoffs.Reward
	case candidate == Cooperate && opponent == Defect:
		return payoffs.Sucker, payoffs.Temptation
	case candidate == Defect && opponent == Cooperate:
		return payoffs.Temptation, payoffs.Sucker
	default:
		return payoffs.Punishment, payoffs.Punishment
	}
}

type Axis string

const (
	Training     Axis = "training"
	Self         Axis = "self"
	Perturbation Axis = "perturbation"
)

type Case struct {
	Axis           Axis     `json:"axis"`
	Self           bool     `json:"self"`
	Opponent       Strategy `json:"-"`
	OpponentAction string   `json:"opponent,omitempty"`
	CandidateFlip  []int    `json:"candidate_flips"`
	OpponentFlip   []int    `json:"opponent_flips"`
}

func (c Case) Canonical() (caseMaterial, bool) {
	material := caseMaterial{Version: CaseVersion, Axis: string(c.Axis), Self: c.Self, CandidateFlip: cloneInts(c.CandidateFlip), OpponentFlip: cloneInts(c.OpponentFlip)}
	if !c.Self {
		material.Opponent = c.Opponent.Actions()
	}
	validAxis := c.Axis == Training || c.Axis == Self || c.Axis == Perturbation
	validSelf := (c.Axis == Self) == c.Self
	return material, validAxis && validSelf && (c.Self || c.Opponent.Valid())
}

type caseMaterial struct {
	Version       string `json:"version"`
	Axis          string `json:"axis"`
	Self          bool   `json:"self"`
	Opponent      string `json:"opponent,omitempty"`
	CandidateFlip []int  `json:"candidate_flips"`
	OpponentFlip  []int  `json:"opponent_flips"`
}

func (c Case) Digest(rounds int) (string, error) {
	material, ok := c.Canonical()
	if !ok || !validFlips(c.CandidateFlip, rounds) || !validFlips(c.OpponentFlip, rounds) {
		return "", fmt.Errorf("invalid case")
	}
	encoded, _ := json.Marshal(material)
	return digest(encoded), nil
}

type Profile struct {
	ExperimentKey    string
	ComparisonMethod string
	Payoffs          Payoffs
	Rounds           int
	Cases            []Case
}

func (p Profile) Valid() bool {
	if p.ExperimentKey == "" || len(p.ExperimentKey) > MaxNameBytes || p.ComparisonMethod == "" || len(p.ComparisonMethod) > MaxNameBytes ||
		!p.Payoffs.Valid() || p.Rounds < 1 || p.Rounds > MaxRounds || len(p.Cases) < 3 || len(p.Cases) > MaxCases {
		return false
	}
	training, self, perturbation := 0, 0, 0
	seen := map[string]bool{}
	for _, c := range p.Cases {
		digest, err := c.Digest(p.Rounds)
		if err != nil || seen[digest] {
			return false
		}
		seen[digest] = true
		switch c.Axis {
		case Training:
			training++
		case Self:
			self++
		case Perturbation:
			perturbation++
		}
	}
	return training >= 1 && training <= MaxTraining && self == 1 && perturbation == 1
}

func (p Profile) Key() (string, error) {
	if !p.Valid() {
		return "", fmt.Errorf("invalid profile")
	}
	type profileMaterial struct {
		Version          string         `json:"version"`
		ExperimentKey    string         `json:"experiment_key"`
		ComparisonMethod string         `json:"comparison_method"`
		Payoffs          Payoffs        `json:"payoffs"`
		Rounds           int            `json:"rounds"`
		Cases            []caseMaterial `json:"cases"`
	}
	material := profileMaterial{Version: ProfileVersion, ExperimentKey: p.ExperimentKey, ComparisonMethod: p.ComparisonMethod, Payoffs: p.Payoffs, Rounds: p.Rounds}
	for _, c := range p.Cases {
		canonical, _ := c.Canonical()
		material.Cases = append(material.Cases, canonical)
	}
	encoded, _ := json.Marshal(material)
	return digest(encoded), nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:v1:" + hex.EncodeToString(sum[:])
}

type Objectives struct {
	TrainingTotal int `json:"training_total"`
	TrainingWorst int `json:"training_worst"`
	SelfScore     int `json:"self_score"`
	PerturbScore  int `json:"perturbation_score"`
}

func (a Objectives) Dominates(b Objectives) bool {
	atLeast := a.TrainingTotal >= b.TrainingTotal && a.TrainingWorst >= b.TrainingWorst && a.SelfScore >= b.SelfScore && a.PerturbScore >= b.PerturbScore
	strict := a.TrainingTotal > b.TrainingTotal || a.TrainingWorst > b.TrainingWorst || a.SelfScore > b.SelfScore || a.PerturbScore > b.PerturbScore
	return atLeast && strict
}

func Frontier(objectives map[int]Objectives) []int {
	var codes []int
	for code, candidate := range objectives {
		dominated := false
		for otherCode, other := range objectives {
			if otherCode != code && other.Dominates(candidate) {
				dominated = true
				break
			}
		}
		if !dominated {
			codes = append(codes, code)
		}
	}
	sort.Ints(codes)
	return codes
}

func ScalarLeaders(objectives map[int]Objectives) []int {
	maximum := -1
	for _, objective := range objectives {
		if objective.TrainingTotal > maximum {
			maximum = objective.TrainingTotal
		}
	}
	var codes []int
	for code, objective := range objectives {
		if objective.TrainingTotal == maximum {
			codes = append(codes, code)
		}
	}
	sort.Ints(codes)
	return codes
}

func cloneInts(values []int) []int {
	if len(values) == 0 {
		return []int{}
	}
	return append([]int(nil), values...)
}
