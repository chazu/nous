package actionrelationrun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"

	"github.com/chazu/nous/internal/actionrelationcompetence"
	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationscore"
)

const developmentEvidenceCap = int64(2593 * 1024 * 1024)

type panelPrerequisites struct {
	Root                 string
	GitCommonDir         string
	Head                 string
	Build                actionrelationexp.BuildAuthority
	Competence           actionrelationcompetence.Root
	PlanReview           actionrelationexp.ReviewManifest
	ImplementationReview actionrelationexp.ReviewManifest
	PlanReviewRef        actionrelationexp.AuthorityRef
	ImplementationRef    actionrelationexp.AuthorityRef
	BuildRef             actionrelationexp.AuthorityRef
	CompetenceRef        actionrelationexp.AuthorityRef
}

type retainedFile struct {
	Path   string
	Bytes  int64
	Digest string
}

type panelWriter struct {
	physicalRoot string
	panelRoot    string
	cap          int64
	total        int64
	files        map[string]retainedFile
}

// ExecuteDevelopment is the sole complete development-panel caller. It is
// reachable only from the exact reviewed binary invocation after committed
// build and competence prerequisites.
func ExecuteDevelopment(ctx context.Context, repoRoot string, argv []string) (actionrelationscore.Report, error) {
	prerequisites, err := loadPanelPrerequisites(ctx, repoRoot)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	binary := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	wantArgv := []string{binary, "actionrelation-trials", "-stage", "execute", "-panel", "development", "-repo-root", prerequisites.Root}
	if !slices.Equal(argv, wantArgv) || !exactProcessEnvironment(competenceEnvironment) {
		return actionrelationscore.Report{}, fmt.Errorf("noncanonical development invocation")
	}
	if err := requirePanelCapacity(prerequisites.Root, 2*developmentEvidenceCap); err != nil {
		return actionrelationscore.Report{}, err
	}
	lifecycle := panelLifecycleAuthority{AttemptCommitment: strings.Repeat("0", 64)}
	start, existing, err := inspectPanelStart(prerequisites, "development", lifecycle)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if existing {
		return recoverStartedPanel(prerequisites, "development", lifecycle)
	}
	if err := requireFreshPanelOutputsAbsent(prerequisites, "development", lifecycle); err != nil {
		return actionrelationscore.Report{}, err
	}
	fail := func(cause error) (actionrelationscore.Report, error) {
		terminalErr := publishInvalidPanel(prerequisites, "development", lifecycle, cause)
		if terminalErr != nil {
			return actionrelationscore.Report{}, fmt.Errorf("%v; retain invalid terminal: %w", cause, terminalErr)
		}
		return actionrelationscore.Report{}, cause
	}
	started, err := consumePanelStart(start)
	if err != nil {
		if started {
			return fail(err)
		}
		return actionrelationscore.Report{}, err
	}
	sealed, err := actionrelationscore.PrepareDevelopmentPanel()
	if err != nil {
		return fail(err)
	}
	isolated, err := executeIsolatedPair(ctx, prerequisites, sealed, developmentEvidenceCap)
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

type panelLifecycleAuthority struct {
	ClaimRef          *actionrelationexp.AuthorityRef
	RunningRef        *actionrelationexp.AuthorityRef
	AttemptCommitment string
}

type publishedCommitError struct {
	err error
}

func (e publishedCommitError) Error() string {
	return "published terminal requires recovery: " + e.err.Error()
}
func (e publishedCommitError) Unwrap() error { return e.err }

func publishPanel(prerequisites panelPrerequisites, writer *panelWriter, summary actionrelationscore.PanelSummary, gates actionrelationscore.MechanicalGates, lifecycle panelLifecycleAuthority) (actionrelationscore.Report, error) {
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(summary.Panel)
	authorityRoot := evidenceRoot + "/authority"
	fixtureRef, _ := actionrelationexp.Reference(authorityRoot+"/fixture-root.json", summary.Fixture.Canonical)
	runEvidenceRef, _ := actionrelationexp.Reference(evidenceRoot+"/manifests/run-evidence-root.json", summary.RunEvidence.Canonical)
	primary, err := actionrelationexp.BuildExecutionManifest(actionrelationexp.ExecutionManifest{
		Role: "primary", Panel: summary.Panel, Authority: summary.Authority, SourceRoot: prerequisites.Build.SourceRoot,
		BinaryDigest: prerequisites.Build.BinaryDigest, Environment: slices.Clone(competenceEnvironment), FixtureRoot: fixtureRef,
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps, RunIDsRoot: summary.RunEvidence.RunIDsRoot,
		TranscriptRowsRoot: summary.RunEvidence.TranscriptRowsRoot, ResultRowsRoot: summary.RunEvidence.ResultRowsRoot, TotalRuns: len(summary.RunEvidence.Records),
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	primaryPath := authorityRoot + "/execution-primary.json"
	primaryRef, _ := actionrelationexp.Reference(primaryPath, primary.Canonical)
	audit, err := actionrelationexp.BuildExecutionManifest(actionrelationexp.ExecutionManifest{
		Role: "audit", Panel: summary.Panel, Authority: summary.Authority, SourceRoot: prerequisites.Build.SourceRoot,
		BinaryDigest: prerequisites.Build.BinaryDigest, Environment: slices.Clone(competenceEnvironment), FixtureRoot: fixtureRef,
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps, RunIDsRoot: summary.RunEvidence.RunIDsRoot,
		TranscriptRowsRoot: summary.RunEvidence.TranscriptRowsRoot, ResultRowsRoot: summary.RunEvidence.ResultRowsRoot,
		TotalRuns: len(summary.RunEvidence.Records), PriorExecution: &primaryRef,
	})
	if err != nil {
		return actionrelationscore.Report{}, fmt.Errorf("build audit execution: %w", err)
	}
	if err := actionrelationexp.EqualExecutionEvidence(primary, audit); err != nil {
		return actionrelationscore.Report{}, fmt.Errorf("audit execution differs from primary: %w", err)
	}
	auditPath := authorityRoot + "/execution-audit.json"
	auditRef, _ := actionrelationexp.Reference(auditPath, audit.Canonical)
	attestation, err := actionrelationexp.BuildAuditAttestation(actionrelationexp.AuditAttestation{
		Panel: summary.Panel, Authority: summary.Authority, PrimaryExecution: primaryRef, AuditExecution: auditRef,
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps, RunIDsRoot: summary.RunEvidence.RunIDsRoot,
		TranscriptRowsRoot: summary.RunEvidence.TranscriptRowsRoot, ResultRowsRoot: summary.RunEvidence.ResultRowsRoot, TotalRuns: len(summary.RunEvidence.Records),
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	attestationPath := authorityRoot + "/audit-attestation.json"
	attestationRef, _ := actionrelationexp.Reference(attestationPath, attestation.Canonical)
	core, err := actionrelationexp.BuildExecutionCore(actionrelationexp.ExecutionCore{
		Panel: summary.Panel, Authority: summary.Authority, SourceRoot: prerequisites.Build.SourceRoot, BinaryDigest: prerequisites.Build.BinaryDigest,
		PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, Environment: slices.Clone(competenceEnvironment),
		FixtureRoot: fixtureRef, PrimaryExecution: primaryRef, AuditExecution: auditRef, AuditAttestation: attestationRef,
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps, RunningReceipt: lifecycle.RunningRef,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	corePath := authorityRoot + "/execution-core.json"
	coreRef, _ := actionrelationexp.Reference(corePath, core.Canonical)
	payload, err := actionrelationexp.BuildEvidencePayload(actionrelationexp.EvidencePayload{
		Panel: summary.Panel, Authority: summary.Authority, FixtureRoot: fixtureRef, ExecutionCore: coreRef,
		PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, AuditAttestation: attestationRef,
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps, StoreBoundaries: summary.StoreBoundaries,
		ObjectPackRoots: summary.ObjectRoots, JournalPackRoots: summary.JournalRoots, InputPackRoots: summary.InputRoots,
		DetailPackRoots: summary.DetailRoots, AcquisitionTables: summary.Tables, IndexRoots: summary.IndexRoots,
		WorldPolicyRowsRoot: summary.WorldPolicyRowsRoot, CurriculumRowsRoot: summary.CurriculumRowsRoot,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	payloadPath := authorityRoot + "/evidence-payload.json"
	payloadRef, _ := actionrelationexp.Reference(payloadPath, payload.Canonical)
	inference, err := actionrelationscore.Infer(summary.Panel, summary.Authority, summary.WorldRows, summary.CurriculumRows)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	report, err := actionrelationscore.BuildReport(summary.Panel, summary.Authority, actionrelationscore.ReportAuthority{
		PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, FixtureRoot: fixtureRef,
		RunningReceipt: lifecycle.RunningRef, CurriculumRowsRoot: summary.CurriculumRowsRoot, EvidencePayload: payloadRef,
	}, gates, inference)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	reportPath := actionrelationexp.ExpectedAuthorityPath(summary.Panel, "report")
	reportRef, _ := actionrelationexp.Reference(reportPath, report.Canonical)
	receipt, err := actionrelationexp.BuildTerminalReceipt(actionrelationexp.TerminalReceipt{
		Panel: summary.Panel, State: "published", SourceRoot: prerequisites.Build.SourceRoot, FixtureRoot: fixtureRef,
		RunningReceipt: lifecycle.RunningRef, AttemptCommitment: lifecycle.AttemptCommitment, Report: reportRef, EvidencePayload: payloadRef, Reason: report.Classification,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	receiptPath := actionrelationexp.ExpectedAuthorityPath(summary.Panel, "terminal-receipt")
	receiptRef, _ := actionrelationexp.Reference(receiptPath, receipt.Canonical)
	publication, err := actionrelationexp.BuildPublication(actionrelationexp.Publication{
		Panel: summary.Panel, PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, PrimaryExecution: primaryRef,
		AuditExecution: auditRef, AuditAttestation: attestationRef, RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps,
		ClaimReceipt: lifecycle.ClaimRef, RunningReceipt: lifecycle.RunningRef, FixtureRoot: fixtureRef, ExecutionCore: coreRef,
		EvidencePayload: payloadRef, Report: reportRef, TerminalReceipt: receiptRef,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := actionrelationexp.VerifyPublicationTerminal(publication, receipt); err != nil {
		return actionrelationscore.Report{}, err
	}
	preexistingBytes, err := retainedLifecycleBytes(prerequisites.Root, lifecycle)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if len(report.Canonical) > 14*1024*1024 || len(primary.Canonical) > 32_768 || len(audit.Canonical) > 32_768 || len(attestation.Canonical) > 8_192 || len(core.Canonical) > 8_192 || len(payload.Canonical) > 2_097_152 || len(receipt.Canonical) > 8_192 || len(publication.Canonical) > 8_192 {
		return actionrelationscore.Report{}, fmt.Errorf("panel authority file exceeds its frozen byte cap")
	}
	newAuthorityBytes := int64(len(primary.Canonical) + len(audit.Canonical) + len(attestation.Canonical) + len(core.Canonical) + len(payload.Canonical) + len(report.Canonical) + len(receipt.Canonical) + len(publication.Canonical))
	if writer.total+preexistingBytes+newAuthorityBytes > writer.cap {
		return actionrelationscore.Report{}, fmt.Errorf("complete published panel exceeds capacity")
	}
	for _, file := range []actionrelationexp.EvidenceFile{
		{Path: primaryPath, Mode: "100644", Data: primary.Canonical},
		{Path: auditPath, Mode: "100644", Data: audit.Canonical},
		{Path: attestationPath, Mode: "100644", Data: attestation.Canonical},
		{Path: corePath, Mode: "100644", Data: core.Canonical},
		{Path: payloadPath, Mode: "100644", Data: payload.Canonical},
	} {
		if err := writer.write(file); err != nil {
			return actionrelationscore.Report{}, err
		}
	}
	if err := writeExclusiveAuthority(filepath.Join(prerequisites.Root, filepath.FromSlash(reportPath)), report.Canonical); err != nil {
		return actionrelationscore.Report{}, err
	}
	publicationPath := authorityRoot + "/publication.json"
	start, err := panelStartAuthority(prerequisites, summary.Panel, lifecycle)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	receiptPhysical := filepath.Join(prerequisites.Root, filepath.FromSlash(receiptPath))
	publicationPhysical := filepath.Join(prerequisites.Root, filepath.FromSlash(publicationPath))
	receiptPending := receiptPhysical + ".pending-" + start.Identity
	publicationPending := publicationPhysical + ".pending-" + start.Identity
	if err := writeExclusiveAuthority(receiptPending, receipt.Canonical); err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := writeExclusiveAuthority(publicationPending, publication.Canonical); err != nil {
		_ = removeExpectedNoFollow(receiptPending, receipt.Canonical, 0o644)
		_ = removeExpectedNoFollow(publicationPending, publication.Canonical, 0o644)
		return actionrelationscore.Report{}, err
	}
	committed, receiptErr := linkStagedNoFollow(receiptPending, receiptPhysical, receipt.Canonical, 0o644)
	if !committed {
		_ = removeExpectedNoFollow(receiptPending, receipt.Canonical, 0o644)
		_ = removeExpectedNoFollow(publicationPending, publication.Canonical, 0o644)
		return actionrelationscore.Report{}, receiptErr
	}
	_, publicationErr := linkStagedNoFollow(publicationPending, publicationPhysical, publication.Canonical, 0o644)
	if joined := errors.Join(receiptErr, publicationErr); joined != nil {
		return actionrelationscore.Report{}, publishedCommitError{err: joined}
	}
	writer.total += preexistingBytes + int64(len(report.Canonical)+len(receipt.Canonical)+len(publication.Canonical))
	writer.files[publicationPath] = retainedFile{Path: publicationPhysical, Bytes: int64(len(publication.Canonical)), Digest: digest(publication.Canonical)}
	return report, nil
}

func retainedLifecycleBytes(root string, lifecycle panelLifecycleAuthority) (int64, error) {
	var total int64
	for _, ref := range []*actionrelationexp.AuthorityRef{lifecycle.ClaimRef, lifecycle.RunningRef} {
		if ref == nil {
			continue
		}
		data, err := readRegularNoFollow(filepath.Join(root, filepath.FromSlash(ref.Path)), 0o644)
		if err != nil || ref.Verify() != nil || digest(data) != ref.Digest || len(data) > 4_096 {
			return 0, fmt.Errorf("panel lifecycle authority differs from frozen cap")
		}
		total += int64(len(data))
	}
	return total, nil
}

func loadPanelPrerequisites(ctx context.Context, repoRoot string) (panelPrerequisites, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return panelPrerequisites{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return panelPrerequisites{}, err
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return panelPrerequisites{}, err
	}
	git := func(args ...string) ([]byte, error) {
		command := exec.CommandContext(ctx, gitPath, append([]string{"-C", root}, args...)...)
		command.Env = []string{"GIT_ATTR_NOSYSTEM=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "TZ=UTC"}
		return command.Output()
	}
	headBytes, err := git("rev-parse", "HEAD")
	if err != nil {
		return panelPrerequisites{}, err
	}
	head := strings.TrimSpace(string(headBytes))
	originBytes, err := git("rev-parse", "origin/main")
	branchBytes, branchErr := git("symbolic-ref", "--short", "HEAD")
	if err != nil || branchErr != nil || head != strings.TrimSpace(string(originBytes)) || strings.TrimSpace(string(branchBytes)) != "main" {
		return panelPrerequisites{}, fmt.Errorf("panel requires clean HEAD at origin/main")
	}
	if _, err := git("diff", "--quiet", "HEAD", "--"); err != nil {
		return panelPrerequisites{}, fmt.Errorf("panel requires no tracked working-tree changes")
	}
	if _, err := git("diff", "--cached", "--quiet", "HEAD", "--"); err != nil {
		return panelPrerequisites{}, fmt.Errorf("panel requires no staged changes")
	}
	buildBytes, err := readCommittedWorking(root, git, head, actionrelationexp.BuildAuthorityPath)
	if err != nil {
		return panelPrerequisites{}, err
	}
	build, err := actionrelationexp.ParseBuildAuthority(buildBytes)
	if err != nil {
		return panelPrerequisites{}, fmt.Errorf("invalid panel build authority: %w", err)
	}
	if err := actionrelationexp.VerifySourceCheckout(root, build.SourceRows); err != nil {
		return panelPrerequisites{}, fmt.Errorf("panel source checkout: %w", err)
	}
	if err := verifyCompetenceBuild(root, build, filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))); err != nil {
		return panelPrerequisites{}, err
	}
	competenceBytes, err := readCommittedWorking(root, git, head, competenceRootDocument)
	if err != nil {
		return panelPrerequisites{}, err
	}
	competence, err := LoadCompetenceRoot(root, competenceBytes, build)
	if err != nil {
		return panelPrerequisites{}, err
	}
	entries, err := os.ReadDir(filepath.Join(root, ".nous", "actionrelations-v1-competence-evidence"))
	if err != nil {
		return panelPrerequisites{}, err
	}
	for _, entry := range entries {
		path := ".nous/actionrelations-v1-competence-evidence/" + entry.Name()
		if _, err := readCommittedWorking(root, git, head, path); err != nil {
			return panelPrerequisites{}, err
		}
	}
	plan, err := actionrelationexp.LoadCommittedReview(root, head, "plan")
	if err != nil {
		return panelPrerequisites{}, err
	}
	implementation, err := actionrelationexp.LoadCommittedReview(root, head, "implementation")
	if err != nil {
		return panelPrerequisites{}, err
	}
	planRef, _ := actionrelationexp.Reference(actionrelationexp.ReviewManifestPath("plan"), plan.Canonical)
	implementationRef, _ := actionrelationexp.Reference(actionrelationexp.ReviewManifestPath("implementation"), implementation.Canonical)
	buildRef, _ := actionrelationexp.Reference(actionrelationexp.BuildAuthorityPath, build.Canonical)
	competenceRef, _ := actionrelationexp.Reference(competenceRootDocument, competence.Canonical)
	if build.PlanReview != planRef || build.ImplementationReview != implementationRef || build.ImplementationCommit != implementation.ReviewedCommit || build.ImplementationArchiveDigest != implementation.ArchiveDigest || competence.BuildAuthority != buildRef {
		return panelPrerequisites{}, fmt.Errorf("prerequisite references do not close exact review and build authority")
	}
	if err := actionrelationexp.VerifyReviewArchive(root, plan); err != nil {
		return panelPrerequisites{}, err
	}
	if err := actionrelationexp.VerifyReviewArchive(root, implementation); err != nil {
		return panelPrerequisites{}, err
	}
	if _, err := git("merge-base", "--is-ancestor", build.BuildHead, head); err != nil {
		return panelPrerequisites{}, fmt.Errorf("build HEAD is not an ancestor of panel HEAD")
	}
	gitVersion, err := git("--version")
	if err != nil || strings.TrimSpace(string(gitVersion)) != build.GitVersion {
		return panelPrerequisites{}, fmt.Errorf("panel Git version differs from build authority")
	}
	commonDir, err := protectedGitCommonDir(git, root)
	if err != nil {
		return panelPrerequisites{}, err
	}
	return panelPrerequisites{Root: root, GitCommonDir: commonDir, Head: head, Build: build, Competence: competence, PlanReview: plan, ImplementationReview: implementation, PlanReviewRef: planRef, ImplementationRef: implementationRef, BuildRef: buildRef, CompetenceRef: competenceRef}, nil
}

func readCommittedWorking(root string, git func(...string) ([]byte, error), commit, path string) ([]byte, error) {
	tree, err := git("ls-tree", "-z", commit, "--", path)
	if err != nil || !bytes.HasSuffix(tree, append([]byte{'\t'}, append([]byte(path), 0)...)) || !bytes.HasPrefix(tree, []byte("100644 blob ")) || bytes.Count(tree, []byte{0}) != 1 {
		return nil, fmt.Errorf("committed authority is not one mode-100644 blob: %s", path)
	}
	committed, err := git("cat-file", "blob", commit+":"+path)
	if err != nil {
		return nil, fmt.Errorf("read committed prerequisite %s: %w", path, err)
	}
	physical := filepath.Join(root, filepath.FromSlash(path))
	info, statErr := os.Lstat(physical)
	working, readErr := readRegularNoFollow(physical, 0o644)
	if statErr != nil || readErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 || !bytes.Equal(committed, working) {
		return nil, fmt.Errorf("prerequisite differs from committed regular file: %s", path)
	}
	return working, nil
}

func newPanelWriter(root, panelRoot string, cap int64) *panelWriter {
	return &panelWriter{physicalRoot: root, panelRoot: panelRoot, cap: cap, files: map[string]retainedFile{}}
}

func (w *panelWriter) writeAll(files []actionrelationexp.EvidenceFile) error {
	for _, file := range files {
		if err := w.write(file); err != nil {
			return err
		}
	}
	return nil
}

func (w *panelWriter) read(path string) ([]byte, error) {
	retained, ok := w.files[path]
	if !ok {
		return nil, fmt.Errorf("panel evidence path is absent: %s", path)
	}
	data, err := readRegularNoFollow(retained.Path, 0o644)
	if err != nil || int64(len(data)) != retained.Bytes || digest(data) != retained.Digest {
		return nil, fmt.Errorf("panel evidence path changed: %s", path)
	}
	return data, nil
}

func (w *panelWriter) write(file actionrelationexp.EvidenceFile) error {
	if file.Mode != "100644" || !strings.HasPrefix(file.Path, w.panelRoot+"/") || filepath.Clean(file.Path) != file.Path || w.files[file.Path].Path != "" {
		return fmt.Errorf("invalid or duplicate panel evidence path: %s", file.Path)
	}
	w.total += int64(len(file.Data))
	if w.total > w.cap {
		return fmt.Errorf("panel evidence exceeds capacity")
	}
	physical := filepath.Join(w.physicalRoot, filepath.FromSlash(file.Path))
	if err := writeExclusiveAuthority(physical, file.Data); err != nil {
		return err
	}
	w.files[file.Path] = retainedFile{Path: physical, Bytes: int64(len(file.Data)), Digest: digest(file.Data)}
	return nil
}

func comparePanelFiles(primary, audit *panelWriter) error {
	if primary.total != audit.total || !reflect.DeepEqual(mapKeys(primary.files), mapKeys(audit.files)) {
		return fmt.Errorf("primary and audit evidence file sets differ")
	}
	for _, path := range mapKeys(primary.files) {
		left, right := primary.files[path], audit.files[path]
		if left.Bytes != right.Bytes || left.Digest != right.Digest {
			return fmt.Errorf("primary and audit evidence identity differs: %s", path)
		}
		equal, err := equalFiles(left.Path, right.Path)
		if err != nil || !equal {
			return fmt.Errorf("primary and audit evidence bytes differ: %s", path)
		}
	}
	return nil
}

func mapKeys(values map[string]retainedFile) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	slices.Sort(result)
	return result
}

func equalFiles(leftPath, rightPath string) (bool, error) {
	left, err := os.Open(leftPath)
	if err != nil {
		return false, err
	}
	defer left.Close()
	right, err := os.Open(rightPath)
	if err != nil {
		return false, err
	}
	defer right.Close()
	leftBuffer, rightBuffer := make([]byte, 128*1024), make([]byte, 128*1024)
	for {
		leftN, leftErr := left.Read(leftBuffer)
		rightN, rightErr := right.Read(rightBuffer)
		if leftN != rightN || !bytes.Equal(leftBuffer[:leftN], rightBuffer[:rightN]) {
			return false, nil
		}
		if leftErr == io.EOF && rightErr == io.EOF {
			return true, nil
		}
		if leftErr != nil || rightErr != nil {
			return false, errors.Join(leftErr, rightErr)
		}
	}
}

func requirePanelCapacity(root string, required int64) error {
	var state syscall.Statfs_t
	if err := syscall.Statfs(root, &state); err != nil {
		return err
	}
	available := int64(state.Bavail) * int64(state.Bsize)
	if available < required {
		return fmt.Errorf("insufficient panel capacity: have %d need %d", available, required)
	}
	return nil
}
