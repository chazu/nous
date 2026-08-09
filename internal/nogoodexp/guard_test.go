package nogoodexp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedAttemptClaimIsDurableAndOneShot(t *testing.T) {
	root := t.TempDir()
	authority := repositoryAuthority{root: root, head: "head", reviews: ImplementationReviewManifest{ImplementationCommit: "implementation"}}
	receipt, err := claimAttempt(authority, "validation", "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != "claimed" {
		t.Fatalf("claim state = %s", receipt.State)
	}
	if _, err := claimAttempt(authority, "validation", ""); err == nil {
		t.Fatal("second protected claim succeeded")
	}
	if err := startAttempt(root, receipt, ""); err != nil {
		t.Fatal(err)
	}
	if err := finalizeAttempt(root, receipt, "invalid", nil); err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(receiptPath(root, "validation"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted AttemptReceipt
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.State != "invalid" {
		t.Fatalf("final receipt state = %s", persisted.State)
	}
	info, err := os.Lstat(transcriptPath(root, "validation"))
	if err != nil || !info.IsDir() {
		t.Fatalf("claimed transcript root was not retained: %v", err)
	}
}

func TestClaimRejectsExistingOrSymlinkEvidencePaths(t *testing.T) {
	for _, target := range []string{"receipt", "report", "transcript"} {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".nous"), 0o755); err != nil {
				t.Fatal(err)
			}
			path := map[string]string{"receipt": receiptPath(root, "locked"), "report": reportPath(root, "locked"), "transcript": transcriptPath(root, "locked")}[target]
			if err := os.Symlink("missing", path); err != nil {
				t.Fatal(err)
			}
			authority := repositoryAuthority{root: root, head: "head", reviews: ImplementationReviewManifest{ImplementationCommit: "implementation"}}
			if _, err := claimAttempt(authority, "locked", ""); err == nil {
				t.Fatal("claim accepted a pre-existing symlink")
			}
		})
	}
}
