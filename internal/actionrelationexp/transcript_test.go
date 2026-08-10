package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

func testDigest(marker string) string { return shaHex([]byte(marker)) }

func TestTranscriptPacksBindEnvelopeJournalAndDetail(t *testing.T) {
	runID := testDigest("run")[:32]
	source := testDigest("reservation")
	guardResults := make([]string, 16)
	for index := range guardResults {
		guardResults[index] = testDigest(fmt.Sprintf("result-%d", index))
	}
	calls := []ChargedCall{
		{Phase: 1, Operation: 1, Status: 1, SourceTaskDigest: source, Payload: []any{"guard-root", testDigest("pattern")}, OutputDigests: []string{testDigest("guard"), testDigest("candidate")}},
		{Phase: 1, Operation: 2, Status: 1, SourceTaskDigest: source, Payload: []any{"candidate-allocate", testDigest("pattern"), testDigest("guard"), testDigest("candidate"), 1}, OutputDigests: []string{testDigest("candidate-1")}},
		{Phase: 1, Operation: 20, Status: 1, SourceTaskDigest: source, Payload: []any{"candidate-result", testDigest("candidate-1"), guardResults, testDigest("views")}, OutputDigests: []string{testDigest("candidate-result")}},
	}
	bundle, err := BuildTranscript(runID, calls)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.JournalFiles) != 1 || len(bundle.InputFiles) != 1 || len(bundle.DetailFiles) != 1 || len(bundle.CallIDs) != len(calls) {
		t.Fatalf("unexpected transcript shape: %+v", bundle)
	}
	if err := VerifyTranscript(bundle); err != nil {
		t.Fatal(err)
	}
	journal := bundle.JournalFiles[0].Data[6:]
	if journal[0] != 1 || journal[1] != 1 || journal[2] != 1 || journal[3] != 1 || journal[4] != 1 || binary.BigEndian.Uint32(journal[8:12]) != 0 || !allZero(journal[28:60]) {
		t.Fatal("first journal record violates frozen layout")
	}
	second := journal[JournalRowBytes : 2*JournalRowBytes]
	firstCallRaw := sha256.Sum256(journal[:JournalRowBytes])
	if !bytes.Equal(second[28:60], firstCallRaw[:]) {
		t.Fatal("journal chain does not name prior call")
	}
	detail := bundle.DetailFiles[0].Data[6:]
	if binary.BigEndian.Uint16(detail[32:34]) != 1 || detail[34] != 1 || detail[35] != 1 || detail[74] != 1 || detail[75] != 2 || !allZero(detail[76:96]) {
		t.Fatal("first detail record violates frozen layout")
	}
}

func TestTranscriptRejectsCorruptChainEnvelopeAndUnusedSlot(t *testing.T) {
	runID := testDigest("run-corrupt")[:32]
	call := ChargedCall{Phase: 2, Operation: 18, Status: 3, SourceTaskDigest: testDigest("reservation"), Payload: []any{"certificate-cache-lookup", testDigest("world"), "complete", testDigest("state"), testDigest("a"), testDigest("b")}, OutputDigests: []string{testDigest("cache")}}
	bundle, err := BuildTranscript(runID, []ChargedCall{call})
	if err != nil {
		t.Fatal(err)
	}
	journalCorrupt := cloneTranscript(bundle)
	journalCorrupt.JournalFiles[0].Data[6+124] = 1
	journalCorrupt.JournalRoot.Shards[0].PackDigest = shaHex(journalCorrupt.JournalFiles[0].Data)
	if err := VerifyTranscript(journalCorrupt); err == nil {
		t.Fatal("accepted nonzero journal padding")
	}
	inputCorrupt := cloneTranscript(bundle)
	inputCorrupt.InputFiles[0].Data[10] ^= 1
	inputCorrupt.InputRoot.Shards[0].PackDigest = shaHex(inputCorrupt.InputFiles[0].Data)
	if err := VerifyTranscript(inputCorrupt); err == nil {
		t.Fatal("accepted envelope bytes inconsistent with detail")
	}
	detailCorrupt := cloneTranscript(bundle)
	detailCorrupt.DetailFiles[0].Data[6+160] = 1
	detailCorrupt.DetailRoot.Shards[0].PackDigest = shaHex(detailCorrupt.DetailFiles[0].Data)
	if err := VerifyTranscript(detailCorrupt); err == nil {
		t.Fatal("accepted nonzero unused output slot")
	}
}

func TestTranscriptRejectsWrongPhaseCounterAndCacheStatus(t *testing.T) {
	source := testDigest("reservation")
	runID := testDigest("bad-runs")[:32]
	for name, call := range map[string]ChargedCall{
		"acquisition-operation-in-utility": {Phase: 2, Operation: 1, Status: 1, SourceTaskDigest: source, Payload: []any{"guard-root", testDigest("pattern")}, OutputDigests: []string{testDigest("a"), testDigest("b")}},
		"cache-hit-on-apply":               {Phase: 2, Operation: 11, Status: 3, SourceTaskDigest: source, Payload: []any{"utility-apply", testDigest("state"), testDigest("occurrence"), testDigest("applicability")}, OutputDigests: []string{testDigest("a")}},
	} {
		if _, err := BuildTranscript(runID, []ChargedCall{call}); err == nil {
			t.Fatalf("%s: accepted invalid call", name)
		}
	}
}

func cloneTranscript(bundle TranscriptBundle) TranscriptBundle {
	result := bundle
	result.JournalFiles = append([]EvidenceFile(nil), bundle.JournalFiles...)
	result.InputFiles = append([]EvidenceFile(nil), bundle.InputFiles...)
	result.DetailFiles = append([]EvidenceFile(nil), bundle.DetailFiles...)
	for index := range result.JournalFiles {
		result.JournalFiles[index].Data = bytes.Clone(result.JournalFiles[index].Data)
		result.InputFiles[index].Data = bytes.Clone(result.InputFiles[index].Data)
		result.DetailFiles[index].Data = bytes.Clone(result.DetailFiles[index].Data)
	}
	result.JournalRoot.Shards = append([]TranscriptShard(nil), bundle.JournalRoot.Shards...)
	result.InputRoot.Shards = append([]TranscriptShard(nil), bundle.InputRoot.Shards...)
	result.DetailRoot.Shards = append([]TranscriptShard(nil), bundle.DetailRoot.Shards...)
	return result
}
