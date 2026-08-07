package tinystack

import (
	"reflect"
	"testing"
)

func TestInstructionSemantics(t *testing.T) {
	tests := []struct {
		input []int
		op    Opcode
		want  []int
	}{
		{[]int{1, 2}, Dup, []int{1, 2, 2}},
		{[]int{1, 2}, Swap, []int{2, 1}},
		{[]int{1, 2}, Drop, []int{1}},
		{[]int{1, 2}, Over, []int{1, 2, 1}},
		{[]int{1, 2}, Add, []int{3}},
		{[]int{1, 2}, Mul, []int{2}},
		{[]int{1, 2}, Double, []int{1, 4}},
		{[]int{1, 2}, Neg, []int{1, -2}},
	}
	for _, testCase := range tests {
		got, err := Step(testCase.input, testCase.op)
		if err != nil || !reflect.DeepEqual(got, testCase.want) {
			t.Fatalf("%v %s = (%v,%v), want %v", testCase.input, testCase.op, got, err, testCase.want)
		}
	}
}

func TestUndefinedAndBoundedExecution(t *testing.T) {
	for _, testCase := range []struct {
		input []int
		op    Opcode
	}{
		{nil, Dup}, {nil, Drop}, {[]int{1}, Swap}, {[]int{1}, Over}, {[]int{1}, Add},
		{[]int{1, 2, 3, 4, 5, 6, 7}, Dup}, {[]int{MaxValueAbs}, Double},
	} {
		if result, err := Step(testCase.input, testCase.op); err == nil || result != nil {
			t.Fatalf("undefined %v %s = (%v,%v)", testCase.input, testCase.op, result, err)
		}
	}
	if ValidOpcode("wat") || ValidInput([]int{101}) || ValidInput([]int{1, 2, 3, 4, 5}) || ValidStack([]int{MaxValueAbs + 1}) {
		t.Fatal("invalid bound accepted")
	}
}

func TestProgramsUseCompleteStacks(t *testing.T) {
	got, err := Execute([]int{2, 3}, []Opcode{Over, Add})
	if err != nil || !reflect.DeepEqual(got, []int{2, 5}) {
		t.Fatalf("over add = (%v,%v)", got, err)
	}
	longer, err := Execute([]int{2, 3}, []Opcode{Over, Swap, Add})
	if err != nil || !reflect.DeepEqual(longer, got) {
		t.Fatalf("over swap add = (%v,%v), want %v", longer, err, got)
	}
	if result, err := Execute([]int{1}, nil); err == nil || result != nil {
		t.Fatalf("empty program = (%v,%v)", result, err)
	}
	if result, err := Execute([]int{1}, []Opcode{Dup, Dup, Dup, Dup}); err == nil || result != nil {
		t.Fatalf("oversized program = (%v,%v)", result, err)
	}
}
