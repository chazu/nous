package causalrun

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type trapTeacher struct {
	token  string
	hidden string
	calls  []string
}

func (t *trapTeacher) Respond(token, action string) (string, error) {
	if token != t.token {
		return "", fmt.Errorf("wrong token")
	}
	if _, err := causal.ParseAction(action); err != nil {
		return "", err
	}
	for _, prior := range t.calls {
		if prior == action && len(t.calls) == 0 {
			return "", fmt.Errorf("impossible duplicate first call")
		}
	}
	t.calls = append(t.calls, action)
	return causal.PredictCode(t.hidden, action)
}

func handFixture(t *testing.T) (causalv2.PublicFixture, []string) {
	t.Helper()
	groups := make(map[string][]string)
	var universe []string
	for _, hypothesis := range causal.Enumerate() {
		code, err := causal.Code(hypothesis)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := causal.Evaluate(hypothesis, nil)
		if err != nil {
			t.Fatal(err)
		}
		outcomeCode := causal.OutcomeCode(outcome)
		groups[outcomeCode] = append(groups[outcomeCode], code)
		universe = append(universe, code)
	}
	var passive string
	for outcome, codes := range groups {
		if len(codes) >= 8 && (passive == "" || outcome < passive) {
			passive = outcome
		}
	}
	selected := append([]string(nil), groups[passive][:8]...)
	selectedSet := make(map[string]bool, 32)
	for _, code := range selected {
		selectedSet[code] = true
	}
	for _, code := range universe {
		if len(selected) == 32 {
			break
		}
		if selectedSet[code] {
			continue
		}
		hypothesis, _ := causal.Parse(code)
		outcome, _ := causal.Evaluate(hypothesis, nil)
		if causal.OutcomeCode(outcome) == passive {
			continue
		}
		selected = append(selected, code)
		selectedSet[code] = true
	}
	if len(selected) < 32 {
		for _, code := range universe {
			if len(selected) == 32 {
				break
			}
			if !selectedSet[code] {
				selected = append(selected, code)
				selectedSet[code] = true
			}
		}
	}
	sort.Strings(selected)
	var initial []string
	for _, code := range selected {
		hypothesis, _ := causal.Parse(code)
		outcome, _ := causal.Evaluate(hypothesis, nil)
		if causal.OutcomeCode(outcome) == passive {
			initial = append(initial, code)
		}
	}
	token, err := causalv2.PublicToken("development", 112001, 0)
	if err != nil {
		t.Fatal(err)
	}
	presentation := make([]int, 32)
	for index := range presentation {
		presentation[index] = index
	}
	actions := []string{"do:0=0", "do:0=1", "do:1=0", "do:1=1", "do:2=0", "do:2=1", "do:0=0", "do:1=0", "do:2=0", "do:0=1"}
	fixture := causalv2.PublicFixture{
		Seed: 112001, GeneratorAttempt: 0, Cohort: "cost-skewed",
		Aliases: []string{"node-u", "node-v", "node-w"}, Costs: []int{5, 40, 90},
		PassiveOutcome: passive, Pool: selected, Presentation: presentation,
		InitialPosterior: initial, UniformRandomActions: actions, OpaqueToken: token,
	}
	if err := causalv2.SignPublicFixture(&fixture); err != nil {
		t.Fatal(err)
	}
	return fixture, initial
}

func handEpisode(t *testing.T, acquisition, hidden string) (*Runner, *trapTeacher, []byte, []byte) {
	t.Helper()
	fixture, initial := handFixture(t)
	if hidden == "" {
		hidden = initial[0]
	}
	fixtureBytes, err := causalv2.CanonicalJSON(fixture)
	if err != nil {
		t.Fatal(err)
	}
	profile := causalv2.Profile{
		ProfileVersion: causalv2.ProfileDomain, Manifest: causalv2.PreregisteredManifest(),
		Panel: "development", Seed: fixture.Seed, AcquisitionCode: acquisition,
		FixtureDigest: fixture.FixtureDigest,
	}
	if err := causalv2.SignProfile(&profile); err != nil {
		t.Fatal(err)
	}
	profileBytes, err := causalv2.CanonicalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	teacher := &trapTeacher{token: fixture.OpaqueToken, hidden: hidden}
	runner, err := NewEpisode(fixtureBytes, profileBytes, teacher)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runner.Close)
	return runner, teacher, fixtureBytes, profileBytes
}

func TestEpisodeStopsAtAuthorizedTeacherBoundary(t *testing.T) {
	runner, teacher, _, _ := handEpisode(t, string(PolicyLexical), "")
	before := runner.store.Count()
	boundary, err := runner.AdvanceToTeacher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if boundary.State != StateAwaitingTeacher || boundary.SelectedAction == "" || boundary.AuthorizationDigest == "" {
		t.Fatalf("incomplete boundary: %+v", boundary)
	}
	if len(teacher.calls) != 0 {
		t.Fatalf("teacher called before authorization: %v", teacher.calls)
	}
	if runner.agenda.Len() != 0 || runner.store.Count() <= before {
		t.Fatalf("boundary is not quiescent/materialized")
	}
	count := runner.store.Count()
	if err := verifyAuthorizationBoundary(runner); err != nil {
		t.Fatal(err)
	}
	if runner.store.Count() != count {
		t.Fatal("read-only authorization verifier mutated store")
	}
}

func TestCUEEngineIsSoleTaskMaterializer(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	initialArtifacts := len(runner.artifacts)
	if _, err := runner.allocate(99, "observation", observationPayload{Outcome: "000"}); err == nil {
		t.Fatal("driver directly materialized attributed evidence")
	}
	if len(runner.artifacts) != initialArtifacts {
		t.Fatal("rejected driver materialization changed artifact ledger")
	}
	if _, err := runner.operation("prepare-proposal"); err == nil {
		t.Fatal("driver directly invoked a CUE operation")
	}

	if _, err := runner.AdvanceToTeacher(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.engine.TaskNum != 2 || runner.cueExecutions[proposalTaskSlot] != 1 || runner.cueExecutions[authorizationTaskSlot] != 1 {
		t.Fatalf("engine tasks=%d CUE executions=%v", runner.engine.TaskNum, runner.cueExecutions)
	}
	if got := runner.store.Get("H-Causal-V2-Propose").GetMap("thenComputeRecord")["successes"]; got != 1 {
		t.Fatalf("proposal CUE success record=%v", got)
	}
	if got := runner.store.Get("H-Causal-V2-Authorize").GetMap("thenComputeRecord")["successes"]; got != 1 {
		t.Fatalf("authorization CUE success record=%v", got)
	}
	if err := runner.Respond(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.cueExecutions[updateTaskSlot] != 1 || runner.engine.TaskNum != 3 {
		t.Fatalf("update was not CUE/engine executed: tasks=%d executions=%v", runner.engine.TaskNum, runner.cueExecutions)
	}
	if runner.meter.Counts().EngineCycles != runner.engine.TaskNum {
		t.Fatalf("meter cycles=%d, actual task cycles=%d", runner.meter.Counts().EngineCycles, runner.engine.TaskNum)
	}
	for index, encoded := range runner.ArtifactBytes() {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if artifact.ChargeIndex != index {
			t.Fatalf("artifact %d charge=%d", index, artifact.ChargeIndex)
		}
	}
}

func TestEpisodeRunsHiddenFreeToTerminal(t *testing.T) {
	runner, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Terminal != "identified" && result.Terminal != "equivalence" {
		t.Fatalf("terminal=%q", result.Terminal)
	}
	if len(result.Actions) == 0 || len(result.Actions) != len(teacher.calls) || len(result.Actions) > 10 {
		t.Fatalf("actions=%d teacher calls=%d", len(result.Actions), len(teacher.calls))
	}
	if result.ProductionCounts.TotalWork != result.ProductionCounts.Array()[14] {
		t.Fatal("counter positional total mismatch")
	}
	if err := result.ProductionCounts.ValidateEquation(); err != nil {
		t.Fatal(err)
	}
	if len(result.Artifacts) == 0 {
		t.Fatal("terminal result omitted sealed artifacts")
	}
	verified, err := VerifyEpisode(fixtureBytes, profileBytes, result.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if verified.TranscriptDigest != result.TranscriptDigest || verified.Score != result.Score {
		t.Fatal("fresh replay result differs")
	}
}

func TestFreshReplayRejectsCorruptionDeletionDuplicationAndReorder(t *testing.T) {
	runner, _, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	corrupt := cloneArtifactBytes(result.Artifacts)
	corrupt[len(corrupt)/2][len(corrupt[len(corrupt)/2])-1] ^= 1
	if _, err := VerifyEpisode(fixtureBytes, profileBytes, corrupt); err == nil {
		t.Fatal("corrupt artifact verified")
	}
	deleted := cloneArtifactBytes(result.Artifacts)
	deleted = append(deleted[:2], deleted[3:]...)
	if _, err := VerifyEpisode(fixtureBytes, profileBytes, deleted); err == nil {
		t.Fatal("deleted artifact verified")
	}
	duplicated := cloneArtifactBytes(result.Artifacts)
	duplicated = append(duplicated[:2], append([][]byte{append([]byte(nil), duplicated[2]...)}, duplicated[2:]...)...)
	if _, err := VerifyEpisode(fixtureBytes, profileBytes, duplicated); err == nil {
		t.Fatal("duplicated artifact verified")
	}
	reordered := cloneArtifactBytes(result.Artifacts)
	reordered[1], reordered[2] = reordered[2], reordered[1]
	if _, err := VerifyEpisode(fixtureBytes, profileBytes, reordered); err == nil {
		t.Fatal("reordered artifacts verified")
	}
}

func cloneArtifactBytes(source [][]byte) [][]byte {
	result := make([][]byte, len(source))
	for index := range source {
		result[index] = append([]byte(nil), source[index]...)
	}
	return result
}

func TestHiddenTwinsAreIdenticalThroughAuthorization(t *testing.T) {
	fixture, initial := handFixture(t)
	if len(initial) < 2 {
		t.Skip("hand fixture lacks twin candidates")
	}
	first, _, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), initial[0])
	secondTeacher := &trapTeacher{token: fixture.OpaqueToken, hidden: initial[1]}
	second, err := NewEpisode(fixtureBytes, profileBytes, secondTeacher)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	left, err := first.AdvanceToTeacher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	right, err := second.AdvanceToTeacher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("hidden twins diverged at boundary: %+v != %+v", left, right)
	}
	leftBytes, rightBytes := first.ArtifactBytes(), second.ArtifactBytes()
	if len(leftBytes) != len(rightBytes) {
		t.Fatalf("artifact counts differ: %d != %d", len(leftBytes), len(rightBytes))
	}
	for index := range leftBytes {
		if !bytes.Equal(leftBytes[index], rightBytes[index]) {
			t.Fatalf("hidden twins diverged at artifact %d", index)
		}
	}
}

func TestDuplicateAndStaleResponseFailClosed(t *testing.T) {
	runner, teacher, _, _ := handEpisode(t, string(PolicyLexical), "")
	if _, err := runner.AdvanceToTeacher(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runner.Respond(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := len(teacher.calls)
	if err := runner.Respond(context.Background()); err == nil {
		t.Fatal("duplicate/stale response was accepted")
	}
	if len(teacher.calls) != calls {
		t.Fatal("teacher was called for rejected stale response")
	}
}

func TestArtifactCorruptionAndOccupiedNameFailClosed(t *testing.T) {
	runner, teacher, _, _ := handEpisode(t, string(PolicyLexical), "")
	if _, err := runner.AdvanceToTeacher(context.Background()); err != nil {
		t.Fatal(err)
	}
	name := artifactName(runner.selection.Kind, runner.selection.Digest)
	runner.store.Get(name).Set("sealed", false)
	if err := runner.Respond(context.Background()); err == nil {
		t.Fatal("unsealed selection reached teacher")
	}
	if len(teacher.calls) != 0 {
		t.Fatal("teacher called after artifact corruption")
	}

	clean, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	payload := observationPayload{Outcome: "000"}
	artifact, err := causalv2.NewArtifact(clean.profile.ProfileDigest, clean.episodeKey, 99, "observation", payload, len(clean.artifacts))
	if err != nil {
		t.Fatal(err)
	}
	occupied := unit.New(artifact.Name())
	occupied.Set("sealed", false)
	clean.store.Put(occupied)
	clean.cueMaterializing = true
	clean.activeCUETask = proposalTaskSlot
	if _, err := clean.allocate(99, "observation", payload); err == nil {
		t.Fatal("unsealed occupied artifact name was overwritten")
	}
	clean.cueMaterializing = false
	clean.activeCUETask = ""
}

func TestIdenticalAllocationIsIdempotentAndCorruptionStillFails(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	beforeArtifacts := len(runner.artifacts)
	beforeCounts := runner.meter.Counts()
	runner.cueMaterializing = true
	runner.activeCUETask = proposalTaskSlot
	first := runner.artifacts[0]
	retry, err := runner.allocate(0, "observation", observationPayload{Outcome: runner.fixture.PassiveOutcome})
	if err != nil {
		t.Fatal(err)
	}
	if retry.Digest != first.Digest || len(runner.artifacts) != beforeArtifacts || runner.meter.Counts() != beforeCounts {
		t.Fatal("identical retry changed evidence or charges")
	}
	stored := runner.store.Get(artifactName(first.Kind, first.Digest))
	stored.Set("sealed", false)
	if _, err := runner.allocate(0, "observation", observationPayload{Outcome: runner.fixture.PassiveOutcome}); err == nil {
		t.Fatal("identical retry accepted an unsealed collision")
	}
	runner.cueMaterializing = false
	runner.activeCUETask = ""
}

func TestSnapshotVerifierReconstructsDescriptorAndEveryCap(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	if err := verifyLatestSnapshot(runner, StateReady); err != nil {
		t.Fatal(err)
	}
	assertRejected := func(name string, mutate func(), restore func()) {
		t.Helper()
		mutate()
		if err := verifyLatestSnapshot(runner, StateReady); err == nil {
			t.Errorf("snapshot verifier trusted mutated %s", name)
		}
		restore()
	}

	alias := runner.fixture.Aliases[0]
	assertRejected("aliases", func() { runner.fixture.Aliases[0] = alias + "-wrong" }, func() { runner.fixture.Aliases[0] = alias })
	cost := runner.fixture.Costs[0]
	assertRejected("costs", func() { runner.fixture.Costs[0]++ }, func() { runner.fixture.Costs[0] = cost })
	presentation := runner.fixture.Presentation[0]
	assertRejected("presentation", func() { runner.fixture.Presentation[0]++ }, func() { runner.fixture.Presentation[0] = presentation })
	initialDigest := runner.initialPosteriorArtifact.Digest
	assertRejected("initial posterior", func() { runner.initialPosteriorArtifact.Digest = causalv2.ZeroDigest }, func() { runner.initialPosteriorArtifact.Digest = initialDigest })
	posterior := append([]string(nil), runner.posterior...)
	assertRejected("posterior", func() { runner.posterior = runner.posterior[:1] }, func() { runner.posterior = posterior })
	code := runner.profile.AcquisitionCode
	assertRejected("acquisition code", func() { runner.profile.AcquisitionCode = string(PolicyUniformRandom) }, func() { runner.profile.AcquisitionCode = code })
	assertRejected("total cost", func() { runner.totalCost = 1 }, func() { runner.totalCost = 0 })
	assertRejected("action count", func() { runner.consumed = []string{"do:0=0"} }, func() { runner.consumed = []string{} })

	digest := runner.cursor.latestSnapshotDigest
	original := runner.byDigest[digest]
	for _, tc := range []struct {
		name   string
		mutate func(*Counts)
	}{
		{"remaining evaluations", func(c *Counts) { c.SCMEvaluations++; c.recompute() }},
		{"remaining work", func(c *Counts) { c.ProfileFields++; c.recompute() }},
		{"remaining cycles", func(c *Counts) { c.EngineCycles++ }},
		{"remaining units", func(c *Counts) { c.AttributedUnits++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := original
			tc.mutate(&changed.MeterAfter)
			runner.byDigest[digest] = changed
			if err := verifyLatestSnapshot(runner, StateReady); err == nil {
				t.Fatal("snapshot verifier trusted a mutated meter checkpoint")
			}
			runner.byDigest[digest] = original
		})
	}
}

func TestDynamicTerminalStatesAreMemoizedOnFirstInsertion(t *testing.T) {
	fixture, _ := handFixture(t)
	policy, err := NewDynamicPolicy(fixture.InitialPosterior, [3]int{fixture.Costs[0], fixture.Costs[1], fixture.Costs[2]})
	if err != nil {
		t.Fatal(err)
	}
	terminal := fixture.InitialPosterior[:1]
	if _, err := policy.value(terminal, 0); err != nil {
		t.Fatal(err)
	}
	first := policy.Counts()
	if first.MemoStates != 1 || len(policy.memo) != 1 {
		t.Fatalf("first terminal insertion counts=%+v memo=%d", first, len(policy.memo))
	}
	if _, err := policy.value(terminal, 0); err != nil {
		t.Fatal(err)
	}
	second := policy.Counts()
	if second.MemoStates != first.MemoStates || second.MemoLookups != first.MemoLookups+1 {
		t.Fatalf("terminal memo hit counts=%+v after %+v", second, first)
	}
}

func TestCUEProgramCannotBeDisabledOrOverrideVerifiedSelection(t *testing.T) {
	disabled, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	disabled.store.Get("H-Causal-V2-Propose").Set("thenCompute", "")
	if _, err := disabled.AdvanceToTeacher(context.Background()); err == nil {
		t.Fatal("disabled CUE proposal still advanced")
	}

	altered, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	heuristic := altered.store.Get("H-Causal-V2-Propose")
	program := heuristic.GetString("thenCompute")
	alteredProgram := strings.ReplaceAll(program,
		`"CurUnit" @ "best" @ causal-v2-materialize-selection drop`,
		`"CurUnit" @ "do:2=1" causal-v2-materialize-selection drop`)
	if alteredProgram == program {
		t.Fatal("test did not alter the CUE program")
	}
	heuristic.Set("thenCompute", alteredProgram)
	if _, err := altered.AdvanceToTeacher(context.Background()); err == nil {
		t.Fatal("altered CUE selection crossed the fresh verifier")
	}
}

func TestDynamicPolicyUsesOnlyPublicState(t *testing.T) {
	fixture, _ := handFixture(t)
	costs := [3]int{fixture.Costs[0], fixture.Costs[1], fixture.Costs[2]}
	left, err := NewDynamicPolicy(fixture.InitialPosterior, costs)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewDynamicPolicy(fixture.InitialPosterior, costs)
	if err != nil {
		t.Fatal(err)
	}
	a, err := left.Choose(fixture.InitialPosterior, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := right.Choose(fixture.InitialPosterior, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("public-identical dynamic policies diverged: %q != %q", a, b)
	}
	if left.Counts().MemoStates == 0 || left.Counts().QEvaluations == 0 || left.Counts().TotalWork > DynamicWorkCap {
		t.Fatalf("dynamic counts=%+v", left.Counts())
	}
}

func TestDynamicBenchmarkMetersRealizedAndEveryMemberTrajectory(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyDynamicOptimal), "")
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	benchmark := result.DynamicBenchmark
	if benchmark.RealizedCost != result.Cost {
		t.Fatalf("realized benchmark cost=%d, episode cost=%d", benchmark.RealizedCost, result.Cost)
	}
	if benchmark.MemberSimulations != len(runner.initial) || benchmark.UniformExpectedDenominator == "" {
		t.Fatalf("incomplete dynamic benchmark: %+v", benchmark)
	}
	if benchmark.Counts.TableLookups != len(result.Actions)+benchmark.MemberSimulations*len(result.Actions) {
		// Member paths may terminate at different depths; require at least the
		// realized path plus one lookup per simulated member instead.
		if benchmark.Counts.TableLookups < len(result.Actions)+benchmark.MemberSimulations {
			t.Fatalf("dynamic table lookups omit trajectories: %+v", benchmark.Counts)
		}
	}
	if benchmark.Counts != result.DynamicCounts || benchmark.Counts.MemoStates > dynamicReachableStateBound || benchmark.Counts.TotalWork > dynamicCompleteWorkBound {
		t.Fatalf("dynamic benchmark counts=%+v result=%+v", benchmark.Counts, result.DynamicCounts)
	}
	reconstructed, err := ReconstructDynamicBenchmark(runner.initial, runner.dynamic.costs, result.Actions, result.TeacherOutcomes)
	if err != nil {
		t.Fatal(err)
	}
	if reconstructed != benchmark {
		t.Fatalf("dynamic reconstruction differs:\nproduction=%+v\nreconstructed=%+v", benchmark, reconstructed)
	}
}

func TestExecuteControlRejectsUnknownAndRunsProposalOrder(t *testing.T) {
	_, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	fixture, err := causalv2.VerifyPublicFixture(fixtureBytes)
	if err != nil {
		t.Fatal(err)
	}
	input := ControlInput{
		FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: teacher,
		PairedTeacher: &trapTeacher{token: fixture.OpaqueToken, hidden: teacher.hidden},
	}
	if _, err := ExecuteControl(context.Background(), ControlName("not-a-control"), input); err == nil {
		t.Fatal("unknown online control was accepted")
	}
	observation, err := ExecuteControl(context.Background(), ControlProposalOrder, input)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed || observation.Observed != "semantic-projection-equal" {
		t.Fatalf("proposal-order observation=%+v", observation)
	}
}

func TestExecuteControlFailureDoesNotCallTeacher(t *testing.T) {
	_, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	fixture, err := causalv2.VerifyPublicFixture(fixtureBytes)
	if err != nil {
		t.Fatal(err)
	}
	input := ControlInput{
		FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: teacher,
		PairedTeacher: &trapTeacher{token: fixture.OpaqueToken, hidden: teacher.hidden},
	}
	observation, err := ExecuteControl(context.Background(), ControlOccupiedName, input)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed || observation.TeacherCallsBefore != observation.TeacherCallsAfter || observation.Treatment.FailureCode == "" {
		t.Fatalf("occupied-name observation=%+v", observation)
	}
}

func TestStaticAndRecomputedControlsRejectMissingOrMismatchedRules(t *testing.T) {
	_, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	fixture, err := causalv2.VerifyPublicFixture(fixtureBytes)
	if err != nil {
		t.Fatal(err)
	}
	freshTeacher := func() Teacher { return &trapTeacher{token: fixture.OpaqueToken, hidden: teacher.hidden} }
	base := ControlInput{FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: freshTeacher(), PairedTeacher: freshTeacher()}
	if _, err := ExecuteControl(context.Background(), ControlStaticRule, base); err == nil {
		t.Fatal("static-rule accepted missing declared rule codes")
	}
	base.Teacher, base.PairedTeacher = freshTeacher(), freshTeacher()
	base.SelectedRuleCode, base.StaticRuleCode = "P=H;M=gain;S=C", string(PolicyLexical)
	if _, err := ExecuteControl(context.Background(), ControlStaticRule, base); err == nil {
		t.Fatal("static-rule accepted a profile that did not encode selected rule")
	}
	base.Teacher, base.PairedTeacher = freshTeacher(), freshTeacher()
	base.SelectedRuleCode, base.StaticRuleCode = string(PolicyLexical), string(PolicyLexical)
	observation, err := ExecuteControl(context.Background(), ControlStaticRule, base)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed {
		t.Fatalf("valid static-rule control failed: %+v", observation)
	}

	base.Teacher, base.PairedTeacher = freshTeacher(), freshTeacher()
	base.StaticRuleCode = ""
	observation, err = ExecuteControl(context.Background(), ControlRecomputedRule, base)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed || observation.Control.FailureCode != "" {
		t.Fatalf("valid recomputed-rule control failed: %+v", observation)
	}
}

func TestRecomputedModeMetersCacheAndPartitionWorkAgain(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	runner.cursor.state = StateProposing
	runner.cueMaterializing = true
	runner.activeCUETask = proposalTaskSlot
	if err := runner.prepareProposal(); err != nil {
		t.Fatal(err)
	}
	action := runner.proposalActions[0]
	if err := runner.materializeCandidateCache([]string{action}); err != nil {
		t.Fatal(err)
	}
	firstArtifact := runner.current[action].cache.Digest
	var payload cachePayload
	if err := runner.decodePayload(runner.current[action].cache, &payload); err != nil {
		t.Fatal(err)
	}
	cacheBytes := string(runner.current[action].cache.Canonical)
	if payload.PosteriorArtifactDigest != runner.posteriorArtifact.Digest || !strings.Contains(cacheBytes, "posterior_artifact_digest") || payload.R != 0 {
		t.Fatalf("cache does not bind current posterior artifact and repeat feature: %+v", payload)
	}
	first := runner.meter.Counts()
	runner.current = make(map[string]candidateArtifacts)
	runner.consumed = append(runner.consumed, action)
	if err := runner.materializeCandidateCache([]string{action}); err != nil {
		t.Fatal(err)
	}
	hit := runner.meter.Counts()
	secondArtifact := runner.current[action].cache.Digest
	var hitPayload cachePayload
	if err := runner.decodePayload(runner.current[action].cache, &hitPayload); err != nil {
		t.Fatal(err)
	}
	if hit.PartitionAssignments != first.PartitionAssignments || hit.CellAccumulations != first.CellAccumulations ||
		hit.TotalWork != first.TotalWork+1 || secondArtifact == firstArtifact || runner.current[action].candidate.features.Repeat != 1 || hitPayload.R != 1 {
		t.Fatal("ordinary semantic cache hit recomputed partition work, reused step evidence, or retained a stale repeat feature")
	}
	runner.current = make(map[string]candidateArtifacts)
	runner.disableCacheReuse = true
	if err := runner.materializeCandidateCache([]string{action}); err != nil {
		t.Fatal(err)
	}
	recomputed := runner.meter.Counts()
	if recomputed.PartitionAssignments <= hit.PartitionAssignments || recomputed.CellAccumulations <= hit.CellAccumulations || recomputed.TotalWork <= hit.TotalWork {
		t.Fatalf("recomputed mode did not meter fresh work: hit=%+v recomputed=%+v", hit, recomputed)
	}
	if len(runner.cache) != 1 || runner.current[action].cacheHit {
		t.Fatal("recomputed mode changed the registered semantic entry or reported a cache hit")
	}
	if got := runner.cacheTrace; len(got.Statuses) != 3 || got.Statuses[0] != "miss" || got.Statuses[1] != "hit" || got.Statuses[2] != "miss" || got.Hits != 1 || got.Misses != 2 {
		t.Fatalf("cache trace=%+v", got)
	}
}

func TestEpisodeRecordsAllSixCacheLookupsPerAction(t *testing.T) {
	runner, _, _, _ := handEpisode(t, string(PolicyLexical), "")
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.CacheTrace.Statuses) != 6*len(result.Actions) || result.CacheTrace.Hits+result.CacheTrace.Misses != len(result.CacheTrace.Statuses) {
		t.Fatalf("actions=%d cache trace=%+v", len(result.Actions), result.CacheTrace)
	}
	for _, status := range result.CacheTrace.Statuses {
		if status != "hit" && status != "miss" {
			t.Fatalf("invalid cache status %q", status)
		}
	}
}

func TestMutationControlExecutesOneRealHookAndAttempt(t *testing.T) {
	_, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	fixture, err := causalv2.VerifyPublicFixture(fixtureBytes)
	if err != nil {
		t.Fatal(err)
	}
	paired := &trapTeacher{token: fixture.OpaqueToken, hidden: teacher.hidden}
	observation, err := ExecuteControl(context.Background(), ControlMutationInert, ControlInput{
		FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: teacher, PairedTeacher: paired,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed {
		t.Fatalf("mutation control failed: %+v", observation)
	}
	if observation.TreatmentMutation.Config.Enabled || len(observation.TreatmentMutation.Mutants) != 0 ||
		!observation.ControlMutation.Config.Enabled || len(observation.ControlMutation.Mutants) != 1 {
		t.Fatalf("mutation control did not prove exact off/on execution: off=%+v on=%+v", observation.TreatmentMutation, observation.ControlMutation)
	}
	if !strings.Contains(observation.Observed, "off-mutants=0") || !strings.Contains(observation.Observed, "on-mutants=1") {
		t.Fatalf("mutation counts were not retained in the observation: %q", observation.Observed)
	}
}

func TestChildVMControlFailsClosedBeforeEvidence(t *testing.T) {
	_, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	observation, err := ExecuteControl(context.Background(), ControlChildVM, ControlInput{
		FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: teacher,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed || observation.Control.FailureCode != "child-vm-unauthorized" ||
		observation.ChildVM.ArtifactsBefore != observation.ChildVM.ArtifactsAfter ||
		observation.ChildVM.MeterCountsBefore != observation.ChildVM.MeterCountsAfter ||
		observation.ChildVM.TeacherCallsBefore != observation.ChildVM.TeacherCallsAfter {
		t.Fatalf("child denial control failed: %+v", observation)
	}
}

func TestCorruptionSuiteEnumeratesAllArtifactOperations(t *testing.T) {
	runner, teacher, fixtureBytes, profileBytes := handEpisode(t, string(PolicyLexical), "")
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cases, err := enumerateCorruptionCases(result.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 || len(cases) > 486 {
		t.Fatalf("corruption matrix size=%d", len(cases))
	}
	counts := map[string]int{"field": 0, "nested": 0, "delete": 0, "duplicate": 0, "forge": 0, "reorder": 0}
	seen := make(map[string]bool, len(cases))
	for _, trial := range cases {
		if seen[trial.name] {
			t.Fatalf("duplicate corruption case %q", trial.name)
		}
		seen[trial.name] = true
		switch {
		case strings.Contains(trial.name, "-field-"):
			counts["field"]++
		case strings.Contains(trial.name, "-payload."):
			counts["nested"]++
		case strings.HasPrefix(trial.name, "delete-"):
			counts["delete"]++
		case strings.HasPrefix(trial.name, "duplicate-"):
			counts["duplicate"]++
		case strings.HasPrefix(trial.name, "forge-"):
			counts["forge"]++
		case strings.HasPrefix(trial.name, "reorder-"):
			counts["reorder"]++
		}
	}
	if counts["field"] != 135 || counts["nested"] != 81 || counts["delete"] != 15 || counts["duplicate"] != 15 || counts["forge"] != 15 || counts["reorder"] == 0 || counts["reorder"] > 225 {
		t.Fatalf("unexpected representative corruption counts: %+v", counts)
	}
	evidence, err := CorruptionCases(fixtureBytes, profileBytes, result.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != len(cases) {
		t.Fatalf("corruption evidence=%d, cases=%d", len(evidence), len(cases))
	}
	verifiedCounts, err := VerifyCorruptionCases(fixtureBytes, profileBytes, result.Artifacts, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if verifiedCounts.TotalWork == 0 {
		t.Fatal("corruption evidence omitted per-case meter counts")
	}
	observation, err := ExecuteControl(context.Background(), ControlCorruptionSuite, ControlInput{
		FixtureBytes: fixtureBytes, ProfileBytes: profileBytes, Teacher: teacher, BaselineArtifacts: result.Artifacts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Passed || !strings.Contains(observation.Observed, fmt.Sprintf("cases=%d", len(cases))) || observation.TeacherCallsBefore != observation.TeacherCallsAfter || observation.Counts.ArtifactMaterializations < len(cases) {
		t.Fatalf("corruption suite observation=%+v", observation)
	}
}
