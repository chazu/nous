package actionrelationexp

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
)

func TestAcquisitionBoundaryIndexesOnlyNonTableLogicalObjects(t *testing.T) {
	session, err := actionrelationacquire.Begin("../../domains", "boundary")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := CompleteAcquisition(session, 2)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := BuildAcquisitionBoundary(evidence, 2, "nous")
	if err != nil {
		t.Fatal(err)
	}
	if err := boundary.Verify(evidence); err != nil {
		t.Fatal(err)
	}
	seenKinds := map[uint16]bool{}
	for _, file := range boundary.Preboundary.IndexFiles {
		for offset := len(IndexHeader); offset < len(file.Data); offset += ObjectIndexRowBytes {
			kind := binary.BigEndian.Uint16(file.Data[offset+44 : offset+46])
			if kind >= 101 {
				t.Fatalf("fixed table kind %d leaked into object index", kind)
			}
			seenKinds[kind] = true
		}
	}
	for _, kind := range []uint16{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 27, 28, 46} {
		if !seenKinds[kind] {
			t.Fatalf("missing acquisition object decoder kind %d", kind)
		}
	}
	var wire []any
	if json.Unmarshal(boundary.Canonical, &wire) != nil || len(wire) != 6 || len(wire[3].([]any)) != 8 {
		t.Fatalf("boundary wire=%v", wire)
	}
}

func TestAcquisitionBoundaryDetectsLaterLogicalObject(t *testing.T) {
	session, err := actionrelationacquire.Begin("../../domains", "boundary-mutation")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := CompleteAcquisition(session, 3)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := BuildAcquisitionBoundary(evidence, 3, "nous")
	if err != nil {
		t.Fatal(err)
	}
	experiment := evidence.Run.Store.Get(evidence.Run.Experiment)
	patternName := experiment.GetString("patternUnit")
	pattern := evidence.Run.Store.Get(patternName)
	pattern.Set("canonicalObject", "[\"action-relation-pattern/v1\",[\"add\",\"emit\"],[0,-1,-1,-1]]")
	pattern.Set("objectDigest", shaHex([]byte(pattern.GetString("canonicalObject"))))
	if err := boundary.Verify(evidence); err == nil {
		t.Fatal("boundary accepted a post-boundary logical mutation")
	}
}
