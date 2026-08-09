package nogoodexp

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	ReviewManifestPath = "docs/nogood-v2-implementation-reviews.json"
	receiptVersion     = "nogood-attempt/v2"
)

type ImplementationReview struct {
	Scope          string `json:"scope"`
	Status         string `json:"status"`
	ReviewedCommit string `json:"reviewed_commit"`
}

type ImplementationReviewManifest struct {
	Version              string                 `json:"version"`
	PlanCommit           string                 `json:"plan_commit"`
	ImplementationCommit string                 `json:"implementation_commit"`
	Reviews              []ImplementationReview `json:"reviews"`
	ProtectedPaths       map[string]string      `json:"protected_paths"`
}

type AttemptReceipt struct {
	Version               string `json:"version"`
	Panel                 string `json:"panel"`
	State                 string `json:"state"`
	Head                  string `json:"head"`
	ImplementationCommit  string `json:"implementation_commit"`
	PlanCommit            string `json:"plan_commit"`
	StartedUTC            string `json:"started_utc"`
	RootSHA256            string `json:"root_sha256,omitempty"`
	ReportSHA256          string `json:"report_sha256,omitempty"`
	RootManifestSHA256    string `json:"root_manifest_sha256,omitempty"`
	FixtureBundleSHA256   string `json:"fixture_bundle_sha256,omitempty"`
	PrimaryManifestSHA256 string `json:"primary_manifest_sha256,omitempty"`
	AuditManifestSHA256   string `json:"audit_manifest_sha256,omitempty"`
	EvidenceGraphSHA256   string `json:"evidence_graph_sha256,omitempty"`
}

type repositoryAuthority struct {
	root    string
	head    string
	reviews ImplementationReviewManifest
}

func ExecuteDevelopment(repoRoot, domainsDir string) (Report, error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return Report{}, err
	}
	if err := requireAbsent(reportPath(authority.root, "development")); err != nil {
		return Report{}, err
	}
	if err := requireAbsent(transcriptPath(authority.root, "development")); err != nil {
		return Report{}, err
	}
	tasks, err := developmentTasks()
	if err != nil {
		return Report{}, err
	}
	fixtureJSON, err := encodeFixtureBundle("development", tasks)
	if err != nil {
		return Report{}, err
	}
	if err := claimDevelopmentEvidenceRoot(authority.root); err != nil {
		return Report{}, err
	}
	materialized, err := materializeClaimedFixtureBundle(authority.root, "development", fixtureJSON)
	if err != nil {
		return Report{}, err
	}
	evidence, err := buildPanelEvidenceFromFixtures(filepath.Join(authority.root, "domains"), authority.reviews.ImplementationCommit, "development", materialized, "development", publicStatisticsAuthority("development", 832001), EstimateDevelopmentPower)
	if err != nil {
		return Report{}, err
	}
	if err := persistEvidence(authority.root, "development", evidence, true); err != nil {
		return Report{}, err
	}
	return evidence.Report, nil
}

func ExecuteValidation(repoRoot, domainsDir string) (report Report, returnErr error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return Report{}, err
	}
	development, err := verifyCommittedEvidence(authority, "development")
	if err != nil {
		return Report{}, fmt.Errorf("validation prerequisite: %w", err)
	}
	if !development.Payload.DevelopmentPower.Authorized {
		return Report{}, fmt.Errorf("development power does not authorize validation")
	}
	receipt, err := claimAttempt(authority, "validation", "")
	if err != nil {
		return Report{}, err
	}
	defer func() {
		if returnErr != nil {
			_ = finalizeAttempt(authority.root, receipt, "invalid", nil)
		}
	}()
	if err := startAttempt(authority.root, receipt, ""); err != nil {
		return Report{}, err
	}
	tasks, err := validationPanel()
	if err != nil {
		return Report{}, err
	}
	fixtureJSON, err := encodeFixtureBundle("validation", tasks)
	if err != nil {
		return Report{}, err
	}
	materialized, err := materializeClaimedFixtureBundle(authority.root, "validation", fixtureJSON)
	if err != nil {
		return Report{}, err
	}
	evidence, err := buildPanelEvidenceFromFixtures(filepath.Join(authority.root, "domains"), authority.reviews.ImplementationCommit, "validation", materialized, "validation", publicStatisticsAuthority("validation", 833001), func(PanelExecution) (PowerEstimate, error) {
		return development.Payload.DevelopmentPower, nil
	})
	if err != nil {
		return Report{}, err
	}
	if err := persistEvidence(authority.root, "validation", evidence, true); err != nil {
		return Report{}, err
	}
	if err := finalizeAttempt(authority.root, receipt, "published", &evidence); err != nil {
		return Report{}, err
	}
	return evidence.Report, nil
}

func ExecuteLocked(repoRoot, domainsDir, unlockToken string) (report Report, returnErr error) {
	authority, err := authorizeRepository(repoRoot, domainsDir)
	if err != nil {
		return Report{}, err
	}
	if unlockToken != "nogoods/v2:"+authority.head {
		return Report{}, fmt.Errorf("locked unlock token does not name exact clean HEAD")
	}
	validation, err := verifyCommittedEvidence(authority, "validation")
	if err != nil {
		return Report{}, fmt.Errorf("locked prerequisite: %w", err)
	}
	if !validation.Payload.DevelopmentPower.Authorized {
		return Report{}, fmt.Errorf("committed development power does not authorize locked execution")
	}
	receipt, err := claimAttempt(authority, "locked", "")
	if err != nil {
		return Report{}, err
	}
	defer func() {
		if returnErr != nil {
			_ = finalizeAttempt(authority.root, receipt, "invalid", nil)
		}
	}()
	rootBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, rootBytes); err != nil {
		return Report{}, err
	}
	if err := startAttempt(authority.root, receipt, digestHex(rootBytes)); err != nil {
		return Report{}, err
	}
	privateRoot := hex.EncodeToString(rootBytes)
	tasks, err := lockedPanel(privateRoot)
	if err != nil {
		privateRoot = ""
		for index := range rootBytes {
			rootBytes[index] = 0
		}
		return Report{}, err
	}
	fixtureJSON, fixtureErr := encodeFixtureBundle("locked", tasks)
	tasks = nil
	inferenceAuthority := lockedStatisticsAuthority(privateRoot)
	privateRoot = ""
	for index := range rootBytes {
		rootBytes[index] = 0
	}
	if fixtureErr != nil {
		return Report{}, fixtureErr
	}
	materialized, err := materializeClaimedFixtureBundle(authority.root, "locked", fixtureJSON)
	if err != nil {
		return Report{}, err
	}
	evidence, err := buildPanelEvidenceFromFixtures(filepath.Join(authority.root, "domains"), authority.reviews.ImplementationCommit, "locked", materialized, "validation", inferenceAuthority, func(PanelExecution) (PowerEstimate, error) {
		return validation.Payload.DevelopmentPower, nil
	})
	inferenceAuthority = statisticsAuthority{}
	if err != nil {
		return Report{}, err
	}
	if err := persistEvidence(authority.root, "locked", evidence, true); err != nil {
		return Report{}, err
	}
	if err := finalizeAttempt(authority.root, receipt, "published", &evidence); err != nil {
		return Report{}, err
	}
	return evidence.Report, nil
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
	wantDomains := filepath.Join(root, "domains")
	domains, err := filepath.Abs(domainsDir)
	if err != nil {
		return repositoryAuthority{}, err
	}
	domains, err = filepath.EvalSymlinks(domains)
	if err != nil || domains != wantDomains {
		return repositoryAuthority{}, fmt.Errorf("domains path is not canonical repository domains/")
	}
	if _, err := os.Lstat(filepath.Join(root, "go.work")); err == nil {
		return repositoryAuthority{}, fmt.Errorf("go.work is forbidden for nogood evidence execution")
	} else if !os.IsNotExist(err) {
		return repositoryAuthority{}, err
	}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return repositoryAuthority{}, err
	}
	if bytes.Contains(module, []byte("replace ")) || bytes.Contains(module, []byte("replace(")) || bytes.Contains(module, []byte("replace (")) {
		return repositoryAuthority{}, fmt.Errorf("module replacements are forbidden")
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return repositoryAuthority{}, err
	}
	status, err := gitOutput(root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return repositoryAuthority{}, err
	}
	if status != "" {
		return repositoryAuthority{}, fmt.Errorf("repository is not clean")
	}
	if _, err := gitOutput(root, "ls-files", "--error-unmatch", ReviewManifestPath); err != nil {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest is not committed")
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, ReviewManifestPath))
	if err != nil {
		return repositoryAuthority{}, err
	}
	var manifest ImplementationReviewManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return repositoryAuthority{}, err
	}
	canonical, err := canonicalJSON(manifest)
	if err != nil || !bytes.Equal(canonical, manifestBytes) {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest is not canonical")
	}
	if manifest.Version != "nogood-implementation-reviews/v2" || manifest.PlanCommit != PlanCommit || len(manifest.ImplementationCommit) != 40 {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest identity is invalid")
	}
	if err := exec.Command("git", "-C", root, "merge-base", "--is-ancestor", manifest.ImplementationCommit, head).Run(); err != nil {
		return repositoryAuthority{}, fmt.Errorf("reviewed implementation is not an ancestor of clean HEAD")
	}
	wantScopes := []string{"architecture", "constraint-semantics", "experimental-validity"}
	if len(manifest.Reviews) != len(wantScopes) {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest does not contain three reviews")
	}
	for index, review := range manifest.Reviews {
		if review.Scope != wantScopes[index] || review.Status != "accepted" || review.ReviewedCommit != manifest.ImplementationCommit {
			return repositoryAuthority{}, fmt.Errorf("implementation review %d is not exact accepted authority", index)
		}
	}
	if len(manifest.ProtectedPaths) == 0 {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest has no protected paths")
	}
	requiredPaths, err := requiredProtectedPaths(root)
	if err != nil {
		return repositoryAuthority{}, err
	}
	if len(manifest.ProtectedPaths) != len(requiredPaths) {
		return repositoryAuthority{}, fmt.Errorf("implementation review manifest protects %d paths, want exact %d-path source surface", len(manifest.ProtectedPaths), len(requiredPaths))
	}
	for _, path := range requiredPaths {
		if _, ok := manifest.ProtectedPaths[path]; !ok {
			return repositoryAuthority{}, fmt.Errorf("implementation review manifest omits protected path %s", path)
		}
	}
	for path, wantDigest := range manifest.ProtectedPaths {
		if filepath.IsAbs(path) || strings.Contains(path, "..") || len(wantDigest) != 64 {
			return repositoryAuthority{}, fmt.Errorf("invalid protected path entry %q", path)
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || digestHex(data) != wantDigest {
			return repositoryAuthority{}, fmt.Errorf("protected path changed: %s", path)
		}
		reviewedData, err := gitFileAtCommit(root, manifest.ImplementationCommit, path)
		if err != nil || digestHex(reviewedData) != wantDigest {
			return repositoryAuthority{}, fmt.Errorf("protected path was not reviewed at implementation commit: %s", path)
		}
	}
	return repositoryAuthority{root: root, head: head, reviews: manifest}, nil
}

func gitFileAtCommit(repoRoot, commit, path string) ([]byte, error) {
	if strings.Contains(path, "\x00") || strings.Contains(commit, "\x00") {
		return nil, fmt.Errorf("invalid git object path")
	}
	command := exec.Command("git", "-C", repoRoot, "show", commit+":"+path)
	return command.Output()
}

func requiredProtectedPaths(repoRoot string) ([]string, error) {
	tracked, err := gitOutput(repoRoot, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range strings.Split(tracked, "\x00") {
		if path == "" || path == ReviewManifestPath {
			continue
		}
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".cue") || path == "go.mod" || path == "go.sum" || path == "mise.toml" || path == "docs/constraint-nogood-learning-vocabulary-plan.md" || path == "docs/vocabulary-research-program-v3.md" {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("protected source surface is empty")
	}
	return paths, nil
}

func claimAttempt(authority repositoryAuthority, panel, rootDigest string) (*AttemptReceipt, error) {
	if panel != "validation" && panel != "locked" {
		return nil, fmt.Errorf("cannot claim unprotected panel %q", panel)
	}
	base := filepath.Join(authority.root, ".nous")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	for _, path := range []string{receiptPath(authority.root, panel), reportPath(authority.root, panel), transcriptPath(authority.root, panel)} {
		if err := requireAbsent(path); err != nil {
			return nil, err
		}
	}
	receipt := &AttemptReceipt{Version: receiptVersion, Panel: panel, State: "claimed", Head: authority.head, ImplementationCommit: authority.reviews.ImplementationCommit, PlanCommit: PlanCommit, StartedUTC: time.Now().UTC().Format(time.RFC3339Nano), RootSHA256: rootDigest}
	encoded, err := canonicalJSON(receipt)
	if err != nil {
		return nil, err
	}
	if err := writeExclusiveSync(receiptPath(authority.root, panel), encoded, 0o600); err != nil {
		return nil, err
	}
	if err := os.Mkdir(transcriptPath(authority.root, panel), 0o755); err != nil {
		return nil, err
	}
	if err := syncDirectory(base); err != nil {
		return nil, err
	}
	return receipt, nil
}

func startAttempt(repoRoot string, receipt *AttemptReceipt, rootDigest string) error {
	receipt.RootSHA256 = rootDigest
	return rewriteReceipt(repoRoot, receipt, "started")
}

func finalizeAttempt(repoRoot string, receipt *AttemptReceipt, state string, evidence *DevelopmentEvidence) error {
	if evidence != nil {
		receipt.ReportSHA256 = digestHex(evidence.ReportJSON)
		receipt.RootManifestSHA256 = digestHex(evidence.Bundle.RootManifestJSON)
		receipt.FixtureBundleSHA256 = digestHex(evidence.FixtureJSON)
		receipt.PrimaryManifestSHA256 = digestHex(evidence.Bundle.PrimaryManifestJSON)
		receipt.AuditManifestSHA256 = digestHex(evidence.Bundle.AuditManifestJSON)
		receipt.EvidenceGraphSHA256 = evidenceGraphDigest(*evidence)
	}
	return rewriteReceipt(repoRoot, receipt, state)
}

func evidenceGraphDigest(evidence DevelopmentEvidence) string {
	graph := struct {
		Report, Root, Fixture, Primary, Audit string
		Chunks                                []string
	}{
		Report: digestHex(evidence.ReportJSON), Root: digestHex(evidence.Bundle.RootManifestJSON),
		Fixture: digestHex(evidence.FixtureJSON), Primary: digestHex(evidence.Bundle.PrimaryManifestJSON),
		Audit: digestHex(evidence.Bundle.AuditManifestJSON),
	}
	for _, execution := range []PanelExecution{evidence.Primary, evidence.Audit} {
		for _, policy := range execution.Policies {
			graph.Chunks = append(graph.Chunks, digestHex(policy.Transcript.Gzip))
		}
	}
	encoded, err := canonicalJSON(graph)
	if err != nil {
		panic(err)
	}
	return digestHex(encoded)
}

func rewriteReceipt(repoRoot string, receipt *AttemptReceipt, state string) error {
	receipt.State = state
	encoded, err := canonicalJSON(receipt)
	if err != nil {
		return err
	}
	path := receiptPath(repoRoot, receipt.Panel)
	temporary := path + ".final"
	if err := writeExclusiveSync(temporary, encoded, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func verifyCommittedEvidence(authority repositoryAuthority, panel string) (Report, error) {
	relativeReport, err := filepath.Rel(authority.root, reportPath(authority.root, panel))
	if err != nil {
		return Report{}, err
	}
	if _, err := gitOutput(authority.root, "ls-files", "--error-unmatch", filepath.ToSlash(relativeReport)); err != nil {
		return Report{}, fmt.Errorf("%s report is not committed", panel)
	}
	reportBytes, err := os.ReadFile(reportPath(authority.root, panel))
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		return Report{}, err
	}
	canonical, err := canonicalJSON(report)
	if err != nil || !bytes.Equal(canonical, reportBytes) || len(reportBytes) > ReportByteCap {
		return Report{}, fmt.Errorf("%s report is not canonical/bounded", panel)
	}
	payload, err := canonicalJSON(report.Payload)
	if err != nil || digestHex(payload) != report.PayloadSHA256 || report.Payload.Panel != panel || report.Payload.PlanCommit != PlanCommit || report.Payload.ImplementationCommit != authority.reviews.ImplementationCommit || !allMechanicalGates(report.Payload.Gates) {
		return Report{}, fmt.Errorf("%s report payload failed verification", panel)
	}
	wantClassification := stageClassification(panel, report.Payload.Inference, report.Payload.DevelopmentPower)
	if report.Classification != wantClassification {
		return Report{}, fmt.Errorf("%s report has invalid stage classification %q", panel, report.Classification)
	}
	if err := verifyEvidenceFiles(authority.root, panel, report); err != nil {
		return Report{}, err
	}
	if panel == "validation" {
		if err := verifyCommittedReceipt(authority, report); err != nil {
			return Report{}, err
		}
	}
	return report, nil
}

func verifyCommittedReceipt(authority repositoryAuthority, report Report) error {
	path := receiptPath(authority.root, "validation")
	relative, err := filepath.Rel(authority.root, path)
	if err != nil {
		return err
	}
	if _, err := gitOutput(authority.root, "ls-files", "--error-unmatch", filepath.ToSlash(relative)); err != nil {
		return fmt.Errorf("validation receipt is not committed")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var receipt AttemptReceipt
	if err := json.Unmarshal(encoded, &receipt); err != nil {
		return err
	}
	canonical, err := canonicalJSON(receipt)
	if err != nil || !bytes.Equal(canonical, encoded) || receipt.Version != receiptVersion || receipt.Panel != "validation" || receipt.State != "published" || receipt.PlanCommit != PlanCommit || receipt.ImplementationCommit != authority.reviews.ImplementationCommit {
		return fmt.Errorf("validation receipt identity is invalid")
	}
	if len(receipt.Head) != 40 || exec.Command("git", "-C", authority.root, "merge-base", "--is-ancestor", receipt.Head, authority.head).Run() != nil {
		return fmt.Errorf("validation receipt execution head is not an ancestor of clean HEAD")
	}
	rootBytes, err := os.ReadFile(filepath.Join(transcriptPath(authority.root, "validation"), "manifest.json"))
	if err != nil {
		return err
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(transcriptPath(authority.root, "validation"), "fixtures.json"))
	if err != nil {
		return err
	}
	primaryBytes, err := os.ReadFile(filepath.Join(transcriptPath(authority.root, "validation"), "primary", "execution-manifest.json"))
	if err != nil {
		return err
	}
	auditBytes, err := os.ReadFile(filepath.Join(transcriptPath(authority.root, "validation"), "audit", "execution-manifest.json"))
	if err != nil {
		return err
	}
	wantGraph, err := evidenceGraphFilesDigest(authority.root, "validation", report, rootBytes, fixtureBytes, primaryBytes, auditBytes)
	if err != nil {
		return err
	}
	if receipt.ReportSHA256 != digestHex(mustCanonicalReport(report)) || receipt.RootManifestSHA256 != digestHex(rootBytes) || receipt.FixtureBundleSHA256 != digestHex(fixtureBytes) || receipt.PrimaryManifestSHA256 != digestHex(primaryBytes) || receipt.AuditManifestSHA256 != digestHex(auditBytes) || receipt.EvidenceGraphSHA256 != wantGraph {
		return fmt.Errorf("validation receipt evidence graph mismatch")
	}
	return nil
}

func mustCanonicalReport(report Report) []byte {
	encoded, err := canonicalJSON(report)
	if err != nil {
		panic(err)
	}
	return encoded
}

func evidenceGraphFilesDigest(repoRoot, panel string, report Report, rootBytes, fixtureBytes, primaryBytes, auditBytes []byte) (string, error) {
	graph := struct {
		Report, Root, Fixture, Primary, Audit string
		Chunks                                []string
	}{
		Report: digestHex(mustCanonicalReport(report)), Root: digestHex(rootBytes), Fixture: digestHex(fixtureBytes),
		Primary: digestHex(primaryBytes), Audit: digestHex(auditBytes),
	}
	for _, role := range []string{"primary", "audit"} {
		for _, policy := range RequiredPolicies {
			data, err := os.ReadFile(filepath.Join(transcriptPath(repoRoot, panel), role, policy+".ngt.gz"))
			if err != nil {
				return "", err
			}
			graph.Chunks = append(graph.Chunks, digestHex(data))
		}
	}
	encoded, err := canonicalJSON(graph)
	if err != nil {
		return "", err
	}
	return digestHex(encoded), nil
}

func verifyEvidenceFiles(repoRoot, panel string, report Report) error {
	rootBytes, err := os.ReadFile(filepath.Join(transcriptPath(repoRoot, panel), "manifest.json"))
	if err != nil || digestHex(rootBytes) != report.RootManifestSHA256 {
		return fmt.Errorf("%s root manifest digest mismatch", panel)
	}
	var root RootManifest
	if err := json.Unmarshal(rootBytes, &root); err != nil {
		return err
	}
	canonical, _ := canonicalJSON(root)
	if !bytes.Equal(canonical, rootBytes) || root.Panel != panel || root.ReportPayloadSHA256 != report.PayloadSHA256 || root.FinalReportReference != fmt.Sprintf("nogoods-v2-%s-report.json", panel) {
		return fmt.Errorf("%s root manifest is invalid", panel)
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(transcriptPath(repoRoot, panel), "fixtures.json"))
	if err != nil || len(fixtureBytes) != root.FixtureBundleSize || len(fixtureBytes) > FixtureBundleByteCap || digestHex(fixtureBytes) != root.FixtureBundleSHA256 {
		return fmt.Errorf("%s fixture bundle digest mismatch", panel)
	}
	if _, err := decodeFixtureBundle(panel, fixtureBytes); err != nil {
		return fmt.Errorf("%s fixture bundle is invalid: %w", panel, err)
	}
	var manifests [2]ExecutionManifest
	for index, role := range []string{"primary", "audit"} {
		manifestBytes, err := os.ReadFile(filepath.Join(transcriptPath(repoRoot, panel), role, "execution-manifest.json"))
		if err != nil {
			return err
		}
		wantDigest := root.PrimaryExecutionSHA256
		if role == "audit" {
			wantDigest = root.AuditExecutionSHA256
		}
		if digestHex(manifestBytes) != wantDigest || json.Unmarshal(manifestBytes, &manifests[index]) != nil || manifests[index].ExecutionRole != role || len(manifests[index].Policies) != PolicyCount {
			return fmt.Errorf("%s %s execution manifest is invalid", panel, role)
		}
		canonicalManifest, _ := canonicalJSON(manifests[index])
		if !bytes.Equal(canonicalManifest, manifestBytes) {
			return fmt.Errorf("%s %s execution manifest is not canonical", panel, role)
		}
		for policyIndex, chunk := range manifests[index].Policies {
			if chunk.Policy != RequiredPolicies[policyIndex] {
				return fmt.Errorf("%s %s policy order mismatch", panel, role)
			}
			gzipBytes, err := os.ReadFile(filepath.Join(transcriptPath(repoRoot, panel), role, chunk.Policy+".ngt.gz"))
			if err != nil || len(gzipBytes) != chunk.GzipSize || digestHex(gzipBytes) != chunk.GzipSHA256 {
				return fmt.Errorf("%s %s chunk %s mismatch", panel, role, chunk.Policy)
			}
			compressedReader := bytes.NewReader(gzipBytes)
			reader, err := gzip.NewReader(compressedReader)
			if err != nil || reader.Name != "" || reader.Comment != "" || !reader.ModTime.IsZero() {
				return fmt.Errorf("%s chunk gzip header invalid", chunk.Policy)
			}
			reader.Multistream(false)
			raw, err := io.ReadAll(reader)
			closeErr := reader.Close()
			canonicalGzipBytes, canonicalErr := canonicalGzip(raw)
			if err != nil || closeErr != nil || compressedReader.Len() != 0 || canonicalErr != nil || !bytes.Equal(canonicalGzipBytes, gzipBytes) || len(raw) != chunk.RawSize || digestHex(raw) != chunk.RawSHA256 {
				return fmt.Errorf("%s chunk raw stream mismatch", chunk.Policy)
			}
			decoded, err := DecodeTranscript(raw)
			if err != nil || transcriptWorkFromVector(decoded.Vector) != chunk.EventCount {
				return fmt.Errorf("%s chunk transcript reduction failed", chunk.Policy)
			}
		}
	}
	for index := range manifests[0].Policies {
		if manifests[0].Policies[index].RawSHA256 != manifests[1].Policies[index].RawSHA256 || manifests[0].Policies[index].GzipSHA256 != manifests[1].Policies[index].GzipSHA256 {
			return fmt.Errorf("%s dual transcript hash mismatch at policy %d", panel, index)
		}
	}
	return nil
}

func allMechanicalGates(gates MechanicalGates) bool {
	return gates.ManifestValid && gates.CompetencePassed && gates.DualExecutionEqual && gates.TranscriptHashesEqual && gates.TranscriptConservation && gates.OracleParity && gates.PrunesSound
}

func transcriptWorkFromVector(vector [12]int64) int64 {
	var work int64
	for _, count := range vector {
		work += count
	}
	return work
}

func gitOutput(repoRoot string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func reportPath(repoRoot, panel string) string {
	return filepath.Join(repoRoot, ".nous", fmt.Sprintf("nogoods-v2-%s-report.json", panel))
}

func transcriptPath(repoRoot, panel string) string {
	return filepath.Join(repoRoot, ".nous", fmt.Sprintf("nogoods-v2-%s-transcripts", panel))
}

func receiptPath(repoRoot, panel string) string {
	return filepath.Join(repoRoot, ".nous", fmt.Sprintf("nogoods-v2-%s-receipt.json", panel))
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
