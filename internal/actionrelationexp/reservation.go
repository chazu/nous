package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

const acquisitionLifecycleCap = 2_000_000

// WorkReservation is the immutable authority for one ordered block of charged
// primitives. Acquisition currently gives each opaque primitive task a
// one-operation block; later compound search tasks may reserve larger blocks.
type WorkReservation struct {
	Canonical      []byte
	Digest         string
	RunID          string
	TaskDigest     string
	OperationCodes []uint8
	TotalBefore    int
	TotalAfter     int
	Status         string
}

func BuildWorkReservation(runID, taskDigest string, operationCodes []uint8, totalBefore, cap int) (WorkReservation, error) {
	if !runIDText(runID) || !digestText(taskDigest) || len(operationCodes) == 0 || totalBefore < 0 || cap < 1 {
		return WorkReservation{}, fmt.Errorf("invalid work reservation identity")
	}
	for _, code := range operationCodes {
		if _, ok := operationCounters[code]; !ok {
			return WorkReservation{}, fmt.Errorf("invalid reserved operation %d", code)
		}
	}
	totalAfter := totalBefore + len(operationCodes)
	status := "reserved"
	if totalAfter >= cap { // The last lifecycle unit is permanently code-19-only.
		totalAfter, status = totalBefore, "rejected-cap"
	}
	wire, _ := json.Marshal([]any{"compound-work-reservation/v1", runID, taskDigest, operationCodes, totalBefore, totalAfter, status})
	return WorkReservation{
		Canonical: wire, Digest: shaHex(wire), RunID: runID, TaskDigest: taskDigest,
		OperationCodes: slices.Clone(operationCodes), TotalBefore: totalBefore, TotalAfter: totalAfter, Status: status,
	}, nil
}

func VerifyWorkReservation(value WorkReservation, cap int) error {
	if value.Digest != shaHex(value.Canonical) || ValidateObject(27, value.Canonical) != nil || cap < 1 {
		return fmt.Errorf("invalid work reservation")
	}
	var row []json.RawMessage
	if json.Unmarshal(value.Canonical, &row) != nil || len(row) != 7 {
		return fmt.Errorf("invalid work reservation wire")
	}
	canonical, _ := json.Marshal(row)
	var version, runID, taskDigest, status string
	var codes []uint8
	var before, after int
	if !bytes.Equal(canonical, value.Canonical) ||
		json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &runID) != nil ||
		json.Unmarshal(row[2], &taskDigest) != nil || json.Unmarshal(row[3], &codes) != nil ||
		json.Unmarshal(row[4], &before) != nil || json.Unmarshal(row[5], &after) != nil ||
		json.Unmarshal(row[6], &status) != nil || version != "compound-work-reservation/v1" ||
		runID != value.RunID || taskDigest != value.TaskDigest || !slices.Equal(codes, value.OperationCodes) ||
		before != value.TotalBefore || after != value.TotalAfter || status != value.Status ||
		!runIDText(runID) || !digestText(taskDigest) || len(codes) == 0 || before < 0 {
		return fmt.Errorf("work reservation authority mismatch")
	}
	for _, code := range codes {
		if _, ok := operationCounters[code]; !ok {
			return fmt.Errorf("unknown reserved operation")
		}
	}
	if status == "reserved" && after != before+len(codes) || status == "rejected-cap" && (after != before || before+len(codes) < cap) || status != "reserved" && status != "rejected-cap" {
		return fmt.Errorf("invalid work reservation status")
	}
	return nil
}
