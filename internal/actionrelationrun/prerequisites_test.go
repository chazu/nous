package actionrelationrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chazu/nous/internal/actionrelationcompetence"
)

func TestPersistCompetenceEvidenceIsExclusiveAndExact(t *testing.T) {
	root := t.TempDir()
	cases := []actionrelationcompetence.CaseRow{{Suite: "suite", CaseID: "case", Input: digest([]byte("input")), Expected: digest([]byte("result"))}}
	results := []actionrelationcompetence.ResultRow{{Suite: "suite", CaseID: "case", Production: digest([]byte("result")), Oracle: digest([]byte("result"))}}
	evidence, err := actionrelationcompetence.BuildEvidence(cases, results)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".nous"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := persistCompetenceEvidence(root, evidence); err != nil {
		t.Fatal(err)
	}
	for _, file := range append(append([]actionrelationcompetence.EvidenceFile{}, evidence.CaseFiles...), evidence.ResultFiles...) {
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if readErr != nil || string(data) != string(file.Data) {
			t.Fatalf("readback %s: %v", file.Path, readErr)
		}
	}
	if err := persistCompetenceEvidence(root, evidence); err == nil {
		t.Fatal("accepted reused competence namespace")
	}
}

func TestExclusiveAuthorityAllowsOnlyByteIdenticalReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority.json")
	if err := writeExclusiveAuthority(path, []byte("exact")); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveAuthority(path, []byte("exact")); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveAuthority(path, []byte("changed")); err == nil {
		t.Fatal("accepted changed authority")
	}
}
