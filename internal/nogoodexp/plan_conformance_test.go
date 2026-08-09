package nogoodexp_test

import "testing"

func TestV2LiteralLedgerAndAttainability(t *testing.T) {
	// These literals deliberately import no production, fixture, baseline,
	// engine, or experiment code. A protocol change must alter this derivation.
	root := [12]int{0, 1, 2, 0, 2, 0, 0, 0, 0, 0, 0, 0}
	request := [12]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 26}
	matcher := [12]int{3, 20, 0, 0, 0, 0, 0, 10, 0, 0, 0, 12}
	completion := [12]int{0, 2, 0, 3, 0, 0, 0, 0, 1, 19, 0, 7}
	disposition := [12]int{0, 0, 1, 0, 1, 0, 0, 0, 0, 6, 1, 8}

	if sum(request) != 26 || sum(matcher) != 45 || sum(completion) != 32 || sum(disposition) != 17 {
		t.Fatalf("ledger row drift: request=%d matcher=%d completion=%d disposition=%d", sum(request), sum(matcher), sum(completion), sum(disposition))
	}
	proposal := add(root, request, matcher, completion, disposition)
	if sum(proposal) != 125 || sum(proposal) > 128 {
		t.Fatalf("proposal maximum = %d, want 125 within cap 128", sum(proposal))
	}
	noMatch := sum(request) + sum(matcher) + 10
	if noMatch != 81 || noMatch > 83 {
		t.Fatalf("zero-completion resume maximum = %d, want 81 within cap 83", noMatch)
	}
	attainability := 2000 + 78*((128-177)+(128-150)+(128-153)+(128-150)) + 72*83
	if attainability != -1228 || attainability >= 0 {
		t.Fatalf("hard-cap attainability = %d, want -1228", attainability)
	}
}

func sum(vector [12]int) int {
	total := 0
	for _, value := range vector {
		total += value
	}
	return total
}

func add(vectors ...[12]int) [12]int {
	var total [12]int
	for _, vector := range vectors {
		for index, value := range vector {
			total[index] += value
		}
	}
	return total
}
