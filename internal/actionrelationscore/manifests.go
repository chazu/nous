package actionrelationscore

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
)

type CurriculumManifests struct {
	Curriculum      int
	ManifestFiles   []actionrelationexp.EvidenceFile
	PackFiles       []actionrelationexp.EvidenceFile
	ObjectRoots     []actionrelationexp.ObjectManifestRef
	IndexRoots      []actionrelationexp.ObjectManifestRef
	JournalRoots    []actionrelationexp.TranscriptManifestRef
	InputRoots      []actionrelationexp.TranscriptManifestRef
	DetailRoots     []actionrelationexp.TranscriptManifestRef
	Tables          []actionrelationexp.TableManifestRef
	StructuralMap   actionrelationexp.AuthorityRef
	StoreBoundaries []actionrelationexp.StoreBoundaryRow
}

func BuildCurriculumManifests(scored CurriculumResult, evidence CurriculumEvidence) (CurriculumManifests, error) {
	result := CurriculumManifests{Curriculum: scored.Curriculum}
	if evidence.Curriculum != scored.Curriculum || len(evidence.Transcripts) != 44 || len(evidence.RunEvidence) != 44 || actionrelationexp.VerifyStructuralOutputMap(evidence.StructuralMap) != nil {
		return result, fmt.Errorf("invalid curriculum manifest input")
	}
	evidenceRoot, err := actionrelationexp.EvidenceRoot(scored.Panel)
	if err != nil || evidence.StructuralMap.EvidenceRoot != evidenceRoot {
		return result, fmt.Errorf("curriculum manifest panel root mismatch")
	}
	bundles := []actionrelationexp.ObjectBundle{evidence.NousPreboundary, evidence.NoGuardPreboundary, evidence.Utility, evidence.Authority}
	for _, bundle := range bundles {
		if actionrelationexp.VerifyObjectBundle(bundle) != nil || bundle.Scope.Curriculum != scored.Curriculum {
			return result, fmt.Errorf("invalid curriculum object bundle")
		}
		objectCanonical, _ := bundle.ObjectRoot.CanonicalJSON()
		indexCanonical, _ := bundle.IndexRoot.CanonicalJSON()
		objectPath := fmt.Sprintf("%s/manifests/curriculum-%04d/%s-object-root.json", evidenceRoot, scored.Curriculum, bundle.Scope.Class)
		indexPath := fmt.Sprintf("%s/manifests/curriculum-%04d/%s-index-root.json", evidenceRoot, scored.Curriculum, bundle.Scope.Class)
		result.ManifestFiles = append(result.ManifestFiles,
			actionrelationexp.EvidenceFile{Path: objectPath, Mode: "100644", Data: objectCanonical},
			actionrelationexp.EvidenceFile{Path: indexPath, Mode: "100644", Data: indexCanonical},
		)
		result.PackFiles = append(result.PackFiles, bundle.ObjectFiles...)
		result.PackFiles = append(result.PackFiles, bundle.IndexFiles...)
		result.ObjectRoots = append(result.ObjectRoots, actionrelationexp.ObjectManifestRef{Scope: bundle.Scope, Path: objectPath, Digest: objectDigest(objectCanonical)})
		result.IndexRoots = append(result.IndexRoots, actionrelationexp.ObjectManifestRef{Scope: bundle.Scope, Path: indexPath, Digest: objectDigest(indexCanonical)})
	}
	runIDs := make([]string, 0, len(evidence.Transcripts))
	for runID := range evidence.Transcripts {
		runIDs = append(runIDs, runID)
	}
	slices.Sort(runIDs)
	for _, runID := range runIDs {
		transcript := evidence.Transcripts[runID]
		if actionrelationexp.VerifyTranscript(transcript) != nil || transcript.RunID != runID {
			return result, fmt.Errorf("invalid transcript manifest %s", runID)
		}
		for _, item := range []struct {
			class string
			root  actionrelationexp.TranscriptRoot
			files []actionrelationexp.EvidenceFile
		}{
			{"journal", transcript.JournalRoot, transcript.JournalFiles},
			{"input", transcript.InputRoot, transcript.InputFiles},
			{"detail", transcript.DetailRoot, transcript.DetailFiles},
		} {
			canonical, _ := item.root.CanonicalJSON()
			path := fmt.Sprintf("%s/manifests/runs/%s-%s-root.json", evidenceRoot, runID, item.class)
			result.ManifestFiles = append(result.ManifestFiles, actionrelationexp.EvidenceFile{Path: path, Mode: "100644", Data: canonical})
			result.PackFiles = append(result.PackFiles, item.files...)
			reference := actionrelationexp.TranscriptManifestRef{RunID: runID, Path: path, Digest: objectDigest(canonical)}
			switch item.class {
			case "journal":
				result.JournalRoots = append(result.JournalRoots, reference)
			case "input":
				result.InputRoots = append(result.InputRoots, reference)
			case "detail":
				result.DetailRoots = append(result.DetailRoots, reference)
			}
		}
	}
	for _, acquisition := range []Acquisition{scored.Nous, scored.NoGuard} {
		scope := acquisition.Boundary.Scope
		kinds := []uint16{101, 102, 103, 104, 105, 106, 107, 108}
		if scope == "no-guard" {
			kinds = []uint16{102, 103, 105, 106, 107, 108}
		}
		for _, kind := range kinds {
			bundle, ok := acquisition.Evidence.Tables[kind]
			if !ok || actionrelationexp.VerifyTableBundle(bundle) != nil {
				return result, fmt.Errorf("invalid acquisition table %s/%d", scope, kind)
			}
			canonical, _ := bundle.Manifest.CanonicalJSON()
			path := fmt.Sprintf("%s/manifests/curriculum-%04d/%s-table-%03d.json", evidenceRoot, scored.Curriculum, scope, kind)
			result.ManifestFiles = append(result.ManifestFiles, actionrelationexp.EvidenceFile{Path: path, Mode: "100644", Data: canonical})
			result.PackFiles = append(result.PackFiles, bundle.Files...)
			result.Tables = append(result.Tables, actionrelationexp.TableManifestRef{Curriculum: scored.Curriculum, Scope: scope, Kind: kind, Path: path, Digest: objectDigest(canonical)})
		}
		indexDigest, _ := acquisition.Boundary.Preboundary.IndexRoot.Digest()
		result.StoreBoundaries = append(result.StoreBoundaries, actionrelationexp.StoreBoundaryRow{Curriculum: scored.Curriculum, Scope: scope, BoundaryDigest: acquisition.Boundary.BoundaryDigest, PreboundaryIndexRoot: indexDigest})
	}
	structuralPath := fmt.Sprintf("%s/manifests/curriculum-%04d/structural-output-map.json", evidenceRoot, scored.Curriculum)
	result.StructuralMap, err = actionrelationexp.Reference(structuralPath, evidence.StructuralMap.Canonical)
	if err != nil {
		return result, err
	}
	result.ManifestFiles = append(result.ManifestFiles, actionrelationexp.EvidenceFile{Path: structuralPath, Mode: "100644", Data: evidence.StructuralMap.Canonical})
	if evidence.StructuralMap.File != nil {
		result.PackFiles = append(result.PackFiles, *evidence.StructuralMap.File)
	}
	if err := verifyCurriculumManifestFiles(result); err != nil {
		return CurriculumManifests{}, err
	}
	return result, nil
}

func verifyCurriculumManifestFiles(value CurriculumManifests) error {
	if len(value.ManifestFiles) != 155 || len(value.ObjectRoots) != 4 || len(value.IndexRoots) != 4 || len(value.JournalRoots) != 44 || len(value.InputRoots) != 44 || len(value.DetailRoots) != 44 || len(value.Tables) != 14 || len(value.StoreBoundaries) != 2 || value.StructuralMap.Verify() != nil {
		return fmt.Errorf("curriculum manifest cardinality mismatch")
	}
	seen := map[string]bool{}
	for _, file := range append(slices.Clone(value.ManifestFiles), value.PackFiles...) {
		if file.Mode != "100644" || file.Path == "" || seen[file.Path] {
			return fmt.Errorf("duplicate or invalid curriculum evidence file %q", file.Path)
		}
		seen[file.Path] = true
	}
	return nil
}
