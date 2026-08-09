package transformexp

import (
	"bytes"
	"testing"
)

func TestEvidenceRootIsCanonicalAndRejectsPaths(t *testing.T) {
	files := map[string][]byte{"a/x.json": []byte(`[]`), "b.json": []byte(`{}`)}
	first, err := canonicalEvidenceRoot("transform-evidence-graph/v1", "safe", files)
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicalEvidenceRoot("transform-evidence-graph/v1", "safe", map[string][]byte{"b.json": []byte(`{}`), "a/x.json": []byte(`[]`)})
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("nondeterministic root err=%v", err)
	}
	for _, invalid := range []string{"", "/a", "../a", "a/../b", "a\\b", "a//b"} {
		if _, err := canonicalEvidenceRoot("transform-evidence-graph/v1", "safe", map[string][]byte{invalid: []byte{}}); err == nil {
			t.Fatalf("accepted path %q", invalid)
		}
	}
}

func TestPanelEvidenceGraphBindsFixturesTranscriptsAndObjects(t *testing.T) {
	curricula := make([]curriculum, 9)
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 841700+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		curricula[family] = c
	}
	evidence, err := buildPanelEvidence("../../domains", "safe", curricula, 841001, []byte(`["transform-reviews/v1"]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.ReportBytes) == 0 || len(evidence.EvidenceGraph) == 0 || evidence.Report.EvidenceGraphDigest != digestBytes(evidence.EvidenceGraph) || evidence.Report.FixtureRootDigest == "" || evidence.Report.PrimaryManifestDigest == "" || evidence.Report.AuditManifestDigest == "" {
		t.Fatalf("evidence shape report=%+v files=%d", evidence.Report, len(evidence.Files))
	}
	for _, required := range []string{"fixture-root.json", "primary/execution-manifest.json", "audit/execution-manifest.json", "review-authority.json", "competence/root.json"} {
		if len(evidence.Files[required]) == 0 {
			t.Fatalf("missing evidence leaf %s", required)
		}
	}
	if bytes.Contains(evidence.EvidenceGraph, evidence.ReportBytes) {
		t.Fatal("evidence graph contains report and creates a hash cycle")
	}
}
