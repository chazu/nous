package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
)

// verifiedTrainingEvidence is the package-private authority minted only after
// exact bytes, fresh execution, retained controls, and the development witness
// all reconstruct in repository context.
type verifiedTrainingEvidence struct {
	Report            TrainingReport
	Bundle            TrainingBundle
	CorruptionFixture PrivateFixture
}

// verifiedEvaluationEvidence is the package-private authority produced only
// after the committed training context, complete protected matrix, controls,
// mechanics, and canonical report bytes all reconstruct at the recorded C.
type verifiedEvaluationEvidence struct {
	Report   EvaluationReport
	Training verifiedTrainingEvidence
}

func contextuallyVerifyTrainingEvidence(ctx context.Context, repoRoot string, reportBytes, bundleBytes []byte) (verifiedTrainingEvidence, error) {
	_ = ctx
	if repoRoot == "" {
		return verifiedTrainingEvidence{}, errors.New("contextual training verification requires repository root")
	}
	report, err := VerifyTrainingReportBytes(reportBytes)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	bundle, err := VerifyTrainingBundleBytes(bundleBytes)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	if err := VerifyTrainingEvidence(report, bundle); err != nil {
		return verifiedTrainingEvidence{}, err
	}
	fixture, err := verifyFixedCorruptionWitness(ctx, report.ControlEvidence.Corruption)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	central := causalv2.CentralProfile{CentralProfileVersion: causalv2.CentralProfileDomain, Manifest: report.Manifest, PlanCommit: report.PlanCommit, PretrainingCommit: report.PretrainingCommit, CreditEnabled: true}
	if err := causalv2.SignCentralProfile(&central); err != nil {
		return verifiedTrainingEvidence{}, err
	}
	episodes, certificates := make([][]byte, len(bundle.Episodes)), make([][]byte, len(report.Applications))
	for index := range episodes {
		episodes[index], certificates[index] = mustCanonical(bundle.Episodes[index]), mustCanonical(report.Applications[index])
	}
	curriculum, err := centralCurriculumAdapter(ctx, mustCanonical(central), episodes, certificates)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	reference, err := runEpisode(ctx, PanelTraining, bundle.Fixtures[0], report.SelectedRule, "", false)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	var freshEvidence ControlEvidence
	freshBundle, err := executeControlSuite(ctx, repoRoot, reference, curriculum, mustCanonical(central), episodes, certificates, bundle.Fixtures, fixture, &freshEvidence)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	dependencySummary, err := causalrun.AuditDependencyBoundary(repoRoot)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	freshEvidence.Dependency, err = rootedDependencyProofAt(repoRoot, report.PretrainingCommit, dependencySummary)
	if err != nil {
		return verifiedTrainingEvidence{}, err
	}
	freshEvidence.ControlEvidenceDigest = ""
	if err := causalv2.SignControlEvidence(&freshEvidence); err != nil {
		return verifiedTrainingEvidence{}, err
	}
	if !bytes.Equal(mustCanonical(freshBundle), mustCanonical(report.ControlBundle)) || !bytes.Equal(mustCanonical(freshEvidence), mustCanonical(report.ControlEvidence)) {
		return verifiedTrainingEvidence{}, errors.New("contextual controls differ from complete fresh reconstruction")
	}
	return verifiedTrainingEvidence{Report: report, Bundle: bundle, CorruptionFixture: fixture}, nil
}

func contextuallyVerifyEvaluationEvidence(ctx context.Context, repoRoot string, reportBytes []byte) (verifiedEvaluationEvidence, error) {
	if repoRoot == "" {
		return verifiedEvaluationEvidence{}, errors.New("contextual evaluation verification requires repository root")
	}
	report, err := VerifyEvaluationReportBytes(reportBytes)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	if err := VerifyEvaluationEvidence(report); err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	if !state.Clean || state.Head != report.ImplementationCommit {
		return verifiedEvaluationEvidence{}, errors.New("contextual evaluation verification is not at clean recorded candidate C")
	}
	trainingBytes, err := gitFile(ctx, state.Root, report.TrainingReportCommit, TrainingEvidenceDirectory+"/"+TrainingReportName)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	bundleBytes, err := gitFile(ctx, state.Root, report.TrainingReportCommit, TrainingEvidenceDirectory+"/"+TrainingEpisodesName)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	training, err := contextuallyVerifyTrainingEvidence(ctx, state.Root, trainingBytes, bundleBytes)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	if report.TrainingReportCommit != FrozenTrainingReportCommit || report.TrainingDigest != training.Report.TrainingDigest || report.FrozenRule != training.Report.SelectedRule || report.PretrainingCommit != training.Report.PretrainingCommit {
		return verifiedEvaluationEvidence{}, errors.New("evaluation report differs from contextually verified frozen training identity")
	}
	var seeds causalv2.SeedRange
	switch Panel(report.Panel) {
	case PanelValidation:
		seeds = report.Manifest.ValidationSeeds
	case PanelLocked:
		seeds = report.Manifest.LockedSeeds
	default:
		return verifiedEvaluationEvidence{}, errors.New("contextual authority is available only for validation or locked reports")
	}
	fixtures := make([]PrivateFixture, seeds.Count)
	for index := range fixtures {
		seed := seeds.Start + int64(index)*seeds.Step
		fixtures[index], err = generate(Panel(report.Panel), seed, index)
		if err != nil {
			return verifiedEvaluationEvidence{}, err
		}
	}
	freshBytes, err := buildEvaluationReport(ctx, state.Root, Panel(report.Panel), report.ImplementationCommit, report.TrainingReportCommit, fixtures, training)
	if err != nil {
		return verifiedEvaluationEvidence{}, err
	}
	if !bytes.Equal(reportBytes, freshBytes) {
		return verifiedEvaluationEvidence{}, errors.New("evaluation report differs from complete contextual reconstruction")
	}
	return verifiedEvaluationEvidence{Report: report, Training: training}, nil
}

func verifyFixedCorruptionWitness(ctx context.Context, proof causalv2.CorruptionProof) (PrivateFixture, error) {
	manifest := causalv2.PreregisteredManifest()
	freshFixture, err := generate(PanelDevelopment, manifest.DevelopmentSeeds.Start, 0)
	if err != nil {
		return PrivateFixture{}, err
	}
	retainedFixtureBytes, err := proof.FixtureBytes.Bytes()
	if err != nil || !bytes.Equal(retainedFixtureBytes, mustCanonical(freshFixture)) {
		return PrivateFixture{}, errors.New("corruption witness fixture differs from fixed development seed 112001 attempt zero")
	}
	episode, err := runEpisode(ctx, PanelDevelopment, freshFixture, "P=H;M=gain;S=C", "", false)
	if err != nil {
		return PrivateFixture{}, err
	}
	if episode.result.Terminal != "identified" || len(episode.result.Actions) != 2 || len(episode.result.Artifacts) != 77 {
		return PrivateFixture{}, fmt.Errorf("fixed corruption witness shape=%s/%d/%d, want identified/2/77", episode.result.Terminal, len(episode.result.Actions), len(episode.result.Artifacts))
	}
	profileBytes, err := proof.ProfileBytes.Bytes()
	if err != nil || !bytes.Equal(profileBytes, episode.profileBytes) {
		return PrivateFixture{}, errors.New("corruption witness profile differs from fixed development execution")
	}
	if len(proof.BaselineArtifacts) != len(episode.result.Artifacts) {
		return PrivateFixture{}, errors.New("corruption witness ledger cardinality differs from fixed execution")
	}
	wantCounts := map[string]int{"descriptor-snapshot": 5, "observation": 1, "posterior": 3, "cache": 12, "partition": 12, "proposal": 12, "score": 12, "tie": 2, "selection": 2, "authorization": 2, "result": 2, "elimination": 7, "consumption": 2, "transcript": 2, "terminal": 1}
	gotCounts := make(map[string]int, len(wantCounts))
	for index, retained := range proof.BaselineArtifacts {
		encoded, decodeErr := retained.Bytes()
		if decodeErr != nil || !bytes.Equal(encoded, episode.result.Artifacts[index]) {
			return PrivateFixture{}, fmt.Errorf("corruption witness artifact %d differs from fixed ledger", index)
		}
		artifact, verifyErr := causalv2.VerifyArtifact(encoded)
		if verifyErr != nil {
			return PrivateFixture{}, verifyErr
		}
		gotCounts[artifact.Kind]++
	}
	if !equalCanonical(gotCounts, wantCounts) {
		return PrivateFixture{}, fmt.Errorf("corruption witness kind counts=%v, want %v", gotCounts, wantCounts)
	}
	return freshFixture, nil
}
