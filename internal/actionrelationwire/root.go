// Package actionrelationwire implements the shared domain-separated roots used
// by the guarded-action-relations evidence format.
package actionrelationwire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

var rootTags = map[string]bool{
	"competence-cases":           true,
	"competence-results":         true,
	"curriculum-policy-rows":     true,
	"expected-run-ids":           true,
	"generator-draws":            true,
	"guard-result-vector":        true,
	"indexed-object-set":         true,
	"local-fact-pair":            true,
	"occurrence-map":             true,
	"original-actions":           true,
	"result-rows":                true,
	"run-charged-outputs":        true,
	"run-structural-outputs":     true,
	"scorer-shards":              true,
	"semantic-training":          true,
	"structural-output-map":      true,
	"transcript-rows":            true,
	"unanimous-relation-matches": true,
	"view-evidence":              true,
	"world-policy-rows":          true,
}

func RootDigest(tag string, value any) (string, error) {
	if !rootTags[tag] {
		return "", fmt.Errorf("unknown action-relation root tag %q", tag)
	}
	wire, err := json.Marshal([]any{"actionrelation-root/v1", tag, value})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}
