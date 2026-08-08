package causalexpv2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
)

// ExecuteValidation owns the preregistered validation panel and exclusively
// publishes its reconstructed report. It accepts no seeds or report fields.
func ExecuteValidation(ctx context.Context, repoRoot string) (returnErr error) {
	if err := orchestrationAvailable(); err != nil {
		return err
	}
	capability, err := beginValidationAttempt(ctx, repoRoot)
	if err != nil {
		return err
	}
	return executeAndPublishEvaluation(ctx, repoRoot, capability)
}

// ExecuteLocked owns the one-shot locked panel. beginLockedAttempt requires a
// still-valid published validation result for the same clean candidate.
func ExecuteLocked(ctx context.Context, repoRoot string) (returnErr error) {
	if err := orchestrationAvailable(); err != nil {
		return err
	}
	capability, err := beginLockedAttempt(ctx, repoRoot)
	if err != nil {
		return err
	}
	return executeAndPublishEvaluation(ctx, repoRoot, capability)
}

func executeAndPublishEvaluation(ctx context.Context, repoRoot string, capability *attemptCapability) (returnErr error) {
	defer func() {
		if returnErr != nil {
			capability.mu.Lock()
			consumed := capability.consumed
			capability.mu.Unlock()
			if !consumed {
				returnErr = errors.Join(returnErr, capability.Fail())
			}
		}
	}()
	reportBytes, err := regenerateEvaluationReport(ctx, repoRoot, capability)
	if err != nil {
		return err
	}
	verified, err := contextuallyVerifyEvaluationEvidence(ctx, repoRoot, reportBytes)
	if err != nil {
		return err
	}
	capability.mu.Lock()
	capability.expectedReportDigest = verified.Report.ReportDigest
	capability.mu.Unlock()
	return capability.publishEvaluationReport(ctx, repoRoot, reportBytes)
}

func regenerateEvaluationReport(ctx context.Context, repoRoot string, capability *attemptCapability) ([]byte, error) {
	if capability == nil || (capability.record.Panel != PanelValidation && capability.record.Panel != PanelLocked) {
		return nil, errors.New("missing evaluation regeneration authority")
	}
	trainingBytes, err := gitFile(ctx, capability.repositoryRoot, FrozenTrainingReportCommit, TrainingEvidenceDirectory+"/"+TrainingReportName)
	if err != nil {
		return nil, err
	}
	bundleBytes, err := gitFile(ctx, capability.repositoryRoot, FrozenTrainingReportCommit, TrainingEvidenceDirectory+"/"+TrainingEpisodesName)
	if err != nil {
		return nil, err
	}
	verifiedTraining, err := contextuallyVerifyTrainingEvidence(ctx, capability.repositoryRoot, trainingBytes, bundleBytes)
	if err != nil || verifiedTraining.Report.TrainingDigest != FrozenTrainingDigest || verifiedTraining.Report.SelectedRule != FrozenRule {
		return nil, errors.New("frozen training evidence does not reconstruct")
	}
	if err := requireUsableTrainingReport(verifiedTraining.Report); err != nil {
		return nil, err
	}
	fixtures := make([]PrivateFixture, capability.record.SeedRange.Count)
	for index := range fixtures {
		seed := capability.record.SeedRange.Start + int64(index)*capability.record.SeedRange.Step
		fixtures[index], err = capability.generateFixture(seed, index)
		if err != nil {
			return nil, err
		}
	}
	return buildEvaluationReport(ctx, repoRoot, capability.record.Panel, capability.record.ExecutableCommit, FrozenTrainingReportCommit, fixtures, verifiedTraining)
}

func buildEvaluationReport(ctx context.Context, repoRoot string, panel Panel, implementationCommit, trainingReportCommit string, fixtures []PrivateFixture, verifiedTraining verifiedTrainingEvidence) ([]byte, error) {
	training, trainingBundle := verifiedTraining.Report, verifiedTraining.Bundle
	manifest := causalv2.PreregisteredManifest()
	report := EvaluationReport{ReportVersion: "causal-diagnosis-report/v2", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: training.PretrainingCommit, TrainingReportCommit: trainingReportCommit, TrainingDigest: training.TrainingDigest, FrozenRule: training.SelectedRule, ImplementationCommit: implementationCommit, Panel: string(panel), Status: "invalid", Policies: []PolicyReport{}, Contrasts: []Contrast{}, Limitations: []string{"Synthetic deterministic three-variable SCMs do not establish production causal diagnosis."}}
	var episodeItems [][]MeterItem
	expectedTotal := new(big.Rat)
	maxDescriptor := 0
	transcriptValid := true
	for _, policy := range evaluationPolicies {
		policyReport := PolicyReport{Name: policy, Fixtures: []EvaluationFixture{}, Cohorts: []Aggregate{}}
		for _, fixture := range fixtures {
			acquisition := developmentAcquisition(policy)
			if policy == "learned" {
				acquisition = training.SelectedRule
			}
			episode, runErr := runEpisode(ctx, panel, fixture, acquisition, "", policy == "dynamic-optimal")
			if runErr != nil {
				return nil, fmt.Errorf("%s seed %d: %w", policy, fixture.PublicFixture.Seed, runErr)
			}
			maxDescriptor = maxInt(maxDescriptor, len(episode.profileBytes))
			transcriptValid = transcriptValid && equalCanonical(episode.result, episode.replay)
			if policy == "dynamic-optimal" {
				expected, ok := new(big.Rat).SetString(episode.dynamic.UniformExpectedNumerator + "/" + episode.dynamic.UniformExpectedDenominator)
				if !ok || episode.dynamic.MemberSimulations != len(fixture.PublicFixture.InitialPosterior) {
					return nil, errors.New("dynamic benchmark exact expectation or member count is invalid")
				}
				expectedTotal.Add(expectedTotal, expected)
			}
			row := evaluationFixtureFromEpisode(episode)
			policyReport.Fixtures = append(policyReport.Fixtures, row)
			episodeItems = append(episodeItems, row.MeterItems)
		}
		policyReport.Overall = aggregateEvaluation(string(policy), policyReport.Fixtures)
		for _, cohort := range []Cohort{CohortCostSkewed, CohortBalanced, CohortEquivalence, CohortIrrelevant} {
			var subset []EvaluationFixture
			for _, fixture := range policyReport.Fixtures {
				if fixture.Cohort == cohort {
					subset = append(subset, fixture)
				}
			}
			policyReport.Cohorts = append(policyReport.Cohorts, aggregateEvaluation(string(cohort), subset))
		}
		report.Policies = append(report.Policies, policyReport)
	}
	if len(fixtures) == 0 {
		return nil, errors.New("evaluation fixture panel is empty")
	}
	expectedTotal.Quo(expectedTotal, new(big.Rat).SetInt64(int64(len(fixtures))))
	report.DynamicBenchmark.UniformExpectedMeanCost, _ = expectedTotal.Float64()
	report.Mechanical.MaxDescriptorBytes = maxDescriptor
	report.Mechanical.TranscriptValid = transcriptValid
	report.Mechanical.TrainingFreezeValid = evaluationTrainingFreezeValid(ctx, repoRoot, implementationCommit, trainingReportCommit, verifiedTraining)
	central := causalv2.CentralProfile{CentralProfileVersion: causalv2.CentralProfileDomain, Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: training.PretrainingCommit, CreditEnabled: true}
	if err := causalv2.SignCentralProfile(&central); err != nil {
		return nil, err
	}
	centralBytes := mustCanonical(central)
	certificateBytes := make([][]byte, len(training.Applications))
	episodeBytes := make([][]byte, len(trainingBundle.Episodes))
	for index := range training.Applications {
		certificateBytes[index] = mustCanonical(training.Applications[index])
		episodeBytes[index] = mustCanonical(trainingBundle.Episodes[index])
	}
	curriculum, err := centralCurriculumAdapter(ctx, centralBytes, episodeBytes, certificateBytes)
	if err != nil {
		return nil, err
	}
	reference, err := runEpisode(ctx, PanelTraining, trainingBundle.Fixtures[0], training.SelectedRule, "", false)
	if err != nil {
		return nil, err
	}
	var controlEvidence ControlEvidence
	controls, err := controlSuiteAdapter(ctx, repoRoot, reference, curriculum, centralBytes, episodeBytes, certificateBytes, trainingBundle.Fixtures, verifiedTraining.CorruptionFixture, &controlEvidence)
	if err != nil {
		return nil, err
	}
	report.ControlBundle, report.ControlBundleDigest = controls, controls.ControlBundleDigest
	report.ControlEvidence, report.ControlEvidenceDigest = controlEvidence, controlEvidence.ControlEvidenceDigest
	report.TaskMeterItems, err = freshTrainingReplayTasks(ctx, training, trainingBundle)
	if err != nil {
		return nil, err
	}
	report.TaskMeterItemsDigest, err = causalv2.TaskMeterItemsDigest(report.TaskMeterItems)
	if err != nil {
		return nil, err
	}
	controlCounts := make([][15]int64, len(controls.Certificates))
	for index := range controls.Certificates {
		controlCounts[index] = controls.Certificates[index].MeterCounts
	}
	report.Mechanical.Meters, _, err = ReconstructMeters(MeterEvaluation, episodeItems, report.TaskMeterItems, controlCounts)
	if err != nil {
		return nil, err
	}
	reconstructEvaluationDerivations(&report)
	encoded, err := FinalizeEvaluationReport(&report)
	if err != nil {
		return nil, err
	}
	verified, err := VerifyEvaluationReportBytes(encoded)
	if err != nil {
		return nil, err
	}
	if err := VerifyEvaluationEvidence(verified); err != nil {
		return nil, err
	}
	return encoded, nil
}

func evaluationTrainingFreezeValid(ctx context.Context, repoRoot, implementationCommit, trainingReportCommit string, verified verifiedTrainingEvidence) bool {
	report := verified.Report
	if trainingReportCommit != FrozenTrainingReportCommit || report.TrainingDigest != FrozenTrainingDigest || report.SelectedRule != FrozenRule || requireUsableTrainingReport(report) != nil {
		return false
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil || !state.Clean || state.Head != implementationCommit {
		return false
	}
	if verifyEvidenceCommitShape(ctx, state.Root, report.PretrainingCommit, trainingReportCommit) != nil || verifyEmptyFreezeAt(ctx, state.Root, report.PretrainingCommit) != nil || verifyEmptyFreezeAt(ctx, state.Root, trainingReportCommit) != nil {
		return false
	}
	if verifyCandidateConstantsState(ctx, state, trainingReportCommit) != nil || verifyFrozenCandidate(state.Root, report, trainingReportCommit) != nil {
		return false
	}
	return true
}

func freshTrainingReplayTasks(ctx context.Context, training TrainingReport, bundle TrainingBundle) ([]TaskMeterItem, error) {
	rules := orderedRules()
	manifest := causalv2.PreregisteredManifest()
	certificateReplay := make([]TaskMeterItem, 0, len(training.Applications))
	postSelectionReplay := make([]TaskMeterItem, 0, len(training.Applications))
	for seedIndex := 0; seedIndex < manifest.TrainingSeeds.Count; seedIndex++ {
		seed := manifest.TrainingSeeds.Start + int64(seedIndex)*manifest.TrainingSeeds.Step
		if len(bundle.Fixtures) != manifest.TrainingSeeds.Count {
			return nil, errors.New("fresh replay requires opened training fixtures")
		}
		fixture := bundle.Fixtures[seedIndex]
		if fixture.PublicFixture.Seed != seed {
			return nil, errors.New("opened training fixtures are not in seed order")
		}
		for ruleIndex, rule := range rules {
			index := seedIndex*len(rules) + ruleIndex
			episode, err := runEpisode(ctx, PanelTraining, fixture, rule.Code(), rule.Code(), ruleIndex == 0)
			if err != nil {
				return nil, err
			}
			if !slices.Equal(mustCanonical(episode.certificate), mustCanonical(training.Applications[index])) {
				return nil, fmt.Errorf("fresh certificate replay %d differs from frozen training evidence", index)
			}
			certificateReplay = append(certificateReplay, TaskMeterItem{Name: "certificate-replay", Subject: episode.certificate.CertificateDigest, Counts: counts64(episode.replay.ProductionCounts)})
			post, err := causalrun.VerifyEpisode(episode.publicBytes, episode.profileBytes, episode.result.Artifacts)
			if err != nil || !slices.Equal(mustCanonical(post), mustCanonical(episode.replay)) {
				return nil, fmt.Errorf("fresh post-selection replay %d differs", index)
			}
			postSelectionReplay = append(postSelectionReplay, TaskMeterItem{Name: "post-selection-replay", Subject: episode.certificate.CertificateDigest, Counts: counts64(post.ProductionCounts)})
		}
	}
	return append(certificateReplay, postSelectionReplay...), nil
}

func evaluationFixtureFromEpisode(episode executedEpisode) EvaluationFixture {
	correct := slices.Contains(episode.result.FinalPosterior, episode.fixture.HiddenHypothesis)
	equivalence := episode.result.Terminal == "equivalence" && causaloracle.CompleteClass(episode.fixture.PublicFixture.InitialPosterior, episode.result.FinalPosterior)
	return EvaluationFixture{Seed: episode.result.Seed, Cohort: Cohort(episode.fixture.PublicFixture.Cohort), Terminal: episode.result.Terminal, Score: episode.result.Score, InterventionCost: episode.result.Cost, Actions: append([]string(nil), episode.result.Actions...), ActionCount: len(episode.result.Actions), InitialPosterior: len(episode.fixture.PublicFixture.InitialPosterior), FinalPosterior: len(episode.result.FinalPosterior), Correct: correct, TeacherRetained: correct, EquivalenceComplete: equivalence, HypothesisEvaluations: episode.result.ProductionCounts.SCMEvaluations, SemanticWork: episode.result.ProductionCounts.TotalWork, EngineCycles: episode.result.ProductionCounts.EngineCycles, AttributedUnits: episode.result.ProductionCounts.AttributedUnits, CacheHits: episode.replay.CacheTrace.Hits, CacheMisses: episode.replay.CacheTrace.Misses, TranscriptDigest: episode.replay.TranscriptDigest, OracleAgreements: episode.evidence.OracleAgreements, OracleDisagreements: episode.evidence.OracleDisagreements, MeterItems: episode.evidence.MeterItems, AllCapsValid: episode.evidence.AllCapsValid}
}

func aggregateEvaluation(name string, fixtures []EvaluationFixture) Aggregate {
	result := Aggregate{Name: name, Fixtures: len(fixtures)}
	actions := 0
	for _, fixture := range fixtures {
		switch fixture.Terminal {
		case "identified":
			result.Identified++
		case "equivalence":
			result.Equivalence++
		default:
			result.BudgetExhausted++
		}
		if fixture.Correct {
			result.Correct++
		}
		result.TotalScore += fixture.Score
		result.TotalCost += fixture.InterventionCost
		actions += fixture.ActionCount
	}
	if result.Fixtures != 0 {
		n := float64(result.Fixtures)
		result.MeanScore = float64(result.TotalScore) / n
		result.MeanCost = float64(result.TotalCost) / n
		result.MeanActions = float64(actions) / n
		result.Accuracy = float64(result.Correct) / n
	}
	return result
}

func evaluationScores(policy PolicyReport, cohort Cohort) []float64 {
	var values []float64
	for _, fixture := range policy.Fixtures {
		if cohort == "" || fixture.Cohort == cohort {
			values = append(values, float64(fixture.Score))
		}
	}
	return values
}

func reconstructEvaluationDerivations(report *EvaluationReport) {
	for index := range report.Policies {
		report.Policies[index].Overall = aggregateEvaluation(string(report.Policies[index].Name), report.Policies[index].Fixtures)
		report.Policies[index].Cohorts = []Aggregate{}
		for _, cohort := range []Cohort{CohortCostSkewed, CohortBalanced, CohortEquivalence, CohortIrrelevant} {
			var subset []EvaluationFixture
			for _, fixture := range report.Policies[index].Fixtures {
				if fixture.Cohort == cohort {
					subset = append(subset, fixture)
				}
			}
			report.Policies[index].Cohorts = append(report.Policies[index].Cohorts, aggregateEvaluation(string(cohort), subset))
		}
	}
	learned, information := report.Policies[0], report.Policies[1]
	primary := pairedContrast("information-gain", evaluationScores(learned, ""), evaluationScores(information, ""))
	skewed := pairedContrast("cost-skewed", evaluationScores(learned, CohortCostSkewed), evaluationScores(information, CohortCostSkewed))
	report.Contrasts = []Contrast{primary, skewed}
	learnedAccurate, informationAccurate := learned.Overall.Accuracy == 1, information.Overall.Accuracy == 1
	for _, cohort := range learned.Cohorts {
		learnedAccurate = learnedAccurate && cohort.Accuracy == 1
	}
	for _, cohort := range information.Cohorts {
		informationAccurate = informationAccurate && cohort.Accuracy == 1
	}
	report.Gates = Gates{learnedAccurate, informationAccurate, primary.RelativeReduction >= .10, primary.PValue < .05, primary.CI95[0] > 0, skewed.RelativeReduction >= .10, skewed.PValue < .05, skewed.CI95[0] > 0}
	passed := map[string]bool{}
	for _, certificate := range report.ControlBundle.Certificates {
		passed[certificate.Name] = certificate.Passed
	}
	report.Controls = Controls{passed["hidden-twin"], passed["no-credit"], passed["wrong-context"], passed["static-rule"], passed["recomputed-rule"], passed["opaque-alias"], passed["presentation-order"], passed["proposal-order"], passed["cost-perturbation"], passed["occupied-name"], passed["alternate-descriptor"], passed["mutation-inert"], passed["corruption-suite"], passed["deterministic-json"]}
	m := &report.Mechanical
	m.DependencyBoundary = passed["dependency"]
	m.ProfileValid = len(report.Policies) == len(evaluationPolicies)
	m.OracleAgreements, m.OracleDisagreements, m.MaxFixtureRecordBytes = 0, 0, 0
	expectedMean := report.DynamicBenchmark.UniformExpectedMeanCost
	report.DynamicBenchmark = DynamicBenchmark{UniformExpectedMeanCost: expectedMean}
	for policyIndex, policy := range report.Policies {
		for _, fixture := range policy.Fixtures {
			m.OracleAgreements += fixture.OracleAgreements
			m.OracleDisagreements += fixture.OracleDisagreements
			m.MaxFixtureRecordBytes = maxInt(m.MaxFixtureRecordBytes, len(mustCanonical(fixture)))
			if policy.Name == "dynamic-optimal" {
				report.DynamicBenchmark.RealizedMeanCost += float64(fixture.InterventionCost)
				counts := fixture.MeterItems[5].Counter()
				report.DynamicBenchmark.TotalDPStates += int(counts.MemoStates)
				report.DynamicBenchmark.MaxDPStates = maxInt(report.DynamicBenchmark.MaxDPStates, int(counts.MemoStates))
				report.DynamicBenchmark.TotalDPWork += int(counts.TotalWork)
				report.DynamicBenchmark.MaxDPWork = maxInt(report.DynamicBenchmark.MaxDPWork, int(counts.TotalWork))
			}
		}
		m.ProfileValid = m.ProfileValid && policyIndex < len(evaluationPolicies) && policy.Name == evaluationPolicies[policyIndex]
	}
	if len(report.Policies) == len(evaluationPolicies) && len(report.Policies[6].Fixtures) != 0 {
		report.DynamicBenchmark.RealizedMeanCost /= float64(len(report.Policies[6].Fixtures))
	}
	allMeters := true
	for _, meter := range m.Meters {
		allMeters = allMeters && meter.Valid
	}
	allControls := true
	for _, certificate := range report.ControlBundle.Certificates {
		allControls = allControls && certificate.Passed
	}
	m.AllCapsValid = allMeters && m.MaxFixtureRecordBytes <= report.Manifest.FixtureRecordByteCap && report.DynamicBenchmark.MaxDPStates <= report.Manifest.DynamicStateCap && report.DynamicBenchmark.MaxDPWork <= report.Manifest.DynamicWorkCap
	m.AllValid = m.AllCapsValid && m.OracleDisagreements == 0 && m.DependencyBoundary && m.ProfileValid && m.TranscriptValid && m.TrainingFreezeValid && allControls
	report.Status = "invalid"
	if m.AllValid {
		report.Status = "valid-null"
		if report.Panel == "locked" && primary.Passed && skewed.Passed && report.Gates.LearnedAccuracy && report.Gates.InformationGainAccuracy {
			report.Status = "valid-positive"
		}
	}
}
