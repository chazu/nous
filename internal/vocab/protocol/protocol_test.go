package protocol

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func handshakeRecords() []string {
	return []string{
		"trans:waiting,ack>established",
		"state:established",
		"event:begin",
		"state:idle",
		"accept:established",
		"event:ack",
		"state:waiting",
		"start:idle",
		"trans:idle,begin>waiting",
	}
}

func mustParse(t *testing.T, records []string) Machine {
	t.Helper()
	machine, err := Parse(records)
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func TestParseCanonicalizesAndDeduplicates(t *testing.T) {
	records := append(handshakeRecords(), "state:idle", "event:ack", "accept:established", "trans:idle,begin>waiting")
	machine := mustParse(t, records)
	want := []string{
		"state:established", "state:idle", "state:waiting",
		"event:ack", "event:begin",
		"start:idle", "accept:established",
		"trans:idle,begin>waiting", "trans:waiting,ack>established",
	}
	if got := machine.Records(); !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical records = %v, want %v", got, want)
	}
	again := mustParse(t, machine.Records())
	if !SameEncoding(machine, again) {
		t.Fatal("canonicalization is not idempotent")
	}
}

func TestParseRejectsMalformedRecords(t *testing.T) {
	tests := map[string][]string{
		"unknown tag":       {"state:a", "start:a", "wat:x"},
		"whitespace":        {"state:a b", "start:a"},
		"missing start":     {"state:a"},
		"duplicate start":   {"state:a", "start:a", "start:a"},
		"undeclared accept": {"state:a", "start:a", "accept:b"},
		"undeclared event":  {"state:a", "state:b", "start:a", "trans:a,x>b"},
		"conflict":          {"state:a", "state:b", "state:c", "event:x", "start:a", "trans:a,x>b", "trans:a,x>c"},
		"bad delimiter":     {"state:a", "state:b", "event:x", "start:a", "trans:a,x>b>c"},
	}
	for name, records := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(records); err == nil {
				t.Fatalf("Parse(%v) unexpectedly succeeded", records)
			}
		})
	}
}

func TestReachabilityTrapsAndTrim(t *testing.T) {
	records := append(handshakeRecords(),
		"state:orphan", "state:failed", "event:timeout",
		"trans:orphan,begin>orphan", "trans:waiting,timeout>failed", "trans:failed,timeout>failed",
	)
	machine := mustParse(t, records)
	if got, want := machine.ReachableStates(), []string{"established", "failed", "idle", "waiting"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reachable = %v, want %v", got, want)
	}
	if got, want := machine.RejectingTrapStates(), []string{"failed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("traps = %v, want %v", got, want)
	}
	trimmed := machine.TrimUnreachable()
	for _, record := range trimmed.Records() {
		if record == "state:orphan" || record == "trans:orphan,begin>orphan" {
			t.Fatalf("trim retained unreachable record %q", record)
		}
	}
	if !SameEncoding(trimmed, trimmed.TrimUnreachable()) {
		t.Fatal("trim is not idempotent")
	}
	if equivalent, witness := Compare(machine, trimmed); !equivalent {
		t.Fatalf("trim changed language; witness=%v", witness)
	}
}

func TestTraceAcceptance(t *testing.T) {
	machine := mustParse(t, handshakeRecords())
	if machine.Accepts(nil) {
		t.Fatal("empty trace should be rejected")
	}
	if !machine.Accepts([]string{"begin", "ack"}) {
		t.Fatal("handshake trace should be accepted")
	}
	for _, trace := range [][]string{{"begin"}, {"ack"}, {"begin", "unknown"}, {"begin", "ack", "ack"}} {
		if machine.Accepts(trace) {
			t.Fatalf("trace %v should be rejected", trace)
		}
	}
}

func TestCompareReturnsDeterministicShortestWitness(t *testing.T) {
	a := mustParse(t, handshakeRecords())
	b := mustParse(t, append([]string(nil),
		"state:idle", "state:waiting", "state:established",
		"event:ack", "event:begin", "start:idle", "accept:waiting",
		"trans:idle,begin>waiting", "trans:waiting,ack>established",
	))
	equivalent, witness := Compare(a, b)
	if equivalent || !reflect.DeepEqual(witness, []string{"begin"}) {
		t.Fatalf("Compare = (%v,%v), want (false,[begin])", equivalent, witness)
	}
	if a.Accepts(witness) == b.Accepts(witness) {
		t.Fatal("witness does not distinguish machines")
	}

	emptyAccept := mustParse(t, []string{"state:a", "start:a", "accept:a"})
	emptyReject := mustParse(t, []string{"state:a", "start:a"})
	if eq, got := Compare(emptyAccept, emptyReject); eq || len(got) != 0 {
		t.Fatalf("initial mismatch = (%v,%v), want false with empty witness", eq, got)
	}
}

func TestEquivalencePropertiesAndAlphaRename(t *testing.T) {
	a := mustParse(t, handshakeRecords())
	b := mustParse(t, []string{
		"state:s0", "state:s1", "state:s2", "event:ack", "event:begin",
		"start:s0", "accept:s2", "trans:s0,begin>s1", "trans:s1,ack>s2",
	})
	c := b.TrimUnreachable()
	for _, pair := range [][2]Machine{{a, a}, {a, b}, {b, a}, {b, c}, {a, c}} {
		if eq, witness := Compare(pair[0], pair[1]); !eq {
			t.Fatalf("expected equivalence, witness=%v", witness)
		}
	}
}

func TestUnreachableStructureAndInputOrderDoNotChangeLanguage(t *testing.T) {
	base := mustParse(t, handshakeRecords())
	variants := [][]string{
		append(append([]string(nil), handshakeRecords()...),
			"state:orphan", "event:loop", "trans:orphan,loop>orphan"),
		append(append([]string(nil), handshakeRecords()...),
			"state:u0", "state:u1", "accept:u1", "event:skip",
			"trans:u0,skip>u1", "trans:u1,skip>u0"),
	}
	for i, records := range variants {
		machine := mustParse(t, records)
		if equivalent, witness := Compare(base, machine); !equivalent {
			t.Fatalf("variant %d changed language; witness=%v", i, witness)
		}
	}

	reversed := append([]string(nil), handshakeRecords()...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	if got := mustParse(t, reversed).Records(); !reflect.DeepEqual(got, base.Records()) {
		t.Fatalf("record permutation changed canonical form: %v", got)
	}
}

func TestCompareDifferentialAgainstExhaustiveTinyMachines(t *testing.T) {
	var machines []Machine
	for code := 0; code < 16; code++ {
		machines = append(machines, tinyMachine(t, code, code%4))
	}
	for i, a := range machines {
		for j, b := range machines {
			equivalent, witness := Compare(a, b)
			bound := (len(a.States)+1)*(len(b.States)+1) - 1
			exhaustive, found := exhaustiveWitness(a, b, bound)
			if equivalent == found {
				t.Fatalf("pair (%d,%d): Compare=(%v,%v), exhaustive=%v", i, j, equivalent, witness, exhaustive)
			}
			if !equivalent && !reflect.DeepEqual(witness, exhaustive) {
				t.Fatalf("pair (%d,%d): witness=%v, want deterministic shortest %v", i, j, witness, exhaustive)
			}
		}
	}
}

func tinyMachine(t *testing.T, transitionCode, acceptingMask int) Machine {
	t.Helper()
	records := []string{"state:q0", "state:q1", "event:a", "event:b", "start:q0"}
	for state := 0; state < 2; state++ {
		if acceptingMask&(1<<state) != 0 {
			records = append(records, fmt.Sprintf("accept:q%d", state))
		}
	}
	code := transitionCode
	for state := 0; state < 2; state++ {
		for _, event := range []string{"a", "b"} {
			choice := code % 3
			code /= 3
			if choice != 0 {
				records = append(records, fmt.Sprintf("trans:q%d,%s>q%d", state, event, choice-1))
			}
		}
	}
	return mustParse(t, records)
}

func exhaustiveWitness(a, b Machine, maxDepth int) ([]string, bool) {
	alphabetSet := map[string]bool{}
	for _, event := range append(append([]string(nil), a.Events...), b.Events...) {
		alphabetSet[event] = true
	}
	var alphabet []string
	for event := range alphabetSet {
		alphabet = append(alphabet, event)
	}
	sort.Strings(alphabet)
	queue := [][]string{nil}
	for len(queue) > 0 {
		trace := queue[0]
		queue = queue[1:]
		if a.Accepts(trace) != b.Accepts(trace) {
			return trace, true
		}
		if len(trace) == maxDepth {
			continue
		}
		for _, event := range alphabet {
			next := append(append([]string(nil), trace...), event)
			queue = append(queue, next)
		}
	}
	return nil, false
}
