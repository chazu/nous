package nogoodexp

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func TestCompletePolicyMatrixOnUtilitySmokePanel(t *testing.T) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	// One case from each cohort exercises prune, near-miss continuation,
	// irrelevant continuation, and independent unsatisfiability.
	smoke := []nogoodfixture.Task{tasks[0], tasks[56], tasks[80], tasks[88]}
	execution, err := runPanelExecution("../../domains", "primary", "development-smoke", smoke)
	if err != nil {
		t.Fatal(err)
	}
	if len(execution.Policies) != len(RequiredPolicies) || execution.AcquisitionWork <= 0 || execution.AcquisitionWork > 2000 {
		t.Fatalf("execution shape/acquisition = %d/%d", len(execution.Policies), execution.AcquisitionWork)
	}
	for _, policy := range execution.Policies {
		if !slices.Contains(RequiredPolicies, policy.Policy) || len(policy.Tasks) != len(smoke) {
			t.Fatalf("policy shape = %s/%d", policy.Policy, len(policy.Tasks))
		}
		if _, err := DecodeTranscript(policy.Transcript.Raw); err != nil {
			t.Fatalf("%s transcript: %v", policy.Policy, err)
		}
		for _, outcome := range policy.Tasks {
			if !outcome.PruneSound || outcome.Work <= 0 {
				t.Fatalf("%s outcome = %#v", policy.Policy, outcome)
			}
		}
	}
	learned := execution.Policies[slices.Index(RequiredPolicies, "nous-generalized")]
	if learned.Tasks[0].Disposition != "propose-prune" || learned.Tasks[0].Work != 125 {
		t.Fatalf("learned reusable = %#v", learned.Tasks[0])
	}
	for _, outcome := range learned.Tasks[1:] {
		if outcome.Disposition != "resume" {
			t.Fatalf("learned control disposition = %#v", outcome)
		}
	}
	noArtifact := execution.Policies[slices.Index(RequiredPolicies, "no-artifact")]
	reset := execution.Policies[slices.Index(RequiredPolicies, "reset")]
	for index := range smoke {
		if got, want := reset.Tasks[index].Work, noArtifact.Tasks[index].Work+54; got != want {
			t.Fatalf("reset task %d work = %d, want fresh-profile work %d", smoke[index].Ordinal, got, want)
		}
	}
}
