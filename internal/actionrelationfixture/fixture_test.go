package actionrelationfixture

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestDrawBlockHasExactFrozen66RowSchedule(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 5, CurriculumSeed: 851006, Attempt: 0}
	block, err := PrecommitDraws(context)
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Draws) != 66 || block.Draws[0].Namespace != "skeleton-variant" || block.Draws[3].Namespace != "cell-permutation" || block.Draws[3].Index != 2 || block.Draws[10].Namespace != "store-preoccupation-count" || block.Draws[65].Index != 5 {
		t.Fatalf("draw schedule endpoints=%+v %+v", block.Draws[0], block.Draws[65])
	}
	again, _ := PrecommitDraws(context)
	if block.Root != again.Root || !bytes.Equal(block.Draws[37].Canonical, again.Draws[37].Canonical) {
		t.Fatal("draw schedule is not deterministic")
	}
	if _, err := PrecommitDraws(DrawContext{Panel: "development", Authority: "wrong", Curriculum: 5, CurriculumSeed: 851006}); err == nil {
		t.Fatal("accepted wrong public authority")
	}
}

func TestCurriculumUsesEveryStratumTwiceAndRenormalizes(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 7, CurriculumSeed: 851008, Attempt: 0}
	curriculum, err := BuildCurriculum(context)
	if err != nil {
		t.Fatal(err)
	}
	if curriculum.Family != 7 || curriculum.WithinFamilyOrdinal != 0 || len(curriculum.Worlds) != 6 {
		t.Fatalf("curriculum shape=%+v", curriculum)
	}
	counts := map[string]int{}
	digests := map[string]bool{}
	for _, view := range curriculum.Worlds {
		counts[view.Stratum]++
		if digests[view.Core.Digest] {
			t.Fatalf("duplicate semantic core %s", view.Core.Digest)
		}
		digests[view.Core.Digest] = true
		normalized, err := (actionrelations.World{State: view.State, Actions: view.Actions}).Normalize()
		if err != nil {
			t.Fatal(err)
		}
		wire, _ := normalized.CanonicalJSON()
		if !bytes.Equal(wire, view.Core.Canonical) || len(view.CellPermutation) != 3 || len(view.ActionPermutation) != 6 {
			t.Fatal("presentation permutation changed semantic authority")
		}
	}
	for _, stratum := range []string{actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.Neutral, actionrelationfixturecore.Adverse} {
		if counts[stratum] != 2 {
			t.Fatalf("stratum %s count=%d", stratum, counts[stratum])
		}
	}
}
