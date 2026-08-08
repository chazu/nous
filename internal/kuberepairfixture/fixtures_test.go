package kuberepairfixture

import (
	"testing"

	"github.com/chazu/nous/internal/kuberepairoracle"
)

func TestTrainingCasesHaveUniqueOneEditSolutions(t *testing.T) {
	cases, err := Training(741001)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("training cases = %d", len(cases))
	}
	for _, caseData := range cases {
		analysis, err := kuberepairoracle.Analyze(caseData.Public, kuberepairoracle.Intent{DesiredPods: caseData.Intent.DesiredPods, BackendPort: caseData.Intent.BackendPort, ReadinessPorts: caseData.Intent.ReadinessPorts, ProtectedDigest: caseData.Intent.ProtectedDigest}, 3)
		if err != nil {
			t.Fatal(err)
		}
		result := analysis.Result
		if result.Terminal != "solution" || result.MinimumLength != 1 || len(result.Plans) != 1 {
			for index, edit := range caseData.Edits {
				t.Logf("edit[%d]=%s", index, edit)
			}
			t.Fatalf("case %s oracle = %#v", caseData.ID, result)
		}
	}
}

func TestRecompositionCasesUseTwoOrThreeEdits(t *testing.T) {
	for _, mask := range []int{FaultTemplate | FaultService, FaultTemplate | FaultExtraSelector, FaultService | FaultExtraSelector, FaultTemplate | FaultService | FaultExtraSelector} {
		caseData, err := Recomposition(743001+int64(mask), mask)
		if err != nil {
			t.Fatal(err)
		}
		analysis, err := kuberepairoracle.Analyze(caseData.Public, kuberepairoracle.Intent{DesiredPods: caseData.Intent.DesiredPods, BackendPort: caseData.Intent.BackendPort, ReadinessPorts: caseData.Intent.ReadinessPorts, ProtectedDigest: caseData.Intent.ProtectedDigest}, 3)
		if err != nil {
			t.Fatal(err)
		}
		result := analysis.Result
		want := 2
		if mask == FaultTemplate|FaultService|FaultExtraSelector {
			want = 3
		}
		if result.Terminal != "solution" || result.MinimumLength != want {
			t.Fatalf("mask %d oracle = %#v", mask, result)
		}
	}
}

func TestControlCasesHavePinnedOracleClasses(t *testing.T) {
	for _, testCase := range []struct {
		build    func() (Case, error)
		terminal string
		minimum  int
	}{
		{func() (Case, error) { return CrossRole(743101) }, "solution", 1},
		{func() (Case, error) { return Unrelated(743201, false) }, "solution", 1},
		{func() (Case, error) { return Unrelated(743301, true) }, "no-solution", 4},
	} {
		caseData, err := testCase.build()
		if err != nil {
			t.Fatal(err)
		}
		analysis, err := kuberepairoracle.Analyze(caseData.Public, kuberepairoracle.Intent{DesiredPods: caseData.Intent.DesiredPods, BackendPort: caseData.Intent.BackendPort, ReadinessPorts: caseData.Intent.ReadinessPorts, ProtectedDigest: caseData.Intent.ProtectedDigest}, 3)
		if err != nil {
			t.Fatal(err)
		}
		result := analysis.Result
		if result.Terminal != testCase.terminal || result.MinimumLength != testCase.minimum {
			t.Fatalf("case %s = %#v", caseData.ID, result)
		}
	}
}
