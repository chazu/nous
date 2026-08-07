package dsl

import protocolvocab "github.com/chazu/nous/internal/vocab/protocol"

func init() {
	registerVocabularyWords("protocol", map[string]builtinFn{
		"protocol-valid?":                bProtocolValid,
		"protocol-canonicalize":          bProtocolCanonicalize,
		"protocol-reachable-states":      bProtocolReachableStates,
		"protocol-rejecting-trap-states": bProtocolRejectingTrapStates,
		"protocol-remove-unreachable":    bProtocolRemoveUnreachable,
		"protocol-drop-first-transition": bProtocolDropFirstTransition,
		"protocol-accepts?":              bProtocolAccepts,
		"protocol-equivalent?":           bProtocolEquivalent,
		"protocol-same-encoding?":        bProtocolSameEncoding,
	})
}

func protocolStrings(value Value) ([]string, bool) {
	if value.Kind() != VList {
		return nil, false
	}
	items := value.AsList()
	out := make([]string, len(items))
	for i, item := range items {
		if item.Kind() != VString {
			return nil, false
		}
		out[i] = item.AsString()
	}
	return out, true
}

func protocolList(values []string) Value {
	out := make([]Value, len(values))
	for i, value := range values {
		out[i] = StringVal(value)
	}
	return ListVal(out)
}

func parseProtocolValue(value Value) (protocolvocab.Machine, bool) {
	records, ok := protocolStrings(value)
	if !ok {
		return protocolvocab.Machine{}, false
	}
	machine, err := protocolvocab.Parse(records)
	return machine, err == nil
}

func bProtocolValid(vm *VM) error {
	_, ok := parseProtocolValue(vm.pop())
	vm.push(BoolVal(ok))
	return nil
}

func bProtocolCanonicalize(vm *VM) error {
	machine, ok := parseProtocolValue(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(protocolList(machine.Records()))
	return nil
}

func bProtocolReachableStates(vm *VM) error {
	machine, ok := parseProtocolValue(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(protocolList(machine.ReachableStates()))
	return nil
}

func bProtocolRejectingTrapStates(vm *VM) error {
	machine, ok := parseProtocolValue(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(protocolList(machine.RejectingTrapStates()))
	return nil
}

func bProtocolRemoveUnreachable(vm *VM) error {
	machine, ok := parseProtocolValue(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(protocolList(machine.TrimUnreachable().Records()))
	return nil
}

func bProtocolDropFirstTransition(vm *VM) error {
	machine, ok := parseProtocolValue(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(protocolList(machine.DropFirstTransition().Records()))
	return nil
}

func bProtocolAccepts(vm *VM) error {
	trace, traceOK := protocolStrings(vm.pop())
	machine, machineOK := parseProtocolValue(vm.pop())
	if !traceOK || !machineOK {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(machine.Accepts(trace)))
	return nil
}

func bProtocolEquivalent(vm *VM) error {
	b, bOK := parseProtocolValue(vm.pop())
	a, aOK := parseProtocolValue(vm.pop())
	if !aOK || !bOK {
		vm.push(Nil())
		return nil
	}
	equivalent, _ := protocolvocab.Compare(a, b)
	vm.push(BoolVal(equivalent))
	return nil
}

func bProtocolSameEncoding(vm *VM) error {
	b, bOK := parseProtocolValue(vm.pop())
	a, aOK := parseProtocolValue(vm.pop())
	if !aOK || !bOK {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(protocolvocab.SameEncoding(a, b)))
	return nil
}
