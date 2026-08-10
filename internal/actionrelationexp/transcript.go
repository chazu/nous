package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

const (
	JournalHeader   = "ARJR1\n"
	InputHeader     = "ARIN1\n"
	DetailHeader    = "ARCD1\n"
	JournalRowBytes = 128
	DetailRowBytes  = 192
)

var operationCounters = map[uint8]uint8{
	1: 1, 2: 1, 3: 2, 4: 3, 5: 4, 6: 4, 7: 5, 8: 6, 9: 7, 10: 7,
	11: 8, 12: 9, 13: 10, 14: 10, 15: 10, 16: 11, 17: 11, 18: 11,
	19: 12, 20: 5, 21: 10, 22: 5, 23: 10, 24: 10, 25: 11,
}

type ChargedCall struct {
	Phase            uint8
	Operation        uint8
	Status           uint8
	SourceTaskDigest string
	Payload          any
	OutputDigests    []string
}

type TranscriptShard struct {
	PackOrdinal   int
	Path          string
	FirstSequence uint32
	LastSequence  uint32
	RecordCount   int
	ByteLength    int
	PackDigest    string
}

type TranscriptRoot struct {
	Class        string
	RunID        string
	RecordSize   int
	TotalRecords int
	Shards       []TranscriptShard
}

func (r TranscriptRoot) CanonicalJSON() ([]byte, error) {
	wantSize := map[string]int{"journal": JournalRowBytes, "input": 0, "detail": DetailRowBytes}[r.Class]
	if wantSize != r.RecordSize || !runIDText(r.RunID) || r.TotalRecords < 1 || len(r.Shards) < 1 {
		return nil, fmt.Errorf("invalid transcript root")
	}
	rows := make([]any, len(r.Shards))
	expected := uint32(0)
	total := 0
	for index, shard := range r.Shards {
		if shard.PackOrdinal != index || !safeEvidencePath(shard.Path) || shard.FirstSequence != expected || shard.LastSequence < shard.FirstSequence || shard.RecordCount != int(shard.LastSequence-shard.FirstSequence)+1 || shard.RecordCount < 1 || shard.ByteLength < 6 || shard.ByteLength > MaximumPackBytes || !digestText(shard.PackDigest) {
			return nil, fmt.Errorf("invalid transcript shard %d", index)
		}
		if r.RecordSize != 0 && shard.ByteLength != 6+shard.RecordCount*r.RecordSize {
			return nil, fmt.Errorf("invalid fixed transcript shard length")
		}
		rows[index] = []any{shard.PackOrdinal, shard.Path, shard.FirstSequence, shard.LastSequence, shard.RecordCount, shard.ByteLength, shard.PackDigest}
		expected = shard.LastSequence + 1
		total += shard.RecordCount
	}
	if total != r.TotalRecords {
		return nil, fmt.Errorf("transcript root count mismatch")
	}
	return json.Marshal([]any{"actionrelation-pack-root/v1", r.Class, r.RunID, r.RecordSize, r.TotalRecords, rows})
}

func (r TranscriptRoot) Digest() (string, error) { return canonicalDigest(r.CanonicalJSON()) }

type TranscriptBundle struct {
	RunID           string
	JournalFiles    []EvidenceFile
	InputFiles      []EvidenceFile
	DetailFiles     []EvidenceFile
	JournalRoot     TranscriptRoot
	InputRoot       TranscriptRoot
	DetailRoot      TranscriptRoot
	CallIDs         []string
	EnvelopeDigests []string
}

type builtCall struct {
	envelope       []byte
	envelopeDigest [32]byte
	outputDigests  [][32]byte
	journal        [JournalRowBytes]byte
	detail         [DetailRowBytes]byte
	callID         [32]byte
}

func BuildTranscript(runID string, calls []ChargedCall) (TranscriptBundle, error) {
	if !runIDText(runID) || len(calls) == 0 {
		return TranscriptBundle{}, fmt.Errorf("invalid transcript identity/count")
	}
	runRaw, _ := hex.DecodeString(runID)
	built := make([]builtCall, len(calls))
	var previous [32]byte
	for sequence, call := range calls {
		counter, ok := operationCounters[call.Operation]
		if !ok || call.Phase < 1 || call.Phase > 2 || call.Status < 1 || call.Status > 3 || call.Status == 3 && call.Operation != 16 && call.Operation != 18 || !digestText(call.SourceTaskDigest) || len(call.OutputDigests) > 2 {
			return TranscriptBundle{}, fmt.Errorf("invalid charged call %d", sequence)
		}
		if call.Phase != calls[0].Phase || !phaseAllows(call.Phase, call.Operation) || !outputCountAllowed(call.Operation, call.Status, len(call.OutputDigests)) {
			return TranscriptBundle{}, fmt.Errorf("invalid call phase/output %d", sequence)
		}
		envelope, err := json.Marshal([]any{"action-charged-input/v1", call.Phase, call.Operation, call.SourceTaskDigest, call.Payload})
		if err != nil || len(envelope) > envelopeCap(call.Operation) {
			return TranscriptBundle{}, fmt.Errorf("invalid envelope %d", sequence)
		}
		if _, err := verifyEnvelope(envelope, call.Phase, call.Operation); err != nil {
			return TranscriptBundle{}, fmt.Errorf("invalid envelope %d: %w", sequence, err)
		}
		built[sequence].envelope = envelope
		built[sequence].envelopeDigest = sha256.Sum256(envelope)
		outputHex := make([]string, len(call.OutputDigests))
		for index, digest := range call.OutputDigests {
			raw, err := hex.DecodeString(digest)
			if err != nil || len(raw) != 32 {
				return TranscriptBundle{}, fmt.Errorf("invalid output digest %d/%d", sequence, index)
			}
			var value [32]byte
			copy(value[:], raw)
			built[sequence].outputDigests = append(built[sequence].outputDigests, value)
			outputHex[index] = digest
		}
		inputRoot := vectorDigest("actionrelation-input-vector/v1", []string{hex.EncodeToString(built[sequence].envelopeDigest[:])})
		outputRoot := vectorDigest("actionrelation-output-vector/v1", outputHex)
		journal := built[sequence].journal[:]
		journal[0], journal[1], journal[2], journal[3], journal[4] = 1, call.Phase, call.Operation, call.Status, counter
		binary.BigEndian.PutUint32(journal[8:12], uint32(sequence))
		copy(journal[12:28], runRaw)
		copy(journal[28:60], previous[:])
		copy(journal[60:92], inputRoot[:])
		copy(journal[92:124], outputRoot[:])
		built[sequence].callID = sha256.Sum256(journal)
		previous = built[sequence].callID

		detail := built[sequence].detail[:]
		copy(detail[:32], built[sequence].callID[:])
		binary.BigEndian.PutUint16(detail[32:34], uint16(call.Operation))
		detail[34], detail[35], detail[36], detail[37] = call.Phase, call.Operation, call.Status, counter
		binary.BigEndian.PutUint32(detail[38:42], uint32(sequence))
		sourceRaw, _ := hex.DecodeString(call.SourceTaskDigest)
		copy(detail[42:74], sourceRaw)
		detail[74], detail[75] = 1, byte(len(call.OutputDigests))
		copy(detail[96:128], built[sequence].envelopeDigest[:])
		for index, output := range built[sequence].outputDigests {
			copy(detail[128+index*32:160+index*32], output[:])
		}
	}
	return packTranscript(runID, built)
}

func packTranscript(runID string, calls []builtCall) (TranscriptBundle, error) {
	bundle := TranscriptBundle{RunID: runID}
	var journalShards, inputShards, detailShards []TranscriptShard
	for first := 0; first < len(calls); {
		last := first
		journalSize, inputSize, detailSize := len(JournalHeader), len(InputHeader), len(DetailHeader)
		for last < len(calls) {
			nextJournal := journalSize + JournalRowBytes
			nextInput := inputSize + 4 + len(calls[last].envelope)
			nextDetail := detailSize + DetailRowBytes
			if nextJournal > MaximumPackBytes || nextInput > MaximumPackBytes || nextDetail > MaximumPackBytes {
				break
			}
			journalSize, inputSize, detailSize = nextJournal, nextInput, nextDetail
			last++
		}
		if last == first {
			return TranscriptBundle{}, fmt.Errorf("aligned transcript frame exceeds cap")
		}
		ordinal := len(bundle.JournalFiles)
		journal := make([]byte, journalSize)
		input := make([]byte, inputSize)
		detail := make([]byte, detailSize)
		copy(journal, JournalHeader)
		copy(input, InputHeader)
		copy(detail, DetailHeader)
		journalOffset, inputOffset, detailOffset := 6, 6, 6
		for index := first; index < last; index++ {
			copy(journal[journalOffset:], calls[index].journal[:])
			journalOffset += JournalRowBytes
			binary.BigEndian.PutUint32(input[inputOffset:inputOffset+4], uint32(len(calls[index].envelope)))
			inputOffset += 4
			copy(input[inputOffset:], calls[index].envelope)
			inputOffset += len(calls[index].envelope)
			copy(detail[detailOffset:], calls[index].detail[:])
			detailOffset += DetailRowBytes
			bundle.CallIDs = append(bundle.CallIDs, hex.EncodeToString(calls[index].callID[:]))
			bundle.EnvelopeDigests = append(bundle.EnvelopeDigests, hex.EncodeToString(calls[index].envelopeDigest[:]))
		}
		journalPath := fmt.Sprintf("E/packs/runs/%s/journal-%04d.arjr", runID, ordinal)
		inputPath := fmt.Sprintf("E/packs/runs/%s/input-%04d.arin", runID, ordinal)
		detailPath := fmt.Sprintf("E/packs/runs/%s/detail-%04d.arcd", runID, ordinal)
		bundle.JournalFiles = append(bundle.JournalFiles, EvidenceFile{Path: journalPath, Mode: "100644", Data: journal})
		bundle.InputFiles = append(bundle.InputFiles, EvidenceFile{Path: inputPath, Mode: "100644", Data: input})
		bundle.DetailFiles = append(bundle.DetailFiles, EvidenceFile{Path: detailPath, Mode: "100644", Data: detail})
		journalShards = append(journalShards, transcriptShard(ordinal, journalPath, first, last, journal))
		inputShards = append(inputShards, transcriptShard(ordinal, inputPath, first, last, input))
		detailShards = append(detailShards, transcriptShard(ordinal, detailPath, first, last, detail))
		first = last
	}
	bundle.JournalRoot = TranscriptRoot{Class: "journal", RunID: runID, RecordSize: JournalRowBytes, TotalRecords: len(calls), Shards: journalShards}
	bundle.InputRoot = TranscriptRoot{Class: "input", RunID: runID, RecordSize: 0, TotalRecords: len(calls), Shards: inputShards}
	bundle.DetailRoot = TranscriptRoot{Class: "detail", RunID: runID, RecordSize: DetailRowBytes, TotalRecords: len(calls), Shards: detailShards}
	for _, root := range []TranscriptRoot{bundle.JournalRoot, bundle.InputRoot, bundle.DetailRoot} {
		if _, err := root.CanonicalJSON(); err != nil {
			return TranscriptBundle{}, err
		}
	}
	return bundle, nil
}

func transcriptShard(ordinal int, path string, first, last int, data []byte) TranscriptShard {
	return TranscriptShard{PackOrdinal: ordinal, Path: path, FirstSequence: uint32(first), LastSequence: uint32(last - 1), RecordCount: last - first, ByteLength: len(data), PackDigest: shaHex(data)}
}

func VerifyTranscript(bundle TranscriptBundle) error {
	if !runIDText(bundle.RunID) || len(bundle.JournalFiles) == 0 || len(bundle.JournalFiles) != len(bundle.InputFiles) || len(bundle.JournalFiles) != len(bundle.DetailFiles) || len(bundle.JournalFiles) != len(bundle.JournalRoot.Shards) || len(bundle.InputFiles) != len(bundle.InputRoot.Shards) || len(bundle.DetailFiles) != len(bundle.DetailRoot.Shards) {
		return fmt.Errorf("invalid transcript bundle shape")
	}
	for _, root := range []TranscriptRoot{bundle.JournalRoot, bundle.InputRoot, bundle.DetailRoot} {
		if root.RunID != bundle.RunID {
			return fmt.Errorf("transcript run mismatch")
		}
		if _, err := root.CanonicalJSON(); err != nil {
			return err
		}
	}
	if bundle.JournalRoot.TotalRecords != bundle.InputRoot.TotalRecords || bundle.JournalRoot.TotalRecords != bundle.DetailRoot.TotalRecords {
		return fmt.Errorf("unaligned transcript totals")
	}
	runRaw, _ := hex.DecodeString(bundle.RunID)
	var previous [32]byte
	var callIDs, envelopeDigests []string
	var runPhase uint8
	for shardOrdinal := range bundle.JournalFiles {
		journalFile, inputFile, detailFile := bundle.JournalFiles[shardOrdinal], bundle.InputFiles[shardOrdinal], bundle.DetailFiles[shardOrdinal]
		journalShard, inputShard, detailShard := bundle.JournalRoot.Shards[shardOrdinal], bundle.InputRoot.Shards[shardOrdinal], bundle.DetailRoot.Shards[shardOrdinal]
		if journalShard.FirstSequence != inputShard.FirstSequence || journalShard.FirstSequence != detailShard.FirstSequence || journalShard.LastSequence != inputShard.LastSequence || journalShard.LastSequence != detailShard.LastSequence {
			return fmt.Errorf("unaligned transcript ranges")
		}
		for _, pair := range []struct {
			file   EvidenceFile
			shard  TranscriptShard
			header string
		}{{journalFile, journalShard, JournalHeader}, {inputFile, inputShard, InputHeader}, {detailFile, detailShard, DetailHeader}} {
			if pair.file.Path != pair.shard.Path || pair.file.Mode != "100644" || len(pair.file.Data) != pair.shard.ByteLength || len(pair.file.Data) < 6 || string(pair.file.Data[:6]) != pair.header || shaHex(pair.file.Data) != pair.shard.PackDigest || len(pair.file.Data) > MaximumPackBytes {
				return fmt.Errorf("invalid transcript physical shard")
			}
		}
		inputRows, err := parseInputFrames(inputFile.Data, inputShard.RecordCount)
		if err != nil {
			return err
		}
		for local := 0; local < journalShard.RecordCount; local++ {
			sequence := journalShard.FirstSequence + uint32(local)
			journal := journalFile.Data[6+local*JournalRowBytes:][:JournalRowBytes]
			detail := detailFile.Data[6+local*DetailRowBytes:][:DetailRowBytes]
			if journal[0] != 1 || !allZero(journal[5:8]) || !allZero(journal[124:128]) || binary.BigEndian.Uint32(journal[8:12]) != sequence || !bytes.Equal(journal[12:28], runRaw) || !bytes.Equal(journal[28:60], previous[:]) {
				return fmt.Errorf("invalid journal row %d", sequence)
			}
			phase, operation, status, counter := journal[1], journal[2], journal[3], journal[4]
			if runPhase == 0 {
				runPhase = phase
			}
			wantCounter, ok := operationCounters[operation]
			if !ok || phase != runPhase || counter != wantCounter || !phaseAllows(phase, operation) || status < 1 || status > 3 || status == 3 && operation != 16 && operation != 18 {
				return fmt.Errorf("invalid journal operation %d", sequence)
			}
			callID := sha256.Sum256(journal)
			if !bytes.Equal(detail[:32], callID[:]) || binary.BigEndian.Uint16(detail[32:34]) != uint16(operation) || detail[34] != phase || detail[35] != operation || detail[36] != status || detail[37] != counter || binary.BigEndian.Uint32(detail[38:42]) != sequence || detail[74] != 1 || detail[75] > 2 || !allZero(detail[76:96]) {
				return fmt.Errorf("journal/detail mismatch %d", sequence)
			}
			envelope := inputRows[local]
			envelopeDigest := sha256.Sum256(envelope)
			if !bytes.Equal(detail[96:128], envelopeDigest[:]) || !allZero(detail[128+int(detail[75])*32:192]) {
				return fmt.Errorf("detail digest/padding mismatch %d", sequence)
			}
			source, err := verifyEnvelope(envelope, phase, operation)
			if err != nil || !bytes.Equal(detail[42:74], source[:]) || len(envelope) > envelopeCap(operation) || !outputCountAllowed(operation, status, int(detail[75])) {
				return fmt.Errorf("invalid aligned envelope %d", sequence)
			}
			outputHex := make([]string, int(detail[75]))
			for index := range outputHex {
				outputHex[index] = hex.EncodeToString(detail[128+index*32 : 160+index*32])
			}
			inputRoot := vectorDigest("actionrelation-input-vector/v1", []string{hex.EncodeToString(envelopeDigest[:])})
			outputRoot := vectorDigest("actionrelation-output-vector/v1", outputHex)
			if !bytes.Equal(journal[60:92], inputRoot[:]) || !bytes.Equal(journal[92:124], outputRoot[:]) {
				return fmt.Errorf("vector root mismatch %d", sequence)
			}
			previous = callID
			callIDs = append(callIDs, hex.EncodeToString(callID[:]))
			envelopeDigests = append(envelopeDigests, hex.EncodeToString(envelopeDigest[:]))
		}
		if shardOrdinal+1 < len(bundle.JournalFiles) {
			nextInputs, err := parseInputFrames(bundle.InputFiles[shardOrdinal+1].Data, bundle.InputRoot.Shards[shardOrdinal+1].RecordCount)
			if err != nil || len(nextInputs) == 0 {
				return fmt.Errorf("invalid next input shard")
			}
			if len(journalFile.Data)+JournalRowBytes <= MaximumPackBytes && len(detailFile.Data)+DetailRowBytes <= MaximumPackBytes && len(inputFile.Data)+4+len(nextInputs[0]) <= MaximumPackBytes {
				return fmt.Errorf("non-greedy aligned transcript shard")
			}
		}
	}
	if !slices.Equal(callIDs, bundle.CallIDs) || !slices.Equal(envelopeDigests, bundle.EnvelopeDigests) {
		return fmt.Errorf("transcript identity list mismatch")
	}
	return nil
}

func parseInputFrames(data []byte, count int) ([][]byte, error) {
	if len(data) < 6 || string(data[:6]) != InputHeader {
		return nil, fmt.Errorf("invalid input pack header")
	}
	rows := make([][]byte, 0, count)
	for offset := 6; offset < len(data); {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("truncated input length")
		}
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if length < 1 || offset+length > len(data) {
			return nil, fmt.Errorf("invalid input frame")
		}
		rows = append(rows, data[offset:offset+length])
		offset += length
	}
	if len(rows) != count {
		return nil, fmt.Errorf("input frame count mismatch")
	}
	return rows, nil
}

func verifyEnvelope(data []byte, phase, operation uint8) ([32]byte, error) {
	var zero [32]byte
	var row []json.RawMessage
	if json.Unmarshal(data, &row) != nil || len(row) != 5 {
		return zero, fmt.Errorf("invalid envelope")
	}
	canonical, _ := json.Marshal(row)
	if !bytes.Equal(canonical, data) {
		return zero, fmt.Errorf("noncanonical envelope")
	}
	var version, source string
	var gotPhase, gotOperation uint8
	if json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &gotPhase) != nil || json.Unmarshal(row[2], &gotOperation) != nil || json.Unmarshal(row[3], &source) != nil || version != "action-charged-input/v1" || gotPhase != phase || gotOperation != operation {
		return zero, fmt.Errorf("envelope identity mismatch")
	}
	if err := validateChargedPayload(operation, row[4]); err != nil {
		return zero, err
	}
	raw, err := hex.DecodeString(source)
	if err != nil || len(raw) != 32 {
		return zero, fmt.Errorf("invalid source task digest")
	}
	var result [32]byte
	copy(result[:], raw)
	return result, nil
}

func vectorDigest(version string, digests []string) [32]byte {
	wire, _ := json.Marshal([]any{version, digests})
	return sha256.Sum256(wire)
}

func envelopeCap(operation uint8) int {
	switch operation {
	case 8:
		return 65536
	case 20:
		return 2048
	default:
		return 1024
	}
}

func phaseAllows(phase, operation uint8) bool {
	if phase == 1 {
		return operation >= 1 && operation <= 8 || operation == 20 || operation == 22
	}
	return phase == 2 && (operation >= 9 && operation <= 19 || operation >= 21 && operation <= 25)
}

func outputCountAllowed(operation, status uint8, count int) bool {
	if status == 2 {
		return count >= 0 && count <= 2
	}
	if status == 3 {
		return count == 1
	}
	switch operation {
	case 1, 3:
		return count == 2
	case 4, 11, 12:
		return count == 1 || count == 2
	case 17, 18:
		return count == 0 || count == 1
	default:
		return count == 1
	}
}

func runIDText(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == string(bytes.ToLower([]byte(value)))
}
