package transformexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/chazu/nous/internal/transformfixturecore"
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
	return buildPanelEvidenceFromPrepared(domainsDir, panel, files, fixtureRoot, len(curricula), authority, reviewAuthority)
}

func buildPanelEvidenceFromPrepared(domainsDir, panel string, files map[string][]byte, fixtureRoot []byte, count int, authority uint64, reviewAuthority []byte) (panelEvidence, error) {
	var lockedPairs [][2]uint64
	if panel == "locked" {
		rootCommitment, err := parseLockedStatisticsAuthority(files["statistics/authority.json"])
		if err != nil {
			return panelEvidence{}, err
		}
		lockedPairs, err = lockedStatisticsPairs(rootCommitment)
		if err != nil {
			return panelEvidence{}, err
		}
	} else if _, exists := files["statistics/authority.json"]; exists {
		return panelEvidence{}, errors.New("statistics authority supplied outside locked panel")
	}
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
	primaryManifest, err := addExecutionEvidence(files, "primary", executionCurricula, report.Rows, artifacts.Primary, artifacts.PrimaryStores, artifacts.PrimaryPrograms)
	if err != nil {
		return panelEvidence{}, err
	}
	auditManifest, err := addExecutionEvidence(files, "audit", executionCurricula, report.Rows, artifacts.Audit, artifacts.AuditStores, artifacts.AuditPrograms)
	if err != nil {
		return panelEvidence{}, err
	}
	files["primary/execution-manifest.json"] = primaryManifest
	files["audit/execution-manifest.json"] = auditManifest
	report.PrimaryManifestDigest = digestBytes(primaryManifest)
	report.AuditManifestDigest = digestBytes(auditManifest)
	competenceFiles, err := runTransformMicrocases(domainsDir)
	if err != nil {
		return panelEvidence{}, err
	}
	for name, value := range competenceFiles {
		files[name] = value
	}
	competenceBytes, _ := json.Marshal(report.Competence)
	files["competence/report.json"] = competenceBytes
	competenceRoot, err := canonicalEvidenceRoot("transform-competence-root/v2", "", competenceFiles)
	if err != nil {
		return panelEvidence{}, err
	}
	files["competence/root.json"] = competenceRoot
	files["acceptance/generator.json"] = acceptanceDiagnosticsBytes("generator", report.GeneratorAcceptance)
	files["acceptance/oracle.json"] = acceptanceDiagnosticsBytes("oracle", report.OracleAcceptance)
	graph, err := canonicalEvidenceRoot("transform-evidence-graph/v2", panel, files)
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

func buildPreparedEvidence(panel string, curricula []curriculum, statisticAuthority ...string) (map[string][]byte, []byte, error) {
	files, fixtureRoot, err := buildFixtureEvidence(panel, curricula, statisticAuthority...)
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

func buildFixtureEvidence(panel string, curricula []curriculum, statisticAuthority ...string) (map[string][]byte, []byte, error) {
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
		fixtureFiles[base+"/queue.json"] = policyQueueBytes(c)
	}
	if panel == "locked" {
		if len(statisticAuthority) != 1 || !isLowerHex(statisticAuthority[0], 64) {
			return nil, nil, errors.New("locked fixture requires one root commitment")
		}
		fixtureFiles["statistics/authority.json"] = mustJSON([]any{"transform-statistics-authority/v2", "locked", statisticAuthority[0], 10000, 10000})
	} else if len(statisticAuthority) != 0 {
		return nil, nil, errors.New("statistics authority supplied outside locked fixture")
	}
	fixtureRoot, err := canonicalEvidenceRoot("transform-fixture-root/v2", panel, fixtureFiles)
	if err != nil {
		return nil, nil, err
	}
	return fixtureFiles, fixtureRoot, nil
}

func parseLockedStatisticsAuthority(data []byte) (string, error) {
	var wire []json.RawMessage
	var version, panel, commitment string
	var bootstrap, randomization int
	if !canonicalJSON(data) || json.Unmarshal(data, &wire) != nil || len(wire) != 5 ||
		json.Unmarshal(wire[0], &version) != nil || version != "transform-statistics-authority/v2" ||
		json.Unmarshal(wire[1], &panel) != nil || panel != "locked" ||
		json.Unmarshal(wire[2], &commitment) != nil || !isLowerHex(commitment, 64) ||
		json.Unmarshal(wire[3], &bootstrap) != nil || bootstrap != 10000 ||
		json.Unmarshal(wire[4], &randomization) != nil || randomization != 10000 {
		return "", errors.New("invalid locked statistics authority")
	}
	return commitment, nil
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
		var familyWire []json.RawMessage
		var familyVersion, panelCommitment string
		var familyOrdinal, family int
		if json.Unmarshal(familyBytes, &familyWire) != nil || len(familyWire) != 3 || json.Unmarshal(familyWire[0], &familyVersion) != nil || familyVersion != "transform-family-assignment/v1" || json.Unmarshal(familyWire[1], &familyOrdinal) != nil || familyOrdinal != ordinal || json.Unmarshal(familyWire[2], &family) != nil || family < 0 || family >= len(familySchemas) {
			return nil, fmt.Errorf("invalid prepared family %d", ordinal)
		}
		panelCommitment, err = preparedPanelCommitment(files, queue)
		if err != nil {
			return nil, fmt.Errorf("invalid prepared panel commitment %d: %w", ordinal, err)
		}
		// scorer is an opaque sealed blob here. Its framing and truth-bearing
		// contents are decoded only after the policy terminal is immutable.
		c := curriculum{Ordinal: ordinal, Family: family, Panel: panel, PanelCommitment: panelCommitment, Training: training, Heldout: heldout, Queue: queue, Scorer: scorer}
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

func preparedPanelCommitment(files map[string][]byte, queue []byte) (string, error) {
	var queueWire []json.RawMessage
	var rows [][]json.RawMessage
	if json.Unmarshal(queue, &queueWire) != nil || len(queueWire) != 2 || json.Unmarshal(queueWire[1], &rows) != nil || len(rows) == 0 || len(rows[0]) != 4 {
		return "", errors.New("policy queue wire")
	}
	var policy Policy
	var token string
	if json.Unmarshal(rows[0][0], &policy) != nil || policy != empiricalPolicies[0] || json.Unmarshal(rows[0][1], &token) != nil {
		return "", errors.New("policy queue first row")
	}
	premanifest := files[fmt.Sprintf("pre/%s/%s.json", policy, token)]
	var wire []json.RawMessage
	var commitment string
	if json.Unmarshal(premanifest, &wire) != nil || len(wire) != 10 || json.Unmarshal(wire[3], &commitment) != nil || !digestString(commitment) {
		return "", errors.New("policy premanifest panel commitment")
	}
	return commitment, nil
}

func addExecutionEvidence(files map[string][]byte, role string, curricula []curriculum, reportRows []PolicyReportRow, bundles map[string]TransformTranscriptBundle, trainingStores, trainingPrograms map[string][]byte) ([]byte, error) {
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
			trainingStore := trainingStores[key]
			storeBacked := policy == NousRefine || policy == PositiveLGG || policy == ConcreteReplay || policy == NoEqualityGuard
			if storeBacked && (len(trainingStore) == 0 || !canonicalStoreJSON(trainingStore)) {
				return nil, fmt.Errorf("missing canonical %s training Store %s", role, key)
			}
			if !storeBacked && len(trainingStore) != 0 {
				return nil, fmt.Errorf("stateless %s policy has a training Store %s", role, key)
			}
			trainingStoreDigest := ""
			if storeBacked {
				files[base+"/training-store.json"] = bytes.Clone(trainingStore)
				trainingStoreDigest = digestBytes(trainingStore)
			}
			if programs := trainingPrograms[key]; len(programs) != 0 {
				if _, err := transformfixturecore.ParseProgramBatch(programs); err != nil {
					return nil, fmt.Errorf("invalid %s training programs %s", role, key)
				}
				files[base+"/training-programs.json"] = bytes.Clone(programs)
			}
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
			objectRoot, err := canonicalEvidenceRoot("transform-objects/v2", "", objectFiles)
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
			rows = append(rows, []any{policy, c.Ordinal, c.PolicyTokens[policy], digestBytes(premanifest), digestBytes(bundle.Gzip), len(bundle.Raw), len(bundle.Gzip), bytes.Count(bundle.Raw, []byte{'\n'}), digestBytes(objectRoot), bundle.Vector, bundle.Work, row.Applications, row.Terminal, artifactDigest, trainingStoreDigest, digestBytes(heldoutResults)})
		}
	}
	return mustJSON([]any{"transform-execution/v2", role, rows}), nil
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

func canonicalStoreJSON(data []byte) bool {
	if len(data) == 0 || len(data) > 16777216 {
		return false
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) == nil {
		return false
	}
	root, ok := value.(map[string]any)
	if !ok || len(root) > 20000 {
		return false
	}
	for name, rawSlots := range root {
		if !validStoreKey(name) {
			return false
		}
		slots, ok := rawSlots.(map[string]any)
		if !ok || len(slots) > 256 {
			return false
		}
		for slot, slotValue := range slots {
			if !validStoreKey(slot) || !validStoreValue(slotValue, 0) {
				return false
			}
		}
	}
	canonical, err := json.Marshal(value)
	return err == nil && bytes.Equal(canonical, data)
}

func validStoreKey(value string) bool {
	return value != "" && len(value) <= 256 && utf8.ValidString(value)
}

func validStoreValue(value any, depth int) bool {
	if depth > 64 {
		return false
	}
	switch typed := value.(type) {
	case nil, bool:
		return true
	case json.Number:
		_, err := typed.Int64()
		return err == nil
	case string:
		return len(typed) <= 61440 && utf8.ValidString(typed)
	case []any:
		for _, item := range typed {
			if !validStoreValue(item, depth+1) {
				return false
			}
		}
		return true
	case map[string]any:
		for key, item := range typed {
			if !validStoreKey(key) || !validStoreValue(item, depth+1) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
