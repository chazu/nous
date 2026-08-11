package actionrelationscore

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
)

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
