package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalcurriculum"
	"github.com/chazu/nous/internal/causalv2"
)

type TrainingDigestInput struct {
	DigestInputVersion      string                   `json:"digest_input_version"`
	Manifest                causalv2.Manifest        `json:"manifest"`
	PlanCommit              string                   `json:"plan_commit"`
	PretrainingCommit       string                   `json:"pretraining_commit"`
	CentralProfileDigest    string                   `json:"central_profile_digest"`
	EpisodeBundleDigest     string                   `json:"episode_bundle_digest"`
	ControlBundleDigest     string                   `json:"control_bundle_digest"`
	ControlEvidenceDigest   string                   `json:"control_evidence_digest"`
	TaskMeterItemsDigest    string                   `json:"task_meter_items_digest"`
	MeterDigest             string                   `json:"meter_digest"`
	FixtureDigests          []string                 `json:"fixture_digests"`
	ApplicationCertificates []ApplicationCertificate `json:"application_certificates"`
	RuleAggregates          []RuleAggregate          `json:"rule_aggregates"`
	WinnerTies              []string                 `json:"winner_ties"`
	SelectedRule            string                   `json:"selected_rule"`
}

type CentralTranscriptEvent struct {
	EventVersion          string `json:"event_version"`
	Index                 int    `json:"index"`
	PreviousDigest        string `json:"previous_digest"`
	Kind                  string `json:"kind"`
	SubjectArtifactDigest string `json:"subject_artifact_digest"`
	WorkBefore            int64  `json:"work_before"`
	WorkAfter             int64  `json:"work_after"`
	EventDigest           string `json:"event_digest"`
}

func SignApplicationCertificate(certificate *ApplicationCertificate) error {
	return causalv2.SignApplicationCertificate(certificate)
}

func VerifyApplicationCertificateBytes(data []byte) (ApplicationCertificate, error) {
	return causalv2.VerifyApplicationCertificate(data)
}

func VerifyTrainingBundleBytes(data []byte) (TrainingBundle, error) {
	bundle, err := causalv2.StrictDecode[TrainingBundle](data)
	if err != nil {
		return bundle, err
	}
	if len(bundle.Fixtures) != 12 || len(bundle.Episodes) != 480 {
		return bundle, errors.New("training bundle does not contain 12 fixtures and 480 episodes")
	}
	manifest := causalv2.PreregisteredManifest()
	for i, fixture := range bundle.Fixtures {
		wantSeed := manifest.TrainingSeeds.Start + int64(i)*manifest.TrainingSeeds.Step
		if fixture.PublicFixture.Seed != wantSeed {
			return bundle, errors.New("training fixtures are not in seed order")
		}
		if _, verifyErr := causalv2.VerifyPrivateFixture(mustCanonical(fixture)); verifyErr != nil {
			return bundle, verifyErr
		}
		encoded, _ := causalv2.CanonicalJSON(fixture)
		if len(encoded) > manifest.TrainingFixtureByteCap {
			return bundle, errors.New("private training fixture exceeds byte cap")
		}
	}
	rulesBySeed := make(map[int64][]EpisodeEvidence, 12)
	rules := orderedRules()
	for episodeIndex, episode := range bundle.Episodes {
		if err := VerifyEpisode(episode); err != nil {
			return bundle, err
		}
		seedIndex, ruleIndex := episodeIndex/len(rules), episodeIndex%len(rules)
		wantSeed := manifest.TrainingSeeds.Start + int64(seedIndex)*manifest.TrainingSeeds.Step
		if episode.Seed != wantSeed || episode.RuleCode != rules[ruleIndex].Code() {
			return bundle, fmt.Errorf("episode %d is not in exact seed-major/rule-major order", episodeIndex)
		}
		rulesBySeed[episode.Seed] = append(rulesBySeed[episode.Seed], episode)
	}
	lexicalFirst := rules[0].Code()
	for _, fixture := range bundle.Fixtures {
		episodes := rulesBySeed[fixture.PublicFixture.Seed]
		if len(episodes) != 40 {
			return bundle, fmt.Errorf("seed %d has %d episodes", fixture.PublicFixture.Seed, len(episodes))
		}
		for _, episode := range episodes {
			if episode.FixtureDigest != fixture.PublicFixture.FixtureDigest {
				return bundle, errors.New("episode fixture digest mismatch")
			}
			if episode.MeterItems[0].Name != "production" || !episode.MeterItems[0].Active || !episode.MeterItems[1].Active || !episode.MeterItems[4].Active {
				return bundle, errors.New("required training episode meter inactive")
			}
			if episode.MeterItems[5].Active != (episode.RuleCode == lexicalFirst) {
				return bundle, errors.New("training DP ownership is not the lexical-first rule")
			}
			valid, err := episodeCapsValid(MeterTraining, episode.MeterItems)
			if err != nil || episode.AllCapsValid != valid {
				return bundle, errors.New("training episode all_caps_valid is not reconstructed")
			}
		}
	}
	copy := bundle
	want, err := FinalizeTrainingBundle(&copy)
	if err != nil || !bytes.Equal(data, want) {
		return bundle, errors.New("training bundle digest or bytes do not reconstruct")
	}
	return bundle, nil
}

func episodeCapsValid(scope MeterScope, items []MeterItem) (bool, error) {
	if err := causalv2.ValidateMeterItems(items); err != nil {
		return false, err
	}
	valid := true
	for _, item := range items {
		if !item.Active {
			continue
		}
		meter, err := causalv2.NewAggregateMeter(item.Name, string(scope), []Counter{item.Counter()})
		if err != nil {
			return false, err
		}
		valid = valid && meter.Valid
	}
	return valid, nil
}

func VerifyTrainingEvidence(report TrainingReport, bundle TrainingBundle) error {
	if report.TrainingReportCommit != "" || report.EpisodeBundleDigest != bundle.BundleDigest || report.EpisodeBundleBytes != len(mustCanonical(bundle)) || report.PlanCommit != PlanCommit || bundle.PlanCommit != PlanCommit || report.PretrainingCommit != bundle.PretrainingCommit {
		return errors.New("training report and bundle provenance mismatch")
	}
	if len(report.Applications) != 480 || len(report.Rules) != 40 || len(report.FixtureDigests) != 12 || len(bundle.Fixtures) != 12 || len(bundle.Episodes) != 480 {
		return errors.New("training evidence matrix cardinality mismatch")
	}
	for i, fixture := range bundle.Fixtures {
		if report.FixtureDigests[i] != fixture.PublicFixture.FixtureDigest {
			return errors.New("training fixture digest list mismatch")
		}
	}
	for i, certificate := range report.Applications {
		encoded, _ := causalv2.CanonicalJSON(certificate)
		if _, err := VerifyApplicationCertificateBytes(encoded); err != nil {
			return fmt.Errorf("application %d: %w", i, err)
		}
		if certificate.EpisodeReportDigest != bundle.Episodes[i].EpisodeReportDigest {
			return errors.New("application does not bind corresponding episode")
		}
	}
	maxDescriptor, err := verifyFreshTrainingMatrix(report, bundle)
	if err != nil {
		return err
	}
	if err := verifyExecutedControlBundle(report.ControlBundle); err != nil {
		return err
	}
	verifiedEvidence, err := causalv2.VerifyControlEvidence(mustCanonical(report.ControlEvidence))
	if err != nil || verifiedEvidence.ControlEvidenceDigest != report.ControlEvidenceDigest {
		return errors.New("training control evidence does not verify")
	}
	if err := verifyRetainedControlEvidence(report.ControlBundle, verifiedEvidence); err != nil {
		return err
	}
	if err := verifyFreshMatrixProofs(context.Background(), report.ControlBundle, verifiedEvidence, bundle.Fixtures, bundle.Episodes, report.Applications); err != nil {
		return err
	}
	rules := orderedRules()
	if len(report.Rules) != len(rules) {
		return errors.New("training rule aggregate count mismatch")
	}
	for ruleIndex, rule := range rules {
		var ruleApplications []ApplicationCertificate
		for seedIndex := 0; seedIndex < 12; seedIndex++ {
			ruleApplications = append(ruleApplications, report.Applications[seedIndex*len(rules)+ruleIndex])
		}
		want := RuleAggregate{Code: rule.Code(), Applications: len(ruleApplications)}
		for _, application := range ruleApplications {
			want.TotalScore += application.Score
			want.TotalCost += application.Cost
			switch application.Terminal {
			case "identified":
				want.Identified++
			case "equivalence":
				want.Equivalence++
			default:
				want.BudgetExhausted++
			}
		}
		want.Worth = want.Applications*report.Manifest.InvalidOrExhaustedScore - want.TotalScore
		want.ApplicationDigest, _ = causalv2.Digest("causal-rule-applications/v2", ruleApplications)
		if !bytes.Equal(mustCanonical(want), mustCanonical(report.Rules[ruleIndex])) {
			return fmt.Errorf("rule aggregate %q does not reconstruct", rule.Code())
		}
	}
	if !selectionMatchesAggregates(report.SelectedRule, report.WinnerTies, report.Rules) {
		return errors.New("selected rule does not reconstruct from aggregates")
	}
	wantSelection := selectionMatchesAggregates(report.SelectedRule, report.WinnerTies, report.Rules)
	if report.Mechanical.SelectionVerified != wantSelection || report.Mechanical.CreditRecomputed != wantSelection {
		return errors.New("training selection/credit mechanical fields do not reconstruct")
	}
	if len(report.TaskMeterItems) != 1485 {
		return fmt.Errorf("training task meter items=%d, want 1485", len(report.TaskMeterItems))
	}
	for i, certificate := range report.Applications {
		if report.TaskMeterItems[i].Name != "certificate-replay" || report.TaskMeterItems[i].Subject != certificate.CertificateDigest || report.TaskMeterItems[480+i].Name != "post-selection-replay" || report.TaskMeterItems[480+i].Subject != certificate.CertificateDigest {
			return errors.New("training replay task order does not match seed-major/rule-major certificates")
		}
	}
	episodeItems := make([][]MeterItem, len(bundle.Episodes))
	for i := range bundle.Episodes {
		episodeItems[i] = bundle.Episodes[i].MeterItems
	}
	controlItems := make([][15]int64, len(report.ControlBundle.Certificates))
	for i, certificate := range report.ControlBundle.Certificates {
		controlItems[i] = certificate.MeterCounts
	}
	if err := VerifyAggregateMeters(report.Mechanical.Meters, MeterTraining, episodeItems, report.TaskMeterItems, controlItems); err != nil {
		return err
	}
	if err := verifyMeterCardinality(MeterTraining, report.Mechanical.Meters, 12); err != nil {
		return err
	}
	allCaps := true
	for _, meter := range report.Mechanical.Meters {
		allCaps = allCaps && meter.Valid
	}
	if report.Mechanical.AllCapsValid != allCaps {
		return errors.New("training all_caps_valid is not reconstructed")
	}
	maxEpisode, maxCertificate, agreements, disagreements := 0, 0, 0, 0
	for i, episode := range bundle.Episodes {
		maxEpisode = maxInt(maxEpisode, len(mustCanonical(episode)))
		maxCertificate = maxInt(maxCertificate, len(mustCanonical(report.Applications[i])))
		agreements += episode.OracleAgreements
		disagreements += episode.OracleDisagreements
	}
	if report.Mechanical.MaxTrainingEpisodeReportBytes != maxEpisode || report.Mechanical.MaxApplicationCertificateBytes != maxCertificate || report.Mechanical.OracleAgreements != agreements || report.Mechanical.OracleDisagreements != disagreements || report.Mechanical.MaxDescriptorBytes != maxDescriptor {
		return errors.New("training mechanical maxima or oracle totals do not reconstruct")
	}
	wantControls := trainingControlBooleans(report.ControlBundle)
	if report.Controls != wantControls {
		return errors.New("training control booleans do not reconstruct from certificates")
	}
	allControls := true
	for _, certificate := range report.ControlBundle.Certificates {
		allControls = allControls && certificate.Passed
	}
	wantAllValid := allCaps && disagreements == 0 && report.Mechanical.SelectionVerified && report.Mechanical.CreditRecomputed && allControls
	if report.Mechanical.AllValid != wantAllValid || report.Status != map[bool]string{true: "valid", false: "invalid"}[wantAllValid] {
		return errors.New("training validity/status does not reconstruct")
	}
	if !bytes.Equal(mustCanonical(report.Limitations), mustCanonical([]string{"Synthetic deterministic three-variable SCMs do not establish production causal diagnosis."})) {
		return errors.New("training limitations differ from the preregistered limitation")
	}
	for _, limitation := range report.Limitations {
		if len(mustCanonical(limitation)) > report.Manifest.LimitationByteCap {
			return errors.New("training limitation exceeds byte cap")
		}
	}
	central := causalv2.CentralProfile{CentralProfileVersion: causalv2.CentralProfileDomain, Manifest: report.Manifest, PlanCommit: report.PlanCommit, PretrainingCommit: report.PretrainingCommit, CreditEnabled: true}
	if err := causalv2.SignCentralProfile(&central); err != nil {
		return err
	}
	episodeBytes := make([][]byte, len(bundle.Episodes))
	certificateBytes := make([][]byte, len(report.Applications))
	for index := range bundle.Episodes {
		episodeBytes[index] = mustCanonical(bundle.Episodes[index])
		certificateBytes[index] = mustCanonical(report.Applications[index])
	}
	curriculum, err := causalcurriculum.Run(context.Background(), mustCanonical(central), episodeBytes, certificateBytes)
	if err != nil {
		return fmt.Errorf("fresh curriculum verification: %w", err)
	}
	if err := causalcurriculum.Verify(mustCanonical(central), episodeBytes, certificateBytes, curriculum); err != nil {
		return err
	}
	if !bytes.Equal(mustCanonical(curriculum.TaskMeterItems), mustCanonical(report.TaskMeterItems[960:])) || !bytes.Equal(mustCanonical(curriculum.Applications), mustCanonical(report.Applications)) || !bytes.Equal(mustCanonical(curriculum.Aggregates), mustCanonical(report.Rules)) || !bytes.Equal(mustCanonical(curriculum.WinnerTies), mustCanonical(report.WinnerTies)) || curriculum.SelectedRule != report.SelectedRule || curriculum.Unresolved {
		return errors.New("training curriculum evidence differs from fresh exact reconstruction")
	}
	noCreditProfile := central
	noCreditProfile.CreditEnabled = false
	noCreditProfile.TrainingKey, noCreditProfile.ProfileDigest = "", ""
	if err := causalv2.SignCentralProfile(&noCreditProfile); err != nil {
		return err
	}
	noCredit, err := causalcurriculum.Run(context.Background(), mustCanonical(noCreditProfile), episodeBytes, certificateBytes)
	if err != nil {
		return fmt.Errorf("fresh no-credit curriculum: %w", err)
	}
	if err := causalcurriculum.Verify(mustCanonical(noCreditProfile), episodeBytes, certificateBytes, noCredit); err != nil {
		return err
	}
	if !bytes.Equal(mustCanonical(noCredit.Applications), mustCanonical(report.Applications)) {
		return errors.New("no-credit run did not admit the exact 480 training certificates")
	}
	if !bytes.Equal(mustCanonical(noCreditProof(noCredit, mustCanonical(noCreditProfile), certificateBytes)), mustCanonical(report.ControlEvidence.NoCredit)) {
		return errors.New("no-credit retained admissions/artifacts/deltas differ from full fresh 480-application rerun")
	}
	_, meterDigest, err := ReconstructMeters(MeterTraining, episodeItems, report.TaskMeterItems, controlItems)
	if err != nil {
		return err
	}
	trainingInput := TrainingDigestInput{DigestInputVersion: "causal-training-digest-input/v2", Manifest: report.Manifest, PlanCommit: report.PlanCommit, PretrainingCommit: report.PretrainingCommit, CentralProfileDigest: central.ProfileDigest, EpisodeBundleDigest: report.EpisodeBundleDigest, ControlBundleDigest: report.ControlBundleDigest, ControlEvidenceDigest: report.ControlEvidenceDigest, TaskMeterItemsDigest: report.TaskMeterItemsDigest, MeterDigest: meterDigest, FixtureDigests: report.FixtureDigests, ApplicationCertificates: report.Applications, RuleAggregates: report.Rules, WinnerTies: report.WinnerTies, SelectedRule: report.SelectedRule}
	wantTrainingDigest, err := TrainingDigest(trainingInput)
	if err != nil || report.TrainingDigest != wantTrainingDigest {
		return errors.New("training digest does not reconstruct")
	}
	return nil
}

func verifyFreshTrainingMatrix(report TrainingReport, bundle TrainingBundle) (int, error) {
	rules := orderedRules()
	maxDescriptor := 0
	for seedIndex := 0; seedIndex < report.Manifest.TrainingSeeds.Count; seedIndex++ {
		seed := report.Manifest.TrainingSeeds.Start + int64(seedIndex)*report.Manifest.TrainingSeeds.Step
		fixture := bundle.Fixtures[seedIndex]
		if fixture.PublicFixture.Seed != seed || report.FixtureDigests[seedIndex] != fixture.PublicFixture.FixtureDigest {
			return 0, fmt.Errorf("training fixture %d differs from exact regeneration", seedIndex)
		}
		for ruleIndex, rule := range rules {
			index := seedIndex*len(rules) + ruleIndex
			episode, err := runEpisode(context.Background(), PanelTraining, fixture, rule.Code(), rule.Code(), ruleIndex == 0)
			if err != nil {
				return 0, fmt.Errorf("fresh training episode %d: %w", index, err)
			}
			if !bytes.Equal(mustCanonical(episode.evidence), mustCanonical(bundle.Episodes[index])) {
				return 0, fmt.Errorf("training episode %d differs from exact regeneration", index)
			}
			if !bytes.Equal(mustCanonical(episode.certificate), mustCanonical(report.Applications[index])) {
				return 0, fmt.Errorf("application certificate %d differs from exact regenerated episode", index)
			}
			maxDescriptor = maxInt(maxDescriptor, len(episode.profileBytes))
		}
	}
	return maxDescriptor, nil
}

var evaluationPolicies = []Policy{"learned", "information-gain-per-cost", "worst-split-per-cost", "lexical-fixed", "uniform-random", "passive-only", "dynamic-optimal"}

func VerifyEvaluationEvidence(report EvaluationReport) error {
	if report.ControlBundleDigest != report.ControlBundle.ControlBundleDigest || report.ControlEvidenceDigest != report.ControlEvidence.ControlEvidenceDigest {
		return errors.New("evaluation control digest copies differ")
	}
	if _, err := causalv2.VerifyControlBundle(mustCanonical(report.ControlBundle)); err != nil {
		return err
	}
	verifiedEvidence, err := causalv2.VerifyControlEvidence(mustCanonical(report.ControlEvidence))
	if err != nil {
		return err
	}
	if err := verifyExecutedControlBundle(report.ControlBundle); err != nil {
		return err
	}
	if err := verifyRetainedControlEvidence(report.ControlBundle, verifiedEvidence); err != nil {
		return err
	}
	seedCount := 0
	switch report.Panel {
	case "development":
		seedCount = report.Manifest.DevelopmentSeeds.Count
	case "validation":
		seedCount = report.Manifest.ValidationSeeds.Count
	case "locked":
		seedCount = report.Manifest.LockedSeeds.Count
	default:
		return errors.New("invalid evaluation panel")
	}
	if len(report.Policies) != len(evaluationPolicies) {
		return errors.New("evaluation policy cardinality mismatch")
	}
	if len(report.TaskMeterItems) != 960 {
		return fmt.Errorf("evaluation task meter items=%d, want exactly 960 replay items", len(report.TaskMeterItems))
	}
	for index, item := range report.TaskMeterItems {
		wantName := "certificate-replay"
		if index >= 480 {
			wantName = "post-selection-replay"
		}
		if item.Name != wantName {
			return fmt.Errorf("evaluation task meter item %d is %q, want %q", index, item.Name, wantName)
		}
	}
	var episodeItems [][]MeterItem
	for policyIndex, policy := range report.Policies {
		if policy.Name != evaluationPolicies[policyIndex] || len(policy.Fixtures) != seedCount {
			return fmt.Errorf("evaluation policy %d is %q with %d fixtures, want %q with %d", policyIndex, policy.Name, len(policy.Fixtures), evaluationPolicies[policyIndex], seedCount)
		}
		seedRange := report.Manifest.DevelopmentSeeds
		if report.Panel == "validation" {
			seedRange = report.Manifest.ValidationSeeds
		} else if report.Panel == "locked" {
			seedRange = report.Manifest.LockedSeeds
		}
		for fixtureIndex, fixture := range policy.Fixtures {
			wantSeed := seedRange.Start + int64(fixtureIndex)*seedRange.Step
			if fixture.Seed != wantSeed || fixture.Cohort != cohortFor(fixtureIndex) {
				return fmt.Errorf("policy %q fixture %d has non-preregistered seed/cohort", policy.Name, fixtureIndex)
			}
			if err := VerifyEpisodeMeterItems(fixture.MeterItems); err != nil {
				return err
			}
			if fixture.CacheHits < 0 || fixture.CacheMisses < 0 || fixture.CacheHits+fixture.CacheMisses != 6*fixture.ActionCount {
				return errors.New("evaluation cache counts do not cover all six action lookups per step")
			}
			if !fixture.MeterItems[0].Active || !fixture.MeterItems[1].Active || !fixture.MeterItems[4].Active || fixture.MeterItems[5].Active != (policy.Name == "dynamic-optimal") {
				return errors.New("evaluation meter activity ownership mismatch")
			}
			valid, err := episodeCapsValid(MeterEvaluation, fixture.MeterItems)
			if err != nil || fixture.AllCapsValid != valid {
				return errors.New("evaluation fixture all_caps_valid is not reconstructed")
			}
			base := fixture
			base.MeterItems = []MeterItem{}
			baseBytes, _ := causalv2.CanonicalJSON(base)
			meterBytes, _ := causalv2.CanonicalJSON(fixture.MeterItems)
			if len(baseBytes) > report.Manifest.EvaluationFixtureBaseByteCap || len(meterBytes)-2 > report.Manifest.EvaluationFixtureMeterItemsByteCap || len(mustCanonical(fixture)) > report.Manifest.FixtureRecordByteCap {
				return errors.New("evaluation fixture encoding subcap exceeded")
			}
			episodeItems = append(episodeItems, fixture.MeterItems)
		}
	}
	controlItems := make([][15]int64, len(report.ControlBundle.Certificates))
	for i, certificate := range report.ControlBundle.Certificates {
		controlItems[i] = certificate.MeterCounts
	}
	if err := VerifyAggregateMeters(report.Mechanical.Meters, MeterEvaluation, episodeItems, report.TaskMeterItems, controlItems); err != nil {
		return err
	}
	if err := verifyMeterCardinality(MeterEvaluation, report.Mechanical.Meters, seedCount); err != nil {
		return err
	}
	reconstructed := report
	reconstructEvaluationDerivations(&reconstructed)
	type derivedEvaluation struct {
		Policies         []PolicyReport       `json:"policies"`
		Contrasts        []Contrast           `json:"contrasts"`
		Gates            Gates                `json:"gates"`
		Controls         Controls             `json:"controls"`
		DynamicBenchmark DynamicBenchmark     `json:"dynamic_benchmark"`
		Mechanical       EvaluationMechanical `json:"mechanical"`
		Status           string               `json:"status"`
	}
	got := derivedEvaluation{report.Policies, report.Contrasts, report.Gates, report.Controls, report.DynamicBenchmark, report.Mechanical, report.Status}
	want := derivedEvaluation{reconstructed.Policies, reconstructed.Contrasts, reconstructed.Gates, reconstructed.Controls, reconstructed.DynamicBenchmark, reconstructed.Mechanical, reconstructed.Status}
	// Byte-count fields are reconstructed by the report codec rather than the
	// semantic derivation pass.
	want.Mechanical.NonrecordBytes = got.Mechanical.NonrecordBytes
	want.Mechanical.ReportBytes = got.Mechanical.ReportBytes
	gotBytes, _ := causalv2.CanonicalJSON(got)
	wantBytes, _ := causalv2.CanonicalJSON(want)
	if string(gotBytes) != string(wantBytes) {
		return errors.New("evaluation aggregates, statistics, gates, controls, or mechanics do not reconstruct")
	}
	if !bytes.Equal(mustCanonical(report.Limitations), mustCanonical([]string{"Synthetic deterministic three-variable SCMs do not establish production causal diagnosis."})) {
		return errors.New("evaluation limitations differ from the preregistered limitation")
	}
	return nil
}

func SignCentralTranscript(events []CentralTranscriptEvent) ([]CentralTranscriptEvent, error) {
	if len(events) != 521 {
		return nil, errors.New("central transcript must contain 521 events")
	}
	previous := causalv2.ZeroDigest
	for i := range events {
		wantKind := "admission"
		if i >= 480 && i < 520 {
			wantKind = "aggregate"
		} else if i == 520 {
			wantKind = "selection"
		}
		if events[i].EventVersion != "causal-central-transcript/v2" || events[i].Index != i || events[i].Kind != wantKind || events[i].PreviousDigest != previous || events[i].WorkAfter < events[i].WorkBefore {
			return nil, fmt.Errorf("central transcript event %d is invalid", i)
		}
		events[i].EventDigest = ""
		digest, err := causalv2.Digest("causal-central-transcript-event/v2", events[i])
		if err != nil {
			return nil, err
		}
		events[i].EventDigest, previous = digest, digest
	}
	return events, nil
}

func TrainingDigest(input TrainingDigestInput) (string, error) {
	if input.DigestInputVersion != "causal-training-digest-input/v2" || input.PlanCommit != PlanCommit {
		return "", errors.New("invalid training digest input identity")
	}
	if err := causalv2.ValidateManifest(input.Manifest); err != nil {
		return "", err
	}
	return causalv2.Digest("causal-training-digest-input/v2", input)
}
