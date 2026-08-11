package actionrelationscore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationwire"
)

const MaximumReportBytes = 14 * 1024 * 1024

type AuthorityRef = actionrelationexp.AuthorityRef

type MechanicalGates struct {
	AuthorityClosure       bool
	PrimaryAuditEqual      bool
	SemanticAgreement      bool
	WorkConservation       bool
	ArtifactsImmutable     bool
	NousZeroFalseMatches   bool
	RequiredBehaviorEqual  bool
	FreshCertificatesValid bool
}

func (g MechanicalGates) Wire() []any {
	return []any{"actionrelation-mechanical-gates/v1", g.AuthorityClosure, g.PrimaryAuditEqual, g.SemanticAgreement, g.WorkConservation, g.ArtifactsImmutable, g.NousZeroFalseMatches, g.RequiredBehaviorEqual, g.FreshCertificatesValid}
}

func (g MechanicalGates) All() bool {
	return g.AuthorityClosure && g.PrimaryAuditEqual && g.SemanticAgreement && g.WorkConservation && g.ArtifactsImmutable && g.NousZeroFalseMatches && g.RequiredBehaviorEqual && g.FreshCertificatesValid
}

type ReportAuthority struct {
	PlanReview           AuthorityRef
	ImplementationReview AuthorityRef
	BuildAuthority       AuthorityRef
	Competence           AuthorityRef
	FixtureRoot          AuthorityRef
	RunningReceipt       *AuthorityRef
	CurriculumRowsRoot   string
	EvidencePayload      AuthorityRef
}

type Report struct {
	Panel           string
	Authority       string
	ManifestDigest  string
	Refs            ReportAuthority
	MechanicalGates MechanicalGates
	Inference       Inference
	Classification  string
	Canonical       []byte
	Digest          string
}

func ParseReport(data []byte) (Report, error) {
	var fields []json.RawMessage
	if len(data) > MaximumReportBytes || json.Unmarshal(data, &fields) != nil || len(fields) != 21 {
		return Report{}, fmt.Errorf("invalid report wire")
	}
	var version string
	value := Report{Canonical: bytes.Clone(data), Digest: digest(data)}
	if json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-report/v3" || json.Unmarshal(fields[1], &value.Panel) != nil || json.Unmarshal(fields[2], &value.Authority) != nil || json.Unmarshal(fields[3], &value.ManifestDigest) != nil || json.Unmarshal(fields[10], &value.Refs.CurriculumRowsRoot) != nil || json.Unmarshal(fields[19], &value.Classification) != nil {
		return Report{}, fmt.Errorf("invalid report fields")
	}
	var err error
	refs := []*AuthorityRef{&value.Refs.PlanReview, &value.Refs.ImplementationReview, &value.Refs.BuildAuthority, &value.Refs.Competence, &value.Refs.FixtureRoot}
	for index, target := range refs {
		*target, err = parseReportRef(fields[4+index])
		if err != nil {
			return Report{}, err
		}
	}
	if bytes.Equal(fields[9], []byte(`"`+zeroDigest+`"`)) {
		value.Refs.RunningReceipt = nil
	} else {
		running, parseErr := parseReportRef(fields[9])
		if parseErr != nil {
			return Report{}, parseErr
		}
		value.Refs.RunningReceipt = &running
	}
	if value.MechanicalGates, err = parseMechanicalGates(fields[11]); err != nil {
		return Report{}, err
	}
	if value.Inference.PrimarySearchRatio, err = parseFraction(fields[12]); err != nil {
		return Report{}, err
	}
	if value.Inference.LifecycleRatio, err = parseFraction(fields[13]); err != nil {
		return Report{}, err
	}
	if value.Inference.AmortizationRows, err = parseAmortizationRows(fields[14]); err != nil {
		return Report{}, err
	}
	var interval []json.RawMessage
	if json.Unmarshal(fields[15], &interval) != nil || len(interval) != 2 {
		return Report{}, fmt.Errorf("invalid report confidence interval")
	}
	for index := range interval {
		value.Inference.ConfidenceInterval[index], err = parseFraction(interval[index])
		if err != nil {
			return Report{}, err
		}
	}
	if value.Inference.RandomizationP, err = parseFraction(fields[16]); err != nil {
		return Report{}, err
	}
	value.Inference.RandomizationExtreme = value.Inference.RandomizationP.Numerator - 1
	if value.Inference.SavingCoverage, err = parseFraction(fields[17]); err != nil {
		return Report{}, err
	}
	if value.Inference.Power, err = parseFraction(fields[18]); err != nil {
		return Report{}, err
	}
	if value.Panel == "development" {
		value.Inference.PowerSuccesses = value.Inference.Power.Numerator
	}
	value.Refs.EvidencePayload, err = parseReportRef(fields[20])
	if err != nil || VerifyReport(value) != nil {
		return Report{}, fmt.Errorf("report does not reconstruct")
	}
	return value, nil
}

func CurriculumPolicyRowsRoot(rows []CurriculumPolicyRow) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("empty curriculum row root")
	}
	digests := make([]string, len(rows))
	previousCurriculum, previousPolicy := -1, -1
	for index, row := range rows {
		policy := policyIndex(row.Policy)
		if VerifyCurriculumPolicyRow(row) != nil || policy < 0 || index > 0 && (row.Curriculum < previousCurriculum || row.Curriculum == previousCurriculum && policy <= previousPolicy) {
			return "", fmt.Errorf("curriculum rows are not in canonical order")
		}
		digests[index] = row.Digest
		previousCurriculum, previousPolicy = row.Curriculum, policy
	}
	return actionrelationwire.RootDigest("curriculum-policy-rows", digests)
}

func WorldPolicyRowsRoot(rows []WorldPolicyRow) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("empty world row root")
	}
	digests := make([]string, len(rows))
	previousCurriculum, previousWorld, previousPolicy := -1, -1, -1
	for index, row := range rows {
		policy := policyIndex(row.Policy)
		if VerifyWorldPolicyRow(row) != nil || policy < 0 || index > 0 && (row.Curriculum < previousCurriculum || row.Curriculum == previousCurriculum && (row.WorldOrdinal < previousWorld || row.WorldOrdinal == previousWorld && policy <= previousPolicy)) {
			return "", fmt.Errorf("world rows are not in canonical order")
		}
		digests[index] = row.Digest
		previousCurriculum, previousWorld, previousPolicy = row.Curriculum, row.WorldOrdinal, policy
	}
	return actionrelationwire.RootDigest("world-policy-rows", digests)
}

func BuildReport(panel, authority string, refs ReportAuthority, gates MechanicalGates, inference Inference) (Report, error) {
	manifestDigest := digest([]byte(actionrelationexp.PreregisteredManifestJSON))
	classification, err := reportClassification(panel, gates, inference)
	if err != nil {
		return Report{}, err
	}
	report := Report{Panel: panel, Authority: authority, ManifestDigest: manifestDigest, Refs: refs, MechanicalGates: gates, Inference: inference, Classification: classification}
	running := any(zeroDigest)
	if refs.RunningReceipt != nil {
		running = refs.RunningReceipt.Wire()
	}
	amortization := make([]any, len(inference.AmortizationRows))
	for index, row := range inference.AmortizationRows {
		amortization[index] = row.wire()
	}
	report.Canonical, _ = json.Marshal([]any{
		"actionrelation-report/v3", panel, authority, manifestDigest, refs.PlanReview.Wire(), refs.ImplementationReview.Wire(),
		refs.BuildAuthority.Wire(), refs.Competence.Wire(), refs.FixtureRoot.Wire(), running, refs.CurriculumRowsRoot,
		gates.Wire(), inference.PrimarySearchRatio.Wire(), inference.LifecycleRatio.Wire(), amortization,
		[]any{inference.ConfidenceInterval[0].Wire(), inference.ConfidenceInterval[1].Wire()}, inference.RandomizationP.Wire(),
		inference.SavingCoverage.Wire(), inference.Power.Wire(), classification, refs.EvidencePayload.Wire(),
	})
	report.Digest = digest(report.Canonical)
	if err := VerifyReport(report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func VerifyReport(report Report) error {
	if report.Panel != "development" && report.Panel != "validation" && report.Panel != "locked" || !digestText(report.ManifestDigest) || report.ManifestDigest != digest([]byte(actionrelationexp.PreregisteredManifestJSON)) || len(report.Canonical) > MaximumReportBytes || report.Digest != digest(report.Canonical) {
		return fmt.Errorf("invalid report identity")
	}
	for _, ref := range []AuthorityRef{report.Refs.PlanReview, report.Refs.ImplementationReview, report.Refs.BuildAuthority, report.Refs.Competence, report.Refs.FixtureRoot, report.Refs.EvidencePayload} {
		if err := ref.Verify(); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		ref  AuthorityRef
		path string
	}{
		{report.Refs.PlanReview, actionrelationexp.ReviewManifestPath("plan")},
		{report.Refs.ImplementationReview, actionrelationexp.ReviewManifestPath("implementation")},
		{report.Refs.BuildAuthority, actionrelationexp.BuildAuthorityPath},
		{report.Refs.Competence, "docs/actionrelations-competence-root.json"},
		{report.Refs.FixtureRoot, actionrelationexp.ExpectedAuthorityPath(report.Panel, "fixture-root")},
		{report.Refs.EvidencePayload, actionrelationexp.ExpectedAuthorityPath(report.Panel, "evidence-payload")},
	} {
		if item.ref.Path != item.path {
			return fmt.Errorf("noncanonical report authority path")
		}
	}
	if !digestText(report.Refs.CurriculumRowsRoot) {
		return fmt.Errorf("invalid curriculum-policy rows root")
	}
	if report.Panel == "development" {
		if report.Refs.RunningReceipt != nil {
			return fmt.Errorf("development report has a running receipt")
		}
	} else if report.Refs.RunningReceipt == nil || report.Refs.RunningReceipt.Verify() != nil || report.Refs.RunningReceipt.Path != actionrelationexp.ExpectedAuthorityPath(report.Panel, "running") {
		return fmt.Errorf("protected report lacks a running receipt")
	}
	classification, err := reportClassification(report.Panel, report.MechanicalGates, report.Inference)
	if err != nil || classification != report.Classification {
		return fmt.Errorf("report classification does not reconstruct")
	}
	rebuilt, err := BuildReportCanonical(report)
	if err != nil || !bytes.Equal(rebuilt, report.Canonical) {
		return fmt.Errorf("report wire does not reconstruct")
	}
	return nil
}

func BuildReportCanonical(report Report) ([]byte, error) {
	running := any(zeroDigest)
	if report.Refs.RunningReceipt != nil {
		running = report.Refs.RunningReceipt.Wire()
	}
	amortization := make([]any, len(report.Inference.AmortizationRows))
	for index, row := range report.Inference.AmortizationRows {
		amortization[index] = row.wire()
	}
	return json.Marshal([]any{
		"actionrelation-report/v3", report.Panel, report.Authority, report.ManifestDigest,
		report.Refs.PlanReview.Wire(), report.Refs.ImplementationReview.Wire(), report.Refs.BuildAuthority.Wire(), report.Refs.Competence.Wire(),
		report.Refs.FixtureRoot.Wire(), running, report.Refs.CurriculumRowsRoot, report.MechanicalGates.Wire(),
		report.Inference.PrimarySearchRatio.Wire(), report.Inference.LifecycleRatio.Wire(), amortization,
		[]any{report.Inference.ConfidenceInterval[0].Wire(), report.Inference.ConfidenceInterval[1].Wire()}, report.Inference.RandomizationP.Wire(),
		report.Inference.SavingCoverage.Wire(), report.Inference.Power.Wire(), report.Classification, report.Refs.EvidencePayload.Wire(),
	})
}

func reportClassification(panel string, gates MechanicalGates, inference Inference) (string, error) {
	if panel != "development" && panel != "validation" && panel != "locked" {
		return "", fmt.Errorf("invalid report panel")
	}
	if err := verifyReportInference(panel, inference); err != nil {
		return "", fmt.Errorf("invalid report inference")
	}
	if !gates.All() {
		return "invalid", nil
	}
	switch panel {
	case "development":
		if slices.ContainsFunc(inference.AmortizationRows, func(row AmortizationRow) bool { return row.Status == "incomplete" }) {
			return "interim-power-unauthorized", nil
		}
		if compareFraction(inference.Power, Fraction{80, 100}) >= 0 {
			return "interim-power-authorized", nil
		}
		return "interim-power-unauthorized", nil
	case "validation":
		return "interim-valid", nil
	case "locked":
		if slices.ContainsFunc(inference.AmortizationRows, func(row AmortizationRow) bool { return row.Status == "incomplete" }) {
			return "valid-null", nil
		}
		if compareFraction(inference.PrimarySearchRatio, Fraction{85, 100}) <= 0 && compareFraction(inference.ConfidenceInterval[1], Fraction{1, 1}) < 0 && compareFraction(inference.SavingCoverage, Fraction{80, 100}) >= 0 && compareFraction(inference.RandomizationP, Fraction{5, 100}) < 0 {
			return "valid-positive", nil
		}
		return "valid-null", nil
	}
	panic("unreachable")
}

func validFraction(value Fraction) bool {
	return value.Numerator >= 0 && value.Denominator > 0
}

func verifyReportInference(panel string, inference Inference) error {
	fractions := []Fraction{inference.PrimarySearchRatio, inference.LifecycleRatio, inference.ConfidenceInterval[0], inference.ConfidenceInterval[1], inference.RandomizationP, inference.SavingCoverage, inference.Power}
	for _, value := range fractions {
		if !validFraction(value) {
			return fmt.Errorf("invalid fraction")
		}
	}
	if compareFraction(inference.ConfidenceInterval[0], inference.ConfidenceInterval[1]) > 0 || compareFraction(inference.RandomizationP, Fraction{1, 1}) > 0 || compareFraction(inference.SavingCoverage, Fraction{1, 1}) > 0 || compareFraction(inference.Power, Fraction{1, 1}) > 0 || inference.RandomizationP.Denominator != InferenceReplicates+1 || inference.RandomizationP.Numerator != inference.RandomizationExtreme+1 {
		return fmt.Errorf("invalid bounded inference")
	}
	want := map[string]int{"development": 16, "validation": 24, "locked": 32}[panel]
	if len(inference.AmortizationRows) != want {
		return fmt.Errorf("invalid amortization cardinality")
	}
	nousSearch, dynamicSearch, nousLifecycle := 0, 0, 0
	savings := 0
	incomplete := false
	for curriculum, row := range inference.AmortizationRows {
		if row.Panel != panel || row.Curriculum != curriculum || row.Family != curriculum%8 || row.Acquisition < 0 || row.DynamicSearch < 0 || row.NousSearch < 0 || row.Status != "complete" && row.Status != "incomplete" {
			return fmt.Errorf("invalid amortization row")
		}
		difference := row.DynamicSearch - row.NousSearch
		incomplete = incomplete || row.Status == "incomplete"
		if row.Status == "incomplete" {
			if !row.Infinite || row.Batches != 0 {
				return fmt.Errorf("invalid incomplete amortization")
			}
		} else if row.DynamicSearch <= 0 {
			return fmt.Errorf("complete amortization has zero dynamic search")
		} else if difference <= 0 {
			if !row.Infinite || row.Batches != 0 {
				return fmt.Errorf("invalid infinite amortization")
			}
		} else {
			if row.Infinite || row.Batches != (row.Acquisition+difference-1)/difference {
				return fmt.Errorf("invalid finite amortization")
			}
			savings++
		}
		nousSearch += row.NousSearch
		dynamicSearch += row.DynamicSearch
		nousLifecycle += row.NousSearch + row.Acquisition
	}
	if incomplete {
		wantPower := Fraction{0, 1}
		if panel == "development" {
			wantPower = Fraction{0, PowerOuterReplicates}
		}
		if inference.PrimarySearchRatio != (Fraction{0, 1}) || inference.LifecycleRatio != (Fraction{0, 1}) || inference.ConfidenceInterval != ([2]Fraction{{0, 1}, {0, 1}}) || inference.RandomizationP != (Fraction{InferenceReplicates + 1, InferenceReplicates + 1}) || inference.RandomizationExtreme != InferenceReplicates || inference.SavingCoverage != (Fraction{0, want}) || inference.Power != wantPower {
			return fmt.Errorf("incomplete inference sentinel changed")
		}
		return nil
	}
	if inference.PrimarySearchRatio != (Fraction{nousSearch, dynamicSearch}) || inference.LifecycleRatio != (Fraction{nousLifecycle, dynamicSearch}) || inference.SavingCoverage != (Fraction{savings, want}) {
		return fmt.Errorf("aggregate inference does not reconstruct")
	}
	if panel == "development" && (inference.Power.Denominator != PowerOuterReplicates || inference.Power.Numerator != inference.PowerSuccesses) {
		return fmt.Errorf("development power does not reconstruct")
	}
	return nil
}

func parseReportRef(data json.RawMessage) (AuthorityRef, error) {
	var fields []json.RawMessage
	var value AuthorityRef
	if json.Unmarshal(data, &fields) != nil || len(fields) != 3 || json.Unmarshal(fields[0], &value.Path) != nil || json.Unmarshal(fields[1], &value.Digest) != nil || json.Unmarshal(fields[2], &value.Mode) != nil || value.Verify() != nil {
		return AuthorityRef{}, fmt.Errorf("invalid report authority reference")
	}
	return value, nil
}

func parseMechanicalGates(data json.RawMessage) (MechanicalGates, error) {
	var fields []json.RawMessage
	var version string
	var values [8]bool
	if json.Unmarshal(data, &fields) != nil || len(fields) != 9 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-mechanical-gates/v1" {
		return MechanicalGates{}, fmt.Errorf("invalid mechanical gates wire")
	}
	for index := range values {
		if json.Unmarshal(fields[index+1], &values[index]) != nil {
			return MechanicalGates{}, fmt.Errorf("invalid mechanical gate")
		}
	}
	return MechanicalGates{values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7]}, nil
}

func parseFraction(data json.RawMessage) (Fraction, error) {
	var values []int
	if json.Unmarshal(data, &values) != nil || len(values) != 2 {
		return Fraction{}, fmt.Errorf("invalid fraction wire")
	}
	return Fraction{values[0], values[1]}, nil
}

func parseAmortizationRows(data json.RawMessage) ([]AmortizationRow, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid amortization rows wire")
	}
	rows := make([]AmortizationRow, len(wires))
	for index, wire := range wires {
		var version string
		var saving int
		if len(wire) != 10 || json.Unmarshal(wire[0], &version) != nil || version != "actionrelation-amortization/v1" || json.Unmarshal(wire[1], &rows[index].Panel) != nil || json.Unmarshal(wire[2], &rows[index].Curriculum) != nil || json.Unmarshal(wire[3], &rows[index].Family) != nil || json.Unmarshal(wire[4], &rows[index].Acquisition) != nil || json.Unmarshal(wire[5], &rows[index].NousSearch) != nil || json.Unmarshal(wire[6], &rows[index].DynamicSearch) != nil || json.Unmarshal(wire[7], &saving) != nil || json.Unmarshal(wire[9], &rows[index].Status) != nil || saving != rows[index].DynamicSearch-rows[index].NousSearch {
			return nil, fmt.Errorf("invalid amortization row %d", index)
		}
		if bytes.Equal(wire[8], []byte(`"infinite"`)) {
			rows[index].Infinite = true
		} else if json.Unmarshal(wire[8], &rows[index].Batches) != nil {
			return nil, fmt.Errorf("invalid amortization batches")
		}
	}
	return rows, nil
}
