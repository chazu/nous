package actionrelationscore

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
)

type PanelCurriculumEvidence struct {
	Curriculum     int
	ManifestFiles  []actionrelationexp.EvidenceFile
	PackFiles      []actionrelationexp.EvidenceFile
	WorldRows      []WorldPolicyRow
	CurriculumRows []CurriculumPolicyRow
}

type PanelSummary struct {
	Panel               string
	Authority           string
	Fixture             actionrelationfixture.PanelFixture
	WorldRows           []WorldPolicyRow
	CurriculumRows      []CurriculumPolicyRow
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
	WorldPolicyRowsRoot string
	CurriculumRowsRoot  string
}

// ExecuteDevelopmentPanel runs the exact public 16-curriculum panel. Evidence
// is delivered one curriculum at a time so callers can retain it without ever
// holding the multi-gigabyte panel in memory.
func ExecuteDevelopmentPanel(domainsDir string, prepare func(actionrelationfixture.PanelFixture) error, consume func(PanelCurriculumEvidence) error) (PanelSummary, error) {
	if prepare == nil || consume == nil {
		return PanelSummary{}, fmt.Errorf("development panel requires preparation and evidence consumers")
	}
	attempts, fixture, err := actionrelationfixture.GenerateDevelopmentPanel()
	if err != nil {
		return PanelSummary{}, err
	}
	if err := prepare(fixture); err != nil {
		return PanelSummary{}, err
	}
	result := PanelSummary{Panel: "development", Authority: "development-public-v1", Fixture: fixture}
	var records []actionrelationexp.RunEvidenceRecord
	for curriculum, generated := range attempts {
		scored, err := ExecuteCurriculum(domainsDir, generated)
		if err != nil {
			return PanelSummary{}, fmt.Errorf("development curriculum %d: %w", curriculum, err)
		}
		evidence, err := BuildCurriculumEvidence(generated, scored)
		if err != nil {
			return PanelSummary{}, fmt.Errorf("development curriculum %d evidence: %w", curriculum, err)
		}
		manifests, err := BuildCurriculumManifests(scored, evidence)
		if err != nil {
			return PanelSummary{}, fmt.Errorf("development curriculum %d manifests: %w", curriculum, err)
		}
		chunk := PanelCurriculumEvidence{
			Curriculum: curriculum, ManifestFiles: manifests.ManifestFiles, PackFiles: manifests.PackFiles,
			WorldRows: slices.Clone(scored.WorldRows), CurriculumRows: slices.Clone(scored.CurriculumRows),
		}
		if err := consume(chunk); err != nil {
			return PanelSummary{}, err
		}
		result.WorldRows = append(result.WorldRows, scored.WorldRows...)
		result.CurriculumRows = append(result.CurriculumRows, scored.CurriculumRows...)
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
	result.RunEvidence, err = actionrelationexp.BuildRunEvidencePack(result.Panel, result.Authority, records)
	if err != nil {
		return PanelSummary{}, err
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(result.Panel)
	result.RunEvidenceManifest = actionrelationexp.EvidenceFile{Path: evidenceRoot + "/manifests/run-evidence-root.json", Mode: "100644", Data: result.RunEvidence.Canonical}
	result.WorldPolicyRowsRoot, err = WorldPolicyRowsRoot(result.WorldRows)
	if err != nil {
		return PanelSummary{}, err
	}
	result.CurriculumRowsRoot, err = CurriculumPolicyRowsRoot(result.CurriculumRows)
	if err != nil {
		return PanelSummary{}, err
	}
	if err := VerifyPanelSummary(result); err != nil {
		return PanelSummary{}, err
	}
	return result, nil
}

func VerifyPanelSummary(value PanelSummary) error {
	wantCurricula := map[string]int{"development": 16, "validation": 24, "locked": 32}[value.Panel]
	if wantCurricula == 0 || actionrelationfixture.VerifyPanelFixture(value.Fixture) != nil || value.Fixture.Panel != value.Panel || value.Fixture.Authority != value.Authority || len(value.WorldRows) != wantCurricula*6*len(Policies) || len(value.CurriculumRows) != wantCurricula*len(Policies) || len(value.ObjectRoots) != wantCurricula*4 || len(value.IndexRoots) != wantCurricula*4 || len(value.JournalRoots) != wantCurricula*44 || len(value.InputRoots) != wantCurricula*44 || len(value.DetailRoots) != wantCurricula*44 || len(value.Tables) != wantCurricula*14 || len(value.StructuralMaps) != wantCurricula || len(value.StoreBoundaries) != wantCurricula*2 || actionrelationexp.VerifyRunEvidencePack(value.RunEvidence) != nil {
		return fmt.Errorf("invalid panel summary cardinality")
	}
	worldRoot, err := WorldPolicyRowsRoot(value.WorldRows)
	if err != nil || worldRoot != value.WorldPolicyRowsRoot {
		return fmt.Errorf("panel world rows root mismatch")
	}
	curriculumRoot, err := CurriculumPolicyRowsRoot(value.CurriculumRows)
	if err != nil || curriculumRoot != value.CurriculumRowsRoot {
		return fmt.Errorf("panel curriculum rows root mismatch")
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(value.Panel)
	if value.RunEvidenceManifest.Path != evidenceRoot+"/manifests/run-evidence-root.json" || value.RunEvidenceManifest.Mode != "100644" || !slices.Equal(value.RunEvidenceManifest.Data, value.RunEvidence.Canonical) {
		return fmt.Errorf("panel run-evidence manifest mismatch")
	}
	return nil
}
