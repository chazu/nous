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
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationscore"
)

const developmentEvidenceCap = int64(2593 * 1024 * 1024)

type panelPrerequisites struct {
	Root                 string
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
	evidenceRoot, _ := actionrelationexp.EvidenceRoot("development")
	for _, path := range []string{
		evidenceRoot,
		".nous/actionrelations-v1-development-report.json",
		".nous/actionrelations-v1-development-terminal-receipt.json",
	} {
		if _, err := os.Lstat(filepath.Join(prerequisites.Root, filepath.FromSlash(path))); err == nil {
			return actionrelationscore.Report{}, fmt.Errorf("development output already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return actionrelationscore.Report{}, err
		}
	}
	if err := requirePanelCapacity(prerequisites.Root, 2*developmentEvidenceCap); err != nil {
		return actionrelationscore.Report{}, err
	}
	primary := newPanelWriter(prerequisites.Root, evidenceRoot, developmentEvidenceCap)
	fixturePath := evidenceRoot + "/authority/fixture-root.json"
	preparePrimary := func(fixture actionrelationfixture.PanelFixture) error {
		return primary.write(actionrelationexp.EvidenceFile{Path: fixturePath, Mode: "100644", Data: fixture.Canonical})
	}
	consumePrimary := func(chunk actionrelationscore.PanelCurriculumEvidence) error {
		return primary.writeAll(append(slices.Clone(chunk.ManifestFiles), chunk.PackFiles...))
	}
	domains := filepath.Join(prerequisites.Root, "domains")
	primarySummary, err := actionrelationscore.ExecuteDevelopmentPanel(domains, preparePrimary, consumePrimary)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := primary.write(primarySummary.RunEvidenceManifest); err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := primary.write(primarySummary.RunEvidence.File); err != nil {
		return actionrelationscore.Report{}, err
	}
	auditRoot, err := os.MkdirTemp("", "nous-actionrelation-audit-")
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	defer os.RemoveAll(auditRoot)
	audit := newPanelWriter(auditRoot, evidenceRoot, developmentEvidenceCap)
	prepareAudit := func(fixture actionrelationfixture.PanelFixture) error {
		return audit.write(actionrelationexp.EvidenceFile{Path: fixturePath, Mode: "100644", Data: fixture.Canonical})
	}
	consumeAudit := func(chunk actionrelationscore.PanelCurriculumEvidence) error {
		return audit.writeAll(append(slices.Clone(chunk.ManifestFiles), chunk.PackFiles...))
	}
	auditSummary, err := actionrelationscore.ExecuteDevelopmentPanel(domains, prepareAudit, consumeAudit)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := audit.write(auditSummary.RunEvidenceManifest); err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := audit.write(auditSummary.RunEvidence.File); err != nil {
		return actionrelationscore.Report{}, err
	}
	if !reflect.DeepEqual(primarySummary, auditSummary) {
		return actionrelationscore.Report{}, fmt.Errorf("primary and audit panel summaries differ")
	}
	if err := comparePanelFiles(primary, audit); err != nil {
		return actionrelationscore.Report{}, err
	}
	return publishDevelopment(prerequisites, primary, primarySummary)
}

func publishDevelopment(prerequisites panelPrerequisites, writer *panelWriter, summary actionrelationscore.PanelSummary) (actionrelationscore.Report, error) {
	evidenceRoot, _ := actionrelationexp.EvidenceRoot("development")
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
	if err != nil || actionrelationexp.EqualExecutionEvidence(primary, audit) != nil {
		return actionrelationscore.Report{}, fmt.Errorf("build audit execution: %w", err)
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
		RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps,
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
	gates := actionrelationscore.MechanicalGates{AuthorityClosure: true, PrimaryAuditEqual: true, SemanticAgreement: true, WorkConservation: true, ArtifactsImmutable: true, NousZeroFalseMatches: true, RequiredBehaviorEqual: true, FreshCertificatesValid: true}
	report, err := actionrelationscore.BuildReport(summary.Panel, summary.Authority, actionrelationscore.ReportAuthority{
		PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, FixtureRoot: fixtureRef,
		CurriculumRowsRoot: summary.CurriculumRowsRoot, EvidencePayload: payloadRef,
	}, gates, inference)
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	reportPath := ".nous/actionrelations-v1-development-report.json"
	reportRef, _ := actionrelationexp.Reference(reportPath, report.Canonical)
	receipt, err := actionrelationexp.BuildTerminalReceipt(actionrelationexp.TerminalReceipt{
		Panel: summary.Panel, State: "published", SourceRoot: prerequisites.Build.SourceRoot, FixtureRoot: fixtureRef,
		AttemptCommitment: strings.Repeat("0", 64), Report: reportRef, EvidencePayload: payloadRef, Reason: report.Classification,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
	}
	receiptPath := ".nous/actionrelations-v1-development-terminal-receipt.json"
	receiptRef, _ := actionrelationexp.Reference(receiptPath, receipt.Canonical)
	publication, err := actionrelationexp.BuildPublication(actionrelationexp.Publication{
		Panel: summary.Panel, PlanReview: prerequisites.PlanReviewRef, ImplementationReview: prerequisites.ImplementationRef,
		BuildAuthority: prerequisites.BuildRef, Competence: prerequisites.CompetenceRef, PrimaryExecution: primaryRef,
		AuditExecution: auditRef, AuditAttestation: attestationRef, RunEvidence: runEvidenceRef, StructuralMaps: summary.StructuralMaps,
		FixtureRoot: fixtureRef, ExecutionCore: coreRef, EvidencePayload: payloadRef, Report: reportRef, TerminalReceipt: receiptRef,
	})
	if err != nil {
		return actionrelationscore.Report{}, err
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
	if err := writeExclusiveAuthority(filepath.Join(prerequisites.Root, filepath.FromSlash(receiptPath)), receipt.Canonical); err != nil {
		return actionrelationscore.Report{}, err
	}
	if err := writer.write(actionrelationexp.EvidenceFile{Path: authorityRoot + "/publication.json", Mode: "100644", Data: publication.Canonical}); err != nil {
		return actionrelationscore.Report{}, err
	}
	return report, nil
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
	if err != nil || head != strings.TrimSpace(string(originBytes)) {
		return panelPrerequisites{}, fmt.Errorf("panel requires clean HEAD at origin/main")
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
	return panelPrerequisites{Root: root, Head: head, Build: build, Competence: competence, PlanReview: plan, ImplementationReview: implementation, PlanReviewRef: planRef, ImplementationRef: implementationRef, BuildRef: buildRef, CompetenceRef: competenceRef}, nil
}

func readCommittedWorking(root string, git func(...string) ([]byte, error), commit, path string) ([]byte, error) {
	committed, err := git("cat-file", "blob", commit+":"+path)
	if err != nil {
		return nil, fmt.Errorf("read committed prerequisite %s: %w", path, err)
	}
	physical := filepath.Join(root, filepath.FromSlash(path))
	info, statErr := os.Lstat(physical)
	working, readErr := os.ReadFile(physical)
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
