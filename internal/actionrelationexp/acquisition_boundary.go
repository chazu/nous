package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/unit"
)

var acquisitionObjectKinds = map[string]uint16{
	"FiniteActionState":           1,
	"FiniteSemanticAction":        2,
	"ActionOccurrence":            3,
	"FiniteActionWorldCore":       4,
	"ActionRelationPattern":       6,
	"ActionGuard":                 7,
	"ActionLocalFacts":            8,
	"GuardedActionRelation":       9,
	"GuardedActionArtifact":       10,
	"ActionTrainingEvidence":      11,
	"ActionPresentationView":      12,
	"ActionNormalizationProof":    13,
	"CompoundWorkReservation":     27,
	"ActionGuardSearchBarrier":    28,
	"ActionRelationOperationRoot": 46,
}

var acquisitionTableCategories = map[string]bool{
	"ActionGuardLiteralRow": true, "ActionGuardResult": true, "ActionGuardCandidate": true,
	"ActionGuardRefinement": true, "ActionRelationObservation": true, "ActionPresentationViewEvidence": true,
	"ActionApplicabilityRow": true, "ActionTransitionRow": true, "ActionStateEqualityRow": true,
	"ActionGuardCandidateResult": true,
}

type AcquisitionBoundary struct {
	EvidenceRoot   string
	Curriculum     int
	Scope          string
	Preboundary    ObjectBundle
	Canonical      []byte
	BoundaryDigest string
	BoundaryUnit   string
}

func BuildAcquisitionBoundary(evidence AcquisitionEvidence, curriculum int, scope string) (AcquisitionBoundary, error) {
	root, _ := EvidenceRoot("development")
	return BuildAcquisitionBoundaryAt(root, evidence, curriculum, scope)
}

func BuildAcquisitionBoundaryAt(evidenceRoot string, evidence AcquisitionEvidence, curriculum int, scope string) (AcquisitionBoundary, error) {
	if evidence.Run.Store == nil || evidence.Run.Experiment == "" || curriculum < 0 || scope != "nous" && scope != "no-guard" {
		return AcquisitionBoundary{}, fmt.Errorf("invalid acquisition boundary input")
	}
	records, err := collectAcquisitionObjects(evidence.Run.Store)
	if err != nil {
		return AcquisitionBoundary{}, err
	}
	class := "acquisition-" + scope + "-preboundary"
	preboundary, err := BuildObjectBundleAt(evidenceRoot, ObjectScope{Curriculum: curriculum, Class: class}, records)
	if err != nil {
		return AcquisitionBoundary{}, err
	}
	tableKinds := []uint16{101, 102, 103, 104, 105, 106, 107, 108}
	if scope == "no-guard" {
		tableKinds = []uint16{102, 103, 105, 106, 107, 108}
	}
	tableManifestDigests := make([]string, len(tableKinds))
	for index, kind := range tableKinds {
		table, ok := evidence.Tables[kind]
		if !ok || table.Manifest.Curriculum != curriculum || table.Manifest.Scope != scope {
			return AcquisitionBoundary{}, fmt.Errorf("missing or cross-scope table %d", kind)
		}
		tableManifestDigests[index], err = canonicalDigest(table.Manifest.CanonicalJSON())
		if err != nil {
			return AcquisitionBoundary{}, err
		}
	}
	indexRootDigest, err := preboundary.IndexRoot.Digest()
	if err != nil {
		return AcquisitionBoundary{}, err
	}
	wire, _ := json.Marshal([]any{"action-store-boundary/v3", curriculum, scope, tableManifestDigests, preboundary.IndexRoot.ObjectSetRoot, indexRootDigest})
	if err := ValidateObject(35, wire); err != nil {
		return AcquisitionBoundary{}, err
	}
	digest := shaHex(wire)
	name := "AR.Boundary." + digest
	u := unit.New(name)
	u.Set("isA", []string{"ActionStoreBoundary", "Anything"})
	u.Set("canonicalObject", string(wire))
	u.Set("objectDigest", digest)
	evidence.Run.Store.Put(u)
	experiment := evidence.Run.Store.Get(evidence.Run.Experiment)
	experiment.Set("storeBoundaryUnit", name)
	experiment.Set("storeBoundaryDigest", digest)
	return AcquisitionBoundary{EvidenceRoot: evidenceRoot, Curriculum: curriculum, Scope: scope, Preboundary: preboundary, Canonical: wire, BoundaryDigest: digest, BoundaryUnit: name}, nil
}

func (b AcquisitionBoundary) Verify(evidence AcquisitionEvidence) error {
	if !validEvidenceRoot(b.EvidenceRoot) || b.Curriculum < 0 || b.Scope != "nous" && b.Scope != "no-guard" || b.BoundaryDigest != shaHex(b.Canonical) || VerifyObjectBundle(b.Preboundary) != nil || ValidateObject(35, b.Canonical) != nil {
		return fmt.Errorf("invalid acquisition boundary")
	}
	var row []json.RawMessage
	if json.Unmarshal(b.Canonical, &row) != nil || len(row) != 6 {
		return fmt.Errorf("invalid boundary wire")
	}
	canonical, _ := json.Marshal(row)
	var version, scope, objectSetRoot, indexRootDigest string
	var curriculum int
	var tableDigests []string
	if !bytes.Equal(canonical, b.Canonical) || json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &curriculum) != nil || json.Unmarshal(row[2], &scope) != nil || json.Unmarshal(row[3], &tableDigests) != nil || json.Unmarshal(row[4], &objectSetRoot) != nil || json.Unmarshal(row[5], &indexRootDigest) != nil || version != "action-store-boundary/v3" || curriculum != b.Curriculum || scope != b.Scope || objectSetRoot != b.Preboundary.IndexRoot.ObjectSetRoot {
		return fmt.Errorf("boundary authority mismatch")
	}
	wantIndex, _ := b.Preboundary.IndexRoot.Digest()
	if indexRootDigest != wantIndex {
		return fmt.Errorf("boundary index root mismatch")
	}
	kinds := []uint16{101, 102, 103, 104, 105, 106, 107, 108}
	if b.Scope == "no-guard" {
		kinds = []uint16{102, 103, 105, 106, 107, 108}
	}
	var wants []string
	for _, kind := range kinds {
		table, ok := evidence.Tables[kind]
		if !ok {
			return fmt.Errorf("missing boundary table %d", kind)
		}
		digest, _ := canonicalDigest(table.Manifest.CanonicalJSON())
		wants = append(wants, digest)
	}
	if !slices.Equal(tableDigests, wants) {
		return fmt.Errorf("boundary table roots mismatch")
	}
	records, err := collectAcquisitionObjects(evidence.Run.Store)
	if err != nil {
		return err
	}
	fresh, err := BuildObjectBundleAt(b.EvidenceRoot, b.Preboundary.Scope, records)
	if err != nil || fresh.IndexRoot.ObjectSetRoot != b.Preboundary.IndexRoot.ObjectSetRoot {
		return fmt.Errorf("post-boundary Store mutation")
	}
	unit := evidence.Run.Store.Get(b.BoundaryUnit)
	if unit == nil || unit.GetString("objectDigest") != b.BoundaryDigest {
		return fmt.Errorf("missing boundary object")
	}
	return nil
}

func collectAcquisitionObjects(store *unit.Store) ([]ObjectRecord, error) {
	byDigest := map[string]ObjectRecord{}
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil || u.GetString("canonicalObject") == "" || u.GetString("objectDigest") == "" {
			continue
		}
		var kind uint16
		isTable := false
		for _, category := range u.GetStrings("isA") {
			if value := acquisitionObjectKinds[category]; value != 0 {
				if kind != 0 && kind != value {
					return nil, fmt.Errorf("object %q has multiple decoder kinds", name)
				}
				kind = value
			}
			isTable = isTable || acquisitionTableCategories[category]
		}
		if isTable {
			if kind != 0 {
				return nil, fmt.Errorf("object %q is both table and indexed", name)
			}
			continue
		}
		// The boundary is deliberately excluded from its preboundary index.
		if slices.Contains(u.GetStrings("isA"), "ActionStoreBoundary") {
			continue
		}
		if kind == 0 {
			return nil, fmt.Errorf("unclassified acquisition object %q", name)
		}
		canonical := []byte(u.GetString("canonicalObject"))
		if shaHex(canonical) != u.GetString("objectDigest") || ValidateObject(kind, canonical) != nil {
			return nil, fmt.Errorf("invalid acquisition object %q", name)
		}
		digest := u.GetString("objectDigest")
		if previous, exists := byDigest[digest]; exists {
			if previous.Kind != kind || !bytes.Equal(previous.Bytes, canonical) {
				return nil, fmt.Errorf("cross-kind object digest collision")
			}
			continue
		}
		byDigest[digest] = ObjectRecord{Kind: kind, Bytes: canonical}
	}
	records := make([]ObjectRecord, 0, len(byDigest))
	for _, record := range byDigest {
		records = append(records, record)
	}
	return records, nil
}
