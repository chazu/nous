package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
)

type curriculumOutcome struct {
	Applications             []ApplicationCertificate
	Aggregates               []RuleAggregate
	WinnerTies               []string
	SelectedRule             string
	Unresolved               bool
	Counts                   Counter
	TaskMeterItems           []TaskMeterItem
	TerminalTranscriptDigest string
}

type trainingEpisodeRecord struct {
	executed executedEpisode
}

// These adapters fail closed until their owning packages provide the accepted
// typed APIs. ExecuteTraining preflights them before creating an attempt.
var centralCurriculumAdapter func(context.Context, []byte, [][]byte, [][]byte) (curriculumOutcome, error)
var controlSuiteAdapter func(context.Context, string, executedEpisode, curriculumOutcome, []byte, [][]byte, [][]byte, []PrivateFixture, PrivateFixture, *ControlEvidence) (ControlBundle, error)

func orchestrationAvailable() error {
	if centralCurriculumAdapter == nil {
		return errors.New("causal curriculum API is not linked")
	}
	if controlSuiteAdapter == nil {
		return errors.New("causal control API is not linked")
	}
	return nil
}

// ExecuteTraining owns the complete protected training operation. It accepts
// no seeds, report fields, validity booleans, executable, or publication path.
func ExecuteTraining(ctx context.Context, repoRoot string) (returnErr error) {
	if err := orchestrationAvailable(); err != nil {
		return err
	}
	capability, err := beginTrainingAttempt(ctx, repoRoot)
	if err != nil {
		return err
	}
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
	evidence, err := regenerateTrainingEvidence(ctx, repoRoot, capability)
	if err != nil {
		return err
	}
	verified, err := contextuallyVerifyTrainingEvidence(ctx, repoRoot, evidence.Report, evidence.Bundle)
	if err != nil {
		return err
	}
	capability.mu.Lock()
	capability.expectedReportDigest = verified.Report.TrainingDigest
	capability.expectedBundleDigest = verified.Bundle.BundleDigest
	capability.mu.Unlock()
	return capability.publishTrainingEvidence(repoRoot, evidence)
}

func regenerateTrainingEvidence(ctx context.Context, repoRoot string, capability *attemptCapability) (EvidenceBytes, error) {
	if capability == nil || capability.record.Panel != PanelTraining || capability.record.PretrainingCommit == "" {
		return EvidenceBytes{}, errors.New("missing training regeneration authority")
	}
	manifest := causalv2.PreregisteredManifest()
	rules := orderedRules()
	lexicalFirst := rules[0].Code()
	bundle := TrainingBundle{BundleVersion: "causal-training-episode-bundle/v2", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: capability.record.PretrainingCommit, Fixtures: []PrivateFixture{}, Episodes: []EpisodeEvidence{}}
	var records []trainingEpisodeRecord
	var applications []ApplicationCertificate
	var fixtureDigests []string
	var replayTasks []TaskMeterItem
	maxDescriptor, maxEpisode, maxCertificate := 0, 0, 0
	for seedIndex := 0; seedIndex < manifest.TrainingSeeds.Count; seedIndex++ {
		seed := manifest.TrainingSeeds.Start + int64(seedIndex)*manifest.TrainingSeeds.Step
		fixture, err := capability.generateFixture(seed, seedIndex)
		if err != nil {
			return EvidenceBytes{}, err
		}
		bundle.Fixtures = append(bundle.Fixtures, fixture)
		fixtureDigests = append(fixtureDigests, fixture.PublicFixture.FixtureDigest)
		for _, rule := range rules {
			episode, err := runEpisode(ctx, PanelTraining, fixture, rule.Code(), rule.Code(), rule.Code() == lexicalFirst)
			if err != nil {
				return EvidenceBytes{}, fmt.Errorf("training seed %d rule %s: %w", seed, rule.Code(), err)
			}
			bundle.Episodes = append(bundle.Episodes, episode.evidence)
			records = append(records, trainingEpisodeRecord{episode})
			applications = append(applications, episode.certificate)
			replayTasks = append(replayTasks, TaskMeterItem{Name: "certificate-replay", Subject: episode.certificate.CertificateDigest, Counts: counts64(episode.replay.ProductionCounts)})
			maxDescriptor = maxInt(maxDescriptor, len(episode.profileBytes))
			maxEpisode = maxInt(maxEpisode, len(mustCanonical(episode.evidence)))
			maxCertificate = maxInt(maxCertificate, len(mustCanonical(episode.certificate)))
		}
	}
	bundleBytes, err := FinalizeTrainingBundle(&bundle)
	if err != nil {
		return EvidenceBytes{}, err
	}
	centralProfile := causalv2.CentralProfile{CentralProfileVersion: causalv2.CentralProfileDomain, Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: capability.record.PretrainingCommit, CreditEnabled: true}
	if err := causalv2.SignCentralProfile(&centralProfile); err != nil {
		return EvidenceBytes{}, err
	}
	centralProfileBytes := mustCanonical(centralProfile)
	certificateBytes := make([][]byte, len(applications))
	episodeBytes := make([][]byte, len(bundle.Episodes))
	for i := range applications {
		certificateBytes[i] = mustCanonical(applications[i])
		episodeBytes[i] = mustCanonical(bundle.Episodes[i])
	}
	curriculum, err := centralCurriculumAdapter(ctx, centralProfileBytes, episodeBytes, certificateBytes)
	if err != nil {
		return EvidenceBytes{}, err
	}
	if curriculum.SelectedRule == "" || curriculum.Unresolved || len(curriculum.Applications) != 480 || len(curriculum.Aggregates) != 40 {
		return EvidenceBytes{}, errors.New("curriculum did not produce the complete selected training matrix")
	}
	for i := range applications {
		if !bytes.Equal(mustCanonical(applications[i]), mustCanonical(curriculum.Applications[i])) {
			return EvidenceBytes{}, errors.New("curriculum application order differs from certificates")
		}
	}
	postSelectionTasks := make([]TaskMeterItem, 0, len(records))
	for _, record := range records {
		replayed, err := causalrun.VerifyEpisode(record.executed.publicBytes, record.executed.profileBytes, record.executed.result.Artifacts)
		if err != nil || !bytes.Equal(mustCanonical(replayed), mustCanonical(record.executed.replay)) {
			return EvidenceBytes{}, errors.New("post-selection replay differs from certificate replay")
		}
		postSelectionTasks = append(postSelectionTasks, TaskMeterItem{Name: "post-selection-replay", Subject: record.executed.certificate.CertificateDigest, Counts: counts64(replayed.ProductionCounts)})
	}
	var corruptionFixture PrivateFixture
	if capability.replayCorruptionFixture != nil {
		corruptionFixture = *capability.replayCorruptionFixture
	} else {
		corruptionFixture, err = generate(PanelDevelopment, manifest.DevelopmentSeeds.Start, 0)
		if err != nil {
			return EvidenceBytes{}, err
		}
	}
	var controlEvidence ControlEvidence
	controlBundle, err := controlSuiteAdapter(ctx, repoRoot, records[0].executed, curriculum, centralProfileBytes, episodeBytes, certificateBytes, bundle.Fixtures, corruptionFixture, &controlEvidence)
	if err != nil {
		return EvidenceBytes{}, err
	}
	controlBytes := mustCanonical(controlBundle)
	verifiedControls, err := causalv2.VerifyControlBundle(controlBytes)
	if err != nil {
		return EvidenceBytes{}, err
	}
	if err := verifyExecutedControlBundle(verifiedControls); err != nil {
		return EvidenceBytes{}, err
	}
	verifiedEvidence, err := causalv2.VerifyControlEvidence(mustCanonical(controlEvidence))
	if err != nil {
		return EvidenceBytes{}, err
	}
	if err := verifyRetainedControlEvidence(verifiedControls, verifiedEvidence); err != nil {
		return EvidenceBytes{}, err
	}
	taskItems := append(append(replayTasks, postSelectionTasks...), curriculum.TaskMeterItems...)
	taskDigest, err := causalv2.TaskMeterItemsDigest(taskItems)
	if err != nil {
		return EvidenceBytes{}, err
	}
	episodeItems := make([][]MeterItem, len(bundle.Episodes))
	for i := range bundle.Episodes {
		episodeItems[i] = bundle.Episodes[i].MeterItems
	}
	controlCounts := make([][15]int64, len(verifiedControls.Certificates))
	for i := range verifiedControls.Certificates {
		controlCounts[i] = verifiedControls.Certificates[i].MeterCounts
	}
	meters, meterDigest, err := ReconstructMeters(MeterTraining, episodeItems, taskItems, controlCounts)
	if err != nil {
		return EvidenceBytes{}, err
	}
	allCaps := true
	for _, meter := range meters {
		allCaps = allCaps && meter.Valid
	}
	report := TrainingReport{ReportVersion: "causal-training-report/v2", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: capability.record.PretrainingCommit, TrainingReportCommit: "", EpisodeBundleDigest: bundle.BundleDigest, EpisodeBundleBytes: len(bundleBytes), ControlBundle: verifiedControls, ControlBundleDigest: verifiedControls.ControlBundleDigest, ControlEvidence: verifiedEvidence, ControlEvidenceDigest: verifiedEvidence.ControlEvidenceDigest, TaskMeterItems: taskItems, TaskMeterItemsDigest: taskDigest, Panel: "training", Status: "invalid", FixtureDigests: fixtureDigests, Applications: applications, Rules: curriculum.Aggregates, WinnerTies: curriculum.WinnerTies, SelectedRule: curriculum.SelectedRule, Limitations: []string{"Synthetic deterministic three-variable SCMs do not establish production causal diagnosis."}}
	trainingInput := TrainingDigestInput{DigestInputVersion: "causal-training-digest-input/v2", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: capability.record.PretrainingCommit, CentralProfileDigest: centralProfile.ProfileDigest, EpisodeBundleDigest: bundle.BundleDigest, ControlBundleDigest: verifiedControls.ControlBundleDigest, ControlEvidenceDigest: verifiedEvidence.ControlEvidenceDigest, TaskMeterItemsDigest: taskDigest, MeterDigest: meterDigest, FixtureDigests: fixtureDigests, ApplicationCertificates: applications, RuleAggregates: curriculum.Aggregates, WinnerTies: curriculum.WinnerTies, SelectedRule: curriculum.SelectedRule}
	report.TrainingDigest, err = TrainingDigest(trainingInput)
	if err != nil {
		return EvidenceBytes{}, err
	}
	agreements, disagreements := 0, 0
	for _, episode := range bundle.Episodes {
		agreements += episode.OracleAgreements
		disagreements += episode.OracleDisagreements
	}
	controlsPassed := true
	for _, certificate := range verifiedControls.Certificates {
		controlsPassed = controlsPassed && certificate.Passed
	}
	selectionVerified := selectionMatchesAggregates(curriculum.SelectedRule, curriculum.WinnerTies, curriculum.Aggregates)
	report.Controls = trainingControlBooleans(verifiedControls)
	creditRecomputed := len(curriculum.Applications) == len(applications) && len(curriculum.Aggregates) == len(orderedRules()) && selectionMatchesAggregates(curriculum.SelectedRule, curriculum.WinnerTies, curriculum.Aggregates)
	report.Mechanical = TrainingMechanical{AllValid: false, CreditRecomputed: creditRecomputed, SelectionVerified: selectionVerified, OracleAgreements: agreements, OracleDisagreements: disagreements, Meters: meters, MaxDescriptorBytes: maxDescriptor, MaxTrainingEpisodeReportBytes: maxEpisode, MaxApplicationCertificateBytes: maxCertificate, AllCapsValid: allCaps}
	report.Mechanical.AllValid = allCaps && disagreements == 0 && selectionVerified && controlsPassed
	if report.Mechanical.AllValid {
		report.Status = "valid"
	}
	reportBytes, err := FinalizeTrainingReport(&report)
	if err != nil {
		return EvidenceBytes{}, err
	}
	if _, err := VerifyTrainingReportBytes(reportBytes); err != nil {
		return EvidenceBytes{}, err
	}
	if err := VerifyTrainingEvidence(report, bundle); err != nil {
		return EvidenceBytes{}, err
	}
	return EvidenceBytes{Report: reportBytes, Bundle: bundleBytes}, nil
}

func selectionMatchesAggregates(selected string, ties []string, aggregates []RuleAggregate) bool {
	if len(aggregates) != 40 || selected == "" {
		return false
	}
	best := aggregates[0]
	for _, candidate := range aggregates[1:] {
		if candidate.Worth > best.Worth || candidate.Worth == best.Worth && candidate.BudgetExhausted < best.BudgetExhausted || candidate.Worth == best.Worth && candidate.BudgetExhausted == best.BudgetExhausted && candidate.Code < best.Code {
			best = candidate
		}
	}
	var wantTies []string
	for _, candidate := range aggregates {
		if candidate.Worth == best.Worth && candidate.BudgetExhausted == best.BudgetExhausted {
			wantTies = append(wantTies, candidate.Code)
		}
	}
	sort.Strings(wantTies)
	return selected == wantTies[0] && slices.Equal(ties, wantTies)
}

func trainingControlBooleans(bundle ControlBundle) TrainingControls {
	passed := map[string]bool{}
	for _, certificate := range bundle.Certificates {
		passed[certificate.Name] = certificate.Passed
	}
	return TrainingControls{NoCreditChangesSelection: passed["no-credit"], HiddenTwin: passed["hidden-twin"], WrongContext: passed["wrong-context"], StaticRule: passed["static-rule"], DeterministicJSON: passed["deterministic-json"]}
}

func maxInt(left, right int) int {
	if right > left {
		return right
	}
	return left
}
