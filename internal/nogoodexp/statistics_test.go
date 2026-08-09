package nogoodexp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestDevelopmentRandomizationGoldenDraws(t *testing.T) {
	encoded, err := json.Marshal([]any{"part3/nogoods/v1", "development", 832001, 0, "randomization/nous-vs-mac"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	seeds := statisticsSeeds("development", 832001, 0, "randomization/nous-vs-mac")
	if hex.EncodeToString(digest[:]) != "4c310ed3e073097cc03302dcdc59eeb3c8eca9b31b23ded25c70e58481abca34" || seeds != [2]uint64{5490185723907869052, 13849416426707349171} {
		t.Fatalf("replicate-zero authority drifted: %x/%v", digest, seeds)
	}
	rng := statisticsStream("development", 832001, 0, "randomization/nous-vs-mac")
	got := make([]uint64, 8)
	for index := range got {
		got[index] = rng.Uint64N(2)
	}
	want := []uint64{0, 1, 1, 0, 1, 0, 0, 0}
	if !slices.Equal(got, want) {
		t.Fatalf("replicate-zero draws = %v, want %v", got, want)
	}
}

func TestLockedStatisticsAuthorityRetainsSeedsWithoutRoot(t *testing.T) {
	root := "0000000000000000000000000000000000000000000000000000000000000000"
	authority := lockedStatisticsAuthority(root)
	if authority.root != nil || authority.panel != "locked" || len(authority.seeds) != 2 {
		t.Fatalf("locked authority retained root or incomplete seeds: %#v", authority)
	}
	for _, purpose := range []string{"bootstrap/nous-vs-mac", "randomization/nous-vs-mac"} {
		for _, replicate := range []int{0, 9999} {
			got := authority.stream(replicate, purpose).Uint64()
			want := statisticsStream("locked", root, replicate, purpose).Uint64()
			if got != want {
				t.Fatalf("%s replicate %d seed authority mismatch", purpose, replicate)
			}
		}
	}
}

func TestInferenceRejectsSignedOverflow(t *testing.T) {
	execution := PanelExecution{Panel: "development", Role: "primary", AcquisitionWork: math.MaxInt64}
	for _, policy := range RequiredPolicies {
		execution.Policies = append(execution.Policies, PolicyExecution{Policy: policy})
	}
	learnedIndex, macIndex := slices.Index(RequiredPolicies, "nous-generalized"), slices.Index(RequiredPolicies, "mac-cbj")
	for ordinal := 0; ordinal < 96; ordinal++ {
		cohort := "reusable"
		if ordinal >= 88 {
			cohort = "independent-unsat"
		} else if ordinal >= 80 {
			cohort = "irrelevant"
		} else if ordinal >= 56 {
			cohort = "near-miss"
		}
		execution.Policies[macIndex].Tasks = append(execution.Policies[macIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 1})
		execution.Policies[learnedIndex].Tasks = append(execution.Policies[learnedIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 2})
	}
	if _, err := InferDevelopment(execution); err == nil {
		t.Fatal("overflowing inference input was accepted")
	}
}

func TestInferenceIsDeterministicAndUsesAllFrozenStrata(t *testing.T) {
	execution := PanelExecution{Panel: "development", Role: "primary", AcquisitionWork: 500}
	for _, policy := range RequiredPolicies {
		execution.Policies = append(execution.Policies, PolicyExecution{Policy: policy})
	}
	learnedIndex, macIndex := slices.Index(RequiredPolicies, "nous-generalized"), slices.Index(RequiredPolicies, "mac-cbj")
	for ordinal := 0; ordinal < 96; ordinal++ {
		cohort := "reusable"
		if ordinal >= 88 {
			cohort = "independent-unsat"
		} else if ordinal >= 80 {
			cohort = "irrelevant"
		} else if ordinal >= 56 {
			cohort = "near-miss"
		}
		mac := TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 200, PruneSound: true}
		learnedWork := int64(125)
		if cohort != "reusable" {
			learnedWork = 215
		}
		learned := TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: learnedWork, PruneSound: true}
		execution.Policies[macIndex].Tasks = append(execution.Policies[macIndex].Tasks, mac)
		execution.Policies[learnedIndex].Tasks = append(execution.Policies[learnedIndex].Tasks, learned)
	}
	first, err := InferDevelopment(execution)
	if err != nil {
		t.Fatal(err)
	}
	second, err := InferDevelopment(execution)
	if err != nil || first != second {
		t.Fatalf("inference not deterministic: %#v %#v %v", first, second, err)
	}
	if first.Point.Denominator != 96*200 || first.RandomizationP.Denominator != 10001 {
		t.Fatalf("inference denominators = %#v", first)
	}
}
