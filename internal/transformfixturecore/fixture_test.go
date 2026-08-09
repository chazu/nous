package transformfixturecore

import (
	"strings"
	"testing"
)

func TestExactProgramBatchMaximum(t *testing.T) {
	program := []byte(`["concrete-program/v1",[["set-value/v1",8,"abcdefghijklmnop"],["set-value/v1",9,"abcdefghijklmnop"],["set-value/v1",10,"abcdefghijklmnop"],["set-value/v1",11,"abcdefghijklmnop"]]]`)
	batch := ProgramBatch{}
	for i := 0; i < 4; i++ {
		batch.Rows = append(batch.Rows, ProgramRow{Token: []string{"0000000000000000", "1111111111111111", "2222222222222222", "3333333333333333"}[i], BeforeDigest: strings.Repeat("f", 64), Program: program})
	}
	b, err := batch.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1104 {
		t.Fatalf("maximum batch bytes=%d", len(b))
	}
	parsed, err := ParseProgramBatch(b)
	if err != nil || len(parsed.Rows) != 4 {
		t.Fatalf("parse rows=%d err=%v", len(parsed.Rows), err)
	}
	if _, err := ParseProgramBatch(append(b, ' ')); err == nil {
		t.Fatal("accepted noncanonical program batch")
	}
}

func TestProfileDigestStable(t *testing.T) {
	if len(ProfileDigest()) != 64 || ProfileDigest() != ProfileDigest() {
		t.Fatal("unstable profile digest")
	}
}
