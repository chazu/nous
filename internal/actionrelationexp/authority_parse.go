package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func ParseExecutionManifest(data []byte) (ExecutionManifest, error) {
	var fields []json.RawMessage
	var version, terminal string
	value := ExecutionManifest{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if len(data) > 32768 || json.Unmarshal(data, &fields) != nil || len(fields) != 16 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-execution-manifest/v2" || json.Unmarshal(fields[1], &value.Role) != nil || json.Unmarshal(fields[2], &value.Panel) != nil || json.Unmarshal(fields[3], &value.Authority) != nil || json.Unmarshal(fields[4], &value.SourceRoot) != nil || json.Unmarshal(fields[5], &value.BinaryDigest) != nil || json.Unmarshal(fields[10], &value.RunIDsRoot) != nil || json.Unmarshal(fields[11], &value.TranscriptRowsRoot) != nil || json.Unmarshal(fields[12], &value.ResultRowsRoot) != nil || json.Unmarshal(fields[13], &value.TotalRuns) != nil || json.Unmarshal(fields[15], &terminal) != nil || terminal != "completed" {
		return ExecutionManifest{}, fmt.Errorf("invalid execution manifest wire")
	}
	var err error
	if value.Environment, err = parseEnvironmentRows(fields[6]); err != nil {
		return ExecutionManifest{}, err
	}
	if value.FixtureRoot, err = parseAuthorityRef(fields[7]); err != nil {
		return ExecutionManifest{}, err
	}
	if value.RunEvidence, err = parseAuthorityRef(fields[8]); err != nil {
		return ExecutionManifest{}, err
	}
	if value.StructuralMaps, err = parseAuthorityRefs(fields[9]); err != nil {
		return ExecutionManifest{}, err
	}
	if !bytes.Equal(fields[14], []byte(`"`+zeroAuthorityDigest+`"`)) {
		prior, parseErr := parseAuthorityRef(fields[14])
		if parseErr != nil {
			return ExecutionManifest{}, parseErr
		}
		value.PriorExecution = &prior
	}
	if VerifyExecutionManifest(value) != nil {
		return ExecutionManifest{}, fmt.Errorf("execution manifest does not reconstruct")
	}
	return value, nil
}

func ParseAuditAttestation(data []byte) (AuditAttestation, error) {
	var fields []json.RawMessage
	var version, terminal string
	value := AuditAttestation{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if len(data) > 8192 || json.Unmarshal(data, &fields) != nil || len(fields) != 12 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-audit-attestation/v3" || json.Unmarshal(fields[1], &value.Panel) != nil || json.Unmarshal(fields[2], &value.Authority) != nil || json.Unmarshal(fields[7], &value.RunIDsRoot) != nil || json.Unmarshal(fields[8], &value.TranscriptRowsRoot) != nil || json.Unmarshal(fields[9], &value.ResultRowsRoot) != nil || json.Unmarshal(fields[10], &value.TotalRuns) != nil || json.Unmarshal(fields[11], &terminal) != nil || terminal != "isolated-byte-identical" {
		return AuditAttestation{}, fmt.Errorf("invalid audit attestation wire")
	}
	var err error
	refs := []*AuthorityRef{&value.PrimaryExecution, &value.AuditExecution, &value.RunEvidence}
	for index, target := range refs {
		*target, err = parseAuthorityRef(fields[index+3])
		if err != nil {
			return AuditAttestation{}, err
		}
	}
	if value.StructuralMaps, err = parseAuthorityRefs(fields[6]); err != nil {
		return AuditAttestation{}, err
	}
	if VerifyAuditAttestation(value) != nil {
		return AuditAttestation{}, fmt.Errorf("audit attestation does not reconstruct")
	}
	return value, nil
}

func ParseExecutionCore(data []byte) (ExecutionCore, error) {
	var fields []json.RawMessage
	var version string
	value := ExecutionCore{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if len(data) > 8192 || json.Unmarshal(data, &fields) != nil || len(fields) != 17 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-execution-core/v3" || json.Unmarshal(fields[1], &value.Panel) != nil || json.Unmarshal(fields[2], &value.Authority) != nil || json.Unmarshal(fields[3], &value.SourceRoot) != nil || json.Unmarshal(fields[4], &value.BinaryDigest) != nil {
		return ExecutionCore{}, fmt.Errorf("invalid execution core wire")
	}
	var err error
	refs := []*AuthorityRef{
		&value.PlanReview, &value.ImplementationReview, &value.BuildAuthority, &value.Competence,
		&value.FixtureRoot, &value.PrimaryExecution, &value.AuditExecution, &value.AuditAttestation, &value.RunEvidence,
	}
	indices := []int{5, 6, 7, 8, 10, 11, 12, 13, 14}
	for index, target := range refs {
		*target, err = parseAuthorityRef(fields[indices[index]])
		if err != nil {
			return ExecutionCore{}, err
		}
	}
	if value.Environment, err = parseEnvironmentRows(fields[9]); err != nil {
		return ExecutionCore{}, err
	}
	if value.StructuralMaps, err = parseAuthorityRefs(fields[15]); err != nil {
		return ExecutionCore{}, err
	}
	if !bytes.Equal(fields[16], []byte(`"`+zeroAuthorityDigest+`"`)) {
		running, parseErr := parseAuthorityRef(fields[16])
		if parseErr != nil {
			return ExecutionCore{}, parseErr
		}
		value.RunningReceipt = &running
	}
	if VerifyExecutionCore(value) != nil {
		return ExecutionCore{}, fmt.Errorf("execution core does not reconstruct")
	}
	return value, nil
}

func ParseEvidencePayload(panel, authority string, data []byte) (EvidencePayload, error) {
	var fields []json.RawMessage
	var version string
	value := EvidencePayload{Panel: panel, Authority: authority, Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if len(data) > 2*1024*1024 || json.Unmarshal(data, &fields) != nil || len(fields) != 19 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-evidence-payload/v3" || json.Unmarshal(fields[17], &value.WorldPolicyRowsRoot) != nil || json.Unmarshal(fields[18], &value.CurriculumRowsRoot) != nil {
		return EvidencePayload{}, fmt.Errorf("invalid evidence payload wire")
	}
	var err error
	refs := []*AuthorityRef{&value.FixtureRoot, &value.ExecutionCore, &value.PlanReview, &value.ImplementationReview, &value.BuildAuthority, &value.Competence, &value.AuditAttestation, &value.RunEvidence}
	for index, target := range refs {
		*target, err = parseAuthorityRef(fields[index+1])
		if err != nil {
			return EvidencePayload{}, err
		}
	}
	if value.StructuralMaps, err = parseAuthorityRefs(fields[9]); err != nil {
		return EvidencePayload{}, err
	}
	if value.StoreBoundaries, err = parseBoundaries(fields[10]); err != nil {
		return EvidencePayload{}, err
	}
	if value.ObjectPackRoots, err = parseObjectManifestRefs(fields[11]); err != nil {
		return EvidencePayload{}, err
	}
	if value.JournalPackRoots, err = parseTranscriptManifestRefs(fields[12]); err != nil {
		return EvidencePayload{}, err
	}
	if value.InputPackRoots, err = parseTranscriptManifestRefs(fields[13]); err != nil {
		return EvidencePayload{}, err
	}
	if value.DetailPackRoots, err = parseTranscriptManifestRefs(fields[14]); err != nil {
		return EvidencePayload{}, err
	}
	if value.AcquisitionTables, err = parseTableManifestRefs(fields[15]); err != nil {
		return EvidencePayload{}, err
	}
	if value.IndexRoots, err = parseObjectManifestRefs(fields[16]); err != nil {
		return EvidencePayload{}, err
	}
	if VerifyEvidencePayload(value) != nil {
		return EvidencePayload{}, fmt.Errorf("evidence payload does not reconstruct")
	}
	return value, nil
}

func parseAuthorityRefs(data json.RawMessage) ([]AuthorityRef, error) {
	var wires []json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid authority reference rows")
	}
	values := make([]AuthorityRef, len(wires))
	for index, wire := range wires {
		value, err := parseAuthorityRef(wire)
		if err != nil {
			return nil, err
		}
		values[index] = value
	}
	return values, nil
}

func parseBoundaries(data json.RawMessage) ([]StoreBoundaryRow, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid boundary rows")
	}
	values := make([]StoreBoundaryRow, len(wires))
	for index, wire := range wires {
		if len(wire) != 4 || json.Unmarshal(wire[0], &values[index].Curriculum) != nil || json.Unmarshal(wire[1], &values[index].Scope) != nil || json.Unmarshal(wire[2], &values[index].BoundaryDigest) != nil || json.Unmarshal(wire[3], &values[index].PreboundaryIndexRoot) != nil {
			return nil, fmt.Errorf("invalid boundary row")
		}
	}
	return values, nil
}

func parseObjectManifestRefs(data json.RawMessage) ([]ObjectManifestRef, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid object manifest rows")
	}
	values := make([]ObjectManifestRef, len(wires))
	for index, wire := range wires {
		var scope []json.RawMessage
		var scopeKind string
		if len(wire) != 3 || json.Unmarshal(wire[0], &scope) != nil || len(scope) != 3 || json.Unmarshal(scope[0], &scopeKind) != nil || scopeKind != "curriculum" || json.Unmarshal(scope[1], &values[index].Scope.Curriculum) != nil || json.Unmarshal(scope[2], &values[index].Scope.Class) != nil || json.Unmarshal(wire[1], &values[index].Path) != nil || json.Unmarshal(wire[2], &values[index].Digest) != nil {
			return nil, fmt.Errorf("invalid object manifest row")
		}
	}
	return values, nil
}

func parseTranscriptManifestRefs(data json.RawMessage) ([]TranscriptManifestRef, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid transcript manifest rows")
	}
	values := make([]TranscriptManifestRef, len(wires))
	for index, wire := range wires {
		if len(wire) != 3 || json.Unmarshal(wire[0], &values[index].RunID) != nil || json.Unmarshal(wire[1], &values[index].Path) != nil || json.Unmarshal(wire[2], &values[index].Digest) != nil {
			return nil, fmt.Errorf("invalid transcript manifest row")
		}
	}
	return values, nil
}

func parseTableManifestRefs(data json.RawMessage) ([]TableManifestRef, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid table manifest rows")
	}
	wantKinds := append([]uint16{101, 102, 103, 104, 105, 106, 107, 108}, []uint16{102, 103, 105, 106, 107, 108}...)
	values := make([]TableManifestRef, len(wires))
	for index, wire := range wires {
		if len(wire) != 4 || json.Unmarshal(wire[0], &values[index].Curriculum) != nil || json.Unmarshal(wire[1], &values[index].Scope) != nil || json.Unmarshal(wire[2], &values[index].Path) != nil || json.Unmarshal(wire[3], &values[index].Digest) != nil {
			return nil, fmt.Errorf("invalid table manifest row")
		}
		values[index].Kind = wantKinds[index%len(wantKinds)]
	}
	return values, nil
}
