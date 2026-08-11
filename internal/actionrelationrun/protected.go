package actionrelationrun

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/actionrelationcap"
	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationscore"
	"golang.org/x/sys/unix"
)

const (
	validationEvidenceCap = int64(3881 * 1024 * 1024)
	lockedEvidenceCap     = int64(5169 * 1024 * 1024)
)

func ExecuteProtected(ctx context.Context, repoRoot, panel string, argv []string) (actionrelationscore.Report, error) {
	if panel != "validation" && panel != "locked" {
		return actionrelationscore.Report{}, fmt.Errorf("protected execute requires validation or locked panel")
	}
	prerequisites, err := loadPanelPrerequisites(ctx, repoRoot)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := requireProtectedInvocation(prerequisites, panel, "execute", argv); err != nil {
		return actionrelationscore.Report{}, err
	}
	git, err := protectedGit(ctx, prerequisites.Root)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := requireProtectedProgression(prerequisites, git, panel); err != nil {
		return actionrelationscore.Report{}, err
	}
	capBytes := map[string]int64{"validation": validationEvidenceCap, "locked": lockedEvidenceCap}[panel]
	if err := requirePanelCapacity(prerequisites.Root, 2*capBytes); err != nil {
		return actionrelationscore.Report{}, err
	}
	lifecycle, inspectedClaim, inspectedRunning, err := loadProtectedLifecycle(prerequisites, git, panel)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	start, existing, err := inspectPanelStart(prerequisites, panel, lifecycle)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if existing {
		if err := eraseRecoverySecret(prerequisites, panel, inspectedClaim, inspectedRunning); err != nil {
			if terminalErr := publishInvalidPanel(prerequisites, panel, lifecycle, err); terminalErr != nil {
				return actionrelationscore.Report{}, fmt.Errorf("%v; retain invalid terminal: %w", err, terminalErr)
			}
			return actionrelationscore.Report{}, err
		}
		return recoverStartedPanel(prerequisites, panel, lifecycle)
	}
	if err := requireFreshPanelOutputsAbsent(prerequisites, panel, lifecycle); err != nil {
		return actionrelationscore.Report{}, err
	}
	token, claim, running, err := actionrelationcap.Authorize(ctx, prerequisites.Root, panel)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if !bytes.Equal(claim.Canonical, inspectedClaim.Canonical) || !bytes.Equal(running.Canonical, inspectedRunning.Canonical) {
		token.ReleaseForRetry()
		return actionrelationscore.Report{}, fmt.Errorf("protected authority changed during authorization")
	}
	fail := func(cause error) (actionrelationscore.Report, error) {
		terminalErr := publishInvalidPanel(prerequisites, panel, lifecycle, cause)
		if terminalErr != nil {
			return actionrelationscore.Report{}, fmt.Errorf("%v; retain invalid terminal: %w", cause, terminalErr)
		}
		return actionrelationscore.Report{}, cause
	}
	started, err := consumePanelStart(start)
	if err != nil {
		if !started {
			token.ReleaseForRetry()
			return actionrelationscore.Report{}, err
		}
		destroyErr := token.Destroy()
		return fail(errors.Join(err, destroyErr))
	}
	sealed, err := actionrelationscore.PrepareProtectedPanel(token)
	if err != nil {
		return fail(errors.Join(err, token.Destroy()))
	}
	if err := token.Destroy(); err != nil {
		return fail(err)
	}
	isolated, err := executeIsolatedPair(ctx, prerequisites, sealed, capBytes)
	if err != nil {
		return fail(err)
	}
	report, err := publishPanel(prerequisites, isolated.writer, isolated.summary, isolated.gates, lifecycle)
	if err != nil {
		var committed publishedCommitError
		if errors.As(err, &committed) {
			return actionrelationscore.Report{}, err
		}
		return fail(err)
	}
	return report, nil
}

func publishInvalidPanel(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority, cause error) error {
	publicationPath := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "publication")))
	if err := requireAbsentNoFollow(publicationPath); err != nil {
		return fmt.Errorf("invalid terminal cannot coexist with publication: %w", err)
	}
	reason, err := invalidPanelReason(prerequisites, panel, lifecycle, terminalReason(cause))
	if err != nil {
		return err
	}
	fixtureRef, err := ensureInvalidAuthorityRef(prerequisites, actionrelationexp.ExpectedAuthorityPath(panel, "fixture-root"), panel, "fixture-root", lifecycle.AttemptCommitment, reason)
	if err != nil {
		return err
	}
	payloadRef, err := ensureInvalidAuthorityRef(prerequisites, actionrelationexp.ExpectedAuthorityPath(panel, "evidence-payload"), panel, "evidence-payload", lifecycle.AttemptCommitment, reason)
	if err != nil {
		return err
	}
	reportRef, err := ensureInvalidAuthorityRef(prerequisites, actionrelationexp.ExpectedAuthorityPath(panel, "report"), panel, "report", lifecycle.AttemptCommitment, reason)
	if err != nil {
		return err
	}
	receipt, err := actionrelationexp.BuildTerminalReceipt(actionrelationexp.TerminalReceipt{
		Panel: panel, State: "invalid", RunningReceipt: lifecycle.RunningRef, SourceRoot: prerequisites.Build.SourceRoot,
		FixtureRoot: fixtureRef, AttemptCommitment: lifecycle.AttemptCommitment, Report: reportRef, EvidencePayload: payloadRef, Reason: reason,
	})
	if err != nil {
		return err
	}
	return writeExclusiveAuthority(filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt"))), receipt.Canonical)
}

func invalidPanelReason(prerequisites panelPrerequisites, panel string, lifecycle panelLifecycleAuthority, fallback string) (string, error) {
	reason := ""
	adopt := func(candidate string) error {
		if reason == "" {
			reason = candidate
			return nil
		}
		if reason != candidate {
			return fmt.Errorf("partial invalid authority reasons disagree")
		}
		return nil
	}
	for _, kind := range []string{"fixture-root", "evidence-payload", "report"} {
		path := actionrelationexp.ExpectedAuthorityPath(panel, kind)
		data, readErr := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(path)), 0o644)
		if errors.Is(readErr, unix.ENOENT) {
			continue
		}
		if readErr != nil {
			return "", readErr
		}
		failure, parseErr := actionrelationexp.ParseInvalidAuthority(data)
		if parseErr == nil {
			if failure.Panel != panel || failure.Kind != kind || failure.SourceRoot != prerequisites.Build.SourceRoot || failure.AttemptCommitment != lifecycle.AttemptCommitment {
				return "", fmt.Errorf("partial invalid authority tuple changed: %s", path)
			}
			if err := adopt(failure.Reason); err != nil {
				return "", err
			}
		}
	}
	receiptPath := actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt")
	data, readErr := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(receiptPath)), 0o644)
	if readErr == nil {
		receipt, parseErr := actionrelationexp.ParseTerminalReceipt(data)
		if parseErr != nil || receipt.Panel != panel || receipt.State != "invalid" || receipt.SourceRoot != prerequisites.Build.SourceRoot || receipt.AttemptCommitment != lifecycle.AttemptCommitment {
			return "", fmt.Errorf("existing terminal receipt is not the expected invalid authority")
		}
		if err := adopt(receipt.Reason); err != nil {
			return "", err
		}
	} else if !errors.Is(readErr, unix.ENOENT) {
		return "", readErr
	}
	if reason != "" {
		return reason, nil
	}
	return fallback, nil
}

func ensureInvalidAuthorityRef(prerequisites panelPrerequisites, path, panel, kind, attemptCommitment, reason string) (actionrelationexp.AuthorityRef, error) {
	physical := filepath.Join(prerequisites.Root, filepath.FromSlash(path))
	if data, err := readRegularNoFollow(physical, 0o644); err == nil {
		authority := "development-public-v1"
		if panel == "validation" {
			authority = "validation-public-v1"
		} else if panel == "locked" {
			authority = attemptCommitment
		}
		valid := false
		if failure, parseErr := actionrelationexp.ParseInvalidAuthority(data); parseErr == nil {
			valid = failure.Panel == panel && failure.Kind == kind && failure.SourceRoot == prerequisites.Build.SourceRoot && failure.AttemptCommitment == attemptCommitment && failure.Reason == reason
		} else {
			switch kind {
			case "fixture-root":
				fixture, parseErr := actionrelationfixture.ParsePanelFixture(data)
				valid = parseErr == nil && fixture.Panel == panel && fixture.Authority == authority
			case "evidence-payload":
				_, parseErr := actionrelationexp.ParseEvidencePayload(panel, authority, data)
				valid = parseErr == nil
			case "report":
				report, parseErr := actionrelationscore.ParseReport(data)
				valid = parseErr == nil && report.Panel == panel && report.Authority == authority
			}
		}
		if !valid {
			return actionrelationexp.AuthorityRef{}, fmt.Errorf("existing partial authority does not match successful or invalid tuple: %s", path)
		}
		return actionrelationexp.Reference(path, data)
	} else if !errors.Is(err, unix.ENOENT) {
		return actionrelationexp.AuthorityRef{}, err
	}
	failure, err := actionrelationexp.BuildInvalidAuthority(actionrelationexp.InvalidAuthority{Panel: panel, Kind: kind, SourceRoot: prerequisites.Build.SourceRoot, AttemptCommitment: attemptCommitment, Reason: reason})
	if err != nil {
		return actionrelationexp.AuthorityRef{}, err
	}
	if err := writeExclusiveAuthority(physical, failure.Canonical); err != nil {
		return actionrelationexp.AuthorityRef{}, err
	}
	return actionrelationexp.Reference(path, failure.Canonical)
}

func terminalReason(cause error) string {
	value := "panel execution failed"
	if cause != nil {
		value = cause.Error()
	}
	var result strings.Builder
	for _, char := range value {
		if char >= 0x20 && char <= 0x7e {
			result.WriteRune(char)
		} else {
			result.WriteByte('?')
		}
		if result.Len() >= 1000 {
			break
		}
	}
	if result.Len() == 0 {
		return "panel execution failed"
	}
	return result.String()
}

func ClaimProtected(ctx context.Context, repoRoot, panel string, argv []string) (actionrelationexp.Claim, error) {
	if panel != "validation" && panel != "locked" {
		return actionrelationexp.Claim{}, fmt.Errorf("claim requires validation or locked panel")
	}
	prerequisites, err := loadPanelPrerequisites(ctx, repoRoot)
	if err != nil {
		return actionrelationexp.Claim{}, err
	}
	if err := requireProtectedInvocation(prerequisites, panel, "claim", argv); err != nil {
		return actionrelationexp.Claim{}, err
	}
	git, err := protectedGit(ctx, prerequisites.Root)
	if err != nil {
		return actionrelationexp.Claim{}, err
	}
	if err := requireProtectedProgression(prerequisites, git, panel); err != nil {
		return actionrelationexp.Claim{}, err
	}
	if err := requireProtectedOutputsAbsent(prerequisites.Root, panel, false); err != nil {
		return actionrelationexp.Claim{}, err
	}
	authority := "validation-public-v1"
	if panel == "locked" {
		authority = actionrelationcap.LockedClaimAuthority(prerequisites.Head, prerequisites.Build.SourceRoot)
	}
	claim, err := actionrelationexp.BuildClaim(actionrelationexp.Claim{
		Panel: panel, BaseCommit: prerequisites.Head, SourceRoot: prerequisites.Build.SourceRoot, Authority: authority,
	})
	if err != nil {
		return actionrelationexp.Claim{}, err
	}
	path := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "claim")))
	if err := writeExclusiveAuthority(path, claim.Canonical); err != nil {
		return actionrelationexp.Claim{}, err
	}
	return claim, nil
}

func PrepareProtected(ctx context.Context, repoRoot, panel string, argv []string) (actionrelationexp.Running, error) {
	if panel != "validation" && panel != "locked" {
		return actionrelationexp.Running{}, fmt.Errorf("protected prepare requires validation or locked panel")
	}
	prerequisites, err := loadPanelPrerequisites(ctx, repoRoot)
	if err != nil {
		return actionrelationexp.Running{}, err
	}
	if err := requireProtectedInvocation(prerequisites, panel, "prepare", argv); err != nil {
		return actionrelationexp.Running{}, err
	}
	if err := requireProtectedOutputsAbsent(prerequisites.Root, panel, true); err != nil {
		return actionrelationexp.Running{}, err
	}
	git, err := protectedGit(ctx, prerequisites.Root)
	if err != nil {
		return actionrelationexp.Running{}, err
	}
	if err := requireProtectedProgression(prerequisites, git, panel); err != nil {
		return actionrelationexp.Running{}, err
	}
	claimPath := actionrelationexp.ExpectedAuthorityPath(panel, "claim")
	claimBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, claimPath)
	if err != nil {
		return actionrelationexp.Running{}, err
	}
	claim, err := actionrelationexp.ParseClaim(claimBytes)
	if err != nil || claim.Panel != panel || claim.SourceRoot != prerequisites.Build.SourceRoot || panel == "validation" && claim.Authority != "validation-public-v1" || panel == "locked" && claim.Authority != actionrelationcap.LockedClaimAuthority(claim.BaseCommit, claim.SourceRoot) {
		return actionrelationexp.Running{}, fmt.Errorf("committed claim does not match protected panel authority")
	}
	if claim.BaseCommit == prerequisites.Head {
		return actionrelationexp.Running{}, fmt.Errorf("claim has not been committed after its base")
	}
	if _, err := git("merge-base", "--is-ancestor", claim.BaseCommit, prerequisites.Head); err != nil {
		return actionrelationexp.Running{}, fmt.Errorf("claim base is not an ancestor of claim commit")
	}
	running := actionrelationexp.Running{
		Panel: panel, ClaimReceiptDigest: claim.Digest, ClaimCommit: prerequisites.Head,
		SourceRoot: prerequisites.Build.SourceRoot, AttemptCommitment: actionrelationcap.ValidationAttemptCommitment(),
	}
	if panel == "locked" {
		secret := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, secret); err != nil {
			return actionrelationexp.Running{}, fmt.Errorf("select locked attempt root: %w", err)
		}
		running.AttemptCommitment = digest(secret)
		location, locationDigest, err := actionrelationcap.LockedSecretLocation(claim.Digest)
		if err != nil {
			return actionrelationexp.Running{}, err
		}
		running.SecretLocationDigest = &locationDigest
		commonDir, err := protectedGitCommonDir(git, prerequisites.Root)
		if err != nil {
			return actionrelationexp.Running{}, err
		}
		secretPath := filepath.Join(commonDir, filepath.FromSlash(location))
		if err := writeExclusiveSyncedMode(secretPath, secret, 0o600); err != nil {
			return actionrelationexp.Running{}, fmt.Errorf("persist locked attempt root: %w", err)
		}
		readback, err := readRegularNoFollow(secretPath, 0o600)
		info, statErr := os.Lstat(secretPath)
		if err != nil || statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !slices.Equal(readback, secret) {
			return actionrelationexp.Running{}, fmt.Errorf("locked attempt root readback mismatch")
		}
	}
	running, err = actionrelationexp.BuildRunning(running)
	if err != nil {
		return actionrelationexp.Running{}, err
	}
	path := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, "running")))
	if err := writeExclusiveAuthority(path, running.Canonical); err != nil {
		return actionrelationexp.Running{}, err
	}
	return running, nil
}

func requireProtectedInvocation(prerequisites panelPrerequisites, panel, stage string, argv []string) error {
	binary := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	want := []string{binary, "actionrelation-trials", "-stage", stage, "-panel", panel, "-repo-root", prerequisites.Root}
	if panel == "locked" && stage == "execute" {
		want = append(want, "-unlock-token", "actionrelations/v1:"+prerequisites.Head)
	}
	if !slices.Equal(argv, want) || !exactProcessEnvironment(competenceEnvironment) {
		return fmt.Errorf("noncanonical protected %s invocation", stage)
	}
	return nil
}

func requireProtectedProgression(prerequisites panelPrerequisites, git func(...string) ([]byte, error), panel string) error {
	prior := []struct {
		panel          string
		classification string
	}{{panel: "development", classification: "interim-power-authorized"}}
	if panel == "locked" {
		prior = append(prior, struct {
			panel          string
			classification string
		}{panel: "validation", classification: "interim-valid"})
	}
	for _, gate := range prior {
		if err := verifyCommittedPanelOutcome(prerequisites, git, gate.panel, gate.classification); err != nil {
			return fmt.Errorf("%s progression gate: %w", gate.panel, err)
		}
	}
	return nil
}

func verifyCommittedPanelOutcome(prerequisites panelPrerequisites, git func(...string) ([]byte, error), panel, classification string) error {
	reportPath := actionrelationexp.ExpectedAuthorityPath(panel, "report")
	reportBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, reportPath)
	if err != nil {
		return err
	}
	report, err := actionrelationscore.ParseReport(reportBytes)
	if err != nil || report.Panel != panel || report.Classification != classification {
		return fmt.Errorf("prior report is not an authorized %s result", classification)
	}
	receiptPath := actionrelationexp.ExpectedAuthorityPath(panel, "terminal-receipt")
	receiptBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, receiptPath)
	if err != nil {
		return err
	}
	receipt, err := actionrelationexp.ParseTerminalReceipt(receiptBytes)
	if err != nil || receipt.Panel != panel || receipt.State != "published" || receipt.SourceRoot != prerequisites.Build.SourceRoot || receipt.Reason != classification {
		return fmt.Errorf("prior terminal receipt is not published authority")
	}
	publicationPath := actionrelationexp.ExpectedAuthorityPath(panel, "publication")
	publicationBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, publicationPath)
	if err != nil {
		return err
	}
	publication, err := actionrelationexp.ParsePublication(panel, publicationBytes)
	if err != nil {
		return err
	}
	if err := actionrelationexp.VerifyPublicationTerminal(publication, receipt); err != nil {
		return err
	}
	wantReportRef, _ := actionrelationexp.Reference(reportPath, report.Canonical)
	wantReceiptRef, _ := actionrelationexp.Reference(receiptPath, receipt.Canonical)
	if publication.PlanReview != prerequisites.PlanReviewRef || publication.ImplementationReview != prerequisites.ImplementationRef || publication.BuildAuthority != prerequisites.BuildRef || publication.Competence != prerequisites.CompetenceRef || publication.Report != wantReportRef || publication.TerminalReceipt != wantReceiptRef || report.Refs.PlanReview != prerequisites.PlanReviewRef || report.Refs.ImplementationReview != prerequisites.ImplementationRef || report.Refs.BuildAuthority != prerequisites.BuildRef || report.Refs.Competence != prerequisites.CompetenceRef || report.Refs.FixtureRoot != publication.FixtureRoot || report.Refs.EvidencePayload != publication.EvidencePayload || receipt.FixtureRoot != publication.FixtureRoot || receipt.Report != publication.Report || receipt.EvidencePayload != publication.EvidencePayload {
		return fmt.Errorf("prior publication does not close prerequisite and report authority")
	}
	if panel == "development" {
		if publication.ClaimReceipt != nil || publication.RunningReceipt != nil || report.Refs.RunningReceipt != nil || receipt.RunningReceipt != nil || receipt.AttemptCommitment != strings.Repeat("0", 64) {
			return fmt.Errorf("development progression contains protected authority")
		}
	} else {
		if publication.ClaimReceipt == nil || publication.RunningReceipt == nil || report.Refs.RunningReceipt == nil || receipt.RunningReceipt == nil || *publication.RunningReceipt != *report.Refs.RunningReceipt || *publication.RunningReceipt != *receipt.RunningReceipt {
			return fmt.Errorf("protected progression lacks identical running authority")
		}
		claimBytes, err := verifyCommittedReference(prerequisites, git, *publication.ClaimReceipt)
		if err != nil {
			return err
		}
		claim, err := actionrelationexp.ParseClaim(claimBytes)
		if err != nil || claim.Panel != panel || claim.SourceRoot != prerequisites.Build.SourceRoot {
			return fmt.Errorf("prior claim authority changed")
		}
		runningBytes, err := verifyCommittedReference(prerequisites, git, *publication.RunningReceipt)
		if err != nil {
			return err
		}
		running, err := actionrelationexp.ParseRunning(runningBytes)
		if err != nil || running.Panel != panel || running.SourceRoot != prerequisites.Build.SourceRoot || running.ClaimReceiptDigest != claim.Digest || receipt.AttemptCommitment != running.AttemptCommitment {
			return fmt.Errorf("prior running authority changed")
		}
	}
	refs := []actionrelationexp.AuthorityRef{
		publication.PrimaryExecution, publication.AuditExecution, publication.AuditAttestation, publication.RunEvidence,
		publication.FixtureRoot, publication.ExecutionCore, publication.EvidencePayload,
	}
	refs = append(refs, publication.StructuralMaps...)
	resolved := make(map[string][]byte, len(refs))
	for _, ref := range refs {
		data, err := verifyCommittedReference(prerequisites, git, ref)
		if err != nil {
			return err
		}
		resolved[ref.Path] = data
	}
	fixture, err := actionrelationfixture.ParsePanelFixture(resolved[publication.FixtureRoot.Path])
	if err != nil || fixture.Panel != panel || fixture.Authority != report.Authority {
		return fmt.Errorf("prior fixture root does not reconstruct")
	}
	primary, err := actionrelationexp.ParseExecutionManifest(resolved[publication.PrimaryExecution.Path])
	if err != nil {
		return err
	}
	audit, err := actionrelationexp.ParseExecutionManifest(resolved[publication.AuditExecution.Path])
	if err != nil || actionrelationexp.EqualExecutionEvidence(primary, audit) != nil {
		return fmt.Errorf("prior primary and audit execution differ")
	}
	attestation, err := actionrelationexp.ParseAuditAttestation(resolved[publication.AuditAttestation.Path])
	if err != nil {
		return err
	}
	core, err := actionrelationexp.ParseExecutionCore(resolved[publication.ExecutionCore.Path])
	if err != nil {
		return err
	}
	payload, err := actionrelationexp.ParseEvidencePayload(panel, report.Authority, resolved[publication.EvidencePayload.Path])
	if err != nil {
		return err
	}
	if primary.Panel != panel || primary.Authority != report.Authority || primary.SourceRoot != prerequisites.Build.SourceRoot || primary.BinaryDigest != prerequisites.Build.BinaryDigest || !slices.Equal(primary.Environment, competenceEnvironment) || primary.FixtureRoot != publication.FixtureRoot || primary.RunEvidence != publication.RunEvidence || !slices.Equal(primary.StructuralMaps, publication.StructuralMaps) || audit.Digest != publication.AuditExecution.Digest || primary.Digest != publication.PrimaryExecution.Digest || attestation.PrimaryExecution != publication.PrimaryExecution || attestation.AuditExecution != publication.AuditExecution || attestation.RunEvidence != publication.RunEvidence || !slices.Equal(attestation.StructuralMaps, publication.StructuralMaps) || attestation.RunIDsRoot != primary.RunIDsRoot || attestation.TranscriptRowsRoot != primary.TranscriptRowsRoot || attestation.ResultRowsRoot != primary.ResultRowsRoot || attestation.TotalRuns != primary.TotalRuns || core.SourceRoot != prerequisites.Build.SourceRoot || core.BinaryDigest != prerequisites.Build.BinaryDigest || !slices.Equal(core.Environment, competenceEnvironment) || core.PlanReview != prerequisites.PlanReviewRef || core.ImplementationReview != prerequisites.ImplementationRef || core.BuildAuthority != prerequisites.BuildRef || core.Competence != prerequisites.CompetenceRef || core.FixtureRoot != publication.FixtureRoot || core.PrimaryExecution != publication.PrimaryExecution || core.AuditExecution != publication.AuditExecution || core.AuditAttestation != publication.AuditAttestation || core.RunEvidence != publication.RunEvidence || !slices.Equal(core.StructuralMaps, publication.StructuralMaps) || payload.FixtureRoot != publication.FixtureRoot || payload.ExecutionCore != publication.ExecutionCore || payload.PlanReview != prerequisites.PlanReviewRef || payload.ImplementationReview != prerequisites.ImplementationRef || payload.BuildAuthority != prerequisites.BuildRef || payload.Competence != prerequisites.CompetenceRef || payload.AuditAttestation != publication.AuditAttestation || payload.RunEvidence != publication.RunEvidence || !slices.Equal(payload.StructuralMaps, publication.StructuralMaps) || payload.CurriculumRowsRoot != report.Refs.CurriculumRowsRoot {
		return fmt.Errorf("prior execution authority DAG does not close")
	}
	if panel == "development" {
		if core.RunningReceipt != nil {
			return fmt.Errorf("development core contains running authority")
		}
	} else if core.RunningReceipt == nil || publication.RunningReceipt == nil || *core.RunningReceipt != *publication.RunningReceipt {
		return fmt.Errorf("protected core running authority changed")
	}
	runPackPath := strings.TrimSuffix(publication.RunEvidence.Path, "/manifests/run-evidence-root.json") + "/packs/run-evidence-0000.arrv"
	runPackBytes, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, runPackPath)
	if err != nil {
		return err
	}
	runPack, err := actionrelationexp.ParseRunEvidencePack(panel, report.Authority, resolved[publication.RunEvidence.Path], runPackBytes)
	if err != nil || runPack.RunIDsRoot != primary.RunIDsRoot || runPack.TranscriptRowsRoot != primary.TranscriptRowsRoot || runPack.ResultRowsRoot != primary.ResultRowsRoot {
		return fmt.Errorf("prior run evidence does not close execution roots")
	}
	readRetained := func(path string) ([]byte, error) {
		return readCommittedWorking(prerequisites.Root, git, prerequisites.Head, path)
	}
	reachable, err := actionrelationexp.VerifyRetainedPacks(actionrelationexp.RetainedPackRefs{
		Panel: panel, Authority: report.Authority, Fixture: publication.FixtureRoot, RunEvidence: publication.RunEvidence,
		ObjectRoots: payload.ObjectPackRoots, IndexRoots: payload.IndexRoots,
		JournalRoots: payload.JournalPackRoots, InputRoots: payload.InputPackRoots, DetailRoots: payload.DetailPackRoots,
		Tables: payload.AcquisitionTables, StructuralMaps: payload.StructuralMaps, StoreBoundaries: payload.StoreBoundaries,
	}, readRetained)
	if err != nil {
		return fmt.Errorf("prior retained evidence DAG: %w", err)
	}
	authorityRoot := strings.TrimSuffix(publication.RunEvidence.Path, "/manifests/run-evidence-root.json") + "/authority"
	reachable = append(reachable,
		publication.PrimaryExecution.Path, publication.AuditExecution.Path,
		publication.AuditAttestation.Path, publication.ExecutionCore.Path, publication.EvidencePayload.Path,
		authorityRoot+"/publication.json",
	)
	slices.Sort(reachable)
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(panel)
	tree, err := git("ls-tree", "-rz", "--name-only", prerequisites.Head, "--", evidenceRoot)
	if err != nil {
		return err
	}
	committedPaths := splitNULTerminated(tree)
	if !slices.Equal(committedPaths, reachable) {
		return fmt.Errorf("prior evidence namespace has missing or unreachable files")
	}
	return nil
}

func splitNULTerminated(data []byte) []string {
	if len(data) == 0 || data[len(data)-1] != 0 {
		return nil
	}
	parts := bytes.Split(data[:len(data)-1], []byte{0})
	result := make([]string, len(parts))
	for index, part := range parts {
		result[index] = string(part)
	}
	slices.Sort(result)
	return result
}

func verifyCommittedReference(prerequisites panelPrerequisites, git func(...string) ([]byte, error), ref actionrelationexp.AuthorityRef) ([]byte, error) {
	if ref.Verify() != nil {
		return nil, fmt.Errorf("invalid committed authority reference")
	}
	data, err := readCommittedWorking(prerequisites.Root, git, prerequisites.Head, ref.Path)
	if err != nil {
		return nil, err
	}
	want, err := actionrelationexp.Reference(ref.Path, data)
	if err != nil || want != ref {
		return nil, fmt.Errorf("committed authority digest mismatch: %s", ref.Path)
	}
	return data, nil
}

func requireProtectedOutputsAbsent(root, panel string, keepClaim bool) error {
	kinds := []string{"claim", "running", "terminal-receipt", "report"}
	for _, kind := range kinds {
		if keepClaim && kind == "claim" {
			continue
		}
		path := actionrelationexp.ExpectedAuthorityPath(panel, kind)
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); err == nil {
			return fmt.Errorf("protected authority already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(panel)
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(evidenceRoot))); err == nil {
		return fmt.Errorf("protected evidence namespace already exists: %s", evidenceRoot)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func protectedGit(ctx context.Context, root string) (func(...string) ([]byte, error), error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	return func(args ...string) ([]byte, error) {
		command := exec.CommandContext(ctx, gitPath, append([]string{"-C", root}, args...)...)
		command.Env = []string{"GIT_ATTR_NOSYSTEM=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "TZ=UTC"}
		return command.Output()
	}, nil
}

func protectedGitCommonDir(git func(...string) ([]byte, error), root string) (string, error) {
	encoded, err := git("rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	directory := strings.TrimSpace(string(encoded))
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(root, directory)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func writeExclusiveSyncedMode(path string, data []byte, mode os.FileMode) error {
	directoryMode := os.FileMode(0o755)
	if mode.Perm() == 0o600 {
		directoryMode = 0o700
	}
	return writeExclusiveNoFollow(path, data, mode, directoryMode)
}
