package actionrelations

import "testing"

func TestAllEightActionSemantics(t *testing.T) {
	tests := []struct {
		name   string
		state  State
		action SemanticAction
		want   State
	}{
		{"add", twoCellState(1, 2), SemanticAction{Kind: "add", XRole: "c0", N: 2}, twoCellState(3, 2)},
		{"set-zero", twoCellState(3, 2), SemanticAction{Kind: "set", XRole: "c0", N: 0}, twoCellState(0, 2)},
		{"transfer", twoCellState(2, 1), SemanticAction{Kind: "transfer", XRole: "c0", YRole: "c1", N: 2}, twoCellState(0, 3)},
		{"swap", twoCellState(1, 3), SemanticAction{Kind: "swap", XRole: "c0", YRole: "c1"}, twoCellState(3, 1)},
		{"claim", twoCellState(0, 3), SemanticAction{Kind: "claim", XRole: "c0"}, twoCellState(1, 3)},
		{"release", twoCellState(1, 3), SemanticAction{Kind: "release", XRole: "c0"}, twoCellState(0, 3)},
		{"check-zero", twoCellState(0, 3), SemanticAction{Kind: "check", XRole: "c0", N: 0}, twoCellState(0, 3)},
		{"emit", twoCellState(0, 3), SemanticAction{Kind: "emit", Symbol: "ping"}, State{Cells: twoCellState(0, 3).Cells, Events: []string{"ping"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, outcome, err := Apply(test.state, test.action)
			if err != nil || outcome != "applied" {
				t.Fatalf("Apply: outcome=%q err=%v", outcome, err)
			}
			comparison, err := CompareStates(got, test.want)
			if err != nil || comparison != 0 {
				t.Fatalf("got %#v want %#v err=%v", got, test.want, err)
			}
		})
	}
}

func TestInapplicabilityDoesNotMutateState(t *testing.T) {
	state := twoCellState(0, 3)
	for _, action := range []SemanticAction{
		{Kind: "add", XRole: "c0", N: -1},
		{Kind: "transfer", XRole: "c0", YRole: "c1", N: 1},
		{Kind: "claim", XRole: "c1"},
		{Kind: "release", XRole: "c0"},
		{Kind: "check", XRole: "c0", N: 1},
		{Kind: "set", XRole: "c2", N: 0},
	} {
		got, outcome, err := Apply(state, action)
		if err != nil || outcome != "inapplicable" {
			t.Fatalf("%#v: outcome=%q err=%v", action, outcome, err)
		}
		comparison, _ := CompareStates(got, state)
		if comparison != 0 {
			t.Fatalf("inapplicable action changed state: %#v", action)
		}
	}
}

func TestEventOrderIsState(t *testing.T) {
	state := twoCellState(0, 0)
	one, _, _ := Apply(state, SemanticAction{Kind: "emit", Symbol: "a"})
	ab, _, _ := Apply(one, SemanticAction{Kind: "emit", Symbol: "b"})
	two, _, _ := Apply(state, SemanticAction{Kind: "emit", Symbol: "b"})
	ba, _, _ := Apply(two, SemanticAction{Kind: "emit", Symbol: "a"})
	comparison, err := CompareStates(ab, ba)
	if err != nil || comparison == 0 {
		t.Fatalf("event order collapsed: comparison=%d err=%v", comparison, err)
	}
}

func TestLocalFactsRoundTripUsesEmptyArrays(t *testing.T) {
	state := twoCellState(0, 1)
	facts, err := Facts(state, Occurrence{Action: SemanticAction{Kind: "set", XRole: "c0", N: 0}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := facts.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseLocalFacts(encoded)
	if err != nil {
		t.Fatalf("round trip %s: %v", encoded, err)
	}
	if parsed.ReadRoles == nil || len(parsed.ReadRoles) != 0 {
		t.Fatalf("empty read footprint was not an array: %#v", parsed.ReadRoles)
	}
}

func TestExecuteHistoryRunsOnlyTheSuppliedOrder(t *testing.T) {
	history := []Occurrence{
		{Action: SemanticAction{Kind: "set", XRole: "c0", N: 2}},
		{Action: SemanticAction{Kind: "add", XRole: "c0", N: 1}},
		{Action: SemanticAction{Kind: "emit", Symbol: "done"}},
	}
	states, terminal, err := ExecuteHistory(twoCellState(0, 0), history)
	if err != nil || terminal != "completed" || len(states) != 4 || states[3].Cells[0].Value != 3 || len(states[3].Events) != 1 {
		t.Fatalf("states=%#v terminal=%q err=%v", states, terminal, err)
	}
}

func twoCellState(a, b int) State {
	return State{Cells: []Cell{{Name: "c0", Value: a}, {Name: "c1", Value: b}}}
}
