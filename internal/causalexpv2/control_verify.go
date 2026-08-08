package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

func semanticProjectionEqual(left, right ControlResult) bool {
	return slices.Equal(left.Actions, right.Actions) && slices.Equal(left.Outcomes, right.Outcomes) && slices.Equal(left.PosteriorDigests, right.PosteriorDigests) && slices.Equal(left.Costs, right.Costs) && left.Terminal == right.Terminal && left.Score == right.Score
}

func verifyExecutedControlBundle(bundle ControlBundle) error {
	encoded, err := causalv2.CanonicalJSON(bundle)
	if err != nil {
		return err
	}
	verified, err := causalv2.VerifyControlBundle(encoded)
	if err != nil {
		return err
	}
	for index, certificate := range verified.Certificates {
		if index < 16 && certificate.FixtureDigest == "" {
			return fmt.Errorf("control %q lacks executed fixture provenance", certificate.Name)
		}
		left, right := certificate.TreatmentEvidence, certificate.ControlEvidence
		passed, observed := false, ""
		switch certificate.Name {
		case "hidden-twin", "recomputed-rule", "opaque-alias", "presentation-order", "proposal-order":
			passed = semanticProjectionEqual(left, right) && left.FailureCode == "" && right.FailureCode == ""
			observed = "semantic-projection-equal"
		case "static-rule":
			passed = left.FailureCode == "" && right.FailureCode == ""
			observed = "static-baseline-executed"
		case "mutation-inert":
			var offMutants, onMutants int
			matched, scanErr := fmt.Sscanf(certificate.Observed, "semantic-projection-equal;off-mutants=%d;on-mutants=%d", &offMutants, &onMutants)
			observed = fmt.Sprintf("semantic-projection-equal;off-mutants=%d;on-mutants=%d", offMutants, onMutants)
			passed = scanErr == nil && matched == 2 && certificate.Observed == observed && offMutants == 0 && onMutants >= 1 && semanticProjectionEqual(left, right) && left.FailureCode == "" && right.FailureCode == ""
		case "child-vm":
			passed = right.FailureCode == "child-vm-unauthorized"
			observed = "fail-closed:child-vm-unauthorized"
		case "deterministic-json":
			passed = semanticProjectionEqual(left, right) && left.ProfileDigest == right.ProfileDigest && left.TranscriptDigest == right.TranscriptDigest
			observed = "canonical-bytes-equal"
		case "cost-perturbation":
			passed = slices.Equal(left.Actions, right.Actions) && slices.Equal(left.Outcomes, right.Outcomes) && !slices.Equal(left.Costs, right.Costs) && left.FailureCode == "" && right.FailureCode == ""
			observed = "stale-rejected-fresh-recomputed"
		case "wrong-context", "occupied-name", "alternate-descriptor", "stale-response", "duplicate-response":
			failure := left.FailureCode
			if failure == "" {
				failure = right.FailureCode
			}
			passed = failure != ""
			observed = "fail-closed:" + failure
		case "corruption-suite":
			const prefix = "fail-closed:corruption-suite-rejected;cases="
			countText, found := strings.CutPrefix(certificate.Observed, prefix)
			count, countErr := strconv.Atoi(countText)
			meter := causalv2.CounterFromCounts(certificate.MeterCounts)
			passed = found && countErr == nil && count > 0 && left.FailureCode == "corruption-suite-rejected" && right.FailureCode == "" && meter.ProfileFields == int64(count) && meter.ArtifactMaterializations >= int64(count)
			observed = certificate.Observed
		case "no-credit":
			passed = right.FailureCode == "unresolved-selection" && right.Terminal == "" && right.Score == 0
			observed = "credit-disabled-unresolved"
		case "dependency":
			var forbidden, files, imports, methods int
			var lookups int64
			matched, scanErr := fmt.Sscanf(certificate.Observed, "forbidden-dependencies=%d;files=%d;imports=%d;methods=%d;lookups=%d", &forbidden, &files, &imports, &methods, &lookups)
			meter := causalv2.CounterFromCounts(certificate.MeterCounts)
			observed = fmt.Sprintf("forbidden-dependencies=%d;files=%d;imports=%d;methods=%d;lookups=%d", forbidden, files, imports, methods, lookups)
			passed = scanErr == nil && matched == 5 && certificate.Observed == observed && forbidden == 0 && files > 0 && imports > 0 && methods > 0 && lookups > 0 && meter.ArtifactMaterializations == 1 && meter.TableLookups == lookups && certificate.FixtureDigest != "" && left.FailureCode == "" && right.FailureCode == "" && len(left.Actions) == 0 && len(right.Actions) == 0
		default:
			return fmt.Errorf("unknown control %q", certificate.Name)
		}
		if certificate.Passed != passed || certificate.Observed != observed {
			return fmt.Errorf("control %q boolean/result was not reconstructed", certificate.Name)
		}
	}
	if len(verified.Certificates) != 18 {
		return errors.New("control suite did not execute all 18 controls")
	}
	return nil
}

func verifyRetainedControlEvidence(bundle ControlBundle, evidence ControlEvidence) error {
	if rules := causal.Rules(); len(rules) == 0 || evidence.StaticRule != rules[0].Code() {
		return errors.New("semantic-first static rule is unavailable")
	}
	for name, rows := range map[string][]causalv2.MatrixControlRow{"static": evidence.StaticMatrix, "recomputed": evidence.RecomputedMatrix} {
		for index, row := range rows {
			if row.Treatment.FailureCode != "" || row.Control.FailureCode != "" {
				return fmt.Errorf("%s matrix row %d failed", name, index)
			}
			if name == "recomputed" {
				if !semanticProjectionEqual(row.Treatment, row.Control) || row.ControlCache.Hits != 0 || row.ControlCache.Misses != 6*len(row.Control.Actions) || causalv2.CounterFromCounts(row.ControlMeterCounts).TotalWork < causalv2.CounterFromCounts(row.TreatmentMeterCounts).TotalWork {
					return fmt.Errorf("recomputed matrix row %d violates equality/work/cache predicates", index)
				}
			}
		}
	}
	certificate := bundle.Certificates[14]
	privateBytes, err := evidence.Corruption.FixtureBytes.Bytes()
	if err != nil {
		return err
	}
	privateFixture, err := causalv2.StrictDecode[PrivateFixture](privateBytes)
	if err != nil {
		return err
	}
	profileBytes, err := evidence.Corruption.ProfileBytes.Bytes()
	if err != nil {
		return err
	}
	baseline := make([][]byte, len(evidence.Corruption.BaselineArtifacts))
	for i, item := range evidence.Corruption.BaselineArtifacts {
		baseline[i], err = item.Bytes()
		if err != nil {
			return err
		}
	}
	cases := make([]causalrun.CorruptionCaseEvidence, len(evidence.Corruption.Cases))
	for i, item := range evidence.Corruption.Cases {
		cases[i] = causalrun.CorruptionCaseEvidence{Name: item.Name, MutationDescriptor: item.MutationDescriptor, MutatedBytesDigest: item.MutatedBytesDigest, RejectionCode: item.RejectionCode, MeterCounts: countsFrom64(item.MeterCounts)}
	}
	corruptionCounts, err := causalrun.VerifyCorruptionCases(mustCanonical(privateFixture.PublicFixture), profileBytes, baseline, cases)
	if err != nil {
		return err
	}
	if causalv2.CounterFromCounts(certificate.MeterCounts).ProfileFields != int64(len(evidence.Corruption.Cases)) || corruptionCounts.ProfileFields != len(evidence.Corruption.Cases) {
		return errors.New("corruption certificate does not cover every retained case")
	}
	return nil
}

func verifyFreshMatrixProofs(ctx context.Context, bundle ControlBundle, evidence ControlEvidence, fixtures []PrivateFixture, episodes []EpisodeEvidence, certificates []ApplicationCertificate) error {
	rules := causal.Rules()
	if len(rules) == 0 {
		return errors.New("causal grammar has no semantic-first rule")
	}
	for _, trial := range []struct {
		name causalrun.ControlName
		rows []causalv2.MatrixControlRow
	}{{causalrun.ControlStaticRule, evidence.StaticMatrix}, {causalrun.ControlRecomputedRule, evidence.RecomputedMatrix}} {
		certificate, rows, err := executeTrainingMatrixControl(ctx, trial.name, evidence.SelectedRule, rules[0].Code(), fixtures, episodes, certificates)
		if err != nil {
			return err
		}
		if !bytes.Equal(mustCanonical(rows), mustCanonical(trial.rows)) {
			return fmt.Errorf("%s retained rows differ from fresh exact rerun", trial.name)
		}
		for _, retained := range bundle.Certificates {
			if retained.Name == string(trial.name) && !bytes.Equal(mustCanonical(certificate), mustCanonical(retained)) {
				return fmt.Errorf("%s certificate differs from fresh exact rerun", trial.name)
			}
		}
	}
	return nil
}

func countsFrom64(values [15]int64) causalrun.Counts {
	return causalrun.Counts{SCMEvaluations: int(values[0]), PartitionAssignments: int(values[1]), CellAccumulations: int(values[2]), RuleComparisons: int(values[3]), PosteriorChecks: int(values[4]), ArtifactMaterializations: int(values[5]), TranscriptFields: int(values[6]), ProfileFields: int(values[7]), MemoStates: int(values[8]), MemoLookups: int(values[9]), QEvaluations: int(values[10]), TableLookups: int(values[11]), EngineCycles: int(values[12]), AttributedUnits: int(values[13]), TotalWork: int(values[14])}
}
