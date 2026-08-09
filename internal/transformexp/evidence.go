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
	if panel != "safe" {
		return panelEvidence{}, fmt.Errorf("generic evidence builder cannot construct protected panel %q", panel)
	}
	files, fixtureRoot, err := buildPreparedEvidence(panel, curricula)
	if err != nil {
		return panelEvidence{}, err
	}
	return buildPanelEvidenceFromPrepared(domainsDir, panel, files, fixtureRoot, len(curricula), authority, nil, reviewAuthority)
}

func buildPanelEvidenceFromPrepared(domainsDir, panel string, files map[string][]byte, fixtureRoot []byte, count int, authority uint64, lockedPairs [][2]uint64, reviewAuthority []byte) (panelEvidence, error) {
	executionCurricula, err := decodePreparedCurricula(files, panel, count)
	if err != nil {
		return panelEvidence{}, err
	}
	report, artifacts, err := runPanelDetailedWithPairs(domainsDir, panel, executionCurricula, authority, lockedPairs)
	if err != nil {
		return panelEvidence{}, err
	}
	report.FixtureRootDigest = digestBytes(fixtureRoot)
	if len(reviewAuthority) != 0 {
		if !canonicalJSON(reviewAuthority) {
			return panelEvidence{}, fmt.Errorf("review authority is not canonical JSON")
		}
		files["review-authority.json"] = bytes.Clone(reviewAuthority)
	}
	primaryManifest, err := addExecutionEvidence(files, "primary", executionCurricula, report.Rows, artifacts.Primary)
	if err != nil {
		return panelEvidence{}, err
	}
	auditManifest, err := addExecutionEvidence(files, "audit", executionCurricula, report.Rows, artifacts.Audit)
	if err != nil {
		return panelEvidence{}, err
	}
	files["primary/execution-manifest.json"] = primaryManifest
	files["audit/execution-manifest.json"] = auditManifest
	report.PrimaryManifestDigest = digestBytes(primaryManifest)
	report.AuditManifestDigest = digestBytes(auditManifest)
	competenceFiles, err := runTransformMicrocases()
	if err != nil {
		return panelEvidence{}, err
	}
	for name, value := range competenceFiles {
		files[name] = value
	}
	competenceBytes, _ := json.Marshal(report.Competence)
	files["competence/report.json"] = competenceBytes
	competenceRoot, err := canonicalEvidenceRoot("transform-competence-root/v1", "", competenceFiles)
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

func buildPreparedEvidence(panel string, curricula []curriculum) (map[string][]byte, []byte, error) {
	files, fixtureRoot, err := buildFixtureEvidence(panel, curricula)
	if err != nil {
		return nil, nil, err
	}
	files["fixture-root.json"] = fixtureRoot
	for _, policy := range empiricalPolicies {
		for _, c := range curricula {
			view, err := decodePolicyView(c)
			if err != nil {
				return nil, nil, err
			}
			path := fmt.Sprintf("pre/%s/%s.json", policy, c.PolicyTokens[policy])
			if _, exists := files[path]; exists {
				return nil, nil, fmt.Errorf("duplicate premanifest path %s", path)
			}
			files[path] = policyManifestBytes(view, policy)
		}
	}
	return files, fixtureRoot, nil
}

func buildFixtureEvidence(panel string, curricula []curriculum) (map[string][]byte, []byte, error) {
	fixtureFiles := map[string][]byte{}
	for _, c := range curricula {
		base := fmt.Sprintf("fixtures/%03d", c.Ordinal)
		fixtureFiles[base+"/training.json"] = bytes.Clone(c.Training)
		fixtureFiles[base+"/heldout.json"] = bytes.Clone(c.Heldout)
		scorer, err := scorerFixtureBytes(c)
		if err != nil {
			return nil, nil, err
		}
		fixtureFiles[base+"/scorer.json"] = scorer
		fixtureFiles[base+"/family.json"] = mustJSON([]any{"transform-family-assignment/v1", c.Ordinal, c.Family})
		fixtureFiles[base+"/acceptance.json"] = mustJSON([]any{"transform-generator-acceptance-ledger/v1", c.GeneratorLedger.Applications, c.GeneratorLedger.Work, c.GeneratorLedger.MatrixSHA256, c.GeneratorLedger.Accepted})
		fixtureFiles[base+"/authority.json"] = mustJSON([]any{"transform-fixture-authority/v1", c.Ordinal, panel, c.PanelCommitment})
		fixtureFiles[base+"/queue.json"] = policyQueueBytes(c)
	}
	fixtureRoot, err := canonicalEvidenceRoot("transform-fixture-root/v1", panel, fixtureFiles)
	if err != nil {
		return nil, nil, err
	}
	return fixtureFiles, fixtureRoot, nil
}

func decodePreparedCurricula(files map[string][]byte, panel string, count int) ([]curriculum, error) {
	result := make([]curriculum, count)
	for ordinal := 0; ordinal < count; ordinal++ {
		base := fmt.Sprintf("fixtures/%03d", ordinal)
		read := func(name string) ([]byte, error) {
			value, ok := files[base+"/"+name]
			if !ok {
				return nil, fmt.Errorf("missing prepared fixture %s/%s", base, name)
			}
			return bytes.Clone(value), nil
		}
		training, err := read("training.json")
		if err != nil {
			return nil, err
		}
		heldout, err := read("heldout.json")
		if err != nil {
			return nil, err
		}
		queue, err := read("queue.json")
		if err != nil {
			return nil, err
		}
		scorer, err := read("scorer.json")
		if err != nil {
			return nil, err
		}
		familyBytes, err := read("family.json")
		if err != nil {
			return nil, err
		}
		acceptanceBytes, err := read("acceptance.json")
		if err != nil {
			return nil, err
		}
		authorityBytes, err := read("authority.json")
		if err != nil {
			return nil, err
		}
		var familyWire, acceptanceWire, authorityWire []json.RawMessage
		var familyVersion, acceptanceVersion, authorityVersion, gotPanel, panelCommitment string
		var familyOrdinal, family, authorityOrdinal int
		ledger := acceptanceLedger{}
		if json.Unmarshal(familyBytes, &familyWire) != nil || len(familyWire) != 3 || json.Unmarshal(familyWire[0], &familyVersion) != nil || familyVersion != "transform-family-assignment/v1" || json.Unmarshal(familyWire[1], &familyOrdinal) != nil || familyOrdinal != ordinal || json.Unmarshal(familyWire[2], &family) != nil || family < 0 || family >= len(familySchemas) {
			return nil, fmt.Errorf("invalid prepared family %d", ordinal)
		}
		if json.Unmarshal(acceptanceBytes, &acceptanceWire) != nil || len(acceptanceWire) != 5 || json.Unmarshal(acceptanceWire[0], &acceptanceVersion) != nil || acceptanceVersion != "transform-generator-acceptance-ledger/v1" || json.Unmarshal(acceptanceWire[1], &ledger.Applications) != nil || json.Unmarshal(acceptanceWire[2], &ledger.Work) != nil || json.Unmarshal(acceptanceWire[3], &ledger.MatrixSHA256) != nil || json.Unmarshal(acceptanceWire[4], &ledger.Accepted) != nil {
			return nil, fmt.Errorf("invalid prepared acceptance ledger %d", ordinal)
		}
		if json.Unmarshal(authorityBytes, &authorityWire) != nil || len(authorityWire) != 4 || json.Unmarshal(authorityWire[0], &authorityVersion) != nil || authorityVersion != "transform-fixture-authority/v1" || json.Unmarshal(authorityWire[1], &authorityOrdinal) != nil || authorityOrdinal != ordinal || json.Unmarshal(authorityWire[2], &gotPanel) != nil || gotPanel != panel || json.Unmarshal(authorityWire[3], &panelCommitment) != nil || !digestString(panelCommitment) {
			return nil, fmt.Errorf("invalid prepared authority %d", ordinal)
		}
		c := curriculum{Ordinal: ordinal, Family: family, Panel: panel, PanelCommitment: panelCommitment, Training: training, Heldout: heldout, Queue: queue, Scorer: scorer, GeneratorLedger: ledger}
		scorerView, err := decodeScorerView(c)
		if err != nil || scorerView.Family != family {
			return nil, fmt.Errorf("invalid prepared scorer %d: %w", ordinal, err)
		}
		c.SeedCommitment, c.AcceptedAttempt, c.Latent, c.Expected = scorerView.SeedCommitment, scorerView.AcceptedAttempt, scorerView.Latent, scorerView.Expected
		policyView, err := decodePolicyView(c)
		if err != nil {
			return nil, fmt.Errorf("invalid prepared policy view %d: %w", ordinal, err)
		}
		if _, err := decodeHeldoutInputs(c); err != nil {
			return nil, fmt.Errorf("invalid prepared heldout %d: %w", ordinal, err)
		}
		c.PolicyTokens, c.PolicyRandomness = policyView.PolicyTokens, policyView.PolicyRandomness
		result[ordinal] = c
	}
	return result, nil
}

func addExecutionEvidence(files map[string][]byte, role string, curricula []curriculum, reportRows []PolicyReportRow, bundles map[string]TransformTranscriptBundle) ([]byte, error) {
	rowsByKey := map[string]PolicyReportRow{}
	for _, row := range reportRows {
		rowsByKey[fmt.Sprintf("%s/%03d", row.Policy, row.Ordinal)] = row
	}
	var rows []any
	for _, policy := range empiricalPolicies {
		for _, c := range curricula {
			policyView, err := decodePolicyView(c)
			if err != nil {
				return nil, err
			}
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
			premanifestPath := fmt.Sprintf("pre/%s/%s.json", policy, c.PolicyTokens[policy])
			premanifest, ok := files[premanifestPath]
			if !ok || !bytes.Equal(premanifest, policyManifestBytes(policyView, policy)) {
				return nil, fmt.Errorf("missing or changed pre-execution manifest %s", premanifestPath)
			}
			row := rowsByKey[key]
			artifactDigest, err := reconstructArtifactDigest(bundle.Raw, bundle.Objects, policy, row.Terminal)
			if err != nil || artifactDigest != row.SchemaSHA256 {
				return nil, fmt.Errorf("reconstruct %s frozen artifact %s: got %s want %s: %w", role, key, artifactDigest, row.SchemaSHA256, err)
			}
			heldoutResults, err := reconstructHeldoutResults(bundle.Raw, bundle.Objects, c.Heldout)
			if err != nil {
				return nil, fmt.Errorf("reconstruct %s heldout results %s: %w", role, key, err)
			}
			rows = append(rows, []any{policy, c.Ordinal, c.PolicyTokens[policy], digestBytes(premanifest), digestBytes(bundle.Gzip), len(bundle.Raw), len(bundle.Gzip), bytes.Count(bundle.Raw, []byte{'\n'}), digestBytes(objectRoot), bundle.Vector, bundle.Work, row.Applications, row.Terminal, artifactDigest, digestBytes(c.Training), digestBytes(heldoutResults)})
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
