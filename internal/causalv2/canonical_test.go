package causalv2

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCanonicalJSONAndStrictDecode(t *testing.T) {
	type sample struct {
		Label string `json:"label"`
		Value int    `json:"value"`
	}
	want := []byte(`{"label":"<λ>","value":3}`)
	encoded, err := CanonicalJSON(sample{"<λ>", 3})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(want) {
		t.Fatalf("canonical=%s, want %s", encoded, want)
	}
	if _, err := StrictDecode[sample](encoded); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{
		[]byte(`{"label":"<λ>","value":3,"extra":0}`),
		[]byte(`{ "label":"<λ>","value":3}`),
		[]byte(`{"label":"<λ>","value":3} {}`),
		[]byte("{\"label\":\"<\\u03bb>\",\"value\":3}"),
	} {
		if _, err := StrictDecode[sample](bad); err == nil {
			t.Fatalf("accepted noncanonical/invalid JSON %q", bad)
		}
	}
}

func TestManifestGolden(t *testing.T) {
	encoded, err := CanonicalJSON(PreregisteredManifest())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(encoded), `{"experiment_version":"active-causal-diagnosis/v2","generator_version":"three-binary-scm/v2"`) {
		t.Fatalf("manifest field order changed: %.120s", encoded)
	}
	hash := sha256.Sum256(encoded)
	got := hex.EncodeToString(hash[:])
	const want = "1c0c6c80130c8b8b9d66606b2c74bb6674ca76ebb682ac16434f9fea6feb76a1"
	if got != want {
		t.Fatalf("manifest golden sha256=%s", got)
	}
}

func TestFixedWidthBytes(t *testing.T) {
	got, err := FixedWidthBytes(123)
	if err != nil || got != "00000123" {
		t.Fatalf("FixedWidthBytes=%q, %v", got, err)
	}
	if _, err := FixedWidthBytes(100000000); err == nil {
		t.Fatal("accepted unrepresentable byte count")
	}
}
