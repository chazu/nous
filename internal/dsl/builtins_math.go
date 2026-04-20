package dsl

import (
	"fmt"
	"math"
	"sort"

	"github.com/chazu/nous/internal/unit"
)

func init() {
	// Number predicates
	builtins["even?"] = bIsEven
	builtins["odd?"] = bIsOdd
	builtins["prime?"] = bIsPrime
	builtins["zero?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().AsInt() == 0)); return nil }
	builtins["positive?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().AsInt() > 0)); return nil }

	// Type predicates (Phase 5.6 slice C.1)
	builtins["is-int?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().Kind() == VInt)); return nil }
	builtins["is-list?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().Kind() == VList)); return nil }
	builtins["is-string?"] = func(vm *VM) error { vm.push(BoolVal(vm.pop().Kind() == VString)); return nil }

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
	builtins["apply-op-args"] = bApplyOpArgs // ( argList opName -- result )
	builtins["apply-pred"] = bApplyPred
	builtins["run-generator"] = bRunGenerator
	builtins["transpose-op"] = bTransposeOp
	builtins["compose-ops"] = bComposeOps
	builtins["applics-redundant?"] = bApplicsRedundant
}

// run-generator: (unitName count -- list)
// Returns up to `count` values produced by iterating the unit's `generator`
// slot. Generator format: map{"initial": []any, "step": "<dsl program>"}.
// Initial values appear first; if fewer than count, the step program is run
// repeatedly with the previous value on the stack and its top-of-stack
// result is appended. Returns empty list if the unit has no generator or
// if initial is empty and step fails.
func bRunGenerator(vm *VM) error {
	count := vm.pop().AsInt()
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(ListVal(nil))
		return nil
	}
	raw := u.Get("generator")
	gen, ok := raw.(map[string]any)
	if !ok {
		vm.push(ListVal(nil))
		return nil
	}
	stepProg, _ := gen["step"].(string)
	var out []Value
	switch init := gen["initial"].(type) {
	case []any:
		for _, v := range init {
			out = append(out, anyToValue(v))
		}
	case []int:
		for _, v := range init {
			out = append(out, IntVal(v))
		}
	}
	for len(out) < count && stepProg != "" && len(out) > 0 {
		last := out[len(out)-1]
		sub := NewVM(vm.Store, vm.Ag, vm.Rng)
		sub.Out = vm.Out
		for k, val := range vm.env {
			sub.env[k] = val
		}
		sub.stack = append(sub.stack, last)
		result, err := subExecute(sub, stepProg)
		if err != nil {
			break
		}
		out = append(out, result)
	}
	if len(out) > count {
		out = out[:count]
	}
	vm.push(ListVal(out))
	return nil
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

	sub := NewVM(vm.Store, vm.Ag, vm.Rng)
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
	// Phase 5.10: if this op is a Pred, update its Rarity record
	// [freqT, numT, numF] so H24 and interestingness heuristics can find
	// rare predicates. Applies regardless of arity.
	if vm.Store.IsA(opName, "Pred") {
		updateRarity(vm.Store, opName, result.Truthy())
	}
	vm.push(result)
	return nil
}

// updateRarity increments the True or False counter on a predicate unit's
// Rarity slot and recomputes frequency-True. The slot value is stored as
// a 3-element []any: [freqTrue (float64), numT (int), numF (int)].
func updateRarity(store *unit.Store, predName string, truthy bool) {
	u := store.Get(predName)
	if u == nil {
		return
	}
	numT, numF := 0, 0
	if existing, ok := u.Get("rarity").([]any); ok && len(existing) == 3 {
		numT = toInt(existing[1])
		numF = toInt(existing[2])
	}
	if truthy {
		numT++
	} else {
		numF++
	}
	total := numT + numF
	freq := 0.0
	if total > 0 {
		freq = float64(numT) / float64(total)
	}
	u.Set("rarity", []any{freq, numT, numF})
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
}

// apply-pred: ( args... predUnitName -- bool )
func bApplyPred(vm *VM) error {
	return bApplyOp(vm)
}

// apply-op-args: ( argList opName -- result )
// Applies opName's defn to args looked up from the unit names in argList.
// Each arg name is resolved to its `data` slot value; those values are
// pushed onto a sub-VM and opName's defn is executed. Used by H20 to
// cross-apply an op to arg tuples recorded on sibling ops' applics.
//
// Returns Nil on: missing unit, empty defn, missing data for any arg,
// or errors during defn execution.
func bApplyOpArgs(vm *VM) error {
	opName := vm.pop().AsString()
	argList := vm.pop()
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
	sub := NewVM(vm.Store, vm.Ag, vm.Rng)
	sub.Out = vm.Out
	for k, v := range vm.env {
		sub.env[k] = v
	}
	for _, argName := range argList.AsList() {
		argUnit := vm.Store.Get(argName.AsString())
		if argUnit == nil {
			vm.push(Nil())
			return nil
		}
		data := argUnit.Get("data")
		if data == nil {
			vm.push(Nil())
			return nil
		}
		sub.stack = append(sub.stack, anyToValue(data))
	}
	result, err := subExecute(sub, defn)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(result)
	return nil
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

// transpose-op: ( opName -- newOpName | nil )
// Creates Transpose-<opName> for any BinaryOp: domain reversed, defn
// prefixed with `swap`. Idempotent — if Transpose-<op> already exists,
// returns its name without modifying. Returns nil on precondition failure
// (not a BinaryOp, missing defn, wrong-arity domain).
func bTransposeOp(vm *VM) error {
	opName := vm.pop().AsString()
	u := vm.Store.Get(opName)
	if u == nil {
		vm.push(Nil())
		return nil
	}
	if !vm.Store.IsA(opName, "BinaryOp") {
		vm.push(Nil())
		return nil
	}
	defn := u.GetString("defn")
	if defn == "" {
		vm.push(Nil())
		return nil
	}
	domain := u.GetStrings("domain")
	if len(domain) != 2 {
		vm.push(Nil())
		return nil
	}

	newName := "Transpose-" + opName
	if vm.Store.Has(newName) {
		vm.push(StringVal(newName))
		return nil
	}

	newU := unit.New(newName)
	newU.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	newU.SetWorth(500)
	newU.Set("domain", []string{domain[1], domain[0]})
	newU.Set("range", u.GetStrings("range"))
	newU.Set("defn", "swap "+defn)
	newU.Set("creditors", []string{"H-Transpose"})
	vm.Store.Put(newU)
	vm.Store.SetSlot(newName, "generalizations", []string{opName})
	vm.push(StringVal(newName))
	return nil
}

// compose-ops: ( fName gName -- newOpName | nil )
// Creates Compose-<f>-<g> when range(f) matches domain(g) as ordered
// string slices. Composed defn chains apply-op on f then g. Arity of
// the result matches f's arity; range matches g's. Idempotent.
func bComposeOps(vm *VM) error {
	gName := vm.pop().AsString()
	fName := vm.pop().AsString()
	f := vm.Store.Get(fName)
	g := vm.Store.Get(gName)
	if f == nil || g == nil {
		vm.push(Nil())
		return nil
	}
	fDefn := f.GetString("defn")
	gDefn := g.GetString("defn")
	if fDefn == "" || gDefn == "" {
		vm.push(Nil())
		return nil
	}
	fRange := f.GetStrings("range")
	gDomain := g.GetStrings("domain")
	if !stringSlicesEqual(fRange, gDomain) {
		vm.push(Nil())
		return nil
	}

	newName := "Compose-" + fName + "-" + gName
	if vm.Store.Has(newName) {
		vm.push(StringVal(newName))
		return nil
	}

	fDomain := f.GetStrings("domain")
	arityBucket := "UnaryOp"
	if len(fDomain) == 2 {
		arityBucket = "BinaryOp"
	} else if len(fDomain) != 1 {
		vm.push(Nil())
		return nil
	}

	newU := unit.New(newName)
	newU.Set("isA", []string{arityBucket, "Op", "MathOp", "Anything"})
	newU.SetWorth(500)
	newU.Set("domain", append([]string{}, fDomain...))
	newU.Set("range", append([]string{}, g.GetStrings("range")...))
	newU.Set("defn", fmt.Sprintf(`"%s" apply-op "%s" apply-op`, fName, gName))
	newU.Set("creditors", []string{"H-Compose"})
	vm.Store.Put(newU)
	vm.Store.SetSlot(newName, "generalizations", []string{fName, gName})
	vm.push(StringVal(newName))
	return nil
}

// applics-redundant?: ( unitName parentName -- bool )
// Returns true iff every applic on unitName has an output that matches
// what parentName's defn produces on the same args. Requires >=3 applics
// for meaningful evidence; returns false on sparse evidence, missing
// targets, or any mismatch.
func bApplicsRedundant(vm *VM) error {
	parentName := vm.pop().AsString()
	unitName := vm.pop().AsString()

	u := vm.Store.Get(unitName)
	parent := vm.Store.Get(parentName)
	if u == nil || parent == nil {
		vm.push(BoolVal(false))
		return nil
	}
	parentDefn := parent.GetString("defn")
	if parentDefn == "" {
		vm.push(BoolVal(false))
		return nil
	}

	applicsRaw, ok := u.Get("applics").([]map[string]any)
	if !ok || len(applicsRaw) < 3 {
		vm.push(BoolVal(false))
		return nil
	}

	for _, a := range applicsRaw {
		var args []string
		switch v := a["args"].(type) {
		case []string:
			args = v
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					args = append(args, s)
				}
			}
		}
		outName, _ := a["output"].(string)
		if len(args) == 0 || outName == "" {
			vm.push(BoolVal(false))
			return nil
		}

		sub := NewVM(vm.Store, vm.Ag, vm.Rng)
		sub.Out = vm.Out
		for k, val := range vm.env {
			sub.env[k] = val
		}
		argsOK := true
		for _, argName := range args {
			argU := vm.Store.Get(argName)
			if argU == nil {
				argsOK = false
				break
			}
			data := argU.Get("data")
			if data == nil {
				argsOK = false
				break
			}
			sub.stack = append(sub.stack, anyToValue(data))
		}
		if !argsOK {
			vm.push(BoolVal(false))
			return nil
		}

		parentResult, err := subExecute(sub, parentDefn)
		if err != nil {
			vm.push(BoolVal(false))
			return nil
		}

		outU := vm.Store.Get(outName)
		if outU == nil {
			vm.push(BoolVal(false))
			return nil
		}
		outData := outU.Get("data")
		if outData == nil {
			vm.push(BoolVal(false))
			return nil
		}
		observed := anyToValue(outData)

		if !semanticValuesEqual(parentResult, observed) {
			vm.push(BoolVal(false))
			return nil
		}
	}

	vm.push(BoolVal(true))
	return nil
}

// semanticValuesEqual compares two Values with set-semantics for lists
// (order-insensitive, dedupe-insensitive for int lists) and strict
// equality otherwise.
func semanticValuesEqual(a, b Value) bool {
	if a.Kind() == VList && b.Kind() == VList {
		as, bs := toIntSet(a), toIntSet(b)
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
		return true
	}
	return a.Equal(b)
}

// stringSlicesEqual compares two []string for elementwise equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
