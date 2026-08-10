package actionrelationexp

import (
	"fmt"
	"testing"
)

func TestEvidencePayloadClosesEveryFrozenDevelopmentReferenceClass(t *testing.T) {
	root, _ := EvidenceRoot("development")
	ref := func(path string) AuthorityRef {
		value, err := Reference(path, []byte(path))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	value := EvidencePayload{
		Panel: "development", Authority: "development-public-v1",
		FixtureRoot: ref(root + "/authority/fixture-root.json"), ExecutionCore: ref(root + "/authority/execution-core.json"),
		PlanReview: ref("docs/actionrelations-plan-reviews.json"), ImplementationReview: ref("docs/actionrelations-implementation-reviews.json"),
		BuildAuthority: ref("docs/actionrelations-build-authority.json"), Competence: ref("docs/actionrelations-competence-root.json"),
		AuditAttestation: ref(root + "/authority/audit-attestation.json"), RunEvidence: ref(root + "/manifests/run-evidence-root.json"),
		WorldPolicyRowsRoot: testAuthorityDigest("world rows"), CurriculumRowsRoot: testAuthorityDigest("curriculum rows"),
	}
	for curriculum := 0; curriculum < 16; curriculum++ {
		value.StructuralMaps = append(value.StructuralMaps, ref(fmt.Sprintf("%s/manifests/curriculum-%04d/structural-output-map.json", root, curriculum)))
		for _, scope := range []string{"nous", "no-guard"} {
			value.StoreBoundaries = append(value.StoreBoundaries, StoreBoundaryRow{Curriculum: curriculum, Scope: scope, BoundaryDigest: testAuthorityDigest(fmt.Sprintf("boundary-%d-%s", curriculum, scope)), PreboundaryIndexRoot: testAuthorityDigest(fmt.Sprintf("index-%d-%s", curriculum, scope))})
		}
		for _, class := range objectScopeOrder {
			scope := ObjectScope{Curriculum: curriculum, Class: class}
			value.ObjectPackRoots = append(value.ObjectPackRoots, ObjectManifestRef{Scope: scope, Path: fmt.Sprintf("%s/manifests/curriculum-%04d/%s-object-root.json", root, curriculum, class), Digest: testAuthorityDigest("object" + fmt.Sprint(curriculum) + class)})
			value.IndexRoots = append(value.IndexRoots, ObjectManifestRef{Scope: scope, Path: fmt.Sprintf("%s/manifests/curriculum-%04d/%s-index-root.json", root, curriculum, class), Digest: testAuthorityDigest("index" + fmt.Sprint(curriculum) + class)})
		}
		for within, kind := range append([]uint16{101, 102, 103, 104, 105, 106, 107, 108}, []uint16{102, 103, 105, 106, 107, 108}...) {
			scope := "nous"
			if within >= 8 {
				scope = "no-guard"
			}
			value.AcquisitionTables = append(value.AcquisitionTables, TableManifestRef{Curriculum: curriculum, Scope: scope, Kind: kind, Path: fmt.Sprintf("%s/manifests/curriculum-%04d/%s-table-%03d.json", root, curriculum, scope, kind), Digest: testAuthorityDigest(fmt.Sprintf("table-%d-%s-%d", curriculum, scope, kind))})
		}
	}
	for run := 0; run < 704; run++ {
		runID := fmt.Sprintf("%032x", run+1)
		value.JournalPackRoots = append(value.JournalPackRoots, TranscriptManifestRef{RunID: runID, Path: fmt.Sprintf("%s/manifests/runs/%s-journal-root.json", root, runID), Digest: testAuthorityDigest("journal" + runID)})
		value.InputPackRoots = append(value.InputPackRoots, TranscriptManifestRef{RunID: runID, Path: fmt.Sprintf("%s/manifests/runs/%s-input-root.json", root, runID), Digest: testAuthorityDigest("input" + runID)})
		value.DetailPackRoots = append(value.DetailPackRoots, TranscriptManifestRef{RunID: runID, Path: fmt.Sprintf("%s/manifests/runs/%s-detail-root.json", root, runID), Digest: testAuthorityDigest("detail" + runID)})
	}
	payload, err := BuildEvidencePayload(value)
	if err != nil || VerifyEvidencePayload(payload) != nil {
		t.Fatal(err)
	}
	corrupt := payload
	corrupt.StoreBoundaries = append([]StoreBoundaryRow(nil), payload.StoreBoundaries...)
	corrupt.StoreBoundaries[1].Scope = "nous"
	if VerifyEvidencePayload(corrupt) == nil {
		t.Fatal("accepted reordered boundary authority")
	}
}
