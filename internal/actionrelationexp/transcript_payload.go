package actionrelationexp

import (
	"encoding/json"
	"fmt"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func validateChargedPayload(operation uint8, raw json.RawMessage) error {
	var row []json.RawMessage
	if json.Unmarshal(raw, &row) != nil || len(row) == 0 {
		return fmt.Errorf("invalid charged payload")
	}
	tag, ok := rawString(row[0])
	if !ok {
		return fmt.Errorf("invalid payload tag")
	}
	d := func(index int) bool { return index < len(row) && rawDigest(row[index]) }
	ds := func(index, minimum, maximum int) bool {
		return index < len(row) && rawDigestList(row[index], minimum, maximum)
	}
	s := func(index int) (string, bool) {
		if index >= len(row) {
			return "", false
		}
		return rawString(row[index])
	}
	i := func(index, minimum, maximum int) bool {
		if index >= len(row) {
			return false
		}
		var value int
		return json.Unmarshal(row[index], &value) == nil && value >= minimum && value <= maximum
	}
	b := func(index int) bool {
		if index >= len(row) {
			return false
		}
		var value bool
		return json.Unmarshal(row[index], &value) == nil
	}
	policy := func(index int) bool {
		value, valid := s(index)
		return valid && policyNames[value]
	}
	atom := func(index int) bool {
		value, valid := s(index)
		if !valid {
			return false
		}
		for _, candidate := range actionrelations.Atoms {
			if value == candidate {
				return true
			}
		}
		return false
	}
	valid := false
	switch operation {
	case 1:
		valid = tag == "guard-root" && len(row) == 2 && d(1)
	case 2:
		valid = tag == "candidate-allocate" && len(row) == 5 && d(1) && d(2) && d(3) && i(4, 1, 450)
	case 3:
		valid = tag == "guard-extend" && len(row) == 5 && d(1) && atom(2) && b(3) && i(4, 0, 449)
	case 4:
		valid = tag == "training-apply" && len(row) == 4 && d(1) && d(2) && d(3)
	case 5:
		valid = tag == "training-applicable" && len(row) == 3 && d(1) && d(2)
	case 6:
		valid = tag == "training-equality" && len(row) == 3 && d(1) && d(2)
	case 7:
		valid = tag == "training-literal" && len(row) == 7 && d(1) && d(2) && d(3) && d(4) && atom(5) && b(6)
	case 8:
		valid = tag == "artifact-freeze" && len(row) == 4 && d(1) && ds(2, 1, 451) && d(3)
	case 9:
		valid = tag == "relation-match" && len(row) == 8 && d(1) && d(2) && d(3) && d(4) && d(5) && d(6) && ds(7, 0, 2)
	case 10:
		valid = tag == "artifact-load" && len(row) == 3 && d(1) && d(2)
	case 11:
		valid = tag == "utility-apply" && len(row) == 4 && d(1) && d(2) && d(3)
	case 12:
		valid = tag == "certificate-apply" && len(row) == 4 && d(1) && d(2) && d(3)
	case 13:
		valid = tag == "certificate-applicable" && len(row) == 3 && d(1) && d(2)
	case 14:
		valid = tag == "certificate-equality" && len(row) == 3 && d(1) && d(2)
	case 15:
		valid = tag == "learned-literal" && len(row) == 6 && d(1) && d(2) && d(3) && atom(4) && b(5)
	case 16:
		valid = tag == "search-node-lookup" && len(row) == 4 && d(1) && d(2) && d(3)
	case 17:
		valid = tag == "proof-map-lookup" && len(row) == 4 && d(1) && d(2) && d(3)
	case 18:
		valid = tag == "certificate-cache-lookup" && len(row) == 6 && d(1) && policy(2) && d(3) && d(4) && d(5)
	case 19:
		valid = tag == "terminal-construct" && len(row) == 4 && d(1) && d(2) && ds(3, 0, 8) ||
			tag == "budget-terminal" && len(row) == 2 && d(1)
	case 20:
		valid = tag == "candidate-result" && len(row) == 4 && d(1) && ds(2, 16, 16) && d(3)
	case 21:
		valid = tag == "relation-instance-applicable" && len(row) == 3 && d(1) && d(2)
	case 22:
		valid = tag == "guard-result" && len(row) == 4 && d(1) && d(2) && ds(3, 0, 2)
	case 23:
		valid = tag == "search-applicable" && len(row) == 6 && d(1) && policy(2) && d(3) && d(4) && d(5)
	case 24:
		valid = tag == "static-footprint" && len(row) == 8 && d(1) && d(2) && d(3) && d(4) && d(5) && d(6) && d(7)
	case 25:
		valid = tag == "certificate-cache-finalize" && len(row) == 9 && d(1) && policy(2) && d(3) && d(4) && d(5) && d(6) && d(7) && d(8)
	}
	if !valid {
		return fmt.Errorf("payload does not match operation %d matrix", operation)
	}
	return nil
}

var policyNames = map[string]bool{
	"complete": true, "lexical-order": true, "static-rw-sleep": true,
	"dynamic-diamond-sleep": true, "nous-guarded-sleep": true,
	"no-guard-sleep": true, "learned-no-use": true,
}

func rawString(raw json.RawMessage) (string, bool) {
	var value string
	return value, json.Unmarshal(raw, &value) == nil
}

func rawDigest(raw json.RawMessage) bool {
	value, ok := rawString(raw)
	return ok && digestText(value)
}

func rawDigestList(raw json.RawMessage, minimum, maximum int) bool {
	var values []string
	if json.Unmarshal(raw, &values) != nil || len(values) < minimum || len(values) > maximum {
		return false
	}
	for _, value := range values {
		if !digestText(value) {
			return false
		}
	}
	return true
}
