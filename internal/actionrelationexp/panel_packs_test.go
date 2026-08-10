package actionrelationexp

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

func fortyFourRunIDs(prefix string) []string {
	result := make([]string, 44)
	for index := range result {
		result[index] = testDigest(fmt.Sprintf("%s-%d", prefix, index))[:32]
	}
	slices.Sort(result)
	return result
}

func TestStructuralOutputMapUsesFrozenRunBitmap(t *testing.T) {
	runIDs := fortyFourRunIDs("structural")
	shared := testDigest("shared-object")
	value, err := BuildStructuralOutputMap(8, runIDs, []StructuralAttribution{
		{Kind: 20, Digest: shared, RunIDs: []string{runIDs[0], runIDs[8], runIDs[43]}},
		{Kind: 21, Digest: testDigest("edge"), RunIDs: []string{runIDs[1]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyStructuralOutputMap(value); err != nil {
		t.Fatal(err)
	}
	if value.File == nil || string(value.File.Data[:6]) != StructuralMapHeader || len(value.File.Data) != 6+2*StructuralMapRowSize {
		t.Fatalf("invalid structural pack shape")
	}
	if value.RunRoots[runIDs[0]] == value.RunRoots[runIDs[2]] || value.RunRoots[runIDs[0]] != value.RunRoots[runIDs[8]] {
		t.Fatal("run-specific bitmap selection did not reconstruct")
	}
	corrupt := value
	file := *value.File
	file.Data = bytes.Clone(file.Data)
	file.Data[len(file.Data)-1] |= 1
	corrupt.File = &file
	if err := VerifyStructuralOutputMap(corrupt); err == nil {
		t.Fatal("accepted nonzero unused bitmap bit")
	}
}

func TestEmptyStructuralMapStillYieldsPerRunEmptyRoots(t *testing.T) {
	runIDs := fortyFourRunIDs("empty-structural")
	value, err := BuildStructuralOutputMap(1, runIDs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if value.File != nil || len(value.RunRoots) != 44 {
		t.Fatal("empty structural map physical/semantic shape mismatch")
	}
	if err := VerifyStructuralOutputMap(value); err != nil {
		t.Fatal(err)
	}
}

func TestDevelopmentRunEvidencePackHasExact240ByteRows(t *testing.T) {
	records := make([]RunEvidenceRecord, panelRunCounts["development"])
	for index := range records {
		records[index] = RunEvidenceRecord{
			RunID: testDigest(fmt.Sprintf("run-%d", index))[:32], JournalRoot: testDigest(fmt.Sprintf("journal-%d", index)),
			InputRoot: testDigest(fmt.Sprintf("input-%d", index)), DetailRoot: testDigest(fmt.Sprintf("detail-%d", index)),
			OperationRoot: testDigest(fmt.Sprintf("operation-%d", index)), ChargedRoot: testDigest(fmt.Sprintf("charged-%d", index)),
			StructuralRoot: testDigest(fmt.Sprintf("structural-%d", index)),
		}
	}
	value, err := BuildRunEvidencePack("development", testDigest("authority"), records)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.File.Data) != len(RunEvidenceHeader)+704*RunEvidenceRowSize || string(value.File.Data[:6]) != RunEvidenceHeader {
		t.Fatal("run-evidence physical shape mismatch")
	}
	if err := VerifyRunEvidencePack(value); err != nil {
		t.Fatal(err)
	}
	corrupt := value
	corrupt.File.Data = bytes.Clone(value.File.Data)
	corrupt.File.Data[6+208] = 1
	if err := VerifyRunEvidencePack(corrupt); err == nil {
		t.Fatal("accepted a forged work-terminal digest")
	}
}
