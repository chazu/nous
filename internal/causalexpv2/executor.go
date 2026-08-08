package causalexpv2

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type DevelopmentDiagnostics struct {
	Panel               string             `json:"panel"`
	Seeds               causalv2.SeedRange `json:"seeds"`
	Episodes            int                `json:"episodes"`
	OracleAgreements    int                `json:"oracle_agreements"`
	OracleDisagreements int                `json:"oracle_disagreements"`
	PolicyScores        map[string][]int   `json:"policy_scores"`
}

type executedEpisode struct {
	fixture       PrivateFixture
	publicBytes   []byte
	profile       causalv2.Profile
	profileBytes  []byte
	result        causalrun.EpisodeResult
	replay        causalrun.EpisodeResult
	evidence      EpisodeEvidence
	certificate   ApplicationCertificate
	oracleCounter Counter
	dynamic       causalrun.DynamicBenchmark
}

func runEpisode(ctx context.Context, panel Panel, fixture PrivateFixture, acquisition string, ruleCode string, dpOwner bool) (executedEpisode, error) {
	publicBytes, err := causalv2.CanonicalJSON(fixture.PublicFixture)
	if err != nil {
		return executedEpisode{}, err
	}
	if _, err := causalv2.VerifyPublicFixtureForPanel(publicBytes, string(panel)); err != nil {
		return executedEpisode{}, err
	}
	if panel != PanelDevelopment {
		if err := causalv2.VerifyPreregisteredFixtureContext(fixture.PublicFixture, string(panel)); err != nil {
			return executedEpisode{}, err
		}
	}
	profile := causalv2.Profile{ProfileVersion: causalv2.ProfileDomain, Manifest: causalv2.PreregisteredManifest(), Panel: string(panel), Seed: fixture.PublicFixture.Seed, AcquisitionCode: acquisition, FixtureDigest: fixture.PublicFixture.FixtureDigest}
	if err := causalv2.SignProfile(&profile); err != nil {
		return executedEpisode{}, err
	}
	profileBytes, err := causalv2.CanonicalJSON(profile)
	if err != nil {
		return executedEpisode{}, err
	}
	registry := NewTeacherRegistry()
	if err := registry.Bind(fixture); err != nil {
		return executedEpisode{}, err
	}
	teacher, err := registry.Teacher(fixture.PublicFixture.OpaqueToken)
	if err != nil {
		return executedEpisode{}, err
	}
	runner, err := causalrun.NewEpisode(publicBytes, profileBytes, teacher)
	if err != nil {
		return executedEpisode{}, err
	}
	result, runErr := runner.Run(ctx)
	runner.Close()
	if runErr != nil {
		return executedEpisode{}, runErr
	}
	replay, err := causalrun.VerifyEpisode(publicBytes, profileBytes, result.Artifacts)
	if err != nil {
		return executedEpisode{}, fmt.Errorf("fresh production replay: %w", err)
	}
	if !equalCanonical(result, replay) {
		return executedEpisode{}, errors.New("fresh artifact replay result differs from production result")
	}
	agreements, disagreements, oracleCounter, err := auditEpisode(fixture, result)
	if err != nil {
		return executedEpisode{}, err
	}
	items := inactiveMeterItems()
	items[0] = MeterItem{Name: "production", Active: true, Counts: counts64(result.ProductionCounts)}
	items[1] = MeterItem{Name: "teacher", Active: true, Counts: counts64(result.TeacherCounts)}
	items[4] = MeterItem{Name: "oracle-audit", Active: true, Counts: oracleCounter.Counts()}
	dynamicBenchmark := result.DynamicBenchmark
	if dpOwner {
		dynamicActions := result.Actions
		dynamicOutcomes := result.TeacherOutcomes
		if acquisition != string(causalrun.PolicyDynamicOptimal) {
			dynamicEpisode, dynamicErr := runEpisode(ctx, panel, fixture, string(causalrun.PolicyDynamicOptimal), "", false)
			if dynamicErr != nil {
				return executedEpisode{}, fmt.Errorf("complete dynamic benchmark: %w", dynamicErr)
			}
			dynamicBenchmark = dynamicEpisode.result.DynamicBenchmark
			dynamicActions = dynamicEpisode.result.Actions
			dynamicOutcomes = dynamicEpisode.result.TeacherOutcomes
		}
		costs := [3]int{fixture.PublicFixture.Costs[0], fixture.PublicFixture.Costs[1], fixture.PublicFixture.Costs[2]}
		reconstructed, dynamicErr := causalrun.ReconstructDynamicBenchmark(fixture.PublicFixture.InitialPosterior, costs, dynamicActions, dynamicOutcomes)
		if dynamicErr != nil || !equalCanonical(dynamicBenchmark, reconstructed) {
			return executedEpisode{}, errors.New("complete dynamic benchmark does not independently reconstruct")
		}
		items[5] = MeterItem{Name: "dp", Active: true, Counts: counts64(dynamicBenchmark.Counts)}
	}
	valid, err := episodeCapsValid(scopeForPanel(panel), items)
	if err != nil {
		return executedEpisode{}, err
	}
	evidence := EpisodeEvidence{EpisodeReportVersion: "causal-training-episode/v2", Seed: result.Seed, ProfileDigest: result.ProfileDigest, FixtureDigest: result.FixtureDigest, RuleCode: ruleCode, Actions: append([]string(nil), result.Actions...), TeacherOutcomes: append([]string(nil), result.TeacherOutcomes...), Terminal: result.Terminal, Score: result.Score, Cost: result.Cost, FinalPosterior: append([]string(nil), result.FinalPosterior...), PosteriorDigest: result.PosteriorDigest, TranscriptDigest: result.TranscriptDigest, HypothesisEvaluations: result.ProductionCounts.SCMEvaluations, SemanticWork: result.ProductionCounts.TotalWork, AttributedUnits: result.ProductionCounts.AttributedUnits, EngineCycles: result.ProductionCounts.EngineCycles, OracleAgreements: agreements, OracleDisagreements: disagreements, MeterItems: items, AllCapsValid: valid}
	if ruleCode != "" {
		if err := SignEpisode(&evidence); err != nil {
			return executedEpisode{}, err
		}
	}
	certificate := ApplicationCertificate{}
	if ruleCode != "" {
		certificate = ApplicationCertificate{Seed: evidence.Seed, ProfileDigest: evidence.ProfileDigest, FixtureDigest: evidence.FixtureDigest, RuleCode: ruleCode, Score: evidence.Score, Terminal: evidence.Terminal, Cost: evidence.Cost, PosteriorDigest: evidence.PosteriorDigest, TranscriptDigest: evidence.TranscriptDigest, OracleAgreements: agreements, OracleDisagreements: disagreements, AllCapsValid: evidence.AllCapsValid, EpisodeReportDigest: evidence.EpisodeReportDigest}
		if err := causalv2.SignApplicationCertificate(&certificate); err != nil {
			return executedEpisode{}, err
		}
	}
	return executedEpisode{fixture: fixture, publicBytes: publicBytes, profile: profile, profileBytes: profileBytes, result: result, replay: replay, evidence: evidence, certificate: certificate, oracleCounter: oracleCounter, dynamic: dynamicBenchmark}, nil
}

func equalCanonical(left, right any) bool {
	leftBytes, leftErr := causalv2.CanonicalJSON(left)
	rightBytes, rightErr := causalv2.CanonicalJSON(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}

func scopeForPanel(panel Panel) MeterScope {
	if panel == PanelTraining {
		return MeterTraining
	}
	return MeterEvaluation
}

func inactiveMeterItems() []MeterItem {
	items := make([]MeterItem, len(causalv2.MeterNames))
	for i, name := range causalv2.MeterNames {
		items[i].Name = name
	}
	return items
}

func counts64(counts causalrun.Counts) [15]int64 {
	values := counts.Array()
	var out [15]int64
	for i, value := range values {
		out[i] = int64(value)
	}
	return out
}

func auditEpisode(fixture PrivateFixture, result causalrun.EpisodeResult) (int, int, Counter, error) {
	posterior := append([]string(nil), fixture.PublicFixture.InitialPosterior...)
	agreements, disagreements := 0, 0
	counts := Counter{}
	for step, action := range result.Actions {
		cells, err := causaloracle.Partition(posterior, action)
		if err != nil {
			return 0, 0, Counter{}, err
		}
		counts.SCMEvaluations += int64(len(posterior))
		counts.PartitionAssignments += int64(len(posterior))
		counts.CellAccumulations += int64(len(cells))
		var next []string
		for _, cell := range cells {
			for _, code := range cell.Models {
				if code == fixture.HiddenHypothesis {
					next = append([]string(nil), cell.Models...)
					if step < len(result.TeacherOutcomes) && cell.Outcome == result.TeacherOutcomes[step] {
						agreements++
					} else {
						disagreements++
					}
					break
				}
			}
		}
		if len(next) == 0 {
			return 0, 0, Counter{}, errors.New("oracle audit eliminated hidden hypothesis")
		}
		counts.PosteriorChecks += int64(len(next))
		posterior = next
	}
	counts.PosteriorChecks += int64(len(posterior))
	counts.TotalWork = counts.ComputedTotalWork()
	want := append([]string(nil), result.FinalPosterior...)
	sort.Strings(want)
	sort.Strings(posterior)
	if fmt.Sprint(want) != fmt.Sprint(posterior) {
		disagreements++
	} else {
		agreements++
	}
	terminal := "budget-exhausted"
	if len(posterior) == 1 {
		terminal = "identified"
	} else if causaloracle.CompleteClass(fixture.PublicFixture.InitialPosterior, posterior) {
		terminal = "equivalence"
	}
	if result.Terminal != terminal {
		disagreements++
	} else {
		agreements++
	}
	return agreements, disagreements, counts, counts.Validate()
}

func developmentAcquisition(policy Policy) string {
	switch policy {
	case "learned":
		return orderedRules()[0].Code()
	case "information-gain-per-cost":
		return "P=H;M=gain;S=C"
	case "worst-split-per-cost":
		return "P=W;M=gain;S=C"
	default:
		return string(policy)
	}
}

func orderedRules() []causal.Rule {
	rules := causal.Rules()
	sort.Slice(rules, func(i, j int) bool { return rules[i].Code() < rules[j].Code() })
	return rules
}

// RunDevelopment is repeatable, non-acceptance diagnostics. It cannot create
// attempts, certificates, evidence, or an acceptance status.
func RunDevelopment(ctx context.Context, repoRoot string) (DevelopmentDiagnostics, error) {
	if _, err := resolveGitState(ctx, repoRoot); err != nil {
		return DevelopmentDiagnostics{}, err
	}
	manifest := causalv2.PreregisteredManifest()
	diagnostics := DevelopmentDiagnostics{Panel: "development", Seeds: manifest.DevelopmentSeeds, PolicyScores: make(map[string][]int)}
	capability := NewDiagnosticDevelopmentCapability()
	for seedIndex := 0; seedIndex < manifest.DevelopmentSeeds.Count; seedIndex++ {
		seed := manifest.DevelopmentSeeds.Start + int64(seedIndex)*manifest.DevelopmentSeeds.Step
		fixture, err := capability.GenerateDevelopment(seed, seedIndex)
		if err != nil {
			return diagnostics, err
		}
		for _, policy := range evaluationPolicies {
			episode, err := runEpisode(ctx, PanelDevelopment, fixture, developmentAcquisition(policy), "", policy == "dynamic-optimal")
			if err != nil {
				return diagnostics, fmt.Errorf("development seed %d policy %s: %w", seed, policy, err)
			}
			diagnostics.Episodes++
			diagnostics.OracleAgreements += episode.evidence.OracleAgreements
			diagnostics.OracleDisagreements += episode.evidence.OracleDisagreements
			diagnostics.PolicyScores[string(policy)] = append(diagnostics.PolicyScores[string(policy)], episode.result.Score)
		}
	}
	return diagnostics, nil
}
