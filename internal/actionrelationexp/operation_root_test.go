package actionrelationexp

import (
	"bytes"
	"testing"
)

func TestOperationRangeBindsExactContiguousCalls(t *testing.T) {
	runID := testDigest("operation-run")[:32]
	source := testDigest("reservation")
	calls := []ChargedCall{
		{Phase: 1, Operation: 5, Status: 1, SourceTaskDigest: source, Payload: []any{"training-applicable", testDigest("s0"), testDigest("a")}, OutputDigests: []string{testDigest("r0")}},
		{Phase: 1, Operation: 6, Status: 1, SourceTaskDigest: source, Payload: []any{"training-equality", testDigest("s1"), testDigest("s2")}, OutputDigests: []string{testDigest("r1")}},
		{Phase: 1, Operation: 5, Status: 1, SourceTaskDigest: source, Payload: []any{"training-applicable", testDigest("s3"), testDigest("b")}, OutputDigests: []string{testDigest("r2")}},
	}
	transcript, err := BuildTranscript(runID, calls)
	if err != nil {
		t.Fatal(err)
	}
	root, err := BuildOperationRange(runID, 1, 1, transcript.CallIDs[1:])
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyOperationRange(root, transcript); err != nil {
		t.Fatal(err)
	}
	corrupt := root
	corrupt.Canonical = bytes.Clone(root.Canonical)
	corrupt.Canonical[len(corrupt.Canonical)-2] ^= 1
	corrupt.Digest = shaHex(corrupt.Canonical)
	if err := VerifyOperationRange(corrupt, transcript); err == nil {
		t.Fatal("accepted a corrupted call-range root")
	}
}

func TestEmptyCallRootAndConcatAreDomainSeparated(t *testing.T) {
	runID := testDigest("empty-operation-run")[:32]
	empty, err := BuildOperationRange(runID, 2, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := BuildOperationRange(runID, 2, 0, []string{testDigest("call")})
	if err != nil {
		t.Fatal(err)
	}
	concat, err := BuildOperationConcat(testDigest("context"), []string{empty.Digest, child.Digest})
	if err != nil {
		t.Fatal(err)
	}
	roots := map[string]OperationRoot{empty.Digest: empty, child.Digest: child}
	if err := VerifyOperationConcat(concat, func(digest string) (OperationRoot, bool) {
		root, ok := roots[digest]
		return root, ok
	}); err != nil {
		t.Fatal(err)
	}
}
