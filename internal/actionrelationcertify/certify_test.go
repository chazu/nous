package actionrelationcertify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestOrdinaryCUECertificateTaskAcceptsOnlyCommutingDiamond(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	digest := sha256.Sum256([]byte("dynamic"))
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", hex.EncodeToString(digest[:])})
	operation := sha256.Sum256([]byte("operations"))
	operationRoot := hex.EncodeToString(operation[:])
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	result, err := Execute(store, state, a, b, witness, operationRoot, "commutes")
	if err != nil || result.Terminal != "certified" || result.Certificate == "" || result.Attempt == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if err := actionrelationexp.ValidateObject(17, []byte(store.Get(result.Certificate).GetString("canonicalObject"))); err != nil {
		t.Fatal(err)
	}
	if err := actionrelationexp.ValidateObject(44, []byte(store.Get(result.Attempt).GetString("canonicalObject"))); err != nil {
		t.Fatal(err)
	}
	conflict := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 3}}
	result, err = Execute(store, state, a, conflict, witness, operationRoot, "conflicts")
	if err != nil || result.Terminal != "not-certified" || result.Certificate != "" || result.Attempt == "" {
		t.Fatalf("conflict result=%+v err=%v", result, err)
	}
	if err := actionrelationexp.ValidateObject(44, []byte(store.Get(result.Attempt).GetString("canonicalObject"))); err != nil {
		t.Fatal(err)
	}
}
