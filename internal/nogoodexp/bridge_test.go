package nogoodexp

import (
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func learnedArtifact(t *testing.T) FrozenArtifact {
	t.Helper()
	training, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	artifact, encoded, err := FreezeArtifact(training)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseFrozenArtifact(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Digest != artifact.Digest {
		t.Fatal("artifact digest changed across freeze/load")
	}
	return decoded
}

func TestLearnedArtifactPrunesOnlyReusableDevelopmentCases(t *testing.T) {
	artifact := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, &artifact)
		if err != nil {
			t.Fatalf("task %d (%s): %v", task.Ordinal, task.Cohort, err)
		}
		want := "resume"
		if task.Cohort == nogoodfixture.Reusable {
			want = "propose-prune"
		}
		if disposition.Status != want {
			t.Fatalf("task %d (%s) disposition = %s, want %s", task.Ordinal, task.Cohort, disposition.Status, want)
		}
		if disposition.TasksPopped != 1 {
			t.Fatalf("task %d popped %d bridge tasks", task.Ordinal, disposition.TasksPopped)
		}
	}
}

func TestEmptyArtifactUsesSameBridgeAndNeverPrunes(t *testing.T) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks[:4] {
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, nil)
		if err != nil {
			t.Fatal(err)
		}
		if disposition.Status != "resume" || disposition.TasksPopped != 1 {
			t.Fatalf("empty artifact disposition = %#v", disposition)
		}
	}
}
