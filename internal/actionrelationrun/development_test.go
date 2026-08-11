package actionrelationrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
)

func TestPanelStartMarkerIsExactAndNoReplace(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	common, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prerequisites := panelPrerequisites{
		Root: root, GitCommonDir: common, Head: "1111111111111111111111111111111111111111",
		Build: actionrelationexp.BuildAuthority{SourceRoot: digest([]byte("source"))},
	}
	lifecycle := panelLifecycleAuthority{AttemptCommitment: "0000000000000000000000000000000000000000000000000000000000000000"}
	start, exists, err := inspectPanelStart(prerequisites, "development", lifecycle)
	if err != nil || exists {
		t.Fatalf("fresh marker inspection: exists=%v err=%v", exists, err)
	}
	started, err := consumePanelStart(start)
	if err != nil || !started {
		t.Fatalf("consume marker: started=%v err=%v", started, err)
	}
	data, _, err := readRegularNoFollowAllowLinks(start.Path, 0o600)
	if err != nil || !bytes.Equal(data, start.Bytes) {
		t.Fatalf("marker bytes changed: %q %v", data, err)
	}
	if _, exists, err := inspectPanelStart(prerequisites, "development", lifecycle); err != nil || !exists {
		t.Fatalf("existing marker was not recoverable: exists=%v err=%v", exists, err)
	}
	if started, err := consumePanelStart(start); err == nil || started {
		t.Fatalf("second marker transition succeeded: started=%v err=%v", started, err)
	}
}

func TestStartedDevelopmentRecoversToOneInvalidTuple(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	common, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prerequisites := panelPrerequisites{
		Root: root, GitCommonDir: common, Head: "2222222222222222222222222222222222222222",
		Build: actionrelationexp.BuildAuthority{SourceRoot: digest([]byte("source"))},
	}
	lifecycle := panelLifecycleAuthority{AttemptCommitment: "0000000000000000000000000000000000000000000000000000000000000000"}
	start, _ := panelStartAuthority(prerequisites, "development", lifecycle)
	if started, err := consumePanelStart(start); err != nil || !started {
		t.Fatal(err)
	}
	if _, err := recoverStartedPanel(prerequisites, "development", lifecycle); err == nil {
		t.Fatal("interrupted attempt did not become invalid")
	}
	receiptPath := filepath.Join(root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath("development", "terminal-receipt")))
	first, err := readRegularNoFollow(receiptPath, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := actionrelationexp.ParseTerminalReceipt(first)
	if err != nil || receipt.State != "invalid" || receipt.Reason != interruptedAttemptReason {
		t.Fatalf("unexpected recovered receipt: %+v %v", receipt, err)
	}
	if err := publishInvalidPanel(prerequisites, "development", lifecycle, errors.New("different later error")); err != nil {
		t.Fatal(err)
	}
	second, err := readRegularNoFollow(receiptPath, 0o644)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("idempotent recovery changed receipt: %v", err)
	}
}

func TestInvalidRecoveryRejectsDivergentReceiptWithoutOverwrite(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prerequisites := panelPrerequisites{Root: root, Build: actionrelationexp.BuildAuthority{SourceRoot: digest([]byte("source"))}}
	lifecycle := panelLifecycleAuthority{AttemptCommitment: "0000000000000000000000000000000000000000000000000000000000000000"}
	if err := publishInvalidPanel(prerequisites, "development", lifecycle, errors.New("first")); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath("development", "terminal-receipt")))
	if err := os.WriteFile(receiptPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishInvalidPanel(prerequisites, "development", lifecycle, errors.New("second")); err == nil {
		t.Fatal("divergent terminal receipt was accepted")
	}
	data, err := os.ReadFile(receiptPath)
	if err != nil || string(data) != "corrupt" {
		t.Fatalf("divergent receipt was overwritten: %q %v", data, err)
	}
}

func TestInvalidTerminalizationReferencesExactSuccessfulFixture(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prerequisites := panelPrerequisites{Root: root, Build: actionrelationexp.BuildAuthority{SourceRoot: digest([]byte("source"))}}
	roots := make([]string, 16)
	for index := range roots {
		roots[index] = digest([]byte{byte(index + 1)})
	}
	fixtureBytes, err := json.Marshal([]any{"actionrelation-fixture-root/v2", "development", "development-public-v1", roots, digest([]byte("scorer"))})
	if err != nil {
		t.Fatal(err)
	}
	path := actionrelationexp.ExpectedAuthorityPath("development", "fixture-root")
	physical := filepath.Join(root, filepath.FromSlash(path))
	if err := writeExclusiveAuthority(physical, fixtureBytes); err != nil {
		t.Fatal(err)
	}
	ref, err := ensureInvalidAuthorityRef(prerequisites, path, "development", "fixture-root", "0000000000000000000000000000000000000000000000000000000000000000", "failed")
	if err != nil || ref.Digest != digest(fixtureBytes) {
		t.Fatalf("successful fixture was not retained directly: %+v %v", ref, err)
	}
	changed, _ := json.Marshal([]any{"actionrelation-fixture-root/v2", "development", "wrong-authority", roots, digest([]byte("scorer"))})
	otherRoot, _ := filepath.EvalSymlinks(t.TempDir())
	otherPrerequisites := panelPrerequisites{Root: otherRoot, Build: prerequisites.Build}
	otherPhysical := filepath.Join(otherRoot, filepath.FromSlash(path))
	if err := writeExclusiveAuthority(otherPhysical, changed); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureInvalidAuthorityRef(otherPrerequisites, path, "development", "fixture-root", "0000000000000000000000000000000000000000000000000000000000000000", "failed"); err == nil {
		t.Fatal("mismatched successful fixture authority was accepted")
	}
}

func TestDevelopmentPostStartFailureRetainsClosedInvalidTerminal(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prerequisites := panelPrerequisites{Root: root, Build: actionrelationexp.BuildAuthority{SourceRoot: digest([]byte("source"))}}
	// Development's attempt commitment is the raw all-zero authority digest.
	lifecycle := panelLifecycleAuthority{AttemptCommitment: "0000000000000000000000000000000000000000000000000000000000000000"}
	if err := publishInvalidPanel(prerequisites, "development", lifecycle, errors.New("audit failed")); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, filepath.FromSlash(actionrelationexp.ExpectedAuthorityPath("development", "terminal-receipt")))
	receiptBytes, err := readRegularNoFollow(receiptPath, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := actionrelationexp.ParseTerminalReceipt(receiptBytes)
	if err != nil || receipt.State != "invalid" || receipt.Reason != "audit failed" {
		t.Fatalf("invalid terminal did not reconstruct: %+v %v", receipt, err)
	}
	for kind, ref := range map[string]actionrelationexp.AuthorityRef{"fixture-root": receipt.FixtureRoot, "report": receipt.Report, "evidence-payload": receipt.EvidencePayload} {
		data, err := readRegularNoFollow(filepath.Join(root, filepath.FromSlash(ref.Path)), 0o644)
		if err != nil {
			t.Fatal(err)
		}
		failure, err := actionrelationexp.ParseInvalidAuthority(data)
		if err != nil || failure.Kind != kind || failure.SourceRoot != prerequisites.Build.SourceRoot || failure.Reason != receipt.Reason {
			t.Fatalf("failure authority %s did not close receipt: %+v %v", kind, failure, err)
		}
	}
}

func TestPanelWriterAndAuditComparisonRequireExactFiles(t *testing.T) {
	primaryRoot, auditRoot := t.TempDir(), t.TempDir()
	logicalRoot := ".nous/actionrelations-v1-development-evidence"
	primary, audit := newPanelWriter(primaryRoot, logicalRoot, 1024), newPanelWriter(auditRoot, logicalRoot, 1024)
	files := []actionrelationexp.EvidenceFile{
		{Path: logicalRoot + "/packs/a.bin", Mode: "100644", Data: []byte("a")},
		{Path: logicalRoot + "/manifests/a.json", Mode: "100644", Data: []byte("manifest")},
	}
	if err := primary.writeAll(files); err != nil {
		t.Fatal(err)
	}
	if err := audit.writeAll(files); err != nil {
		t.Fatal(err)
	}
	if err := comparePanelFiles(primary, audit); err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(auditRoot, filepath.FromSlash(files[0].Path))
	if err := os.WriteFile(changed, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := comparePanelFiles(primary, audit); err == nil {
		t.Fatal("accepted changed audit bytes")
	}
	if err := primary.write(files[0]); err == nil {
		t.Fatal("accepted duplicate evidence path")
	}
}

func TestIsolatedSandboxAllowsOnlyWorkerOutputAndDeniesPrimaryRead(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Seatbelt profile is macOS-specific")
	}
	allowed, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	denied, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(denied, "primary.pack")
	outside := filepath.Join(t.TempDir(), "outside.pack")
	inside := filepath.Join(allowed, "audit.pack")
	if err := os.WriteFile(secret, []byte("primary"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := isolatedSandboxProfile("/bin/bash", allowed, []string{denied})
	run := func(script, path string) error {
		command := exec.Command("/usr/bin/sandbox-exec", "-p", profile, "/bin/bash", "--noprofile", "--norc", "-c", script, "worker", path)
		return command.Run()
	}
	if err := run(`printf isolated > "$1"`, inside); err != nil {
		t.Fatalf("sandbox rejected worker output: %v", err)
	}
	if err := run(`IFS= read -r value < "$1"`, secret); err == nil {
		t.Fatal("sandbox exposed primary output to audit")
	}
	if err := run(`printf escaped > "$1"`, outside); err == nil {
		t.Fatal("sandbox allowed a write outside the isolated namespace")
	}
}

func TestPanelWriterRejectsCapacityAndEscapingPath(t *testing.T) {
	root := ".nous/actionrelations-v1-development-evidence"
	writer := newPanelWriter(t.TempDir(), root, 1)
	if err := writer.write(actionrelationexp.EvidenceFile{Path: root + "/two", Mode: "100644", Data: []byte("xx")}); err == nil {
		t.Fatal("accepted evidence beyond capacity")
	}
	writer = newPanelWriter(t.TempDir(), root, 10)
	if err := writer.write(actionrelationexp.EvidenceFile{Path: root + "/../escape", Mode: "100644", Data: []byte("x")}); err == nil {
		t.Fatal("accepted escaping evidence path")
	}
}
