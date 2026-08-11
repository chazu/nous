package actionrelationrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/actionrelationcap"
	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationscore"
	"golang.org/x/sys/unix"
)

const interruptedAttemptReason = "interrupted post-start attempt"

type panelStart struct {
	Identity string
	Path     string
	Bytes    []byte
}

func panelStartAuthority(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority) (panelStart, error) {
	identity := prerequisites.Build.SourceRoot
	if panel != "development" {
		if lifecycle.RunningRef == nil || lifecycle.RunningRef.Verify() != nil {
			return panelStart{}, fmt.Errorf("protected start lacks running authority")
		}
		identity = lifecycle.RunningRef.Digest
	}
	if len(identity) != 64 {
		return panelStart{}, fmt.Errorf("invalid panel start identity")
	}
	for _, character := range identity {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return panelStart{}, fmt.Errorf("noncanonical panel start identity")
		}
	}
	encoded, err := json.Marshal([]any{"actionrelation-start-marker/v1", panel, prerequisites.Head, identity})
	if err != nil {
		return panelStart{}, err
	}
	return panelStart{
		Identity: identity,
		Path:     filepath.Join(prerequisites.GitCommonDir, "nous-actionrelations-v1", "starts", panel+"-"+identity+".start"),
		Bytes:    encoded,
	}, nil
}

func inspectPanelStart(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority) (panelStart, bool, error) {
	start, err := panelStartAuthority(prerequisites, panel, lifecycle)
	if err != nil {
		return panelStart{}, false, err
	}
	data, _, err := readRegularNoFollowAllowLinks(start.Path, 0o600)
	if errors.Is(err, unix.ENOENT) {
		return start, false, nil
	}
	if err != nil || !bytes.Equal(data, start.Bytes) {
		return panelStart{}, false, fmt.Errorf("panel start marker does not match accepted authority")
	}
	// Reconcile the deterministic write temporary if interruption happened
	// after its link but before cleanup.
	linked, err := installAtomicNoFollow(start.Path, start.Bytes, 0o600, 0o700)
	if err != nil || linked {
		return panelStart{}, false, fmt.Errorf("reconcile panel start marker: %w", err)
	}
	return start, true, nil
}

func consumePanelStart(start panelStart) (bool, error) {
	linked, err := installAtomicNoFollow(start.Path, start.Bytes, 0o600, 0o700)
	if err != nil {
		return linked, err
	}
	if !linked {
		return false, fmt.Errorf("panel attempt already started")
	}
	return true, nil
}

func requireFreshPanelOutputsAbsent(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority) error {
	evidenceRoot, err := actionrelationexp.EvidenceRoot(panel)
	if err != nil {
		return err
	}
	paths := []string{
		evidenceRoot,
		actionrelationexp.ExpectedAuthorityPath(panel, "report"),
		actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt"),
	}
	start, err := panelStartAuthority(prerequisites, panel, lifecycle)
	if err != nil {
		return err
	}
	paths = append(paths, actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt")+".pending-"+start.Identity)
	for _, path := range paths {
		if err := requireAbsentNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("fresh panel output is not absent: %s: %w", path, err)
		}
	}
	return nil
}

func loadProtectedLifecycle(prerequisites panelPrerequisites, git func(...string) ([]byte, error), panel string) (panelLifecycleAuthority, actionrelationexp.Claim, actionrelationexp.Running, error) {
	claimPath := actionrelationexp.ExpectedAuthorityPath(panel, "claim")
	runningPath := actionrelationexp.ExpectedAuthorityPath(panel, "running")
	claimBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, claimPath)
	if err != nil {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	claim, err := actionrelationexp.ParseClaim(claimBytes)
	if err != nil || claim.Panel != panel || claim.SourceRoot != prerequisites.Build.SourceRoot || panel == "validation" && claim.Authority != "validation-public-v1" || panel == "locked" && claim.Authority != actionrelationcap.LockedClaimAuthority(claim.BaseCommit, claim.SourceRoot) {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected claim does not match build authority")
	}
	runningBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, runningPath)
	if err != nil {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	running, err := actionrelationexp.ParseRunning(runningBytes)
	if err != nil || running.Panel != panel || running.SourceRoot != prerequisites.Build.SourceRoot || running.ClaimReceiptDigest != claim.Digest {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected running receipt does not close claim")
	}
	committedClaim, err := git("cat-file", "blob", running.ClaimCommit+":"+claimPath)
	if err != nil || !bytes.Equal(committedClaim, claim.Canonical) {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("running receipt names a different claim commit")
	}
	if _, err := git("merge-base", "--is-ancestor", claim.BaseCommit, running.ClaimCommit); err != nil {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("claim base is not an ancestor of claim commit")
	}
	if _, err := git("merge-base", "--is-ancestor", running.ClaimCommit, prerequisites.Head); err != nil || running.ClaimCommit == prerequisites.Head {
		return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("running receipt was not committed after claim")
	}
	if panel == "validation" {
		if running.SecretLocationDigest != nil || running.AttemptCommitment != actionrelationcap.ValidationAttemptCommitment() {
			return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("validation attempt commitment changed")
		}
	} else {
		location, locationDigest, err := actionrelationcap.LockedSecretLocation(claim.Digest)
		if err != nil || running.SecretLocationDigest == nil || *running.SecretLocationDigest != locationDigest || !strings.HasPrefix(location, "nous-actionrelations-v1/secrets/") {
			return panelLifecycleAuthority{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("locked secret location commitment changed")
		}
	}
	claimRef, _ := actionrelationexp.Reference(claimPath, claim.Canonical)
	runningRef, _ := actionrelationexp.Reference(runningPath, running.Canonical)
	return panelLifecycleAuthority{ClaimRef: &claimRef, RunningRef: &runningRef, AttemptCommitment: running.AttemptCommitment}, claim, running, nil
}

func eraseRecoverySecret(prerequisites panelPrerequisites, panel string, claim actionrelationexp.Claim, running actionrelationexp.Running) error {
	if panel != "locked" {
		return nil
	}
	location, locationDigest, err := actionrelationcap.LockedSecretLocation(claim.Digest)
	if err != nil || running.SecretLocationDigest == nil || *running.SecretLocationDigest != locationDigest {
		return fmt.Errorf("locked recovery secret authority changed")
	}
	path := filepath.Join(prerequisites.GitCommonDir, filepath.FromSlash(location))
	data, _, err := readRegularNoFollowAllowLinks(path, 0o600)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil || len(data) != 32 {
		return fmt.Errorf("locked recovery secret preimage changed")
	}
	alreadyErased := bytes.Equal(data, make([]byte, 32))
	if !alreadyErased && digest(data) != running.AttemptCommitment {
		return fmt.Errorf("locked recovery secret preimage changed")
	}
	parent, leaf, err := openParentNoFollow(path, false, 0)
	if err != nil {
		return err
	}
	defer unix.Close(parent)
	if !alreadyErased {
		fd, openErr := unix.Openat(parent, leaf, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			return openErr
		}
		file := os.NewFile(uintptr(fd), path)
		zeros := make([]byte, 32)
		_, writeErr := file.WriteAt(zeros, 0)
		if writeErr == nil {
			writeErr = file.Sync()
		}
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err := unix.Unlinkat(parent, leaf, 0); err != nil {
		return err
	}
	return unix.Fsync(parent)
}

type successfulPanelAuthority struct {
	Report      actionrelationscore.Report
	Receipt     actionrelationexp.TerminalReceipt
	Publication actionrelationexp.Publication
}

func recoverStartedPanel(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority) (actionrelationscore.Report, error) {
	start, err := panelStartAuthority(prerequisites, panel, lifecycle)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	receiptPath := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt")))
	publicationPath := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "publication")))
	receiptPending := receiptPath + ".pending-" + start.Identity
	publicationPending := publicationPath + ".pending-" + start.Identity

	receiptBytes, _, receiptErr := readRegularNoFollowAllowLinks(receiptPath, 0o644)
	publicationBytes, _, publicationErr := readRegularNoFollowAllowLinks(publicationPath, 0o644)
	if receiptErr == nil {
		receipt, parseErr := actionrelationexp.ParseTerminalReceipt(receiptBytes)
		if parseErr != nil || receipt.Panel != panel || receipt.SourceRoot != prerequisites.Build.SourceRoot || receipt.AttemptCommitment != lifecycle.AttemptCommitment || !sameOptionalRef(receipt.RunningReceipt, lifecycle.RunningRef) {
			return actionrelationscore.Report{}, fmt.Errorf("existing terminal receipt conflicts with started attempt")
		}
		if err := reconcileFinalAuthority(receiptPending, receiptPath, receipt.Canonical); err != nil {
			return actionrelationscore.Report{}, err
		}
		switch receipt.State {
		case "invalid":
			if publicationErr == nil {
				return actionrelationscore.Report{}, fmt.Errorf("invalid terminal has a publication")
			}
			if !errors.Is(publicationErr, unix.ENOENT) {
				return actionrelationscore.Report{}, publicationErr
			}
			if err := cleanupInvalidSuccessStaging(prerequisites, panel, lifecycle, receiptPending, publicationPending); err != nil {
				return actionrelationscore.Report{}, err
			}
			if err := publishInvalidPanel(prerequisites, panel, lifecycle, errors.New(receipt.Reason)); err != nil {
				return actionrelationscore.Report{}, err
			}
			return actionrelationscore.Report{}, fmt.Errorf("panel attempt is terminally invalid: %s", receipt.Reason)
		case "published":
			authority, err := reconstructSuccessfulPanel(prerequisites, panel, lifecycle)
			if err != nil {
				return actionrelationscore.Report{}, fmt.Errorf("published receipt cannot reconstruct successful authority: %w", err)
			}
			if !bytes.Equal(authority.Receipt.Canonical, receipt.Canonical) {
				return actionrelationscore.Report{}, fmt.Errorf("published receipt differs from reconstructed authority")
			}
			if publicationErr == nil {
				if !bytes.Equal(publicationBytes, authority.Publication.Canonical) {
					return actionrelationscore.Report{}, fmt.Errorf("published authority bytes changed")
				}
				if err := reconcileFinalAuthority(publicationPending, publicationPath, authority.Publication.Canonical); err != nil {
					return actionrelationscore.Report{}, publishedCommitError{err: err}
				}
			} else if errors.Is(publicationErr, unix.ENOENT) {
				if err := writeExclusiveAuthority(publicationPending, authority.Publication.Canonical); err != nil {
					return actionrelationscore.Report{}, publishedCommitError{err: err}
				}
				if _, err := linkStagedNoFollow(publicationPending, publicationPath, authority.Publication.Canonical, 0o644); err != nil {
					return actionrelationscore.Report{}, publishedCommitError{err: err}
				}
			} else {
				return actionrelationscore.Report{}, publicationErr
			}
			_ = removeExpectedNoFollow(receiptPending, authority.Receipt.Canonical, 0o644)
			if err := verifyLocalSuccessfulPanel(prerequisites, authority, publicationPath); err != nil {
				return actionrelationscore.Report{}, publishedCommitError{err: err}
			}
			return authority.Report, nil
		default:
			return actionrelationscore.Report{}, fmt.Errorf("unknown terminal state")
		}
	}
	if !errors.Is(receiptErr, unix.ENOENT) {
		return actionrelationscore.Report{}, receiptErr
	}
	if publicationErr == nil {
		return actionrelationscore.Report{}, fmt.Errorf("publication exists without terminal receipt")
	}
	if !errors.Is(publicationErr, unix.ENOENT) {
		return actionrelationscore.Report{}, publicationErr
	}
	pendingReceipt, _, pendingReceiptErr := readRegularNoFollowAllowLinks(receiptPending, 0o644)
	pendingPublication, _, pendingPublicationErr := readRegularNoFollowAllowLinks(publicationPending, 0o644)
	havePendingReceipt := pendingReceiptErr == nil
	havePendingPublication := pendingPublicationErr == nil
	if pendingReceiptErr != nil && !errors.Is(pendingReceiptErr, unix.ENOENT) {
		return actionrelationscore.Report{}, pendingReceiptErr
	}
	if pendingPublicationErr != nil && !errors.Is(pendingPublicationErr, unix.ENOENT) {
		return actionrelationscore.Report{}, pendingPublicationErr
	}
	if havePendingReceipt || havePendingPublication {
		authority, err := reconstructSuccessfulPanel(prerequisites, panel, lifecycle)
		if err != nil {
			return actionrelationscore.Report{}, fmt.Errorf("success staging cannot reconstruct: %w", err)
		}
		if havePendingReceipt && !bytes.Equal(pendingReceipt, authority.Receipt.Canonical) || havePendingPublication && !bytes.Equal(pendingPublication, authority.Publication.Canonical) {
			return actionrelationscore.Report{}, fmt.Errorf("success staging differs from reconstructed authority")
		}
		if havePendingReceipt && havePendingPublication {
			if err := verifyLocalSuccessfulPanel(prerequisites, authority, publicationPending); err != nil {
				return actionrelationscore.Report{}, err
			}
			committed, receiptLinkErr := linkStagedNoFollow(receiptPending, receiptPath, authority.Receipt.Canonical, 0o644)
			if !committed {
				return actionrelationscore.Report{}, receiptLinkErr
			}
			_, publicationLinkErr := linkStagedNoFollow(publicationPending, publicationPath, authority.Publication.Canonical, 0o644)
			if err := errors.Join(receiptLinkErr, publicationLinkErr); err != nil {
				return actionrelationscore.Report{}, publishedCommitError{err: err}
			}
			return authority.Report, nil
		}
		if havePendingReceipt {
			if err := removeExpectedNoFollow(receiptPending, authority.Receipt.Canonical, 0o644); err != nil {
				return actionrelationscore.Report{}, err
			}
		}
		if havePendingPublication {
			if err := removeExpectedNoFollow(publicationPending, authority.Publication.Canonical, 0o644); err != nil {
				return actionrelationscore.Report{}, err
			}
		}
	}
	if err := publishInvalidPanel(prerequisites, panel, lifecycle, errors.New(interruptedAttemptReason)); err != nil {
		return actionrelationscore.Report{}, err
	}
	return actionrelationscore.Report{}, fmt.Errorf("panel attempt is terminally invalid: %s", interruptedAttemptReason)
}

func cleanupInvalidSuccessStaging(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority, receiptPending, publicationPending string) error {
	receiptBytes, _, receiptErr := readRegularNoFollowAllowLinks(receiptPending, 0o644)
	publicationBytes, _, publicationErr := readRegularNoFollowAllowLinks(publicationPending, 0o644)
	if errors.Is(receiptErr, unix.ENOENT) && errors.Is(publicationErr, unix.ENOENT) {
		return nil
	}
	if receiptErr != nil && !errors.Is(receiptErr, unix.ENOENT) {
		return receiptErr
	}
	if publicationErr != nil && !errors.Is(publicationErr, unix.ENOENT) {
		return publicationErr
	}
	authority, err := reconstructSuccessfulPanel(prerequisites, panel, lifecycle)
	if err != nil {
		return fmt.Errorf("invalid terminal has unreconciled success staging: %w", err)
	}
	if receiptErr == nil && !bytes.Equal(receiptBytes, authority.Receipt.Canonical) || publicationErr == nil && !bytes.Equal(publicationBytes, authority.Publication.Canonical) {
		return fmt.Errorf("invalid terminal has changed success staging")
	}
	if receiptErr == nil {
		if err := removeExpectedNoFollow(receiptPending, authority.Receipt.Canonical, 0o644); err != nil {
			return err
		}
	}
	if publicationErr == nil {
		if err := removeExpectedNoFollow(publicationPending, authority.Publication.Canonical, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func reconcileFinalAuthority(pendingPath, finalPath string, data []byte) error {
	pending, _, pendingErr := readRegularNoFollowAllowLinks(pendingPath, 0o644)
	if pendingErr == nil {
		if !bytes.Equal(pending, data) {
			return fmt.Errorf("pending authority differs from final bytes: %s", pendingPath)
		}
		_, err := linkStagedNoFollow(pendingPath, finalPath, data, 0o644)
		return err
	}
	if !errors.Is(pendingErr, unix.ENOENT) {
		return pendingErr
	}
	_, err := installAtomicNoFollow(finalPath, data, 0o644, 0o755)
	return err
}

func sameOptionalRef(left, right *actionrelationexp.AuthorityRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func reconstructSuccessfulPanel(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority) (successfulPanelAuthority, error) {
	authority := "development-public-v1"
	if panel == "validation" {
		authority = "validation-public-v1"
	} else if panel == "locked" {
		authority = lifecycle.AttemptCommitment
	}
	reportPath := actionrelationexp.ExpectedAuthorityPath(panel, "report")
	reportBytes, err := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(reportPath)), 0o644)
	if err != nil {
		return successfulPanelAuthority{}, err
	}
	report, err := actionrelationscore.ParseReport(reportBytes)
	if err != nil || report.Panel != panel || report.Authority != authority || report.Refs.PlanReview != prerequisites.PlanReviewRef || report.Refs.ImplementationReview != prerequisites.ImplementationRef || report.Refs.BuildAuthority != prerequisites.BuildRef || report.Refs.Competence != prerequisites.CompetenceRef || !sameOptionalRef(report.Refs.RunningReceipt, lifecycle.RunningRef) {
		return successfulPanelAuthority{}, fmt.Errorf("successful report authority changed")
	}
	reportRef, _ := actionrelationexp.Reference(reportPath, report.Canonical)
	receipt, err := actionrelationexp.BuildTerminalReceipt(actionrelationexp.TerminalReceipt{
		Panel: panel, State: "published", RunningReceipt: lifecycle.RunningRef, SourceRoot: prerequisites.Build.SourceRoot,
		FixtureRoot: report.Refs.FixtureRoot, AttemptCommitment: lifecycle.AttemptCommitment, Report: reportRef,
		EvidencePayload: report.Refs.EvidencePayload, Reason: report.Classification,
	})
	if err != nil {
		return successfulPanelAuthority{}, err
	}
	receiptPath := actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt")
	receiptRef, _ := actionrelationexp.Reference(receiptPath, receipt.Canonical)
	corePath := actionrelationexp.ExpectedAuthorityPath(panel, "execution-core")
	coreBytes, err := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(corePath)), 0o644)
	if err != nil {
		return successfulPanelAuthority{}, err
	}
	core, err := actionrelationexp.ParseExecutionCore(coreBytes)
	if err != nil {
		return successfulPanelAuthority{}, err
	}
	coreRef, _ := actionrelationexp.Reference(corePath, core.Canonical)
	publication, err := actionrelationexp.BuildPublication(actionrelationexp.Publication{
		Panel: panel, PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, ClaimReceipt: lifecycle.ClaimRef,
		RunningReceipt: lifecycle.RunningRef, PrimaryExecution: core.PrimaryExecution, AuditExecution: core.AuditExecution,
		AuditAttestation: core.AuditAttestation, RunEvidence: core.RunEvidence, StructuralMaps: slices.Clone(core.StructuralMaps),
		FixtureRoot: core.FixtureRoot, ExecutionCore: coreRef, EvidencePayload: report.Refs.EvidencePayload,
		Report: reportRef, TerminalReceipt: receiptRef,
	})
	if err != nil {
		return successfulPanelAuthority{}, fmt.Errorf("successful publication cannot reconstruct: %w", err)
	}
	if err := actionrelationexp.VerifyPublicationTerminal(publication, receipt); err != nil {
		return successfulPanelAuthority{}, fmt.Errorf("successful publication cannot reconstruct: %w", err)
	}
	return successfulPanelAuthority{Report: report, Receipt: receipt, Publication: publication}, nil
}

func verifyLocalSuccessfulPanel(prerequisites panelPrerequisites, authority successfulPanelAuthority, publicationPhysical string) error {
	publication := authority.Publication
	report := authority.Report
	readRef := func(ref actionrelationexp.AuthorityRef) ([]byte, error) {
		if ref.Verify() != nil {
			return nil, fmt.Errorf("invalid local authority reference")
		}
		data, err := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(ref.Path)), 0o644)
		if err != nil {
			return nil, err
		}
		want, err := actionrelationexp.Reference(ref.Path, data)
		if err != nil || want != ref {
			return nil, fmt.Errorf("local authority digest mismatch: %s", ref.Path)
		}
		return data, nil
	}
	refs := []actionrelationexp.AuthorityRef{
		publication.PrimaryExecution, publication.AuditExecution, publication.AuditAttestation, publication.RunEvidence,
		publication.FixtureRoot, publication.ExecutionCore, publication.EvidencePayload,
	}
	refs = append(refs, publication.StructuralMaps...)
	resolved := make(map[string][]byte, len(refs))
	for _, ref := range refs {
		data, err := readRef(ref)
		if err != nil {
			return err
		}
		resolved[ref.Path] = data
	}
	fixture, err := actionrelationfixture.ParsePanelFixture(resolved[publication.FixtureRoot.Path])
	if err != nil || fixture.Panel != publication.Panel || fixture.Authority != report.Authority {
		return fmt.Errorf("local fixture root does not reconstruct")
	}
	primary, err := actionrelationexp.ParseExecutionManifest(resolved[publication.PrimaryExecution.Path])
	if err != nil {
		return err
	}
	audit, err := actionrelationexp.ParseExecutionManifest(resolved[publication.AuditExecution.Path])
	if err != nil || actionrelationexp.EqualExecutionEvidence(primary, audit) != nil {
		return fmt.Errorf("local primary and audit execution differ")
	}
	attestation, err := actionrelationexp.ParseAuditAttestation(resolved[publication.AuditAttestation.Path])
	if err != nil {
		return err
	}
	core, err := actionrelationexp.ParseExecutionCore(resolved[publication.ExecutionCore.Path])
	if err != nil {
		return err
	}
	payload, err := actionrelationexp.ParseEvidencePayload(publication.Panel, report.Authority, resolved[publication.EvidencePayload.Path])
	if err != nil {
		return err
	}
	if primary.Panel != publication.Panel || primary.Authority != report.Authority || primary.SourceRoot != prerequisites.Build.SourceRoot || primary.BinaryDigest != prerequisites.Build.BinaryDigest || !slices.Equal(primary.Environment, competenceEnvironment) || primary.FixtureRoot != publication.FixtureRoot || primary.RunEvidence != publication.RunEvidence || !slices.Equal(primary.StructuralMaps, publication.StructuralMaps) || primary.Digest != publication.PrimaryExecution.Digest || audit.Digest != publication.AuditExecution.Digest || attestation.PrimaryExecution != publication.PrimaryExecution || attestation.AuditExecution != publication.AuditExecution || attestation.RunEvidence != publication.RunEvidence || !slices.Equal(attestation.StructuralMaps, publication.StructuralMaps) || attestation.RunIDsRoot != primary.RunIDsRoot || attestation.TranscriptRowsRoot != primary.TranscriptRowsRoot || attestation.ResultRowsRoot != primary.ResultRowsRoot || attestation.TotalRuns != primary.TotalRuns || core.SourceRoot != prerequisites.Build.SourceRoot || core.BinaryDigest != prerequisites.Build.BinaryDigest || !slices.Equal(core.Environment, competenceEnvironment) || !sameOptionalRef(core.RunningReceipt, publication.RunningReceipt) || core.PlanReview != prerequisites.PlanReviewRef || core.ImplementationReview != prerequisites.ImplementationRef || core.BuildAuthority != prerequisites.BuildRef || core.Competence != prerequisites.CompetenceRef || core.FixtureRoot != publication.FixtureRoot || core.PrimaryExecution != publication.PrimaryExecution || core.AuditExecution != publication.AuditExecution || core.AuditAttestation != publication.AuditAttestation || core.RunEvidence != publication.RunEvidence || !slices.Equal(core.StructuralMaps, publication.StructuralMaps) || payload.FixtureRoot != publication.FixtureRoot || payload.ExecutionCore != publication.ExecutionCore || payload.PlanReview != prerequisites.PlanReviewRef || payload.ImplementationReview != prerequisites.ImplementationRef || payload.BuildAuthority != prerequisites.BuildRef || payload.Competence != prerequisites.CompetenceRef || payload.AuditAttestation != publication.AuditAttestation || payload.RunEvidence != publication.RunEvidence || !slices.Equal(payload.StructuralMaps, publication.StructuralMaps) || payload.CurriculumRowsRoot != report.Refs.CurriculumRowsRoot {
		return fmt.Errorf("local successful authority DAG does not close")
	}
	readRetained := func(path string) ([]byte, error) {
		return readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(path)), 0o644)
	}
	reachable, err := actionrelationexp.VerifyRetainedPacks(actionrelationexp.RetainedPackRefs{
		Panel: publication.Panel, Authority: report.Authority, RunEvidence: publication.RunEvidence,
		ObjectRoots: payload.ObjectPackRoots, IndexRoots: payload.IndexRoots,
		JournalRoots: payload.JournalPackRoots, InputRoots: payload.InputPackRoots, DetailRoots: payload.DetailPackRoots,
		Tables: payload.AcquisitionTables, StructuralMaps: payload.StructuralMaps, StoreBoundaries: payload.StoreBoundaries,
	}, readRetained)
	if err != nil {
		return fmt.Errorf("local retained evidence DAG: %w", err)
	}
	reachable = append(reachable,
		publication.FixtureRoot.Path, publication.PrimaryExecution.Path, publication.AuditExecution.Path,
		publication.AuditAttestation.Path, publication.ExecutionCore.Path, publication.EvidencePayload.Path,
	)
	publicationLogical, err := filepath.Rel(prerequisites.Root, publicationPhysical)
	if err != nil {
		return err
	}
	reachable = append(reachable, filepath.ToSlash(publicationLogical))
	slices.Sort(reachable)
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(publication.Panel)
	physicalRoot := filepath.Join(prerequisites.Root, filepath.FromSlash(evidenceRoot))
	actual := make([]string, 0, len(reachable))
	err = filepath.WalkDir(physicalRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
			return fmt.Errorf("local evidence contains noncanonical file: %s", path)
		}
		relative, err := filepath.Rel(prerequisites.Root, path)
		if err != nil {
			return err
		}
		actual = append(actual, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(actual)
	if !slices.Equal(actual, reachable) {
		return fmt.Errorf("local evidence namespace has missing or unreachable files")
	}
	return nil
}
