package game

import (
	"reflect"
	"testing"
)

func mustStrategy(t *testing.T, actions string) Strategy {
	t.Helper()
	if len(actions) != 5 {
		t.Fatalf("bad fixture %q", actions)
	}
	records := make([]string, 5)
	for index, name := range recordNames {
		records[index] = name + ":" + actions[index:index+1]
	}
	strategy, err := ParseStrategy(records)
	if err != nil {
		t.Fatal(err)
	}
	return strategy
}

func TestAllStrategiesRoundTripCanonicalCodes(t *testing.T) {
	strategies := AllStrategies()
	if len(strategies) != 32 {
		t.Fatalf("count = %d", len(strategies))
	}
	seen := map[string]bool{}
	for code, strategy := range strategies {
		if strategy.Code() != code {
			t.Fatalf("code %d round trip = %d", code, strategy.Code())
		}
		parsed, err := ParseStrategy(strategy.Records())
		if err != nil || parsed != strategy {
			t.Fatalf("code %d parse = (%v,%v)", code, parsed, err)
		}
		if seen[strategy.Actions()] {
			t.Fatalf("duplicate strategy %s", strategy.Actions())
		}
		seen[strategy.Actions()] = true
	}
	if strategies[0].Actions() != "DDDDD" || strategies[31].Actions() != "CCCCC" {
		t.Fatalf("boundary actions = %s/%s", strategies[0].Actions(), strategies[31].Actions())
	}
}

func TestStrategyRejectsMalformedRecords(t *testing.T) {
	valid := mustStrategy(t, "CCDCD").Records()
	cases := [][]string{
		nil,
		valid[:4],
		{"initial:C", "after-CC:C", "after-CD:D", "after-DC:C", "after-DC:D"},
		{"initial:C", "after-CC:C", "after-CD:X", "after-DC:C", "after-DD:D"},
		{"after-CC:C", "initial:C", "after-CD:D", "after-DC:C", "after-DD:D"},
	}
	for _, records := range cases {
		if _, err := ParseStrategy(records); err == nil {
			t.Fatalf("accepted malformed strategy %v", records)
		}
	}
}

func TestPlayUsesOpponentPerspectiveAndRealizedFlips(t *testing.T) {
	payoffs := Payoffs{Temptation: 5, Reward: 3, Punishment: 1, Sucker: 0}
	tft := mustStrategy(t, "CCDCD")
	alld := mustStrategy(t, "DDDDD")
	result, err := Play(tft, alld, payoffs, MatchSpec{Rounds: 4})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateTrace != "CDDD" || result.OpponentTrace != "DDDD" || result.CandidateScore != 3 || result.OpponentScore != 8 {
		t.Fatalf("TFT/AllD = %#v", result)
	}

	result, err = Play(tft, tft, payoffs, MatchSpec{Rounds: 5, CandidateFlip: []int{1}, OpponentFlip: []int{2}})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateTrace != "CDCCC" || result.OpponentTrace != "CCCCC" {
		t.Fatalf("flipped traces = %s/%s", result.CandidateTrace, result.OpponentTrace)
	}
}

func TestPlayRejectsInvalidPayoffsRoundsAndFlipSchedules(t *testing.T) {
	strategy := mustStrategy(t, "CCDCD")
	validPayoffs := Payoffs{Temptation: 5, Reward: 3, Punishment: 1, Sucker: 0}
	cases := []struct {
		payoffs Payoffs
		spec    MatchSpec
	}{
		{Payoffs{Temptation: 3, Reward: 3, Punishment: 1, Sucker: 0}, MatchSpec{Rounds: 5}},
		{validPayoffs, MatchSpec{Rounds: 0}},
		{validPayoffs, MatchSpec{Rounds: MaxRounds + 1}},
		{validPayoffs, MatchSpec{Rounds: 5, CandidateFlip: []int{2, 2}}},
		{validPayoffs, MatchSpec{Rounds: 5, OpponentFlip: []int{4, 3}}},
		{validPayoffs, MatchSpec{Rounds: 5, CandidateFlip: []int{5}}},
	}
	for _, testCase := range cases {
		if _, err := Play(strategy, strategy, testCase.payoffs, testCase.spec); err == nil {
			t.Fatalf("accepted invalid match: %#v", testCase)
		}
	}
}

func TestProfileKeyIsSemanticAndOrderSensitive(t *testing.T) {
	tft := mustStrategy(t, "CCDCD")
	base := Profile{
		ExperimentKey: "test/v1", ComparisonMethod: "exhaustive/v1",
		Payoffs: Payoffs{Temptation: 5, Reward: 3, Punishment: 1, Sucker: 0}, Rounds: 60,
		Cases: []Case{
			{Axis: Training, Opponent: mustStrategy(t, "CCCCC")},
			{Axis: Self, Self: true},
			{Axis: Perturbation, Opponent: tft, CandidateFlip: []int{10}},
		},
	}
	first, err := base.Key()
	if err != nil {
		t.Fatal(err)
	}
	reordered := base
	reordered.Cases = []Case{base.Cases[1], base.Cases[0], base.Cases[2]}
	second, err := reordered.Key()
	if err != nil || first == second {
		t.Fatalf("order keys = %q/%q err=%v", first, second, err)
	}
	duplicate := base
	duplicate.Cases = append(append([]Case(nil), base.Cases...), base.Cases[0])
	if duplicate.Valid() {
		t.Fatal("profile accepted duplicate semantic case")
	}
}

func TestSeedProfileKeyIsFrozen(t *testing.T) {
	profile := Profile{
		ExperimentKey: "game/memory-one-pd/profile-a/v1", ComparisonMethod: "exhaustive-memory-one-profile/v1",
		Payoffs: Payoffs{Temptation: 5, Reward: 3, Punishment: 1, Sucker: 0}, Rounds: 60,
		Cases: []Case{
			{Axis: Training, Opponent: mustStrategy(t, "CCCCC")},
			{Axis: Training, Opponent: mustStrategy(t, "DDDDD")},
			{Axis: Training, Opponent: mustStrategy(t, "CCDCD")},
			{Axis: Training, Opponent: mustStrategy(t, "CDDCC")},
			{Axis: Self, Self: true},
			{Axis: Perturbation, Opponent: mustStrategy(t, "CCDCD"), CandidateFlip: []int{10}, OpponentFlip: []int{20}},
		},
	}
	key, err := profile.Key()
	const want = "sha256:v1:a546aa6e4374be3f4bd13e96e6eab698361db85471d79da435dd04e70123f0b4"
	if err != nil || key != want {
		t.Fatalf("seed profile key = %q, want %q (err %v)", key, want, err)
	}
}

func TestDominanceFrontierAndScalarLeaders(t *testing.T) {
	objectives := map[int]Objectives{
		0: {TrainingTotal: 10, TrainingWorst: 1, SelfScore: 5, PerturbScore: 3},
		1: {TrainingTotal: 10, TrainingWorst: 2, SelfScore: 5, PerturbScore: 3},
		2: {TrainingTotal: 9, TrainingWorst: 3, SelfScore: 6, PerturbScore: 4},
	}
	if !objectives[1].Dominates(objectives[0]) || objectives[0].Dominates(objectives[1]) {
		t.Fatal("dominance direction is wrong")
	}
	if got := Frontier(objectives); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("frontier = %v", got)
	}
	if got := ScalarLeaders(objectives); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatalf("scalar leaders = %v", got)
	}
}
