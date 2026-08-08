package causalexpv2

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chazu/nous/internal/causalv2"
	"golang.org/x/sys/unix"
)

func diagnosticSnapshotForTest(t *testing.T) ([]string, diagnosticOuterSnapshot) {
	t.Helper()
	environment, worker := v5EnvironmentForTest(t)
	return environment, diagnosticOuterSnapshot{
		base: worker.base, home: worker.home, xdg: worker.xdg,
		baseDevice: worker.baseDevice, baseInode: worker.baseInode,
		homeDevice: worker.homeDevice, homeInode: worker.homeInode,
		xdgDevice: worker.xdgDevice, xdgInode: worker.xdgInode,
	}
}

func diagnosticPinnedGoEnvironment(t *testing.T) []string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	privateBase := t.TempDir()
	workerEnvironment, err := pinnedWorkerEnvironment(context.Background(), privateBase)
	if err != nil {
		t.Fatal(err)
	}
	return append(fixedGoEnvironment(filepath.Join(home, "go", "pkg", "mod"), filepath.Join(cache, "go-build"), t.TempDir()), workerEnvironment...)
}

func diagnosticPinnedGoCommand(t *testing.T, arguments ...string) *exec.Cmd {
	t.Helper()
	command := exec.Command(pinnedGoPath, arguments...)
	command.Env = diagnosticPinnedGoEnvironment(t)
	return command
}

func TestCacheDiagnosticSyntheticWorkerProcess(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "cache-diagnostic-worker" {
		return
	}
	input := os.NewFile(3, "diagnostic-input")
	output := os.NewFile(4, "diagnostic-output")
	if input == nil || output == nil {
		os.Exit(2)
	}
	encoded, err := io.ReadAll(input)
	if err != nil || len(encoded) == 0 {
		os.Exit(2)
	}
	if info, err := output.Stat(); err != nil || !info.IsDir() {
		os.Exit(2)
	}
	_, _ = fmt.Fprintln(os.Stderr, "regenerated replay report has the wrong training digest")
	os.Exit(1)
}

func TestCacheDiagnosticKilledOperatorProcess(t *testing.T) {
	marker := -1
	for index, argument := range os.Args {
		if argument == "cache-diagnostic-kill-harness" {
			marker = index
			break
		}
	}
	if marker < 0 {
		return
	}
	if marker+1 >= len(os.Args) {
		os.Exit(2)
	}
	base := os.Args[marker+1]
	privateBase := filepath.Join(base, "private")
	if err := os.Mkdir(privateBase, 0o700); err != nil {
		os.Exit(2)
	}
	environment, err := pinnedWorkerEnvironment(context.Background(), privateBase)
	if err != nil {
		os.Exit(2)
	}
	outer, err := captureDiagnosticOuterEnvelope(environment, privateBase)
	if err != nil {
		os.Exit(2)
	}
	executable, err := os.Executable()
	if err != nil {
		os.Exit(2)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		os.Exit(2)
	}
	digest, err := regularExecutableDigest(executable)
	if err != nil {
		os.Exit(2)
	}
	commonDirectory := filepath.Join(base, "common")
	output := filepath.Join(base, "output")
	for _, directory := range []string{commonDirectory, output, filepath.Join(base, "build"), filepath.Join(base, "execution")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			os.Exit(2)
		}
	}
	state := gitState{Root: base, CommonDir: commonDirectory, Head: strings.Repeat("h", 40), Clean: true}
	record := newCacheDiagnosticRecord(strings.Repeat("h", 40), digest, digest, strings.Repeat("e", 64), strings.Repeat("b", 64))
	if err := createCacheDiagnosticRecord(commonDirectory, record); err != nil {
		os.Exit(2)
	}
	worker := regenerationExecutable{Path: executable, PrefixArgs: []string{"-test.run=^TestCacheDiagnosticSyntheticWorkerProcess$", "--", "cache-diagnostic-worker"}, Environment: environment}
	hooks := cacheDiagnosticExecutionHooks{
		verifyExecutionState: func(string) error { return nil },
		afterStartedPersist:  func() error { os.Exit(86); return nil },
	}
	_, _ = executeCacheDiagnosticWorker(context.Background(), state, outer, filepath.Join(base, "build"), filepath.Join(base, "execution"), output, executable, worker, []byte("synthetic fixed input"), &record, hooks)
	os.Exit(2)
}

type diagnosticExecutionHarness struct {
	state             gitState
	outer             diagnosticOuterSnapshot
	buildWorktree     string
	executionWorktree string
	output            string
	operatorPath      string
	worker            regenerationExecutable
	record            cacheDiagnosticRecord
}

func newDiagnosticExecutionHarness(t *testing.T) *diagnosticExecutionHarness {
	t.Helper()
	environment, outer := diagnosticSnapshotForTest(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := regularExecutableDigest(executable)
	if err != nil {
		t.Fatal(err)
	}
	commonDirectory := t.TempDir()
	output := filepath.Join(t.TempDir(), "output")
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatal(err)
	}
	harness := &diagnosticExecutionHarness{
		state:             gitState{Root: t.TempDir(), CommonDir: commonDirectory, Head: strings.Repeat("h", 40), Clean: true},
		outer:             outer,
		buildWorktree:     t.TempDir(),
		executionWorktree: t.TempDir(),
		output:            output,
		operatorPath:      executable,
		worker: regenerationExecutable{
			Path: executable, PrefixArgs: []string{"-test.run=^TestCacheDiagnosticSyntheticWorkerProcess$", "--", "cache-diagnostic-worker"}, Environment: environment,
		},
		record: newCacheDiagnosticRecord(strings.Repeat("h", 40), digest, digest, strings.Repeat("e", 64), strings.Repeat("b", 64)),
	}
	harness.record.CreatedUTC = time.Unix(0, 0).UTC().Format(time.RFC3339)
	if err := createCacheDiagnosticRecord(commonDirectory, harness.record); err != nil {
		t.Fatal(err)
	}
	return harness
}

func readCacheDiagnosticRecordForTest(t *testing.T, commonDirectory string) cacheDiagnosticRecord {
	t.Helper()
	encoded, err := os.ReadFile(cacheDiagnosticRecordPath(commonDirectory))
	if err != nil {
		t.Fatal(err)
	}
	record, err := causalv2.StrictDecode[cacheDiagnosticRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		t.Fatalf("decode cache diagnostic receipt: %+v, %v", record, err)
	}
	return record
}

func assertDiagnosticWalkerVerdicts(t *testing.T, snapshot diagnosticOuterSnapshot, want bool) {
	t.Helper()
	current := currentAuditWorkerCacheRoots(snapshot) == nil
	candidate := candidateAuditWorkerCacheRoots(snapshot) == nil
	reference, err := referenceAuditWorkerCacheRoots(snapshot)
	if err != nil {
		t.Fatalf("reference audit failed mechanically: %v", err)
	}
	if current != want || candidate != want || reference.accepted != want {
		t.Fatalf("walker verdicts current=%v candidate=%v reference=%v want=%v", current, candidate, reference.accepted, want)
	}
	again, err := referenceAuditWorkerCacheRoots(snapshot)
	if err != nil || again != reference {
		t.Fatalf("reference audit was unstable: first=%+v second=%+v err=%v", reference, again, err)
	}
}

func TestCacheDiagnosticInputAndProtocolIdentities(t *testing.T) {
	protectedGeneratorCalls.Store(0)
	encoded, digest, err := buildCacheDiagnosticInput()
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != cacheDiagnosticInputBytes || sha256Hex(encoded) != cacheDiagnosticInputSHA256 || digest != cacheDiagnosticInputDigest {
		t.Fatalf("input identity bytes=%d sha=%s digest=%s", len(encoded), sha256Hex(encoded), digest)
	}
	if protectedGeneratorCalls.Load() != 0 {
		t.Fatal("diagnostic input called protected generation")
	}
	input, err := causalv2.StrictDecode[ReplayInput](encoded)
	if err != nil || input.PretrainingCommit != replayPretrainingCommit || input.EvidenceCommit != replayEvidenceCommit || input.TrainingDigest != strings.Repeat("a", 64) || input.BundleDigest != strings.Repeat("b", 64) {
		t.Fatalf("diagnostic input provenance changed: %+v err=%v", input, err)
	}
	if err := verifyCacheDiagnosticProtocolDigests(); err != nil {
		t.Fatal(err)
	}
	for bits := 0; bits < 8; bits++ {
		reference, current, candidate := bits&4 != 0, bits&2 != 0, bits&1 != 0
		want := cacheDiagnosticUnsafeDisagreement
		switch {
		case !reference && !current && !candidate:
			want = cacheDiagnosticContractRejection
		case reference && !current && candidate:
			want = cacheDiagnosticImplementationDefect
		case reference && current && candidate:
			want = cacheDiagnosticNonReproduction
		}
		if got := classifyCacheDiagnostic(reference, current, candidate); got != want {
			t.Fatalf("tuple %03b=%q, want %q", bits, got, want)
		}
	}
	if cacheDiagnosticFailure != "diagnostic-failure" {
		t.Fatal("diagnostic failure identity changed")
	}
}

func TestCacheDiagnosticWalkersAcceptedBoundaries(t *testing.T) {
	t.Run("empty and nested", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		if err := os.MkdirAll(filepath.Join(snapshot.home, "z", "b", "a"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(snapshot.home, "z", "b", "a", "entry"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(snapshot.xdg, "entry"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
	t.Run("exact entry cap split across roots", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		for _, root := range []string{snapshot.home, snapshot.xdg} {
			for index := 0; index < 2048; index++ {
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("entry-%04d", index)), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
	t.Run("exact path byte cap", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		populateDiagnosticPathBytes(t, snapshot.home, false)
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
	t.Run("exact depth cap with empty deepest directory", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		path := snapshot.home
		for index := 0; index < 32; index++ {
			path = filepath.Join(path, "d")
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
	t.Run("exact logical byte cap split across roots", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		for _, root := range []string{snapshot.home, snapshot.xdg} {
			file, err := os.OpenFile(filepath.Join(root, "sparse"), os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(32 << 20); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
	t.Run("enumeration and raw filename permutations", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		for _, name := range []string{"z", "a", "m", string([]byte{'r', 'a', 'w', '-', 0xff})} {
			if err := os.WriteFile(filepath.Join(snapshot.home, name), []byte(name), 0o600); err != nil {
				if strings.Contains(err.Error(), "illegal byte sequence") {
					t.Skip("filesystem rejects invalid UTF-8 names")
				}
				t.Fatal(err)
			}
		}
		assertDiagnosticWalkerVerdicts(t, snapshot, true)
	})
}

func populateDiagnosticPathBytes(t *testing.T, root string, over bool) {
	t.Helper()
	parent := filepath.Join(root, "d")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	longNames := 255
	if over {
		longNames++
	}
	for index := 0; index < 4095; index++ {
		length := 254
		if index < longNames {
			length = 255
		}
		prefix := fmt.Sprintf("%04x-", index)
		name := prefix + strings.Repeat("p", length-len(prefix))
		if err := os.WriteFile(filepath.Join(parent, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCacheDiagnosticWalkersRejectForbiddenAndOverLimitTrees(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, diagnosticOuterSnapshot)
	}{
		{"symlink", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			if err := os.Symlink("target", filepath.Join(snapshot.home, "link")); err != nil {
				t.Fatal(err)
			}
		}},
		{"hardlink", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			first := filepath.Join(snapshot.home, "first")
			if err := os.WriteFile(first, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(first, filepath.Join(snapshot.xdg, "second")); err != nil {
				t.Fatal(err)
			}
		}},
		{"special", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			if err := unix.Mkfifo(filepath.Join(snapshot.home, "fifo"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"entry over cap", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			for _, root := range []string{snapshot.home, snapshot.xdg} {
				for index := 0; index < 2049; index++ {
					if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("entry-%04d", index)), nil, 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}},
		{"path bytes over cap", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			populateDiagnosticPathBytes(t, snapshot.home, true)
		}},
		{"depth over cap", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			path := snapshot.home
			for index := 0; index < 33; index++ {
				path = filepath.Join(path, "d")
			}
			if err := os.MkdirAll(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"logical bytes over cap", func(t *testing.T, snapshot diagnosticOuterSnapshot) {
			for _, root := range []string{snapshot.home, snapshot.xdg} {
				file, err := os.OpenFile(filepath.Join(root, "sparse"), os.O_CREATE|os.O_WRONLY, 0o600)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Truncate(32 << 20); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			file, err := os.OpenFile(filepath.Join(snapshot.home, "extra"), os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, snapshot := diagnosticSnapshotForTest(t)
			test.mutate(t, snapshot)
			assertDiagnosticWalkerVerdicts(t, snapshot, false)
		})
	}
}

func TestCacheDiagnosticReferenceDigestPreservesRawNames(t *testing.T) {
	_, snapshot := diagnosticSnapshotForTest(t)
	firstName := string([]byte{'r', 'a', 'w', '-', 0xff})
	secondName := string([]byte{'r', 'a', 'w', '-', 0xfe})
	firstPath := filepath.Join(snapshot.home, firstName)
	if err := os.WriteFile(firstPath, []byte("x"), 0o600); err != nil {
		if strings.Contains(err.Error(), "illegal byte sequence") {
			t.Skip("filesystem rejects invalid UTF-8 names")
		}
		t.Fatal(err)
	}
	first, err := referenceAuditWorkerCacheRoots(snapshot)
	if err != nil || !first.accepted {
		t.Fatalf("first reference: %+v, %v", first, err)
	}
	if err := os.Rename(firstPath, filepath.Join(snapshot.home, secondName)); err != nil {
		t.Fatal(err)
	}
	second, err := referenceAuditWorkerCacheRoots(snapshot)
	if err != nil || !second.accepted {
		t.Fatalf("second reference: %+v, %v", second, err)
	}
	if first.digest == second.digest {
		t.Fatal("reference digest did not bind raw filename bytes")
	}
}

func TestCacheDiagnosticReferenceHardCeilingsAreMechanicalFailures(t *testing.T) {
	t.Run("depth", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		path := snapshot.home
		for index := 0; index < 65; index++ {
			path = filepath.Join(path, "d")
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := referenceAuditWorkerCacheRoots(snapshot); err == nil {
			t.Fatal("reference depth ceiling returned a partial verdict")
		}
	})
	t.Run("logical bytes", func(t *testing.T) {
		_, snapshot := diagnosticSnapshotForTest(t)
		file, err := os.OpenFile(filepath.Join(snapshot.home, "sparse"), os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate((128 << 20) + 1); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := referenceAuditWorkerCacheRoots(snapshot); err == nil {
			t.Fatal("reference logical ceiling returned a partial verdict")
		}
	})
}

func TestCacheDiagnosticEnvironmentAndOuterEnvelope(t *testing.T) {
	environmentA, snapshotA := v5EnvironmentForTest(t)
	digestA, err := cacheDiagnosticEnvironmentDigest(environmentA, snapshotA.base)
	if err != nil {
		t.Fatal(err)
	}
	environmentB, snapshotB := v5EnvironmentForTest(t)
	digestB, err := cacheDiagnosticEnvironmentDigest(environmentB, snapshotB.base)
	if err != nil {
		t.Fatal(err)
	}
	if digestA != digestB {
		t.Fatalf("normalized environment depends on private root: %s != %s", digestA, digestB)
	}
	outer, err := captureDiagnosticOuterEnvelope(environmentA, snapshotA.base)
	if err != nil || verifyDiagnosticOuterEnvelope(environmentA, outer) != nil {
		t.Fatalf("capture/verify outer envelope: %v", err)
	}
	hostile := append([]string(nil), environmentA...)
	hostile[0] = "HOME=/tmp/not-the-private-home"
	if _, err := cacheDiagnosticEnvironmentDigest(hostile, snapshotA.base); err == nil {
		t.Fatal("hostile environment was accepted")
	}
	oldHome := snapshotA.home + ".old"
	if err := os.Rename(snapshotA.home, oldHome); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(snapshotA.home)
		_ = os.Rename(oldHome, snapshotA.home)
	}()
	if err := os.Mkdir(snapshotA.home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyDiagnosticOuterEnvelope(environmentA, outer); err == nil || strings.Contains(err.Error(), snapshotA.home) {
		t.Fatalf("root replacement was accepted or leaked a path: %v", err)
	}
}

func TestCacheDiagnosticRequiresExactPhysicalRepositoryRoot(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyCacheDiagnosticInvocationRoot(root, root); err != nil {
		t.Fatal(err)
	}
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := cacheDiagnosticRecordPath(state.CommonDir)
	receiptBefore, readBeforeErr := os.ReadFile(receiptPath)
	for _, invocation := range []string{".", filepath.Join(root, "internal")} {
		if err := verifyCacheDiagnosticInvocationRoot(invocation, root); err == nil {
			t.Fatalf("non-root invocation %q was accepted", invocation)
		}
		if _, err := ExecuteCacheDiagnostic(context.Background(), invocation); err == nil || !strings.Contains(err.Error(), "exact repository root") {
			t.Fatalf("actual non-root invocation %q was not rejected at the root boundary: %v", invocation, err)
		}
	}
	link := filepath.Join(t.TempDir(), "repository-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if err := verifyCacheDiagnosticInvocationRoot(link, root); err == nil {
		t.Fatal("symlinked repository-root invocation was accepted")
	}
	if _, err := ExecuteCacheDiagnostic(context.Background(), link); err == nil || !strings.Contains(err.Error(), "exact repository root") {
		t.Fatalf("actual symlink-root invocation was not rejected at the root boundary: %v", err)
	}
	receiptAfter, readAfterErr := os.ReadFile(receiptPath)
	sameReadState := readBeforeErr == nil && readAfterErr == nil || os.IsNotExist(readBeforeErr) && os.IsNotExist(readAfterErr)
	if !sameReadState || !bytes.Equal(receiptBefore, receiptAfter) {
		t.Fatal("zero-start root failures changed the diagnostic receipt slot")
	}
}

func copyDiagnosticExecutableForTest(t *testing.T, source string) string {
	t.Helper()
	encoded, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "diagnostic-executable")
	if err := os.WriteFile(target, encoded, 0o700); err != nil {
		t.Fatal(err)
	}
	return target
}

func TestCacheDiagnosticRevalidatesExecutablesAtStartBoundary(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *diagnosticExecutionHarness)
	}{
		{"operator", func(t *testing.T, harness *diagnosticExecutionHarness) {
			harness.operatorPath = copyDiagnosticExecutableForTest(t, harness.operatorPath)
			digest, err := regularExecutableDigest(harness.operatorPath)
			if err != nil {
				t.Fatal(err)
			}
			harness.record.OperatorSHA256 = digest
		}},
		{"worker", func(t *testing.T, harness *diagnosticExecutionHarness) {
			harness.worker.Path = copyDiagnosticExecutableForTest(t, harness.worker.Path)
			digest, err := regularExecutableDigest(harness.worker.Path)
			if err != nil {
				t.Fatal(err)
			}
			harness.record.WorkerSHA256 = digest
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newDiagnosticExecutionHarness(t)
			test.mutate(t, harness)
			if err := persistCacheDiagnosticRecord(harness.state.CommonDir, harness.record); err != nil {
				t.Fatal(err)
			}
			target := harness.operatorPath
			if test.name == "worker" {
				target = harness.worker.Path
			}
			hooks := cacheDiagnosticExecutionHooks{verifyExecutionState: func(stage string) error {
				if stage == "before" {
					if err := os.WriteFile(target, []byte("replaced"), 0o700); err != nil {
						t.Fatal(err)
					}
				}
				return nil
			}}
			result, err := executeCacheDiagnosticWorker(context.Background(), harness.state, harness.outer, harness.buildWorktree, harness.executionWorktree, harness.output, harness.operatorPath, harness.worker, []byte("synthetic fixed input"), &harness.record, hooks)
			if err == nil || result != cacheDiagnosticFailure || harness.record.WorkerStarts != 0 {
				t.Fatalf("executable replacement result=%q starts=%d err=%v", result, harness.record.WorkerStarts, err)
			}
			if err := persistCacheDiagnosticTerminal(harness.state.CommonDir, &harness.record, "failed", cacheDiagnosticFailure); err != nil {
				t.Fatal(err)
			}
			failed := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
			if failed.State != "failed" || failed.WorkerStarts != 0 {
				t.Fatalf("zero-start replacement receipt mismatch: %+v", failed)
			}
		})
	}
}

func TestCacheDiagnosticReceiptLifecycleAndExclusivity(t *testing.T) {
	commonDirectory := t.TempDir()
	record := newCacheDiagnosticRecord(strings.Repeat("c", 40), strings.Repeat("o", 64), strings.Repeat("w", 64), strings.Repeat("e", 64), strings.Repeat("b", 64))
	record.CreatedUTC = time.Unix(0, 0).UTC().Format(time.RFC3339)
	if err := createCacheDiagnosticRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	if err := createCacheDiagnosticRecord(commonDirectory, record); err == nil {
		t.Fatal("exclusive diagnostic receipt allowed a retry")
	}
	assertCacheDiagnosticRecord := func(want cacheDiagnosticRecord) {
		t.Helper()
		encoded, err := os.ReadFile(cacheDiagnosticRecordPath(commonDirectory))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := causalv2.StrictDecode[cacheDiagnosticRecord](encoded)
		if err != nil || decoded != want || !bytes.Equal(encoded, mustCanonical(decoded)) {
			t.Fatalf("receipt mismatch: %+v, %v", decoded, err)
		}
		for _, forbidden := range [][]byte{[]byte(`"training_digest"`), []byte(`"bundle_digest"`)} {
			if bytes.Contains(encoded, forbidden) {
				t.Fatalf("diagnostic receipt leaked empirical identity field %s", forbidden)
			}
		}
	}
	assertCacheDiagnosticRecord(record)
	record.WorkerStarts = 1
	if err := persistCacheDiagnosticRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	assertCacheDiagnosticRecord(record)
	record.State = "completed"
	record.Result = cacheDiagnosticNonReproduction
	if err := persistCacheDiagnosticRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	assertCacheDiagnosticRecord(record)
	if _, err := os.Lstat(cacheDiagnosticRecordPath(commonDirectory) + ".next"); !os.IsNotExist(err) {
		t.Fatalf("terminal receipt left a temporary path: %v", err)
	}
	for _, panel := range []Panel{PanelValidation, PanelLocked} {
		for _, path := range []string{attemptRecordPath(commonDirectory, panel), attemptProofRecordPath(commonDirectory, panel), resultPath(commonDirectory, panel)} {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("diagnostic created an empirical attempt artifact at %s", path)
			}
		}
	}
}

func TestCacheDiagnosticActualWorkerLifecycleAndBracket(t *testing.T) {
	harness := newDiagnosticExecutionHarness(t)
	stages := []string{}
	steps := []int{}
	hooks := cacheDiagnosticExecutionHooks{
		verifyExecutionState: func(stage string) error {
			stages = append(stages, stage)
			return nil
		},
		bracketStep: func(step int) { steps = append(steps, step) },
	}
	result, err := executeCacheDiagnosticWorker(context.Background(), harness.state, harness.outer, harness.buildWorktree, harness.executionWorktree, harness.output, harness.operatorPath, harness.worker, []byte("synthetic fixed input"), &harness.record, hooks)
	if err != nil || result != cacheDiagnosticNonReproduction {
		t.Fatalf("synthetic diagnostic result=%q err=%v", result, err)
	}
	if fmt.Sprint(stages) != "[before after]" || fmt.Sprint(steps) != "[1 2 3 4 5 6]" {
		t.Fatalf("execution chronology stages=%v steps=%v", stages, steps)
	}
	started := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
	if started.State != "started" || started.WorkerStarts != 1 || started.Result != "" {
		t.Fatalf("worker start was not durably recorded before input: %+v", started)
	}
	if err := persistCacheDiagnosticTerminal(harness.state.CommonDir, &harness.record, "completed", result); err != nil {
		t.Fatal(err)
	}
	completed := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
	if completed.State != "completed" || completed.WorkerStarts != 1 || completed.Result != cacheDiagnosticNonReproduction {
		t.Fatalf("terminal record mismatch: %+v", completed)
	}
	entries, err := os.ReadDir(harness.output)
	if err != nil || len(entries) != 0 {
		t.Fatalf("synthetic worker wrote diagnostic output: %v, %v", entries, err)
	}
	if err := requireDiagnosticSlotsAbsent(harness.state.CommonDir); err == nil {
		t.Fatal("slot check ignored the consumed diagnostic receipt")
	}
	for _, panel := range []Panel{PanelValidation, PanelLocked} {
		for _, path := range []string{attemptRecordPath(harness.state.CommonDir, panel), attemptProofRecordPath(harness.state.CommonDir, panel), resultPath(harness.state.CommonDir, panel)} {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("synthetic diagnostic created an empirical record at %s", path)
			}
		}
	}
}

func TestCacheDiagnosticStartedReceiptSurvivesInjectedKill(t *testing.T) {
	harness := newDiagnosticExecutionHarness(t)
	hookObserved := false
	hooks := cacheDiagnosticExecutionHooks{
		verifyExecutionState: func(string) error { return nil },
		afterStartedPersist: func() error {
			hookObserved = true
			record := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
			if record.State != "started" || record.WorkerStarts != 1 {
				t.Fatalf("kill hook observed non-durable start: %+v", record)
			}
			return errors.New("injected process death")
		},
	}
	result, err := executeCacheDiagnosticWorker(context.Background(), harness.state, harness.outer, harness.buildWorktree, harness.executionWorktree, harness.output, harness.operatorPath, harness.worker, []byte("synthetic fixed input"), &harness.record, hooks)
	if err == nil || result != cacheDiagnosticFailure || !hookObserved {
		t.Fatalf("injected kill result=%q observed=%v err=%v", result, hookObserved, err)
	}
	residue := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
	if residue.State != "started" || residue.WorkerStarts != 1 {
		t.Fatalf("kill did not leave a consumed started residue: %+v", residue)
	}
	if err := createCacheDiagnosticRecord(harness.state.CommonDir, harness.record); err == nil {
		t.Fatal("started residue allowed a retry")
	}
}

func TestCacheDiagnosticActualOperatorDeathLeavesConsumedStartedReceipt(t *testing.T) {
	base := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestCacheDiagnosticKilledOperatorProcess$", "--", "cache-diagnostic-kill-harness", base)
	output, err := command.CombinedOutput()
	exitError, ok := err.(*exec.ExitError)
	if !ok || exitError.ExitCode() != 86 || len(output) != 0 {
		t.Fatalf("kill harness exit=%v output=%q", err, output)
	}
	commonDirectory := filepath.Join(base, "common")
	residue := readCacheDiagnosticRecordForTest(t, commonDirectory)
	if residue.State != "started" || residue.WorkerStarts != 1 || residue.Result != "" {
		t.Fatalf("operator death did not leave consumed count-one residue: %+v", residue)
	}
	if err := createCacheDiagnosticRecord(commonDirectory, residue); err == nil {
		t.Fatal("operator-death residue allowed a retry")
	}
}

func TestCacheDiagnosticActualBracketInstabilityBecomesFailure(t *testing.T) {
	harness := newDiagnosticExecutionHarness(t)
	forbidden := filepath.Join(harness.outer.home, "unstable-link")
	if err := os.Symlink("target", forbidden); err != nil {
		t.Fatal(err)
	}
	hooks := cacheDiagnosticExecutionHooks{
		verifyExecutionState: func(string) error { return nil },
		bracketStep: func(step int) {
			if step == 2 {
				if err := os.Remove(forbidden); err != nil {
					t.Fatal(err)
				}
			}
		},
	}
	result, err := executeCacheDiagnosticWorker(context.Background(), harness.state, harness.outer, harness.buildWorktree, harness.executionWorktree, harness.output, harness.operatorPath, harness.worker, []byte("synthetic fixed input"), &harness.record, hooks)
	if err == nil || result != cacheDiagnosticFailure || !strings.Contains(err.Error(), "not stable") {
		t.Fatalf("unstable bracket result=%q err=%v", result, err)
	}
	if err := persistCacheDiagnosticTerminal(harness.state.CommonDir, &harness.record, "failed", cacheDiagnosticFailure); err != nil {
		t.Fatal(err)
	}
	failed := readCacheDiagnosticRecordForTest(t, harness.state.CommonDir)
	if failed.State != "failed" || failed.Result != cacheDiagnosticFailure || failed.WorkerStarts != 1 {
		t.Fatalf("unstable bracket terminal mismatch: %+v", failed)
	}
}

func TestCacheDiagnosticCleanupIsConfinedBeforeCompletion(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	base := filepath.Join(parent, "owned-diagnostic-root")
	sibling := filepath.Join(parent, "outside-sentinel")
	if err := os.MkdirAll(filepath.Join(base, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "nested", "entry"), []byte("x"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(base, "nested"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupReplayWorktreeSet(root, nil, base); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(base); !os.IsNotExist(err) {
		t.Fatalf("owned diagnostic root survived cleanup: %v", err)
	}
	if encoded, err := os.ReadFile(sibling); err != nil || string(encoded) != "preserve" {
		t.Fatalf("cleanup escaped its owned root: %q, %v", encoded, err)
	}
}

func TestCacheDiagnosticFailedV5PredecessorBinding(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(filepath.Join(state.CommonDir, "nous-attempts", "active-causal-diagnosis-v5-replay.json"))
	if err != nil {
		t.Fatal(err)
	}
	commonDirectory := t.TempDir()
	directory := filepath.Join(commonDirectory, "nous-attempts")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "active-causal-diagnosis-v5-replay.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyFailedV5DiagnosticPredecessor(commonDirectory); err != nil {
		t.Fatal(err)
	}
	encoded[len(encoded)-2] ^= 1
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyFailedV5DiagnosticPredecessor(commonDirectory); err == nil {
		t.Fatal("altered v5 predecessor was accepted")
	}
}

func parseDiagnosticSource(t *testing.T) (*token.FileSet, *ast.File, []byte) {
	t.Helper()
	path := filepath.Join("cache_diagnostic.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	return fileSet, file, source
}

func diagnosticFunction(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("function %s is absent", name)
	return nil
}

func TestCacheDiagnosticCandidateBodyTransplantsWithoutAdapters(t *testing.T) {
	fileSet, file, _ := parseDiagnosticSource(t)
	function := diagnosticFunction(t, file, "candidateAuditWorkerCacheRoot")
	var body bytes.Buffer
	if err := format.Node(&body, fileSet, function.Body); err != nil {
		t.Fatal(err)
	}
	scaffold := fmt.Sprintf(`package transplant
import (
  "errors"
  "path/filepath"
  "strings"
  "golang.org/x/sys/unix"
)
type workerCacheBudget struct { entries int; pathBytes int; logicalBytes int64 }
func auditWorkerCacheRoot(root string, rootDevice, rootInode uint64, budget *workerCacheBudget) error %s
`, body.String())
	generatedSet := token.NewFileSet()
	generated, err := parser.ParseFile(generatedSet, "transplant.go", scaffold, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	lookup := func(importPath string) (io.ReadCloser, error) {
		command := diagnosticPinnedGoCommand(t, "list", "-export", "-f", "{{.Export}}", importPath)
		output, err := command.Output()
		if err != nil {
			return nil, err
		}
		return os.Open(strings.TrimSpace(string(output)))
	}
	config := types.Config{Importer: importer.ForCompiler(generatedSet, "gc", lookup)}
	if _, err := config.Check("transplant", generatedSet, []*ast.File{generated}, nil); err != nil {
		t.Fatalf("candidate body is not a direct type-correct transplant: %v\n%s", err, scaffold)
	}
	generatedBody := diagnosticFunction(t, generated, "auditWorkerCacheRoot").Body
	var normalized bytes.Buffer
	if err := format.Node(&normalized, token.NewFileSet(), generatedBody); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body.Bytes(), normalized.Bytes()) {
		t.Fatal("transplant changed the normalized candidate body")
	}
}

func TestCacheDiagnosticWalkerSourceIndependenceAndImmutability(t *testing.T) {
	_, file, source := parseDiagnosticSource(t)
	for _, name := range []string{"candidateAuditWorkerCacheRoot", "referenceAuditWorkerCacheRoot"} {
		function := diagnosticFunction(t, file, name)
		allowed := map[string]bool{
			"errors.New": true, "filepath.Join": true, "filepath.ToSlash": true,
			"strings.ContainsRune": true, "unix.Open": true, "unix.Openat": true,
			"unix.Close": true, "unix.Fstat": true, "unix.Fstatat": true,
			"unix.Lstat": true, "unix.ReadDirent": true, "unix.ParseDirent": true,
			"os.NewFile": true, "errors.Is": true, "file.Close": true,
			"file.Readdirnames": true, "walk": true, "make": true, "append": true,
			"len": true, "uint32": true, "uint64": true, "uintptr": true,
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			called := ""
			switch expression := call.Fun.(type) {
			case *ast.Ident:
				called = expression.Name
			case *ast.SelectorExpr:
				if qualifier, ok := expression.X.(*ast.Ident); ok {
					called = qualifier.Name + "." + expression.Sel.Name
				}
			}
			if !allowed[called] {
				t.Errorf("%s calls non-allowlisted operation %q", name, called)
			}
			if called == "unix.Open" || called == "unix.Openat" {
				flagIndex := 1
				if called == "unix.Openat" {
					flagIndex = 2
				}
				var flags bytes.Buffer
				if len(call.Args) <= flagIndex || format.Node(&flags, token.NewFileSet(), call.Args[flagIndex]) != nil {
					t.Errorf("%s has an unreadable open flag expression", name)
				} else {
					flagText := flags.String()
					if !strings.Contains(flagText, "unix.O_RDONLY") {
						t.Errorf("%s open is not explicitly read-only: %s", name, flagText)
					}
					for _, writeFlag := range []string{"O_WRONLY", "O_RDWR", "O_CREAT", "O_TRUNC", "O_APPEND"} {
						if strings.Contains(flagText, writeFlag) {
							t.Errorf("%s open contains write flag %s", name, writeFlag)
						}
					}
				}
			}
			return true
		})
	}
	for _, forbidden := range []string{"ExecuteReplay(", "publishTrainingEvidence(", "NewProtected", "compareReplay", "result comparison", "StrictDecode[replaySuccessRecord]"} {
		if bytes.Contains(source, []byte(forbidden)) {
			t.Errorf("diagnostic source contains forbidden protected operation %q", forbidden)
		}
	}
	for _, leaked := range []string{"96b1cdf7579c0a186e5cd9aeb7aaa42f0c224ffe19989bf78b5b3aa320b17fa0", "117a0322464cdf26022b7c21b2d5401c67cbad974640f042e2591c920d982503", `json:"training_digest"`, `json:"bundle_digest"`} {
		if bytes.Contains(source, []byte(leaked)) {
			t.Errorf("diagnostic source republishes a protected empirical identity %q", leaked)
		}
	}

	function := diagnosticFunction(t, file, "cacheDiagnosticCandidateBodyDigest")
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if ok && identifier.Name == "gitFile" {
			if len(call.Args) != 4 {
				t.Errorf("gitFile called with %d arguments", len(call.Args))
			} else if path, ok := call.Args[3].(*ast.Ident); !ok || path.Name != "cacheDiagnosticSourcePath" {
				t.Error("diagnostic reads a Git path other than its own source")
			}
		}
		return true
	})
	execute := diagnosticFunction(t, file, "ExecuteCacheDiagnostic")
	slotChecks := 0
	ast.Inspect(execute.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "requireDiagnosticSlotsAbsent" {
			slotChecks++
		}
		return true
	})
	if slotChecks != 2 {
		t.Fatalf("diagnostic must check protected slots before preflight and immediately before receipt creation; got %d checks", slotChecks)
	}
}

func TestCacheDiagnosticReplayControlFlowReachesDigestRejectionAfterRegeneration(t *testing.T) {
	path := filepath.Join("replay_hook.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	positions := make([]int, 0, 4)
	for _, marker := range [][]byte{
		[]byte("regenerateTrainingEvidence(ctx, state.Root, capability)"),
		[]byte("VerifyTrainingReportBytes(evidence.Report)"),
		[]byte("VerifyTrainingBundleBytes(evidence.Bundle)"),
		[]byte("writeReplayOutputAt(outputDirectory, TrainingReportName"),
	} {
		position := bytes.Index(source, marker)
		if position < 0 {
			t.Fatalf("replay control marker %q is absent", marker)
		}
		positions = append(positions, position)
	}
	for index := 1; index < len(positions); index++ {
		if positions[index-1] >= positions[index] {
			t.Fatalf("replay control order changed: %v", positions)
		}
	}
	latePath := source[positions[2]:positions[3]]
	for _, forbidden := range [][]byte{[]byte("exec.Command"), []byte("cue"), []byte("HOME"), []byte("XDG_CONFIG_HOME"), []byte("gitStringOutput")} {
		if bytes.Contains(latePath, forbidden) {
			t.Errorf("late replay verification path contains %q", forbidden)
		}
	}
}

func TestCacheDiagnosticCommittedTopologyAndCandidateDigest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := gitStringOutput(context.Background(), root, "rev-parse", state.Head+"^")
	if err != nil || parent != cacheDiagnosticPlanCommit || !state.Clean {
		t.Skip("topology is enforceable only on clean committed X6")
	}
	if err := verifyCacheDiagnosticTopology(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	operator := filepath.Join(t.TempDir(), "cache-diagnostic")
	build := diagnosticPinnedGoCommand(t, "build", "-o", operator, "./internal/causalexpv2/cachediagexec")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build committed diagnostic operator: %v: %s", err, output)
	}
	info, err := buildinfo.ReadFile(operator)
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["vcs.revision"] != state.Head || settings["vcs.modified"] != "false" {
		t.Fatalf("separately built operator is not self-bound: %+v", settings)
	}
	digest, err := cacheDiagnosticCandidateBodyDigest(context.Background(), state)
	if err != nil || len(digest) != 64 {
		t.Fatalf("candidate digest: %q, %v", digest, err)
	}
}

func TestCacheDiagnosticCommandFailureOutputIsGeneric(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "cache-diagnostic")
	build := diagnosticPinnedGoCommand(t, "build", "-o", binary, "./internal/causalexpv2/cachediagexec")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build diagnostic command: %v: %s", err, output)
	}
	command := exec.Command(binary, "unexpected-argument")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("diagnostic command accepted an argument")
	}
	if string(output) != "diagnostic: diagnostic-failure\n" {
		t.Fatalf("diagnostic command exposed non-generic failure: %q", output)
	}
}
