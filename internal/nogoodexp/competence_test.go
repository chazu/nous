package nogoodexp

import "testing"

func TestCompetencePanelsRejectEveryFrozenNearMissAndCorruption(t *testing.T) {
	for _, panel := range []string{"development", "validation"} {
		t.Run(panel, func(t *testing.T) {
			execution, err := RunCompetence("../../domains", panel)
			if err != nil {
				t.Fatal(err)
			}
			wantCount := 8
			if panel == "validation" {
				wantCount = 16
			}
			if len(execution.Outcomes) != wantCount || execution.Artifact == "" {
				t.Fatalf("competence execution = %#v", execution)
			}
			for _, outcome := range execution.Outcomes {
				if (outcome.Kind == "duplicate-completion" || outcome.Kind == "cross-decision" || outcome.Kind == "stale-target") && !outcome.CorruptionRejected {
					t.Fatalf("corruption case passed: %#v", outcome)
				}
			}
		})
	}
}
