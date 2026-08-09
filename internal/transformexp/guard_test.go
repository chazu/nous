package transformexp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewManifestCanonicalRoundTripAndScopeOrder(t *testing.T) {
	commit := "0123456789abcdef0123456789abcdef01234567"
	manifest := ImplementationReviewManifest{
		Version: "transform-implementation-reviews/v1", PlanCommit: PlanCommit, ImplementationCommit: commit,
		Reviews:        []ImplementationReview{{"architecture", "accepted", commit}, {"semantics", "accepted", commit}, {"experiment", "accepted", commit}},
		ProtectedPaths: map[string]string{"go.mod": strings.Repeat("a", 64)},
	}
	encoded, err := encodeReviewManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeReviewManifest(encoded)
	if err != nil || decoded.ImplementationCommit != commit {
		t.Fatalf("round trip failed: %+v %v", decoded, err)
	}
	wrong := strings.Replace(string(encoded), `"architecture"`, `"experiment"`, 1)
	if _, err := decodeReviewManifest([]byte(wrong)); err == nil {
		t.Fatal("wrong review scope order was accepted")
	}
}

func TestAttemptReceiptIsExclusiveAndMonotone(t *testing.T) {
	root := t.TempDir()
	authority := repositoryAuthority{Root: root, Head: strings.Repeat("a", 40), Reviews: ImplementationReviewManifest{ImplementationCommit: strings.Repeat("b", 40)}}
	receipt, err := claimAttempt(authority, "validation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claimAttempt(authority, "validation"); err == nil {
		t.Fatal("duplicate receipt claim succeeded")
	}
	if err := startAttempt(root, receipt, "", ""); err != nil {
		t.Fatal(err)
	}
	fixture, report, graph := strings.Repeat("c", 64), strings.Repeat("d", 64), strings.Repeat("e", 64)
	if err := finalizeAttempt(root, receipt, "published", fixture, report, graph); err != nil {
		t.Fatal(err)
	}
	if err := finalizeAttempt(root, receipt, "invalid", "", "", ""); err == nil {
		t.Fatal("published receipt transitioned")
	}
	data, err := os.ReadFile(receiptPath(root, "validation"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"published"`) || !strings.Contains(string(data), fixture) {
		t.Fatalf("final receipt lost committed fields: %s", data)
	}
}

func TestFailedReceiptRewritePreservesClaimForInvalidation(t *testing.T) {
	root := t.TempDir()
	authority := repositoryAuthority{Root: root, Head: strings.Repeat("a", 40), Reviews: ImplementationReviewManifest{ImplementationCommit: strings.Repeat("b", 40)}}
	receipt, err := claimAttempt(authority, "locked")
	if err != nil {
		t.Fatal(err)
	}
	temporary := receiptPath(root, "locked") + ".next"
	if err := os.WriteFile(temporary, []byte("obstruction"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := startAttempt(root, receipt, strings.Repeat("c", 64), ""); err == nil {
		t.Fatal("obstructed receipt rewrite succeeded")
	}
	if receipt.State != "claimed" || receipt.RootCommitment != "" {
		t.Fatalf("failed rewrite mutated in-memory authority: %+v", receipt)
	}
	if err := os.Remove(temporary); err != nil {
		t.Fatal(err)
	}
	if err := finalizeAttempt(root, receipt, "invalid", "", "", ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(receiptPath(root, "locked"))
	if err != nil || !strings.Contains(string(data), `"invalid"`) {
		t.Fatalf("invalid receipt was not durable: %s err=%v", data, err)
	}
}

func TestReviewedFilesystemRejectsIgnoredSourceAndSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	protected := map[string]string{"go.mod": strings.Repeat("a", 64)}
	if err := verifyReviewedFilesystem(root, protected); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.go"), []byte("package ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyReviewedFilesystem(root, protected); err == nil {
		t.Fatal("unreviewed compiler input was accepted")
	}
	if err := os.Remove(filepath.Join(root, "ignored.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("go.mod", filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := verifyReviewedFilesystem(root, protected); err == nil {
		t.Fatal("repository symlink was accepted")
	}
}

func TestProtectedPanelConstructorsHaveExactlyOneProductionCaller(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	expectedCallers := map[string]string{"developmentPanel": "ExecuteDevelopment", "validationPanel": "ExecuteValidation", "lockedPanel": "ExecuteLocked"}
	counts := map[string]int{}
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "//go:linkname") {
			t.Fatalf("linkname can bypass constructor authority in %s", entry.Name())
		}
		parsed, err := parser.ParseFile(fset, entry.Name(), data, 0)
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = parsed
	}
	for filename, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				expectedCaller, protected := expectedCallers[identifier.Name]
				if !protected {
					return true
				}
				parentIsCall := false
				ast.Inspect(function.Body, func(parent ast.Node) bool {
					if call, ok := parent.(*ast.CallExpr); ok && call.Fun == identifier {
						parentIsCall = true
						return false
					}
					return !parentIsCall
				})
				if !parentIsCall || function.Name.Name != expectedCaller || filename != "guard.go" {
					t.Errorf("%s is referenced outside its sole guarded call at %s", identifier.Name, fset.Position(identifier.Pos()))
				} else {
					counts[identifier.Name]++
				}
				return true
			})
		}
	}
	for surface := range expectedCallers {
		if counts[surface] != 1 {
			t.Fatalf("%s direct guarded calls = %d, want one", surface, counts[surface])
		}
	}
}

func TestRepositoryAuthorityRejectsGitAlternatesAndLocalIncludes(t *testing.T) {
	setup := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "domains"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command("git", "init", "-q", root)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, output)
		}
		return root
	}
	t.Run("alternates", func(t *testing.T) {
		root := setup(t)
		path := filepath.Join(root, ".git", "objects", "info", "alternates")
		if err := os.WriteFile(path, []byte("/tmp/untrusted\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := authorizeRepository(root, filepath.Join(root, "domains")); err == nil || !strings.Contains(err.Error(), "alternates") {
			t.Fatalf("alternates authority error = %v", err)
		}
	})
	t.Run("include", func(t *testing.T) {
		root := setup(t)
		command := exec.Command("git", "-C", root, "config", "--local", "include.path", "/tmp/untrusted")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git config: %v %s", err, output)
		}
		if _, err := authorizeRepository(root, filepath.Join(root, "domains")); err == nil || !strings.Contains(err.Error(), "local Git authority") {
			t.Fatalf("local include authority error = %v", err)
		}
	})
}
