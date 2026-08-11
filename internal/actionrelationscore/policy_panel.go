package actionrelationscore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationledger"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type PolicyPanelCurriculumEvidence struct {
	Curriculum    int
	ManifestFiles []actionrelationexp.EvidenceFile
	PackFiles     []actionrelationexp.EvidenceFile
}

type PolicyPanelSummary struct {
	Panel               string
	Authority           string
	FixtureDigest       string
	Curricula           []PolicyCurriculumSummary
	RunEvidence         actionrelationexp.RunEvidencePack
	RunEvidenceManifest actionrelationexp.EvidenceFile
	ObjectRoots         []actionrelationexp.ObjectManifestRef
	IndexRoots          []actionrelationexp.ObjectManifestRef
	JournalRoots        []actionrelationexp.TranscriptManifestRef
	InputRoots          []actionrelationexp.TranscriptManifestRef
	DetailRoots         []actionrelationexp.TranscriptManifestRef
	Tables              []actionrelationexp.TableManifestRef
	StructuralMaps      []actionrelationexp.AuthorityRef
	StoreBoundaries     []actionrelationexp.StoreBoundaryRow
}

func ExecutePublicPanel(domainsDir string, public PublicPanel, consume func(PolicyPanelCurriculumEvidence) error) (PolicyPanelSummary, error) {
	if consume == nil || verifyPublicPanelWire(publicPanelWire{Version: "actionrelation-public-policy-panel/v1", Panel: public.panel, Authority: public.authority, FixtureDigest: public.fixtureDigest, Curricula: public.curricula}) != nil {
		return PolicyPanelSummary{}, fmt.Errorf("invalid public policy panel")
	}
	result := PolicyPanelSummary{Panel: public.panel, Authority: public.authority, FixtureDigest: public.fixtureDigest}
	var records []actionrelationexp.RunEvidenceRecord
	for _, curriculum := range public.curricula {
		raw, err := executePolicyCurriculum(domainsDir, public.panel, public.authority, curriculum)
		if err != nil {
			return PolicyPanelSummary{}, fmt.Errorf("%s curriculum %d: %w", public.panel, curriculum.Curriculum, err)
		}
		evidence, err := BuildPolicyCurriculumEvidence(raw)
		if err != nil {
			return PolicyPanelSummary{}, err
		}
		manifests, err := BuildPolicyCurriculumManifests(raw, evidence)
		if err != nil {
			return PolicyPanelSummary{}, err
		}
		if err := consume(PolicyPanelCurriculumEvidence{Curriculum: curriculum.Curriculum, ManifestFiles: manifests.ManifestFiles, PackFiles: manifests.PackFiles}); err != nil {
			return PolicyPanelSummary{}, err
		}
		summary, err := summarizePolicyCurriculum(raw)
		if err != nil {
			return PolicyPanelSummary{}, err
		}
		result.Curricula = append(result.Curricula, summary)
		result.ObjectRoots = append(result.ObjectRoots, manifests.ObjectRoots...)
		result.IndexRoots = append(result.IndexRoots, manifests.IndexRoots...)
		result.JournalRoots = append(result.JournalRoots, manifests.JournalRoots...)
		result.InputRoots = append(result.InputRoots, manifests.InputRoots...)
		result.DetailRoots = append(result.DetailRoots, manifests.DetailRoots...)
		result.Tables = append(result.Tables, manifests.Tables...)
		result.StructuralMaps = append(result.StructuralMaps, manifests.StructuralMap)
		result.StoreBoundaries = append(result.StoreBoundaries, manifests.StoreBoundaries...)
		records = append(records, evidence.RunEvidence...)
	}
	var err error
	result.RunEvidence, err = actionrelationexp.BuildRunEvidencePack(result.Panel, result.Authority, records)
	if err != nil {
		return PolicyPanelSummary{}, err
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(result.Panel)
	result.RunEvidenceManifest = actionrelationexp.EvidenceFile{Path: evidenceRoot + "/manifests/run-evidence-root.json", Mode: "100644", Data: result.RunEvidence.Canonical}
	if err := VerifyPolicyPanelSummaryForPublic(result, public); err != nil {
		return PolicyPanelSummary{}, err
	}
	return result, nil
}

func VerifyPolicyPanelSummary(value PolicyPanelSummary) error {
	want := map[string]int{"development": 16, "validation": 24, "locked": 32}[value.Panel]
	if want == 0 || value.Authority == "" || !digestText(value.FixtureDigest) || len(value.Curricula) != want || len(value.ObjectRoots) != want*3 || len(value.IndexRoots) != want*3 || len(value.JournalRoots) != want*44 || len(value.InputRoots) != want*44 || len(value.DetailRoots) != want*44 || len(value.Tables) != want*14 || len(value.StructuralMaps) != want || len(value.StoreBoundaries) != want*2 || actionrelationexp.VerifyRunEvidencePack(value.RunEvidence) != nil {
		return fmt.Errorf("invalid public policy summary cardinality")
	}
	for index, curriculum := range value.Curricula {
		if curriculum.Curriculum != index || len(curriculum.Acquisitions) != 2 || len(curriculum.Worlds) != 42 {
			return fmt.Errorf("invalid public policy curriculum summary")
		}
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(value.Panel)
	if value.RunEvidenceManifest.Path != evidenceRoot+"/manifests/run-evidence-root.json" || value.RunEvidenceManifest.Mode != "100644" || !bytes.Equal(value.RunEvidenceManifest.Data, value.RunEvidence.Canonical) {
		return fmt.Errorf("public policy run-evidence manifest mismatch")
	}
	return nil
}

func VerifyPolicyPanelSummaryForPublic(value PolicyPanelSummary, public PublicPanel) error {
	if VerifyPolicyPanelSummary(value) != nil || value.Panel != public.panel || value.Authority != public.authority || value.FixtureDigest != public.fixtureDigest {
		return fmt.Errorf("policy summary differs from public authority")
	}
	runEvidence := make(map[string]actionrelationexp.RunEvidenceRecord, len(value.RunEvidence.Records))
	for _, record := range value.RunEvidence.Records {
		runEvidence[record.RunID] = record
	}
	for curriculum, publicCurriculum := range public.curricula {
		summary := value.Curricula[curriculum]
		for acquisition, scope := range []string{"nous", "no-guard"} {
			runID, err := actionrelationledger.AcquisitionRunID(value.Panel, value.Authority, curriculum, scope)
			item := summary.Acquisitions[acquisition]
			record, ok := runEvidence[runID]
			if err != nil || item.Scope != scope || !ok || record.OperationRoot != item.OperationRoot.Digest || actionrelationexp.ValidateObject(46, item.OperationRoot.Canonical) != nil || actionrelationexp.ValidateObject(35, item.BoundaryCanonical) != nil {
				return fmt.Errorf("policy acquisition authority changed: %d/%s", curriculum, scope)
			}
		}
		for policyOrdinal, policy := range Policies {
			for worldOrdinal, publicWorld := range publicCurriculum.Worlds {
				normalized, err := (actionrelations.World{State: publicWorld.State, Actions: publicWorld.Actions}).Normalize()
				if err != nil {
					return err
				}
				worldDigest, _ := normalized.Digest()
				runID, err := actionrelationledger.UtilityRunID(value.Panel, value.Authority, curriculum, string(policy), worldOrdinal, worldDigest)
				item := summary.Worlds[policyOrdinal*6+worldOrdinal]
				record, ok := runEvidence[runID]
				if err != nil || item.Policy != policy || item.WorldOrdinal != worldOrdinal || item.RunID != runID || item.WorldDigest != worldDigest || !ok || record.OperationRoot != item.OperationRoot.Digest || record.WorkTerminal != emptyToZeroWorkTerminal(item.WorkTerminal) || actionrelationexp.ValidateObject(46, item.OperationRoot.Canonical) != nil {
					return fmt.Errorf("policy utility authority changed: %d/%s/%d", curriculum, policy, worldOrdinal)
				}
			}
		}
	}
	return nil
}

func emptyToZeroWorkTerminal(value string) string {
	if value == zeroDigest {
		return ""
	}
	return value
}

func MarshalPolicyPanelSummary(summary PolicyPanelSummary) ([]byte, error) {
	if err := VerifyPolicyPanelSummary(summary); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(summary)
	if err != nil || int64(len(canonical)) > maximumPanelSummaryBytes {
		return nil, fmt.Errorf("policy panel summary exceeds isolated result cap")
	}
	return canonical, nil
}

func ParsePolicyPanelSummary(reader io.Reader, size int64) (PolicyPanelSummary, error) {
	if size < 1 || size > maximumPanelSummaryBytes {
		return PolicyPanelSummary{}, fmt.Errorf("invalid isolated policy summary size")
	}
	data, err := io.ReadAll(io.LimitReader(reader, size+1))
	if err != nil || int64(len(data)) != size {
		return PolicyPanelSummary{}, fmt.Errorf("invalid isolated policy summary bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value PolicyPanelSummary
	if err := decoder.Decode(&value); err != nil || decoder.Decode(&struct{}{}) != io.EOF || VerifyPolicyPanelSummary(value) != nil {
		return PolicyPanelSummary{}, fmt.Errorf("invalid isolated policy summary")
	}
	want, _ := json.Marshal(value)
	if !bytes.Equal(want, data) {
		return PolicyPanelSummary{}, fmt.Errorf("noncanonical isolated policy summary")
	}
	return value, nil
}

func FinalizePolicyPanel(sealed SealedPanel, raw PolicyPanelSummary, reopen func(string) ([]byte, error), consume func([]actionrelationexp.EvidenceFile) error) (PanelSummary, error) {
	if reopen == nil || consume == nil || actionrelationfixture.VerifyGeneratedPanel(sealed.attempts, sealed.fixture) != nil || VerifyPolicyPanelSummaryForPublic(raw, sealed.public) != nil {
		return PanelSummary{}, fmt.Errorf("private scorer and public policy panel differ")
	}
	result := PanelSummary{Panel: raw.Panel, Authority: raw.Authority, Fixture: sealed.fixture, RunEvidence: raw.RunEvidence, RunEvidenceManifest: raw.RunEvidenceManifest, ObjectRoots: slices.Clone(raw.ObjectRoots), IndexRoots: slices.Clone(raw.IndexRoots), JournalRoots: slices.Clone(raw.JournalRoots), InputRoots: slices.Clone(raw.InputRoots), DetailRoots: slices.Clone(raw.DetailRoots), Tables: slices.Clone(raw.Tables), StructuralMaps: slices.Clone(raw.StructuralMaps), StoreBoundaries: slices.Clone(raw.StoreBoundaries)}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(raw.Panel)
	fixtureFile := actionrelationexp.EvidenceFile{Path: actionrelationexp.ExpectedAuthorityPath(raw.Panel, "fixture-root"), Mode: "100644", Data: sealed.fixture.Canonical}
	delayed := []actionrelationexp.EvidenceFile{fixtureFile}
	for curriculum, generated := range sealed.attempts {
		scored, err := scorePolicyCurriculumSummary(generated, raw.Curricula[curriculum])
		if err != nil {
			return PanelSummary{}, fmt.Errorf("score curriculum %d: %w", curriculum, err)
		}
		records, err := curriculumAuthorityObjectsDelayed(generated, scored, raw.Curricula[curriculum].Acquisitions[0].BoundaryCanonical, raw.Curricula[curriculum].Acquisitions[1].BoundaryCanonical)
		if err != nil {
			return PanelSummary{}, err
		}
		authority, err := actionrelationexp.BuildObjectBundleAt(evidenceRoot, actionrelationexp.ObjectScope{Curriculum: curriculum, Class: "authority"}, records)
		if err != nil {
			return PanelSummary{}, err
		}
		files, objectRef, indexRef, err := BuildDelayedAuthorityFiles(raw.Panel, curriculum, authority)
		if err != nil {
			return PanelSummary{}, err
		}
		delayed = append(delayed, files...)
		result.ObjectRoots = append(result.ObjectRoots, objectRef)
		result.IndexRoots = append(result.IndexRoots, indexRef)
		result.WorldRows = append(result.WorldRows, scored.WorldRows...)
		result.CurriculumRows = append(result.CurriculumRows, scored.CurriculumRows...)
	}
	result.WorldPolicyRowsRoot, _ = WorldPolicyRowsRoot(result.WorldRows)
	result.CurriculumRowsRoot, _ = CurriculumPolicyRowsRoot(result.CurriculumRows)
	if err := VerifyPanelSummary(result); err != nil {
		return PanelSummary{}, err
	}
	provisional := make(map[string][]byte, len(delayed))
	for _, file := range delayed {
		if file.Mode != "100644" || provisional[file.Path] != nil {
			return PanelSummary{}, fmt.Errorf("duplicate provisional scorer authority path")
		}
		provisional[file.Path] = file.Data
	}
	read := func(path string) ([]byte, error) {
		if data, ok := provisional[path]; ok {
			return slices.Clone(data), nil
		}
		return reopen(path)
	}
	fixtureRef, _ := actionrelationexp.Reference(fixtureFile.Path, fixtureFile.Data)
	runRef, err := actionrelationexp.Reference(raw.RunEvidenceManifest.Path, raw.RunEvidenceManifest.Data)
	if err != nil {
		return PanelSummary{}, err
	}
	if _, err := actionrelationexp.VerifyRetainedPacks(actionrelationexp.RetainedPackRefs{
		Panel: result.Panel, Authority: result.Authority, Fixture: fixtureRef, RunEvidence: runRef,
		ObjectRoots: result.ObjectRoots, IndexRoots: result.IndexRoots,
		JournalRoots: result.JournalRoots, InputRoots: result.InputRoots, DetailRoots: result.DetailRoots,
		Tables: result.Tables, StructuralMaps: result.StructuralMaps, StoreBoundaries: result.StoreBoundaries,
	}, read); err != nil {
		return PanelSummary{}, fmt.Errorf("reconstruct scorer authority from retained typed evidence: %w", err)
	}
	if err := consume(delayed); err != nil {
		return PanelSummary{}, err
	}
	return result, nil
}
