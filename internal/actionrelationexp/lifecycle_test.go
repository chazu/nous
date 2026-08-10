package actionrelationexp

import (
	"fmt"
	"testing"
)

func TestDevelopmentTerminalAndPublicationUseExactZeroReceiptAuthority(t *testing.T) {
	ref := func(path string) AuthorityRef {
		value, err := Reference(path, []byte(path))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	base := ".nous/actionrelations-v1-development-evidence/"
	receipt, err := BuildTerminalReceipt(TerminalReceipt{
		Panel: "development", State: "published", SourceRoot: shaHex([]byte("source")),
		FixtureRoot: ref(base + "authority/fixture-root.json"), AttemptCommitment: zeroAuthorityDigest,
		Report: ref(".nous/actionrelations-v1-development-report.json"), EvidencePayload: ref(base + "authority/evidence-payload.json"), Reason: "interim-power-authorized",
	})
	if err != nil {
		t.Fatal(err)
	}
	receiptRef, _ := Reference(".nous/actionrelations-v1-development-terminal-receipt.json", receipt.Canonical)
	structural := make([]AuthorityRef, 16)
	for index := range structural {
		structural[index] = ref(fmt.Sprintf("%smanifests/curriculum-%04d/structural-output-map.json", base, index))
	}
	publication, err := BuildPublication(Publication{
		Panel: "development", PlanReview: ref("docs/actionrelations-plan-reviews.json"), ImplementationReview: ref("docs/actionrelations-implementation-reviews.json"),
		BuildAuthority: ref(BuildAuthorityPath), Competence: ref("docs/actionrelations-competence-root.json"),
		PrimaryExecution: ref(base + "authority/execution-primary.json"), AuditExecution: ref(base + "authority/execution-audit.json"), AuditAttestation: ref(base + "authority/audit-attestation.json"),
		RunEvidence: ref(base + "manifests/run-evidence-root.json"), StructuralMaps: structural, FixtureRoot: ref(base + "authority/fixture-root.json"),
		ExecutionCore: ref(base + "authority/execution-core.json"), EvidencePayload: ref(base + "authority/evidence-payload.json"), Report: ref(".nous/actionrelations-v1-development-report.json"), TerminalReceipt: receiptRef,
	})
	if err != nil || VerifyPublication(publication) != nil {
		t.Fatal(err)
	}
	corrupt := publication
	corrupt.ClaimReceipt = &publication.PlanReview
	if VerifyPublication(corrupt) == nil {
		t.Fatal("accepted development claim receipt")
	}
}

func TestProtectedClaimRunningTransitionClosesCommitment(t *testing.T) {
	claim, err := BuildClaim(Claim{Panel: "validation", BaseCommit: fmt.Sprintf("%040d", 1), SourceRoot: shaHex([]byte("source")), Authority: "validation-public-v1"})
	if err != nil {
		t.Fatal(err)
	}
	running, err := BuildRunning(Running{Panel: "validation", ClaimReceiptDigest: claim.Digest, ClaimCommit: fmt.Sprintf("%040d", 2), SourceRoot: claim.SourceRoot, AttemptCommitment: shaHex([]byte("attempt"))})
	if err != nil || VerifyRunning(running) != nil {
		t.Fatal(err)
	}
	secret := shaHex([]byte("secret path"))
	if _, err := BuildRunning(Running{Panel: "validation", ClaimReceiptDigest: claim.Digest, ClaimCommit: fmt.Sprintf("%040d", 2), SourceRoot: claim.SourceRoot, AttemptCommitment: shaHex([]byte("attempt")), SecretLocationDigest: &secret}); err == nil {
		t.Fatal("accepted validation secret location")
	}
}
