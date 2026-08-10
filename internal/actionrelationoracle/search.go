package actionrelationoracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
)

type oracleOccurrence struct {
	action    action
	canonical []byte
	digest    string
}

// CompleteTerminalDigests independently enumerates the complete interleaving
// universe for one normalized state and ordered semantic-action multiset.
func CompleteTerminalDigests(stateJSON []byte, actionJSONs [][]byte) ([]string, error) {
	initial, err := parseState(stateJSON)
	if err != nil || len(actionJSONs) < 1 || len(actionJSONs) > 8 {
		return nil, ErrInvalid
	}
	type parsedAction struct {
		value     action
		canonical []byte
	}
	parsed := make([]parsedAction, len(actionJSONs))
	for index, data := range actionJSONs {
		parsed[index].value, err = parseAction(data)
		if err != nil {
			return nil, err
		}
		parsed[index].canonical = bytes.Clone(data)
	}
	slices.SortFunc(parsed, func(a, b parsedAction) int { return bytes.Compare(a.canonical, b.canonical) })
	occurrences := make([]oracleOccurrence, len(parsed))
	ordinal := 0
	for index, item := range parsed {
		if index > 0 && !bytes.Equal(parsed[index-1].canonical, item.canonical) {
			ordinal = 0
		}
		canonical, _ := json.Marshal([]any{"action-occurrence/v1", json.RawMessage(item.canonical), ordinal})
		digest := sha256.Sum256(canonical)
		occurrences[index] = oracleOccurrence{action: item.value, canonical: canonical, digest: hex.EncodeToString(digest[:])}
		ordinal++
	}
	memo := map[string][]string{}
	var visit func(state, []oracleOccurrence) ([]string, error)
	visit = func(current state, remaining []oracleOccurrence) ([]string, error) {
		currentJSON, _ := encodeState(current)
		remainingDigests := oracleOccurrenceDigests(remaining)
		keyBytes, _ := json.Marshal([]any{"oracle-search-key/v1", json.RawMessage(currentJSON), remainingDigests})
		keyHash := sha256.Sum256(keyBytes)
		key := hex.EncodeToString(keyHash[:])
		if cached, ok := memo[key]; ok {
			return slices.Clone(cached), nil
		}
		var terminals []string
		enabled := false
		for index, occurrence := range remaining {
			next, applicable := transition(current, occurrence.action)
			if !applicable {
				continue
			}
			enabled = true
			child := append(slices.Clone(remaining[:index]), remaining[index+1:]...)
			values, err := visit(next, child)
			if err != nil {
				return nil, err
			}
			terminals = append(terminals, values...)
		}
		if !enabled {
			terminal := "complete"
			if len(remaining) > 0 {
				terminal = "deadlock"
			}
			wire, _ := json.Marshal([]any{"action-terminal/v1", json.RawMessage(currentJSON), remainingDigests, terminal})
			digest := sha256.Sum256(wire)
			terminals = []string{hex.EncodeToString(digest[:])}
		}
		slices.Sort(terminals)
		terminals = slices.Compact(terminals)
		memo[key] = slices.Clone(terminals)
		return terminals, nil
	}
	return visit(initial, occurrences)
}

func oracleOccurrenceDigests(occurrences []oracleOccurrence) []string {
	result := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		result[index] = occurrence.digest
	}
	slices.Sort(result)
	return result
}
