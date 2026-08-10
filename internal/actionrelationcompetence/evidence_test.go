package actionrelationcompetence

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
)

func TestCompetenceEvidencePacksRetainExactCaseResultPreimages(t *testing.T) {
	cases := []CaseRow{
		{Suite: "guards", CaseID: "0001", Input: shaHex([]byte("input-1")), Expected: shaHex([]byte("result-1"))},
		{Suite: "sequences", CaseID: "0001", Input: shaHex([]byte("input-2")), Expected: shaHex([]byte("result-2"))},
	}
	results := []ResultRow{
		{Suite: "guards", CaseID: "0001", Production: cases[0].Expected, Oracle: cases[0].Expected},
		{Suite: "sequences", CaseID: "0001", Production: cases[1].Expected, Oracle: cases[1].Expected},
	}
	evidence, err := BuildEvidence(cases, results)
	if err != nil || VerifyEvidence(evidence) != nil {
		t.Fatal(err)
	}
	build, _ := actionrelationexp.Reference("docs/actionrelations-build-authority.json", []byte("build"))
	caseRef, _ := actionrelationexp.Reference(competenceRootPath+"/cases-root.json", evidence.CaseManifest.Canonical)
	resultRef, _ := actionrelationexp.Reference(competenceRootPath+"/results-root.json", evidence.ResultManifest.Canonical)
	root, err := BuildRoot(Root{SourceRoot: shaHex([]byte("source")), BinaryDigest: shaHex([]byte("binary")), BuildAuthority: build, CommandArgv: []string{".nous/bin/actionrelation-nous-v1", "-stage", "competence"}, Environment: []actionrelationexp.EnvironmentRow{{Key: "GOMAXPROCS", Value: "1"}}, Evidence: evidence, CaseManifestRef: caseRef, ResultManifestRef: resultRef})
	if err != nil || root.Digest == "" || VerifyRoot(root) != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range append(append([]EvidenceFile{}, evidence.CaseFiles...), evidence.ResultFiles...) {
		files[file.Path] = file.Data
	}
	parsedEvidence, err := ParseEvidence(evidence.CaseManifest.Canonical, evidence.ResultManifest.Canonical, files)
	if err != nil {
		t.Fatal(err)
	}
	parsedRoot, err := ParseRoot(root.Canonical, parsedEvidence)
	if err != nil || parsedRoot.Digest != root.Digest {
		t.Fatalf("parse root: %v", err)
	}
	if _, err := ParseRoot(append(append([]byte{}, root.Canonical...), '\n'), parsedEvidence); err == nil {
		t.Fatal("accepted trailing competence root bytes")
	}
	corruptFiles := map[string][]byte{}
	for path, data := range files {
		corruptFiles[path] = append([]byte{}, data...)
	}
	corruptFiles[evidence.CaseFiles[0].Path] = append(corruptFiles[evidence.CaseFiles[0].Path], 0)
	if _, err := ParseEvidence(evidence.CaseManifest.Canonical, evidence.ResultManifest.Canonical, corruptFiles); err == nil {
		t.Fatal("accepted trailing competence pack bytes")
	}
	corrupt := evidence
	corrupt.ResultFiles[0].Data[10] ^= 1
	if VerifyEvidence(corrupt) == nil {
		t.Fatal("accepted corrupted competence result pack")
	}
}

func TestCompetenceEvidenceRejectsMismatchedAndUnorderedRows(t *testing.T) {
	caseRow := CaseRow{Suite: "suite", CaseID: "case", Input: shaHex([]byte("input")), Expected: shaHex([]byte("expected"))}
	if _, err := BuildEvidence([]CaseRow{caseRow}, []ResultRow{{Suite: "suite", CaseID: "case", Production: caseRow.Expected, Oracle: shaHex([]byte("wrong"))}}); err == nil {
		t.Fatal("accepted production/oracle disagreement")
	}
	if _, err := BuildEvidence([]CaseRow{caseRow, caseRow}, []ResultRow{{Suite: "suite", CaseID: "case", Production: caseRow.Expected, Oracle: caseRow.Expected}, {Suite: "suite", CaseID: "case", Production: caseRow.Expected, Oracle: caseRow.Expected}}); err == nil {
		t.Fatal("accepted duplicate competence key")
	}
}
