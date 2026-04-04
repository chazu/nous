package dsl

import (
	"math"
	"sort"
)

func init() {
	// Number predicates
	builtins["even?"] = bIsEven
	builtins["odd?"] = bIsOdd
	builtins["prime?"] = bIsPrime
	builtins["zero?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().AsInt() == 0)); return nil }
	builtins["positive?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().AsInt() > 0)); return nil }

	// Number operations
	builtins["abs"] = func(vm *VM) error {
		n := vm.pop().AsInt()
		if n < 0 {
			n = -n
		}
		vm.push(IntVal(n))
		return nil
	}
	builtins["max"] = func(vm *VM) error {
		b, a := vm.pop().AsInt(), vm.pop().AsInt()
		if a > b {
			vm.push(IntVal(a))
		} else {
			vm.push(IntVal(b))
		}
		return nil
	}
	builtins["min"] = func(vm *VM) error {
		b, a := vm.pop().AsInt(), vm.pop().AsInt()
		if a < b {
			vm.push(IntVal(a))
		} else {
			vm.push(IntVal(b))
		}
		return nil
	}
	builtins["divisors"] = bDivisors
	builtins["gcd"] = bGCD
	builtins["range"] = bRange // ( start end -- list )

	// Set operations (lists treated as sets — unique, unordered)
	builtins["set-union"] = bSetUnion
	builtins["set-intersect"] = bSetIntersect
	builtins["set-diff"] = bSetDiff
	builtins["set-member?"] = bSetMember
	builtins["set-subset?"] = bSetSubset
	builtins["set-equal?"] = bSetEqual
	builtins["set-size"] = func(vm *VM) error { vm.push(IntVal(len(vm.pop().AsList()))); return nil }
	builtins["make-set"] = bMakeSet // ( list -- set ) deduplicate + sort
	builtins["set-empty?"] = func(vm *VM) error { vm.push(BoolVal(len(vm.pop().AsList()) == 0)); return nil }

	// List operations
	builtins["first"] = bFirst
	builtins["rest"] = bRest
	builtins["last"] = bLast
	builtins["reverse"] = bReverse
	builtins["sort"] = bSort
	builtins["list-empty?"] = func(vm *VM) error { vm.push(BoolVal(len(vm.pop().AsList()) == 0)); return nil }
	builtins["list-get"] = bListGet // ( list index -- element )
	builtins["list-take"] = bListTake
	builtins["list-filter-gt"] = bListFilterGt // ( list threshold -- filtered )

	// Constructors
	builtins["iota"] = bIota // ( n -- [0 1 2 ... n-1] )

	// Functional
	builtins["apply-op"] = bApplyOp // ( arg1 arg2 opUnitName -- result ) run defn slot
	builtins["apply-pred"] = bApplyPred
}

func bIsEven(vm *VM) error {
	vm.push(BoolVal(vm.pop().AsInt()%2 == 0))
	return nil
}

func bIsOdd(vm *VM) error {
	vm.push(BoolVal(vm.pop().AsInt()%2 != 0))
	return nil
}

func bIsPrime(vm *VM) error {
	n := vm.pop().AsInt()
	if n < 2 {
		vm.push(BoolVal(false))
		return nil
	}
	for i := 2; i <= int(math.Sqrt(float64(n))); i++ {
		if n%i == 0 {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

func bDivisors(vm *VM) error {
	n := vm.pop().AsInt()
	if n <= 0 {
		vm.push(ListVal(nil))
		return nil
	}
	var divs []Value
	for i := 1; i <= n; i++ {
		if n%i == 0 {
			divs = append(divs, IntVal(i))
		}
	}
	vm.push(ListVal(divs))
	return nil
}

func bGCD(vm *VM) error {
	b, a := vm.pop().AsInt(), vm.pop().AsInt()
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		a = -a
	}
	vm.push(IntVal(a))
	return nil
}

func bRange(vm *VM) error {
	end := vm.pop().AsInt()
	start := vm.pop().AsInt()
	var vals []Value
	for i := start; i < end; i++ {
		vals = append(vals, IntVal(i))
	}
	vm.push(ListVal(vals))
	return nil
}

func bIota(vm *VM) error {
	n := vm.pop().AsInt()
	vals := make([]Value, n)
	for i := 0; i < n; i++ {
		vals[i] = IntVal(i)
	}
	vm.push(ListVal(vals))
	return nil
}

// Set operations — sets are represented as sorted deduplicated lists of ints

func toIntSet(v Value) []int {
	list := v.AsList()
	seen := make(map[int]bool)
	var result []int
	for _, el := range list {
		n := el.AsInt()
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	sort.Ints(result)
	return result
}

func intSetToValue(s []int) Value {
	vals := make([]Value, len(s))
	for i, n := range s {
		vals[i] = IntVal(n)
	}
	return ListVal(vals)
}

func bSetUnion(vm *VM) error {
	b, a := toIntSet(vm.pop()), toIntSet(vm.pop())
	seen := make(map[int]bool)
	for _, n := range a {
		seen[n] = true
	}
	for _, n := range b {
		seen[n] = true
	}
	var result []int
	for n := range seen {
		result = append(result, n)
	}
	sort.Ints(result)
	vm.push(intSetToValue(result))
	return nil
}

func bSetIntersect(vm *VM) error {
	b, a := toIntSet(vm.pop()), toIntSet(vm.pop())
	bSet := make(map[int]bool)
	for _, n := range b {
		bSet[n] = true
	}
	var result []int
	for _, n := range a {
		if bSet[n] {
			result = append(result, n)
		}
	}
	sort.Ints(result)
	vm.push(intSetToValue(result))
	return nil
}

func bSetDiff(vm *VM) error {
	b, a := toIntSet(vm.pop()), toIntSet(vm.pop())
	bSet := make(map[int]bool)
	for _, n := range b {
		bSet[n] = true
	}
	var result []int
	for _, n := range a {
		if !bSet[n] {
			result = append(result, n)
		}
	}
	sort.Ints(result)
	vm.push(intSetToValue(result))
	return nil
}

func bSetMember(vm *VM) error {
	set := toIntSet(vm.pop())
	elem := vm.pop().AsInt()
	for _, n := range set {
		if n == elem {
			vm.push(BoolVal(true))
			return nil
		}
	}
	vm.push(BoolVal(false))
	return nil
}

func bSetSubset(vm *VM) error {
	super := toIntSet(vm.pop())
	sub := toIntSet(vm.pop())
	superSet := make(map[int]bool)
	for _, n := range super {
		superSet[n] = true
	}
	for _, n := range sub {
		if !superSet[n] {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

func bSetEqual(vm *VM) error {
	b, a := toIntSet(vm.pop()), toIntSet(vm.pop())
	if len(a) != len(b) {
		vm.push(BoolVal(false))
		return nil
	}
	for i := range a {
		if a[i] != b[i] {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

func bMakeSet(vm *VM) error {
	vm.push(intSetToValue(toIntSet(vm.pop())))
	return nil
}

// List operations

func bFirst(vm *VM) error {
	list := vm.pop().AsList()
	if len(list) == 0 {
		vm.push(Nil())
	} else {
		vm.push(list[0])
	}
	return nil
}

func bRest(vm *VM) error {
	list := vm.pop().AsList()
	if len(list) <= 1 {
		vm.push(ListVal(nil))
	} else {
		cp := make([]Value, len(list)-1)
		copy(cp, list[1:])
		vm.push(ListVal(cp))
	}
	return nil
}

func bLast(vm *VM) error {
	list := vm.pop().AsList()
	if len(list) == 0 {
		vm.push(Nil())
	} else {
		vm.push(list[len(list)-1])
	}
	return nil
}

func bReverse(vm *VM) error {
	list := vm.pop().AsList()
	n := len(list)
	rev := make([]Value, n)
	for i, v := range list {
		rev[n-1-i] = v
	}
	vm.push(ListVal(rev))
	return nil
}

func bSort(vm *VM) error {
	vm.push(intSetToValue(toIntSet(vm.pop()))) // reuses toIntSet which sorts
	return nil
}

func bListGet(vm *VM) error {
	idx := vm.pop().AsInt()
	list := vm.pop().AsList()
	if idx < 0 || idx >= len(list) {
		vm.push(Nil())
	} else {
		vm.push(list[idx])
	}
	return nil
}

func bListTake(vm *VM) error {
	n := vm.pop().AsInt()
	list := vm.pop().AsList()
	if n > len(list) {
		n = len(list)
	}
	if n <= 0 {
		vm.push(ListVal(nil))
		return nil
	}
	cp := make([]Value, n)
	copy(cp, list[:n])
	vm.push(ListVal(cp))
	return nil
}

func bListFilterGt(vm *VM) error {
	threshold := vm.pop().AsInt()
	list := vm.pop().AsList()
	var result []Value
	for _, v := range list {
		if v.AsInt() > threshold {
			result = append(result, v)
		}
	}
	vm.push(ListVal(result))
	return nil
}

// apply-op: ( arg1 arg2 opUnitName -- result ) for binary ops
//           ( arg1 opUnitName -- result ) for unary ops
// Looks up the "defn" slot of opUnitName and executes it with args.
// Checks the unit's isA to determine arity.
func bApplyOp(vm *VM) error {
	opName := vm.pop().AsString()
	u := vm.Store.Get(opName)
	if u == nil {
		vm.push(Nil())
		return nil
	}
	defn := u.GetString("defn")
	if defn == "" {
		vm.push(Nil())
		return nil
	}

	sub := NewVM(vm.Store, vm.Ag)
	sub.Out = vm.Out
	for k, v := range vm.env {
		sub.env[k] = v
	}

	if vm.Store.IsA(opName, "BinaryOp") || vm.Store.IsA(opName, "BinaryPred") {
		arg2 := vm.pop()
		arg1 := vm.pop()
		sub.stack = append(sub.stack, arg1, arg2)
	} else {
		arg1 := vm.pop()
		sub.stack = append(sub.stack, arg1)
	}

	result, err := subExecute(sub, defn)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(result)
	return nil
}

// apply-pred: ( args... predUnitName -- bool )
func bApplyPred(vm *VM) error {
	return bApplyOp(vm)
}

// subExecute runs a program on a fresh VM and returns the top of stack.
func subExecute(sub *VM, program string) (Value, error) {
	tokens := Tokenize(program)
	err := sub.run(tokens, 0, len(tokens))
	if err != nil {
		return Nil(), err
	}
	if len(sub.stack) == 0 {
		return Nil(), nil
	}
	return sub.stack[len(sub.stack)-1], nil
}
