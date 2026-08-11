package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// CurriculumReplay is the complete logical evidence for one curriculum after
// bundle construction and before panel-wide physical retention.
type CurriculumReplay struct {
	Panel         string
	Authority     string
	Curriculum    int
	Objects       []ObjectBundle
	Tables        []TableBundle
	StructuralMap StructuralOutputMap
	RunEvidence   []RunEvidenceRecord
	Transcripts   map[string]TranscriptBundle
}

// VerifyCurriculumReplay exercises the same semantic/work replay used after
// physical packs are reopened while retaining the panel verifier's fixed
// all-curriculum cardinality requirement.
func VerifyCurriculumReplay(value CurriculumReplay) error {
	if !validPanelAuthority(value.Panel, value.Authority) || value.Curriculum < 0 || len(value.Objects) != 4 || len(value.Tables) != 14 || len(value.RunEvidence) != 44 || len(value.Transcripts) != 44 || VerifyStructuralOutputMap(value.StructuralMap) != nil || value.StructuralMap.Curriculum != value.Curriculum {
		return fmt.Errorf("invalid curriculum replay authority")
	}
	objects := map[string]retainedObjectValue{}
	scopes := map[string]map[string]retainedObjectValue{}
	for _, bundle := range value.Objects {
		if bundle.Scope.Curriculum != value.Curriculum || VerifyObjectBundle(bundle) != nil {
			return fmt.Errorf("invalid curriculum replay object bundle")
		}
		payloads := map[string][]byte{}
		for _, file := range bundle.ObjectFiles {
			for offset := len(ObjectHeader); offset < len(file.Data); {
				length := int(binary.BigEndian.Uint32(file.Data[offset : offset+4]))
				offset += 4
				canonical := file.Data[offset : offset+length]
				payloads[shaHex(canonical)] = bytes.Clone(canonical)
				offset += length
			}
		}
		for _, file := range bundle.IndexFiles {
			for offset := len(IndexHeader); offset < len(file.Data); offset += ObjectIndexRowBytes {
				row := file.Data[offset:][:ObjectIndexRowBytes]
				digest := hex.EncodeToString(row[:32])
				kind := binary.BigEndian.Uint16(row[44:46])
				prior := objects[digest]
				if prior.canonical != nil && (prior.kind != kind || !bytes.Equal(prior.canonical, payloads[digest])) {
					return fmt.Errorf("curriculum replay object collision")
				}
				if prior.classes == nil {
					prior = retainedObjectValue{kind: kind, canonical: payloads[digest], classes: map[string]bool{}}
				}
				prior.classes[bundle.Scope.Class] = true
				objects[digest] = prior
				if scopes[bundle.Scope.Class] == nil {
					scopes[bundle.Scope.Class] = map[string]retainedObjectValue{}
				}
				scopes[bundle.Scope.Class][digest] = retainedObjectValue{kind: kind, canonical: payloads[digest]}
			}
		}
	}
	tables := map[string][]retainedTableLeaf{}
	for _, bundle := range value.Tables {
		if bundle.Manifest.Curriculum != value.Curriculum || VerifyTableBundle(bundle) != nil {
			return fmt.Errorf("invalid curriculum replay table bundle")
		}
		manifestCanonical, _ := bundle.Manifest.CanonicalJSON()
		manifestDigest := shaHex(manifestCanonical)
		for shardOrdinal, file := range bundle.Files {
			shard := bundle.Manifest.Shards[shardOrdinal]
			for ordinal := shard.FirstOrdinal; ordinal <= shard.LastOrdinal; ordinal++ {
				local := int(ordinal - shard.FirstOrdinal)
				start := len(TableHeader) + local*bundle.Manifest.RecordSize
				record := file.Data[start : start+bundle.Manifest.RecordSize]
				digest := TableLeafDigest(bundle.Manifest.Kind, ordinal, record)
				text := hex.EncodeToString(digest[:])
				tables[text] = append(tables[text], retainedTableLeaf{kind: bundle.Manifest.Kind, curriculum: value.Curriculum, scope: bundle.Manifest.Scope, manifest: manifestDigest, ordinal: ordinal, record: bytes.Clone(record)})
			}
		}
	}
	authority := retainedCurriculumAuthority{}
	if err := verifyCurriculumAuthorityObjects(value.Panel, value.Authority, value.Curriculum, objects, &authority); err != nil {
		return err
	}
	attributions, err := decodeStructuralAttributions(value.StructuralMap)
	if err != nil {
		return err
	}
	structural := map[string]map[string]bool{}
	for _, attribution := range attributions {
		key := fmt.Sprintf("%d:%s", attribution.Kind, attribution.Digest)
		for _, runID := range attribution.RunIDs {
			if structural[runID] == nil {
				structural[runID] = map[string]bool{}
			}
			structural[runID][key] = true
		}
	}
	seen := map[string]bool{}
	for _, record := range value.RunEvidence {
		expected, ok := authority.runs[record.RunID]
		transcript, transcriptOK := value.Transcripts[record.RunID]
		operationObject, operationOK := objects[record.OperationRoot]
		if !ok || !transcriptOK || !operationOK || operationObject.kind != 46 || record.OperationRoot != expected.operationRoot || !structural[record.RunID][fmt.Sprintf("46:%s", record.OperationRoot)] || VerifyTranscript(transcript) != nil {
			return fmt.Errorf("curriculum replay run identity changed")
		}
		operation := OperationRoot{Canonical: operationObject.canonical, Digest: record.OperationRoot}
		if VerifyOperationRange(operation, transcript) != nil {
			return fmt.Errorf("curriculum replay operation range changed")
		}
		wantTerminal := expected.workTerminal
		if wantTerminal == zeroObjectDigest {
			wantTerminal = ""
		}
		if record.WorkTerminal != wantTerminal {
			return fmt.Errorf("curriculum replay work terminal changed")
		}
		calls, err := decodeRetainedCalls(transcript)
		if err != nil {
			return err
		}
		runObjects, _, err := retainedObjectsForRun(expected, scopes)
		if err != nil {
			return fmt.Errorf("curriculum replay run %s scope: %w", record.RunID, err)
		}
		if err := verifyRetainedRunReplay(record, expected, calls, runObjects, tables, structural[record.RunID]); err != nil {
			return fmt.Errorf("curriculum replay run %s policy=%s world=%d: %w", record.RunID, expected.policy, expected.worldOrdinal, err)
		}
		seen[record.RunID] = true
	}
	if len(seen) != 44 || len(authority.runs) != 44 {
		return fmt.Errorf("curriculum replay run coverage mismatch")
	}
	return nil
}
