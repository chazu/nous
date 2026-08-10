package actionrelationcertify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/dsl"
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

func TestCertificateAssemblyWaitsForClosedOperationRange(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	witnessDigest := strings.Repeat("c", 64)
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", witnessDigest})
	codes := []uint16{13, 13, 12, 12, 13, 13, 12, 12, 14}
	plan := make([]dsl.ActionRelationMeterPlanEntry, len(codes))
	for index, code := range codes {
		plan[index] = dsl.ActionRelationMeterPlanEntry{Code: code, SourceTaskDigest: strings.Repeat("d", 64)}
	}
	if err := dsl.RegisterActionRelationMeterPlan("certificate-stages", plan); err != nil {
		t.Fatal(err)
	}
	defer dsl.UnregisterActionRelationMeter("certificate-stages")
	request, err := Begin(store, state, a, b, witness, "stages", "certificate-stages")
	if err != nil {
		t.Fatal(err)
	}
	if err := RunInitial(store, request); err != nil || len(CrossOperationCodes(store, request)) != 4 {
		t.Fatalf("initial err=%v cross=%v", err, CrossOperationCodes(store, request))
	}
	if err := RunCross(store, request); err != nil || len(EqualityOperationCodes(store, request)) != 1 {
		t.Fatalf("cross err=%v equality=%v", err, EqualityOperationCodes(store, request))
	}
	if err := RunEquality(store, request); err != nil {
		t.Fatal(err)
	}
	if store.Get(request).GetString("operationRoot") != "" || store.Get(request).GetString("certificateAttemptUnit") != "" {
		t.Fatal("certificate authority was assembled before its operation range closed")
	}
	records, _ := dsl.ActionRelationMeterSnapshot("certificate-stages")
	if len(records) != len(codes) {
		t.Fatalf("records=%#v", records)
	}
	operation := sha256.Sum256([]byte("closed-after-proof"))
	result, err := Complete(store, request, hex.EncodeToString(operation[:]))
	if err != nil || result.Terminal != "certified" || result.Attempt == "" || result.Certificate == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	again, _ := dsl.ActionRelationMeterSnapshot("certificate-stages")
	if len(again) != len(records) {
		t.Fatal("uncharged assembly emitted a semantic call")
	}
}
