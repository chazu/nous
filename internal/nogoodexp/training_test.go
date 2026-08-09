package nogoodexp

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/vocab/nogoods"
)

func TestOrdinaryHeuristicAcquisitionPromotesUniqueFullMask(t *testing.T) {
	run, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "promoted" || run.Artifact == "" {
		var diagnostics []int
		for _, name := range run.Store.Examples("NogoodCandidate") {
			if name != "NogoodCandidate" && run.Store.Get(name).GetBool("trainingExact") {
				diagnostics = append(diagnostics, run.Store.Get(name).GetInt("mask"))
			}
		}
		t.Fatalf("terminal/artifact = %q/%q exact=%v", run.Terminal, run.Artifact, diagnostics)
	}
	if run.TasksPopped <= 1 || run.TasksPopped >= TrainingTaskCap {
		t.Fatalf("tasks popped = %d", run.TasksPopped)
	}
	if len(run.MeterRecords) == 0 {
		t.Fatal("training emitted no verifier-owned meter records")
	}
	artifact := run.Store.Get(run.Artifact)
	if artifact == nil || artifact.GetInt("mask") != int(nogoods.FullMask) || !artifact.GetBool("frozen") || artifact.GetInt("promotionProofCount") != 24 {
		t.Fatalf("artifact = %#v", artifact)
	}

	var masks []int
	var exact []int
	for _, name := range run.Store.Examples("NogoodCandidate") {
		if name == "NogoodCandidate" {
			continue
		}
		candidate := run.Store.Get(name)
		masks = append(masks, candidate.GetInt("mask"))
		if candidate.GetBool("trainingExact") {
			exact = append(exact, candidate.GetInt("mask"))
		}
		if !candidate.GetBool("evidenceComplete") || candidate.GetInt("exampleCount") != 4 || len(candidate.GetStrings("evidenceUnits")) != 4 {
			t.Fatalf("incomplete candidate evidence %s", name)
		}
		if candidate.GetInt("barrierCount") != 4 {
			t.Fatalf("candidate %s sealed %d evidence barriers", name, candidate.GetInt("barrierCount"))
		}
	}
	slices.Sort(masks)
	if !slices.Equal(masks, []int{0, 1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("candidate masks = %v", masks)
	}
	if !slices.Equal(exact, []int{7}) {
		t.Fatalf("exact masks = %v", exact)
	}
	if got := len(run.Store.Examples("NogoodPromotionProof")) - 1; got != 24 {
		t.Fatalf("promotion proofs = %d", got)
	}
}

func TestTrainingIsDeterministicAtStoreBoundary(t *testing.T) {
	first, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	if first.TasksPopped != second.TasksPopped || first.Terminal != second.Terminal || first.Artifact != second.Artifact {
		t.Fatalf("runs differ: %#v != %#v", first, second)
	}
	if !slices.Equal(first.Store.All(), second.Store.All()) {
		t.Fatal("store unit identities differ between deterministic runs")
	}
}

func TestFreezeRefusesCorruptedPromotedStoreAuthority(t *testing.T) {
	for name, corrupt := range map[string]func(TrainingRun){
		"artifact-mask": func(run TrainingRun) { run.Store.Get(run.Artifact).Set("mask", 5) },
		"promotion-color": func(run TrainingRun) {
			proofs := run.Store.Get(run.Artifact).GetStrings("promotionProofs")
			promotionCase := run.Store.Get(run.Store.Get(proofs[0]).GetString("case"))
			promotionCase.Set("blocked", 99)
		},
	} {
		t.Run(name, func(t *testing.T) {
			run, err := RunTraining("../../domains")
			if err != nil {
				t.Fatal(err)
			}
			corrupt(run)
			if _, _, _, err := FreezeArtifact(run); err == nil {
				t.Fatal("corrupted training store minted artifact authority")
			}
		})
	}
}
