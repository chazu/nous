package causalexpv2

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
)

func TestEvaluationPolicyNamesMapToCanonicalAcquisitionCodes(t *testing.T) {
	cases := map[Policy]string{
		"information-gain-per-cost": "P=H;M=gain;S=C",
		"worst-split-per-cost":      "P=W;M=gain;S=C",
		"lexical-fixed":             "lexical-fixed",
		"uniform-random":            "uniform-random",
		"passive-only":              "passive-only",
		"dynamic-optimal":           "dynamic-optimal",
	}
	for policy, want := range cases {
		if got := developmentAcquisition(policy); got != want {
			t.Fatalf("policy %q mapped to %q, want %q", policy, got, want)
		}
	}
	if got := developmentAcquisition("learned"); got == "" || got == "learned" {
		t.Fatalf("learned policy did not map to a grammar rule: %q", got)
	}
}

func TestRunEpisodeCarriesCompleteSameSourceDPAndCacheEvidence(t *testing.T) {
	fixture, err := generate(PanelDevelopment, causalv2.PreregisteredManifest().DevelopmentSeeds.Start, 0)
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := runEpisode(context.Background(), PanelDevelopment, fixture, string(causalrun.PolicyDynamicOptimal), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if dynamic.dynamic.MemberSimulations != len(fixture.PublicFixture.InitialPosterior) || dynamic.dynamic.Counts != dynamic.result.DynamicCounts || dynamic.evidence.MeterItems[5].Counts != counts64(dynamic.result.DynamicCounts) {
		t.Fatalf("dynamic benchmark was not retained from the production runner: benchmark=%+v result=%+v item=%+v", dynamic.dynamic, dynamic.result.DynamicCounts, dynamic.evidence.MeterItems[5])
	}
	if dynamic.result.CacheTrace.Hits+dynamic.result.CacheTrace.Misses != 6*len(dynamic.result.Actions) {
		t.Fatalf("cache trace=%+v actions=%d", dynamic.result.CacheTrace, len(dynamic.result.Actions))
	}
	lexical, err := runEpisode(context.Background(), PanelDevelopment, fixture, orderedRules()[0].Code(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if lexical.dynamic.MemberSimulations != len(fixture.PublicFixture.InitialPosterior) || lexical.evidence.MeterItems[5].Counts != counts64(lexical.dynamic.Counts) {
		t.Fatal("training DP owner did not retain one complete auxiliary production benchmark")
	}
}

func TestMaximumWidthMeterItemsGolden(t *testing.T) {
	counts := [15]int64{4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 531441, 4000000, 4000000, 4000000, 5000, 18000, 4000000}
	items := make([]MeterItem, len(causalv2.MeterNames))
	for index, name := range causalv2.MeterNames {
		items[index] = MeterItem{Name: name, Active: true, Counts: counts}
		encoded, err := causalv2.CanonicalJSON(items[index])
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > 1024 {
			t.Fatalf("maximum-width meter item %q bytes=%d", name, len(encoded))
		}
	}
	if got := items[0].Counter(); got.MemoStates != 531441 || got.MemoLookups != 4000000 || got.EngineCycles != 5000 || got.AttributedUnits != 18000 {
		t.Fatalf("counter field order changed: %+v", got)
	}
	encoded, err := causalv2.CanonicalJSON(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded)-2 > causalv2.PreregisteredManifest().EpisodeMeterItemsByteCap {
		t.Fatalf("maximum-width meter_items contents=%d cap=%d", len(encoded)-2, causalv2.PreregisteredManifest().EpisodeMeterItemsByteCap)
	}
}

func TestFixedCorruptionWitnessRegeneratesExactLedger(t *testing.T) {
	fixture, err := generate(PanelDevelopment, causalv2.PreregisteredManifest().DevelopmentSeeds.Start, 0)
	if err != nil {
		t.Fatal(err)
	}
	episode, err := runEpisode(context.Background(), PanelDevelopment, fixture, "P=H;M=gain;S=C", "", false)
	if err != nil {
		t.Fatal(err)
	}
	names, err := causalrun.CorruptionCaseNames(episode.result.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := corruptionProof(episode, names)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyFixedCorruptionWitness(context.Background(), proof); err != nil {
		t.Fatal(err)
	}
	other, err := generate(PanelDevelopment, causalv2.PreregisteredManifest().DevelopmentSeeds.Start+1, 1)
	if err != nil {
		t.Fatal(err)
	}
	proof.FixtureBytes = causalv2.EncodeBase64URL(mustCanonical(other))
	if _, err := verifyFixedCorruptionWitness(context.Background(), proof); err == nil {
		t.Fatal("self-consistent but non-fixed corruption fixture was accepted")
	}
}

func TestDevelopmentGeneratorIsDeterministicAndHiddenIndependent(t *testing.T) {
	capability := NewDiagnosticDevelopmentCapability()
	first, err := capability.GenerateDevelopment(112001, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := capability.GenerateDevelopment(112001, 0)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := causalv2.CanonicalJSON(first)
	secondBytes, _ := causalv2.CanonicalJSON(second)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("development generator is not deterministic")
	}
	if first.PublicFixture.Cohort != "cost-skewed" {
		t.Fatalf("cohort=%q", first.PublicFixture.Cohort)
	}
	costs := append([]int(nil), first.PublicFixture.Costs...)
	low, medium, high := false, false, false
	for _, cost := range costs {
		low = low || cost >= 1 && cost <= 10
		medium = medium || cost >= 30 && cost <= 50
		high = high || cost >= 80 && cost <= 100
	}
	if !low || !medium || !high {
		t.Fatalf("cost-skewed draw did not contain one value from each range: %v", costs)
	}

	publicBytes, _ := causalv2.CanonicalJSON(first.PublicFixture)
	var twinHidden, differingAction string
	for _, candidate := range first.PublicFixture.InitialPosterior {
		if candidate == first.HiddenHypothesis {
			continue
		}
		for _, action := range causaloracle.Actions() {
			left, _ := causaloracle.Predict(first.HiddenHypothesis, action.Code())
			right, _ := causaloracle.Predict(candidate, action.Code())
			if left != right {
				twinHidden, differingAction = candidate, action.Code()
				break
			}
		}
		if twinHidden != "" {
			break
		}
	}
	if twinHidden == "" {
		t.Fatal("development fixture had no distinguishable hidden twin")
	}
	twin := PrivateFixture{PublicFixture: first.PublicFixture, HiddenHypothesis: twinHidden}
	if err := causalv2.SignPrivateFixture(&twin); err != nil {
		t.Fatal(err)
	}
	twinPublicBytes, _ := causalv2.CanonicalJSON(twin.PublicFixture)
	if !bytes.Equal(publicBytes, twinPublicBytes) || twin.PublicFixture.OpaqueToken != first.PublicFixture.OpaqueToken {
		t.Fatal("counterfactual hidden twin changed public bytes or token")
	}
	leftRegistry, rightRegistry := NewTeacherRegistry(), NewTeacherRegistry()
	if err := leftRegistry.Bind(first); err != nil {
		t.Fatal(err)
	}
	if err := rightRegistry.Bind(twin); err != nil {
		t.Fatal(err)
	}
	left, _ := leftRegistry.Teacher(first.PublicFixture.OpaqueToken)
	right, _ := rightRegistry.Teacher(twin.PublicFixture.OpaqueToken)
	leftOutcome, _ := left.Respond(first.PublicFixture.OpaqueToken, differingAction)
	rightOutcome, _ := right.Respond(twin.PublicFixture.OpaqueToken, differingAction)
	if leftOutcome == rightOutcome {
		t.Fatal("private teacher bindings did not substitute hidden behavior")
	}
}

func TestDevelopmentEpisodeUsesProductionRunnerAndIndependentReplay(t *testing.T) {
	fixture, err := NewDiagnosticDevelopmentCapability().GenerateDevelopment(112001, 0)
	if err != nil {
		t.Fatal(err)
	}
	episode, err := runEpisode(context.Background(), PanelDevelopment, fixture, "lexical-fixed", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if episode.evidence.OracleDisagreements != 0 || episode.result.TranscriptDigest != episode.replay.TranscriptDigest || len(episode.result.Artifacts) == 0 {
		t.Fatalf("episode failed independent production/oracle checks: %+v", episode.evidence)
	}
}

func TestSignedControlContextTransformationsExecute(t *testing.T) {
	fixture, err := NewDiagnosticDevelopmentCapability().GenerateDevelopment(112001, 0)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := runEpisode(context.Background(), PanelDevelopment, fixture, "lexical-fixed", "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, trial := range []struct {
		name      causalrun.ControlName
		transform func(*PublicFixture)
	}{
		{causalrun.ControlOpaqueAlias, func(fixture *PublicFixture) {
			fixture.Aliases = []string{"opaque-alpha", "opaque-beta", "opaque-gamma"}
		}},
		{causalrun.ControlPresentationOrder, func(fixture *PublicFixture) { slices.Reverse(fixture.Presentation) }},
		{causalrun.ControlCostPerturbation, perturbCosts},
	} {
		paired, fixtureBytes, profileBytes, transformErr := transformedContext(reference, trial.transform)
		if transformErr != nil {
			t.Fatalf("%s transform: %v", trial.name, transformErr)
		}
		primary, _ := causaloracle.NewTeacher(fixture.PublicFixture.OpaqueToken, fixture.HiddenHypothesis)
		secondary, _ := causaloracle.NewTeacher(paired.PublicFixture.OpaqueToken, paired.HiddenHypothesis)
		observation, executeErr := causalrun.ExecuteControl(context.Background(), trial.name, causalrun.ControlInput{FixtureBytes: reference.publicBytes, ProfileBytes: reference.profileBytes, Teacher: primary, PairedFixtureBytes: fixtureBytes, PairedProfileBytes: profileBytes, PairedTeacher: secondary})
		if executeErr != nil || !observation.Passed {
			t.Fatalf("%s observation=%+v err=%v", trial.name, observation, executeErr)
		}
	}
}

func TestCompleteExecutedControlAdapter(t *testing.T) {
	bundle := executedControlBundle(t)
	if len(bundle.Certificates) != len(causalv2.ControlNames) {
		t.Fatalf("control count=%d", len(bundle.Certificates))
	}
	if err := verifyExecutedControlBundle(bundle); err != nil {
		t.Fatal(err)
	}
	encoded := mustCanonical(bundle)
	if len(encoded) > causalv2.PreregisteredManifest().ControlBundleByteCap {
		t.Fatalf("exact control evidence bytes=%d exceed cap", len(encoded))
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 3 {
		t.Fatalf("control bundle fields=%d, want exact three-field schema", len(decoded))
	}
}

func TestExactControlProofForgeryRejected(t *testing.T) {
	bundle := executedControlBundle(t)
	bundle.Certificates[0].TreatmentEvidence.Score++
	if err := causalv2.SignControlBundle(&bundle); err == nil {
		t.Fatal("bundle accepted certificate with stale digest")
	}
}

func executedControlBundle(t *testing.T) ControlBundle {
	t.Helper()
	encoded := mustCanonical(validControlBundle(t))
	bundle, err := causalv2.VerifyControlBundle(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestDiagnosticCapabilityRejectsProtectedSeed(t *testing.T) {
	_, err := NewDiagnosticDevelopmentCapability().GenerateDevelopment(122001, 0)
	if err == nil {
		t.Fatal("diagnostic capability exposed a training seed")
	}
}

func TestMeterReconstructionIncludesActiveZeroAndExcludesInactive(t *testing.T) {
	items := make([]MeterItem, len(causalv2.MeterNames))
	for i, name := range causalv2.MeterNames {
		items[i] = MeterItem{Name: name}
	}
	items[0].Active = true
	meters, digest, err := ReconstructMeters(MeterEvaluation, [][]MeterItem{items}, []TaskMeterItem{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if digest == "" || meters[0].Episodes != 1 || meters[1].Episodes != 0 || meters[1].Totals != (Counter{}) || meters[1].Maxima != (Counter{}) {
		t.Fatal("active-zero/inactive aggregate semantics were not preserved")
	}
}

func validControlBundle(t *testing.T) ControlBundle {
	t.Helper()
	bundle := ControlBundle{ControlBundleVersion: causalv2.ControlBundleDomain}
	for _, name := range causalv2.ControlNames {
		result := ControlResult{Actions: []string{}, Outcomes: []string{}, PosteriorDigests: []string{}, Costs: []int{}}
		left, right, observed := result, result, "semantic-projection-equal"
		if name == "static-rule" {
			observed = "static-baseline-executed"
		}
		switch name {
		case "wrong-context", "occupied-name", "alternate-descriptor", "stale-response", "duplicate-response", "corruption-suite":
			left.FailureCode = name + "-rejected"
			observed = "fail-closed:" + left.FailureCode
			if name == "corruption-suite" {
				observed += ";cases=1"
			}
		case "cost-perturbation":
			left.Actions, right.Actions = []string{"do:0=0"}, []string{"do:0=0"}
			left.Outcomes, right.Outcomes = []string{"000"}, []string{"000"}
			left.Costs, right.Costs = []int{1}, []int{2}
			observed = "stale-rejected-fresh-recomputed"
		case "deterministic-json":
			observed = "canonical-bytes-equal"
		case "mutation-inert":
			observed = "semantic-projection-equal;off-mutants=0;on-mutants=1"
		case "child-vm":
			right.FailureCode = "child-vm-unauthorized"
			observed = "fail-closed:child-vm-unauthorized"
		case "no-credit":
			right.FailureCode = "unresolved-selection"
			observed = "credit-disabled-unresolved"
		case "dependency":
			observed = "forbidden-dependencies=0;files=1;imports=1;methods=1;lookups=1"
		}
		certificate := ControlCertificate{ControlVersion: causalv2.ControlCertificateDomain, Name: name, FixtureDigest: strings.Repeat("f", 64), TreatmentEvidence: left, ControlEvidence: right, Observed: observed, Passed: true}
		if name == "corruption-suite" {
			counter := Counter{ProfileFields: 1, ArtifactMaterializations: 1}
			counter.TotalWork = counter.ComputedTotalWork()
			certificate.MeterCounts, certificate.Work = counter.Counts(), counter.TotalWork
		} else if name == "dependency" {
			counter := Counter{ArtifactMaterializations: 1, TableLookups: 1}
			counter.TotalWork = counter.ComputedTotalWork()
			certificate.MeterCounts, certificate.Work = counter.Counts(), counter.TotalWork
		}
		if err := causalv2.SignControlCertificate(&certificate); err != nil {
			t.Fatal(err)
		}
		bundle.Certificates = append(bundle.Certificates, certificate)
	}
	if err := causalv2.SignControlBundle(&bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestEvaluationReportByteReconstruction(t *testing.T) {
	tasks := []TaskMeterItem{}
	taskDigest, _ := causalv2.TaskMeterItemsDigest(tasks)
	controls := validControlBundle(t)
	report := EvaluationReport{
		ReportVersion:        "causal-diagnosis-report/v2",
		Manifest:             causalv2.PreregisteredManifest(),
		PlanCommit:           PlanCommit,
		Panel:                "development",
		Status:               "invalid",
		ControlBundle:        controls,
		ControlBundleDigest:  controls.ControlBundleDigest,
		TaskMeterItems:       tasks,
		TaskMeterItemsDigest: taskDigest,
		Policies:             []PolicyReport{},
		Contrasts:            []Contrast{},
		Limitations:          []string{},
	}
	if _, err := FinalizeEvaluationReport(&report); err == nil {
		t.Fatal("synthetic signed controls were accepted as executed evidence")
	}
}
