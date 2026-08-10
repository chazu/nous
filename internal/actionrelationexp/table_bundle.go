package actionrelationexp

import (
	"encoding/hex"
	"fmt"
	"slices"
)

type TableBundle struct {
	Manifest    TableManifest
	Files       []EvidenceFile
	LeafDigests []string
}

func BuildTableBundle(curriculum int, scope string, kind uint16, records [][]byte) (TableBundle, error) {
	root, _ := EvidenceRoot("development")
	return BuildTableBundleAt(root, curriculum, scope, kind, records)
}

func BuildTableBundleAt(evidenceRoot string, curriculum int, scope string, kind uint16, records [][]byte) (TableBundle, error) {
	recordSize, ok := tableRecordSizes[kind]
	if !validEvidenceRoot(evidenceRoot) || !ok || curriculum < 0 || scope != "nous" && scope != "no-guard" || len(records) == 0 {
		return TableBundle{}, fmt.Errorf("invalid table bundle identity")
	}
	maxRows := (MaximumPackBytes - len(TableHeader)) / recordSize
	bundle := TableBundle{}
	for first := 0; first < len(records); {
		last := min(len(records), first+maxRows)
		pack, err := BuildTablePack(kind, uint32(first), records[first:last])
		if err != nil {
			return TableBundle{}, err
		}
		ordinal := len(bundle.Files)
		path := fmt.Sprintf("%s/packs/curriculum-%04d/acquisition-%s/table-%03d-%04d.artb", evidenceRoot, curriculum, scope, kind, ordinal)
		bundle.Files = append(bundle.Files, EvidenceFile{Path: path, Mode: "100644", Data: pack.Bytes})
		bundle.Manifest.Shards = append(bundle.Manifest.Shards, TableShard{
			PackOrdinal: ordinal, RelativePath: path, FirstOrdinal: pack.FirstOrdinal, LastOrdinal: pack.LastOrdinal,
			ByteLength: len(pack.Bytes), PackDigest: pack.Digest,
		})
		first = last
	}
	root := tableMerkleRoot(kind, 0, records)
	bundle.Manifest.Curriculum = curriculum
	bundle.Manifest.Scope = scope
	bundle.Manifest.Kind = kind
	bundle.Manifest.RecordSize = recordSize
	bundle.Manifest.Count = len(records)
	bundle.Manifest.MerkleRoot = hex.EncodeToString(root[:])
	if _, err := bundle.Manifest.CanonicalJSON(); err != nil {
		return TableBundle{}, err
	}
	for ordinal, record := range records {
		leaf := TableLeafDigest(kind, uint32(ordinal), record)
		bundle.LeafDigests = append(bundle.LeafDigests, hex.EncodeToString(leaf[:]))
	}
	return bundle, nil
}

func VerifyTableBundle(bundle TableBundle) error {
	if _, err := bundle.Manifest.CanonicalJSON(); err != nil || len(bundle.Files) != len(bundle.Manifest.Shards) {
		return fmt.Errorf("invalid table bundle manifest")
	}
	var records [][]byte
	var leaves []string
	for index, file := range bundle.Files {
		shard := bundle.Manifest.Shards[index]
		if file.Path != shard.RelativePath || file.Mode != "100644" || len(file.Data) != shard.ByteLength || len(file.Data) < len(TableHeader) || string(file.Data[:len(TableHeader)]) != TableHeader || shaHex(file.Data) != shard.PackDigest {
			return fmt.Errorf("invalid table file %d", index)
		}
		count := int(shard.LastOrdinal-shard.FirstOrdinal) + 1
		if len(file.Data) != len(TableHeader)+count*bundle.Manifest.RecordSize {
			return fmt.Errorf("invalid table file length")
		}
		for local := 0; local < count; local++ {
			start := len(TableHeader) + local*bundle.Manifest.RecordSize
			record := file.Data[start : start+bundle.Manifest.RecordSize]
			if err := ValidateTableRecord(bundle.Manifest.Kind, record); err != nil {
				return fmt.Errorf("invalid table row %d: %w", len(records), err)
			}
			records = append(records, record)
			leaf := TableLeafDigest(bundle.Manifest.Kind, uint32(len(records)-1), record)
			leaves = append(leaves, hex.EncodeToString(leaf[:]))
		}
		if index+1 < len(bundle.Files) {
			nextCount := int(bundle.Manifest.Shards[index+1].LastOrdinal-bundle.Manifest.Shards[index+1].FirstOrdinal) + 1
			if nextCount < 1 || len(file.Data)+bundle.Manifest.RecordSize <= MaximumPackBytes {
				return fmt.Errorf("non-greedy table shard")
			}
		}
	}
	if len(records) != bundle.Manifest.Count || !slices.Equal(leaves, bundle.LeafDigests) {
		return fmt.Errorf("table row/leaf coverage mismatch")
	}
	root := tableMerkleRoot(bundle.Manifest.Kind, 0, records)
	if bundle.Manifest.MerkleRoot != hex.EncodeToString(root[:]) {
		return fmt.Errorf("table Merkle root mismatch")
	}
	return nil
}
