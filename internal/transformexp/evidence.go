package transformexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

type evidenceLeaf struct {
	Path   string
	Digest string
	Bytes  int
}

type panelEvidence struct {
	Report        SafePanelReport
	ReportBytes   []byte
	EvidenceGraph []byte
	Files         map[string][]byte
}

func buildPanelEvidence(domainsDir, panel string, curricula []curriculum, authority uint64, reviewAuthority []byte) (panelEvidence, error) {
	report, artifacts, err := runPanelDetailed(domainsDir, panel, curricula, authority)
	if err != nil {
		return panelEvidence{}, err
	}
	files := map[string][]byte{}
	fixtureFiles := map[string][]byte{}
	for _, c := range curricula {
		base := fmt.Sprintf("fixtures/%03d", c.Ordinal)
		fixtureFiles[base+"/training.json"] = bytes.Clone(c.Training)
		fixtureFiles[base+"/heldout.json"] = bytes.Clone(c.Heldout)
		expected := make([]any, len(c.Expected))
		for i, value := range c.Expected {
			var output any
			if len(value.Output) != 0 {
				if json.Unmarshal(value.Output, &output) != nil {
					return panelEvidence{}, fmt.Errorf("expected output JSON")
				}
			}
			expected[i] = []any{value.Token, value.Terminal, output}
		}
		var latent any
		if json.Unmarshal(c.Latent, &latent) != nil {
			return panelEvidence{}, fmt.Errorf("latent JSON")
		}
		fixtureFiles[base+"/scorer.json"] = mustJSON([]any{"transform-scorer-curriculum/v1", c.Family, digestBytes(mustJSON([]any{"transform-seed/v1", panel, c.Seed})), 0, latent, expected})
		fixtureFiles[base+"/family.json"] = mustJSON([]any{"transform-family-assignment/v1", c.Ordinal, c.Family})
		fixtureFiles[base+"/queue.json"] = mustJSON([]any{"transform-policy-queue/v1", c.Ordinal, empiricalPolicies})
	}
	fixtureRoot, err := canonicalEvidenceRoot("transform-fixture-root/v1", panel, fixtureFiles)
	if err != nil {
		return panelEvidence{}, err
	}
	for name, value := range fixtureFiles {
		files[name] = value
	}
	files["fixture-root.json"] = fixtureRoot
	report.FixtureRootDigest = digestBytes(fixtureRoot)
	if len(reviewAuthority) != 0 {
		if !canonicalJSON(reviewAuthority) {
			return panelEvidence{}, fmt.Errorf("review authority is not canonical JSON")
		}
		files["review-authority.json"] = bytes.Clone(reviewAuthority)
	}
	primaryManifest, err := addExecutionEvidence(files, "primary", curricula, report.Rows, artifacts.Primary)
	if err != nil {
		return panelEvidence{}, err
	}
	auditManifest, err := addExecutionEvidence(files, "audit", curricula, report.Rows, artifacts.Audit)
	if err != nil {
		return panelEvidence{}, err
	}
	files["primary/execution-manifest.json"] = primaryManifest
	files["audit/execution-manifest.json"] = auditManifest
	report.PrimaryManifestDigest = digestBytes(primaryManifest)
	report.AuditManifestDigest = digestBytes(auditManifest)
	competenceBytes, _ := json.Marshal(report.Competence)
	files["competence/report.json"] = competenceBytes
	competenceRoot, err := canonicalEvidenceRoot("transform-competence-root/v1", "", map[string][]byte{"competence/report.json": competenceBytes})
	if err != nil {
		return panelEvidence{}, err
	}
	files["competence/root.json"] = competenceRoot
	graph, err := canonicalEvidenceRoot("transform-evidence-graph/v1", panel, files)
	if err != nil {
		return panelEvidence{}, err
	}
	report.EvidenceGraphDigest = digestBytes(graph)
	reportBytes, err := report.JSON()
	if err != nil {
		return panelEvidence{}, err
	}
	return panelEvidence{report, reportBytes, graph, files}, nil
}

func addExecutionEvidence(files map[string][]byte, role string, curricula []curriculum, reportRows []PolicyReportRow, bundles map[string]TransformTranscriptBundle) ([]byte, error) {
	rowsByKey := map[string]PolicyReportRow{}
	for _, row := range reportRows {
		rowsByKey[fmt.Sprintf("%s/%03d", row.Policy, row.Ordinal)] = row
	}
	var rows []any
	for _, policy := range empiricalPolicies {
		for _, c := range curricula {
			key := fmt.Sprintf("%s/%03d", policy, c.Ordinal)
			bundle, ok := bundles[key]
			if !ok || len(bundle.Raw) == 0 || len(bundle.Gzip) == 0 {
				return nil, fmt.Errorf("missing %s bundle %s", role, key)
			}
			base := role + "/" + key
			files[base+"/transcript.jsonl.gz"] = bytes.Clone(bundle.Gzip)
			objectFiles := map[string][]byte{}
			for digest, value := range bundle.Objects {
				if digestBytes(value) != digest {
					return nil, fmt.Errorf("object digest mismatch %s", digest)
				}
				objectPath := base + "/objects/" + digest + ".json"
				files[objectPath] = bytes.Clone(value)
				objectFiles[objectPath] = value
			}
			objectRoot, err := canonicalEvidenceRoot("transform-objects/v1", "", objectFiles)
			if err != nil {
				return nil, err
			}
			files[base+"/object-root.json"] = objectRoot
			premanifest := policyManifestBytes(c, policy)
			files[base+"/premanifest.json"] = premanifest
			row := rowsByKey[key]
			rows = append(rows, []any{policy, c.Ordinal, caseToken(c.Seed, "policy-"+string(policy), 0), digestBytes(premanifest), digestBytes(bundle.Gzip), len(bundle.Raw), len(bundle.Gzip), bytes.Count(bundle.Raw, []byte{'\n'}), digestBytes(objectRoot), bundle.Vector, bundle.Work, row.Applications, row.Terminal, row.SchemaSHA256, digestBytes(c.Training), digestBytes(mustJSON([]any{row.HeldoutCorrectBits, row.FalseApplications}))})
		}
	}
	return mustJSON([]any{"transform-execution/v1", role, rows}), nil
}

func canonicalEvidenceRoot(version, panel string, files map[string][]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("empty evidence root")
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if !validEvidencePath(name) {
			return nil, fmt.Errorf("invalid evidence path %q", name)
		}
		names = append(names, name)
	}
	slices.Sort(names)
	rows := make([]any, len(names))
	for i, name := range names {
		value := files[name]
		rows[i] = []any{name, digestBytes(value), len(value), "100644"}
	}
	if panel == "" {
		return mustJSON([]any{version, rows}), nil
	}
	return mustJSON([]any{version, panel, rows}), nil
}

func validEvidencePath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || path.Clean(value) != value {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	for _, b := range []byte(value) {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func canonicalJSON(data []byte) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	canonical, err := json.Marshal(value)
	return err == nil && bytes.Equal(canonical, data)
}
