package actionrelationledger

import (
	"strings"
	"testing"
)

func TestReservationRoundTripAndCap(t *testing.T) {
	runID, err := AcquisitionRunID("development", "a3e18b10a01cf83315bff398586e91cd33544861", 4, "nous")
	if err != nil {
		t.Fatal(err)
	}
	task := TaskDigest(runID, 7, 19)
	reservation, err := BuildReservation(runID, task, []uint8{19}, 7, 10)
	if err != nil || reservation.Status != "reserved" || VerifyReservation(reservation, 10) != nil {
		t.Fatalf("reservation=%+v err=%v", reservation, err)
	}
	parsed, err := ParseReservation(reservation.Canonical)
	if err != nil || parsed.Digest != reservation.Digest || parsed.TaskDigest != task {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
	rejected, err := BuildReservation(runID, TaskDigest(runID, 9, 20), []uint8{20}, 9, 10)
	if err != nil || rejected.Status != "rejected-cap" || rejected.TotalAfter != 9 || VerifyReservation(rejected, 10) != nil {
		t.Fatalf("rejected=%+v err=%v", rejected, err)
	}
}

func TestReservationRejectsTampering(t *testing.T) {
	runID, _ := AcquisitionRunID("development", "authority", 1, "no-guard")
	reservation, err := BuildReservation(runID, TaskDigest(runID, 0, 8), []uint8{8}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	reservation.TotalAfter = 2
	if err := VerifyReservation(reservation, 10); err == nil {
		t.Fatal("verification accepted a mutated reservation")
	}
}

func TestUtilityRunIDUsesExactWorldAndPolicyContext(t *testing.T) {
	world := strings.Repeat("c", 64)
	first, err := UtilityRunID("development", "authority", 3, "dynamic-diamond-sleep", 2, world)
	if err != nil || len(first) != 32 {
		t.Fatalf("runID=%q err=%v", first, err)
	}
	second, err := UtilityRunID("development", "authority", 3, "nous-guarded-sleep", 2, world)
	if err != nil || first == second {
		t.Fatalf("context collision first=%q second=%q err=%v", first, second, err)
	}
	if _, err := UtilityRunID("development", "authority", 3, "unknown", 2, world); err == nil {
		t.Fatal("unknown utility policy accepted")
	}
}
