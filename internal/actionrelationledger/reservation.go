package actionrelationledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Reservation struct {
	Canonical      []byte
	Digest         string
	RunID          string
	TaskDigest     string
	OperationCodes []uint8
	TotalBefore    int
	TotalAfter     int
	Status         string
}

func AcquisitionRunID(panel, authority string, curriculum int, scope string) (string, error) {
	if panel == "" || authority == "" || curriculum < 0 || scope != "nous" && scope != "no-guard" {
		return "", fmt.Errorf("invalid acquisition run identity")
	}
	wire, _ := json.Marshal([]any{"actionrelation-run-id/v1", panel, authority, curriculum, "acquisition", scope})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:16]), nil
}

func UtilityRunID(panel, authority string, curriculum int, policy string, worldOrdinal int, worldDigest string) (string, error) {
	if panel == "" || authority == "" || curriculum < 0 || worldOrdinal < 0 || worldOrdinal > 5 || !digestText(worldDigest) || !utilityPolicy(policy) {
		return "", fmt.Errorf("invalid utility run identity")
	}
	wire, _ := json.Marshal([]any{"actionrelation-run-id/v1", panel, authority, curriculum, "utility", policy, worldOrdinal, worldDigest})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:16]), nil
}

func BuildReservation(runID, taskDigest string, operationCodes []uint8, totalBefore, cap int) (Reservation, error) {
	if !runIDText(runID) || !digestText(taskDigest) || len(operationCodes) == 0 || totalBefore < 0 || cap < 1 {
		return Reservation{}, fmt.Errorf("invalid work reservation identity")
	}
	for _, code := range operationCodes {
		if code < 1 || code > 25 {
			return Reservation{}, fmt.Errorf("invalid reserved operation %d", code)
		}
	}
	totalAfter := totalBefore + len(operationCodes)
	status := "reserved"
	if totalAfter >= cap {
		totalAfter, status = totalBefore, "rejected-cap"
	}
	wire, _ := json.Marshal([]any{"compound-work-reservation/v1", runID, taskDigest, operationCodes, totalBefore, totalAfter, status})
	return Reservation{Canonical: wire, Digest: shaHex(wire), RunID: runID, TaskDigest: taskDigest, OperationCodes: slices.Clone(operationCodes), TotalBefore: totalBefore, TotalAfter: totalAfter, Status: status}, nil
}

func ParseReservation(data []byte) (Reservation, error) {
	var row []json.RawMessage
	if json.Unmarshal(data, &row) != nil || len(row) != 7 {
		return Reservation{}, fmt.Errorf("invalid reservation wire")
	}
	canonical, _ := json.Marshal(row)
	if !bytes.Equal(canonical, data) {
		return Reservation{}, fmt.Errorf("noncanonical reservation")
	}
	value := Reservation{Canonical: slices.Clone(data), Digest: shaHex(data)}
	var version string
	if json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &value.RunID) != nil || json.Unmarshal(row[2], &value.TaskDigest) != nil || json.Unmarshal(row[3], &value.OperationCodes) != nil || json.Unmarshal(row[4], &value.TotalBefore) != nil || json.Unmarshal(row[5], &value.TotalAfter) != nil || json.Unmarshal(row[6], &value.Status) != nil || version != "compound-work-reservation/v1" {
		return Reservation{}, fmt.Errorf("invalid reservation fields")
	}
	return value, nil
}

func VerifyReservation(value Reservation, cap int) error {
	parsed, err := ParseReservation(value.Canonical)
	if err != nil || parsed.Digest != value.Digest || parsed.RunID != value.RunID || parsed.TaskDigest != value.TaskDigest || !slices.Equal(parsed.OperationCodes, value.OperationCodes) || parsed.TotalBefore != value.TotalBefore || parsed.TotalAfter != value.TotalAfter || parsed.Status != value.Status || cap < 1 || !runIDText(value.RunID) || !digestText(value.TaskDigest) || len(value.OperationCodes) == 0 || value.TotalBefore < 0 {
		return fmt.Errorf("invalid work reservation")
	}
	for _, code := range value.OperationCodes {
		if code < 1 || code > 25 {
			return fmt.Errorf("unknown reserved operation")
		}
	}
	if value.Status == "reserved" && value.TotalAfter != value.TotalBefore+len(value.OperationCodes) || value.Status == "rejected-cap" && (value.TotalAfter != value.TotalBefore || value.TotalBefore+len(value.OperationCodes) < cap) || value.Status != "reserved" && value.Status != "rejected-cap" {
		return fmt.Errorf("invalid work reservation status")
	}
	return nil
}

func TaskDigest(runID string, sequence int, code uint8) string {
	wire, _ := json.Marshal([]any{"actionrelation-task/v1", runID, "acquisition", sequence, code})
	return shaHex(wire)
}

func utilityPolicy(value string) bool {
	switch value {
	case "complete", "lexical-order", "static-rw-sleep", "dynamic-diamond-sleep", "nous-guarded-sleep", "no-guard-sleep", "learned-no-use":
		return true
	default:
		return false
	}
}

func shaHex(data []byte) string { digest := sha256.Sum256(data); return hex.EncodeToString(digest[:]) }
func runIDText(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 16 && value == strings.ToLower(value)
}
func digestText(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 32 && value == strings.ToLower(value)
}
