package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

var emptyCallRoot = sha256.Sum256([]byte("ARCL1-EMPTY\x00"))

type OperationRoot struct {
	Canonical []byte
	Digest    string
}

func BuildOperationRange(runID string, phase uint8, start uint32, callIDs []string) (OperationRoot, error) {
	if !runIDText(runID) || phase < 1 || phase > 2 {
		return OperationRoot{}, fmt.Errorf("invalid operation range identity")
	}
	root, err := callMerkleRoot(start, callIDs)
	if err != nil {
		return OperationRoot{}, err
	}
	wire, _ := json.Marshal([]any{"actionrelation-operation-root/v1", "range", runID, phase, start, len(callIDs), hex.EncodeToString(root[:])})
	return OperationRoot{Canonical: wire, Digest: shaHex(wire)}, nil
}

func BuildOperationConcat(contextDigest string, children []string) (OperationRoot, error) {
	if !digestText(contextDigest) || len(children) == 0 {
		return OperationRoot{}, fmt.Errorf("invalid operation concat")
	}
	for _, child := range children {
		if !digestText(child) {
			return OperationRoot{}, fmt.Errorf("invalid operation concat child")
		}
	}
	wire, _ := json.Marshal([]any{"actionrelation-operation-root/v1", "concat", contextDigest, children})
	return OperationRoot{Canonical: wire, Digest: shaHex(wire)}, nil
}

func VerifyOperationRange(root OperationRoot, transcript TranscriptBundle) error {
	if root.Digest != shaHex(root.Canonical) || VerifyTranscript(transcript) != nil {
		return fmt.Errorf("invalid operation range dependencies")
	}
	var row []json.RawMessage
	if json.Unmarshal(root.Canonical, &row) != nil || len(row) != 7 {
		return fmt.Errorf("invalid operation range wire")
	}
	canonical, _ := json.Marshal(row)
	if !bytes.Equal(canonical, root.Canonical) {
		return fmt.Errorf("noncanonical operation range")
	}
	var version, variant, runID, callRootText string
	var phase uint8
	var start uint32
	var count int
	if json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &variant) != nil || json.Unmarshal(row[2], &runID) != nil || json.Unmarshal(row[3], &phase) != nil || json.Unmarshal(row[4], &start) != nil || json.Unmarshal(row[5], &count) != nil || json.Unmarshal(row[6], &callRootText) != nil || version != "actionrelation-operation-root/v1" || variant != "range" || runID != transcript.RunID || phase < 1 || phase > 2 || count < 0 || int(start)+count > len(transcript.CallIDs) {
		return fmt.Errorf("invalid operation range fields")
	}
	callRoot, err := callMerkleRoot(start, transcript.CallIDs[start:uint32(int(start)+count)])
	if err != nil || callRootText != hex.EncodeToString(callRoot[:]) {
		return fmt.Errorf("operation range Merkle mismatch")
	}
	for sequence := int(start); sequence < int(start)+count; sequence++ {
		shard, local, err := transcriptJournalRow(transcript, uint32(sequence))
		if err != nil || shard[local*JournalRowBytes+1] != phase {
			return fmt.Errorf("operation range phase mismatch")
		}
	}
	return nil
}

func VerifyOperationConcat(root OperationRoot, resolve func(string) (OperationRoot, bool)) error {
	if root.Digest != shaHex(root.Canonical) {
		return fmt.Errorf("invalid concat digest")
	}
	var row []json.RawMessage
	if json.Unmarshal(root.Canonical, &row) != nil || len(row) != 4 {
		return fmt.Errorf("invalid concat wire")
	}
	canonical, _ := json.Marshal(row)
	var version, variant, context string
	var children []string
	if !bytes.Equal(canonical, root.Canonical) || json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &variant) != nil || json.Unmarshal(row[2], &context) != nil || json.Unmarshal(row[3], &children) != nil || version != "actionrelation-operation-root/v1" || variant != "concat" || !digestText(context) || len(children) == 0 {
		return fmt.Errorf("invalid concat fields")
	}
	for _, digest := range children {
		child, ok := resolve(digest)
		if !ok || child.Digest != digest {
			return fmt.Errorf("unresolved concat child")
		}
	}
	return nil
}

func callMerkleRoot(start uint32, callIDs []string) ([32]byte, error) {
	if len(callIDs) == 0 {
		return emptyCallRoot, nil
	}
	level := make([][32]byte, len(callIDs))
	for index, callID := range callIDs {
		raw, err := hex.DecodeString(callID)
		if err != nil || len(raw) != 32 {
			return [32]byte{}, fmt.Errorf("invalid call ID %d", index)
		}
		preimage := make([]byte, 0, len("ARCL1-LEAF\x00")+4+32)
		preimage = append(preimage, []byte("ARCL1-LEAF\x00")...)
		var sequence [4]byte
		binary.BigEndian.PutUint32(sequence[:], start+uint32(index))
		preimage = append(preimage, sequence[:]...)
		preimage = append(preimage, raw...)
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
			preimage := append([]byte("ARCL1-NODE\x00"), left[:]...)
			preimage = append(preimage, right[:]...)
			next[index] = sha256.Sum256(preimage)
		}
		level = next
	}
	return level[0], nil
}

func transcriptJournalRow(transcript TranscriptBundle, sequence uint32) ([]byte, int, error) {
	for index, shard := range transcript.JournalRoot.Shards {
		if sequence >= shard.FirstSequence && sequence <= shard.LastSequence {
			local := int(sequence - shard.FirstSequence)
			return transcript.JournalFiles[index].Data[6:], local, nil
		}
	}
	return nil, 0, fmt.Errorf("sequence outside transcript")
}
