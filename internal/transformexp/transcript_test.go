package transformexp

import (
	"bytes"
	"testing"
)

func TestTransformTranscriptRoundTripAndChainTamper(t *testing.T) {
	manifest := digestBytes([]byte("manifest"))
	sink, err := newTransformTranscriptSink(7, string(NousRefine), "0123456789abcdef", manifest)
	if err != nil {
		t.Fatal(err)
	}
	atom, err := sink.Admit([]byte(`["transform-atom/v1","boolean",true]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(TransformOperation{"verify", "freeze", []string{atom}, []string{atom}, "verified", 11}); err != nil {
		t.Fatal(err)
	}
	terminal, err := sink.Admit([]byte(`["transform-terminal/v1","completed",2,0,0]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(TransformOperation{"terminal", "terminal", []string{atom}, []string{terminal}, "completed", 11}); err != nil {
		t.Fatal(err)
	}
	bundle, err := sink.Bundle()
	if err != nil {
		t.Fatal(err)
	}
	reduced, err := reduceTransformTranscript(bundle.Raw, manifest)
	if err != nil || reduced.Vector != bundle.Vector || reduced.Work != bundle.Work {
		t.Fatalf("reduced=%+v err=%v", reduced, err)
	}
	inflated, err := decodeTransformGzip(bundle.Gzip)
	if err != nil || !bytes.Equal(inflated, bundle.Raw) {
		t.Fatalf("gzip round trip err=%v", err)
	}
	if _, err := decodeTransformGzip(append(bytes.Clone(bundle.Gzip), bundle.Gzip...)); err == nil {
		t.Fatal("accepted concatenated gzip members")
	}
	corrupt := bytes.Clone(bundle.Raw)
	corrupt[bytes.Index(corrupt, []byte("verified"))] = 'x'
	if _, err := reduceTransformTranscript(corrupt, manifest); err == nil {
		t.Fatal("accepted corrupted transcript")
	}
}

func TestTransformEvidenceAttachmentIsImmediateAndSingleUse(t *testing.T) {
	manifest := digestBytes([]byte("manifest"))
	newSink := func() (*TransformTranscriptSink, string) {
		sink, _ := newTransformTranscriptSink(0, string(NousRefine), "0123456789abcdef", manifest)
		atom, _ := sink.Admit([]byte(`["transform-atom/v1","boolean",true]`))
		return sink, atom
	}
	sink, atom := newSink()
	if err := sink.Emit(TransformOperation{"verify", "freeze", []string{atom}, []string{atom}, "verified", 11}); err != nil {
		t.Fatal(err)
	}
	attempt, _ := sink.Admit([]byte(`["transform-evidence-attempt/v1","attached","atom",true,"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","","bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"]`))
	attach := TransformOperation{"evidence-link", "freeze", []string{atom, atom, sink.lastObject}, []string{attempt}, "attached", 10}
	if err := sink.Emit(attach); err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(attach); err == nil {
		t.Fatal("accepted second attachment")
	}
	sink, atom = newSink()
	if err := sink.Emit(TransformOperation{"verify", "freeze", []string{atom}, []string{atom}, "verified", 11}); err != nil {
		t.Fatal(err)
	}
	prior := sink.lastObject
	if err := sink.Emit(TransformOperation{"hash", "freeze", []string{atom}, []string{atom}, "hashed", 11}); err != nil {
		t.Fatal(err)
	}
	attach.Inputs = []string{atom, atom, prior}
	if err := sink.Emit(attach); err == nil {
		t.Fatal("accepted attachment after intervening operation")
	}
}

func TestTransformationLifecycleGoldenApplicationVectors(t *testing.T) {
	for name, test := range map[string]struct {
		vector [12]int64
		work   int64
	}{
		"request-target": {[12]int64{12, 8, 7, 0, 0, 0, 4, 4, 26, 12, 1, 1}, 79},
		"from-value":     {[12]int64{12, 8, 6, 0, 0, 0, 4, 4, 27, 12, 1, 1}, 79},
		"first-local":    {[12]int64{12, 9, 6, 0, 0, 0, 4, 4, 27, 12, 1, 1}, 80},
	} {
		got, err := workForVector(test.vector)
		if err != nil || got != test.work {
			t.Fatalf("%s work=%d err=%v", name, got, err)
		}
	}
}

func TestTransformObjectAdmissionIsCanonicalAndCapped(t *testing.T) {
	table := newTransformObjectTable()
	good := []byte(`["transform-atom/v1","enum","local"]`)
	digest, err := table.admit(good)
	if err != nil || digest != digestBytes(good) {
		t.Fatalf("digest=%s err=%v", digest, err)
	}
	for _, bad := range [][]byte{
		append(bytes.Clone(good), ' '),
		[]byte(`["unknown/v1",true]`),
		[]byte(`["transform-atom/v1", "enum", "local"]`),
	} {
		if _, err := table.admit(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestTransformTranscriptReservesTerminalWork(t *testing.T) {
	manifest := digestBytes([]byte("manifest"))
	sink, _ := newTransformTranscriptSink(0, string(NousRefine), "0123456789abcdef", manifest)
	atom, _ := sink.Admit([]byte(`["transform-atom/v1","boolean",true]`))
	operation := TransformOperation{"verify", "freeze", []string{atom}, []string{atom}, "verified", 11}
	for range LifecycleWorkCap - 1 {
		if err := sink.Emit(operation); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.Emit(operation); err == nil {
		t.Fatal("nonterminal consumed terminal reserve")
	}
}
