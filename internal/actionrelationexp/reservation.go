package actionrelationexp

import (
	"fmt"

	"github.com/chazu/nous/internal/actionrelationledger"
)

const acquisitionLifecycleCap = 2_000_000

// WorkReservation is the immutable authority for one ordered block of charged
// primitives. Acquisition currently gives each opaque primitive task a
// one-operation block; later compound search tasks may reserve larger blocks.
type WorkReservation = actionrelationledger.Reservation

func BuildWorkReservation(runID, taskDigest string, operationCodes []uint8, totalBefore, cap int) (WorkReservation, error) {
	return actionrelationledger.BuildReservation(runID, taskDigest, operationCodes, totalBefore, cap)
}

func VerifyWorkReservation(value WorkReservation, cap int) error {
	if ValidateObject(27, value.Canonical) != nil {
		return fmt.Errorf("reservation does not decode as kind 27")
	}
	return actionrelationledger.VerifyReservation(value, cap)
}
