package causalrun

import (
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

func oneArgument(arguments []string) (string, error) {
	if len(arguments) != 1 || arguments[0] == "" {
		return "", errors.New("causal CUE operation requires one argument")
	}
	return arguments[0], nil
}

func twoArguments(arguments []string) (string, string, error) {
	if len(arguments) != 2 || arguments[0] == "" || arguments[1] == "" {
		return "", "", errors.New("causal CUE operation requires two arguments")
	}
	return arguments[0], arguments[1], nil
}

func (r *Runner) prepareProposal() error {
	if r.activeCUETask != proposalTaskSlot || r.cursor.state != StateProposing {
		return errors.New("proposal preparation outside proposing CUE task")
	}
	if r.selection.Digest != "" || r.authorization.Digest != "" || r.result.Digest != "" {
		return errors.New("stale proposal descendants at proposal start")
	}
	forced, err := r.forcedAction()
	if err != nil {
		return err
	}
	r.forcedActionCode = forced
	r.current = make(map[string]candidateArtifacts, len(causal.Actions()))
	r.ties = nil
	return nil
}

func (r *Runner) materializeCandidateCache(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	if _, exists := r.current[action]; exists {
		return fmt.Errorf("duplicate CUE cache action %q", action)
	}
	if _, err := causal.ParseAction(action); err != nil {
		return err
	}
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	key := r.cacheKey(posteriorDigest, action)
	cached, cacheHit := r.cache[key]
	cacheHit = cacheHit && !r.disableCacheReuse
	var cells []causal.Cell
	var features causal.Features
	if cacheHit {
		cells = canonicalCells(cached.cells)
		features = cloneFeatures(cached.features)
		features.Repeat = 0
		if containsAction(r.consumed, action) {
			features.Repeat = 1
		}
	} else {
		cells, err = partitionWithMeter(r.posterior, action, &r.meter)
		if err != nil {
			return err
		}
		cost, costErr := r.actionCost(action)
		if costErr != nil {
			return costErr
		}
		features, err = featuresFromCells(cells, cost, containsAction(r.consumed, action), &r.meter)
		if err != nil {
			return err
		}
		if !r.disableCacheReuse {
			storedFeatures := cloneFeatures(features)
			storedFeatures.Repeat = 0
			r.cache[key] = cachedCandidate{cells: canonicalCells(cells), features: storedFeatures}
		}
	}
	cache, err := r.allocate(r.step, "cache", cachePayload{
		Action: action, PosteriorArtifactDigest: r.posteriorArtifact.Digest,
		Cells: canonicalCells(cells), E: features.ExpectedNumerator, W: features.Worst,
		H: features.EntropyProduct.String(), C: features.Cost, R: features.Repeat,
	})
	if err != nil {
		return err
	}
	status := "miss"
	if cacheHit {
		status = "hit"
		r.cacheTrace.Hits++
	} else {
		r.cacheTrace.Misses++
	}
	r.cacheTrace.Statuses = append(r.cacheTrace.Statuses, status)
	artifacts := candidateArtifacts{candidate: candidate{action: action, cells: cells, features: features}, cache: cache, cacheHit: cacheHit}
	r.current[action] = artifacts
	return nil
}

func (r *Runner) candidateFor(action string) (candidateArtifacts, error) {
	artifacts, ok := r.current[action]
	if !ok {
		return candidateArtifacts{}, fmt.Errorf("CUE action %q has no cache", action)
	}
	return artifacts, nil
}

func (r *Runner) materializeCandidateProposal(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	artifacts, err := r.candidateFor(action)
	if err != nil {
		return err
	}
	if artifacts.proposal.Digest != "" {
		return errors.New("duplicate proposal materialization")
	}
	artifacts.proposal, err = r.allocate(r.step, "proposal", proposalPayload{Action: action, CacheArtifactDigest: artifacts.cache.Digest})
	if err == nil {
		r.current[action] = artifacts
	}
	return err
}

func (r *Runner) materializeCandidatePartition(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	artifacts, err := r.candidateFor(action)
	if err != nil {
		return err
	}
	if artifacts.proposal.Digest == "" || artifacts.partition.Digest != "" {
		return errors.New("partition CUE order violation")
	}
	artifacts.partition, err = r.allocate(r.step, "partition", partitionPayload{Action: action, PosteriorArtifactDigest: r.posteriorArtifact.Digest, Cells: canonicalCells(artifacts.candidate.cells)})
	if err == nil {
		r.current[action] = artifacts
	}
	return err
}

func (r *Runner) materializeCandidateScore(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	artifacts, err := r.candidateFor(action)
	if err != nil {
		return err
	}
	if artifacts.partition.Digest == "" || artifacts.score.Digest != "" {
		return errors.New("score CUE order violation")
	}
	artifacts.score, err = r.allocate(r.step, "score", scorePayload{Action: action, RuleCode: r.profile.AcquisitionCode, CacheArtifactDigest: artifacts.cache.Digest})
	if err == nil {
		r.current[action] = artifacts
	}
	return err
}

func (r *Runner) candidateBetter(arguments []string) (bool, error) {
	left, right, err := twoArguments(arguments)
	if err != nil {
		return false, err
	}
	a, err := r.candidateFor(left)
	if err != nil {
		return false, err
	}
	b, err := r.candidateFor(right)
	if err != nil {
		return false, err
	}
	if a.score.Digest == "" || b.score.Digest == "" {
		return false, errors.New("CUE comparison precedes score materialization")
	}
	comparison, err := compareCandidates(r.profile.AcquisitionCode, len(r.posterior), a.candidate, b.candidate, r.forcedActionCode, &r.meter)
	return comparison < 0, err
}

func (r *Runner) candidateEqual(arguments []string) (bool, error) {
	left, right, err := twoArguments(arguments)
	if err != nil {
		return false, err
	}
	a, err := r.candidateFor(left)
	if err != nil {
		return false, err
	}
	b, err := r.candidateFor(right)
	if err != nil {
		return false, err
	}
	return candidatesTie(r.profile.AcquisitionCode, len(r.posterior), a.candidate, b.candidate, r.forcedActionCode, &r.meter)
}

func (r *Runner) materializeCandidateTie(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	artifacts, err := r.candidateFor(action)
	if err != nil {
		return err
	}
	if artifacts.score.Digest == "" {
		return errors.New("tie precedes score")
	}
	for _, existing := range r.ties {
		var payload tiePayload
		if err := r.decodePayload(existing, &payload); err != nil {
			return err
		}
		if payload.Action == action {
			return errors.New("duplicate CUE tie action")
		}
	}
	tie, err := r.allocate(r.step, "tie", tiePayload{Action: action, ScoreArtifactDigest: artifacts.score.Digest})
	if err == nil {
		r.ties = append(r.ties, tie)
	}
	return err
}

func (r *Runner) materializeCandidateSelection(arguments []string) error {
	action, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	if _, err := r.candidateFor(action); err != nil {
		return err
	}
	if len(r.ties) == 0 || r.selection.Digest != "" {
		return errors.New("selection requires a nonempty unique CUE tie set")
	}
	digests := make([]string, len(r.ties))
	for index, tie := range r.ties {
		digests[index] = tie.Digest
	}
	r.selection, err = r.allocate(r.step, "selection", selectionPayload{Action: action, TieArtifactDigests: digests})
	return err
}

func (r *Runner) materializeAuthorization() error {
	if r.activeCUETask != authorizationTaskSlot || r.pendingAuthorization.AuthorizationDigest == "" || r.authorization.Digest != "" {
		return errors.New("authorization CUE task lacks verified decision")
	}
	canonical, err := causalv2.CanonicalJSON(r.pendingAuthorization)
	if err != nil {
		return err
	}
	verified, err := causalv2.VerifyAuthorization(canonical)
	if err != nil {
		return err
	}
	r.authorization, err = r.allocate(r.step, "authorization", verified)
	return err
}

func (r *Runner) materializeAwaitingSnapshot() error {
	if r.activeCUETask != authorizationTaskSlot || r.authorization.Digest == "" {
		return errors.New("awaiting snapshot requires a materialized authorization")
	}
	if r.cursor.state != StateProposing {
		return errors.New("awaiting snapshot begins outside proposing state")
	}
	return r.materializeSnapshot(StateAwaitingTeacher)
}

func (r *Runner) prepareUpdate() error {
	if r.activeCUETask != updateTaskSlot {
		return errors.New("update preparation outside update CUE task")
	}
	decision, err := verifyResponse(r)
	if err != nil {
		return err
	}
	r.cursor.state = StateUpdating
	r.pendingUpdate = &decision
	r.pendingPosterior = artifactRef{}
	r.pendingConsumption = artifactRef{}
	r.pendingSnapshotDigest = ""
	r.updateEliminationIndex = 0
	return nil
}

func (r *Runner) updateDecision() (*updateDecision, error) {
	if r.pendingUpdate == nil {
		return nil, errors.New("CUE update has no verified response decision")
	}
	return r.pendingUpdate, nil
}

func (r *Runner) updateEliminated() ([]string, error) {
	decision, err := r.updateDecision()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), decision.eliminated...), nil
}

func (r *Runner) materializeUpdateElimination(arguments []string) error {
	hypothesis, err := oneArgument(arguments)
	if err != nil {
		return err
	}
	decision, err := r.updateDecision()
	if err != nil {
		return err
	}
	if r.updateEliminationIndex >= len(decision.eliminated) || decision.eliminated[r.updateEliminationIndex] != hypothesis {
		return errors.New("CUE elimination is missing, duplicated, or out of order")
	}
	_, err = r.allocate(r.step, "elimination", eliminationPayload{Hypothesis: hypothesis, ResultArtifactDigest: r.result.Digest})
	if err == nil {
		r.updateEliminationIndex++
	}
	return err
}

func (r *Runner) materializeUpdatePosterior() error {
	decision, err := r.updateDecision()
	if err != nil {
		return err
	}
	if r.updateEliminationIndex != len(decision.eliminated) {
		return errors.New("CUE posterior precedes complete elimination set")
	}
	if r.pendingPosterior.Digest != "" {
		return errors.New("duplicate posterior materialization")
	}
	r.pendingPosterior, err = r.materializePosterior(r.step, decision.after)
	return err
}

func (r *Runner) materializeUpdateConsumption() error {
	if r.pendingPosterior.Digest == "" || r.pendingConsumption.Digest != "" {
		return errors.New("consumption precedes posterior")
	}
	var err error
	r.pendingConsumption, err = r.allocate(r.step, "consumption", consumptionPayload{ResultArtifactDigest: r.result.Digest, PosteriorArtifactDigest: r.pendingPosterior.Digest})
	return err
}

func (r *Runner) materializeUpdateTranscript() error {
	decision, err := r.updateDecision()
	if err != nil {
		return err
	}
	if r.pendingConsumption.Digest == "" || len(r.artifacts) == 0 || r.artifacts[len(r.artifacts)-1].Digest != r.pendingConsumption.Digest {
		return errors.New("transcript CUE order violation")
	}
	beforeDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	afterDigest, err := hypothesisSetDigest(decision.after)
	if err != nil {
		return err
	}
	eliminatedDigest, err := canonicalStringSetDigest(decision.eliminated)
	if err != nil {
		return err
	}
	partDigest, err := partitionDigest(decision.partition)
	if err != nil {
		return err
	}
	r.posterior = append([]string(nil), decision.after...)
	r.posteriorArtifact = r.pendingPosterior
	r.totalCost = decision.costAfter
	r.consumed = append(r.consumed, decision.action)
	r.actions = append(r.actions, decision.action)
	r.outcomes = append(r.outcomes, decision.outcome)
	if err := r.meter.chargeTranscript(20); err != nil {
		return err
	}
	remainingEvaluations, remainingWork, remainingCycles, remainingUnits := r.meter.remaining()
	previousTranscript := r.transcriptDigest
	if previousTranscript == "" {
		previousTranscript = causalv2.ZeroDigest
	}
	entry := TranscriptEntry{
		TranscriptVersion: "causal-transcript/v2", Episode: r.episodeKey, Step: r.step,
		PreviousDigest: previousTranscript, RuleCode: r.profile.AcquisitionCode,
		Action: decision.action, PosteriorBeforeDigest: beforeDigest,
		PartitionDigest: partDigest, TeacherOutcome: decision.outcome,
		PosteriorAfterDigest: afterDigest, EliminatedDigest: eliminatedDigest,
		CostBefore: decision.costBefore, CostAfter: decision.costAfter,
		ActionCount: len(r.consumed), CacheStatus: decision.cacheStatus,
		AttributedUnitPrefix:           r.meter.counts.AttributedUnits + 1,
		RemainingHypothesisEvaluations: remainingEvaluations,
		RemainingSemanticWork:          remainingWork - 1,
		RemainingEngineCycles:          remainingCycles,
		RemainingAttributedUnits:       remainingUnits - 1,
	}
	entry.TranscriptDigest, err = transcriptEntryDigest(entry)
	if err != nil {
		return err
	}
	r.lastTranscript, err = r.allocate(r.step, "transcript", entry)
	if err == nil {
		r.transcriptDigest = entry.TranscriptDigest
	}
	return err
}

func (r *Runner) materializeUpdateSnapshot() error {
	if r.lastTranscript.Digest == "" || r.pendingSnapshotDigest != "" {
		return errors.New("snapshot precedes transcript")
	}
	r.terminal = terminalFor(r.initial, r.posterior, len(r.consumed))
	state := StateReady
	if r.terminal != "" {
		state = StateTerminal
	}
	before := r.cursor.latestSnapshotDigest
	if err := r.materializeSnapshot(state); err != nil {
		return err
	}
	if r.cursor.latestSnapshotDigest == before {
		return errors.New("snapshot was not newly materialized")
	}
	r.pendingSnapshotDigest = r.cursor.latestSnapshotDigest
	return nil
}

func (r *Runner) materializeUpdateTerminal() error {
	if r.terminal == "" || r.pendingSnapshotDigest == "" {
		return errors.New("terminal precedes terminal snapshot")
	}
	if len(r.artifacts) != 0 && r.artifacts[len(r.artifacts)-1].Kind == "terminal" {
		return errors.New("duplicate terminal materialization")
	}
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return err
	}
	_, err = r.allocate(r.step, "terminal", terminalPayload{Terminal: r.terminal, PosteriorDigest: posteriorDigest, TotalCost: r.totalCost, ActionCount: len(r.consumed), TranscriptDigest: r.transcriptDigest})
	return err
}

func (r *Runner) finishUpdate() error {
	if r.pendingSnapshotDigest == "" {
		return errors.New("CUE update ended before snapshot")
	}
	if r.terminal != "" {
		if len(r.artifacts) == 0 || r.artifacts[len(r.artifacts)-1].Kind != "terminal" {
			return errors.New("CUE update omitted terminal artifact")
		}
	} else if r.cursor.state != StateReady {
		return errors.New("nonterminal CUE update did not return ready")
	}
	r.step++
	r.current = make(map[string]candidateArtifacts)
	r.ties = nil
	r.selection = artifactRef{}
	r.pendingAuthorization = causalv2.Authorization{}
	r.authorization = artifactRef{}
	r.result = artifactRef{}
	r.teacherCalled = false
	r.forcedActionCode = ""
	r.pendingUpdate = nil
	r.pendingPosterior = artifactRef{}
	r.pendingConsumption = artifactRef{}
	r.pendingSnapshotDigest = ""
	r.updateEliminationIndex = 0
	return nil
}
