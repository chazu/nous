// Package programsynth contains domain-neutral identity rules for bounded
// ordered program synthesis. Search and evaluation remain in Nous heuristics.
package programsynth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	MaxProgramLength = 3
	MaxSemanticBytes = 128
	MaxMethodBytes   = 256
)

var semanticPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func ValidSemanticKey(value string) bool {
	return len(value) > 0 && len(value) <= MaxSemanticBytes && semanticPattern.MatchString(value)
}

func ValidSequence(values []string) bool {
	if len(values) == 0 || len(values) > MaxProgramLength {
		return false
	}
	for _, value := range values {
		if !ValidSemanticKey(value) {
			return false
		}
	}
	return true
}

func DecisionKey(method string, semanticSequence []string) (string, error) {
	if method == "" || len(method) > MaxMethodBytes || !ValidSequence(semanticSequence) {
		return "", fmt.Errorf("invalid bounded-program identity")
	}
	material := struct {
		Method   string   `json:"method"`
		Sequence []string `json:"sequence"`
	}{Method: method, Sequence: append([]string(nil), semanticSequence...)}
	encoded, _ := json.Marshal(material)
	digest := sha256.Sum256(encoded)
	return "sha256:v1:" + hex.EncodeToString(digest[:]), nil
}
