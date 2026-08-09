package nogoodexp

import (
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func learnedArtifact(t *testing.T) (FrozenArtifact, ArtifactAuthority) {
	t.Helper()
	training, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	artifact, encoded, authority, err := FreezeArtifact(training)
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
	return decoded, authority
}

func TestLearnedArtifactPrunesOnlyReusableDevelopmentCases(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, &artifact, &authority)
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
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if disposition.Status != "resume" || disposition.TasksPopped != 1 {
			t.Fatalf("empty artifact disposition = %#v", disposition)
		}
	}
}

func TestParsedOrMutatedArtifactCannotSelfAuthorize(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	parsedOnly := artifact
	result, err := ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &parsedOnly, nil)
	if err != nil || result.Status != "resume" {
		t.Fatalf("parsed-only artifact = %s, %v", result.Status, err)
	}
	mutated := artifact
	mutated.Mask = 5
	mutated.Digest = artifactDigest(mutated)
	result, err = ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &mutated, &authority)
	if err != nil || result.Status != "resume" {
		t.Fatalf("mutated artifact = %s, %v", result.Status, err)
	}
	forged := artifact
	forged.PromotionProofs[1] = forged.PromotionProofs[0]
	forged.Digest = artifactDigest(forged)
	if err := forged.Validate(); err == nil {
		t.Fatal("duplicate self-hashed proof set was accepted")
	}
}
