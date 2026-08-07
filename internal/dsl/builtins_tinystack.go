package dsl

import (
	"reflect"

	stackvocab "github.com/chazu/nous/internal/vocab/tinystack"
)

func init() {
	registerVocabularyWords("stack", map[string]builtinFn{
		"stack-valid?":       bStackValid,
		"stack-input-valid?": bStackInputValid,
		"stack-exec-op":      bStackExecOp,
		"stack-equal?":       bStackEqual,
	})
}

func strictIntList(value Value) ([]int, bool) {
	if value.Kind() != VList {
		return nil, false
	}
	items := value.AsList()
	out := make([]int, len(items))
	for index, item := range items {
		if item.Kind() != VInt {
			return nil, false
		}
		out[index] = item.AsInt()
	}
	return out, true
}

func intListValue(values []int) Value {
	out := make([]Value, len(values))
	for index, value := range values {
		out[index] = IntVal(value)
	}
	return ListVal(out)
}

func bStackValid(vm *VM) error {
	stack, ok := strictIntList(vm.pop())
	vm.push(BoolVal(ok && stackvocab.ValidStack(stack)))
	return nil
}

func bStackInputValid(vm *VM) error {
	stack, ok := strictIntList(vm.pop())
	vm.push(BoolVal(ok && stackvocab.ValidInput(stack)))
	return nil
}

func bStackExecOp(vm *VM) error {
	opcode, opcodeOK := strictString(vm.pop())
	stack, stackOK := strictIntList(vm.pop())
	if !opcodeOK || !stackOK {
		vm.push(Nil())
		return nil
	}
	result, err := stackvocab.Step(stack, stackvocab.Opcode(opcode))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(intListValue(result))
	return nil
}

func bStackEqual(vm *VM) error {
	right, rightOK := strictIntList(vm.pop())
	left, leftOK := strictIntList(vm.pop())
	if !rightOK || !leftOK || !stackvocab.ValidStack(left) || !stackvocab.ValidStack(right) {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(reflect.DeepEqual(left, right)))
	return nil
}
