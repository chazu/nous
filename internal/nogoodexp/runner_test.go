package nogoodexp

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func TestCompletePolicyMatrixOnUtilitySmokePanel(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
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

func TestMACCBJDevelopmentTranscriptIsDeterministic(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	run := func() []TranscriptEvent {
		var events []TranscriptEvent
		for _, task := range tasks {
			_, taskEvents, runErr := runPolicyTask("../../domains", "mac-cbj", task, FrozenArtifact{}, ArtifactAuthority{}, nil)
			if runErr != nil {
				t.Fatal(runErr)
			}
			events = appendEvents(events, taskEvents)
		}
		return events
	}
	left, err := EncodeTranscript(run())
	if err != nil {
		t.Fatal(err)
	}
	right, err := EncodeTranscript(run())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left.Raw, right.Raw) {
		offset := firstDifferentByte(left.Raw, right.Raw)
		leftStart := transcriptHeaderSize + int(binary.BigEndian.Uint64(left.Raw[8:16]))
		rightStart := transcriptHeaderSize + int(binary.BigEndian.Uint64(right.Raw[8:16]))
		leftRecord := leftStart + ((offset-leftStart)/transcriptRecordSize)*transcriptRecordSize
		rightRecord := rightStart + ((offset-rightStart)/transcriptRecordSize)*transcriptRecordSize
		leftID := binary.BigEndian.Uint32(left.Raw[leftRecord+20 : leftRecord+24])
		rightID := binary.BigEndian.Uint32(right.Raw[rightRecord+20 : rightRecord+24])
		t.Fatalf("%s variable=%s/%s", describeTranscriptDifference(left.Raw, right.Raw, offset), left.Dictionary[leftID-1], right.Dictionary[rightID-1])
	}
}
