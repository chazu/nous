package actionrelationexp

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationwire"
)

func TestAcquisitionStoreEncodesFrozenHighVolumeTables(t *testing.T) {
	run, err := actionrelationacquire.Execute("../../domains", "table-test")
	if err != nil {
		t.Fatal(err)
	}
	tables, err := BuildAcquisitionTables(run)
	if err != nil {
		t.Fatal(err)
	}
	operationCount := 0
	for _, record := range run.MeterRecords {
		if record.Code == 4 || record.Code == 5 || record.Code == 6 {
			operationCount++
		}
	}
	if operationCount != 130 {
		t.Fatalf("training operation count=%d want 130", operationCount)
	}
	wants := map[uint16]int{101: 13920, 102: 7216, 103: 451, 104: 450, 105: 16, 106: 32, 107: operationCount, 108: 451}
	for kind, count := range wants {
		pack := tables[kind]
		if int(pack.LastOrdinal-pack.FirstOrdinal)+1 != count || len(pack.Bytes) != len(TableHeader)+count*tableRecordSizes[kind] {
			t.Fatalf("kind %d pack count=%d size=%d want count=%d size=%d", kind, int(pack.LastOrdinal-pack.FirstOrdinal)+1, len(pack.Bytes), count, len(TableHeader)+count*tableRecordSizes[kind])
		}
		if err := VerifyTablePack(pack); err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
	}
	viewUnit := run.Store.Get(run.Store.Get(run.Experiment).GetStrings("viewEvidenceUnits")[0])
	viewRecord := tables[106].Bytes[len(TableHeader) : len(TableHeader)+tableRecordSizes[106]]
	viewDigest, _ := hex.DecodeString(viewUnit.GetString("viewDigest"))
	if !bytes.Equal(viewRecord[:32], viewDigest) || viewRecord[224] != 0 || viewRecord[225] < 1 || viewRecord[225] > 3 || viewRecord[226] != 2 || viewRecord[227] != 1 || !bytes.Equal(viewRecord[228:], make([]byte, 512-228)) {
		t.Fatalf("kind 106 first record violates frozen layout")
	}
	firstResult := run.Store.Get(run.Store.Get(run.Experiment).GetStrings("candidateResultUnits")[0])
	guardResultDigests := make([]string, 0, 16)
	for _, name := range firstResult.GetStrings("guardResults") {
		guardResultDigests = append(guardResultDigests, run.Store.Get(name).GetString("objectDigest"))
	}
	wantVectorRoot, _ := actionrelationwire.RootDigest("guard-result-vector", guardResultDigests)
	gotVectorRoot := tables[108].Bytes[len(TableHeader)+32 : len(TableHeader)+64]
	wantVectorBytes, _ := hex.DecodeString(wantVectorRoot)
	if !bytes.Equal(gotVectorRoot, wantVectorBytes) {
		t.Fatal("kind 108 does not use the frozen guard-result-vector root")
	}
}
