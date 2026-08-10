package actionrelationexp

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

func TestEvidenceBoundAcquisitionClosesBarrierAfterTableManifests(t *testing.T) {
	session, err := actionrelationacquire.BeginFor("../../domains", "evidence-bound", 0, 4, "development", PlanCommit)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := CompleteAcquisition(session, 4)
	if err != nil {
		t.Fatal(err)
	}
	experiment := evidence.Run.Store.Get(evidence.Run.Experiment)
	barrier := evidence.Run.Store.Get(experiment.GetString("guardSearchBarrier"))
	if barrier == nil || evidence.Run.Artifact == "" {
		t.Fatal("evidence-bound acquisition did not freeze its artifact")
	}
	var wire []any
	if json.Unmarshal([]byte(barrier.GetString("canonicalObject")), &wire) != nil || len(wire) != 6 || wire[0] != "action-guard-search-barrier/v1" || wire[5] != "completed" {
		t.Fatalf("barrier wire=%v", wire)
	}
	edgeManifest, _ := evidence.Tables[104].Manifest.CanonicalJSON()
	if wire[2] != shaHex(edgeManifest) {
		t.Fatal("barrier does not name the exact edge-table manifest")
	}
	evaluationRoots := wire[3].([]any)
	for index, kind := range []uint16{101, 102} {
		manifest, _ := evidence.Tables[kind].Manifest.CanonicalJSON()
		if evaluationRoots[index] != shaHex(manifest) {
			t.Fatalf("evaluation root %d does not name kind %d manifest", index, kind)
		}
	}
	candidateLeaves := wire[1].([]any)
	if len(candidateLeaves) != 451 || candidateLeaves[0] != evidence.Tables[103].LeafDigests[0] {
		t.Fatal("barrier candidate authority is not ARTB-103 ordinal order")
	}
	if err := VerifyAcquisitionTranscript(evidence.Transcript, evidence.Run); err != nil {
		t.Fatalf("acquisition transcript: %v", err)
	}
	if len(evidence.Transcript.Transcript.CallIDs) != len(evidence.Run.MeterRecords) || len(evidence.Transcript.Reservations) != len(evidence.Run.MeterRecords) || len(evidence.Transcript.ObservationRoots) != 16 {
		t.Fatal("acquisition transcript does not cover every charged call and observation")
	}
	reservationNames := experiment.GetStrings("reservationUnits")
	for sequence, meter := range evidence.Run.MeterRecords {
		if sequence >= len(reservationNames) {
			t.Fatal("charged call lacks a pre-execution reservation unit")
		}
		reservation := evidence.Run.Store.Get(reservationNames[sequence])
		if reservation == nil || meter.SourceTaskDigest == "" || reservation.GetString("objectDigest") != meter.SourceTaskDigest || evidence.Transcript.Reservations[sequence].Digest != meter.SourceTaskDigest {
			t.Fatalf("charged call %d is not bound to its retained reservation", sequence)
		}
	}
	firstObservation := evidence.Run.Store.Get(experiment.GetStrings("observationUnits")[0])
	firstRecord := evidence.Tables[105].Files[0].Data[len(TableHeader) : len(TableHeader)+tableRecordSizes[105]]
	wantOperationRoot, _ := hex.DecodeString(firstObservation.GetString("operationRoot"))
	if !bytes.Equal(firstRecord[292:324], wantOperationRoot) || firstObservation.GetString("operationRoot") != evidence.Transcript.ObservationRoots[0].Digest {
		t.Fatal("ARTB-105 does not retain the exact post-call operation range")
	}
}

func TestAcquisitionTablesProduceCompletePhysicalManifests(t *testing.T) {
	run, err := actionrelationacquire.Execute("../../domains", "table-bundles")
	if err != nil {
		t.Fatal(err)
	}
	bundles, err := BuildAcquisitionTableBundles(run, 9)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []uint16{101, 102, 103, 104, 105, 106, 107, 108} {
		bundle, ok := bundles[kind]
		if !ok {
			t.Fatalf("missing table kind %d", kind)
		}
		if err := VerifyTableBundle(bundle); err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
		if bundle.Manifest.Curriculum != 9 || bundle.Manifest.Scope != "nous" || len(bundle.LeafDigests) != bundle.Manifest.Count {
			t.Fatalf("kind %d manifest mismatch", kind)
		}
	}
}

func TestNoGuardAcquisitionHasClosedRootOnlyTablesAndTranscript(t *testing.T) {
	session, err := actionrelationacquire.BeginNoGuardFor("../../domains", "no-guard-evidence", 0, 12, "development", PlanCommit)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := CompleteNoGuardAcquisition(session, 12)
	if err != nil {
		t.Fatal(err)
	}
	wants := map[uint16]int{102: 16, 103: 1, 105: 16, 106: 32, 107: 130, 108: 1}
	if len(evidence.Tables) != len(wants) {
		t.Fatalf("tables=%v", evidence.Tables)
	}
	for kind, count := range wants {
		bundle, ok := evidence.Tables[kind]
		if !ok || bundle.Manifest.Scope != "no-guard" || bundle.Manifest.Count != count || VerifyTableBundle(bundle) != nil {
			t.Fatalf("kind %d bundle=%+v", kind, bundle)
		}
	}
	if len(evidence.Transcript.Transcript.CallIDs) != 149 || VerifyAcquisitionTranscript(evidence.Transcript, evidence.Run) != nil {
		t.Fatal("no-guard transcript does not bind T+19 calls")
	}
	boundary, err := BuildAcquisitionBoundary(evidence, 12, "no-guard")
	if err != nil || boundary.Verify(evidence) != nil {
		t.Fatalf("boundary=%v verify=%v", err, boundary.Verify(evidence))
	}
}
