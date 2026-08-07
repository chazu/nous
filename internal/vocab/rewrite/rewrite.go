// Package rewrite implements bounded, deterministic string-rewrite semantics.
// It has no dependency on the Nous engine or DSL.
package rewrite

import (
	"fmt"
	"strings"
)

const (
	MaxTextBytes = 256
	MaxRuleBytes = 8
)

// Rule is one global, non-overlapping, left-to-right replacement pass.
type Rule struct {
	Left  string
	Right string
}

// ValidText reports whether text is a bounded lowercase-ASCII rewrite value.
func ValidText(text string) bool {
	if len(text) > MaxTextBytes {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < 'a' || text[i] > 'z' {
			return false
		}
	}
	return true
}

// ValidRule reports whether rule has a non-empty bounded left side and a
// bounded right side. Deletion rules (an empty right side) are valid.
func ValidRule(rule Rule) bool {
	return rule.Left != "" && len(rule.Left) <= MaxRuleBytes &&
		len(rule.Right) <= MaxRuleBytes && ValidText(rule.Left) && ValidText(rule.Right)
}

// Apply executes one replacement pass.
func Apply(text string, rule Rule) (string, error) {
	if !ValidText(text) {
		return "", fmt.Errorf("invalid rewrite text")
	}
	if !ValidRule(rule) {
		return "", fmt.Errorf("invalid rewrite rule")
	}
	result := strings.ReplaceAll(text, rule.Left, rule.Right)
	if len(result) > MaxTextBytes {
		return "", fmt.Errorf("rewrite result exceeds %d bytes", MaxTextBytes)
	}
	return result, nil
}

// ApplySequence feeds text through each rule exactly once.
func ApplySequence(text string, rules []Rule) (string, error) {
	result := text
	var err error
	for _, rule := range rules {
		result, err = Apply(result, rule)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}
