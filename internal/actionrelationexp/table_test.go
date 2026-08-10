package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestFixedTablePackFramingAndMerkleVerification(t *testing.T) {
	records := make([][]byte, 3)
	for ordinal := range records {
		records[ordinal] = make([]byte, 128)
		copy(records[ordinal][0:32], bytes.Repeat([]byte{byte(ordinal + 1)}, 32))
		copy(records[ordinal][64:96], bytes.Repeat([]byte{byte(ordinal + 4)}, 32))
		binary.BigEndian.PutUint16(records[ordinal][96:98], uint16(ordinal))
		records[ordinal][99] = 1
		if ordinal > 0 {
			copy(records[ordinal][32:64], bytes.Repeat([]byte{byte(ordinal + 7)}, 32))
		}
	}
	pack, err := BuildTablePack(103, 7, records)
	if err != nil {
		t.Fatal(err)
	}
	if string(pack.Bytes[:6]) != "ARTB1\n" || len(pack.Bytes) != 6+3*128 || pack.FirstOrdinal != 7 || pack.LastOrdinal != 9 {
		t.Fatalf("pack=%+v", pack)
	}
	if err := VerifyTablePack(pack); err != nil {
		t.Fatal(err)
	}
	corrupt := pack
	corrupt.Bytes = bytes.Clone(pack.Bytes)
	corrupt.Bytes[6+128] ^= 1
	if err := VerifyTablePack(corrupt); err == nil {
		t.Fatal("accepted corrupted table record")
	}
}

func TestTableManifestRequiresContiguousShardCoverage(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	manifest := TableManifest{Curriculum: 0, Scope: "nous", Kind: 101, RecordSize: 128, Count: 3, MerkleRoot: digest, Shards: []TableShard{
		{PackOrdinal: 0, RelativePath: "a.artb", FirstOrdinal: 0, LastOrdinal: 1, ByteLength: 6 + 2*128, PackDigest: digest},
		{PackOrdinal: 1, RelativePath: "b.artb", FirstOrdinal: 2, LastOrdinal: 2, ByteLength: 6 + 128, PackDigest: digest},
	}}
	if _, err := manifest.CanonicalJSON(); err != nil {
		t.Fatal(err)
	}
	manifest.Shards[1].FirstOrdinal = 3
	if _, err := manifest.CanonicalJSON(); err == nil {
		t.Fatal("accepted shard gap")
	}
}

func TestTableVerificationRejectsSemanticallyInvalidRehashedRow(t *testing.T) {
	row := make([]byte, 96)
	copy(row[0:32], bytes.Repeat([]byte{1}, 32))
	copy(row[32:64], bytes.Repeat([]byte{2}, 32))
	row[64], row[65] = 1, 1
	pack, err := BuildTablePack(102, 0, [][]byte{row})
	if err != nil {
		t.Fatal(err)
	}
	pack.Bytes[6+66] = 1
	pack.Digest = shaHex(pack.Bytes)
	root := tableMerkleRoot(102, 0, [][]byte{pack.Bytes[6:]})
	pack.MerkleRoot = hex.EncodeToString(root[:])
	if err := VerifyTablePack(pack); err == nil {
		t.Fatal("accepted nonzero reserved bytes after recomputing physical roots")
	}
}
