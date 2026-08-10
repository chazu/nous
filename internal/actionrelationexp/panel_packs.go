package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/actionrelationwire"
)

const (
	StructuralMapHeader  = "ARSM1\n"
	StructuralMapRowSize = 40
	RunEvidenceHeader    = "ARRV1\n"
	RunEvidenceRowSize   = 240
)

type StructuralAttribution struct {
	Kind   uint16
	Digest string
	RunIDs []string
}

type StructuralOutputMap struct {
	EvidenceRoot string
	Curriculum   int
	RunIDs       []string
	File         *EvidenceFile
	Canonical    []byte
	Digest       string
	RunRoots     map[string]string
}

func BuildStructuralOutputMap(curriculum int, runIDs []string, attributions []StructuralAttribution) (StructuralOutputMap, error) {
	root, _ := EvidenceRoot("development")
	return BuildStructuralOutputMapAt(root, curriculum, runIDs, attributions)
}

func BuildStructuralOutputMapAt(evidenceRoot string, curriculum int, runIDs []string, attributions []StructuralAttribution) (StructuralOutputMap, error) {
	if !validEvidenceRoot(evidenceRoot) || curriculum < 0 || len(runIDs) != 44 || !sortedUniqueRunIDs(runIDs) {
		return StructuralOutputMap{}, fmt.Errorf("invalid structural-map run authority")
	}
	runOrdinal := make(map[string]int, len(runIDs))
	for index, runID := range runIDs {
		runOrdinal[runID] = index
	}
	type key struct {
		kind   uint16
		digest string
	}
	bitmaps := map[key][6]byte{}
	for _, attribution := range attributions {
		if objectKinds[attribution.Kind] == "" || !digestText(attribution.Digest) || len(attribution.RunIDs) == 0 {
			return StructuralOutputMap{}, fmt.Errorf("invalid structural attribution")
		}
		item := key{attribution.Kind, attribution.Digest}
		bitmap := bitmaps[item]
		for _, runID := range attribution.RunIDs {
			ordinal, ok := runOrdinal[runID]
			if !ok {
				return StructuralOutputMap{}, fmt.Errorf("unknown attributed run")
			}
			bitmap[ordinal/8] |= 1 << (7 - (ordinal % 8))
		}
		bitmaps[item] = bitmap
	}
	keys := make([]key, 0, len(bitmaps))
	for item := range bitmaps {
		keys = append(keys, item)
	}
	slices.SortFunc(keys, func(a, b key) int {
		if a.kind != b.kind {
			return int(a.kind) - int(b.kind)
		}
		return bytes.Compare(mustDecodeDigest(a.digest), mustDecodeDigest(b.digest))
	})
	if len(keys) > 4192 {
		return StructuralOutputMap{}, fmt.Errorf("structural map exceeds row cap")
	}
	result := StructuralOutputMap{EvidenceRoot: evidenceRoot, Curriculum: curriculum, RunIDs: slices.Clone(runIDs), RunRoots: map[string]string{}}
	rootRows := make([]any, len(keys))
	var shardRows []any
	if len(keys) > 0 {
		data := make([]byte, len(StructuralMapHeader)+len(keys)*StructuralMapRowSize)
		copy(data, StructuralMapHeader)
		for index, item := range keys {
			row := data[len(StructuralMapHeader)+index*StructuralMapRowSize:][:StructuralMapRowSize]
			binary.BigEndian.PutUint16(row[0:2], item.kind)
			copy(row[2:34], mustDecodeDigest(item.digest))
			bitmap := bitmaps[item]
			if bitmap[5]&0x0f != 0 || allZero(bitmap[:]) {
				return StructuralOutputMap{}, fmt.Errorf("invalid structural bitmap")
			}
			copy(row[34:40], bitmap[:])
			rootRows[index] = []any{item.kind, item.digest, hex.EncodeToString(bitmap[:])}
		}
		path := fmt.Sprintf("%s/packs/curriculum-%04d/structural-output-map.arsm", evidenceRoot, curriculum)
		file := EvidenceFile{Path: path, Mode: "100644", Data: data}
		result.File = &file
		first, last := keys[0], keys[len(keys)-1]
		shardRows = []any{[]any{0, path, first.kind, first.digest, last.kind, last.digest, len(keys), len(data), shaHex(data)}}
	}
	mapRoot, _ := actionrelationwire.RootDigest("structural-output-map", rootRows)
	result.Canonical, _ = json.Marshal([]any{"actionrelation-structural-output-map/v1", curriculum, runIDs, StructuralMapRowSize, len(keys), mapRoot, shardRows})
	result.Digest = shaHex(result.Canonical)
	for ordinal, runID := range runIDs {
		var selected []any
		for _, item := range keys {
			bitmap := bitmaps[item]
			if bitmap[ordinal/8]&(1<<(7-(ordinal%8))) != 0 {
				selected = append(selected, []any{item.kind, item.digest})
			}
		}
		result.RunRoots[runID], _ = actionrelationwire.RootDigest("run-structural-outputs", selected)
	}
	return result, nil
}

func VerifyStructuralOutputMap(value StructuralOutputMap) error {
	rebuiltAttributions, err := decodeStructuralAttributions(value)
	if err != nil {
		return err
	}
	rebuilt, err := BuildStructuralOutputMapAt(value.EvidenceRoot, value.Curriculum, value.RunIDs, rebuiltAttributions)
	if err != nil || rebuilt.Digest != value.Digest || !bytes.Equal(rebuilt.Canonical, value.Canonical) || !equalOptionalFile(rebuilt.File, value.File) || !mapsEqual(rebuilt.RunRoots, value.RunRoots) {
		return fmt.Errorf("structural output map mismatch")
	}
	return nil
}

func decodeStructuralAttributions(value StructuralOutputMap) ([]StructuralAttribution, error) {
	if value.File == nil {
		return nil, nil
	}
	if value.File.Mode != "100644" || !safeEvidencePath(value.File.Path) || len(value.File.Data) < 6 || string(value.File.Data[:6]) != StructuralMapHeader || (len(value.File.Data)-6)%StructuralMapRowSize != 0 {
		return nil, fmt.Errorf("invalid structural map pack")
	}
	var result []StructuralAttribution
	for offset := 6; offset < len(value.File.Data); offset += StructuralMapRowSize {
		row := value.File.Data[offset : offset+StructuralMapRowSize]
		if row[39]&0x0f != 0 || allZero(row[34:40]) {
			return nil, fmt.Errorf("invalid structural map row")
		}
		attribution := StructuralAttribution{Kind: binary.BigEndian.Uint16(row[0:2]), Digest: hex.EncodeToString(row[2:34])}
		for ordinal, runID := range value.RunIDs {
			if row[34+ordinal/8]&(1<<(7-(ordinal%8))) != 0 {
				attribution.RunIDs = append(attribution.RunIDs, runID)
			}
		}
		result = append(result, attribution)
	}
	return result, nil
}

type RunEvidenceRecord struct {
	RunID          string
	JournalRoot    string
	InputRoot      string
	DetailRoot     string
	OperationRoot  string
	ChargedRoot    string
	StructuralRoot string
	WorkTerminal   string
}

type RunEvidencePack struct {
	Panel     string
	Authority string
	Records   []RunEvidenceRecord
	File      EvidenceFile
	Canonical []byte
	Digest    string
}

func BuildRunEvidencePack(panel, authority string, records []RunEvidenceRecord) (RunEvidencePack, error) {
	if !validPanelAuthority(panel, authority) || len(records) != panelRunCounts[panel] {
		return RunEvidencePack{}, fmt.Errorf("invalid run-evidence authority")
	}
	records = slices.Clone(records)
	slices.SortFunc(records, func(a, b RunEvidenceRecord) int { return bytes.Compare(mustRunID(a.RunID), mustRunID(b.RunID)) })
	runIDRows := make([]string, len(records))
	transcriptRows := make([]any, len(records))
	resultRows := make([]any, len(records))
	data := make([]byte, len(RunEvidenceHeader)+len(records)*RunEvidenceRowSize)
	copy(data, RunEvidenceHeader)
	for index, record := range records {
		if !runIDText(record.RunID) || index > 0 && record.RunID == records[index-1].RunID {
			return RunEvidencePack{}, fmt.Errorf("invalid run-evidence run ID")
		}
		digests := []string{record.JournalRoot, record.InputRoot, record.DetailRoot, record.OperationRoot, record.ChargedRoot, record.StructuralRoot}
		for _, digest := range digests {
			if !digestText(digest) {
				return RunEvidencePack{}, fmt.Errorf("invalid run-evidence root")
			}
		}
		if record.WorkTerminal != "" && !digestText(record.WorkTerminal) {
			return RunEvidencePack{}, fmt.Errorf("invalid work terminal")
		}
		row := data[len(RunEvidenceHeader)+index*RunEvidenceRowSize:][:RunEvidenceRowSize]
		copy(row[0:16], mustRunID(record.RunID))
		for digestIndex, digest := range digests {
			copy(row[16+digestIndex*32:48+digestIndex*32], mustDecodeDigest(digest))
		}
		if record.WorkTerminal != "" {
			copy(row[208:240], mustDecodeDigest(record.WorkTerminal))
		}
		runIDRows[index] = record.RunID
		transcriptRows[index] = []any{record.RunID, record.JournalRoot, record.InputRoot, record.DetailRoot, record.OperationRoot}
		resultRows[index] = []any{record.RunID, record.ChargedRoot, record.StructuralRoot, zeroIfEmpty(record.WorkTerminal)}
	}
	runIDsRoot, _ := actionrelationwire.RootDigest("expected-run-ids", runIDRows)
	transcriptRoot, _ := actionrelationwire.RootDigest("transcript-rows", transcriptRows)
	resultRoot, _ := actionrelationwire.RootDigest("result-rows", resultRows)
	evidenceRoot, _ := EvidenceRoot(panel)
	path := evidenceRoot + "/packs/run-evidence-0000.arrv"
	file := EvidenceFile{Path: path, Mode: "100644", Data: data}
	first, last := records[0].RunID, records[len(records)-1].RunID
	canonical, _ := json.Marshal([]any{"actionrelation-run-evidence-root/v1", panel, authority, RunEvidenceRowSize, len(records), runIDsRoot, transcriptRoot, resultRoot, []any{[]any{0, path, first, last, len(records), len(data), shaHex(data)}}})
	return RunEvidencePack{Panel: panel, Authority: authority, Records: records, File: file, Canonical: canonical, Digest: shaHex(canonical)}, nil
}

func VerifyRunEvidencePack(value RunEvidencePack) error {
	rebuilt, err := BuildRunEvidencePack(value.Panel, value.Authority, value.Records)
	if err != nil || rebuilt.Digest != value.Digest || !bytes.Equal(rebuilt.Canonical, value.Canonical) || !equalFile(rebuilt.File, value.File) {
		return fmt.Errorf("run-evidence pack mismatch")
	}
	return nil
}

// ChargedOutputsRoot reconstructs the frozen per-run result commitment from
// the sequence-aligned detail packs. Rows are [sequence,[outputDigest...]].
func ChargedOutputsRoot(transcript TranscriptBundle) (string, error) {
	if err := VerifyTranscript(transcript); err != nil {
		return "", err
	}
	rows := make([]any, transcript.DetailRoot.TotalRecords)
	for shardOrdinal, file := range transcript.DetailFiles {
		shard := transcript.DetailRoot.Shards[shardOrdinal]
		for local := 0; local < shard.RecordCount; local++ {
			sequence := int(shard.FirstSequence) + local
			row := file.Data[len(DetailHeader)+local*DetailRowBytes:][:DetailRowBytes]
			count := int(row[75])
			outputs := make([]string, count)
			for index := range outputs {
				outputs[index] = hex.EncodeToString(row[128+index*32 : 160+index*32])
			}
			rows[sequence] = []any{sequence, outputs}
		}
	}
	return actionrelationwire.RootDigest("run-charged-outputs", rows)
}

var panelNames = map[string]bool{"development": true, "validation": true, "locked": true}
var panelRunCounts = map[string]int{"development": 704, "validation": 1056, "locked": 1408}

func validPanelAuthority(panel, authority string) bool {
	return panel == "development" && authority == "development-public-v1" ||
		panel == "validation" && authority == "validation-public-v1" ||
		panel == "locked" && digestText(authority)
}

func sortedUniqueRunIDs(values []string) bool {
	for index, value := range values {
		if !runIDText(value) || index > 0 && bytes.Compare(mustRunID(values[index-1]), mustRunID(value)) >= 0 {
			return false
		}
	}
	return true
}

func mustRunID(value string) []byte {
	data, _ := hex.DecodeString(value)
	return data
}

func mustDecodeDigest(value string) []byte {
	data, _ := hex.DecodeString(value)
	return data
}

func zeroIfEmpty(value string) string {
	if value == "" {
		return strings.Repeat("0", 64)
	}
	return value
}

func equalOptionalFile(left, right *EvidenceFile) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return equalFile(*left, *right)
}

func equalFile(left, right EvidenceFile) bool {
	return left.Path == right.Path && left.Mode == right.Mode && bytes.Equal(left.Data, right.Data)
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
