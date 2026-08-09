package transformexp

import "testing"

func TestSixPoliciesExposeExpectedSemanticControls(t *testing.T) {
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 841300+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		results := map[Policy]PolicyOutcome{}
		for _, policy := range empiricalPolicies {
			result, err := executePolicy("../../domains", c, policy)
			if err != nil {
				t.Fatalf("family %d policy %s: %v", family, policy, err)
			}
			results[policy] = result
		}
		if got := results[NousRefine]; got.Terminal != "completed" || got.HeldoutCorrect != 8 || got.FalseApplications != 0 {
			t.Fatalf("family %d nous=%+v", family, got)
		}
		nous := results[NousRefine]
		if len(nous.Transcript.Raw) == 0 || !nous.HeldoutStoreUnchanged {
			t.Fatalf("family %d nous evidence/store boundary=%+v", family, nous)
		}
		if _, err := reduceTransformTranscript(nous.Transcript.Raw, policyManifestDigest(c, NousRefine)); err != nil {
			t.Fatalf("family %d nous transcript: %v", family, err)
		}
		if got := results[PositiveLGG]; got.Terminal != "completed" || got.HeldoutCorrect >= 8 || got.FalseApplications == 0 {
			t.Fatalf("family %d lgg=%+v", family, got)
		}
		if got := results[ConcreteReplay]; got.HeldoutCorrect != 4 || got.FalseApplications != 0 {
			t.Fatalf("family %d replay=%+v", family, got)
		}
		for _, policy := range []Policy{BoundedPBE, RandomPBE} {
			got := results[policy]
			if got.Applications > 48 || got.Terminal == "completed" && got.HeldoutCorrect != 8 || got.Terminal == "budget-exhausted" && (got.Applications != 40 || got.HeldoutCorrect != 0) {
				t.Fatalf("family %d %s=%+v", family, policy, got)
			}
			if len(got.Transcript.Raw) == 0 || got.Transcript.Work != int64(got.TrainingWork) {
				t.Fatalf("family %d %s lacks authoritative transcript: %+v", family, policy, got)
			}
			if _, err := reduceTransformTranscript(got.Transcript.Raw, policyManifestDigest(c, policy)); err != nil {
				t.Fatalf("family %d %s transcript: %v", family, policy, err)
			}
		}
		if family == 0 || family == 2 || family == 4 || family == 6 || family == 8 {
			if got := results[NoEqualityGuard]; got.Terminal != "completed" || got.HeldoutCorrect != 8 {
				t.Fatalf("family %d ablation=%+v", family, got)
			}
		} else if got := results[NoEqualityGuard]; got.Terminal == "completed" {
			t.Fatalf("family %d ablation unexpectedly completed: %+v", family, got)
		}
	}
}
