package transformexp

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
)

func TestCommittedResultsReconstructEveryPolicyScoreAndArtifact(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range empiricalPolicies {
		t.Run(string(policy), func(t *testing.T) {
			view, err := decodePolicyView(c)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := executePolicy("../../domains", view, c.Ordinal, policy)
			if err != nil {
				t.Fatal(err)
			}
			heldout, err := decodeHeldoutInputs(c)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err = executeHeldoutInputs(view, heldout, outcome)
			if err != nil {
				t.Fatal(err)
			}
			scorer, err := decodeScorerView(c)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err = scorePolicyOutcome(scorer, outcome)
			if err != nil {
				t.Fatal(err)
			}
			results, err := reconstructHeldoutResults(outcome.Transcript.Raw, outcome.Transcript.Objects, heldout)
			if err != nil {
				t.Fatal(err)
			}
			scorerBytes, _ := scorerFixtureBytes(c)
			reconstructed, err := scoreCommittedHeldout(results, scorerBytes, outcome.Terminal)
			if err != nil {
				t.Fatal(err)
			}
			wantWork := outcome.NonmatchingWork
			if outcome.Terminal != "completed" {
				wantWork = LifecycleWorkCap
			}
			if reconstructed.Bits != hex.EncodeToString([]byte{outcome.HeldoutCorrectBits}) || reconstructed.FalseApplications != outcome.FalseApplications || reconstructed.NonmatchingWork != wantWork {
				t.Fatalf("reconstructed score=%+v outcome=%+v", reconstructed, outcome)
			}
			artifact, err := reconstructArtifactDigest(outcome.Transcript.Raw, outcome.Transcript.Objects, policy, outcome.Terminal)
			if err != nil {
				t.Fatal(err)
			}
			wantArtifact := ""
			if len(outcome.Schema) != 0 {
				wantArtifact = digestBytes(outcome.Schema)
			}
			if artifact != wantArtifact {
				t.Fatalf("artifact=%s want=%s", artifact, wantArtifact)
			}
			again, err := reconstructHeldoutResults(bytes.Clone(outcome.Transcript.Raw), outcome.Transcript.Objects, heldout)
			if err != nil || !bytes.Equal(results, again) {
				t.Fatal("heldout commitment is not deterministic")
			}
		})
	}
}

func TestCommittedResultsRejectHeldoutInputPermutation(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	view, err := decodePolicyView(c)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := executePolicy("../../domains", view, c.Ordinal, NousRefine)
	if err != nil {
		t.Fatal(err)
	}
	heldout, err := decodeHeldoutInputs(c)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err = executeHeldoutInputs(view, heldout, outcome)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := transformfixturecore.ParseHeldout(heldout)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Cases[0].Before, fixture.Cases[1].Before = fixture.Cases[1].Before, fixture.Cases[0].Before
	permuted, err := fixture.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reconstructHeldoutResults(outcome.Transcript.Raw, outcome.Transcript.Objects, permuted); err == nil {
		t.Fatal("accepted heldout results whose application inputs belong to different committed tokens")
	}
}
