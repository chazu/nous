package rewrite

import (
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	for _, valid := range []string{"", "a", strings.Repeat("z", MaxTextBytes)} {
		if !ValidText(valid) {
			t.Fatalf("ValidText(%q) = false", valid)
		}
	}
	for _, invalid := range []string{"A", "a-b", "a b", strings.Repeat("a", MaxTextBytes+1)} {
		if ValidText(invalid) {
			t.Fatalf("ValidText(%q) = true", invalid)
		}
	}
	for _, rule := range []Rule{{"a", ""}, {"ab", "x"}, {strings.Repeat("a", MaxRuleBytes), strings.Repeat("b", MaxRuleBytes)}} {
		if !ValidRule(rule) {
			t.Fatalf("ValidRule(%#v) = false", rule)
		}
	}
	for _, rule := range []Rule{{"", "a"}, {"A", "a"}, {strings.Repeat("a", MaxRuleBytes+1), "b"}, {"a", strings.Repeat("b", MaxRuleBytes+1)}} {
		if ValidRule(rule) {
			t.Fatalf("ValidRule(%#v) = true", rule)
		}
	}
}

func TestDecisionKeyIsOrderedAndUnambiguous(t *testing.T) {
	first := DecisionKey("a/b", "c")
	if first == DecisionKey("a", "b/c") || first == DecisionKey("c", "a/b") {
		t.Fatal("decision key lost tuple boundaries or order")
	}
	if first != DecisionKey("a/b", "c") {
		t.Fatal("decision key is not deterministic")
	}
}

func TestApplyIsOnePassAndNonOverlapping(t *testing.T) {
	tests := []struct {
		input string
		rule  Rule
		want  string
	}{
		{"aaaaa", Rule{"aa", "b"}, "bba"},
		{"banana", Rule{"ana", "x"}, "bxna"},
		{"abc", Rule{"z", "q"}, "abc"},
		{"ababa", Rule{"ab", ""}, "a"},
	}
	for _, test := range tests {
		got, err := Apply(test.input, test.rule)
		if err != nil || got != test.want {
			t.Fatalf("Apply(%q,%#v) = (%q,%v), want %q", test.input, test.rule, got, err, test.want)
		}
	}
}

func TestSequenceIsOrderedAndAssociativeAsApplication(t *testing.T) {
	a := Rule{"ab", "x"}
	b := Rule{"xc", "y"}
	forward, err := ApplySequence("abc", []Rule{a, b})
	if err != nil || forward != "y" {
		t.Fatalf("forward = (%q,%v)", forward, err)
	}
	reverse, err := ApplySequence("abc", []Rule{b, a})
	if err != nil || reverse != "xc" {
		t.Fatalf("reverse = (%q,%v)", reverse, err)
	}
	first, _ := Apply("abcabc", a)
	sequential, _ := Apply(first, b)
	composed, _ := ApplySequence("abcabc", []Rule{a, b})
	if sequential != composed || composed != "yy" {
		t.Fatalf("sequential=%q composed=%q", sequential, composed)
	}
}

func TestApplyRejectsMalformedAndOverflowingValues(t *testing.T) {
	for _, test := range []struct {
		input string
		rule  Rule
	}{
		{"A", Rule{"a", "b"}},
		{"a", Rule{"", "b"}},
		{strings.Repeat("a", 64), Rule{"a", "aaaaaaaa"}},
	} {
		if _, err := Apply(test.input, test.rule); err == nil {
			t.Fatalf("Apply(%q,%#v) unexpectedly succeeded", test.input, test.rule)
		}
	}
}

func TestApplyDifferentialAgainstReferenceScanner(t *testing.T) {
	rules := []Rule{{"a", ""}, {"a", "bb"}, {"ab", "x"}, {"aa", "b"}, {"ba", "ab"}}
	inputs := []string{"", "a", "aa", "aaa", "ababa", "bbaabb", "cccc"}
	for _, input := range inputs {
		for _, rule := range rules {
			got, err := Apply(input, rule)
			want := scanReference(input, rule.Left, rule.Right)
			if err != nil || got != want {
				t.Fatalf("Apply(%q,%#v) = (%q,%v), reference=%q", input, rule, got, err, want)
			}
		}
	}
}

func scanReference(input, left, right string) string {
	var out strings.Builder
	for position := 0; position < len(input); {
		if strings.HasPrefix(input[position:], left) {
			out.WriteString(right)
			position += len(left)
		} else {
			out.WriteByte(input[position])
			position++
		}
	}
	return out.String()
}
