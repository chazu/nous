// Package tinystack implements the pure, bounded integer-stack semantics used
// by the tiny-stack vocabulary. It does not depend on units, the DSL, or the
// reasoning engine.
package tinystack

import "fmt"

const (
	MaxInputDepth  = 4
	MaxStackDepth  = 7
	MaxInputAbs    = 100
	MaxValueAbs    = 100_000_000
	MaxProgramSize = 3
)

type Opcode string

const (
	Dup    Opcode = "dup"
	Swap   Opcode = "swap"
	Drop   Opcode = "drop"
	Over   Opcode = "over"
	Add    Opcode = "add"
	Mul    Opcode = "mul"
	Double Opcode = "double"
	Neg    Opcode = "neg"
)

func ValidOpcode(opcode Opcode) bool {
	switch opcode {
	case Dup, Swap, Drop, Over, Add, Mul, Double, Neg:
		return true
	default:
		return false
	}
}

func ValidStack(stack []int) bool {
	return validStack(stack, MaxStackDepth, MaxValueAbs)
}

func ValidInput(stack []int) bool {
	return validStack(stack, MaxInputDepth, MaxInputAbs)
}

func validStack(stack []int, maxDepth, maxAbs int) bool {
	if len(stack) > maxDepth {
		return false
	}
	for _, value := range stack {
		if value < -maxAbs || value > maxAbs {
			return false
		}
	}
	return true
}

func Step(stack []int, opcode Opcode) ([]int, error) {
	if !ValidStack(stack) {
		return nil, fmt.Errorf("invalid stack")
	}
	if !ValidOpcode(opcode) {
		return nil, fmt.Errorf("unknown opcode %q", opcode)
	}
	result := append([]int(nil), stack...)
	switch opcode {
	case Dup:
		if len(result) < 1 || len(result) == MaxStackDepth {
			return nil, fmt.Errorf("dup is undefined")
		}
		result = append(result, result[len(result)-1])
	case Swap:
		if len(result) < 2 {
			return nil, fmt.Errorf("swap is undefined")
		}
		result[len(result)-1], result[len(result)-2] = result[len(result)-2], result[len(result)-1]
	case Drop:
		if len(result) < 1 {
			return nil, fmt.Errorf("drop is undefined")
		}
		result = result[:len(result)-1]
	case Over:
		if len(result) < 2 || len(result) == MaxStackDepth {
			return nil, fmt.Errorf("over is undefined")
		}
		result = append(result, result[len(result)-2])
	case Add, Mul:
		if len(result) < 2 {
			return nil, fmt.Errorf("binary opcode is undefined")
		}
		left, right := int64(result[len(result)-2]), int64(result[len(result)-1])
		var value int64
		if opcode == Add {
			value = left + right
		} else {
			value = left * right
		}
		if value < -MaxValueAbs || value > MaxValueAbs {
			return nil, fmt.Errorf("arithmetic overflow")
		}
		result = append(result[:len(result)-2], int(value))
	case Double, Neg:
		if len(result) < 1 {
			return nil, fmt.Errorf("unary arithmetic opcode is undefined")
		}
		value := int64(result[len(result)-1])
		if opcode == Double {
			value *= 2
		} else {
			value = -value
		}
		if value < -MaxValueAbs || value > MaxValueAbs {
			return nil, fmt.Errorf("arithmetic overflow")
		}
		result[len(result)-1] = int(value)
	}
	return result, nil
}

func Execute(stack []int, program []Opcode) ([]int, error) {
	if !ValidInput(stack) {
		return nil, fmt.Errorf("invalid input stack")
	}
	if len(program) == 0 || len(program) > MaxProgramSize {
		return nil, fmt.Errorf("invalid program length")
	}
	result := append([]int(nil), stack...)
	var err error
	for _, opcode := range program {
		result, err = Step(result, opcode)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
