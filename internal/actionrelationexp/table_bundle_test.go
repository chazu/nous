package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestTableBundleRetainsLeafIdentityAndManifest(t *testing.T) {
	records := make([][]byte, 3)
	for ordinal := range records {
		row := make([]byte, 128)
		copy(row[0:32], bytes.Repeat([]byte{byte(ordinal + 1)}, 32))
		copy(row[64:96], bytes.Repeat([]byte{byte(ordinal + 4)}, 32))
		binary.BigEndian.PutUint16(row[96:98], uint16(ordinal))
		row[99] = 1
		if ordinal > 0 {
			copy(row[32:64], bytes.Repeat([]byte{byte(ordinal + 7)}, 32))
		}
		records[ordinal] = row
	}
	bundle, err := BuildTableBundle(3, "nous", 103, records)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTableBundle(bundle); err != nil {
		t.Fatal(err)
	}
	preimage := append([]byte("ARTB1-LEAF\x00\x00g\x00\x00\x00\x01"), records[1]...)
	want := sha256.Sum256(preimage)
	if bundle.LeafDigests[1] != hex.EncodeToString(want[:]) {
		t.Fatalf("leaf=%s want %x", bundle.LeafDigests[1], want)
	}
	manifestBytes, err := bundle.Manifest.CanonicalJSON()
	if err != nil || !bytes.HasPrefix(manifestBytes, []byte(`["actionrelation-table-manifest/v3",3,"nous",103,128,3,`)) {
		t.Fatalf("manifest=%s err=%v", manifestBytes, err)
	}
}

func TestTableBundleRejectsRehashedRowOutsideManifestMerkle(t *testing.T) {
	row := make([]byte, 96)
	copy(row[0:32], bytes.Repeat([]byte{1}, 32))
	copy(row[32:64], bytes.Repeat([]byte{2}, 32))
	row[64], row[65] = 1, 1
	bundle, err := BuildTableBundle(0, "no-guard", 102, [][]byte{row})
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bundle
	corrupt.Files = append([]EvidenceFile(nil), bundle.Files...)
	corrupt.Files[0].Data = bytes.Clone(bundle.Files[0].Data)
	corrupt.Files[0].Data[6+64] = 0
	corrupt.Manifest.Shards = append([]TableShard(nil), bundle.Manifest.Shards...)
	corrupt.Manifest.Shards[0].PackDigest = shaHex(corrupt.Files[0].Data)
	leaf := TableLeafDigest(102, 0, corrupt.Files[0].Data[6:])
	corrupt.LeafDigests = []string{hex.EncodeToString(leaf[:])}
	if err := VerifyTableBundle(corrupt); err == nil {
		t.Fatal("accepted rehashed row that disagrees with the committed table Merkle root")
	}
}
