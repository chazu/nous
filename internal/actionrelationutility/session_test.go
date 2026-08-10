package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

func TestSessionRetainsCompoundReservationBeforeItsCalls(t *testing.T) {
	store := unit.NewStore()
	runID, _ := actionrelationledger.UtilityRunID("development", "authority", 1, "complete", 0, strings.Repeat("d", 64))
	session, err := BeginSession(store, runID, "utility-session", 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Abort()
	taskRaw := sha256.Sum256([]byte("search-node-task"))
	taskDigest := hex.EncodeToString(taskRaw[:])
	reservation, err := session.Reserve(taskDigest, []uint8{16, 23, 23})
	if err != nil || reservation.Status != "reserved" || store.Get(session.Reservations[0]).GetString("objectDigest") != reservation.Digest {
		t.Fatalf("reservation=%+v err=%v", reservation, err)
	}
	for _, event := range []struct {
		code    uint16
		counter uint8
	}{{16, 11}, {23, 10}, {23, 10}} {
		if err := dsl.ChargeActionRelationMeter(session.MeterToken, event.code, event.counter, "test", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	records, err := session.Close()
	if err != nil || len(records) != 3 {
		t.Fatalf("records=%#v err=%v", records, err)
	}
	for _, record := range records {
		if record.SourceTaskDigest != reservation.Digest {
			t.Fatal("compound call lost reservation authority")
		}
	}
}

func TestSessionRejectsBlockAtReservedTerminalUnit(t *testing.T) {
	store := unit.NewStore()
	runID, _ := actionrelationledger.UtilityRunID("development", "authority", 1, "complete", 0, strings.Repeat("e", 64))
	session, err := BeginSession(store, runID, "utility-cap", 8, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Abort()
	initial := [12]int{8}
	if err := session.SetInitialWorkVector(initial); err != nil {
		t.Fatal(err)
	}
	task := strings.Repeat("f", 64)
	reservation, err := session.Reserve(task, []uint8{16, 23})
	if err != nil || reservation.Status != "rejected-cap" || session.Total != 8 {
		t.Fatalf("reservation=%+v err=%v", reservation, err)
	}
	records, _ := session.Snapshot()
	if len(records) != 0 {
		t.Fatal("rejected compound block executed calls")
	}
	terminal, err := session.TerminateBudget(reservation)
	if err != nil || actionrelationexp.ValidateObject(49, terminal.Canonical) != nil {
		t.Fatalf("terminal=%+v err=%v", terminal, err)
	}
	records, err = session.Close()
	if err != nil || len(records) != 1 || records[0].Code != 19 || records[0].SourceTaskDigest == reservation.Digest {
		t.Fatalf("terminal records=%#v err=%v", records, err)
	}
}
