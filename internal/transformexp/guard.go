package transformexp

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const ReviewManifestPath = "docs/transformation-schema-implementation-reviews.json"

type ImplementationReview struct {
	Scope, Status, ReviewedCommit string
}

type ImplementationReviewManifest struct {
	Version, PlanCommit, ImplementationCommit string
	Reviews                                   []ImplementationReview
	ProtectedPaths                            map[string]string
}

type repositoryAuthority struct {
	Root, Head      string
	Reviews         ImplementationReviewManifest
	ReviewAuthority []byte
}

type attemptReceipt struct {
	Panel, State, Head, ImplementationCommit, StartedUTC   string
	RootCommitment, FixtureRoot, ReportDigest, GraphDigest string
}

// ExecuteDevelopment is the only exported path to the public development
// panel. Development is repeatable only under a new clean evidence namespace.
func ExecuteDevelopment(repoRoot, domainsDir string) (protectedReport, error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return protectedReport{}, err
	}
	if err := claimDevelopment(authority.Root); err != nil {
		return protectedReport{}, err
	}
	curricula, err := developmentPanel()
	if err != nil {
		return protectedReport{}, err
	}
	if _, err := persistPreparedFixtures(authority.Root, "development", curricula); err != nil {
		return protectedReport{}, err
	}
	evidence, err := buildPanelEvidence(filepath.Join(authority.Root, "domains"), "development", curricula, 841001, authority.ReviewAuthority)
	if err != nil {
		return protectedReport{}, err
	}
	powerRows, err := pairedRows(evidence.Report.Rows, len(curricula))
	if err != nil {
		return protectedReport{}, err
	}
	power, err := estimateTransformPower(powerRows, 2000, 2000)
	if err != nil {
		return protectedReport{}, err
	}
	report, err := newProtectedReport("development", authority.Reviews.ImplementationCommit, evidence, power)
	if err != nil {
		return protectedReport{}, err
	}
	if err := persistProtected(authority.Root, "development", evidence, report); err != nil {
		return protectedReport{}, err
	}
	return report, nil
}

// ExecuteValidation is the only exported path to the one-shot validation
// panel. Its receipt is claimed before validationPanel is called.
func ExecuteValidation(repoRoot, domainsDir string) (report protectedReport, returnErr error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return protectedReport{}, err
	}
	development, err := verifyCommittedPanel(authority, "development")
	if err != nil || !development.Payload.Power.Authorized || development.Classification != "interim-power-authorized" {
		return protectedReport{}, fmt.Errorf("validation requires committed authorized development evidence: %w", err)
	}
	receipt, err := claimAttempt(authority, "validation")
	if err != nil {
		return protectedReport{}, err
	}
	defer func() {
		if returnErr != nil {
			if invalidErr := finalizeAttempt(authority.Root, receipt, "invalid", "", "", ""); invalidErr != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("persist invalid validation receipt: %w", invalidErr))
			}
		}
	}()
	curricula, err := validationPanel()
	if err != nil {
		return protectedReport{}, err
	}
	fixtureRoot, err := persistPreparedFixtures(authority.Root, "validation", curricula)
	if err != nil {
		return protectedReport{}, err
	}
	if err := startAttempt(authority.Root, receipt, "", fixtureRoot); err != nil {
		return protectedReport{}, err
	}
	evidence, err := buildPanelEvidence(filepath.Join(authority.Root, "domains"), "validation", curricula, 842001, authority.ReviewAuthority)
	if err != nil {
		return protectedReport{}, err
	}
	report, err = newProtectedReport("validation", authority.Reviews.ImplementationCommit, evidence, development.Payload.Power)
	if err != nil {
		return protectedReport{}, err
	}
	if err := persistProtected(authority.Root, "validation", evidence, report); err != nil {
		return protectedReport{}, err
	}
	reportBytes, _ := canonicalProtectedReport(report)
	if err := finalizeAttempt(authority.Root, receipt, "published", evidence.Report.FixtureRootDigest, digestBytes(reportBytes), evidence.Report.EvidenceGraphDigest); err != nil {
		return protectedReport{}, err
	}
	return report, nil
}

// ExecuteLocked is the only exported path to the one-shot locked panel.
func ExecuteLocked(repoRoot, domainsDir, unlockToken string) (report protectedReport, returnErr error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return protectedReport{}, err
	}
	if unlockToken != "transform-schema/v1:"+authority.Head {
		return protectedReport{}, fmt.Errorf("locked token does not name exact clean HEAD")
	}
	development, err := verifyCommittedPanel(authority, "development")
	if err != nil || development.Classification != "interim-power-authorized" || !development.Payload.Power.Authorized {
		return protectedReport{}, fmt.Errorf("locked requires committed authorized development evidence: %w", err)
	}
	validation, err := verifyCommittedPanel(authority, "validation")
	if err != nil || validation.Classification != "interim-valid" || !validation.Payload.Power.Authorized {
		return protectedReport{}, fmt.Errorf("locked requires committed validation evidence: %w", err)
	}
	if validation.Payload.Power != development.Payload.Power {
		return protectedReport{}, fmt.Errorf("validation does not retain committed development power")
	}
	if err := verifyCommittedReceipt(authority, "validation", validation); err != nil {
		return protectedReport{}, err
	}
	receipt, err := claimAttempt(authority, "locked")
	if err != nil {
		return protectedReport{}, err
	}
	defer func() {
		if returnErr != nil {
			if invalidErr := finalizeAttempt(authority.Root, receipt, "invalid", "", "", ""); invalidErr != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("persist invalid locked receipt: %w", invalidErr))
			}
		}
	}()
	root := make([]byte, sha256.Size)
	if _, err := io.ReadFull(rand.Reader, root); err != nil {
		return protectedReport{}, err
	}
	defer func() {
		for index := range root {
			root[index] = 0
		}
	}()
	commitment := digestBytes(root)
	if err := startAttempt(authority.Root, receipt, commitment, ""); err != nil {
		return protectedReport{}, err
	}
	curricula, pairs, err := lockedPanel(root)
	if err != nil {
		return protectedReport{}, err
	}
	fixtureRoot, err := persistPreparedFixtures(authority.Root, "locked", curricula)
	if err != nil {
		return protectedReport{}, err
	}
	if err := bindAttemptFixture(authority.Root, receipt, fixtureRoot); err != nil {
		return protectedReport{}, err
	}
	for index := range root {
		root[index] = 0
	}
	root = nil
	for index := range curricula {
		curricula[index].Seed = 0
	}
	evidence, err := buildPanelEvidenceWithPairs(filepath.Join(authority.Root, "domains"), "locked", curricula, 0, pairs, authority.ReviewAuthority)
	for index := range pairs {
		pairs[index] = [2]uint64{}
	}
	pairs = nil
	if err != nil {
		return protectedReport{}, err
	}
	report, err = newProtectedReport("locked", authority.Reviews.ImplementationCommit, evidence, validation.Payload.Power)
	if err != nil {
		return protectedReport{}, err
	}
	if err := persistProtected(authority.Root, "locked", evidence, report); err != nil {
		return protectedReport{}, err
	}
	reportBytes, _ := canonicalProtectedReport(report)
	if err := finalizeAttempt(authority.Root, receipt, "published", evidence.Report.FixtureRootDigest, digestBytes(reportBytes), evidence.Report.EvidenceGraphDigest); err != nil {
		return protectedReport{}, err
	}
	return report, nil
}

func pairedRows(rows []PolicyReportRow, curricula int) ([]pairedTransformRow, error) {
	result := make([]pairedTransformRow, curricula)
	seen := make([][2]bool, curricula)
	for _, row := range rows {
		if row.Ordinal < 0 || row.Ordinal >= curricula {
			return nil, fmt.Errorf("policy row ordinal outside panel")
		}
		var lane int
		switch row.Policy {
		case NousRefine:
			lane = 0
			result[row.Ordinal].Ordinal = row.Ordinal
			result[row.Ordinal].Family = row.Family
			result[row.Ordinal].NousSuccess = row.HeldoutCorrect == 8
			result[row.Ordinal].FalseApplications = row.FalseApplications
			result[row.Ordinal].NonmatchingNousWork = row.NonmatchingWork
		case BoundedPBE:
			lane = 1
			result[row.Ordinal].PBESuccess = row.HeldoutCorrect == 8
			result[row.Ordinal].NonmatchingPBEWork = row.NonmatchingWork
		default:
			continue
		}
		if seen[row.Ordinal][lane] {
			return nil, fmt.Errorf("duplicate paired policy row")
		}
		seen[row.Ordinal][lane] = true
	}
	for ordinal, lanes := range seen {
		if !lanes[0] || !lanes[1] {
			return nil, fmt.Errorf("incomplete paired row %d", ordinal)
		}
	}
	return result, nil
}

func authorizeRepository(repoRoot, domainsDir string) (repositoryAuthority, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return repositoryAuthority{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return repositoryAuthority{}, err
	}
	domains, err := filepath.Abs(domainsDir)
	if err != nil {
		return repositoryAuthority{}, err
	}
	domains, err = filepath.EvalSymlinks(domains)
	if err != nil || domains != filepath.Join(root, "domains") {
		return repositoryAuthority{}, fmt.Errorf("domains path is not canonical repository domains/")
	}
	for _, forbidden := range []string{"go.work", "go.work.sum"} {
		if _, err := os.Lstat(filepath.Join(root, forbidden)); err == nil {
			return repositoryAuthority{}, fmt.Errorf("%s is forbidden for protected execution", forbidden)
		} else if !os.IsNotExist(err) {
			return repositoryAuthority{}, err
		}
	}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return repositoryAuthority{}, err
	}
	if bytes.Contains(module, []byte("replace ")) || bytes.Contains(module, []byte("replace(")) || bytes.Contains(module, []byte("replace (")) {
		return repositoryAuthority{}, fmt.Errorf("module replacements are forbidden")
	}
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return repositoryAuthority{}, err
	}
	canonicalTop, err := filepath.EvalSymlinks(top)
	if err != nil || canonicalTop != root {
		return repositoryAuthority{}, fmt.Errorf("Git top level is not canonical repository root")
	}
	if shallow, err := gitOutput(root, "rev-parse", "--is-shallow-repository"); err != nil || shallow != "false" {
		return repositoryAuthority{}, fmt.Errorf("shallow Git authority is forbidden")
	}
	if replacements, err := gitOutput(root, "for-each-ref", "--format=%(refname)", "refs/replace"); err != nil || replacements != "" {
		return repositoryAuthority{}, fmt.Errorf("Git replacement authority is forbidden")
	}
	for _, forbidden := range []string{"objects/info/alternates", "info/grafts"} {
		path, pathErr := gitOutput(root, "rev-parse", "--git-path", forbidden)
		if pathErr != nil {
			return repositoryAuthority{}, pathErr
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, statErr := os.Lstat(path); statErr == nil {
			return repositoryAuthority{}, fmt.Errorf("Git %s authority is forbidden", forbidden)
		} else if !os.IsNotExist(statErr) {
			return repositoryAuthority{}, statErr
		}
	}
	localConfig, err := gitOutput(root, "config", "--local", "--name-only", "--get-regexp", ".*")
	if err != nil && localConfig != "" {
		return repositoryAuthority{}, fmt.Errorf("cannot inspect local Git configuration")
	}
	for _, name := range strings.Fields(localConfig) {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "include.") || strings.HasPrefix(lower, "extensions.") || strings.HasPrefix(lower, "submodule.") || lower == "core.worktree" || lower == "core.attributesfile" || lower == "core.hookspath" || lower == "core.sshcommand" {
			return repositoryAuthority{}, fmt.Errorf("local Git authority is forbidden: %s", name)
		}
	}
	for _, operation := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"} {
		path, pathErr := gitOutput(root, "rev-parse", "--git-path", operation)
		if pathErr != nil {
			return repositoryAuthority{}, pathErr
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, statErr := os.Lstat(path); statErr == nil {
			return repositoryAuthority{}, fmt.Errorf("Git operation in progress: %s", operation)
		} else if !os.IsNotExist(statErr) {
			return repositoryAuthority{}, statErr
		}
	}
	status, err := gitOutput(root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || status != "" {
		return repositoryAuthority{}, fmt.Errorf("repository is not clean")
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return repositoryAuthority{}, err
	}
	manifestBytes, err := readCommittedBlob(root, head, ReviewManifestPath)
	if err != nil {
		return repositoryAuthority{}, fmt.Errorf("implementation review authority: %w", err)
	}
	manifest, err := decodeReviewManifest(manifestBytes)
	if err != nil {
		return repositoryAuthority{}, err
	}
	if manifest.PlanCommit != PlanCommit || !isLowerHex(manifest.ImplementationCommit, 40) || gitCommand(root, "merge-base", "--is-ancestor", manifest.ImplementationCommit, head).Run() != nil {
		return repositoryAuthority{}, fmt.Errorf("reviewed implementation is not an ancestor of HEAD")
	}
	required, err := requiredProtectedPaths(root)
	if err != nil || len(required) != len(manifest.ProtectedPaths) {
		return repositoryAuthority{}, fmt.Errorf("review manifest does not bind exact protected source surface")
	}
	for _, path := range required {
		want, ok := manifest.ProtectedPaths[path]
		if !ok || !isLowerHex(want, 64) {
			return repositoryAuthority{}, fmt.Errorf("review manifest omits %s", path)
		}
		working, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		reviewed, reviewErr := gitFileAtCommit(root, manifest.ImplementationCommit, path)
		if readErr != nil || reviewErr != nil || digestBytes(working) != want || digestBytes(reviewed) != want {
			return repositoryAuthority{}, fmt.Errorf("protected path changed: %s", path)
		}
	}
	if err := verifyReviewedFilesystem(root, manifest.ProtectedPaths); err != nil {
		return repositoryAuthority{}, err
	}
	reviewAuthority, err := reviewAuthorityWire(manifest)
	if err != nil {
		return repositoryAuthority{}, err
	}
	return repositoryAuthority{root, head, manifest, reviewAuthority}, nil
}

func decodeReviewManifest(data []byte) (ImplementationReviewManifest, error) {
	var wire struct {
		Version              string `json:"version"`
		PlanCommit           string `json:"plan_commit"`
		ImplementationCommit string `json:"implementation_commit"`
		Reviews              []struct {
			Scope          string `json:"scope"`
			Status         string `json:"status"`
			ReviewedCommit string `json:"reviewed_commit"`
		} `json:"reviews"`
		ProtectedPaths map[string]string `json:"protected_paths"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil || decoder.Decode(new(any)) == nil || wire.Version != "transform-implementation-reviews/v1" {
		return ImplementationReviewManifest{}, fmt.Errorf("invalid implementation review manifest")
	}
	wantScopes := []string{"architecture", "semantics", "experiment"}
	if len(wire.Reviews) != len(wantScopes) {
		return ImplementationReviewManifest{}, fmt.Errorf("implementation review manifest requires three reviews")
	}
	manifest := ImplementationReviewManifest{wire.Version, wire.PlanCommit, wire.ImplementationCommit, nil, wire.ProtectedPaths}
	for index, review := range wire.Reviews {
		if review.Scope != wantScopes[index] || review.Status != "accepted" || review.ReviewedCommit != wire.ImplementationCommit {
			return ImplementationReviewManifest{}, fmt.Errorf("implementation review %d is not exact accepted authority", index)
		}
		manifest.Reviews = append(manifest.Reviews, ImplementationReview{review.Scope, review.Status, review.ReviewedCommit})
	}
	canonical, _ := encodeReviewManifest(manifest)
	if !bytes.Equal(canonical, data) {
		return ImplementationReviewManifest{}, fmt.Errorf("implementation review manifest is not canonical")
	}
	return manifest, nil
}

func encodeReviewManifest(manifest ImplementationReviewManifest) ([]byte, error) {
	reviews := make([]map[string]string, len(manifest.Reviews))
	for index, review := range manifest.Reviews {
		reviews[index] = map[string]string{"scope": review.Scope, "status": review.Status, "reviewed_commit": review.ReviewedCommit}
	}
	return json.Marshal(struct {
		Version              string              `json:"version"`
		PlanCommit           string              `json:"plan_commit"`
		ImplementationCommit string              `json:"implementation_commit"`
		Reviews              []map[string]string `json:"reviews"`
		ProtectedPaths       map[string]string   `json:"protected_paths"`
	}{manifest.Version, manifest.PlanCommit, manifest.ImplementationCommit, reviews, manifest.ProtectedPaths})
}

func reviewAuthorityWire(manifest ImplementationReviewManifest) ([]byte, error) {
	reviews := make([]any, len(manifest.Reviews))
	for index, review := range manifest.Reviews {
		reviews[index] = []any{review.Scope, review.Status, review.ReviewedCommit}
	}
	paths := make([]string, 0, len(manifest.ProtectedPaths))
	for path := range manifest.ProtectedPaths {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	pathRows := make([]any, len(paths))
	for index, path := range paths {
		pathRows[index] = []any{path, manifest.ProtectedPaths[path]}
	}
	return json.Marshal([]any{"transform-reviews/v1", manifest.PlanCommit, manifest.ImplementationCommit, reviews, pathRows})
}

func requiredProtectedPaths(root string) ([]string, error) {
	tracked, err := gitOutput(root, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range strings.Split(tracked, "\x00") {
		if path == "" || path == ReviewManifestPath {
			continue
		}
		if reviewedInput(path) || path == "mise.toml" || path == "docs/transformation-schema-induction-vocabulary-plan.md" || path == "docs/vocabulary-research-program-v3.md" {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths, nil
}

func reviewedInput(path string) bool {
	if path == "go.mod" || path == "go.sum" {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".cue", ".s", ".asm", ".c", ".cc", ".cpp", ".h", ".hpp", ".mod", ".sum", ".work", ".syso", ".swig", ".swigcxx":
		return true
	default:
		return false
	}
}

func verifyReviewedFilesystem(root string, protected map[string]string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() && (relative == ".git" || relative == ".nous") {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("repository contains forbidden symlink: %s", relative)
		}
		if entry.IsDir() || !reviewedInput(relative) {
			return nil
		}
		if _, ok := protected[relative]; !ok {
			return fmt.Errorf("compiler/runtime input is outside reviewed surface: %s", relative)
		}
		return nil
	})
}

func claimDevelopment(root string) error {
	if err := requireAbsent(reportPath(root, "development")); err != nil {
		return err
	}
	if err := requireAbsent(transcriptPath(root, "development")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".nous"), 0o755); err != nil {
		return err
	}
	return os.Mkdir(transcriptPath(root, "development"), 0o755)
}

func claimAttempt(authority repositoryAuthority, panel string) (*attemptReceipt, error) {
	if panel != "validation" && panel != "locked" {
		return nil, fmt.Errorf("invalid protected panel")
	}
	if err := os.MkdirAll(filepath.Join(authority.Root, ".nous"), 0o755); err != nil {
		return nil, err
	}
	for _, path := range []string{receiptPath(authority.Root, panel), reportPath(authority.Root, panel), transcriptPath(authority.Root, panel)} {
		if err := requireAbsent(path); err != nil {
			return nil, err
		}
	}
	receipt := &attemptReceipt{Panel: panel, State: "claimed", Head: authority.Head, ImplementationCommit: authority.Reviews.ImplementationCommit, StartedUTC: time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")}
	if err := writeExclusiveSync(receiptPath(authority.Root, panel), receiptBytes(receipt), 0o600); err != nil {
		return nil, err
	}
	if err := os.Mkdir(transcriptPath(authority.Root, panel), 0o755); err != nil {
		if invalidErr := finalizeAttempt(authority.Root, receipt, "invalid", "", "", ""); invalidErr != nil {
			return nil, errors.Join(err, fmt.Errorf("persist invalid receipt after transcript directory failure: %w", invalidErr))
		}
		return nil, err
	}
	if err := syncDirectory(filepath.Join(authority.Root, ".nous")); err != nil {
		if invalidErr := finalizeAttempt(authority.Root, receipt, "invalid", "", "", ""); invalidErr != nil {
			return nil, errors.Join(err, fmt.Errorf("persist invalid receipt after directory sync failure: %w", invalidErr))
		}
		return nil, err
	}
	return receipt, nil
}

func receiptBytes(receipt *attemptReceipt) []byte {
	return mustJSON([]any{"transform-attempt/v1", receipt.Panel, receipt.State, receipt.Head, receipt.ImplementationCommit, PlanCommit, receipt.StartedUTC, receipt.RootCommitment, receipt.FixtureRoot, receipt.ReportDigest, receipt.GraphDigest})
}

func startAttempt(root string, receipt *attemptReceipt, commitment, fixture string) error {
	next := *receipt
	next.RootCommitment = commitment
	next.FixtureRoot = fixture
	return rewriteReceiptValue(root, receipt, next, "running")
}

func bindAttemptFixture(root string, receipt *attemptReceipt, fixture string) error {
	if receipt.State != "running" || receipt.FixtureRoot != "" || !isLowerHex(fixture, 64) {
		return fmt.Errorf("cannot monotonically bind attempt fixture")
	}
	next := *receipt
	next.FixtureRoot = fixture
	if err := replaceReceiptBytes(root, &next); err != nil {
		return err
	}
	*receipt = next
	return nil
}

func finalizeAttempt(root string, receipt *attemptReceipt, state, fixture, report, graph string) error {
	next := *receipt
	if fixture != "" {
		next.FixtureRoot = fixture
	}
	if report != "" {
		next.ReportDigest = report
	}
	if graph != "" {
		next.GraphDigest = graph
	}
	return rewriteReceiptValue(root, receipt, next, state)
}

func rewriteReceipt(root string, receipt *attemptReceipt, state string) error {
	return rewriteReceiptValue(root, receipt, *receipt, state)
}

func rewriteReceiptValue(root string, receipt *attemptReceipt, next attemptReceipt, state string) error {
	allowed := receipt.State == "claimed" && (state == "running" || state == "invalid") || receipt.State == "running" && (state == "published" || state == "invalid")
	if !allowed {
		return fmt.Errorf("invalid receipt transition %s -> %s", receipt.State, state)
	}
	next.State = state
	if err := replaceReceiptBytes(root, &next); err != nil {
		return err
	}
	*receipt = next
	return nil
}

func replaceReceiptBytes(root string, receipt *attemptReceipt) error {
	path := receiptPath(root, receipt.Panel)
	temporary := path + ".next"
	if err := writeExclusiveSync(temporary, receiptBytes(receipt), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func persistPreparedFixtures(root, panel string, curricula []curriculum) (string, error) {
	files, fixtureRoot, err := buildPreparedEvidence(panel, curricula)
	if err != nil {
		return "", err
	}
	base := transcriptPath(root, panel)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	createdDirs := map[string]bool{base: true}
	for _, name := range names {
		data := files[name]
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		createdDirs[filepath.Dir(path)] = true
		if err := writeExclusiveSync(path, data, 0o600); err != nil {
			return "", err
		}
	}
	if err := syncDirectoriesDeepestFirst(createdDirs); err != nil {
		return "", err
	}
	return digestBytes(fixtureRoot), nil
}

func persistProtected(root, panel string, evidence panelEvidence, report protectedReport) error {
	base := transcriptPath(root, panel)
	names := make([]string, 0, len(evidence.Files))
	for name := range evidence.Files {
		names = append(names, name)
	}
	slices.Sort(names)
	createdDirs := map[string]bool{base: true}
	for _, name := range names {
		data := evidence.Files[name]
		if !validEvidencePath(name) {
			return fmt.Errorf("invalid evidence output path")
		}
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		createdDirs[filepath.Dir(path)] = true
		if existing, err := os.ReadFile(path); err == nil {
			if !bytes.Equal(existing, data) {
				return fmt.Errorf("precommitted evidence differs at %s", name)
			}
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := writeExclusiveSync(path, data, 0o600); err != nil {
			return err
		}
	}
	if err := syncDirectoriesDeepestFirst(createdDirs); err != nil {
		return err
	}
	if err := writeExclusiveSync(filepath.Join(base, "evidence-graph.json"), evidence.EvidenceGraph, 0o600); err != nil {
		return err
	}
	if err := syncDirectory(base); err != nil {
		return err
	}
	reportBytes, err := canonicalProtectedReport(report)
	if err != nil {
		return err
	}
	if err := writeExclusiveSync(reportPath(root, panel), reportBytes, 0o600); err != nil {
		return err
	}
	return syncDirectory(filepath.Join(root, ".nous"))
}

func syncDirectoriesDeepestFirst(directories map[string]bool) error {
	paths := make([]string, 0, len(directories))
	for path := range directories {
		paths = append(paths, path)
	}
	slices.SortFunc(paths, func(left, right string) int {
		leftDepth := strings.Count(filepath.Clean(left), string(filepath.Separator))
		rightDepth := strings.Count(filepath.Clean(right), string(filepath.Separator))
		if leftDepth != rightDepth {
			return rightDepth - leftDepth
		}
		return strings.Compare(left, right)
	})
	for _, path := range paths {
		if err := syncDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func verifyCommittedPanel(authority repositoryAuthority, panel string) (protectedReport, error) {
	reportBytes, err := readCommittedBlob(authority.Root, authority.Head, relativeTo(authority.Root, reportPath(authority.Root, panel)))
	if err != nil {
		return protectedReport{}, err
	}
	report, err := decodeProtectedReport(reportBytes)
	if err != nil || report.Payload.Panel != panel || report.Payload.ImplementationCommit != authority.Reviews.ImplementationCommit {
		return protectedReport{}, fmt.Errorf("invalid committed %s report", panel)
	}
	for _, gate := range report.Payload.Gates {
		if !gate {
			return protectedReport{}, fmt.Errorf("committed %s report failed a mechanical gate", panel)
		}
	}
	graphPath := filepath.Join(transcriptPath(authority.Root, panel), "evidence-graph.json")
	graph, err := readCommittedBlob(authority.Root, authority.Head, relativeTo(authority.Root, graphPath))
	if err != nil || digestBytes(graph) != report.Payload.EvidenceGraph {
		return protectedReport{}, fmt.Errorf("committed %s evidence graph mismatch", panel)
	}
	if err := verifyGraphBlobs(authority, panel, graph); err != nil {
		return protectedReport{}, err
	}
	if err := verifyReportReconstruction(authority, report); err != nil {
		return protectedReport{}, err
	}
	return report, nil
}

func verifyReportReconstruction(authority repositoryAuthority, report protectedReport) error {
	panel := report.Payload.Panel
	base := transcriptPath(authority.Root, panel)
	read := func(name string) ([]byte, error) {
		return readCommittedBlob(authority.Root, authority.Head, relativeTo(authority.Root, filepath.Join(base, filepath.FromSlash(name))))
	}
	checks := []struct {
		Path, Digest string
	}{{"fixture-root.json", report.Payload.FixtureRoot}, {"primary/execution-manifest.json", report.Payload.PrimaryManifest}, {"audit/execution-manifest.json", report.Payload.AuditManifest}, {"competence/root.json", report.Payload.CompetenceRoot}}
	for _, check := range checks {
		data, err := read(check.Path)
		if err != nil || digestBytes(data) != check.Digest {
			return fmt.Errorf("%s report digest mismatch at %s", panel, check.Path)
		}
	}
	reviewBytes, err := read("review-authority.json")
	if err != nil || !bytes.Equal(reviewBytes, authority.ReviewAuthority) {
		return fmt.Errorf("%s review authority leaf mismatch", panel)
	}
	competenceBytes, err := read("competence/report.json")
	if err != nil {
		return err
	}
	var competence CompetenceReport
	if json.Unmarshal(competenceBytes, &competence) != nil || competence != report.Payload.Competence || competence != (CompetenceReport{351, 25272, 7020, 14, true}) {
		return fmt.Errorf("%s competence evidence mismatch", panel)
	}
	wantRows := DevelopmentCount * len(empiricalPolicies)
	if panel == "validation" {
		wantRows = ValidationCount * len(empiricalPolicies)
	}
	if panel == "locked" {
		wantRows = LockedCount * len(empiricalPolicies)
	}
	if len(report.Payload.Rows) != wantRows || len(report.Payload.Limitations) != 0 {
		return fmt.Errorf("%s report row/limitation cardinality mismatch", panel)
	}
	for index, row := range report.Payload.Rows {
		ordinal, policyIndex := index/len(empiricalPolicies), index%len(empiricalPolicies)
		decodedBits, bitsErr := hex.DecodeString(row.HeldoutCorrectBits)
		if row.Ordinal != ordinal || row.Policy != empiricalPolicies[policyIndex] || row.Family < 0 || row.Family >= len(familySchemas) || bitsErr != nil || len(decodedBits) != 1 {
			return fmt.Errorf("%s report row ordering/shape mismatch", panel)
		}
	}
	paired, err := pairedRows(report.Payload.Rows, wantRows/len(empiricalPolicies))
	if err != nil {
		return err
	}
	if panel != "locked" {
		authoritySeed := uint64(841001)
		if panel == "validation" {
			authoritySeed = 842001
		}
		inference, err := computeTransformInference(paired, panel, authoritySeed, 10000, 10000)
		if err != nil || !bytes.Equal(mustJSON(inference), mustJSON(report.Payload.Inference)) {
			return fmt.Errorf("%s inference does not reconstruct from policy rows", panel)
		}
	}
	if panel == "development" {
		power, err := estimateTransformPower(paired, 2000, 2000)
		if err != nil || power != report.Payload.Power {
			return fmt.Errorf("development power does not reconstruct from policy rows")
		}
	} else if report.Payload.Power.Replicates != 2000 || report.Payload.Power.Passing < 0 || report.Payload.Power.Passing > 2000 || report.Payload.Power.Authorized != (report.Payload.Power.Passing >= 1600) {
		return fmt.Errorf("%s retained development power is invalid", panel)
	}
	classification, err := protectedClassification(panel, report.Payload.Inference, report.Payload.Power)
	if err != nil || classification != report.Classification {
		return fmt.Errorf("%s classification does not reconstruct", panel)
	}
	return nil
}

func verifyGraphBlobs(authority repositoryAuthority, panel string, graph []byte) error {
	var wire []json.RawMessage
	if json.Unmarshal(graph, &wire) != nil || len(wire) != 3 {
		return fmt.Errorf("invalid evidence graph")
	}
	var version, graphPanel string
	var leaves [][]json.RawMessage
	if json.Unmarshal(wire[0], &version) != nil || version != "transform-evidence-graph/v1" || json.Unmarshal(wire[1], &graphPanel) != nil || graphPanel != panel || json.Unmarshal(wire[2], &leaves) != nil || len(leaves) == 0 {
		return fmt.Errorf("invalid evidence graph identity")
	}
	previous := ""
	for _, leaf := range leaves {
		var path, digest, mode string
		var size int
		if len(leaf) != 4 || json.Unmarshal(leaf[0], &path) != nil || json.Unmarshal(leaf[1], &digest) != nil || json.Unmarshal(leaf[2], &size) != nil || json.Unmarshal(leaf[3], &mode) != nil || !validEvidencePath(path) || path <= previous || mode != "100644" {
			return fmt.Errorf("invalid evidence graph leaf")
		}
		data, err := readCommittedBlob(authority.Root, authority.Head, relativeTo(authority.Root, filepath.Join(transcriptPath(authority.Root, panel), filepath.FromSlash(path))))
		if err != nil || len(data) != size || digestBytes(data) != digest {
			return fmt.Errorf("evidence graph leaf mismatch: %s", path)
		}
		previous = path
	}
	return nil
}

func verifyCommittedReceipt(authority repositoryAuthority, panel string, report protectedReport) error {
	data, err := readCommittedBlob(authority.Root, authority.Head, relativeTo(authority.Root, receiptPath(authority.Root, panel)))
	if err != nil {
		return err
	}
	var wire []json.RawMessage
	if json.Unmarshal(data, &wire) != nil || len(wire) != 11 {
		return fmt.Errorf("invalid committed receipt")
	}
	var version, gotPanel, state, head, implementation, plan, started, root, fixture, reportDigest, graph string
	targets := []any{&version, &gotPanel, &state, &head, &implementation, &plan, &started, &root, &fixture, &reportDigest, &graph}
	for index := range targets {
		if json.Unmarshal(wire[index], targets[index]) != nil {
			return fmt.Errorf("invalid receipt value")
		}
	}
	reportBytes, _ := canonicalProtectedReport(report)
	rootValid := root == ""
	if panel == "locked" {
		rootValid = isLowerHex(root, 64)
	}
	if version != "transform-attempt/v1" || gotPanel != panel || state != "published" || implementation != authority.Reviews.ImplementationCommit || plan != PlanCommit || gitCommand(authority.Root, "merge-base", "--is-ancestor", head, authority.Head).Run() != nil || fixture != report.Payload.FixtureRoot || reportDigest != digestBytes(reportBytes) || graph != report.Payload.EvidenceGraph || !rootValid {
		return fmt.Errorf("committed receipt provenance mismatch")
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000000000Z", started); err != nil {
		return fmt.Errorf("invalid receipt timestamp")
	}
	return nil
}

func readCommittedBlob(root, head, path string) ([]byte, error) {
	entry, err := gitOutput(root, "ls-tree", head, "--", path)
	if err != nil || !strings.HasPrefix(entry, "100644 blob ") || !strings.HasSuffix(entry, "\t"+path) {
		return nil, fmt.Errorf("%s is not a committed regular blob", path)
	}
	data, err := gitFileAtCommit(root, head, path)
	if err != nil {
		return nil, err
	}
	working, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil || !bytes.Equal(data, working) {
		return nil, fmt.Errorf("%s differs from committed bytes", path)
	}
	return data, nil
}

func gitFileAtCommit(root, commit, path string) ([]byte, error) {
	if strings.ContainsAny(commit+path, "\x00\n") || strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid Git object path")
	}
	return gitCommand(root, "show", commit+":"+path).Output()
}

func gitOutput(root string, args ...string) (string, error) {
	output, err := gitCommand(root, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitCommand(root string, args ...string) *exec.Cmd {
	command := exec.Command("git", append([]string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "core.hooksPath=/dev/null", "-C", root}, args...)...)
	environment := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		name := strings.SplitN(entry, "=", 2)[0]
		if !strings.HasPrefix(name, "GIT_") {
			environment = append(environment, entry)
		}
	}
	command.Env = append(environment, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0")
	return command
}

func reportPath(root, panel string) string {
	return filepath.Join(root, ".nous", "transform-schema-v1-"+panel+"-report.json")
}
func transcriptPath(root, panel string) string {
	return filepath.Join(root, ".nous", "transform-schema-v1-"+panel+"-transcripts")
}
func receiptPath(root, panel string) string {
	return filepath.Join(root, ".nous", "transform-schema-v1-"+panel+"-receipt.json")
}
func relativeTo(root, path string) string {
	value, _ := filepath.Rel(root, path)
	return filepath.ToSlash(value)
}

func requireAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("protected destination already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeExclusiveSync(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}
