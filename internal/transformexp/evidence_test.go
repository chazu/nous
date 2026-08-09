package transformexp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	competenceLeaves := 0
	for name := range evidence.Files {
		if strings.HasPrefix(name, "competence/cases/") {
			competenceLeaves++
		}
	}
	if competenceLeaves != 28 || evidence.Report.Competence.Microcases != 14 {
		t.Fatalf("competence evidence leaves=%d report=%+v", competenceLeaves, evidence.Report.Competence)
	}
	for _, policy := range empiricalPolicies {
		for _, c := range curricula {
			path := "pre/" + string(policy) + "/" + c.PolicyTokens[policy] + ".json"
			if len(evidence.Files[path]) == 0 {
				t.Fatalf("missing shared pre-execution leaf %s", path)
			}
			for _, role := range []string{"primary", "audit"} {
				legacy := role + "/" + string(policy) + "/" + formatOrdinal(c.Ordinal) + "/premanifest.json"
				if _, exists := evidence.Files[legacy]; exists {
					t.Fatalf("role-specific post-execution premanifest remains at %s", legacy)
				}
			}
		}
	}
	if bytes.Contains(evidence.EvidenceGraph, evidence.ReportBytes) {
		t.Fatal("evidence graph contains report and creates a hash cycle")
	}
}

func formatOrdinal(value int) string {
	return fmt.Sprintf("%03d", value)
}

func TestPreparedEvidencePersistsPremanifestBeforeExecution(t *testing.T) {
	c, err := makeCurriculum(0, 0, 841900)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	base := transcriptPath(root, "safe")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	fixtureDigest, err := persistPreparedFixtures(root, "safe", []curriculum{c})
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtureDigest) != 64 {
		t.Fatalf("fixture digest length = %d", len(fixtureDigest))
	}
	for _, policy := range empiricalPolicies {
		path := filepath.Join(base, "pre", string(policy), c.PolicyTokens[policy]+".json")
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		view, err := decodePolicyView(c)
		if err != nil {
			t.Fatal(err)
		}
		if want := policyManifestBytes(view, policy); !bytes.Equal(got, want) {
			t.Fatalf("persisted premanifest differs for %s", policy)
		}
	}
}
