package dsl

import (
	"fmt"
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
	"concat":       bConcat,
	"pack-name":    bPackName,
	"starts-with?": bStartsWith,

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

	// Specialization pipeline
	"add-spec-task":       bAddSpecTask,
	"add-gen-task":        bAddGenTask,
	"replace-slot-value":  bReplaceSlotValue,
	"record-slot-change":  bRecordSlotChange,

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
		"ifPotentiallyRelevant", "ifTrulyRelevant", "ifWorkingOnTask",
		"ifFinishedWorkingOnTask", "thenCompute", "thenAddToAgenda",
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
	var slots []Value
	for k := range u.Slots {
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
	var slots []Value
	for k := range u.Slots {
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
	var slots []Value
	for k := range u.Slots {
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
// reverse. No-op if the unit does not exist.
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
