package dsl

import (
	"fmt"
	"sort"
	"strings"

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
	"get-slot":     bGetSlot,
	"set-slot":     bSetSlot,
	"isa?":         bIsA,
	"examples":     bExamples,
	"create-unit":  bCreateUnit,
	"kill-unit":    bKillUnit,
	"unit-exists?": bUnitExists,

	// Agenda
	"add-task": bAddTask,

	// String
	"concat":       bConcat,
	"pack-name":    bPackName,
	"starts-with?": bStartsWith,

	// List
	"list-length":    bListLength,
	"list-append":    bListAppend,
	"list-contains":  bListContains,
	"to-string-list": bToStringList,

	// List/Bag ops (Phase 5.5 parity)
	"list-concat":          bListConcat,
	"list-equal?":          bListEqual,
	"list-remove-all":      bListRemoveAll,
	"list-remove-one":      bListRemoveOne,
	"list-intersect-ord":   bListIntersectOrd,
	"list-diff-ord":        bListDiffOrd,
	"collection-union":     bCollectionUnion,
	"collection-intersect": bCollectionIntersect,
	"collection-diff":      bCollectionDiff,
	"collection-equal?":    bCollectionEqual,
	"bag-equal?":           bBagEqual,
	"bag-intersect":        bBagIntersect,
	"bag-diff":             bBagDiff,

	// Output
	"print": bPrint,
	".s":    bDotS, // debug: print stack

	// Loop variable
	"it": func(vm *VM) error { vm.push(vm.env["it"]); return nil },

	// Applics inspection
	"get-applics":           bGetApplics,
	"applics-success-ratio": bApplicsSuccessRatio,
	"applics-by-type":       bApplicsByType,

	// Meta-heuristic ops
	"analyze-and-specialize": bAnalyzeAndSpecialize,

	// Slot reasoning
	"all-slots":           bAllSlots,
	"criterial-slots":     bCriterialSlots,
	"non-criterial-slots": bNonCriterialSlots,
	"sib-slots":           bSibSlots,
	"super-slots":         bSuperSlots,
	"sub-slots":           bSubSlots,
	"inverse-slot":        bInverseSlot,
	"slot-type":           bSlotType,

	// Task extra
	"get-task-extra": bGetTaskExtra,
	"set-task-extra": bSetTaskExtra,

	// Random
	"random-choice": bRandomChoice,
	"random-subset": bRandomSubset,
	"random-int":    bRandomInt,

	// Specialization pipeline
	"add-spec-task":      bAddSpecTask,
	"add-gen-task":       bAddGenTask,
	"replace-slot-value": bReplaceSlotValue,
	"record-slot-change": bRecordSlotChange,
	"new-units":          bNewUnits,
	"add-to-slot":        bAddToSlot,
	"record-applic":      bRecordApplic,
	"list-of":            bListOf,
	"list-join":          bListJoin,
	"applics-outputs":    bApplicsOutputs,
	"applics-args":       bApplicsArgs,
	"applics-direct":     bApplicsDirect,
	"applics-bad?":       bApplicsBad,
	"make-protoconjec":   bMakeProtoConjec,
	// "is-interesting?" is registered in init() to avoid an init cycle
	// (it calls vm.Execute, which looks up builtins).

	// Misc
	"noop": func(vm *VM) error { return nil },
}

var vocabularyWordSets = map[string]map[string]builtinFn{}

func registerVocabularyWords(extension string, words map[string]builtinFn) {
	if extension == "" {
		panic("dsl: empty vocabulary extension")
	}
	if _, exists := vocabularyWordSets[extension]; exists {
		panic("dsl: duplicate vocabulary extension " + extension)
	}
	vocabularyWordSets[extension] = cloneWords(words)
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
	vm.push(BoolVal(cmpNumeric(a, b) < 0))
	return nil
}
func bGt(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(cmpNumeric(a, b) > 0))
	return nil
}
func bLte(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(cmpNumeric(a, b) <= 0))
	return nil
}
func bGte(vm *VM) error {
	b, a := vm.pop(), vm.pop()
	vm.push(BoolVal(cmpNumeric(a, b) >= 0))
	return nil
}

// cmpNumeric compares two numeric Values, preferring float comparison when
// either is a float so fractional rarity/interestingness values compare
// correctly instead of truncating to int.
func cmpNumeric(a, b Value) int {
	if a.kind == VFloat || b.kind == VFloat {
		af, bf := a.AsFloat(), b.AsFloat()
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		default:
			return 0
		}
	}
	ai, bi := a.AsInt(), b.AsInt()
	switch {
	case ai < bi:
		return -1
	case ai > bi:
		return 1
	default:
		return 0
	}
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
		vm.Store.SetSlot(unitName.AsString(), slot, []string{value.sval})
	} else {
		vm.Store.SetSlot(unitName.AsString(), slot, valueToAny(value))
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
	u.Set("creationWorth", 500)
	u.Set("lastRewardedWorth", 500)
	u.Set("isNew", true)
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, nameStr)
	vm.push(StringVal(nameStr))
	return nil
}

func bKillUnit(vm *VM) error {
	name := vm.pop()
	nameStr := name.AsString()
	// Skip if already dead (prevents repeated kill logging and HindSight spam)
	if !vm.Store.Has(nameStr) {
		return nil
	}
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

func bStartsWith(vm *VM) error {
	prefix := vm.pop()
	str := vm.pop()
	vm.push(BoolVal(strings.HasPrefix(str.AsString(), prefix.AsString())))
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

// list-concat: ( list1 list2 -- list )
// Append list2 after list1 as-is, preserving order and duplicates.
func bListConcat(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	out := make([]Value, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	vm.push(ListVal(out))
	return nil
}

// list-equal?: ( list1 list2 -- bool )
// Position-wise equality using Value.Equal on each element.
func bListEqual(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	if len(a) != len(b) {
		vm.push(BoolVal(false))
		return nil
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

// list-remove-all: ( list elem -- list )
// Return list with every instance of elem removed.
func bListRemoveAll(vm *VM) error {
	elem := vm.pop()
	list := vm.pop().AsList()
	out := make([]Value, 0, len(list))
	for _, v := range list {
		if !v.Equal(elem) {
			out = append(out, v)
		}
	}
	vm.push(ListVal(out))
	return nil
}

// list-remove-one: ( list elem -- list )
// Return list with first occurrence of elem removed.
func bListRemoveOne(vm *VM) error {
	elem := vm.pop()
	list := vm.pop().AsList()
	out := make([]Value, 0, len(list))
	removed := false
	for _, v := range list {
		if !removed && v.Equal(elem) {
			removed = true
			continue
		}
		out = append(out, v)
	}
	vm.push(ListVal(out))
	return nil
}

// list-intersect-ord: ( list1 list2 -- list )
// Elements of list1 that are present in list2, preserving list1's order and duplicates.
func bListIntersectOrd(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	out := make([]Value, 0, len(a))
	for _, x := range a {
		for _, y := range b {
			if x.Equal(y) {
				out = append(out, x)
				break
			}
		}
	}
	vm.push(ListVal(out))
	return nil
}

// list-diff-ord: ( list1 list2 -- list )
// Elements of list1 NOT in list2, preserving list1's order and duplicates.
func bListDiffOrd(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	out := make([]Value, 0, len(a))
	for _, x := range a {
		found := false
		for _, y := range b {
			if x.Equal(y) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, x)
		}
	}
	vm.push(ListVal(out))
	return nil
}

// collection-* treats lists of any Value kind as mathematical sets. The
// first occurrence determines output order, so results are deterministic and
// useful to non-numeric domain vocabularies as well as the math vocabulary.
func bCollectionUnion(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	out := append([]Value(nil), a...)
	for _, candidate := range b {
		if !containsValue(out, candidate) {
			out = append(out, candidate)
		}
	}
	vm.push(ListVal(deduplicateValues(out)))
	return nil
}

func bCollectionIntersect(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	var out []Value
	for _, candidate := range a {
		if containsValue(b, candidate) && !containsValue(out, candidate) {
			out = append(out, candidate)
		}
	}
	vm.push(ListVal(out))
	return nil
}

func bCollectionDiff(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	var out []Value
	for _, candidate := range a {
		if !containsValue(b, candidate) && !containsValue(out, candidate) {
			out = append(out, candidate)
		}
	}
	vm.push(ListVal(out))
	return nil
}

func bCollectionEqual(vm *VM) error {
	b := deduplicateValues(vm.pop().AsList())
	a := deduplicateValues(vm.pop().AsList())
	if len(a) != len(b) {
		vm.push(BoolVal(false))
		return nil
	}
	for _, candidate := range a {
		if !containsValue(b, candidate) {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

func containsValue(values []Value, candidate Value) bool {
	for _, value := range values {
		if value.Equal(candidate) {
			return true
		}
	}
	return false
}

func deduplicateValues(values []Value) []Value {
	out := make([]Value, 0, len(values))
	for _, value := range values {
		if !containsValue(out, value) {
			out = append(out, value)
		}
	}
	return out
}

// bagCounts builds a count map keyed by v.AsString() — sufficient for seed
// data where elements are primitives rendering to unique strings.
func bagCounts(list []Value) map[string]int {
	m := make(map[string]int, len(list))
	for _, v := range list {
		m[v.AsString()]++
	}
	return m
}

// bag-equal?: ( bag1 bag2 -- bool )
// Multiset equality via count map on v.AsString().
func bBagEqual(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	if len(a) != len(b) {
		vm.push(BoolVal(false))
		return nil
	}
	ca := bagCounts(a)
	cb := bagCounts(b)
	if len(ca) != len(cb) {
		vm.push(BoolVal(false))
		return nil
	}
	for k, n := range ca {
		if cb[k] != n {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

// bag-intersect: ( bag1 bag2 -- bag )
// For each element in bag1's order, keep it if remaining count in bag2 > 0.
// Result count for each key = min(countB1, countB2).
func bBagIntersect(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	cb := bagCounts(b)
	out := make([]Value, 0, len(a))
	for _, v := range a {
		k := v.AsString()
		if cb[k] > 0 {
			out = append(out, v)
			cb[k]--
		}
	}
	vm.push(ListVal(out))
	return nil
}

// bag-diff: ( bag1 bag2 -- bag )
// For each element in bag1's order, emit unless bag2 still has a copy to
// "cancel" it. Result count for each key = max(0, countB1 - countB2).
func bBagDiff(vm *VM) error {
	b := vm.pop().AsList()
	a := vm.pop().AsList()
	cb := bagCounts(b)
	out := make([]Value, 0, len(a))
	for _, v := range a {
		k := v.AsString()
		if cb[k] > 0 {
			cb[k]--
			continue
		}
		out = append(out, v)
	}
	vm.push(ListVal(out))
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

// Applics inspection

func bGetApplics(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	raw := u.Get("applics")
	vm.push(anyToValue(raw))
	return nil
}

func bApplicsSuccessRatio(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(FloatVal(0))
		return nil
	}
	record := u.GetMap("overallRecord")
	if record == nil {
		vm.push(FloatVal(0))
		return nil
	}
	s := toIntDSL(record["successes"])
	f := toIntDSL(record["failures"])
	total := s + f
	if total == 0 {
		vm.push(FloatVal(0))
		return nil
	}
	vm.push(FloatVal(float64(s) / float64(total)))
	return nil
}

func bApplicsByType(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		vm.push(Nil())
		return nil
	}
	type counts struct {
		successes int
		failures  int
	}
	byType := make(map[string]*counts)
	for _, a := range applics {
		target, _ := a["target"].(string)
		result, _ := a["result"].(bool)
		targetUnit := vm.Store.Get(target)
		typeName := "unknown"
		if targetUnit != nil {
			isA := targetUnit.GetStrings("isA")
			if len(isA) > 0 {
				typeName = isA[0]
			}
		}
		c, ok := byType[typeName]
		if !ok {
			c = &counts{}
			byType[typeName] = c
		}
		if result {
			c.successes++
		} else {
			c.failures++
		}
	}
	result := make(map[string]any)
	for typ, c := range byType {
		result[typ] = map[string]any{"s": c.successes, "f": c.failures}
	}
	vm.push(anyToValue(result))
	return nil
}

func bAnalyzeAndSpecialize(vm *VM) error {
	name := vm.pop()
	nameStr := name.AsString()
	u := vm.Store.Get(nameStr)
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}

	applics, _ := u.Get("applics").([]map[string]any)
	if len(applics) < 10 {
		vm.push(BoolVal(false))
		return nil
	}

	// Group by target type
	type counts struct {
		successes int
		failures  int
	}
	byType := make(map[string]*counts)

	for _, a := range applics {
		target, _ := a["target"].(string)
		result, _ := a["result"].(bool)
		targetUnit := vm.Store.Get(target)
		typeName := "unknown"
		if targetUnit != nil {
			isA := targetUnit.GetStrings("isA")
			if len(isA) > 0 {
				typeName = isA[0]
			}
		}
		c, ok := byType[typeName]
		if !ok {
			c = &counts{}
			byType[typeName] = c
		}
		if result {
			c.successes++
		} else {
			c.failures++
		}
	}

	// Find best type (highest success rate with >= 3 data points)
	bestType := ""
	bestRatio := 0.0
	for typ, c := range byType {
		total := c.successes + c.failures
		if total < 3 || typ == "unknown" {
			continue
		}
		ratio := float64(c.successes) / float64(total)
		if ratio > bestRatio {
			bestRatio = ratio
			bestType = typ
		}
	}

	// Need clear skew: best type ratio > 0.7 and overall ratio < 0.7
	if bestType == "" || bestRatio <= 0.7 {
		vm.push(BoolVal(false))
		return nil
	}

	// Create specialized copy
	specName := nameStr + "-on-" + bestType
	if vm.Store.Has(specName) {
		vm.push(BoolVal(false))
		return nil
	}

	spec := &unit.Unit{
		Name:  specName,
		Slots: map[string]any{},
	}
	spec.Set("isA", []string{"Heuristic", "Anything"})
	spec.SetWorth(u.Worth())
	spec.Set("creditors", []string{"H-AnalyzeApplics"})
	spec.Set("specialized_from", nameStr)
	spec.Set("specialized_type", bestType)
	spec.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})

	// Copy program slots, prepending type check to ifPotentiallyRelevant
	for _, slot := range []string{
		"ifAboutToWorkOnTask", "ifPotentiallyRelevant", "ifTrulyRelevant",
		"ifWorkingOnTask", "ifFinishedWorkingOnTask",
		"thenCompute", "thenAddToAgenda",
		"thenDefineNewConcepts", "thenDeleteOldConcepts", "thenPrintToUser",
		"thenConjecture",
	} {
		prog := u.GetString(slot)
		if prog == "" {
			continue
		}
		if slot == "ifPotentiallyRelevant" {
			prog = fmt.Sprintf(`"ArgU" @ "%s" isa? `, bestType) + prog + " and"
		}
		spec.Set(slot, prog)
	}

	if u.GetString("english") != "" {
		spec.Set("english", fmt.Sprintf("Specialized %s for %s targets", nameStr, bestType))
	}

	vm.Store.Put(spec)
	vm.NewUnits = append(vm.NewUnits, specName)
	vm.push(BoolVal(true))
	return nil
}

// Slot reasoning builtins

func bAllSlots(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	keys := make([]string, 0, len(u.Slots))
	for k := range u.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	slots := make([]Value, 0, len(keys))
	for _, k := range keys {
		slots = append(slots, StringVal(k))
	}
	vm.push(ListVal(slots))
	return nil
}

func bCriterialSlots(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	keys := make([]string, 0, len(u.Slots))
	for k := range u.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var slots []Value
	for _, k := range keys {
		if vm.Store.IsA(slotDefName(k), "CriterialSlot") {
			slots = append(slots, StringVal(k))
		}
	}
	vm.push(ListVal(slots))
	return nil
}

func bNonCriterialSlots(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	keys := make([]string, 0, len(u.Slots))
	for k := range u.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var slots []Value
	for _, k := range keys {
		if vm.Store.IsA(slotDefName(k), "NonCriterialSlot") {
			slots = append(slots, StringVal(k))
		}
	}
	vm.push(ListVal(slots))
	return nil
}

// slotDefName maps a slot key (e.g. "domain") to its slot definition unit
// name (e.g. "Domain"). Slot keys on units are lowercase-first while slot
// definition units are PascalCase.
func slotDefName(slotKey string) string {
	if slotKey == "" {
		return slotKey
	}
	return strings.ToUpper(slotKey[:1]) + slotKey[1:]
}

func bSibSlots(vm *VM) error {
	name := vm.pop()
	slotUnit := vm.Store.Get(name.AsString())
	if slotUnit == nil {
		vm.push(Nil())
		return nil
	}
	sibs := slotUnit.GetStrings("sibSlots")
	if sibs == nil {
		vm.push(ListVal(nil))
		return nil
	}
	vals := make([]Value, len(sibs))
	for i, s := range sibs {
		vals[i] = StringVal(s)
	}
	vm.push(ListVal(vals))
	return nil
}

func bSuperSlots(vm *VM) error {
	name := vm.pop()
	slotUnit := vm.Store.Get(name.AsString())
	if slotUnit == nil {
		vm.push(Nil())
		return nil
	}
	supers := slotUnit.GetStrings("superSlots")
	if supers == nil {
		vm.push(ListVal(nil))
		return nil
	}
	vals := make([]Value, len(supers))
	for i, s := range supers {
		vals[i] = StringVal(s)
	}
	vm.push(ListVal(vals))
	return nil
}

func bSubSlots(vm *VM) error {
	name := vm.pop()
	slotUnit := vm.Store.Get(name.AsString())
	if slotUnit == nil {
		vm.push(Nil())
		return nil
	}
	subs := slotUnit.GetStrings("subSlots")
	if subs == nil {
		vm.push(ListVal(nil))
		return nil
	}
	vals := make([]Value, len(subs))
	for i, s := range subs {
		vals[i] = StringVal(s)
	}
	vm.push(ListVal(vals))
	return nil
}

func bInverseSlot(vm *VM) error {
	name := vm.pop()
	slotUnit := vm.Store.Get(name.AsString())
	if slotUnit == nil {
		vm.push(Nil())
		return nil
	}
	inv := slotUnit.GetString("inverse")
	if inv == "" {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(inv))
	return nil
}

func bSlotType(vm *VM) error {
	name := vm.pop()
	slotUnit := vm.Store.Get(name.AsString())
	if slotUnit == nil {
		vm.push(Nil())
		return nil
	}
	dt := slotUnit.GetString("dataType")
	if dt == "" {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(dt))
	return nil
}

// Task extra builtins

func bGetTaskExtra(vm *VM) error {
	key := vm.pop()
	if vm.CurrentTask == nil || vm.CurrentTask.Extra == nil {
		vm.push(Nil())
		return nil
	}
	val, ok := vm.CurrentTask.Extra[key.AsString()]
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(anyToValue(val))
	return nil
}

func bSetTaskExtra(vm *VM) error {
	key := vm.pop()
	value := vm.pop()
	if vm.CurrentTask == nil {
		return nil
	}
	if vm.CurrentTask.Extra == nil {
		vm.CurrentTask.Extra = make(map[string]any)
	}
	vm.CurrentTask.Extra[key.AsString()] = valueToAny(value)
	return nil
}

// Random builtins

func bRandomChoice(vm *VM) error {
	list := vm.pop()
	items := list.AsList()
	if len(items) == 0 {
		vm.push(Nil())
		return nil
	}
	idx := vm.Rng.Intn(len(items))
	vm.push(items[idx])
	return nil
}

// random-int: (n -- i)
// Pushes a pseudo-random integer in [0, n). Returns 0 for n <= 0.
func bRandomInt(vm *VM) error {
	n := vm.pop().AsInt()
	if n <= 0 {
		vm.push(IntVal(0))
		return nil
	}
	vm.push(IntVal(vm.Rng.Intn(n)))
	return nil
}

func bRandomSubset(vm *VM) error {
	list := vm.pop()
	items := list.AsList()
	if len(items) == 0 {
		vm.push(ListVal(nil))
		return nil
	}
	var result []Value
	for _, item := range items {
		if vm.Rng.Intn(2) == 0 {
			result = append(result, item)
		}
	}
	// Ensure at least one element if input was non-empty
	if len(result) == 0 {
		idx := vm.Rng.Intn(len(items))
		result = append(result, items[idx])
	}
	vm.push(ListVal(result))
	return nil
}

// Specialization pipeline builtins

// add-spec-task: (priority unitName slotToChange specFrom specTo --)
// Creates a task on "specializations" slot with Extra values set.
func bAddSpecTask(vm *VM) error {
	specTo := vm.pop()
	specFrom := vm.pop()
	slotToChange := vm.pop()
	unitName := vm.pop()
	priority := vm.pop()
	vm.Ag.Push(&agenda.Task{
		Priority: priority.AsInt(),
		UnitName: unitName.AsString(),
		SlotName: "specializations",
		Reasons:  []string{"Specialize " + slotToChange.AsString()},
		Extra: map[string]any{
			"SlotToChange":   slotToChange.AsString(),
			"SpecializeFrom": specFrom.AsString(),
			"SpecializeTo":   specTo.AsString(),
		},
	})
	return nil
}

// add-gen-task: (priority unitName slotToChange genFrom genTo --)
// Creates a task on "generalizations" slot with Extra values set.
func bAddGenTask(vm *VM) error {
	genTo := vm.pop()
	genFrom := vm.pop()
	slotToChange := vm.pop()
	unitName := vm.pop()
	priority := vm.pop()
	vm.Ag.Push(&agenda.Task{
		Priority: priority.AsInt(),
		UnitName: unitName.AsString(),
		SlotName: "generalizations",
		Reasons:  []string{"Generalize " + slotToChange.AsString()},
		Extra: map[string]any{
			"SlotToChange":   slotToChange.AsString(),
			"GeneralizeFrom": genFrom.AsString(),
			"GeneralizeTo":   genTo.AsString(),
		},
	})
	return nil
}

// replace-slot-value: (unitName slotName from to -- bool)
// Replaces first occurrence of 'from' with 'to' in the slot value.
func bReplaceSlotValue(vm *VM) error {
	to := vm.pop()
	from := vm.pop()
	slotName := vm.pop()
	unitName := vm.pop()

	u := vm.Store.Get(unitName.AsString())
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}

	slot := slotName.AsString()
	val := u.Get(slot)

	switch v := val.(type) {
	case []string:
		newVal := make([]string, len(v))
		copy(newVal, v)
		replaced := false
		for i, s := range newVal {
			if s == from.AsString() && !replaced {
				newVal[i] = to.AsString()
				replaced = true
			}
		}
		if replaced {
			vm.Store.SetSlot(unitName.AsString(), slot, newVal)
			vm.push(BoolVal(true))
		} else {
			vm.push(BoolVal(false))
		}
	case string:
		if v == from.AsString() {
			vm.Store.SetSlot(unitName.AsString(), slot, to.AsString())
			vm.push(BoolVal(true))
		} else {
			vm.push(BoolVal(false))
		}
	default:
		vm.push(BoolVal(false))
	}
	return nil
}

// record-slot-change: (unitName slot from to --)
// Writes cSlot/cFrom/cTo provenance onto the unit so HindSight heuristics
// (H12/H13/H14) can later analyze what was changed. Direction convention:
// cFrom is the pre-change value, cTo is the post-change value — for
// specialization cFrom is wider and cTo narrower; for generalization the
// reverse. Also copies the current task's CurSlot as gSlot — the task slot
// being computed when the unit was created (e.g. "specializations"),
// EURISKO's GSlot. No-op if the unit does not exist.
func bRecordSlotChange(vm *VM) error {
	to := vm.pop()
	from := vm.pop()
	slot := vm.pop()
	unitName := vm.pop()
	name := unitName.AsString()
	if vm.Store.Get(name) == nil {
		return nil
	}
	vm.Store.SetSlot(name, "cSlot", slot.AsString())
	vm.Store.SetSlot(name, "cFrom", from.AsString())
	vm.Store.SetSlot(name, "cTo", to.AsString())
	if gSlot := vm.GetEnv("CurSlot"); gSlot.AsString() != "" {
		vm.Store.SetSlot(name, "gSlot", gSlot.AsString())
	}
	return nil
}

func init() {
	builtins["is-interesting?"] = bIsInteresting
}

// is-interesting?: (unit candidate -- bool)
// Runs the unit's `interestingness` slot as a DSL program, with env
// "candidate" bound to the candidate name. Returns the truthiness of the
// program's top-of-stack result. False if the unit has no interestingness
// predicate, or if the predicate errors (including abort).
func bIsInteresting(vm *VM) error {
	cand := vm.pop()
	unitName := vm.pop()
	u := vm.Store.Get(unitName.AsString())
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}
	prog := u.GetString("interestingness")
	if prog == "" {
		vm.push(BoolVal(false))
		return nil
	}
	prev, hadPrev := vm.env["candidate"]
	vm.env["candidate"] = cand
	v, err := vm.Execute(prog)
	if hadPrev {
		vm.env["candidate"] = prev
	} else {
		delete(vm.env, "candidate")
	}
	if err != nil {
		vm.push(BoolVal(false))
		return nil
	}
	vm.push(BoolVal(v.Truthy()))
	return nil
}

// add-to-slot: (value unit slot --)
// Appends value (string) to a list-valued slot on the unit, via Store.SetSlot
// so inverse maintenance fires. Idempotent — skips if value already present.
// No-op if unit does not exist.
func bAddToSlot(vm *VM) error {
	slot := vm.pop()
	unitName := vm.pop()
	value := vm.pop()
	name := unitName.AsString()
	u := vm.Store.Get(name)
	if u == nil {
		return nil
	}
	slotKey := slot.AsString()
	existing := u.GetStrings(slotKey)
	v := value.AsString()
	for _, e := range existing {
		if e == v {
			return nil
		}
	}
	vm.Store.SetSlot(name, slotKey, append(existing, v))
	return nil
}

// list-join: (list sep -- string)
// Concatenates list elements with sep between each. `["a" "b" "c"] "-" list-join` → "a-b-c".
func bListJoin(vm *VM) error {
	sep := vm.pop().AsString()
	list := vm.pop().AsList()
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = v.AsString()
	}
	vm.push(StringVal(strings.Join(parts, sep)))
	return nil
}

// list-of: (x1 ... xn n -- list)
// Pops n from the top of the stack, then pops n values beneath and pushes
// them as a list in original order. `a b 2 list-of` → `[a b]`.
func bListOf(vm *VM) error {
	n := vm.pop().AsInt()
	if n < 0 {
		vm.push(ListVal(nil))
		return nil
	}
	if n == 0 {
		vm.push(ListVal(nil))
		return nil
	}
	vals := make([]Value, n)
	for i := n - 1; i >= 0; i-- {
		vals[i] = vm.pop()
	}
	vm.push(ListVal(vals))
	return nil
}

// record-applic: (opName argList output --)
// Appends a rich applic entry on opName, recording the inputs (argList, a
// DSL list of strings) and the output unit name. Marks direct=true. Used
// by H-RunOnExamples (and any other heuristic that applies an op to data)
// so H8/H10/H15/H20 can later read actual I/O pairs.
//
// Caps the applics list at 50 most-recent entries, same policy as
// trackApplics for heuristic firings.
func bRecordApplic(vm *VM) error {
	output := vm.pop()
	argList := vm.pop()
	opName := vm.pop().AsString()
	u := vm.Store.Get(opName)
	if u == nil {
		return nil
	}
	args := make([]string, 0, len(argList.AsList()))
	for _, v := range argList.AsList() {
		args = append(args, v.AsString())
	}
	applic := map[string]any{
		"target": opName,
		"result": true,
		"args":   args,
		"output": output.AsString(),
		"direct": true,
	}
	applics, _ := u.Get("applics").([]map[string]any)
	applics = append(applics, applic)
	if len(applics) > 50 {
		applics = applics[len(applics)-50:]
	}
	u.Set("applics", applics)
	return nil
}

// applics-outputs: (opName -- list)
// Returns the list of output unit names recorded across this op's applics.
// Empty-string outputs (from heuristic firings, not op applications) are
// skipped. Duplicates preserved in order.
func bApplicsOutputs(vm *VM) error {
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(ListVal(nil))
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	var out []Value
	for _, a := range applics {
		if s, ok := a["output"].(string); ok && s != "" {
			out = append(out, StringVal(s))
		}
	}
	vm.push(ListVal(out))
	return nil
}

// applics-args: (opName -- list-of-lists)
// Returns the list of arg-tuples recorded across this op's applics.
// Each arg-tuple is a DSL list of unit-name strings. Applics without args
// are skipped.
func bApplicsArgs(vm *VM) error {
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(ListVal(nil))
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	var out []Value
	for _, a := range applics {
		args, ok := a["args"].([]string)
		if !ok || len(args) == 0 {
			continue
		}
		vals := make([]Value, len(args))
		for i, s := range args {
			vals[i] = StringVal(s)
		}
		out = append(out, ListVal(vals))
	}
	vm.push(ListVal(out))
	return nil
}

// applics-direct: (opName -- int)
// Returns the count of direct applics on opName. Entries with no direct
// field are treated as direct (pre-7.3 default).
func bApplicsDirect(vm *VM) error {
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(IntVal(0))
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	n := 0
	for _, a := range applics {
		d, ok := a["direct"].(bool)
		if !ok || d {
			n++
		}
	}
	vm.push(IntVal(n))
	return nil
}

// applics-bad?: (unitName minTotal -- bool)
// True iff the unit has at least minTotal applic entries AND at least 80% of
// them are failures (result=false). Used by H1 to flag ops whose applications
// are mostly bad, per EURISKO's ">4/5 bad" rule.
func bApplicsBad(vm *VM) error {
	minTotal := vm.pop().AsInt()
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	total := len(applics)
	if total < minTotal {
		vm.push(BoolVal(false))
		return nil
	}
	failures := 0
	for _, a := range applics {
		r, ok := a["result"].(bool)
		if ok && !r {
			failures++
		}
	}
	// failures/total >= 0.8  <=>  failures*5 >= total*4
	vm.push(BoolVal(failures*5 >= total*4))
	return nil
}

// make-protoconjec: (kind aboutList statement creditor -- unitName)
// Creates (or dedupes to) a ProtoConjec unit. Name scheme:
// `Conjec-<kind>-<sorted-about-joined-by-dash>`. If a unit with that name
// already exists, its SupportCount is incremented and its existing name is
// returned — no other slots are overwritten.
//
// On creation: isA = [ProtoConjec, Anything]; status = "proposed";
// conjecKind, evidence (same as about), statement, creditors = [creditor]
// are populated. ConjectureAbout is set via Store.SetSlot so the inverse
// Conjectures slot auto-wires on each target unit.
func bMakeProtoConjec(vm *VM) error {
	creditor := vm.pop().AsString()
	statement := vm.pop().AsString()
	aboutVal := vm.pop()
	kind := vm.pop().AsString()

	about := make([]string, 0, len(aboutVal.AsList()))
	for _, v := range aboutVal.AsList() {
		if s := v.AsString(); s != "" {
			about = append(about, s)
		}
	}
	sortedAbout := append([]string(nil), about...)
	sort.Strings(sortedAbout)
	name := "Conjec-" + kind
	if len(sortedAbout) > 0 {
		name = name + "-" + strings.Join(sortedAbout, "-")
	}

	if existing := vm.Store.Get(name); existing != nil {
		n := existing.GetInt("supportCount")
		existing.Set("supportCount", n+1)
		vm.push(StringVal(name))
		return nil
	}

	u := unit.New(name)
	u.Set("isA", []string{"ProtoConjec", "Anything"})
	u.SetWorth(400)
	u.Set("conjecKind", kind)
	u.Set("status", "proposed")
	u.Set("statement", statement)
	u.Set("supportCount", 1)
	if creditor != "" {
		u.Set("creditors", []string{creditor})
	}
	u.Set("evidence", append([]string(nil), about...))
	vm.Store.Put(u)
	// Use SetSlot so Conjectures inverse wires on each target.
	if len(about) > 0 {
		vm.Store.SetSlot(name, "conjectureAbout", append([]string(nil), about...))
	}
	vm.NewUnits = append(vm.NewUnits, name)
	vm.push(StringVal(name))
	return nil
}

// new-units: ( -- list)
// Pushes the list of units created during the current task. Populated by
// create-unit calls from any ThenPart firing; cleared at task start by the
// engine. Intended for HAvoid2/HAvoid3 (H13/H14) ifFinishedWorkingOnTask
// guards that need to inspect or kill units this task just produced.
func bNewUnits(vm *VM) error {
	vals := make([]Value, len(vm.NewUnits))
	for i, n := range vm.NewUnits {
		vals[i] = StringVal(n)
	}
	vm.push(ListVal(vals))
	return nil
}

func toIntDSL(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
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
	case []any:
		vals := make([]Value, len(x))
		for i, e := range x {
			vals[i] = anyToValue(e)
		}
		return ListVal(vals)
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
		// Convert to []string if all elements are strings,
		// []int if all elements are ints, else []Value.
		if len(v.lval) == 0 {
			return []string{}
		}
		allStr := true
		allInt := true
		for _, el := range v.lval {
			if el.kind != VString {
				allStr = false
			}
			if el.kind != VInt {
				allInt = false
			}
		}
		if allStr {
			strs := make([]string, len(v.lval))
			for i, el := range v.lval {
				strs[i] = el.sval
			}
			return strs
		}
		if allInt {
			ints := make([]int, len(v.lval))
			for i, el := range v.lval {
				ints[i] = el.ival
			}
			return ints
		}
		return v.lval
	default:
		return nil
	}
}
