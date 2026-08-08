package causalrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

func artifactName(kind, digest string) string { return "Causal." + kind + "." + digest }

func artifactRequestKey(profile, scope string, step int, kind, semanticKey string) string {
	return fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s", profile, scope, step, kind, semanticKey)
}

func (r *Runner) allocate(step int, kind string, payload any) (artifactRef, error) {
	if !r.cueMaterializing && !r.driverMaterializing {
		return artifactRef{}, errors.New("causal evidence materialization requires CUE task or narrow driver authority")
	}
	if r.driverMaterializing && !r.cueMaterializing {
		initial := step == 0 && ((kind == "observation" && len(r.artifacts) == 0) ||
			(kind == "posterior" && len(r.artifacts) == 1) ||
			(kind == "descriptor-snapshot" && len(r.artifacts) == 2 && r.cursor.latestSnapshotDigest == ""))
		responseCandidate := kind == "result" && r.cursor.state == StateAwaitingTeacher && r.authorization.Digest != ""
		if !initial && !responseCandidate {
			return artifactRef{}, fmt.Errorf("driver is not authorized to materialize %q", kind)
		}
	}
	semanticKey, err := causalv2.SemanticKey(kind, payload)
	if err != nil {
		return artifactRef{}, err
	}
	requestKey := artifactRequestKey(r.profile.ProfileDigest, r.episodeKey, step, kind, semanticKey)
	if prior, ok := r.byRequest[requestKey]; ok {
		existing := r.store.Get(artifactName(prior.Kind, prior.Digest))
		if existing == nil || !existing.GetBool("sealed") || existing.GetString("artifactBytes") != string(prior.Canonical) {
			return artifactRef{}, errors.New("identical causal allocation collides with missing, unsealed, or differing evidence")
		}
		return prior, nil
	}
	artifact, err := causalv2.NewArtifact(
		r.profile.ProfileDigest,
		r.episodeKey,
		step,
		kind,
		payload,
		len(r.artifacts),
	)
	if err != nil {
		return artifactRef{}, err
	}
	canonical, err := causalv2.CanonicalJSON(artifact)
	if err != nil {
		return artifactRef{}, err
	}
	name := artifactName(kind, artifact.ArtifactDigest)
	if existing := r.store.Get(name); existing != nil {
		if !existing.GetBool("sealed") || existing.GetString("artifactBytes") != string(canonical) {
			return artifactRef{}, fmt.Errorf("occupied causal artifact name %q", name)
		}
		return artifactRef{}, errors.New("unexpected duplicate artifact allocation")
	}
	if err := r.meter.chargeArtifact(1); err != nil {
		return artifactRef{}, err
	}
	if err := r.meter.chargeUnit(1); err != nil {
		return artifactRef{}, err
	}
	unitArtifact := unit.New(name)
	unitArtifact.Set("isA", []string{"CausalArtifact", "Anything"})
	unitArtifact.Set("sealed", true)
	unitArtifact.Set("artifactBytes", string(canonical))
	unitArtifact.Set("artifactDigest", artifact.ArtifactDigest)
	unitArtifact.Set("semanticKey", artifact.SemanticKey)
	unitArtifact.Set("kind", artifact.Kind)
	unitArtifact.Set("scope", artifact.Scope)
	unitArtifact.Set("step", artifact.Step)
	unitArtifact.Set("chargeIndex", artifact.ChargeIndex)
	r.store.Put(unitArtifact)
	ref := artifactRef{
		Kind: kind, Digest: artifact.ArtifactDigest, SemanticKey: artifact.SemanticKey,
		Canonical: canonical, ChargeIndex: artifact.ChargeIndex, MeterAfter: r.meter.Counts(),
	}
	r.artifacts = append(r.artifacts, ref)
	r.byDigest[ref.Digest] = ref
	r.byRequest[requestKey] = ref
	return ref, nil
}

func (r *Runner) decodePayload(ref artifactRef, target any) error {
	stored := r.store.Get(artifactName(ref.Kind, ref.Digest))
	if stored == nil || !stored.GetBool("sealed") || stored.GetString("artifactBytes") != string(ref.Canonical) {
		return errors.New("artifact store copy is absent, unsealed, or corrupt")
	}
	artifact, err := causalv2.VerifyArtifact(ref.Canonical)
	if err != nil {
		return err
	}
	if artifact.ArtifactDigest != ref.Digest || artifact.Kind != ref.Kind {
		return errors.New("artifact reference does not match envelope")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifact.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if decoder.Decode(&extra) == nil {
		return errors.New("artifact payload has trailing value")
	}
	canonical, err := causalv2.CanonicalJSON(target)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, artifact.Payload) {
		return errors.New("artifact payload is not exact typed canonical JSON")
	}
	return nil
}

func (r *Runner) materializePosterior(step int, hypotheses []string) (artifactRef, error) {
	digest, err := hypothesisSetDigest(hypotheses)
	if err != nil {
		return artifactRef{}, err
	}
	return r.allocate(step, "posterior", posteriorPayload{
		Hypotheses: append([]string(nil), hypotheses...), SemanticSetDigest: digest,
	})
}

func hypothesisSetDigest(hypotheses []string) (string, error) {
	if err := validatePosterior(hypotheses, 1, causalv2.PreregisteredManifest().MaximumPool); err != nil {
		return "", err
	}
	return causalv2.Digest("causal-hypothesis-set/v2", hypotheses)
}

func canonicalStringSetDigest(items []string) (string, error) {
	canonical := append([]string(nil), items...)
	for index, item := range canonical {
		if _, err := causal.Parse(item); err != nil {
			return "", fmt.Errorf("set item %d: %w", index, err)
		}
	}
	sort.Strings(canonical)
	for index := 1; index < len(canonical); index++ {
		if canonical[index] == canonical[index-1] {
			return "", errors.New("duplicate causal set item")
		}
	}
	return causalv2.Digest("causal-hypothesis-set/v2", canonical)
}

func partitionDigest(cells any) (string, error) {
	return causalv2.Digest("causal-partition/v2", cells)
}

func (r *Runner) materializeSnapshot(state State) error {
	if state != StateReady && state != StateAwaitingTeacher && state != StateTerminal {
		return fmt.Errorf("invalid sealed snapshot state %q", state)
	}
	previous := r.cursor.latestSnapshotDigest
	if previous == "" {
		previous = causalv2.ZeroDigest
	} else if err := r.meter.chargeProfile(14); err != nil {
		return err
	}
	evaluations, work, cycles, units := r.meter.remaining()
	// The snapshot itself is the next materialization and attributed unit.
	evaluationsAfter := evaluations
	workAfter := work - 1
	cyclesAfter := cycles
	unitsAfter := units - 1
	if workAfter < 0 || unitsAfter < 0 {
		return errors.New("snapshot would exceed episode budget")
	}
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	ref, err := r.allocate(r.step, "descriptor-snapshot", descriptorSnapshotPayload{
		PreviousSnapshotArtifactDigest: previous,
		State:                          string(state), Aliases: append([]string(nil), r.fixture.Aliases...),
		Costs:                          append([]int(nil), r.fixture.Costs...),
		Presentation:                   append([]int(nil), r.fixture.Presentation...),
		InitialPosteriorArtifactDigest: r.initialPosteriorArtifact.Digest,
		PosteriorDigest:                posteriorDigest, AcquisitionCode: r.profile.AcquisitionCode,
		TotalCost: r.totalCost, ActionCount: len(r.consumed),
		RemainingEvaluations: evaluationsAfter, RemainingWork: workAfter,
		RemainingCycles: cyclesAfter, RemainingUnits: unitsAfter,
	})
	if err != nil {
		return err
	}
	r.cursor.latestSnapshotDigest = ref.Digest
	r.cursor.state = state
	r.syncCursor()
	return nil
}

// ArtifactBytes returns a detached canonical copy in gap-free charge order.
func (r *Runner) ArtifactBytes() [][]byte {
	result := make([][]byte, len(r.artifacts))
	for index, artifact := range r.artifacts {
		result[index] = append([]byte(nil), artifact.Canonical...)
	}
	return result
}
