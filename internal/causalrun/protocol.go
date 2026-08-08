package causalrun

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

// verifyProposalBoundary is fresh and read-only. It derives authorization
// bytes but does not materialize them or mutate the runner/store.
func verifyProposalBoundary(r *Runner) (causalv2.Authorization, error) {
	return verifyProposalArtifacts(r, StateProposing, true)
}

func verifyProposalArtifacts(r *Runner, expected State, requireNoDescendants bool) (causalv2.Authorization, error) {
	if r.cursor.state != expected {
		return causalv2.Authorization{}, fmt.Errorf("proposal verifier state=%q", r.cursor.state)
	}
	if r.agenda.Len() != 0 {
		return causalv2.Authorization{}, errors.New("proposal boundary has pending task")
	}
	snapshotState := StateReady
	if expected == StateAwaitingTeacher {
		snapshotState = StateAwaitingTeacher
	}
	if err := verifyLatestSnapshot(r, snapshotState); err != nil {
		return causalv2.Authorization{}, err
	}
	if _, err := causalv2.VerifyProfileForFixture(r.profileBytes, r.fixture.FixtureDigest); err != nil {
		return causalv2.Authorization{}, err
	}
	if len(r.current) != len(causal.Actions()) {
		return causalv2.Authorization{}, errors.New("proposal boundary does not contain six actions")
	}
	forced, err := r.forcedActionForVerification()
	if err != nil {
		return causalv2.Authorization{}, err
	}
	var best candidate
	for index, action := range causal.Actions() {
		code := action.Code()
		artifacts, ok := r.current[code]
		if !ok {
			return causalv2.Authorization{}, fmt.Errorf("missing proposal triple for %s", code)
		}
		if err := verifyCandidateArtifacts(r, artifacts, code); err != nil {
			return causalv2.Authorization{}, err
		}
		if index == 0 {
			best = artifacts.candidate
			continue
		}
		verifierMeter := WorkMeter{}
		comparison, err := compareCandidates(r.profile.AcquisitionCode, len(r.posterior), artifacts.candidate, best, forced, &verifierMeter)
		if err != nil {
			return causalv2.Authorization{}, err
		}
		if comparison < 0 {
			best = artifacts.candidate
		}
	}
	var selection selectionPayload
	if err := r.decodePayload(r.selection, &selection); err != nil {
		return causalv2.Authorization{}, err
	}
	if selection.Action != best.action || len(selection.TieArtifactDigests) != len(r.ties) {
		return causalv2.Authorization{}, errors.New("selection does not equal complete score result")
	}
	for index, tie := range r.ties {
		if selection.TieArtifactDigests[index] != tie.Digest {
			return causalv2.Authorization{}, errors.New("selection tie digest order mismatch")
		}
		var payload tiePayload
		if err := r.decodePayload(tie, &payload); err != nil {
			return causalv2.Authorization{}, err
		}
		artifacts, ok := r.current[payload.Action]
		if !ok || payload.ScoreArtifactDigest != artifacts.score.Digest {
			return causalv2.Authorization{}, errors.New("tie does not reference its exact score")
		}
		verifierMeter := WorkMeter{}
		tied, err := candidatesTie(r.profile.AcquisitionCode, len(r.posterior), artifacts.candidate, best, forced, &verifierMeter)
		if err != nil || !tied {
			return causalv2.Authorization{}, errors.New("tie set contains a nontied action")
		}
	}
	if requireNoDescendants && (r.authorization.Digest != "" || r.result.Digest != "") {
		return causalv2.Authorization{}, errors.New("proposal boundary already has response descendants")
	}
	authorization := causalv2.Authorization{
		ProfileDigest: r.profile.ProfileDigest, Episode: r.episodeKey, Step: r.step,
		Action: selection.Action, SelectionArtifactDigest: r.selection.Digest,
		OpaqueToken: r.fixture.OpaqueToken,
	}
	if err := causalv2.SignAuthorization(&authorization); err != nil {
		return causalv2.Authorization{}, err
	}
	return authorization, nil
}

func verifyCandidateArtifacts(r *Runner, artifacts candidateArtifacts, action string) error {
	var cache cachePayload
	if err := r.decodePayload(artifacts.cache, &cache); err != nil {
		return err
	}
	var proposal proposalPayload
	if err := r.decodePayload(artifacts.proposal, &proposal); err != nil {
		return err
	}
	var partition partitionPayload
	if err := r.decodePayload(artifacts.partition, &partition); err != nil {
		return err
	}
	var score scorePayload
	if err := r.decodePayload(artifacts.score, &score); err != nil {
		return err
	}
	if cache.Action != action || proposal.Action != action || partition.Action != action || score.Action != action {
		return errors.New("proposal triple action mismatch")
	}
	if proposal.CacheArtifactDigest != artifacts.cache.Digest || score.CacheArtifactDigest != artifacts.cache.Digest {
		return errors.New("proposal/score cache reference mismatch")
	}
	if cache.PosteriorArtifactDigest != r.posteriorArtifact.Digest || partition.PosteriorArtifactDigest != r.posteriorArtifact.Digest {
		return errors.New("cache or partition uses stale posterior")
	}
	wantCells, err := causal.Partition(r.posterior, action)
	if err != nil {
		return err
	}
	if !cellsEqual(cache.Cells, wantCells) || !cellsEqual(partition.Cells, wantCells) {
		return errors.New("cached or materialized partition differs from public SCM prediction")
	}
	cost, err := r.actionCost(action)
	if err != nil {
		return err
	}
	wantFeatures, err := causal.FeaturesFor(r.posterior, action, cost, containsAction(r.consumed, action))
	if err != nil {
		return err
	}
	if cache.E != wantFeatures.ExpectedNumerator || cache.W != wantFeatures.Worst ||
		cache.H != wantFeatures.EntropyProduct.String() || cache.C != wantFeatures.Cost || cache.R != wantFeatures.Repeat {
		return fmt.Errorf("cache feature mismatch for %s: got E=%d W=%d H=%s C=%d R=%d, want E=%d W=%d H=%s C=%d R=%d", action,
			cache.E, cache.W, cache.H, cache.C, cache.R, wantFeatures.ExpectedNumerator, wantFeatures.Worst,
			wantFeatures.EntropyProduct.String(), wantFeatures.Cost, wantFeatures.Repeat)
	}
	gotFeatures := artifacts.candidate.features
	if gotFeatures.ExpectedNumerator != wantFeatures.ExpectedNumerator || gotFeatures.Worst != wantFeatures.Worst ||
		gotFeatures.EntropyProduct.Cmp(wantFeatures.EntropyProduct) != 0 || gotFeatures.Cost != wantFeatures.Cost || gotFeatures.Repeat != wantFeatures.Repeat {
		return fmt.Errorf("candidate features differ from fresh reconstruction for %s", action)
	}
	if score.RuleCode != r.profile.AcquisitionCode {
		return errors.New("score acquisition code mismatch")
	}
	return nil
}

func cellsEqual(left, right []causal.Cell) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Outcome != right[index].Outcome || !equalStrings(left[index].Hypotheses, right[index].Hypotheses) {
			return false
		}
	}
	return true
}

func (r *Runner) forcedActionForVerification() (string, error) {
	switch r.profile.AcquisitionCode {
	case string(PolicyUniformRandom):
		return r.fixture.UniformRandomActions[r.step], nil
	case string(PolicyDynamicOptimal):
		// The already-materialized selection is checked against a fresh policy,
		// so cached mutable DP state is never trusted by the verifier.
		costs := [3]int{r.fixture.Costs[0], r.fixture.Costs[1], r.fixture.Costs[2]}
		policy, err := NewDynamicPolicy(r.initial, costs)
		if err != nil {
			return "", err
		}
		return policy.Choose(r.posterior, r.consumed)
	default:
		return "", nil
	}
}

func verifyAuthorizationBoundary(r *Runner) error {
	if r.cursor.state != StateAwaitingTeacher || r.agenda.Len() != 0 {
		return errors.New("authorization boundary is not quiescent awaiting-teacher")
	}
	if _, err := causalv2.VerifyProfileForFixture(r.profileBytes, r.fixture.FixtureDigest); err != nil {
		return err
	}
	var authorization causalv2.Authorization
	if err := r.decodePayload(r.authorization, &authorization); err != nil {
		return err
	}
	want, err := verifyProposalArtifacts(r, StateAwaitingTeacher, false)
	if err != nil {
		return err
	}
	if authorization != want {
		return errors.New("authorization tuple differs from verified proposal")
	}
	if err := verifyLatestSnapshot(r, StateAwaitingTeacher); err != nil {
		return err
	}
	if r.result.Digest != "" {
		return errors.New("authorization boundary already has a result")
	}
	return nil
}

type updateDecision struct {
	action      string
	outcome     string
	after       []string
	eliminated  []string
	costBefore  int
	costAfter   int
	partition   []causal.Cell
	cacheStatus string
}

func verifyResponse(r *Runner) (updateDecision, error) {
	if r.cursor.state != StateResponsePresent || r.agenda.Len() != 0 {
		return updateDecision{}, errors.New("response verifier requires quiescent response-present")
	}
	if _, err := causalv2.VerifyProfileForFixture(r.profileBytes, r.fixture.FixtureDigest); err != nil {
		return updateDecision{}, err
	}
	if err := verifyLatestSnapshot(r, StateAwaitingTeacher); err != nil {
		return updateDecision{}, err
	}
	var authorization causalv2.Authorization
	if err := r.decodePayload(r.authorization, &authorization); err != nil {
		return updateDecision{}, err
	}
	var result resultPayload
	if err := r.decodePayload(r.result, &result); err != nil {
		return updateDecision{}, err
	}
	if result.AuthorizationArtifactDigest != r.authorization.Digest || result.Action != authorization.Action {
		return updateDecision{}, errors.New("result does not match authorization tuple")
	}
	for _, artifact := range r.artifacts {
		if artifact.Kind != "consumption" {
			continue
		}
		var consumption consumptionPayload
		if err := r.decodePayload(artifact, &consumption); err != nil {
			return updateDecision{}, err
		}
		if consumption.ResultArtifactDigest == r.result.Digest {
			return updateDecision{}, errors.New("result already consumed")
		}
	}
	selected := r.current[result.Action]
	var after []string
	for _, cell := range selected.candidate.cells {
		if cell.Outcome == result.Outcome {
			after = append([]string(nil), cell.Hypotheses...)
			break
		}
	}
	if len(after) == 0 {
		return updateDecision{}, errors.New("teacher outcome absent from predicted partition")
	}
	if err := r.meter.chargePosterior(len(r.posterior)); err != nil {
		return updateDecision{}, err
	}
	afterSet := make(map[string]bool, len(after))
	for _, hypothesis := range after {
		afterSet[hypothesis] = true
	}
	var eliminated []string
	for _, hypothesis := range r.posterior {
		if !afterSet[hypothesis] {
			eliminated = append(eliminated, hypothesis)
		}
	}
	cost, err := r.actionCost(result.Action)
	if err != nil {
		return updateDecision{}, err
	}
	if r.totalCost+cost > causal.EpisodeCostCap {
		return updateDecision{}, errors.New("authorized action crosses cost ceiling")
	}
	status := "miss"
	if selected.cacheHit {
		status = "hit"
	}
	return updateDecision{
		action: result.Action, outcome: result.Outcome, after: after, eliminated: eliminated,
		costBefore: r.totalCost, costAfter: r.totalCost + cost,
		partition: selected.candidate.cells, cacheStatus: status,
	}, nil
}

func transcriptEntryDigest(entry TranscriptEntry) (string, error) {
	entry.TranscriptDigest = ""
	return causalv2.Digest("causal-transcript-entry/v2", entry)
}

func (r *Runner) finalizeZeroAction(terminal string) error {
	if !r.cueMaterializing {
		return errors.New("terminal materialization is restricted to CUE execution")
	}
	if r.step != 0 || len(r.consumed) != 0 || r.cursor.state != StateReady {
		return errors.New("zero-action terminal requested after execution began")
	}
	emptyDigest, err := causalv2.Digest("causal-empty-transcript/v2", struct{}{})
	if err != nil {
		return err
	}
	r.transcriptDigest = emptyDigest
	r.terminal = terminal
	r.cursor.state = StateTerminal
	r.pendingTerminal = ""
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	_, err = r.allocate(0, "terminal", terminalPayload{
		Terminal: terminal, PosteriorDigest: posteriorDigest, TotalCost: 0,
		ActionCount: 0, TranscriptDigest: emptyDigest,
	})
	return err
}

func verifyReadyOrTerminalBoundary(r *Runner) error {
	if r.cursor.state != StateReady && r.cursor.state != StateTerminal {
		return fmt.Errorf("final verifier state=%q", r.cursor.state)
	}
	if r.agenda.Len() != 0 {
		return errors.New("ready/terminal boundary has pending task")
	}
	if _, err := causalv2.VerifyProfileForFixture(r.profileBytes, r.fixture.FixtureDigest); err != nil {
		return err
	}
	if err := validatePosterior(r.posterior, 1, causal.MaximumPool); err != nil {
		return err
	}
	if len(r.actions) != len(r.outcomes) || len(r.actions) != len(r.consumed) {
		return errors.New("action/outcome/consumption prefix length mismatch")
	}
	if len(r.actions) > causal.MaximumActions || r.totalCost > causal.EpisodeCostCap {
		return errors.New("episode budget exceeded")
	}
	if err := verifyArtifactSequence(r); err != nil {
		return err
	}
	snapshotState := r.cursor.state
	if r.cursor.state == StateTerminal && len(r.actions) == 0 {
		snapshotState = StateReady
	}
	if err := verifyLatestSnapshot(r, snapshotState); err != nil {
		return err
	}
	if r.cursor.state == StateTerminal {
		want := terminalFor(r.initial, r.posterior, len(r.actions))
		if r.profile.AcquisitionCode == string(PolicyPassiveOnly) && len(r.actions) == 0 {
			want = "budget-exhausted"
		}
		if r.terminal != want {
			return fmt.Errorf("terminal=%q, recomputed %q", r.terminal, want)
		}
	}
	return r.meter.Counts().ValidateEquation()
}

// verifyLatestSnapshot reconstructs every sealed descriptor field from fresh
// canonical inputs, current semantic state, the immutable artifact ledger, and
// the meter checkpoint captured immediately after snapshot allocation.
func verifyLatestSnapshot(r *Runner, expected State) error {
	if r.cursor.latestSnapshotDigest == "" {
		return errors.New("descriptor cursor has no latest snapshot")
	}
	ref, ok := r.byDigest[r.cursor.latestSnapshotDigest]
	if !ok || ref.Kind != "descriptor-snapshot" {
		return errors.New("latest descriptor snapshot reference is absent or wrong kind")
	}
	var snapshot descriptorSnapshotPayload
	if err := r.decodePayload(ref, &snapshot); err != nil {
		return err
	}
	previous := causalv2.ZeroDigest
	found := false
	for _, candidate := range r.artifacts {
		if candidate.ChargeIndex > ref.ChargeIndex {
			break
		}
		if candidate.Kind != "descriptor-snapshot" {
			continue
		}
		if candidate.Digest == ref.Digest {
			found = true
			break
		}
		previous = candidate.Digest
	}
	if !found {
		return errors.New("latest snapshot is outside the charge-ordered ledger")
	}
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	counts := ref.MeterAfter
	if err := counts.ValidateEquation(); err != nil {
		return fmt.Errorf("snapshot meter checkpoint: %w", err)
	}
	if snapshot.PreviousSnapshotArtifactDigest != previous ||
		snapshot.State != string(expected) ||
		!slices.Equal(snapshot.Aliases, r.fixture.Aliases) ||
		!slices.Equal(snapshot.Costs, r.fixture.Costs) ||
		!slices.Equal(snapshot.Presentation, r.fixture.Presentation) ||
		snapshot.InitialPosteriorArtifactDigest != r.initialPosteriorArtifact.Digest ||
		snapshot.PosteriorDigest != posteriorDigest ||
		snapshot.AcquisitionCode != r.profile.AcquisitionCode ||
		snapshot.TotalCost != r.totalCost ||
		snapshot.ActionCount != len(r.consumed) ||
		snapshot.RemainingEvaluations != EpisodeEvaluationCap-counts.SCMEvaluations ||
		snapshot.RemainingWork != EpisodeWorkCap-counts.TotalWork ||
		snapshot.RemainingCycles != EpisodeCycleCap-counts.EngineCycles ||
		snapshot.RemainingUnits != EpisodeUnitCap-counts.AttributedUnits {
		return errors.New("descriptor snapshot differs from freshly reconstructed state or caps")
	}
	return nil
}

func verifyArtifactSequence(r *Runner) error {
	for index, ref := range r.artifacts {
		artifact, err := causalv2.VerifyArtifact(ref.Canonical)
		if err != nil {
			return fmt.Errorf("artifact %d: %w", index, err)
		}
		if artifact.ProfileDigest != r.profile.ProfileDigest || artifact.Scope != r.episodeKey || artifact.ChargeIndex != index {
			return fmt.Errorf("artifact %d context/charge mismatch", index)
		}
		name := artifactName(artifact.Kind, artifact.ArtifactDigest)
		unitArtifact := r.store.Get(name)
		if unitArtifact == nil || !unitArtifact.GetBool("sealed") || unitArtifact.GetString("artifactBytes") != string(ref.Canonical) {
			return fmt.Errorf("artifact %d is absent, unsealed, or corrupted", index)
		}
	}
	return nil
}

func sortedCopy(items []string) []string {
	result := append([]string(nil), items...)
	sort.Strings(result)
	return result
}
