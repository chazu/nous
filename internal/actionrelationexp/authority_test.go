package actionrelationexp

import (
	"fmt"
	"testing"
)

func TestAuthorityReferenceRejectsNoncanonicalPaths(t *testing.T) {
	valid, err := Reference(".nous/actionrelations-v1-development-evidence/authority/execution-primary.json", []byte("primary"))
	if err != nil || valid.Verify() != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../escape", "/absolute", "has\\slash", "Uppercase.json", "double//slash"} {
		candidate := valid
		candidate.Path = path
		if candidate.Verify() == nil {
			t.Fatalf("accepted unsafe authority path %q", path)
		}
	}
}

func TestExecutionManifestsAndAuditAttestationBindIdenticalEvidence(t *testing.T) {
	ref := func(path string) AuthorityRef {
		value, err := Reference(path, []byte(path))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	base := ".nous/actionrelations-v1-development-evidence/"
	structural := make([]AuthorityRef, 16)
	for curriculum := range structural {
		structural[curriculum] = ref(fmt.Sprintf("%smanifests/curriculum-%04d/structural-output-map.json", base, curriculum))
	}
	primary, err := BuildExecutionManifest(ExecutionManifest{
		Role: "primary", Panel: "development", Authority: "development-public-v1",
		SourceRoot: testAuthorityDigest("source"), BinaryDigest: testAuthorityDigest("binary"),
		Environment: []EnvironmentRow{{Key: "GOMAXPROCS", Value: "1"}},
		FixtureRoot: ref(base + "authority/fixture-root.json"), RunEvidence: ref(base + "manifests/run-evidence-root.json"),
		StructuralMaps: structural, RunIDsRoot: testAuthorityDigest("runs"), TranscriptRowsRoot: testAuthorityDigest("transcripts"), ResultRowsRoot: testAuthorityDigest("results"), TotalRuns: 704,
	})
	if err != nil {
		t.Fatal(err)
	}
	primaryRef := ref(base + "authority/execution-primary.json")
	primaryRef.Digest = primary.Digest
	audit, err := BuildExecutionManifest(ExecutionManifest{
		Role: "audit", Panel: primary.Panel, Authority: primary.Authority, SourceRoot: primary.SourceRoot, BinaryDigest: primary.BinaryDigest,
		Environment: primary.Environment, FixtureRoot: primary.FixtureRoot, RunEvidence: primary.RunEvidence, StructuralMaps: primary.StructuralMaps,
		RunIDsRoot: primary.RunIDsRoot, TranscriptRowsRoot: primary.TranscriptRowsRoot, ResultRowsRoot: primary.ResultRowsRoot, TotalRuns: primary.TotalRuns, PriorExecution: &primaryRef,
	})
	if err != nil || EqualExecutionEvidence(primary, audit) != nil {
		t.Fatalf("audit=%v equality=%v", err, EqualExecutionEvidence(primary, audit))
	}
	auditRef := ref(base + "authority/execution-audit.json")
	auditRef.Digest = audit.Digest
	attestation, err := BuildAuditAttestation(AuditAttestation{
		Panel: primary.Panel, Authority: primary.Authority, PrimaryExecution: primaryRef, AuditExecution: auditRef,
		RunEvidence: primary.RunEvidence, StructuralMaps: primary.StructuralMaps, RunIDsRoot: primary.RunIDsRoot,
		TranscriptRowsRoot: primary.TranscriptRowsRoot, ResultRowsRoot: primary.ResultRowsRoot, TotalRuns: primary.TotalRuns,
	})
	if err != nil || VerifyAuditAttestation(attestation) != nil {
		t.Fatal(err)
	}
	attestationRef := ref(base + "authority/audit-attestation.json")
	attestationRef.Digest = attestation.Digest
	core, err := BuildExecutionCore(ExecutionCore{
		Panel: primary.Panel, Authority: primary.Authority, SourceRoot: primary.SourceRoot, BinaryDigest: primary.BinaryDigest,
		PlanReview: ref("docs/actionrelations-plan-reviews.json"), ImplementationReview: ref("docs/actionrelations-implementation-reviews.json"),
		BuildAuthority: ref("docs/actionrelations-build-authority.json"), Competence: ref("docs/actionrelations-competence-root.json"),
		Environment: primary.Environment, FixtureRoot: primary.FixtureRoot, PrimaryExecution: primaryRef, AuditExecution: auditRef,
		AuditAttestation: attestationRef, RunEvidence: primary.RunEvidence, StructuralMaps: primary.StructuralMaps,
	})
	if err != nil || VerifyExecutionCore(core) != nil {
		t.Fatal(err)
	}
	corrupt := audit
	corrupt.ResultRowsRoot = testAuthorityDigest("changed")
	if EqualExecutionEvidence(primary, corrupt) == nil {
		t.Fatal("accepted divergent audit evidence")
	}
}

func testAuthorityDigest(value string) string { return shaHex([]byte(value)) }
