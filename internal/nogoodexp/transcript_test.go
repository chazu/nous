package nogoodexp

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"slices"
	"testing"
)

func goldenPreflightEvent() TranscriptEvent {
	return TranscriptEvent{
		Category: 12, Code: 16, TaskOrdinal: 0xffffffff,
		Operands: [8]TranscriptOperand{ID("NG-H-ConsiderPrune"), OptionalID(""), Number(54), ID("profile-preflight"), ID("ok"), Omitted(), Omitted(), Omitted()},
	}
}

func TestNGTGoldenAcquisitionStream(t *testing.T) {
	bundle, err := EncodeTranscript([]TranscriptEvent{goldenPreflightEvent()})
	if err != nil {
		t.Fatal(err)
	}
	wantDictionary, _ := hex.DecodeString("0000000300026f6b001170726f66696c652d707265666c6967687400124e472d482d436f6e73696465725072756e65")
	if !slices.Equal(bundle.Raw[4096:4096+47], wantDictionary) {
		t.Fatalf("dictionary = %x", bundle.Raw[4096:4096+47])
	}
	wantRecord := "010c1000ffffffff00000000000000000000000300000000000000360000000200000001000000000000000000000000e204c485551c6120b3757589a8ea9c9920114284c61ca8067613c0a271a0094300000000000000000000000000000000"
	if got := hex.EncodeToString(bundle.Raw[len(bundle.Raw)-96:]); got != wantRecord {
		t.Fatalf("record = %s", got)
	}
	rawDigest := sha256.Sum256(bundle.Raw)
	if len(bundle.Raw) != 4239 || hex.EncodeToString(rawDigest[:]) != "4ce62bdfec29e813e3b7aaffef74acd2416722d82f114a9a51efb5291c56d7bd" {
		t.Fatalf("raw size/digest = %d/%x", len(bundle.Raw), rawDigest)
	}
	wantGzip, _ := base64.StdEncoding.DecodeString("H4sIAAAAAAAC//NzDzFkYGRIYIAAfSjNyHVWanHiQzVvXpclV/aGR3eden74jtXGq6eyllQ8VOk3+8gwCkbBKBgFo2AUjIJRMApGwSgYBaNgFIyCoQSYGZjysxkEC4ry0zJzUnULilLTcjLTM0oYhPzcdT10nfPzijNTUosCikrzUhl5BBj+AwGKdggwA2Im0NABstmPWI60hsokKmwuLe1c8WrOTAVBp5ZjMivYyoQPLCpcwOmM7hYAEloDNo8QAAA=")
	if !slices.Equal(bundle.Gzip, wantGzip) {
		digest := sha256.Sum256(bundle.Gzip)
		t.Fatalf("gzip size/digest = %d/%x", len(bundle.Gzip), digest)
	}
	decoded, err := DecodeTranscript(bundle.Raw)
	if err != nil || decoded.Vector[11] != 1 {
		t.Fatalf("decode = %#v, %v", decoded.Vector, err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0] != goldenPreflightEvent() {
		t.Fatalf("decoded typed event = %#v", decoded.Events)
	}
}

func TestTranscriptRejectsOverflowingEventCountWithoutPanic(t *testing.T) {
	bundle, err := EncodeTranscript([]TranscriptEvent{goldenPreflightEvent()})
	if err != nil {
		t.Fatal(err)
	}
	corrupt := slices.Clone(bundle.Raw)
	binary.BigEndian.PutUint64(corrupt[16:24], ^uint64(0))
	if _, err := DecodeTranscript(corrupt); err == nil {
		t.Fatal("overflowing transcript event count was accepted")
	}
}

func TestNGTReducerRejectsTupleAndEnvelopeCorruption(t *testing.T) {
	bundle, err := EncodeTranscript([]TranscriptEvent{goldenPreflightEvent()})
	if err != nil {
		t.Fatal(err)
	}
	for name, offset := range map[string]int{"header": 56, "dictionary": 4100, "record": len(bundle.Raw) - 1, "tuple": len(bundle.Raw) - 96 + 20} {
		t.Run(name, func(t *testing.T) {
			corrupt := slices.Clone(bundle.Raw)
			corrupt[offset] ^= 1
			if _, err := DecodeTranscript(corrupt); err == nil {
				t.Fatal("corrupt transcript was accepted")
			}
		})
	}
}
