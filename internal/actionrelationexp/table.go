package actionrelationexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const TableHeader = "ARTB1\n"

var tableRecordSizes = map[uint16]int{
	101: 128,
	102: 96,
	103: 128,
	104: 96,
	105: 512,
	106: 512,
	107: 256,
	108: 128,
}

type TablePack struct {
	Kind         uint16
	FirstOrdinal uint32
	LastOrdinal  uint32
	Bytes        []byte
	Digest       string
	MerkleRoot   string
}

func BuildTablePack(kind uint16, firstOrdinal uint32, records [][]byte) (TablePack, error) {
	recordSize, ok := tableRecordSizes[kind]
	if !ok || len(records) == 0 {
		return TablePack{}, fmt.Errorf("invalid table kind/count")
	}
	bytes := make([]byte, len(TableHeader)+len(records)*recordSize)
	copy(bytes, TableHeader)
	for index, record := range records {
		if len(record) != recordSize {
			return TablePack{}, fmt.Errorf("kind %d record %d size %d want %d", kind, index, len(record), recordSize)
		}
		copy(bytes[len(TableHeader)+index*recordSize:], record)
	}
	digest := sha256.Sum256(bytes)
	root := tableMerkleRoot(kind, firstOrdinal, records)
	return TablePack{Kind: kind, FirstOrdinal: firstOrdinal, LastOrdinal: firstOrdinal + uint32(len(records)) - 1, Bytes: bytes, Digest: hex.EncodeToString(digest[:]), MerkleRoot: hex.EncodeToString(root[:])}, nil
}

func VerifyTablePack(pack TablePack) error {
	recordSize, ok := tableRecordSizes[pack.Kind]
	if !ok || pack.LastOrdinal < pack.FirstOrdinal || len(pack.Bytes) < len(TableHeader) || string(pack.Bytes[:len(TableHeader)]) != TableHeader {
		return fmt.Errorf("invalid table pack shape")
	}
	count := int(pack.LastOrdinal-pack.FirstOrdinal) + 1
	if len(pack.Bytes) != len(TableHeader)+count*recordSize {
		return fmt.Errorf("invalid table pack length")
	}
	records := make([][]byte, count)
	for index := range records {
		start := len(TableHeader) + index*recordSize
		records[index] = pack.Bytes[start : start+recordSize]
	}
	digest := sha256.Sum256(pack.Bytes)
	root := tableMerkleRoot(pack.Kind, pack.FirstOrdinal, records)
	if hex.EncodeToString(digest[:]) != pack.Digest || hex.EncodeToString(root[:]) != pack.MerkleRoot {
		return fmt.Errorf("table pack digest mismatch")
	}
	return nil
}

func tableMerkleRoot(kind uint16, firstOrdinal uint32, records [][]byte) [32]byte {
	level := make([][32]byte, len(records))
	for index, record := range records {
		preimage := make([]byte, 0, 12+len(record))
		preimage = append(preimage, []byte("ARTB1-LEAF\x00")...)
		var kindBytes [2]byte
		var ordinalBytes [4]byte
		binary.BigEndian.PutUint16(kindBytes[:], kind)
		binary.BigEndian.PutUint32(ordinalBytes[:], firstOrdinal+uint32(index))
		preimage = append(preimage, kindBytes[:]...)
		preimage = append(preimage, ordinalBytes[:]...)
		preimage = append(preimage, record...)
		level[index] = sha256.Sum256(preimage)
	}
	for len(level) > 1 {
		next := make([][32]byte, (len(level)+1)/2)
		for index := range next {
			left := level[index*2]
			right := left
			if index*2+1 < len(level) {
				right = level[index*2+1]
			}
			preimage := append([]byte("ARTB1-NODE\x00"), left[:]...)
			preimage = append(preimage, right[:]...)
			next[index] = sha256.Sum256(preimage)
		}
		level = next
	}
	return level[0]
}

type TableShard struct {
	PackOrdinal  int    `json:"pack_ordinal"`
	RelativePath string `json:"relative_path"`
	FirstOrdinal uint32 `json:"first_ordinal"`
	LastOrdinal  uint32 `json:"last_ordinal"`
	ByteLength   int    `json:"byte_length"`
	PackDigest   string `json:"pack_digest"`
}

type TableManifest struct {
	Curriculum int
	Scope      string
	Kind       uint16
	RecordSize int
	Count      int
	Shards     []TableShard
	MerkleRoot string
}

func (m TableManifest) CanonicalJSON() ([]byte, error) {
	if m.Curriculum < 0 || m.Scope != "nous" && m.Scope != "no-guard" || tableRecordSizes[m.Kind] != m.RecordSize || m.Count < 1 || len(m.Shards) < 1 || !digestText(m.MerkleRoot) {
		return nil, fmt.Errorf("invalid table manifest")
	}
	rows := make([]any, len(m.Shards))
	expected := uint32(0)
	for index, shard := range m.Shards {
		if shard.PackOrdinal != index || shard.FirstOrdinal != expected || shard.LastOrdinal < shard.FirstOrdinal || shard.ByteLength != len(TableHeader)+(int(shard.LastOrdinal-shard.FirstOrdinal)+1)*m.RecordSize || !digestText(shard.PackDigest) {
			return nil, fmt.Errorf("invalid table shard %d", index)
		}
		rows[index] = []any{shard.PackOrdinal, shard.RelativePath, shard.FirstOrdinal, shard.LastOrdinal, shard.ByteLength, shard.PackDigest}
		expected = shard.LastOrdinal + 1
	}
	if int(expected) != m.Count {
		return nil, fmt.Errorf("table shard coverage mismatch")
	}
	return json.Marshal([]any{"actionrelation-table-manifest/v3", m.Curriculum, m.Scope, m.Kind, m.RecordSize, m.Count, rows, m.MerkleRoot})
}

func digestText(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
