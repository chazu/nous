package actionrelationexp

import (
	"bytes"
	"testing"
)

func TestFixedTablePackFramingAndMerkleVerification(t *testing.T) {
	records := [][]byte{make([]byte, 128), bytes.Repeat([]byte{0x5a}, 128), bytes.Repeat([]byte{0xff}, 128)}
	pack, err := BuildTablePack(101, 7, records)
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
