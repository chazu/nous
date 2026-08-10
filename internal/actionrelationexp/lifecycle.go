package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Claim struct {
	Panel      string
	BaseCommit string
	SourceRoot string
	Authority  string
	Canonical  []byte
	Digest     string
}

func BuildClaim(value Claim) (Claim, error) {
	value.Canonical, value.Digest = nil, ""
	if value.Panel != "validation" && value.Panel != "locked" || !commitText(value.BaseCommit) || !digestText(value.SourceRoot) || !validPanelAuthority(value.Panel, value.Authority) {
		return Claim{}, fmt.Errorf("invalid claim authority")
	}
	value.Canonical, _ = json.Marshal([]any{"actionrelation-claim/v1", value.Panel, "claimed", value.BaseCommit, value.SourceRoot, value.Authority})
	value.Digest = shaHex(value.Canonical)
	return value, VerifyClaim(value)
}

func VerifyClaim(value Claim) error {
	rebuilt := value
	rebuilt.Canonical, rebuilt.Digest = nil, ""
	want, err := BuildClaimUnchecked(rebuilt)
	if err != nil || len(value.Canonical) > 4096 || !bytes.Equal(want, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid claim")
	}
	return nil
}

func BuildClaimUnchecked(value Claim) ([]byte, error) {
	if value.Panel != "validation" && value.Panel != "locked" || !commitText(value.BaseCommit) || !digestText(value.SourceRoot) || !validPanelAuthority(value.Panel, value.Authority) {
		return nil, fmt.Errorf("invalid claim authority")
	}
	return json.Marshal([]any{"actionrelation-claim/v1", value.Panel, "claimed", value.BaseCommit, value.SourceRoot, value.Authority})
}

type Running struct {
	Panel                string
	ClaimReceiptDigest   string
	ClaimCommit          string
	SourceRoot           string
	AttemptCommitment    string
	SecretLocationDigest *string
	Canonical            []byte
	Digest               string
}

func BuildRunning(value Running) (Running, error) {
	value.Canonical, value.Digest = nil, ""
	canonical, err := runningCanonical(value)
	if err != nil {
		return Running{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	return value, VerifyRunning(value)
}

func VerifyRunning(value Running) error {
	canonical, err := runningCanonical(value)
	if err != nil || len(value.Canonical) > 4096 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid running receipt")
	}
	return nil
}

func runningCanonical(value Running) ([]byte, error) {
	if value.Panel != "validation" && value.Panel != "locked" || !digestText(value.ClaimReceiptDigest) || !commitText(value.ClaimCommit) || !digestText(value.SourceRoot) || !digestText(value.AttemptCommitment) {
		return nil, fmt.Errorf("invalid running authority")
	}
	secret := any(nil)
	if value.Panel == "validation" {
		if value.SecretLocationDigest != nil {
			return nil, fmt.Errorf("validation running receipt has secret location")
		}
	} else {
		if value.SecretLocationDigest == nil || !digestText(*value.SecretLocationDigest) {
			return nil, fmt.Errorf("locked running receipt lacks secret location")
		}
		secret = *value.SecretLocationDigest
	}
	return json.Marshal([]any{"actionrelation-running/v1", value.Panel, "running", value.ClaimReceiptDigest, value.ClaimCommit, value.SourceRoot, value.AttemptCommitment, secret})
}

type TerminalReceipt struct {
	Panel             string
	State             string
	RunningReceipt    *AuthorityRef
	SourceRoot        string
	FixtureRoot       AuthorityRef
	AttemptCommitment string
	Report            AuthorityRef
	EvidencePayload   AuthorityRef
	Reason            string
	Canonical         []byte
	Digest            string
}

func BuildTerminalReceipt(value TerminalReceipt) (TerminalReceipt, error) {
	value.Canonical, value.Digest = nil, ""
	canonical, err := terminalReceiptCanonical(value)
	if err != nil {
		return TerminalReceipt{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	return value, VerifyTerminalReceipt(value)
}

func VerifyTerminalReceipt(value TerminalReceipt) error {
	canonical, err := terminalReceiptCanonical(value)
	if err != nil || len(value.Canonical) > 8192 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid terminal receipt")
	}
	return nil
}

func terminalReceiptCanonical(value TerminalReceipt) ([]byte, error) {
	if !panelNames[value.Panel] || value.State != "published" && value.State != "invalid" || !digestText(value.SourceRoot) || !digestText(value.AttemptCommitment) || value.FixtureRoot.Verify() != nil || value.Report.Verify() != nil || value.EvidencePayload.Verify() != nil || !boundedASCII(value.Reason, 1024) {
		return nil, fmt.Errorf("invalid terminal receipt authority")
	}
	running := any(zeroAuthorityDigest)
	if value.Panel == "development" {
		if value.RunningReceipt != nil || value.AttemptCommitment != zeroAuthorityDigest {
			return nil, fmt.Errorf("development receipt has protected attempt authority")
		}
	} else {
		if value.RunningReceipt == nil || !referenceAt(*value.RunningReceipt, ExpectedAuthorityPath(value.Panel, "running")) {
			return nil, fmt.Errorf("protected receipt lacks running authority")
		}
		running = value.RunningReceipt.Wire()
	}
	for _, item := range []struct {
		ref  AuthorityRef
		path string
	}{{value.FixtureRoot, ExpectedAuthorityPath(value.Panel, "fixture-root")}, {value.Report, ExpectedAuthorityPath(value.Panel, "report")}, {value.EvidencePayload, ExpectedAuthorityPath(value.Panel, "evidence-payload")}} {
		if !referenceAt(item.ref, item.path) {
			return nil, fmt.Errorf("noncanonical terminal receipt authority path")
		}
	}
	return json.Marshal([]any{"actionrelation-terminal-receipt/v2", value.Panel, value.State, running, value.SourceRoot, value.FixtureRoot.Wire(), value.AttemptCommitment, value.Report.Wire(), value.EvidencePayload.Wire(), value.Reason})
}

type Publication struct {
	Panel                string
	PlanReview           AuthorityRef
	ImplementationReview AuthorityRef
	BuildAuthority       AuthorityRef
	Competence           AuthorityRef
	ClaimReceipt         *AuthorityRef
	RunningReceipt       *AuthorityRef
	PrimaryExecution     AuthorityRef
	AuditExecution       AuthorityRef
	AuditAttestation     AuthorityRef
	RunEvidence          AuthorityRef
	StructuralMaps       []AuthorityRef
	FixtureRoot          AuthorityRef
	ExecutionCore        AuthorityRef
	EvidencePayload      AuthorityRef
	Report               AuthorityRef
	TerminalReceipt      AuthorityRef
	Canonical            []byte
	Digest               string
}

func BuildPublication(value Publication) (Publication, error) {
	value.Canonical, value.Digest = nil, ""
	canonical, err := publicationCanonical(value)
	if err != nil {
		return Publication{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	return value, VerifyPublication(value)
}

func VerifyPublication(value Publication) error {
	canonical, err := publicationCanonical(value)
	if err != nil || len(value.Canonical) > 8192 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid publication")
	}
	return nil
}

func publicationCanonical(value Publication) ([]byte, error) {
	if !panelNames[value.Panel] {
		return nil, fmt.Errorf("invalid publication panel")
	}
	for _, reference := range []AuthorityRef{value.PlanReview, value.ImplementationReview, value.BuildAuthority, value.Competence, value.PrimaryExecution, value.AuditExecution, value.AuditAttestation, value.RunEvidence, value.FixtureRoot, value.ExecutionCore, value.EvidencePayload, value.Report, value.TerminalReceipt} {
		if reference.Verify() != nil {
			return nil, fmt.Errorf("invalid publication authority reference")
		}
	}
	paths := []struct {
		ref  AuthorityRef
		path string
	}{
		{value.PlanReview, ReviewManifestPath("plan")}, {value.ImplementationReview, ReviewManifestPath("implementation")},
		{value.BuildAuthority, BuildAuthorityPath}, {value.Competence, "docs/actionrelations-competence-root.json"},
		{value.PrimaryExecution, ExpectedAuthorityPath(value.Panel, "execution-primary")}, {value.AuditExecution, ExpectedAuthorityPath(value.Panel, "execution-audit")},
		{value.AuditAttestation, ExpectedAuthorityPath(value.Panel, "audit-attestation")}, {value.RunEvidence, ExpectedAuthorityPath(value.Panel, "run-evidence")},
		{value.FixtureRoot, ExpectedAuthorityPath(value.Panel, "fixture-root")}, {value.ExecutionCore, ExpectedAuthorityPath(value.Panel, "execution-core")},
		{value.EvidencePayload, ExpectedAuthorityPath(value.Panel, "evidence-payload")}, {value.Report, ExpectedAuthorityPath(value.Panel, "report")},
		{value.TerminalReceipt, ExpectedAuthorityPath(value.Panel, "terminal-receipt")},
	}
	for _, item := range paths {
		if !referenceAt(item.ref, item.path) {
			return nil, fmt.Errorf("noncanonical publication authority path")
		}
	}
	structural, err := structuralMapWires(value.Panel, value.StructuralMaps)
	if err != nil {
		return nil, err
	}
	claim, running := any(zeroAuthorityDigest), any(zeroAuthorityDigest)
	if value.Panel == "development" {
		if value.ClaimReceipt != nil || value.RunningReceipt != nil {
			return nil, fmt.Errorf("development publication has protected receipts")
		}
	} else {
		if value.ClaimReceipt == nil || !referenceAt(*value.ClaimReceipt, ExpectedAuthorityPath(value.Panel, "claim")) || value.RunningReceipt == nil || !referenceAt(*value.RunningReceipt, ExpectedAuthorityPath(value.Panel, "running")) {
			return nil, fmt.Errorf("protected publication lacks receipts")
		}
		claim, running = value.ClaimReceipt.Wire(), value.RunningReceipt.Wire()
	}
	return json.Marshal([]any{"actionrelation-publication/v3", value.PlanReview.Wire(), value.ImplementationReview.Wire(), value.BuildAuthority.Wire(), value.Competence.Wire(), claim, running, value.PrimaryExecution.Wire(), value.AuditExecution.Wire(), value.AuditAttestation.Wire(), value.RunEvidence.Wire(), structural, value.FixtureRoot.Wire(), value.ExecutionCore.Wire(), value.EvidencePayload.Wire(), value.Report.Wire(), value.TerminalReceipt.Wire()})
}
