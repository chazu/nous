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

func TestCertificateCacheFinalizesAfterProofAndReusesBothAuthorityAndResult(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	worldDigest := strings.Repeat("1", 64)
	runID, _ := actionrelationledger.UtilityRunID("development", "authority", 2, "dynamic-diamond-sleep", 1, worldDigest)
	session, err := BeginSession(store, runID, "utility-cache", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Abort()
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", strings.Repeat("2", 64)})
	cache := NewCertificateCache()
	first, err := CertifyCached(session, cache, worldDigest, "dynamic-diamond-sleep", state, a, b, witness, -1, "first")
	if err != nil || !first.Certified || first.CacheHit || first.CacheRow == "" || first.OperationRoot.Digest == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	row := store.Get(first.CacheRow)
	if row == nil || actionrelationexp.ValidateObject(26, []byte(row.GetString("canonicalObject"))) != nil {
		t.Fatal("cache finalization did not retain an exact kind-26 row")
	}
	var rootWire []any
	if json.Unmarshal(first.OperationRoot.Canonical, &rootWire) != nil || len(rootWire) != 7 || rootWire[4] != float64(0) || rootWire[5] != float64(10) {
		t.Fatalf("operation root=%v", rootWire)
	}
	records, _ := session.Snapshot()
	if len(records) != 11 || records[0].Code != 18 || records[0].Status != 1 || records[10].Code != 25 || records[10].Status != 1 {
		t.Fatalf("first records=%#v", records)
	}
	second, err := CertifyCached(session, cache, worldDigest, "dynamic-diamond-sleep", state, a, b, witness, -1, "second")
	if err != nil || !second.Certified || !second.CacheHit || second.CacheRow != first.CacheRow || second.CertificateDigest != first.CertificateDigest {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	records, err = session.Close()
	if err != nil || len(records) != 12 || records[11].Code != 18 || records[11].Status != 3 {
		t.Fatalf("records=%#v err=%v", records, err)
	}
}
