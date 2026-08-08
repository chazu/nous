package dsl

import (
	"encoding/base64"

	kuberepair "github.com/chazu/nous/internal/vocab/kuberepair"
)

func init() {
	registerVocabularyWords("kuberepair", map[string]builtinFn{
		"kube-bundle-valid?":       bKubeBundleValid,
		"kube-repair-value-valid?": bKubeRepairValueValid,
		"kube-edit-valid?":         bKubeEditValid,
		"kube-enumerate-edits":     bKubeEnumerateEdits,
		"kube-apply-edit":          bKubeApplyEdit,
		"kube-apply-edit-b64":      bKubeApplyEditB64,
		"kube-value-matches?":      bKubeValueMatches,
	})
}

func bKubeApplyEditB64(vm *VM) error {
	encoded, encodedOK := strictString(vm.pop())
	bundle, bundleOK := strictString(vm.pop())
	if !encodedOK || !bundleOK {
		vm.push(Nil())
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	result, err := kuberepair.Apply(bundle, string(decoded))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(result))
	return nil
}

func bKubeBundleValid(vm *VM) error {
	value, ok := strictString(vm.pop())
	vm.push(BoolVal(ok && kuberepair.ValidBundle(value)))
	return nil
}

func bKubeRepairValueValid(vm *VM) error {
	value, ok := strictString(vm.pop())
	vm.push(BoolVal(ok && kuberepair.ValidValue(value)))
	return nil
}

func bKubeEditValid(vm *VM) error {
	value, ok := strictString(vm.pop())
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	_, err := kuberepair.DecodeEdit(value)
	vm.push(BoolVal(err == nil))
	return nil
}

func bKubeEnumerateEdits(vm *VM) error {
	value, ok := strictString(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	edits, err := kuberepair.EnumerateEdits(value)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(stringListValue(edits))
	return nil
}

func bKubeApplyEdit(vm *VM) error {
	edit, editOK := strictString(vm.pop())
	bundle, bundleOK := strictString(vm.pop())
	if !editOK || !bundleOK {
		vm.push(Nil())
		return nil
	}
	result, err := kuberepair.Apply(bundle, edit)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(result))
	return nil
}

func bKubeValueMatches(vm *VM) error {
	right, rightOK := strictString(vm.pop())
	left, leftOK := strictString(vm.pop())
	if !leftOK || !rightOK {
		vm.push(BoolVal(false))
		return nil
	}
	vm.push(BoolVal(kuberepair.EqualOrSatisfies(left, right)))
	return nil
}
