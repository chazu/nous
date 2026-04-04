// Package dsl implements a stack-based interpreter for nous heuristic programs.
package dsl

import (
	"fmt"
	"strings"
)

// ValueKind tags what a Value holds.
type ValueKind int

const (
	VNil ValueKind = iota
	VBool
	VInt
	VFloat
	VString
	VList
)

// Value is the stack element type for the DSL interpreter.
type Value struct {
	kind ValueKind
	ival int
	fval float64
	sval string
	bval bool
	lval []Value
}

// Constructors

func Nil() Value          { return Value{kind: VNil} }
func BoolVal(b bool) Value { return Value{kind: VBool, bval: b} }
func IntVal(n int) Value  { return Value{kind: VInt, ival: n} }
func FloatVal(f float64) Value { return Value{kind: VFloat, fval: f} }
func StringVal(s string) Value { return Value{kind: VString, sval: s} }
func ListVal(vs []Value) Value { return Value{kind: VList, lval: vs} }

// Accessors

func (v Value) Kind() ValueKind { return v.kind }
func (v Value) IsNil() bool     { return v.kind == VNil }

func (v Value) Truthy() bool {
	switch v.kind {
	case VNil:
		return false
	case VBool:
		return v.bval
	case VInt:
		return v.ival != 0
	case VFloat:
		return v.fval != 0
	case VString:
		return v.sval != ""
	case VList:
		return len(v.lval) != 0
	default:
		return false
	}
}

func (v Value) AsInt() int {
	switch v.kind {
	case VInt:
		return v.ival
	case VFloat:
		return int(v.fval)
	case VBool:
		if v.bval {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func (v Value) AsFloat() float64 {
	switch v.kind {
	case VFloat:
		return v.fval
	case VInt:
		return float64(v.ival)
	default:
		return 0
	}
}

func (v Value) AsString() string {
	switch v.kind {
	case VString:
		return v.sval
	case VInt:
		return fmt.Sprintf("%d", v.ival)
	case VFloat:
		return fmt.Sprintf("%g", v.fval)
	case VBool:
		if v.bval {
			return "true"
		}
		return "false"
	case VNil:
		return "nil"
	case VList:
		parts := make([]string, len(v.lval))
		for i, el := range v.lval {
			parts[i] = el.AsString()
		}
		return "[" + strings.Join(parts, " ") + "]"
	default:
		return ""
	}
}

func (v Value) AsList() []Value {
	if v.kind == VList {
		return v.lval
	}
	return nil
}

func (v Value) AsBool() bool {
	return v.Truthy()
}

func (v Value) String() string {
	switch v.kind {
	case VNil:
		return "nil"
	case VBool:
		if v.bval {
			return "true"
		}
		return "false"
	case VInt:
		return fmt.Sprintf("%d", v.ival)
	case VFloat:
		return fmt.Sprintf("%g", v.fval)
	case VString:
		return fmt.Sprintf("%q", v.sval)
	case VList:
		parts := make([]string, len(v.lval))
		for i, el := range v.lval {
			parts[i] = el.String()
		}
		return "[" + strings.Join(parts, " ") + "]"
	default:
		return "<?>"
	}
}

// Equal checks value equality.
func (v Value) Equal(other Value) bool {
	if v.kind != other.kind {
		return false
	}
	switch v.kind {
	case VNil:
		return true
	case VBool:
		return v.bval == other.bval
	case VInt:
		return v.ival == other.ival
	case VFloat:
		return v.fval == other.fval
	case VString:
		return v.sval == other.sval
	case VList:
		if len(v.lval) != len(other.lval) {
			return false
		}
		for i := range v.lval {
			if !v.lval[i].Equal(other.lval[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
