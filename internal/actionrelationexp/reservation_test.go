package actionrelationexp

import (
	"bytes"
	"testing"
)

func TestWorkReservationEnforcesTerminalUnitAndExactBlock(t *testing.T) {
	runID := testDigest("reservation-run")[:32]
	task := testDigest("reservation-task")
	reserved, err := BuildWorkReservation(runID, task, []uint8{5, 4, 6}, 7, 20)
	if err != nil || reserved.Status != "reserved" || reserved.TotalAfter != 10 || VerifyWorkReservation(reserved, 20) != nil {
		t.Fatalf("reserved=%+v err=%v", reserved, err)
	}
	rejected, err := BuildWorkReservation(runID, task, []uint8{5, 4}, 18, 20)
	if err != nil || rejected.Status != "rejected-cap" || rejected.TotalAfter != 18 || VerifyWorkReservation(rejected, 20) != nil {
		t.Fatalf("rejected=%+v err=%v", rejected, err)
	}
	corrupt := reserved
	corrupt.Canonical = bytes.Clone(corrupt.Canonical)
	corrupt.Canonical[len(corrupt.Canonical)-2] ^= 1
	corrupt.Digest = shaHex(corrupt.Canonical)
	if VerifyWorkReservation(corrupt, 20) == nil {
		t.Fatal("accepted forged reservation status")
	}
}
