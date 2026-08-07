package dsl

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func testVM(t *testing.T) *VM {
	t.Helper()
	s := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(s, ag, nil)
	vm.Out = &bytes.Buffer{}
	return vm
}

func TestArithmetic(t *testing.T) {
	vm := testVM(t)
	tests := []struct {
		prog string
		want int
	}{
		{"3 4 +", 7},
		{"10 3 -", 7},
		{"6 7 *", 42},
		{"20 4 /", 5},
		{"17 5 mod", 2},
	}
	for _, tt := range tests {
		v, err := vm.Execute(tt.prog)
		if err != nil {
			t.Errorf("%s: %v", tt.prog, err)
			continue
		}
		if v.AsInt() != tt.want {
			t.Errorf("%s: got %d, want %d", tt.prog, v.AsInt(), tt.want)
		}
	}
}

func TestCollectionWordsSupportStringVocabularies(t *testing.T) {
	vm := testVM(t)
	value, err := vm.Execute(`"web>api" "api>core" 2 list-of "api>core" "db>schema" 2 list-of collection-union`)
	if err != nil {
		t.Fatal(err)
	}
	got := value.AsList()
	want := []string{"web>api", "api>core", "db>schema"}
	if len(got) != len(want) {
		t.Fatalf("union length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, expected := range want {
		if got[i].AsString() != expected {
			t.Fatalf("union[%d] = %q, want %q", i, got[i].AsString(), expected)
		}
	}

	equal, err := vm.Execute(`"a" "b" 2 list-of "b" "a" "a" 3 list-of collection-equal?`)
	if err != nil {
		t.Fatal(err)
	}
	if !equal.AsBool() {
		t.Fatal("collection equality should ignore order and duplicates")
	}
}

func TestComparison(t *testing.T) {
	vm := testVM(t)
	tests := []struct {
		prog string
		want bool
	}{
		{"3 3 =", true},
		{"3 4 =", false},
		{"3 4 <", true},
		{"4 3 >", true},
		{"3 3 <=", true},
		{"3 3 >=", true},
	}
	for _, tt := range tests {
		v, err := vm.Execute(tt.prog)
		if err != nil {
			t.Errorf("%s: %v", tt.prog, err)
			continue
		}
		if v.AsBool() != tt.want {
			t.Errorf("%s: got %v, want %v", tt.prog, v.AsBool(), tt.want)
		}
	}
}

func TestVariables(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`42 "x" ! "x" @`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 42 {
		t.Errorf("got %d, want 42", v.AsInt())
	}
}

func TestIfThenEnd(t *testing.T) {
	vm := testVM(t)

	// Forth-style: <cond> if <true-body> then
	v, err := vm.Execute(`true if 42 then`)
	if err != nil {
		t.Fatalf("true branch: %v", err)
	}
	if v.AsInt() != 42 {
		t.Errorf("true branch: got %v, want 42", v)
	}

	// False branch - should not push
	v, err = vm.Execute(`false if 42 then`)
	if err != nil {
		t.Fatalf("false branch: %v", err)
	}
	if !v.IsNil() {
		t.Errorf("false branch: got %v, want nil", v)
	}

	// With else
	v, err = vm.Execute(`false if 1 else 2 then`)
	if err != nil {
		t.Fatalf("else branch: %v", err)
	}
	if v.AsInt() != 2 {
		t.Errorf("else branch: got %v, want 2", v)
	}

	// Multiline if (the pattern used in heuristics)
	v, err = vm.Execute(`
		1 0 >
		if
			99
		then
	`)
	if err != nil {
		t.Fatalf("multiline if: %v", err)
	}
	if v.AsInt() != 99 {
		t.Errorf("multiline if: got %v, want 99", v)
	}

	// Nested if
	v, err = vm.Execute(`true if true if 7 then then`)
	if err != nil {
		t.Fatalf("nested if: %v", err)
	}
	if v.AsInt() != 7 {
		t.Errorf("nested if: got %v, want 7", v)
	}
}

func TestEach(t *testing.T) {
	vm := testVM(t)

	// Create a unit with a list
	u := unit.New("TestUnit")
	u.Set("items", []string{"a", "b", "c"})
	vm.Store.Put(u)

	v, err := vm.Execute(`"TestUnit" "items" get-slot each it print end true`)
	if err != nil {
		t.Fatalf("each: %v", err)
	}
	if !v.Truthy() {
		t.Error("expected true after each loop")
	}
}

func TestStoreOps(t *testing.T) {
	vm := testVM(t)

	u := unit.New("Foo")
	u.Set("worth", 500)
	u.Set("isA", []string{"Anything"})
	vm.Store.Put(u)

	// get-slot
	v, err := vm.Execute(`"Foo" "worth" get-slot`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 500 {
		t.Errorf("get-slot: got %d, want 500", v.AsInt())
	}

	// isa?
	v, err = vm.Execute(`"Foo" "Anything" isa?`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.AsBool() {
		t.Error("isa? should be true")
	}

	// create-unit
	v, err = vm.Execute(`"Bar" "Anything" create-unit`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsString() != "Bar" {
		t.Errorf("create-unit: got %v", v)
	}
	if !vm.Store.Has("Bar") {
		t.Error("Bar should exist")
	}
}

func TestAddTask(t *testing.T) {
	vm := testVM(t)

	_, err := vm.Execute(`500 "MyUnit" "examples" "test reason" add-task`)
	if err != nil {
		t.Fatal(err)
	}
	if vm.Ag.Len() != 1 {
		t.Errorf("expected 1 task, got %d", vm.Ag.Len())
	}
	task := vm.Ag.Pop()
	if task.Priority != 500 || task.UnitName != "MyUnit" || task.SlotName != "examples" {
		t.Errorf("unexpected task: %+v", task)
	}
}

func TestComments(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`
		# This is a comment
		42
		# Another comment
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 42 {
		t.Errorf("got %d, want 42", v.AsInt())
	}
}

func TestAbort(t *testing.T) {
	vm := testVM(t)
	_, err := vm.Execute(`abort`)
	if !IsAbort(err) {
		t.Errorf("expected AbortError, got %v", err)
	}
}

func TestApplicsSuccessRatio(t *testing.T) {
	vm := testVM(t)
	u := unit.New("H-Test")
	u.Set("overallRecord", map[string]any{"successes": 7, "failures": 3})
	vm.Store.Put(u)

	v, err := vm.Execute(`"H-Test" applics-success-ratio`)
	if err != nil {
		t.Fatal(err)
	}
	got := v.AsFloat()
	if got < 0.69 || got > 0.71 {
		t.Errorf("got %f, want ~0.7", got)
	}
}

func TestApplicsSuccessRatioNoRecord(t *testing.T) {
	vm := testVM(t)
	u := unit.New("H-Empty")
	vm.Store.Put(u)

	v, err := vm.Execute(`"H-Empty" applics-success-ratio`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsFloat() != 0.0 {
		t.Errorf("got %f, want 0.0", v.AsFloat())
	}
}

func TestGetApplics(t *testing.T) {
	vm := testVM(t)
	u := unit.New("H-Test2")
	u.Set("applics", []map[string]any{
		{"target": "A", "result": true},
		{"target": "B", "result": false},
	})
	vm.Store.Put(u)

	v, err := vm.Execute(`"H-Test2" get-applics`)
	if err != nil {
		t.Fatal(err)
	}
	if v.IsNil() {
		t.Error("expected non-nil applics")
	}
	// anyToValue converts []map[string]any to IntVal(len), so expect 2
	if v.AsInt() != 2 {
		t.Errorf("got %d, want 2", v.AsInt())
	}
}

func TestAllSlots(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	u := unit.New("TestUnit")
	u.Set("isA", []string{"Anything"})
	u.Set("domain", []string{"Set"})
	u.Set("english", "test")
	store.Put(u)

	v, err := vm.Execute(`"TestUnit" all-slots`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) < 3 {
		t.Errorf("expected at least 3 slots (isA, domain, english), got %d", len(list))
	}
}

func TestCriterialSlots(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	// Create slot ontology units
	critSlot := unit.New("CriterialSlot")
	critSlot.Set("isA", []string{"Slot", "Anything"})
	store.Put(critSlot)

	slotUnit := unit.New("Slot")
	slotUnit.Set("isA", []string{"Anything"})
	store.Put(slotUnit)

	anything := unit.New("Anything")
	store.Put(anything)

	// Slot def units are PascalCase; slot keys are lowercase-first.
	domainSlot := unit.New("Domain")
	domainSlot.Set("isA", []string{"Slot", "CriterialSlot", "Anything"})
	store.Put(domainSlot)

	englishSlot := unit.New("English")
	englishSlot.Set("isA", []string{"Slot", "NonCriterialSlot", "Anything"})
	store.Put(englishSlot)

	nonCritSlot := unit.New("NonCriterialSlot")
	nonCritSlot.Set("isA", []string{"Slot", "Anything"})
	store.Put(nonCritSlot)

	// Create a unit with both slot types
	u := unit.New("TestOp")
	u.Set("isA", []string{"Op"})
	u.Set("domain", []string{"Set"})
	u.Set("english", "test op")
	store.Put(u)

	v, err := vm.Execute(`"TestOp" criterial-slots`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	found := false
	for _, item := range list {
		if item.AsString() == "domain" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'domain' in criterial slots")
	}
}

func TestSibSlots(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	domainSlot := unit.New("Domain")
	domainSlot.Set("isA", []string{"Slot"})
	domainSlot.Set("sibSlots", []string{"Range"})
	store.Put(domainSlot)

	v, err := vm.Execute(`"Domain" sib-slots`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) != 1 || list[0].AsString() != "Range" {
		t.Errorf("expected [Range], got %v", list)
	}
}

func TestInverseSlot(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	domainSlot := unit.New("Domain")
	domainSlot.Set("isA", []string{"Slot"})
	domainSlot.Set("inverse", "InDomainOf")
	store.Put(domainSlot)

	v, err := vm.Execute(`"Domain" inverse-slot`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsString() != "InDomainOf" {
		t.Errorf("expected InDomainOf, got %s", v.AsString())
	}
}

func TestApplicsByType(t *testing.T) {
	vm := testVM(t)

	// Set up target units with different isA types
	t1 := unit.New("Target1")
	t1.Set("isA", []string{"Concept"})
	vm.Store.Put(t1)

	t2 := unit.New("Target2")
	t2.Set("isA", []string{"Heuristic"})
	vm.Store.Put(t2)

	// Set up heuristic with applics
	h := unit.New("H-Analyze")
	h.Set("applics", []map[string]any{
		{"target": "Target1", "result": true},
		{"target": "Target2", "result": false},
		{"target": "Target1", "result": true},
	})
	vm.Store.Put(h)

	v, err := vm.Execute(`"H-Analyze" applics-by-type`)
	if err != nil {
		t.Fatal(err)
	}
	if v.IsNil() {
		t.Error("expected non-nil applics-by-type result")
	}
	// anyToValue converts map[string]any to StringVal, so result should be non-empty
	if v.AsString() == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestGetTaskExtra(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	vm.CurrentTask = &agenda.Task{
		Extra: map[string]any{"SlotToChange": "domain"},
	}

	v, err := vm.Execute(`"SlotToChange" get-task-extra`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsString() != "domain" {
		t.Errorf("expected 'domain', got %s", v.AsString())
	}
}

func TestGetTaskExtraNil(t *testing.T) {
	vm := testVM(t)

	// No CurrentTask set
	v, err := vm.Execute(`"SlotToChange" get-task-extra`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsNil() {
		t.Errorf("expected nil, got %v", v)
	}
}

func TestRandomChoice(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	u := unit.New("TestUnit")
	u.Set("items", []string{"a", "b", "c"})
	store.Put(u)

	v, err := vm.Execute(`"TestUnit" "items" get-slot random-choice`)
	if err != nil {
		t.Fatal(err)
	}
	s := v.AsString()
	if s != "a" && s != "b" && s != "c" {
		t.Errorf("expected one of a/b/c, got %s", s)
	}
}

func TestRandomSubset(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	u := unit.New("TestUnit")
	u.Set("items", []string{"a", "b", "c", "d"})
	store.Put(u)

	v, err := vm.Execute(`"TestUnit" "items" get-slot random-subset`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) == 0 {
		t.Error("expected at least one element in random subset")
	}
	for _, item := range list {
		s := item.AsString()
		if s != "a" && s != "b" && s != "c" && s != "d" {
			t.Errorf("unexpected item %s in subset", s)
		}
	}
}

func TestRecordSlotChange(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag, nil)
	vm.Out = &bytes.Buffer{}

	u := unit.New("Child")
	store.Put(u)

	_, err := vm.Execute(`"Child" "domain" "Anything" "Number" record-slot-change`)
	if err != nil {
		t.Fatal(err)
	}

	got := store.Get("Child")
	if got.GetString("cSlot") != "domain" {
		t.Errorf("cSlot: got %q, want domain", got.GetString("cSlot"))
	}
	if got.GetString("cFrom") != "Anything" {
		t.Errorf("cFrom: got %q, want Anything", got.GetString("cFrom"))
	}
	if got.GetString("cTo") != "Number" {
		t.Errorf("cTo: got %q, want Number", got.GetString("cTo"))
	}
}

func TestRandomInt(t *testing.T) {
	vm := testVM(t)
	for i := 0; i < 20; i++ {
		v, err := vm.Execute(`10 random-int`)
		if err != nil {
			t.Fatal(err)
		}
		n := v.AsInt()
		if n < 0 || n >= 10 {
			t.Errorf("random-int 10 returned %d, want [0, 10)", n)
		}
	}
	// n <= 0 returns 0
	v, _ := vm.Execute(`0 random-int`)
	if v.AsInt() != 0 {
		t.Errorf("0 random-int: got %d, want 0", v.AsInt())
	}
}

func TestListOf(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`"a" "b" "c" 3 list-of`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) != 3 || list[0].AsString() != "a" || list[1].AsString() != "b" || list[2].AsString() != "c" {
		t.Errorf("list-of result: got %v, want [a b c]", list)
	}
}

func TestRecordApplicAndAccessors(t *testing.T) {
	vm := testVM(t)
	op := unit.New("MyOp")
	op.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	vm.Store.Put(op)

	// Record two applics with different args/outputs.
	_, err := vm.Execute(`"MyOp" "A" "B" 2 list-of "Out1" record-applic`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.Execute(`"MyOp" "C" "D" 2 list-of "Out2" record-applic`)
	if err != nil {
		t.Fatal(err)
	}

	// applics-outputs should yield [Out1 Out2]
	v, _ := vm.Execute(`"MyOp" applics-outputs`)
	outs := v.AsList()
	if len(outs) != 2 || outs[0].AsString() != "Out1" || outs[1].AsString() != "Out2" {
		t.Errorf("applics-outputs: got %v, want [Out1 Out2]", outs)
	}

	// applics-args should yield [[A B] [C D]]
	v, _ = vm.Execute(`"MyOp" applics-args`)
	args := v.AsList()
	if len(args) != 2 {
		t.Fatalf("applics-args: got %d tuples, want 2", len(args))
	}
	first := args[0].AsList()
	if len(first) != 2 || first[0].AsString() != "A" || first[1].AsString() != "B" {
		t.Errorf("applics-args[0]: got %v, want [A B]", first)
	}

	// applics-direct should count 2
	v, _ = vm.Execute(`"MyOp" applics-direct`)
	if v.AsInt() != 2 {
		t.Errorf("applics-direct: got %d, want 2", v.AsInt())
	}
}

func TestMakeProtoConjec(t *testing.T) {
	vm := testVM(t)
	vm.Store.RegisterInverse("conjectureAbout", "conjectures")

	// Two target units the conjecture is about.
	a := unit.New("SetOfNumbers")
	a.Set("isA", []string{"Set", "Anything"})
	vm.Store.Put(a)
	b := unit.New("SetOfPrimes")
	b.Set("isA", []string{"Set", "Anything"})
	vm.Store.Put(b)

	// Fire: kind=SetEqual, about=[SetOfNumbers SetOfPrimes], statement=..., creditor=H-Conjecture
	v, err := vm.Execute(`"SetEqual" "SetOfNumbers" "SetOfPrimes" 2 list-of "SetOfNumbers = SetOfPrimes" "H-Conjecture" make-protoconjec`)
	if err != nil {
		t.Fatal(err)
	}
	name := v.AsString()
	// Name: sorted-about is [SetOfNumbers, SetOfPrimes] alphabetically.
	want := "Conjec-SetEqual-SetOfNumbers-SetOfPrimes"
	if name != want {
		t.Fatalf("name: got %q, want %q", name, want)
	}

	u := vm.Store.Get(name)
	if u == nil {
		t.Fatalf("unit %s not created", name)
	}
	if got := u.GetStrings("isA"); len(got) < 1 || got[0] != "ProtoConjec" {
		t.Errorf("isA: got %v, want first=ProtoConjec", got)
	}
	if u.GetString("conjecKind") != "SetEqual" {
		t.Errorf("conjecKind: got %q, want SetEqual", u.GetString("conjecKind"))
	}
	if u.GetString("status") != "proposed" {
		t.Errorf("status: got %q, want proposed", u.GetString("status"))
	}
	if u.GetString("statement") != "SetOfNumbers = SetOfPrimes" {
		t.Errorf("statement mismatch: %q", u.GetString("statement"))
	}
	if u.GetInt("supportCount") != 1 {
		t.Errorf("supportCount: got %d, want 1", u.GetInt("supportCount"))
	}
	if got := u.GetStrings("creditors"); len(got) != 1 || got[0] != "H-Conjecture" {
		t.Errorf("creditors: got %v", got)
	}
	if got := u.GetStrings("conjectureAbout"); len(got) != 2 {
		t.Errorf("conjectureAbout: got %v, want 2 entries", got)
	}

	// Inverse wired on each target.
	if got := vm.Store.Get("SetOfNumbers").GetStrings("conjectures"); len(got) != 1 || got[0] != name {
		t.Errorf("inverse on SetOfNumbers: got %v, want [%s]", got, name)
	}
	if got := vm.Store.Get("SetOfPrimes").GetStrings("conjectures"); len(got) != 1 || got[0] != name {
		t.Errorf("inverse on SetOfPrimes: got %v, want [%s]", got, name)
	}

	// Dedupe: re-deriving the same conjecture (even with args in reverse order)
	// returns the same name and bumps supportCount.
	v2, err := vm.Execute(`"SetEqual" "SetOfPrimes" "SetOfNumbers" 2 list-of "again" "H-Other" make-protoconjec`)
	if err != nil {
		t.Fatal(err)
	}
	if v2.AsString() != name {
		t.Errorf("dedupe: got new name %q, want %q", v2.AsString(), name)
	}
	if got := u.GetInt("supportCount"); got != 2 {
		t.Errorf("supportCount after re-derive: got %d, want 2", got)
	}
}
