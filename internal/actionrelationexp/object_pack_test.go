package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func objectWire(kind uint16, marker string) []byte {
	wire, _ := json.Marshal([]any{objectKinds[kind], marker})
	return wire
}

func TestObjectAndIndexPacksUseFrozenFramesAndOffsets(t *testing.T) {
	scope := ObjectScope{Curriculum: 7, Class: "utility"}
	bundle, err := BuildObjectBundle(scope, []ObjectRecord{
		{Kind: 7, Bytes: objectWire(7, "guard")},
		{Kind: 1, Bytes: objectWire(1, "state")},
		{Kind: 3, Bytes: objectWire(3, "occurrence")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.ObjectFiles) != 1 || len(bundle.IndexFiles) != 1 || bundle.ObjectRoot.TotalRecords != 3 || bundle.IndexRoot.TotalRows != 3 {
		t.Fatalf("unexpected bundle shape: %+v", bundle)
	}
	if err := VerifyObjectBundle(bundle); err != nil {
		t.Fatal(err)
	}
	objectPack := bundle.ObjectFiles[0].Data
	indexPack := bundle.IndexFiles[0].Data
	if string(objectPack[:6]) != ObjectHeader || string(indexPack[:6]) != IndexHeader {
		t.Fatal("wrong physical pack header")
	}
	firstRow := indexPack[len(IndexHeader) : len(IndexHeader)+ObjectIndexRowBytes]
	offset := binary.BigEndian.Uint64(firstRow[32:40])
	length := binary.BigEndian.Uint32(firstRow[40:44])
	if offset != 10 || binary.BigEndian.Uint32(objectPack[offset-4:offset]) != length || !bytes.Equal(objectPack[offset:offset+uint64(length)], objectPack[10:10+length]) {
		t.Fatalf("first indexed object offset=%d length=%d", offset, length)
	}
}

func TestObjectBundleRejectsCorruptionAndCrossKindIndex(t *testing.T) {
	bundle, err := BuildObjectBundle(ObjectScope{Curriculum: 0, Class: "authority"}, []ObjectRecord{{Kind: 1, Bytes: objectWire(1, "state")}})
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bundle
	corrupt.ObjectFiles = append([]EvidenceFile(nil), bundle.ObjectFiles...)
	corrupt.ObjectFiles[0].Data = bytes.Clone(bundle.ObjectFiles[0].Data)
	corrupt.ObjectFiles[0].Data[10] ^= 1
	if err := VerifyObjectBundle(corrupt); err == nil {
		t.Fatal("accepted corrupted object payload")
	}

	crossKind := bundle
	crossKind.IndexFiles = append([]EvidenceFile(nil), bundle.IndexFiles...)
	crossKind.IndexFiles[0].Data = bytes.Clone(bundle.IndexFiles[0].Data)
	row := crossKind.IndexFiles[0].Data[len(IndexHeader):]
	binary.BigEndian.PutUint16(row[44:46], 2)
	crossKind.IndexRoot.Shards = append([]IndexShard(nil), bundle.IndexRoot.Shards...)
	crossKind.IndexRoot.Shards[0].PackDigest = shaHex(crossKind.IndexFiles[0].Data)
	if err := VerifyObjectBundle(crossKind); err == nil {
		t.Fatal("accepted a state payload under the semantic-action decoder")
	}
}

func TestObjectPacksSplitGreedilyAtFrozenCap(t *testing.T) {
	records := make([]ObjectRecord, 260)
	for index := range records {
		padding := bytes.Repeat([]byte{'x'}, 64900)
		wire, _ := json.Marshal([]any{objectKinds[1], index, string(padding)})
		records[index] = ObjectRecord{Kind: 1, Bytes: wire}
	}
	bundle, err := BuildObjectBundle(ObjectScope{Curriculum: 12, Class: "utility"}, records)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.ObjectFiles) != 2 || len(bundle.ObjectFiles[0].Data) > MaximumPackBytes || len(bundle.ObjectFiles[1].Data) > MaximumPackBytes {
		t.Fatalf("pack count/sizes=%d %d/%d", len(bundle.ObjectFiles), len(bundle.ObjectFiles[0].Data), len(bundle.ObjectFiles[1].Data))
	}
	nextLength := int(binary.BigEndian.Uint32(bundle.ObjectFiles[1].Data[len(ObjectHeader) : len(ObjectHeader)+4]))
	if len(bundle.ObjectFiles[0].Data)+4+nextLength <= MaximumPackBytes {
		t.Fatal("first pack did not use greedy maximal framing")
	}
	if err := VerifyObjectBundle(bundle); err != nil {
		t.Fatal(err)
	}
}

func TestObjectDecoderRejectsNoncanonicalUnknownAndTrailingJSON(t *testing.T) {
	if err := ValidateObject(1, []byte("[ \"finite-action-state/v1\" ]")); err == nil {
		t.Fatal("accepted noncanonical whitespace")
	}
	if err := ValidateObject(30, []byte("[\"unknown/v1\"]")); err == nil {
		t.Fatal("accepted unknown decoder kind")
	}
	if err := ValidateObject(1, []byte("[\"finite-action-state/v1\"]x")); err == nil {
		t.Fatal("accepted trailing bytes")
	}
}
