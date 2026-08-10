package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type ObjectManifestRef struct {
	Scope  ObjectScope
	Path   string
	Digest string
}

func (r ObjectManifestRef) wire() []any { return []any{r.Scope.value(), r.Path, r.Digest} }

type TranscriptManifestRef struct {
	RunID  string
	Path   string
	Digest string
}

func (r TranscriptManifestRef) wire() []any { return []any{r.RunID, r.Path, r.Digest} }

type TableManifestRef struct {
	Curriculum int
	Scope      string
	Kind       uint16
	Path       string
	Digest     string
}

func (r TableManifestRef) wire() []any { return []any{r.Curriculum, r.Scope, r.Path, r.Digest} }

type StoreBoundaryRow struct {
	Curriculum           int
	Scope                string
	BoundaryDigest       string
	PreboundaryIndexRoot string
}

func (r StoreBoundaryRow) wire() []any {
	return []any{r.Curriculum, r.Scope, r.BoundaryDigest, r.PreboundaryIndexRoot}
}

type EvidencePayload struct {
	Panel                string
	Authority            string
	FixtureRoot          AuthorityRef
	ExecutionCore        AuthorityRef
	PlanReview           AuthorityRef
	ImplementationReview AuthorityRef
	BuildAuthority       AuthorityRef
	Competence           AuthorityRef
	AuditAttestation     AuthorityRef
	RunEvidence          AuthorityRef
	StructuralMaps       []AuthorityRef
	StoreBoundaries      []StoreBoundaryRow
	ObjectPackRoots      []ObjectManifestRef
	JournalPackRoots     []TranscriptManifestRef
	InputPackRoots       []TranscriptManifestRef
	DetailPackRoots      []TranscriptManifestRef
	AcquisitionTables    []TableManifestRef
	IndexRoots           []ObjectManifestRef
	WorldPolicyRowsRoot  string
	CurriculumRowsRoot   string
	Canonical            []byte
	Digest               string
}

func BuildEvidencePayload(value EvidencePayload) (EvidencePayload, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := evidencePayloadCanonical(value)
	if err != nil {
		return EvidencePayload{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	if err := VerifyEvidencePayload(value); err != nil {
		return EvidencePayload{}, err
	}
	return value, nil
}

func VerifyEvidencePayload(value EvidencePayload) error {
	canonical, err := evidencePayloadCanonical(value)
	if err != nil || len(value.Canonical) > 2*1024*1024 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid evidence payload")
	}
	return nil
}

func evidencePayloadCanonical(value EvidencePayload) ([]byte, error) {
	curricula := panelRunCounts[value.Panel] / 44
	if !validPanelAuthority(value.Panel, value.Authority) || curricula == 0 || !digestText(value.WorldPolicyRowsRoot) || !digestText(value.CurriculumRowsRoot) {
		return nil, fmt.Errorf("invalid evidence payload identity")
	}
	for _, reference := range []AuthorityRef{value.FixtureRoot, value.ExecutionCore, value.PlanReview, value.ImplementationReview, value.BuildAuthority, value.Competence, value.AuditAttestation, value.RunEvidence} {
		if reference.Verify() != nil {
			return nil, fmt.Errorf("invalid evidence payload authority reference")
		}
	}
	structural, err := authorityRefWires(value.Panel, value.StructuralMaps, curricula)
	if err != nil {
		return nil, err
	}
	boundaries, err := boundaryWires(value.StoreBoundaries, curricula)
	if err != nil {
		return nil, err
	}
	objects, err := objectManifestWires(value.Panel, value.ObjectPackRoots, curricula)
	if err != nil {
		return nil, err
	}
	indexes, err := objectManifestWires(value.Panel, value.IndexRoots, curricula)
	if err != nil {
		return nil, err
	}
	journals, err := transcriptManifestWires(value.Panel, value.JournalPackRoots, panelRunCounts[value.Panel])
	if err != nil {
		return nil, err
	}
	inputs, err := transcriptManifestWires(value.Panel, value.InputPackRoots, panelRunCounts[value.Panel])
	if err != nil {
		return nil, err
	}
	details, err := transcriptManifestWires(value.Panel, value.DetailPackRoots, panelRunCounts[value.Panel])
	if err != nil {
		return nil, err
	}
	tables, err := tableManifestWires(value.Panel, value.AcquisitionTables, curricula)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]any{"actionrelation-evidence-payload/v3", value.FixtureRoot.Wire(), value.ExecutionCore.Wire(), value.PlanReview.Wire(), value.ImplementationReview.Wire(), value.BuildAuthority.Wire(), value.Competence.Wire(), value.AuditAttestation.Wire(), value.RunEvidence.Wire(), structural, boundaries, objects, journals, inputs, details, tables, indexes, value.WorldPolicyRowsRoot, value.CurriculumRowsRoot})
}

func authorityRefWires(panel string, values []AuthorityRef, count int) ([]any, error) {
	if len(values) != count {
		return nil, fmt.Errorf("authority reference cardinality mismatch")
	}
	result := make([]any, len(values))
	previous := ""
	for index, value := range values {
		if value.Verify() != nil || !manifestPath(panel, value.Path) || index > 0 && value.Path <= previous {
			return nil, fmt.Errorf("invalid ordered authority references")
		}
		result[index], previous = value.Wire(), value.Path
	}
	return result, nil
}

func boundaryWires(values []StoreBoundaryRow, curricula int) ([]any, error) {
	if len(values) != curricula*2 {
		return nil, fmt.Errorf("store boundary cardinality mismatch")
	}
	result := make([]any, len(values))
	for index, value := range values {
		wantCurriculum, wantScope := index/2, "nous"
		if index%2 == 1 {
			wantScope = "no-guard"
		}
		if value.Curriculum != wantCurriculum || value.Scope != wantScope || !digestText(value.BoundaryDigest) || !digestText(value.PreboundaryIndexRoot) {
			return nil, fmt.Errorf("invalid store boundary row %d", index)
		}
		result[index] = value.wire()
	}
	return result, nil
}

var objectScopeOrder = []string{"acquisition-nous-preboundary", "acquisition-no-guard-preboundary", "utility", "authority"}

func objectManifestWires(panel string, values []ObjectManifestRef, curricula int) ([]any, error) {
	if len(values) != curricula*len(objectScopeOrder) {
		return nil, fmt.Errorf("object manifest cardinality mismatch")
	}
	result := make([]any, len(values))
	for index, value := range values {
		wantCurriculum := index / len(objectScopeOrder)
		wantClass := objectScopeOrder[index%len(objectScopeOrder)]
		if value.Scope.Curriculum != wantCurriculum || value.Scope.Class != wantClass || value.Scope.validate() != nil || !manifestPath(panel, value.Path) || !digestText(value.Digest) {
			return nil, fmt.Errorf("invalid object manifest reference %d", index)
		}
		result[index] = value.wire()
	}
	return result, nil
}

func transcriptManifestWires(panel string, values []TranscriptManifestRef, count int) ([]any, error) {
	if len(values) != count {
		return nil, fmt.Errorf("transcript manifest cardinality mismatch")
	}
	result := make([]any, len(values))
	for index, value := range values {
		if !runIDText(value.RunID) || !manifestPath(panel, value.Path) || !digestText(value.Digest) || index > 0 && value.RunID <= values[index-1].RunID {
			return nil, fmt.Errorf("invalid transcript manifest reference %d", index)
		}
		result[index] = value.wire()
	}
	return result, nil
}

func tableManifestWires(panel string, values []TableManifestRef, curricula int) ([]any, error) {
	wantKinds := append([]uint16{101, 102, 103, 104, 105, 106, 107, 108}, []uint16{102, 103, 105, 106, 107, 108}...)
	if len(values) != curricula*len(wantKinds) {
		return nil, fmt.Errorf("table manifest cardinality mismatch")
	}
	result := make([]any, len(values))
	for index, value := range values {
		within := index % len(wantKinds)
		wantScope := "nous"
		if within >= 8 {
			wantScope = "no-guard"
		}
		if value.Curriculum != index/len(wantKinds) || value.Scope != wantScope || value.Kind != wantKinds[within] || !manifestPath(panel, value.Path) || !digestText(value.Digest) {
			return nil, fmt.Errorf("invalid table manifest reference %d", index)
		}
		result[index] = value.wire()
	}
	return result, nil
}

func manifestPath(panel, path string) bool {
	root, err := EvidenceRoot(panel)
	if err != nil || !strings.HasPrefix(path, root+"/manifests/") {
		return false
	}
	placeholder := AuthorityRef{Path: path, Digest: strings.Repeat("0", 64), Mode: "100644"}
	return placeholder.Verify() == nil
}

func sortTranscriptRefs(values []TranscriptManifestRef) {
	slices.SortFunc(values, func(a, b TranscriptManifestRef) int { return strings.Compare(a.RunID, b.RunID) })
}
