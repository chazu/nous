package dsl

import (
	"encoding/base64"
	"fmt"
	"strings"

	rewritevocab "github.com/chazu/nous/internal/vocab/rewrite"
)

func init() {
	registerVocabularyWords("rewrite", map[string]builtinFn{
		"rewrite-valid?":        bRewriteValid,
		"rewrite-rule-valid?":   bRewriteRuleValid,
		"rewrite-replace-all":   bRewriteReplaceAll,
		"rewrite-rule-applies?": bRewriteRuleApplies,
		"rewrite-output-length": bRewriteOutputLength,
		"rewrite-compose-name":  bRewriteComposeName,
		"rewrite-artifact-name": bRewriteArtifactName,
	})
}

func rewriteString(value Value) (string, bool) {
	if value.Kind() != VString {
		return "", false
	}
	text := value.AsString()
	return text, rewritevocab.ValidText(text)
}

func rewriteRule(left, right Value) (rewritevocab.Rule, bool) {
	if left.Kind() != VString || right.Kind() != VString {
		return rewritevocab.Rule{}, false
	}
	rule := rewritevocab.Rule{Left: left.AsString(), Right: right.AsString()}
	return rule, rewritevocab.ValidRule(rule)
}

func bRewriteValid(vm *VM) error {
	_, ok := rewriteString(vm.pop())
	vm.push(BoolVal(ok))
	return nil
}

func bRewriteRuleValid(vm *VM) error {
	right, left := vm.pop(), vm.pop()
	_, ok := rewriteRule(left, right)
	vm.push(BoolVal(ok))
	return nil
}

func bRewriteReplaceAll(vm *VM) error {
	right, left, input := vm.pop(), vm.pop(), vm.pop()
	text, textOK := rewriteString(input)
	rule, ruleOK := rewriteRule(left, right)
	if !textOK || !ruleOK {
		vm.push(Nil())
		return nil
	}
	result, err := rewritevocab.Apply(text, rule)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(result))
	return nil
}

func bRewriteRuleApplies(vm *VM) error {
	left, input := vm.pop(), vm.pop()
	text, textOK := rewriteString(input)
	rule, ruleOK := rewriteRule(left, StringVal(""))
	if !textOK || !ruleOK {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(strings.Contains(text, rule.Left)))
	return nil
}

func bRewriteOutputLength(vm *VM) error {
	text, ok := rewriteString(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(IntVal(len(text)))
	return nil
}

func encodeRewriteIdentity(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func freshRewriteName(vm *VM, base string) string {
	if !vm.Store.Has(base) {
		return base
	}
	for suffix := 1; ; suffix++ {
		candidate := fmt.Sprintf("%s-collision-%d", base, suffix)
		if !vm.Store.Has(candidate) {
			return candidate
		}
	}
}

func bRewriteComposeName(vm *VM) error {
	second, first := vm.pop().AsString(), vm.pop().AsString()
	base := "Compose." + encodeRewriteIdentity(first) + "." + encodeRewriteIdentity(second)
	vm.push(StringVal(freshRewriteName(vm, base)))
	return nil
}

func bRewriteArtifactName(vm *VM) error {
	example, program, kind := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	base := "RewriteArtifact." + encodeRewriteIdentity(kind) + "." +
		encodeRewriteIdentity(program) + "." + encodeRewriteIdentity(example)
	vm.push(StringVal(freshRewriteName(vm, base)))
	return nil
}
