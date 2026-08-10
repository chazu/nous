package actionrelationexp

import (
	"encoding/binary"
	"fmt"
)

func ValidateTableRecord(kind uint16, row []byte) error {
	if len(row) != tableRecordSizes[kind] {
		return fmt.Errorf("wrong record size")
	}
	validStatus := func(value byte) bool { return value == 1 || value == 2 }
	boolean := func(value byte) bool { return value <= 1 }
	nonzero := func(data []byte) bool { return !allZero(data) }
	switch kind {
	case 101:
		if !nonzero(row[0:32]) || !nonzero(row[32:64]) || binary.BigEndian.Uint16(row[64:66]) < 1 || binary.BigEndian.Uint16(row[64:66]) > 15 || !boolean(row[66]) || !boolean(row[67]) || !validStatus(row[68]) || row[68] == 2 && row[67] != 0 || !allZero(row[69:96]) || !nonzero(row[96:128]) {
			return fmt.Errorf("invalid signed-literal row")
		}
	case 102:
		if !nonzero(row[0:32]) || !nonzero(row[32:64]) || !boolean(row[64]) || !validStatus(row[65]) || row[65] == 2 && row[64] != 0 || !allZero(row[66:96]) {
			return fmt.Errorf("invalid guard-result row")
		}
	case 103:
		ordinal := binary.BigEndian.Uint16(row[96:98])
		if !nonzero(row[0:32]) || !nonzero(row[64:96]) || ordinal > 450 || row[98] > 2 || !validStatus(row[99]) || !allZero(row[100:128]) || ordinal == 0 && !allZero(row[32:64]) || ordinal != 0 && !nonzero(row[32:64]) {
			return fmt.Errorf("invalid candidate row")
		}
	case 104:
		if !nonzero(row[0:32]) || !nonzero(row[32:64]) || binary.BigEndian.Uint16(row[64:66]) < 1 || binary.BigEndian.Uint16(row[64:66]) > 15 || !boolean(row[66]) || !validStatus(row[67]) || binary.BigEndian.Uint32(row[68:72]) > 449 || !allZero(row[72:96]) {
			return fmt.Errorf("invalid refinement row")
		}
	case 105:
		if row[0] != 1 || row[1] < 1 || row[1] > 9 || !validStatus(row[2]) || row[3]&0xc3 != 0 || !nonzero(row[4:36]) || !nonzero(row[36:68]) || !nonzero(row[68:100]) || !nonzero(row[292:324]) || !allZero(row[324:]) {
			return fmt.Errorf("invalid observation header")
		}
		for bit, start := range []int{100, 132, 164, 196, 228, 260} {
			isNull := row[3]&(1<<bit) != 0
			if isNull != allZero(row[start:start+32]) {
				return fmt.Errorf("observation null bitmap mismatch")
			}
		}
	case 106:
		validDigests := true
		for start := 0; start < 224; start += 32 {
			validDigests = validDigests && nonzero(row[start:start+32])
		}
		if !validDigests || row[224] > 1 || row[225] < 1 || row[225] > 3 || row[226] < 1 || row[226] > 8 || !validStatus(row[227]) || !allZero(row[228:]) {
			return fmt.Errorf("invalid view-evidence row")
		}
	case 107:
		if row[0] != 1 || row[1] < 1 || row[1] > 3 || !validStatus(row[2]) || !nonzero(row[4:36]) || !nonzero(row[36:68]) || !allZero(row[100:]) {
			return fmt.Errorf("invalid training-operation header")
		}
		resultObject := nonzero(row[68:100])
		valid := false
		switch row[1] {
		case 1:
			valid = row[2] == 1 && row[3] <= 1 && !resultObject || row[2] == 2 && row[3] == 2 && !resultObject
		case 2:
			valid = row[2] == 1 && row[3] == 1 && resultObject || row[2] == 1 && row[3] == 2 && !resultObject || row[2] == 2 && row[3] == 3 && !resultObject
		case 3:
			valid = row[2] == 1 && row[3] <= 1 && !resultObject || row[2] == 2 && row[3] == 2 && !resultObject
		}
		if !valid {
			return fmt.Errorf("invalid training-operation result matrix")
		}
	case 108:
		if !nonzero(row[0:32]) || !nonzero(row[32:64]) || row[64] > 16 || row[65] > 16 || !boolean(row[66]) || !boolean(row[67]) || !validStatus(row[68]) || row[68] == 2 && row[67] != 0 || !allZero(row[69:]) {
			return fmt.Errorf("invalid candidate-result row")
		}
	default:
		return fmt.Errorf("unknown table kind")
	}
	return nil
}
