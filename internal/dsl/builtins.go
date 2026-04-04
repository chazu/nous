package dsl

import (
	"fmt"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

type builtinFn func(vm *VM) error

var builtins = map[string]builtinFn{
	// Constants
	"true":  func(vm *VM) error { vm.push(BoolVal(true)); return nil },
	"false": func(vm *VM) error { vm.push(BoolVal(false)); return nil },
	"nil":   func(vm *VM) error { vm.push(Nil()); return nil },

	// Stack ops
	"dup":  bDup,
	"drop": bDrop,
	"swap": bSwap,
	"over": bOver,
	"rot":  bRot,

	// Arithmetic
	"+":   bAdd,
	"-":   bSub,
	"*":   bMul,
	"/":   bDiv,
	"mod": bMod,

	// Comparison
	"=":  bEq,
	"!=": bNeq,
	"<":  bLt,
	">":  bGt,
	"<=": bLte,
	">=": bGte,

	// Logic
	"and": bAnd,
	"or":  bOr,
	"not": bNot,

	// Variables
	"!": bStore,
	"@": bFetch,

	// Unit/store ops
	"get-slot":    bGetSlot,
	"set-slot":    bSetSlot,
	"isa?":        bIsA,
	"examples":    bExamples,
	"create-unit": bCreateUnit,
	"kill-unit":   bKillUnit,
	"unit-exists?": bUnitExists,

	// Agenda
	"add-task": bAddTask,

	// String
	"concat":    bConcat,
	"pack-name": bPackName,

	// List
	"list-length": bListLength,
	"list-append": bListAppend,
	"list-contains": bListContains,
	"to-string-list": bToStringList,

	// Output
	"print": bPrint,
	".s":    bDotS, // debug: print stack

	// Loop variable
	"it": func(vm *VM) error { vm.push(vm.env["it"]); return nil },

	// Misc
	"noop": func(vm *VM) error { return nil },
}

// Stack ops

func bDup(vm *VM) error  { v := vm.peek(); vm.push(v); return nil }
func bDrop(vm *VM) error { vm.pop(); return nil }
func bSwap(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(b)
	vm.push(a)
	return nil
}
func bOver(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(a)
	vm.push(b)
	vm.push(a)
	return nil
}
func bRot(vm *VM) error {
	c, b, a := vm.pop(), vm.pop(), vm.pop()
	vm.push(b)
	vm.push(c)
	vm.push(a)
	return nil
}

// Arithmetic

func bAdd(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	if a.kind == VFloat || b.kind == VFloat {
		vm.push(FloatVal(a.AsFloat() + b.AsFloat()))
	} else {
		vm.push(IntVal(a.AsInt() + b.AsInt()))
	}
	return nil
}

func bSub(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	if a.kind == VFloat || b.kind == VFloat {
		vm.push(FloatVal(a.AsFloat() - b.AsFloat()))
	} else {
		vm.push(IntVal(a.AsInt() - b.AsInt()))
	}
	return nil
}

func bMul(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	if a.kind == VFloat || b.kind == VFloat {
		vm.push(FloatVal(a.AsFloat() * b.AsFloat()))
	} else {
		vm.push(IntVal(a.AsInt() * b.AsInt()))
	}
	return nil
}

func bDiv(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	if b.AsInt() == 0 && b.AsFloat() == 0 {
		vm.push(IntVal(0))
		return nil
	}
	if a.kind == VFloat || b.kind == VFloat {
		vm.push(FloatVal(a.AsFloat() / b.AsFloat()))
	} else {
		vm.push(IntVal(a.AsInt() / b.AsInt()))
	}
	return nil
}

func bMod(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	if b.AsInt() == 0 {
		vm.push(IntVal(0))
		return nil
	}
	vm.push(IntVal(a.AsInt() % b.AsInt()))
	return nil
}

// Comparison

func bEq(vm *VM) error  { b, a := vm.pop(), vm.pop(); vm.push(BoolVal(a.Equal(b))); return nil }
func bNeq(vm *VM) error { b, a := vm.pop(), vm.pop(); vm.push(BoolVal(!a.Equal(b))); return nil }
func bLt(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.AsInt() < b.AsInt()))
	return nil
}
func bGt(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.AsInt() > b.AsInt()))
	return nil
}
func bLte(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.AsInt() <= b.AsInt()))
	return nil
}
func bGte(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.AsInt() >= b.AsInt()))
	return nil
}

// Logic

func bAnd(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.Truthy() && b.Truthy()))
	return nil
}
func bOr(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(a.Truthy() || b.Truthy()))
	return nil
}
func bNot(vm *VM) error {
	a := vm.pop()
	vm.push(BoolVal(!a.Truthy()))
	return nil
}

// Variables

func bStore(vm *VM) error {
	name := vm.pop()
	value := vm.pop()
	vm.env[name.AsString()] = value
	return nil
}

func bFetch(vm *VM) error {
	name := vm.pop()
	v, ok := vm.env[name.AsString()]
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(v)
	}
	return nil
}

// Unit/store ops

func bGetSlot(vm *VM) error {
	slotName := vm.pop()
	unitName := vm.pop()
	u := vm.Store.Get(unitName.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	raw := u.Get(slotName.AsString())
	vm.push(anyToValue(raw))
	return nil
}

func bSetSlot(vm *VM) error {
	slotName := vm.pop()
	unitName := vm.pop()
	value := vm.pop()
	u := vm.Store.Get(unitName.AsString())
	if u == nil {
		return fmt.Errorf("set-slot: unit %q not found", unitName.AsString())
	}
	slot := slotName.AsString()
	// Creditors is always stored as []string
	if slot == "creditors" && value.kind == VString {
		u.Set(slot, []string{value.sval})
	} else {
		u.Set(slot, valueToAny(value))
	}
	return nil
}

func bIsA(vm *VM) error {
	category := vm.pop()
	unitName := vm.pop()
	vm.push(BoolVal(vm.Store.IsA(unitName.AsString(), category.AsString())))
	return nil
}

func bExamples(vm *VM) error {
	category := vm.pop()
	names := vm.Store.Examples(category.AsString())
	vals := make([]Value, len(names))
	for i, n := range names {
		vals[i] = StringVal(n)
	}
	vm.push(ListVal(vals))
	return nil
}

func bCreateUnit(vm *VM) error {
	parentCategory := vm.pop()
	name := vm.pop()
	nameStr := name.AsString()
	u := vm.Store.Get(nameStr)
	if u != nil {
		// Already exists
		vm.push(StringVal(nameStr))
		return nil
	}
	u = &unit.Unit{
		Name:  nameStr,
		Slots: map[string]any{},
	}
	parent := parentCategory.AsString()
	if parent != "" {
		u.Set("isA", []string{parent})
	}
	u.Set("worth", 500) // default worth for new units
	u.Set("isNew", true)
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, nameStr)
	vm.push(StringVal(nameStr))
	return nil
}

func bKillUnit(vm *VM) error {
	name := vm.pop()
	nameStr := name.AsString()
	// Snapshot the unit's slots before deletion for HindSight
	u := vm.Store.Get(nameStr)
	if u != nil {
		snapshot := make(map[string]any)
		for k, v := range u.Slots {
			snapshot[k] = v
		}
		if vm.DeletedSnapshots == nil {
			vm.DeletedSnapshots = make(map[string]map[string]any)
		}
		vm.DeletedSnapshots[nameStr] = snapshot
	}
	vm.Store.Delete(nameStr)
	vm.DeletedUnits = append(vm.DeletedUnits, nameStr)
	return nil
}

func bUnitExists(vm *VM) error {
	name := vm.pop()
	vm.push(BoolVal(vm.Store.Has(name.AsString())))
	return nil
}

// Agenda

func bAddTask(vm *VM) error {
	reason := vm.pop()
	slotName := vm.pop()
	unitName := vm.pop()
	priority := vm.pop()
	vm.Ag.Push(&agenda.Task{
		Priority: priority.AsInt(),
		UnitName: unitName.AsString(),
		SlotName: slotName.AsString(),
		Reasons:  []string{reason.AsString()},
	})
	return nil
}

// String ops

func bConcat(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(StringVal(a.AsString() + b.AsString()))
	return nil
}

func bPackName(vm *VM) error {
	name := vm.pop()
	prefix := vm.pop()
	vm.push(StringVal(prefix.AsString() + "-" + name.AsString()))
	return nil
}

// List ops

func bListLength(vm *VM) error {
	v := vm.pop()
	vm.push(IntVal(len(v.AsList())))
	return nil
}

func bListAppend(vm *VM) error {
	elem := vm.pop()
	list := vm.pop()
	items := list.AsList()
	newItems := make([]Value, len(items)+1)
	copy(newItems, items)
	newItems[len(items)] = elem
	vm.push(ListVal(newItems))
	return nil
}

func bListContains(vm *VM) error {
	target := vm.pop()
	list := vm.pop()
	for _, item := range list.AsList() {
		if item.Equal(target) {
			vm.push(BoolVal(true))
			return nil
		}
	}
	vm.push(BoolVal(false))
	return nil
}

func bToStringList(vm *VM) error {
	v := vm.pop()
	switch v.kind {
	case VList:
		vm.push(v) // already a list
	default:
		vm.push(ListVal([]Value{v}))
	}
	return nil
}

// Output

func bPrint(vm *VM) error {
	v := vm.pop()
	fmt.Fprintln(vm.Out, v.AsString())
	return nil
}

func bDotS(vm *VM) error {
	fmt.Fprintf(vm.Out, "<%d> ", len(vm.stack))
	for _, v := range vm.stack {
		fmt.Fprintf(vm.Out, "%s ", v.String())
	}
	fmt.Fprintln(vm.Out)
	return nil
}

// Conversion helpers

func anyToValue(v any) Value {
	switch x := v.(type) {
	case nil:
		return Nil()
	case bool:
		return BoolVal(x)
	case int:
		return IntVal(x)
	case float64:
		return FloatVal(x)
	case string:
		return StringVal(x)
	case []string:
		vals := make([]Value, len(x))
		for i, s := range x {
			vals[i] = StringVal(s)
		}
		return ListVal(vals)
	case []int:
		vals := make([]Value, len(x))
		for i, n := range x {
			vals[i] = IntVal(n)
		}
		return ListVal(vals)
	case []Value:
		return ListVal(x)
	case []map[string]any:
		// Structured examples — return count for now
		return IntVal(len(x))
	case map[string]any:
		// Represent as a list of key-value pairs for now
		// Individual fields are accessed via get-slot on the unit
		return StringVal(fmt.Sprintf("%v", x))
	default:
		return StringVal(fmt.Sprintf("%v", x))
	}
}

func valueToAny(v Value) any {
	switch v.kind {
	case VNil:
		return nil
	case VBool:
		return v.bval
	case VInt:
		return v.ival
	case VFloat:
		return v.fval
	case VString:
		return v.sval
	case VList:
		// Convert to []string if all elements are strings
		strs := make([]string, 0, len(v.lval))
		for _, el := range v.lval {
			if el.kind != VString {
				// Mixed list — return as []Value
				return v.lval
			}
			strs = append(strs, el.sval)
		}
		return strs
	default:
		return nil
	}
}
