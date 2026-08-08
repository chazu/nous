package causalv2

import (
	"errors"
	"fmt"
)

func CheckByteCap(data []byte, capBytes int) error {
	if capBytes < 0 {
		return errors.New("negative byte cap")
	}
	if len(data) > capBytes {
		return fmt.Errorf("encoded bytes=%d exceed cap=%d", len(data), capBytes)
	}
	return nil
}

// CheckSplitRecord checks a canonical record encoded once with its record array
// empty, the exact bytes between that array's brackets, and their composed cap.
func CheckSplitRecord(emptyArrayRecord, arrayContents []byte, baseCap, contentsCap, totalCap int) error {
	if err := CheckByteCap(emptyArrayRecord, baseCap); err != nil {
		return fmt.Errorf("record shell: %w", err)
	}
	if err := CheckByteCap(arrayContents, contentsCap); err != nil {
		return fmt.Errorf("record array contents: %w", err)
	}
	if len(emptyArrayRecord)+len(arrayContents) > totalCap {
		return fmt.Errorf("composed record bytes=%d exceed cap=%d", len(emptyArrayRecord)+len(arrayContents), totalCap)
	}
	return nil
}

func FixedWidthBytes(value int) (string, error) {
	if value < 0 || value >= 100000000 {
		return "", errors.New("byte count is outside eight-decimal-digit range")
	}
	return fmt.Sprintf("%08d", value), nil
}

func RecordArrayBytes(recordLengths []int) (int, error) {
	total := 0
	for _, length := range recordLengths {
		if length < 0 {
			return 0, errors.New("negative record length")
		}
		total += length
	}
	if len(recordLengths) > 1 {
		total += len(recordLengths) - 1
	}
	return total, nil
}
