package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const zeroAuthorityDigest = "0000000000000000000000000000000000000000000000000000000000000000"

type AuthorityRef struct {
	Path   string
	Digest string
	Mode   string
}

func Reference(path string, canonical []byte) (AuthorityRef, error) {
	digest := sha256.Sum256(canonical)
	result := AuthorityRef{Path: path, Digest: hex.EncodeToString(digest[:]), Mode: "100644"}
	return result, result.Verify()
}

func (r AuthorityRef) Wire() []any { return []any{r.Path, r.Digest, r.Mode} }

func (r AuthorityRef) Verify() error {
	if r.Path == "" || len(r.Path) > 192 || strings.HasPrefix(r.Path, "/") || strings.Contains(r.Path, "\\") || !digestText(r.Digest) || r.Mode != "100644" {
		return fmt.Errorf("invalid authority reference")
	}
	for _, part := range strings.Split(r.Path, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("unsafe authority reference")
		}
		for _, character := range part {
			if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || strings.ContainsRune("._-", character)) {
				return fmt.Errorf("unsafe authority path character")
			}
		}
	}
	return nil
}

type EnvironmentRow struct {
	Key   string
	Value string
}

func environmentWires(rows []EnvironmentRow) ([]any, error) {
	result := make([]any, len(rows))
	previous := ""
	for index, row := range rows {
		if row.Key == "" || index > 0 && row.Key <= previous {
			return nil, fmt.Errorf("environment rows are not unique key ordered")
		}
		for _, character := range row.Key {
			if character > 127 || !(character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_') {
				return nil, fmt.Errorf("invalid environment key")
			}
		}
		if !isASCII(row.Value) {
			return nil, fmt.Errorf("non-ASCII environment value")
		}
		result[index] = []any{row.Key, row.Value}
		previous = row.Key
	}
	return result, nil
}

func isASCII(value string) bool {
	for _, character := range value {
		if character > 127 {
			return false
		}
	}
	return true
}

type ExecutionManifest struct {
	Role               string
	Panel              string
	Authority          string
	SourceRoot         string
	BinaryDigest       string
	Environment        []EnvironmentRow
	FixtureRoot        AuthorityRef
	RunEvidence        AuthorityRef
	StructuralMaps     []AuthorityRef
	RunIDsRoot         string
	TranscriptRowsRoot string
	ResultRowsRoot     string
	TotalRuns          int
	PriorExecution     *AuthorityRef
	Canonical          []byte
	Digest             string
}

func BuildExecutionManifest(value ExecutionManifest) (ExecutionManifest, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := executionManifestCanonical(value)
	if err != nil {
		return ExecutionManifest{}, err
	}
	value.Canonical = canonical
	value.Digest = shaHex(canonical)
	if err := VerifyExecutionManifest(value); err != nil {
		return ExecutionManifest{}, err
	}
	return value, nil
}

func VerifyExecutionManifest(value ExecutionManifest) error {
	canonical, err := executionManifestCanonical(value)
	if err != nil || len(value.Canonical) > 32768 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid execution manifest")
	}
	return nil
}

func executionManifestCanonical(value ExecutionManifest) ([]byte, error) {
	wantRuns := panelRunCounts[value.Panel]
	wantMaps := wantRuns / 44
	if !validPanelAuthority(value.Panel, value.Authority) || wantRuns == 0 || value.TotalRuns != wantRuns || len(value.StructuralMaps) != wantMaps || !digestText(value.SourceRoot) || !digestText(value.BinaryDigest) || !digestText(value.RunIDsRoot) || !digestText(value.TranscriptRowsRoot) || !digestText(value.ResultRowsRoot) || !referenceAt(value.FixtureRoot, ExpectedAuthorityPath(value.Panel, "fixture-root")) || !referenceAt(value.RunEvidence, ExpectedAuthorityPath(value.Panel, "run-evidence")) {
		return nil, fmt.Errorf("invalid execution manifest authority")
	}
	if value.Role == "primary" {
		if value.PriorExecution != nil {
			return nil, fmt.Errorf("primary execution has prior authority")
		}
	} else if value.Role != "audit" || value.PriorExecution == nil || !referenceAt(*value.PriorExecution, ExpectedAuthorityPath(value.Panel, "execution-primary")) {
		return nil, fmt.Errorf("invalid execution role")
	}
	mapWires, err := structuralMapWires(value.Panel, value.StructuralMaps)
	if err != nil {
		return nil, err
	}
	environment, err := environmentWires(value.Environment)
	if err != nil {
		return nil, err
	}
	prior := any(zeroAuthorityDigest)
	if value.PriorExecution != nil {
		prior = value.PriorExecution.Wire()
	}
	return json.Marshal([]any{"actionrelation-execution-manifest/v2", value.Role, value.Panel, value.Authority, value.SourceRoot, value.BinaryDigest, environment, value.FixtureRoot.Wire(), value.RunEvidence.Wire(), mapWires, value.RunIDsRoot, value.TranscriptRowsRoot, value.ResultRowsRoot, value.TotalRuns, prior, "completed"})
}

type AuditAttestation struct {
	Panel              string
	Authority          string
	PrimaryExecution   AuthorityRef
	AuditExecution     AuthorityRef
	RunEvidence        AuthorityRef
	StructuralMaps     []AuthorityRef
	RunIDsRoot         string
	TranscriptRowsRoot string
	ResultRowsRoot     string
	TotalRuns          int
	Canonical          []byte
	Digest             string
}

func BuildAuditAttestation(value AuditAttestation) (AuditAttestation, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := auditAttestationCanonical(value)
	if err != nil {
		return AuditAttestation{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	if err := VerifyAuditAttestation(value); err != nil {
		return AuditAttestation{}, err
	}
	return value, nil
}

func VerifyAuditAttestation(value AuditAttestation) error {
	canonical, err := auditAttestationCanonical(value)
	if err != nil || len(value.Canonical) > 8192 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid audit attestation")
	}
	return nil
}

func auditAttestationCanonical(value AuditAttestation) ([]byte, error) {
	wantRuns := panelRunCounts[value.Panel]
	if !validPanelAuthority(value.Panel, value.Authority) || wantRuns == 0 || value.TotalRuns != wantRuns || len(value.StructuralMaps) != wantRuns/44 || !referenceAt(value.PrimaryExecution, ExpectedAuthorityPath(value.Panel, "execution-primary")) || !referenceAt(value.AuditExecution, ExpectedAuthorityPath(value.Panel, "execution-audit")) || !referenceAt(value.RunEvidence, ExpectedAuthorityPath(value.Panel, "run-evidence")) || !digestText(value.RunIDsRoot) || !digestText(value.TranscriptRowsRoot) || !digestText(value.ResultRowsRoot) {
		return nil, fmt.Errorf("invalid audit attestation authority")
	}
	mapWires, err := structuralMapWires(value.Panel, value.StructuralMaps)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]any{"actionrelation-audit-attestation/v3", value.Panel, value.Authority, value.PrimaryExecution.Wire(), value.AuditExecution.Wire(), value.RunEvidence.Wire(), mapWires, value.RunIDsRoot, value.TranscriptRowsRoot, value.ResultRowsRoot, value.TotalRuns, "isolated-byte-identical"})
}

func EqualExecutionEvidence(primary, audit ExecutionManifest) error {
	if primary.Role != "primary" || audit.Role != "audit" || audit.PriorExecution == nil {
		return fmt.Errorf("invalid execution comparison roles")
	}
	primaryRef, err := Reference(audit.PriorExecution.Path, primary.Canonical)
	if err != nil || primaryRef != *audit.PriorExecution || primary.Panel != audit.Panel || primary.Authority != audit.Authority || primary.SourceRoot != audit.SourceRoot || primary.BinaryDigest != audit.BinaryDigest || !slices.Equal(primary.Environment, audit.Environment) || primary.FixtureRoot != audit.FixtureRoot || primary.RunEvidence != audit.RunEvidence || !slices.Equal(primary.StructuralMaps, audit.StructuralMaps) || primary.RunIDsRoot != audit.RunIDsRoot || primary.TranscriptRowsRoot != audit.TranscriptRowsRoot || primary.ResultRowsRoot != audit.ResultRowsRoot || primary.TotalRuns != audit.TotalRuns {
		return fmt.Errorf("primary and audit evidence differ")
	}
	return nil
}

type ExecutionCore struct {
	Panel                string
	Authority            string
	SourceRoot           string
	BinaryDigest         string
	PlanReview           AuthorityRef
	ImplementationReview AuthorityRef
	BuildAuthority       AuthorityRef
	Competence           AuthorityRef
	Environment          []EnvironmentRow
	FixtureRoot          AuthorityRef
	PrimaryExecution     AuthorityRef
	AuditExecution       AuthorityRef
	AuditAttestation     AuthorityRef
	RunEvidence          AuthorityRef
	StructuralMaps       []AuthorityRef
	RunningReceipt       *AuthorityRef
	Canonical            []byte
	Digest               string
}

func BuildExecutionCore(value ExecutionCore) (ExecutionCore, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := executionCoreCanonical(value)
	if err != nil {
		return ExecutionCore{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	if err := VerifyExecutionCore(value); err != nil {
		return ExecutionCore{}, err
	}
	return value, nil
}

func VerifyExecutionCore(value ExecutionCore) error {
	canonical, err := executionCoreCanonical(value)
	if err != nil || len(value.Canonical) > 8192 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid execution core")
	}
	return nil
}

func executionCoreCanonical(value ExecutionCore) ([]byte, error) {
	wantMaps := panelRunCounts[value.Panel] / 44
	if !validPanelAuthority(value.Panel, value.Authority) || wantMaps == 0 || len(value.StructuralMaps) != wantMaps || !digestText(value.SourceRoot) || !digestText(value.BinaryDigest) {
		return nil, fmt.Errorf("invalid execution core identity")
	}
	for _, reference := range []AuthorityRef{value.PlanReview, value.ImplementationReview, value.BuildAuthority, value.Competence, value.FixtureRoot, value.PrimaryExecution, value.AuditExecution, value.AuditAttestation, value.RunEvidence} {
		if reference.Verify() != nil {
			return nil, fmt.Errorf("invalid execution core reference")
		}
	}
	paths := []struct {
		ref  AuthorityRef
		path string
	}{
		{value.PlanReview, ReviewManifestPath("plan")}, {value.ImplementationReview, ReviewManifestPath("implementation")},
		{value.BuildAuthority, BuildAuthorityPath}, {value.Competence, "docs/actionrelations-competence-root.json"},
		{value.FixtureRoot, ExpectedAuthorityPath(value.Panel, "fixture-root")}, {value.PrimaryExecution, ExpectedAuthorityPath(value.Panel, "execution-primary")},
		{value.AuditExecution, ExpectedAuthorityPath(value.Panel, "execution-audit")}, {value.AuditAttestation, ExpectedAuthorityPath(value.Panel, "audit-attestation")},
		{value.RunEvidence, ExpectedAuthorityPath(value.Panel, "run-evidence")},
	}
	for _, item := range paths {
		if !referenceAt(item.ref, item.path) {
			return nil, fmt.Errorf("noncanonical execution core authority path")
		}
	}
	mapWires, err := structuralMapWires(value.Panel, value.StructuralMaps)
	if err != nil {
		return nil, err
	}
	running := any(zeroAuthorityDigest)
	if value.Panel == "development" {
		if value.RunningReceipt != nil {
			return nil, fmt.Errorf("development core has running receipt")
		}
	} else {
		if value.RunningReceipt == nil || !referenceAt(*value.RunningReceipt, ExpectedAuthorityPath(value.Panel, "running")) {
			return nil, fmt.Errorf("protected core lacks running receipt")
		}
		running = value.RunningReceipt.Wire()
	}
	environment, err := environmentWires(value.Environment)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]any{"actionrelation-execution-core/v3", value.Panel, value.Authority, value.SourceRoot, value.BinaryDigest, value.PlanReview.Wire(), value.ImplementationReview.Wire(), value.BuildAuthority.Wire(), value.Competence.Wire(), environment, value.FixtureRoot.Wire(), value.PrimaryExecution.Wire(), value.AuditExecution.Wire(), value.AuditAttestation.Wire(), value.RunEvidence.Wire(), mapWires, running})
}

func ExpectedAuthorityPath(panel, name string) string {
	root, err := EvidenceRoot(panel)
	if err != nil {
		return ""
	}
	switch name {
	case "fixture-root", "execution-primary", "execution-audit", "audit-attestation", "execution-core", "evidence-payload", "publication":
		return root + "/authority/" + name + ".json"
	case "run-evidence":
		return root + "/manifests/run-evidence-root.json"
	case "report", "terminal-receipt", "claim", "running":
		return ".nous/actionrelations-v1-" + panel + "-" + name + ".json"
	}
	return ""
}

func referenceAt(value AuthorityRef, path string) bool {
	return path != "" && value.Path == path && value.Verify() == nil
}

func structuralMapWires(panel string, values []AuthorityRef) ([]any, error) {
	want := panelRunCounts[panel] / 44
	if want == 0 || len(values) != want {
		return nil, fmt.Errorf("structural-map reference cardinality mismatch")
	}
	root, _ := EvidenceRoot(panel)
	result := make([]any, len(values))
	for index, reference := range values {
		path := fmt.Sprintf("%s/manifests/curriculum-%04d/structural-output-map.json", root, index)
		if !referenceAt(reference, path) {
			return nil, fmt.Errorf("invalid structural-map reference %d", index)
		}
		result[index] = reference.Wire()
	}
	return result, nil
}
