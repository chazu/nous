// Package actionrelationcap mints the unforgeable, one-attempt capability used
// by protected action-relation panels. A zero Token is inert; the only minting
// path verifies committed repository authority without mutating attempt state.
package actionrelationcap

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/chazu/nous/internal/actionrelationexp"
)

const validationAuthority = "validation-public-v1"

type grant struct {
	panel      string
	authority  string
	secret     []byte
	secretPath string
	mu         sync.Mutex
	uses       int
	destroyed  bool
}

// Token is deliberately opaque. Its zero value and copied values whose grant
// was not minted by Authorize cannot construct protected fixtures.
type Token struct {
	grant *grant
}

func (t Token) Panel() (string, bool) {
	if t.grant == nil {
		return "", false
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	if t.grant.destroyed || t.grant.panel != "validation" && t.grant.panel != "locked" {
		return "", false
	}
	return t.grant.panel, true
}

func (t Token) Authority() (string, bool) {
	if t.grant == nil {
		return "", false
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	if t.grant.destroyed || t.grant.panel != "validation" && t.grant.panel != "locked" || !digestOrValidation(t.grant.authority) {
		return "", false
	}
	return t.grant.authority, true
}

func (t Token) CurriculumSeed(curriculum int) (any, bool) {
	if t.grant == nil {
		return nil, false
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	if t.grant.destroyed || curriculum < 0 {
		return nil, false
	}
	if t.grant.panel == "validation" {
		if curriculum >= 24 || t.grant.authority != validationAuthority || len(t.grant.secret) != 0 {
			return nil, false
		}
		return 852001 + curriculum, true
	}
	if t.grant.panel != "locked" || curriculum >= 32 || len(t.grant.secret) != 32 || digestBytes(t.grant.secret) != t.grant.authority {
		return nil, false
	}
	preimage, _ := json.Marshal([]any{"actionrelation-locked-curriculum/v1", curriculum})
	mac := hmac.New(sha256.New, t.grant.secret)
	_, _ = mac.Write(preimage)
	return hex.EncodeToString(mac.Sum(nil)), true
}

func (t Token) VerifyCurriculumSeed(curriculum int, seed any) bool {
	want, ok := t.CurriculumSeed(curriculum)
	if !ok {
		return false
	}
	switch value := want.(type) {
	case int:
		got, valid := seed.(int)
		return valid && got == value
	case string:
		got, valid := seed.(string)
		return valid && hmac.Equal([]byte(got), []byte(value))
	default:
		return false
	}
}

// BeginConstruction consumes the sole protected construction authorized before
// the resulting sealed fixture is replayed by isolated policy workers.
func (t Token) BeginConstruction() (string, string, bool) {
	if t.grant == nil {
		return "", "", false
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	if t.grant.destroyed || t.grant.panel != "validation" && t.grant.panel != "locked" || !digestOrValidation(t.grant.authority) || t.grant.uses >= 1 {
		return "", "", false
	}
	t.grant.uses++
	return t.grant.panel, t.grant.authority, true
}

// Destroy erases the in-memory seed authority and, for locked panels, the
// mode-0600 Git-common preimage before any policy worker may start.
func (t Token) Destroy() error {
	if t.grant == nil {
		return nil
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	if t.grant.destroyed {
		return nil
	}
	for index := range t.grant.secret {
		t.grant.secret[index] = 0
	}
	t.grant.secret = nil
	if t.grant.secretPath != "" {
		if err := eraseSecretFile(t.grant.secretPath); err != nil {
			return err
		}
		t.grant.secretPath = ""
	}
	t.grant.destroyed = true
	return nil
}

// ReleaseForRetry erases only the in-memory copy of a capability when the
// durable start transition was not reached. A locked preimage remains at its
// committed location so a later fresh authorization can retry.
func (t Token) ReleaseForRetry() {
	if t.grant == nil {
		return
	}
	t.grant.mu.Lock()
	defer t.grant.mu.Unlock()
	for index := range t.grant.secret {
		t.grant.secret[index] = 0
	}
	t.grant.secret = nil
	t.grant.secretPath = ""
	t.grant.destroyed = true
}

// Authorize independently reopens committed protected authority and returns a
// capability without consuming the run package's durable start transition.
func Authorize(ctx context.Context, repoRoot, panel string) (Token, actionrelationexp.Claim, actionrelationexp.Running, error) {
	if panel != "validation" && panel != "locked" {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("invalid protected panel")
	}
	haveEnvironment := os.Environ()
	slices.Sort(haveEnvironment)
	if !slices.Equal(haveEnvironment, []string{"GOMAXPROCS=1", "LC_ALL=C", "PATH=/opt/homebrew/bin:/usr/bin:/bin", "TZ=UTC"}) {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected authorization requires exact execution environment")
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	git := func(args ...string) ([]byte, error) {
		command := exec.CommandContext(ctx, gitPath, append([]string{"-C", root}, args...)...)
		command.Env = []string{"GIT_ATTR_NOSYSTEM=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "TZ=UTC"}
		return command.Output()
	}
	headBytes, err := git("rev-parse", "HEAD")
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	head := strings.TrimSpace(string(headBytes))
	origin, originErr := git("rev-parse", "origin/main")
	branch, branchErr := git("symbolic-ref", "--short", "HEAD")
	if originErr != nil || branchErr != nil || head != strings.TrimSpace(string(origin)) || strings.TrimSpace(string(branch)) != "main" {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected authorization requires main at origin/main")
	}
	if _, err := git("diff", "--quiet", "HEAD", "--"); err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected authorization rejects tracked working-tree changes")
	}
	if _, err := git("diff", "--cached", "--quiet", "HEAD", "--"); err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected authorization rejects staged changes")
	}
	buildBytes, err := readCommittedWorking(root, git, head, actionrelationexp.BuildAuthorityPath)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	build, err := actionrelationexp.ParseBuildAuthority(buildBytes)
	if err != nil || actionrelationexp.VerifySourceCheckout(root, build.SourceRows) != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected source differs from reviewed build authority")
	}
	executable, err := os.Executable()
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	wantExecutable := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	wantExecutable, err = filepath.EvalSymlinks(wantExecutable)
	if err != nil || executable != wantExecutable {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected authorization requires reviewed panel binary")
	}
	binaryBytes, err := os.ReadFile(executable)
	if err != nil || digestBytes(binaryBytes) != build.BinaryDigest {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected executable differs from build authority")
	}
	claimPath := actionrelationexp.ExpectedAuthorityPath(panel, "claim")
	runningPath := actionrelationexp.ExpectedAuthorityPath(panel, "running")
	claimBytes, err := readCommittedWorking(root, git, head, claimPath)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	claim, err := actionrelationexp.ParseClaim(claimBytes)
	if err != nil || claim.Panel != panel || claim.SourceRoot != build.SourceRoot || panel == "validation" && claim.Authority != validationAuthority || panel == "locked" && claim.Authority != LockedClaimAuthority(claim.BaseCommit, claim.SourceRoot) {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected claim does not match build authority")
	}
	runningBytes, err := readCommittedWorking(root, git, head, runningPath)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	running, err := actionrelationexp.ParseRunning(runningBytes)
	if err != nil || running.Panel != panel || running.SourceRoot != build.SourceRoot || running.ClaimReceiptDigest != claim.Digest {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("protected running receipt does not close claim")
	}
	committedClaim, err := git("cat-file", "blob", running.ClaimCommit+":"+claimPath)
	if err != nil || !bytes.Equal(committedClaim, claim.Canonical) {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("running receipt names a different claim commit")
	}
	if _, err := git("merge-base", "--is-ancestor", claim.BaseCommit, running.ClaimCommit); err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("claim base is not an ancestor of claim commit")
	}
	if _, err := git("merge-base", "--is-ancestor", running.ClaimCommit, head); err != nil || running.ClaimCommit == head {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("running receipt was not committed after claim")
	}
	for _, kind := range []string{"terminal-receipt", "report"} {
		if err := requireAbsentNoFollow(filepath.Join(root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath(panel, kind)))); err != nil {
			return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
		}
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(panel)
	if err := requireAbsentNoFollow(filepath.Join(root, filepath.FromSlash(evidenceRoot))); err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	commonBytes, err := git("rev-parse", "--git-common-dir")
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	commonDir := strings.TrimSpace(string(commonBytes))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	commonDir, err = filepath.Abs(commonDir)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	commonDir, err = filepath.EvalSymlinks(commonDir)
	if err != nil {
		return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, err
	}
	var secret []byte
	var secretPath string
	authority := validationAuthority
	if panel == "validation" {
		if running.SecretLocationDigest != nil || running.AttemptCommitment != ValidationAttemptCommitment() {
			return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("validation attempt commitment changed")
		}
	} else {
		location := secretLocation(claim.Digest)
		if running.SecretLocationDigest == nil || *running.SecretLocationDigest != digestBytes([]byte(location)) {
			return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("locked secret location commitment changed")
		}
		secretPath = filepath.Join(commonDir, filepath.FromSlash(location))
		secret, err = readSecretNoFollow(secretPath)
		if err != nil || len(secret) != 32 || digestBytes(secret) != running.AttemptCommitment {
			return Token{}, actionrelationexp.Claim{}, actionrelationexp.Running{}, fmt.Errorf("locked secret preimage does not match running authority")
		}
		authority = running.AttemptCommitment
	}
	return Token{grant: &grant{panel: panel, authority: authority, secret: bytes.Clone(secret), secretPath: secretPath}}, claim, running, nil
}

func ValidationAttemptCommitment() string {
	wire, _ := json.Marshal([]any{"actionrelation-attempt-root/v1", "validation", validationAuthority})
	return digestBytes(wire)
}

func LockedClaimAuthority(baseCommit, sourceRoot string) string {
	wire, _ := json.Marshal([]any{"actionrelation-locked-claim-authority/v1", baseCommit, sourceRoot})
	return digestBytes(wire)
}

func LockedSecretLocation(claimDigest string) (string, string, error) {
	if !digestText(claimDigest) {
		return "", "", fmt.Errorf("invalid claim digest")
	}
	location := secretLocation(claimDigest)
	return location, digestBytes([]byte(location)), nil
}

func secretLocation(claimDigest string) string {
	return filepath.ToSlash(filepath.Join("nous-actionrelations-v1", "secrets", "locked-"+claimDigest+".root"))
}

func readCommittedWorking(root string, git func(...string) ([]byte, error), commit, path string) ([]byte, error) {
	tree, err := git("ls-tree", "-z", commit, "--", path)
	if err != nil || !bytes.HasSuffix(tree, append([]byte{'\t'}, append([]byte(path), 0)...)) || !bytes.HasPrefix(tree, []byte("100644 blob ")) || bytes.Count(tree, []byte{0}) != 1 {
		return nil, fmt.Errorf("committed protected authority is not one mode-100644 blob: %s", path)
	}
	committed, err := git("cat-file", "blob", commit+":"+path)
	if err != nil {
		return nil, fmt.Errorf("read committed protected authority %s: %w", path, err)
	}
	physical := filepath.Join(root, filepath.FromSlash(path))
	working, readErr := readRegularNoFollow(physical, 0o644, int64(len(committed)))
	if readErr != nil || !bytes.Equal(committed, working) {
		return nil, fmt.Errorf("protected authority differs from committed regular file: %s", path)
	}
	return working, nil
}

func digestBytes(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func digestText(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func digestOrValidation(value string) bool {
	return value == validationAuthority || digestText(value)
}
