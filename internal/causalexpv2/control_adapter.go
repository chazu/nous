package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/causalcurriculum"
	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

func init() {
	controlSuiteAdapter = executeControlSuite
}

func executeControlSuite(ctx context.Context, repoRoot string, reference executedEpisode, curriculum curriculumOutcome, centralProfileBytes []byte, episodeBytes, certificateBytes [][]byte, trainingFixtures []PrivateFixture, corruptionFixture PrivateFixture, retained *ControlEvidence) (ControlBundle, error) {
	if retained == nil {
		return ControlBundle{}, errors.New("nil control evidence destination")
	}
	if curriculum.SelectedRule == "" {
		return ControlBundle{}, errors.New("control suite requires the selected rule")
	}
	if reference.profile.AcquisitionCode != curriculum.SelectedRule {
		selectedReference, selectedErr := runEpisode(ctx, Panel(reference.profile.Panel), reference.fixture, curriculum.SelectedRule, "", false)
		if selectedErr != nil {
			return ControlBundle{}, fmt.Errorf("construct selected-rule control reference: %w", selectedErr)
		}
		reference = selectedReference
	}
	selectedProfileBytes, err := signedAcquisitionProfile(reference, curriculum.SelectedRule)
	if err != nil {
		return ControlBundle{}, err
	}
	semanticRules := causal.Rules()
	if len(semanticRules) == 0 {
		return ControlBundle{}, errors.New("causal grammar has no semantic-first rule")
	}
	staticRule := semanticRules[0].Code()
	staticProfileBytes, err := signedAcquisitionProfile(reference, staticRule)
	if err != nil {
		return ControlBundle{}, err
	}
	aliasFixture, aliasFixtureBytes, aliasProfileBytes, err := transformedContext(reference, func(fixture *PublicFixture) {
		fixture.Aliases = []string{"opaque-alpha", "opaque-beta", "opaque-gamma"}
	})
	if err != nil {
		return ControlBundle{}, err
	}
	presentationFixture, presentationFixtureBytes, presentationProfileBytes, err := transformedContext(reference, func(fixture *PublicFixture) {
		slices.Reverse(fixture.Presentation)
	})
	if err != nil {
		return ControlBundle{}, err
	}
	costFixture, costFixtureBytes, costProfileBytes, err := transformedContext(reference, perturbCosts)
	if err != nil {
		return ControlBundle{}, err
	}

	primaryTeacher := func() (causalrun.Teacher, error) {
		return causaloracle.NewTeacher(reference.fixture.PublicFixture.OpaqueToken, reference.fixture.HiddenHypothesis)
	}
	pairedTeacher := func(fixture PrivateFixture) (causalrun.Teacher, error) {
		return causaloracle.NewTeacher(fixture.PublicFixture.OpaqueToken, fixture.HiddenHypothesis)
	}
	differentHidden := ""
	for _, hypothesis := range reference.fixture.PublicFixture.InitialPosterior {
		if hypothesis != reference.fixture.HiddenHypothesis {
			differentHidden = hypothesis
			break
		}
	}
	if differentHidden == "" {
		return ControlBundle{}, errors.New("hidden-twin control has no alternate hidden hypothesis")
	}

	onlineNames := []causalrun.ControlName{
		causalrun.ControlHiddenTwin,
		causalrun.ControlWrongContext,
		causalrun.ControlStaticRule,
		causalrun.ControlRecomputedRule,
		causalrun.ControlOpaqueAlias,
		causalrun.ControlPresentationOrder,
		causalrun.ControlProposalOrder,
		causalrun.ControlCostPerturbation,
		causalrun.ControlOccupiedName,
		causalrun.ControlAlternateDescriptor,
		causalrun.ControlMutationInert,
		causalrun.ControlChildVM,
		causalrun.ControlStaleResponse,
		causalrun.ControlDuplicateResponse,
		causalrun.ControlCorruptionSuite,
		causalrun.ControlDeterministicJSON,
	}
	bundle := ControlBundle{ControlBundleVersion: causalv2.ControlBundleDomain, Certificates: make([]ControlCertificate, 0, len(causalv2.ControlNames))}
	evidence := ControlEvidence{ControlEvidenceVersion: causalv2.ControlEvidenceDomain, SelectedRule: curriculum.SelectedRule, StaticRule: staticRule, StaticMatrix: []causalv2.MatrixControlRow{}, RecomputedMatrix: []causalv2.MatrixControlRow{}}
	corruptionReference, err := runEpisode(ctx, PanelDevelopment, corruptionFixture, "P=H;M=gain;S=C", "", false)
	if err != nil {
		return ControlBundle{}, fmt.Errorf("development corruption witness: %w", err)
	}
	for _, name := range onlineNames {
		if name == causalrun.ControlStaticRule || name == causalrun.ControlRecomputedRule {
			certificate, rows, matrixErr := executeTrainingMatrixControl(ctx, name, curriculum.SelectedRule, staticRule, trainingFixtures, decodeEpisodes(episodeBytes), decodeCertificates(certificateBytes))
			if matrixErr != nil {
				return ControlBundle{}, matrixErr
			}
			if name == causalrun.ControlStaticRule {
				evidence.StaticMatrix = rows
			} else {
				evidence.RecomputedMatrix = rows
			}
			bundle.Certificates = append(bundle.Certificates, certificate)
			continue
		}
		teacher, teacherErr := primaryTeacher()
		if teacherErr != nil {
			return ControlBundle{}, teacherErr
		}
		controlReference := reference
		controlProfileBytes := selectedProfileBytes
		if name == causalrun.ControlCorruptionSuite {
			controlReference, controlProfileBytes = corruptionReference, corruptionReference.profileBytes
			teacher, teacherErr = causaloracle.NewTeacher(corruptionFixture.PublicFixture.OpaqueToken, corruptionFixture.HiddenHypothesis)
			if teacherErr != nil {
				return ControlBundle{}, teacherErr
			}
		}
		input := causalrun.ControlInput{
			FixtureBytes:      controlReference.publicBytes,
			ProfileBytes:      controlProfileBytes,
			Teacher:           teacher,
			BaselineArtifacts: controlReference.result.Artifacts,
			StaticRuleCode:    staticRule,
			SelectedRuleCode:  curriculum.SelectedRule,
		}
		switch name {
		case causalrun.ControlHiddenTwin:
			input.PairedTeacher, teacherErr = causaloracle.NewTeacher(reference.fixture.PublicFixture.OpaqueToken, differentHidden)
		case causalrun.ControlWrongContext:
			input.PairedProfileBytes = aliasProfileBytes
		case causalrun.ControlStaticRule:
			input.PairedProfileBytes = staticProfileBytes
			input.PairedTeacher, teacherErr = primaryTeacher()
		case causalrun.ControlRecomputedRule:
			input.PairedProfileBytes = selectedProfileBytes
			input.PairedTeacher, teacherErr = primaryTeacher()
		case causalrun.ControlOpaqueAlias:
			input.PairedFixtureBytes, input.PairedProfileBytes = aliasFixtureBytes, aliasProfileBytes
			input.PairedTeacher, teacherErr = pairedTeacher(aliasFixture)
		case causalrun.ControlPresentationOrder:
			input.PairedFixtureBytes, input.PairedProfileBytes = presentationFixtureBytes, presentationProfileBytes
			input.PairedTeacher, teacherErr = pairedTeacher(presentationFixture)
		case causalrun.ControlCostPerturbation:
			input.PairedFixtureBytes, input.PairedProfileBytes = costFixtureBytes, costProfileBytes
			input.PairedTeacher, teacherErr = pairedTeacher(costFixture)
		case causalrun.ControlProposalOrder, causalrun.ControlOccupiedName, causalrun.ControlMutationInert,
			causalrun.ControlDeterministicJSON:
			input.PairedTeacher, teacherErr = primaryTeacher()
		}
		if teacherErr != nil {
			return ControlBundle{}, teacherErr
		}
		observation, executeErr := causalrun.ExecuteControl(ctx, name, input)
		if executeErr != nil {
			return ControlBundle{}, fmt.Errorf("control %s: %w", name, executeErr)
		}
		certificate, certificateErr := certificateFromObservation(observation)
		if certificateErr != nil {
			return ControlBundle{}, certificateErr
		}
		bundle.Certificates = append(bundle.Certificates, certificate)
		if name == causalrun.ControlMutationInert {
			evidence.Mutation = mutationProof(observation)
		}
		if name == causalrun.ControlChildVM {
			evidence.ChildVM = childVMProof(observation.ChildVM)
		}
		if name == causalrun.ControlCorruptionSuite {
			caseNames, namesErr := causalrun.CorruptionCaseNames(corruptionReference.result.Artifacts)
			if namesErr != nil {
				return ControlBundle{}, namesErr
			}
			evidence.Corruption, namesErr = corruptionProof(corruptionReference, caseNames)
			if namesErr != nil {
				return ControlBundle{}, namesErr
			}
		}
	}

	noCreditProfile, err := causalv2.VerifyCentralProfile(centralProfileBytes)
	if err != nil {
		return ControlBundle{}, err
	}
	treatmentProfileDigest := noCreditProfile.ProfileDigest
	noCreditProfile.CreditEnabled = false
	noCreditProfile.TrainingKey, noCreditProfile.ProfileDigest = "", ""
	if err := causalv2.SignCentralProfile(&noCreditProfile); err != nil {
		return ControlBundle{}, err
	}
	noCreditProfileBytes := mustCanonical(noCreditProfile)
	noCredit, err := causalcurriculum.Run(ctx, noCreditProfileBytes, episodeBytes, certificateBytes)
	if err != nil {
		return ControlBundle{}, fmt.Errorf("no-credit control: %w", err)
	}
	if err := causalcurriculum.Verify(noCreditProfileBytes, episodeBytes, certificateBytes, noCredit); err != nil {
		return ControlBundle{}, fmt.Errorf("verify no-credit control: %w", err)
	}
	noCreditPassed := noCredit.Unresolved && noCredit.SelectedRule == "" && len(noCredit.Applications) == len(certificateBytes) && len(noCredit.Aggregates) == len(orderedRules())
	noCreditCertificate := ControlCertificate{
		ControlVersion:    causalv2.ControlCertificateDomain,
		Name:              "no-credit",
		TreatmentEvidence: emptyExperimentControlResult(treatmentProfileDigest),
		ControlEvidence:   emptyExperimentControlResult(noCredit.ProfileDigest),
		Observed:          "credit-disabled-unresolved",
		Passed:            noCreditPassed,
		MeterCounts:       noCredit.Counts.Counts(),
		Work:              noCredit.Counts.TotalWork,
	}
	noCreditCertificate.ControlEvidence.FailureCode = "unresolved-selection"
	if err := causalv2.SignControlCertificate(&noCreditCertificate); err != nil {
		return ControlBundle{}, err
	}
	bundle.Certificates = append(bundle.Certificates, noCreditCertificate)
	evidence.NoCredit = noCreditProof(noCredit, noCreditProfileBytes, certificateBytes)

	dependencyEvidence, err := causalrun.AuditDependencyBoundary(repoRoot)
	if err != nil {
		return ControlBundle{}, err
	}
	dependencyDigest, err := causalv2.Digest("causal-dependency-evidence/v2", dependencyEvidence)
	if err != nil {
		return ControlBundle{}, err
	}
	dependencyCounts := Counter{ArtifactMaterializations: 1, TableLookups: dependencyEvidence.Lookups}
	dependencyCounts.TotalWork = dependencyCounts.ComputedTotalWork()
	dependencyCertificate := ControlCertificate{
		ControlVersion:    causalv2.ControlCertificateDomain,
		Name:              "dependency",
		FixtureDigest:     dependencyDigest,
		TreatmentEvidence: emptyExperimentControlResult(""),
		ControlEvidence:   emptyExperimentControlResult(""),
		Observed:          fmt.Sprintf("forbidden-dependencies=%d;files=%d;imports=%d;methods=%d;lookups=%d", len(dependencyEvidence.Forbidden), dependencyEvidence.Files, dependencyEvidence.ImportEdges, dependencyEvidence.MethodEdges, dependencyEvidence.Lookups),
		Passed:            len(dependencyEvidence.Forbidden) == 0,
		MeterCounts:       dependencyCounts.Counts(),
		Work:              dependencyCounts.TotalWork,
	}
	if err := causalv2.SignControlCertificate(&dependencyCertificate); err != nil {
		return ControlBundle{}, err
	}
	bundle.Certificates = append(bundle.Certificates, dependencyCertificate)
	evidence.Dependency, err = rootedDependencyProof(repoRoot, dependencyEvidence)
	if err != nil {
		return ControlBundle{}, err
	}
	if err := causalv2.SignControlBundle(&bundle); err != nil {
		return ControlBundle{}, err
	}
	if err := verifyExecutedControlBundle(bundle); err != nil {
		return ControlBundle{}, err
	}
	if err := causalv2.SignControlEvidence(&evidence); err != nil {
		return ControlBundle{}, err
	}
	*retained = evidence
	return bundle, nil
}

func executeTrainingMatrixControl(ctx context.Context, name causalrun.ControlName, selectedRule, staticRule string, fixtures []PrivateFixture, episodes []EpisodeEvidence, certificates []ApplicationCertificate) (ControlCertificate, []causalv2.MatrixControlRow, error) {
	manifest := causalv2.PreregisteredManifest()
	rules := orderedRules()
	if len(fixtures) != manifest.TrainingSeeds.Count || len(episodes) != manifest.TrainingSeeds.Count*len(rules) || len(certificates) != len(episodes) {
		return ControlCertificate{}, nil, errors.New("matrix control requires the already-opened complete training matrix")
	}
	selectedIndex, staticIndex := -1, -1
	for index, rule := range rules {
		if rule.Code() == selectedRule {
			selectedIndex = index
		}
		if rule.Code() == staticRule {
			staticIndex = index
		}
	}
	if selectedIndex < 0 || staticIndex < 0 {
		return ControlCertificate{}, nil, errors.New("matrix control rule absent from training matrix")
	}
	observations := make([]causalrun.ControlObservation, 0, manifest.TrainingSeeds.Count)
	rows := make([]causalv2.MatrixControlRow, 0, manifest.TrainingSeeds.Count)
	for seedIndex := 0; seedIndex < manifest.TrainingSeeds.Count; seedIndex++ {
		seed := manifest.TrainingSeeds.Start + int64(seedIndex)*manifest.TrainingSeeds.Step
		fixture := fixtures[seedIndex]
		if fixture.PublicFixture.Seed != seed {
			return ControlCertificate{}, nil, errors.New("training fixtures are not in seed order")
		}
		selected, err := runEpisode(ctx, PanelTraining, fixture, selectedRule, selectedRule, selectedIndex == 0)
		if err != nil {
			return ControlCertificate{}, nil, err
		}
		pairedAcquisition := selectedRule
		if name == causalrun.ControlStaticRule {
			pairedAcquisition = staticRule
		}
		pairedProfile, err := signedAcquisitionProfile(selected, pairedAcquisition)
		if err != nil {
			return ControlCertificate{}, nil, err
		}
		primaryTeacher, _ := causaloracle.NewTeacher(fixture.PublicFixture.OpaqueToken, fixture.HiddenHypothesis)
		pairedTeacher, _ := causaloracle.NewTeacher(fixture.PublicFixture.OpaqueToken, fixture.HiddenHypothesis)
		observation, err := causalrun.ExecuteControl(ctx, name, causalrun.ControlInput{FixtureBytes: selected.publicBytes, ProfileBytes: selected.profileBytes, Teacher: primaryTeacher, PairedProfileBytes: pairedProfile, PairedTeacher: pairedTeacher, StaticRuleCode: staticRule, SelectedRuleCode: selectedRule})
		if err != nil {
			return ControlCertificate{}, nil, fmt.Errorf("%s matrix seed %d: %w", name, seed, err)
		}
		observations = append(observations, observation)
		selectedRecord := seedIndex*len(rules) + selectedIndex
		controlRecord := seedIndex*len(rules) + staticIndex
		if !bytes.Equal(mustCanonical(selected.evidence), mustCanonical(episodes[selectedRecord])) || !bytes.Equal(mustCanonical(selected.certificate), mustCanonical(certificates[selectedRecord])) {
			return ControlCertificate{}, nil, fmt.Errorf("%s matrix seed %d treatment differs from committed training record", name, seed)
		}
		selectedProjection, err := episodeControlResult(selected)
		if err != nil || !bytes.Equal(mustCanonical(selectedProjection), mustCanonical(retainedControlResult(observation.Treatment))) {
			return ControlCertificate{}, nil, fmt.Errorf("%s matrix seed %d treatment projection differs from committed episode", name, seed)
		}
		row := matrixRow(seed, observation)
		row.TreatmentEpisodeDigest = episodes[selectedRecord].EpisodeReportDigest
		row.TreatmentCertificateDigest = certificates[selectedRecord].CertificateDigest
		if name == causalrun.ControlStaticRule {
			staticEpisode, runErr := runEpisode(ctx, PanelTraining, fixture, staticRule, staticRule, staticIndex == 0)
			if runErr != nil {
				return ControlCertificate{}, nil, runErr
			}
			if !bytes.Equal(mustCanonical(staticEpisode.evidence), mustCanonical(episodes[controlRecord])) || !bytes.Equal(mustCanonical(staticEpisode.certificate), mustCanonical(certificates[controlRecord])) {
				return ControlCertificate{}, nil, fmt.Errorf("static matrix seed %d control differs from committed training record", seed)
			}
			staticProjection, projectionErr := episodeControlResult(staticEpisode)
			if projectionErr != nil || !bytes.Equal(mustCanonical(staticProjection), mustCanonical(retainedControlResult(observation.Control))) {
				return ControlCertificate{}, nil, fmt.Errorf("static matrix seed %d control projection differs from committed episode", seed)
			}
			row.ControlEpisodeDigest = episodes[controlRecord].EpisodeReportDigest
			row.ControlCertificateDigest = certificates[controlRecord].CertificateDigest
		}
		rows = append(rows, row)
	}
	certificate, err := aggregateMatrixControl(name, observations)
	return certificate, rows, err
}

func episodeControlResult(episode executedEpisode) (causalv2.ControlResult, error) {
	posteriorDigests := make([]string, 0, len(episode.result.Actions)+1)
	for _, encoded := range episode.result.Artifacts {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		if artifact.Kind != "posterior" {
			continue
		}
		payload, err := causalv2.StrictDecode[causalv2.PosteriorPayload](artifact.Payload)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		posteriorDigests = append(posteriorDigests, payload.SemanticSetDigest)
	}
	costs := make([]int, len(episode.result.Actions))
	for index, code := range episode.result.Actions {
		action, err := causal.ParseAction(code)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		costs[index] = episode.fixture.PublicFixture.Costs[action.Variable]
	}
	return retainedControlResult(causalv2.ControlResult{
		ProfileDigest: episode.result.ProfileDigest, Actions: append([]string(nil), episode.result.Actions...),
		Outcomes: append([]string(nil), episode.result.TeacherOutcomes...), PosteriorDigests: posteriorDigests,
		Costs: costs, Terminal: episode.result.Terminal, Score: episode.result.Score,
		TranscriptDigest: episode.result.TranscriptDigest,
	}), nil
}

func aggregateMatrixControl(name causalrun.ControlName, observations []causalrun.ControlObservation) (ControlCertificate, error) {
	left, right := emptyExperimentControlResult(""), emptyExperimentControlResult("")
	var counts [15]int64
	passed := len(observations) != 0
	for index, observation := range observations {
		passed = passed && observation.Passed
		if index == 0 {
			left = retainedControlResult(observation.Treatment)
			right = retainedControlResult(observation.Control)
		}
		for index, value := range observation.Counts.Array() {
			counts[index] += int64(value)
		}
	}
	observed := "semantic-projection-equal"
	if name == causalrun.ControlStaticRule {
		observed = "static-baseline-executed"
	}
	certificate := ControlCertificate{ControlVersion: causalv2.ControlCertificateDomain, Name: string(name), FixtureDigest: observations[0].FixtureDigest, TreatmentEvidence: left, ControlEvidence: right, Observed: observed, Passed: passed, MeterCounts: counts, Work: counts[14]}
	if err := causalv2.SignControlCertificate(&certificate); err != nil {
		return ControlCertificate{}, err
	}
	return certificate, nil
}

func signedAcquisitionProfile(reference executedEpisode, acquisition string) ([]byte, error) {
	profile := reference.profile
	profile.AcquisitionCode = acquisition
	profile.FixtureDigest = reference.fixture.PublicFixture.FixtureDigest
	profile.ProfileDigest = ""
	if err := causalv2.SignProfile(&profile); err != nil {
		return nil, err
	}
	return causalv2.CanonicalJSON(profile)
}

func certificateFromObservation(observation causalrun.ControlObservation) (ControlCertificate, error) {
	// The retained schema permits an empty transition projection. Controls also
	// retain the transcript digest; omit a runner-internal superset containing
	// candidate-cell posterior artifacts rather than mislabeling it as the
	// initial-plus-realized transition sequence.
	if len(observation.Treatment.PosteriorDigests) != 0 && len(observation.Treatment.PosteriorDigests) != len(observation.Treatment.Actions)+1 {
		observation.Treatment.PosteriorDigests = []string{}
	}
	if len(observation.Control.PosteriorDigests) != 0 && len(observation.Control.PosteriorDigests) != len(observation.Control.Actions)+1 {
		observation.Control.PosteriorDigests = []string{}
	}
	certificate := ControlCertificate{
		ControlVersion:    causalv2.ControlCertificateDomain,
		Name:              string(observation.Name),
		FixtureDigest:     observation.FixtureDigest,
		TreatmentEvidence: observation.Treatment,
		ControlEvidence:   observation.Control,
		Observed:          observation.Observed,
		Passed:            observation.Passed,
		MeterCounts:       counts64(observation.Counts),
		Work:              int64(observation.Counts.TotalWork),
	}
	if err := causalv2.SignControlCertificate(&certificate); err != nil {
		return ControlCertificate{}, fmt.Errorf("sign control %s: %w", observation.Name, err)
	}
	return certificate, nil
}

func matrixRow(seed int64, observation causalrun.ControlObservation) causalv2.MatrixControlRow {
	treatment, control := retainedControlResult(observation.Treatment), retainedControlResult(observation.Control)
	return causalv2.MatrixControlRow{Seed: seed, FixtureDigest: observation.FixtureDigest, Treatment: treatment, Control: control, TreatmentMeterCounts: counts64(observation.TreatmentCounts), ControlMeterCounts: counts64(observation.ControlCounts), TreatmentCache: cacheTrace(observation.TreatmentCache), ControlCache: cacheTrace(observation.ControlCache)}
}

func cacheTrace(trace causalrun.CacheTrace) causalv2.CacheTrace {
	return causalv2.CacheTrace{Statuses: append([]string(nil), trace.Statuses...), Hits: trace.Hits, Misses: trace.Misses}
}

func retainedControlResult(result ControlResult) ControlResult {
	if len(result.PosteriorDigests) != 0 && len(result.PosteriorDigests) != len(result.Actions)+1 {
		result.PosteriorDigests = []string{}
	}
	return result
}

func noCreditProof(result causalcurriculum.Result, profileBytes []byte, certificateBytes [][]byte) causalv2.NoCreditProof {
	digests := make([]string, len(certificateBytes))
	for index, encoded := range certificateBytes {
		certificate, err := causalv2.StrictDecode[ApplicationCertificate](encoded)
		if err == nil {
			digests[index] = certificate.CertificateDigest
		}
	}
	artifacts := make([]causalv2.Base64URL, len(result.ArtifactBytes))
	for index := range result.ArtifactBytes {
		artifacts[index] = causalv2.EncodeBase64URL(result.ArtifactBytes[index])
	}
	return causalv2.NoCreditProof{CentralProfileBytes: causalv2.EncodeBase64URL(profileBytes), CertificateDigests: digests, ArtifactBytes: artifacts, Aggregates: result.Aggregates, CentralTranscript: []causalv2.CentralTranscriptEvent{}, TaskMeterItems: result.TaskMeterItems, Counts: result.Counts.Counts(), Resolution: "unresolved", WinnerTies: result.WinnerTies, SelectedRule: result.SelectedRule, TerminalTranscriptDigest: result.TerminalTranscriptDigest}
}

func mutationProof(observation causalrun.ControlObservation) causalv2.MutationProof {
	return causalv2.MutationProof{FixtureDigest: observation.FixtureDigest, OffConfig: mutationConfig(observation.TreatmentMutation.Config), OnConfig: mutationConfig(observation.ControlMutation.Config), OffResult: retainedControlResult(observation.Treatment), OnResult: retainedControlResult(observation.Control), OffMutants: mutantRecords(observation.TreatmentMutation.Mutants), OnMutants: mutantRecords(observation.ControlMutation.Mutants), OffMeterCounts: counts64(observation.TreatmentMutation.MeterCounts), OnMeterCounts: counts64(observation.ControlMutation.MeterCounts)}
}

func mutationConfig(value causalrun.MutationConfigEvidence) causalv2.MutationConfig {
	return causalv2.MutationConfig{Enabled: value.Enabled, Interval: value.Interval, MaxMutants: value.MaxMutants, MutantWorth: value.MutantWorth, ValidateOnly: value.ValidateOnly, MinApplics: value.MinApplics, MutationThreshold: value.MutationThreshold}
}

func mutantRecords(values []causalrun.MutantRecord) []causalv2.MutantRecord {
	result := make([]causalv2.MutantRecord, len(values))
	for i, v := range values {
		result[i] = causalv2.MutantRecord{Name: v.Name, MutantOf: v.MutantOf, SourceSlot: v.SourceSlot, Operation: v.Operation, ProgramDigest: v.ProgramDigest, Worth: v.Worth}
	}
	return result
}

func childVMProof(value causalrun.ChildVMEvidence) causalv2.ChildVMProof {
	return causalv2.ChildVMProof{FixtureDigest: value.FixtureDigest, ProfileDigest: value.ProfileDigest, Operation: value.Operation, ArtifactsBefore: value.ArtifactsBefore, ArtifactsAfter: value.ArtifactsAfter, MeterCountsBefore: counts64(value.MeterCountsBefore), MeterCountsAfter: counts64(value.MeterCountsAfter), TeacherCallsBefore: value.TeacherCallsBefore, TeacherCallsAfter: value.TeacherCallsAfter, FailureCode: value.FailureCode}
}

func corruptionProof(reference executedEpisode, names []string) (causalv2.CorruptionProof, error) {
	artifacts := make([]causalv2.Base64URL, len(reference.result.Artifacts))
	for i := range reference.result.Artifacts {
		artifacts[i] = causalv2.EncodeBase64URL(reference.result.Artifacts[i])
	}
	observed, err := causalrun.CorruptionCases(reference.publicBytes, reference.profileBytes, reference.result.Artifacts)
	if err != nil {
		return causalv2.CorruptionProof{}, err
	}
	if len(observed) != len(names) {
		return causalv2.CorruptionProof{}, errors.New("corruption case cardinality differs from closed names")
	}
	cases := make([]causalv2.CorruptionCase, len(observed))
	for i, item := range observed {
		if item.Name != names[i] {
			return causalv2.CorruptionProof{}, errors.New("corruption cases differ from closed order")
		}
		cases[i] = causalv2.CorruptionCase{Name: item.Name, MutationDescriptor: item.MutationDescriptor, MutatedBytesDigest: item.MutatedBytesDigest, RejectionCode: item.RejectionCode, MeterCounts: counts64(item.MeterCounts)}
	}
	caseDigest, _ := causalv2.Digest(causalv2.CorruptionEnumeratorDomain, names)
	return causalv2.CorruptionProof{EnumeratorVersion: causalv2.CorruptionEnumeratorDomain, FixtureBytes: causalv2.EncodeBase64URL(mustCanonical(reference.fixture)), ProfileBytes: causalv2.EncodeBase64URL(reference.profileBytes), BaselineArtifacts: artifacts, CaseCount: len(cases), CaseSetDigest: caseDigest, Cases: cases}, nil
}

func decodeEpisodes(values [][]byte) []EpisodeEvidence {
	result := make([]EpisodeEvidence, len(values))
	for i, v := range values {
		decoded, err := causalv2.StrictDecode[EpisodeEvidence](v)
		if err != nil {
			return nil
		}
		result[i] = decoded
	}
	return result
}
func decodeCertificates(values [][]byte) []ApplicationCertificate {
	result := make([]ApplicationCertificate, len(values))
	for i, v := range values {
		decoded, err := causalv2.StrictDecode[ApplicationCertificate](v)
		if err != nil {
			return nil
		}
		result[i] = decoded
	}
	return result
}

func cloneBytes(values [][]byte) [][]byte {
	result := make([][]byte, len(values))
	for index := range values {
		result[index] = append([]byte(nil), values[index]...)
	}
	return result
}

func transformedContext(reference executedEpisode, transform func(*PublicFixture)) (PrivateFixture, []byte, []byte, error) {
	fixture := reference.fixture
	fixture.PublicFixture.Aliases = append([]string(nil), fixture.PublicFixture.Aliases...)
	fixture.PublicFixture.Costs = append([]int(nil), fixture.PublicFixture.Costs...)
	fixture.PublicFixture.Pool = append([]string(nil), fixture.PublicFixture.Pool...)
	fixture.PublicFixture.Presentation = append([]int(nil), fixture.PublicFixture.Presentation...)
	fixture.PublicFixture.InitialPosterior = append([]string(nil), fixture.PublicFixture.InitialPosterior...)
	fixture.PublicFixture.UniformRandomActions = append([]string(nil), fixture.PublicFixture.UniformRandomActions...)
	fixture.PublicFixture.FixtureDigest = ""
	fixture.PrivateFixtureDigest = ""
	transform(&fixture.PublicFixture)
	if err := causalv2.SignPublicFixture(&fixture.PublicFixture); err != nil {
		return PrivateFixture{}, nil, nil, err
	}
	if err := causalv2.SignPrivateFixture(&fixture); err != nil {
		return PrivateFixture{}, nil, nil, err
	}
	fixtureBytes, err := causalv2.CanonicalJSON(fixture.PublicFixture)
	if err != nil {
		return PrivateFixture{}, nil, nil, err
	}
	profile := reference.profile
	profile.FixtureDigest = fixture.PublicFixture.FixtureDigest
	profile.ProfileDigest = ""
	if err := causalv2.SignProfile(&profile); err != nil {
		return PrivateFixture{}, nil, nil, err
	}
	profileBytes, err := causalv2.CanonicalJSON(profile)
	return fixture, fixtureBytes, profileBytes, err
}

func perturbCosts(fixture *PublicFixture) {
	for index, cost := range fixture.Costs {
		minimum, maximum := 20, 40
		if fixture.Cohort == string(CohortCostSkewed) {
			switch {
			case cost <= 10:
				minimum, maximum = 1, 10
			case cost <= 50:
				minimum, maximum = 30, 50
			default:
				minimum, maximum = 80, 100
			}
		}
		if cost < maximum {
			fixture.Costs[index] = cost + 1
		} else {
			fixture.Costs[index] = maxInt(minimum, cost-1)
		}
	}
}

func emptyExperimentControlResult(profileDigest string) ControlResult {
	return ControlResult{ProfileDigest: profileDigest, Actions: []string{}, Outcomes: []string{}, PosteriorDigests: []string{}, Costs: []int{}}
}
