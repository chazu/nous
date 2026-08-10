package actionrelationwire

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestRootDigestUsesFrozenDomainSeparation(t *testing.T) {
	got, err := RootDigest("local-fact-pair", []string{"aa", "bb"})
	if err != nil {
		t.Fatal(err)
	}
	wantBytes := sha256.Sum256([]byte(`["actionrelation-root/v1","local-fact-pair",["aa","bb"]]`))
	if want := hex.EncodeToString(wantBytes[:]); got != want {
		t.Fatalf("root=%s want %s", got, want)
	}
	if _, err := RootDigest("invented", nil); err == nil {
		t.Fatal("accepted an unregistered authority root tag")
	}
}
