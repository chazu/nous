package actionrelationutility

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestUtilityCertificateUsesReservedCallsAndDerivedOperationRoot(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	runID, _ := actionrelationledger.UtilityRunID("development", "authority", 1, "dynamic-diamond-sleep", 0, strings.Repeat("a", 64))
	session, err := BeginSession(store, runID, "utility-certificate", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Abort()
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", strings.Repeat("b", 64)})
	result, err := Certify(session, state, a, b, witness, 0, "reserved")
	if err != nil || result.Terminal != "certified" || result.Certificate == "" || result.Attempt == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if actionrelationexp.ValidateObject(46, result.OperationRoot.Canonical) != nil || store.Get(result.Certificate).GetString("operationRoot") != result.OperationRoot.Digest || store.Get(result.Attempt).GetString("operationRoot") != result.OperationRoot.Digest {
		t.Fatal("certificate does not name its derived kind-46 operation range")
	}
	records, err := session.Close()
	if err != nil || len(records) != 9 {
		t.Fatalf("records=%#v err=%v", records, err)
	}
	for _, record := range records {
		if record.SourceTaskDigest == "" {
			t.Fatal("certificate call lacks pre-execution reservation authority")
		}
	}
}
