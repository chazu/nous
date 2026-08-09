package transformexp

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
)

type protectedReport struct {
	Classification string
	PayloadDigest  string
	Payload        protectedPayload
}

type protectedPayload struct {
	Panel, ImplementationCommit                                string
	Manifest                                                   json.RawMessage
	FixtureRoot, PrimaryManifest, AuditManifest, EvidenceGraph string
	Competence                                                 CompetenceReport
	CompetenceRoot                                             string
	GeneratorAcceptance, OracleAcceptance                      AcceptanceDiagnostics
	Rows                                                       []PolicyReportRow
	Inference                                                  transformInference
	Power                                                      transformPower
	Gates                                                      [12]bool
	Limitations                                                []string
}

// JSON returns the canonical protected report wire without exposing mutable
// report internals to command callers.
func (r protectedReport) JSON() ([]byte, error) { return canonicalProtectedReport(r) }

func (r protectedReport) MarshalJSON() ([]byte, error) {
	payload, err := r.Payload.wire()
	if err != nil {
		return nil, err
	}
	return json.Marshal([]any{"transform-schema-trials/v2", r.Classification, r.PayloadDigest, json.RawMessage(payload)})
}

func (p protectedPayload) wire() ([]byte, error) {
	var manifest any
	if err := json.Unmarshal(p.Manifest, &manifest); err != nil {
		return nil, err
	}
	competence := []any{"transform-competence/v1", p.Competence.Forests, p.Competence.SchemaApplications, p.Competence.ProgramApplications, p.Competence.Microcases, p.Competence.Passed, p.CompetenceRoot}
	rows := make([]any, len(p.Rows))
	for index, row := range p.Rows {
		rows[index] = []any{row.Ordinal, row.Family, row.Policy, row.Terminal, row.Work, row.Applications, row.SchemaSHA256, row.HeldoutCorrectBits, row.FalseApplications, row.NonmatchingWork}
	}
	i := p.Inference
	inference := []any{"transform-inference/v1", i.Point.Numerator, i.Point.Denominator, i.Lower.Numerator, i.Lower.Denominator, i.Upper.Numerator, i.Upper.Denominator, i.RandomizationExtreme, i.PValue.Numerator, i.PValue.Denominator, i.NousSuccesses, i.PBESuccesses, i.FalseApplications, i.NonmatchingNous, i.NonmatchingPBE}
	power := []any{"transform-power/v1", p.Power.Passing, p.Power.Replicates, p.Power.Authorized}
	return json.Marshal([]any{p.Panel, PlanCommit, p.ImplementationCommit, manifest, p.FixtureRoot, p.PrimaryManifest, p.AuditManifest, p.EvidenceGraph, competence, json.RawMessage(acceptanceDiagnosticsBytes("generator", p.GeneratorAcceptance)), json.RawMessage(acceptanceDiagnosticsBytes("oracle", p.OracleAcceptance)), rows, inference, power, p.Gates, p.Limitations})
}

func newProtectedReport(panel, implementationCommit string, evidence panelEvidence, power transformPower) (protectedReport, error) {
	competenceRoot := digestBytes(evidence.Files["competence/root.json"])
	gates := protectedGates(implementationCommit, evidence)
	payload := protectedPayload{Panel: panel, ImplementationCommit: implementationCommit, Manifest: json.RawMessage(PreregisteredManifestJSON), FixtureRoot: evidence.Report.FixtureRootDigest, PrimaryManifest: evidence.Report.PrimaryManifestDigest, AuditManifest: evidence.Report.AuditManifestDigest, EvidenceGraph: evidence.Report.EvidenceGraphDigest, Competence: evidence.Report.Competence, CompetenceRoot: competenceRoot, GeneratorAcceptance: evidence.Report.GeneratorAcceptance, OracleAcceptance: evidence.Report.OracleAcceptance, Rows: evidence.Report.Rows, Inference: evidence.Report.Inference, Power: power, Gates: gates, Limitations: []string{}}
	payloadBytes, err := payload.wire()
	if err != nil {
		return protectedReport{}, err
	}
	classification, err := protectedClassification(panel, evidence.Report.Inference, power)
	if err != nil {
		return protectedReport{}, err
	}
	for _, gate := range gates {
		if !gate {
			classification = "invalid"
			break
		}
	}
	return protectedReport{classification, digestBytes(payloadBytes), payload}, nil
}

func protectedGates(implementationCommit string, evidence panelEvidence) [12]bool {
	manifest := validateManifest() == nil && bytes.Equal(evidence.Report.Manifest, []byte(PreregisteredManifestJSON))
	transcripts, conservation, applications := evidence.Report.TranscriptHashesEqual, evidence.Report.Conservation, evidence.Report.ApplicationsExact
	for _, row := range evidence.Report.Rows {
		transcripts = transcripts && len(row.TranscriptSHA256) == 64
		conservation = conservation && row.Work > 0
		applications = applications && row.Applications >= 0 && row.Applications <= ApplicationsPerPolicy
	}
	programs := evidence.Report.ProgramsExact
	artifacts := evidence.Report.ArtifactFrozen
	scorers := 0
	for name := range evidence.Files {
		if len(name) > len("/scorer.json") && name[len(name)-len("/scorer.json"):] == "/scorer.json" {
			scorers++
		}
	}
	heldoutSealed := evidence.Report.HeldoutSealed && scorers == len(evidence.Report.Rows)/len(empiricalPolicies)
	sourceAuthority := len(implementationCommit) == 40 && len(evidence.Files["review-authority.json"]) != 0
	rebuiltGraph, graphErr := canonicalEvidenceRoot("transform-evidence-graph/v2", evidence.Report.Panel, evidence.Files)
	evidenceGraph := graphErr == nil && bytes.Equal(rebuiltGraph, evidence.EvidenceGraph) && digestBytes(rebuiltGraph) == evidence.Report.EvidenceGraphDigest
	oracleParity := evidence.Report.OracleParity && evidence.Report.GeneratorAcceptance.Exact && evidence.Report.OracleAcceptance.Exact
	return [12]bool{manifest, evidence.Report.Competence.Passed, evidence.Report.DualExecutionEqual, transcripts, conservation, oracleParity, programs, applications, artifacts, heldoutSealed, sourceAuthority, evidenceGraph}
}

func protectedClassification(panel string, inference transformInference, power transformPower) (string, error) {
	switch panel {
	case "development":
		if power.Authorized {
			return "interim-power-authorized", nil
		}
		return "interim-power-unauthorized", nil
	case "validation":
		return "interim-valid", nil
	case "locked":
		positive := inference.NousSuccesses*100 >= 80*LockedCount &&
			inference.Point.Numerator*10 >= inference.Point.Denominator &&
			inference.Lower.Numerator > 0 &&
			inference.PValue.Numerator*100 < 5*inference.PValue.Denominator &&
			inference.FalseApplications == 0 && inference.NonmatchingPBE > 0 &&
			inference.NonmatchingNous*4 <= inference.NonmatchingPBE*5
		if positive {
			return "valid-positive", nil
		}
		return "valid-null", nil
	default:
		return "", fmt.Errorf("protected report cannot name panel %q", panel)
	}
}

func canonicalProtectedReport(report protectedReport) ([]byte, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	payload, err := report.Payload.wire()
	if err != nil {
		return nil, err
	}
	if digestBytes(payload) != report.PayloadDigest {
		return nil, fmt.Errorf("protected report payload digest mismatch")
	}
	return encoded, nil
}

func decodeProtectedReport(data []byte) (protectedReport, error) {
	var outer []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&outer); err != nil || len(outer) != 4 || decoder.Decode(new(any)) == nil {
		return protectedReport{}, fmt.Errorf("invalid protected report envelope")
	}
	var version, classification, payloadDigest string
	if json.Unmarshal(outer[0], &version) != nil || version != "transform-schema-trials/v2" || json.Unmarshal(outer[1], &classification) != nil || json.Unmarshal(outer[2], &payloadDigest) != nil {
		return protectedReport{}, fmt.Errorf("invalid protected report identity")
	}
	var values []json.RawMessage
	if json.Unmarshal(outer[3], &values) != nil || len(values) != 16 {
		return protectedReport{}, fmt.Errorf("invalid protected report payload")
	}
	var panel, plan, implementation, fixture, primary, audit, graph string
	if json.Unmarshal(values[0], &panel) != nil || json.Unmarshal(values[1], &plan) != nil || plan != PlanCommit || json.Unmarshal(values[2], &implementation) != nil || json.Unmarshal(values[4], &fixture) != nil || json.Unmarshal(values[5], &primary) != nil || json.Unmarshal(values[6], &audit) != nil || json.Unmarshal(values[7], &graph) != nil {
		return protectedReport{}, fmt.Errorf("invalid protected report provenance")
	}
	var competenceWire []json.RawMessage
	if json.Unmarshal(values[8], &competenceWire) != nil || len(competenceWire) != 7 {
		return protectedReport{}, fmt.Errorf("invalid competence wire")
	}
	var competenceVersion, competenceRoot string
	var competence CompetenceReport
	if json.Unmarshal(competenceWire[0], &competenceVersion) != nil || competenceVersion != "transform-competence/v1" || json.Unmarshal(competenceWire[1], &competence.Forests) != nil || json.Unmarshal(competenceWire[2], &competence.SchemaApplications) != nil || json.Unmarshal(competenceWire[3], &competence.ProgramApplications) != nil || json.Unmarshal(competenceWire[4], &competence.Microcases) != nil || json.Unmarshal(competenceWire[5], &competence.Passed) != nil || json.Unmarshal(competenceWire[6], &competenceRoot) != nil {
		return protectedReport{}, fmt.Errorf("invalid competence values")
	}
	if competence != (CompetenceReport{351, 25272, 7020, 14, true}) || !isLowerHex(competenceRoot, 64) {
		return protectedReport{}, fmt.Errorf("competence claim does not match frozen suite")
	}
	decodeAcceptance := func(raw json.RawMessage, role string) (AcceptanceDiagnostics, error) {
		var wire []json.RawMessage
		var version, gotRole string
		var value AcceptanceDiagnostics
		if json.Unmarshal(raw, &wire) != nil || len(wire) != 7 || json.Unmarshal(wire[0], &version) != nil || version != "transform-acceptance-diagnostics/v1" || json.Unmarshal(wire[1], &gotRole) != nil || gotRole != role || json.Unmarshal(wire[2], &value.Curricula) != nil || json.Unmarshal(wire[3], &value.Applications) != nil || json.Unmarshal(wire[4], &value.Work) != nil || json.Unmarshal(wire[5], &value.RootSHA256) != nil || json.Unmarshal(wire[6], &value.Exact) != nil || !digestString(value.RootSHA256) {
			return AcceptanceDiagnostics{}, fmt.Errorf("invalid %s acceptance diagnostics", role)
		}
		return value, nil
	}
	generatorAcceptance, generatorErr := decodeAcceptance(values[9], "generator")
	oracleAcceptance, oracleErr := decodeAcceptance(values[10], "oracle")
	if generatorErr != nil || oracleErr != nil {
		return protectedReport{}, fmt.Errorf("invalid acceptance diagnostics")
	}
	var rowWires [][]json.RawMessage
	if json.Unmarshal(values[11], &rowWires) != nil {
		return protectedReport{}, fmt.Errorf("invalid policy rows")
	}
	rows := make([]PolicyReportRow, len(rowWires))
	for index, wire := range rowWires {
		if len(wire) != 10 || json.Unmarshal(wire[0], &rows[index].Ordinal) != nil || json.Unmarshal(wire[1], &rows[index].Family) != nil || json.Unmarshal(wire[2], &rows[index].Policy) != nil || json.Unmarshal(wire[3], &rows[index].Terminal) != nil || json.Unmarshal(wire[4], &rows[index].Work) != nil || json.Unmarshal(wire[5], &rows[index].Applications) != nil || json.Unmarshal(wire[6], &rows[index].SchemaSHA256) != nil || json.Unmarshal(wire[7], &rows[index].HeldoutCorrectBits) != nil || json.Unmarshal(wire[8], &rows[index].FalseApplications) != nil || json.Unmarshal(wire[9], &rows[index].NonmatchingWork) != nil {
			return protectedReport{}, fmt.Errorf("invalid policy row %d", index)
		}
		decodedBits, decodeErr := hex.DecodeString(rows[index].HeldoutCorrectBits)
		if decodeErr != nil || len(decodedBits) != 1 {
			return protectedReport{}, fmt.Errorf("invalid heldout bit vector")
		}
		rows[index].HeldoutCorrect = bits.OnesCount8(decodedBits[0])
	}
	var inferenceWire []json.RawMessage
	if json.Unmarshal(values[12], &inferenceWire) != nil || len(inferenceWire) != 15 {
		return protectedReport{}, fmt.Errorf("invalid inference wire")
	}
	var inferenceVersion string
	i := transformInference{}
	targets := []any{&inferenceVersion, &i.Point.Numerator, &i.Point.Denominator, &i.Lower.Numerator, &i.Lower.Denominator, &i.Upper.Numerator, &i.Upper.Denominator, &i.RandomizationExtreme, &i.PValue.Numerator, &i.PValue.Denominator, &i.NousSuccesses, &i.PBESuccesses, &i.FalseApplications, &i.NonmatchingNous, &i.NonmatchingPBE}
	for index := range targets {
		if json.Unmarshal(inferenceWire[index], targets[index]) != nil {
			return protectedReport{}, fmt.Errorf("invalid inference value")
		}
	}
	if inferenceVersion != "transform-inference/v1" {
		return protectedReport{}, fmt.Errorf("invalid inference version")
	}
	var powerWire []json.RawMessage
	var powerVersion string
	power := transformPower{}
	if json.Unmarshal(values[13], &powerWire) != nil || len(powerWire) != 4 || json.Unmarshal(powerWire[0], &powerVersion) != nil || powerVersion != "transform-power/v1" || json.Unmarshal(powerWire[1], &power.Passing) != nil || json.Unmarshal(powerWire[2], &power.Replicates) != nil || json.Unmarshal(powerWire[3], &power.Authorized) != nil {
		return protectedReport{}, fmt.Errorf("invalid power wire")
	}
	var gates [12]bool
	var limitations []string
	if json.Unmarshal(values[14], &gates) != nil || json.Unmarshal(values[15], &limitations) != nil {
		return protectedReport{}, fmt.Errorf("invalid report gates")
	}
	payload := protectedPayload{Panel: panel, ImplementationCommit: implementation, Manifest: append(json.RawMessage(nil), values[3]...), FixtureRoot: fixture, PrimaryManifest: primary, AuditManifest: audit, EvidenceGraph: graph, Competence: competence, CompetenceRoot: competenceRoot, GeneratorAcceptance: generatorAcceptance, OracleAcceptance: oracleAcceptance, Rows: rows, Inference: i, Power: power, Gates: gates, Limitations: limitations}
	report := protectedReport{classification, payloadDigest, payload}
	canonical, err := canonicalProtectedReport(report)
	if err != nil || !bytes.Equal(canonical, data) {
		return protectedReport{}, fmt.Errorf("protected report is not canonical")
	}
	return report, nil
}
