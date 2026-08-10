package actionrelationscore

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
)

func TestScorerRowsAreClosedCanonicalKind32And33Objects(t *testing.T) {
	child, err := actionrelationexp.BuildOperationRange("00112233445566778899aabbccddeeff", 2, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	world, err := BuildWorldPolicyRow(WorldPolicyRow{
		Panel: "development", Curriculum: 0, Family: 0, WorldOrdinal: 0, Stratum: "positive-effect",
		WorldDigest: testDigest("world"), Policy: actionrelationsearch.Complete, SearchTerminal: "completed",
		UtilityWorkVector: [12]int{1}, UtilityTotal: 1, TerminalSetDigest: testDigest("terminals"),
		WorkTerminalDigestOrZero: zeroDigest, BehaviorEqual: true, BudgetRemaining: LifecycleCap - 1,
		OperationRoot: child,
	})
	if err != nil {
		t.Fatal(err)
	}
	context := testDigest("context")
	concat, err := actionrelationexp.BuildOperationConcat(context, []string{child.Digest})
	if err != nil {
		t.Fatal(err)
	}
	curriculum, err := BuildCurriculumPolicyRow(CurriculumPolicyRow{
		Panel: "development", Curriculum: 0, Family: 0, Policy: actionrelationsearch.Complete,
		AcquisitionTerminal: "not-applicable", ArtifactDigest: zeroDigest, AcquisitionWorkTerminalDigestOrZero: zeroDigest,
		WorldRowDigests:   []string{world.Digest, testDigest("w1"), testDigest("w2"), testDigest("w3"), testDigest("w4"), testDigest("w5")},
		AggregateTerminal: "completed", SearchWorkVector: [12]int{1}, SearchTotal: 1,
		LifecycleWorkVector: [12]int{1}, LifecycleTotal: 1, BehaviorEqual: true, BudgetRemaining: LifecycleCap - 1,
		OperationRoot: concat,
	})
	if err != nil {
		t.Fatal(err)
	}
	if actionrelationexp.ValidateObject(32, world.Canonical) != nil || actionrelationexp.ValidateObject(33, curriculum.Canonical) != nil {
		t.Fatal("scorer rows did not enter their frozen object kinds")
	}
	corrupt := world
	corrupt.Canonical = bytes.Clone(world.Canonical)
	corrupt.Canonical[len(corrupt.Canonical)-2] ^= 1
	if VerifyWorldPolicyRow(corrupt) == nil {
		t.Fatal("accepted corrupted world row")
	}
}

func testDigest(label string) string { return digest([]byte(label)) }
