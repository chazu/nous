package dsl

import (
	"fmt"
	"io"
	"os"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

// VM is the stack-based interpreter for nous heuristic programs.
type VM struct {
	stack []Value
	env   map[string]Value
	Store *unit.Store
	Ag    *agenda.Agenda
	Out   io.Writer

	// Set by the engine before firing rules
	DeletedUnits    []string
	DeletedSnapshots map[string]map[string]any // name -> slot snapshot at death
	NewUnits        []string
}

// NewVM creates a VM wired to the given store and agenda.
func NewVM(store *unit.Store, ag *agenda.Agenda) *VM {
	return &VM{
		env:   make(map[string]Value),
		Store: store,
		Ag:    ag,
		Out:   os.Stdout,
	}
}

// SetEnv sets a variable binding visible to executing programs.
func (vm *VM) SetEnv(name string, v Value) {
	vm.env[name] = v
}

// GetEnv returns a variable binding.
func (vm *VM) GetEnv(name string) Value {
	v, ok := vm.env[name]
	if !ok {
		return Nil()
	}
	return v
}

// Execute runs a DSL program string and returns the top of stack (or nil).
func (vm *VM) Execute(program string) (Value, error) {
	tokens := Tokenize(program)
	vm.stack = vm.stack[:0] // reset stack
	err := vm.run(tokens, 0, len(tokens))
	if err != nil {
		return Nil(), err
	}
	if len(vm.stack) == 0 {
		return Nil(), nil
	}
	return vm.stack[len(vm.stack)-1], nil
}

func (vm *VM) run(tokens []Token, start, end int) error {
	i := start
	for i < end {
		tok := tokens[i]

		switch tok.Kind {
		case TokInt:
			vm.push(IntVal(tok.Int))
			i++
		case TokFloat:
			vm.push(FloatVal(tok.Float))
			i++
		case TokString:
			vm.push(StringVal(tok.Text))
			i++
		case TokWord:
			switch tok.Text {
			// Control flow: if ... then ... [else ... ] end
			case "if":
				newI, err := vm.execIf(tokens, i+1, end)
				if err != nil {
					return err
				}
				i = newI
			// Loop: each ... end (pops a list, iterates with "it" bound)
			case "each":
				newI, err := vm.execEach(tokens, i+1, end)
				if err != nil {
					return err
				}
				i = newI
			case "abort":
				return &AbortError{}
			default:
				if err := vm.execWord(tok.Text); err != nil {
					return fmt.Errorf("word %q: %w", tok.Text, err)
				}
				i++
			}
		}
	}
	return nil
}

// execIf handles Forth-style: <cond> if <true-body> [else <false-body>] then
// When we arrive here, the condition result is already on the stack.
func (vm *VM) execIf(tokens []Token, start, end int) (int, error) {
	cond := vm.pop()

	// Find matching else and then, respecting nesting.
	// "if" and "each" increase depth; "then" and "end" decrease it.
	elseIdx := -1
	thenIdx := -1
	depth := 0
	for j := start; j < end; j++ {
		if tokens[j].Kind != TokWord {
			continue
		}
		switch tokens[j].Text {
		case "if", "each":
			depth++
		case "else":
			if depth == 0 {
				elseIdx = j
			}
		case "then", "end":
			if depth == 0 {
				thenIdx = j
				goto found
			}
			depth--
		}
	}
found:
	if thenIdx == -1 {
		return end, fmt.Errorf("malformed if: missing then/end")
	}

	if cond.Truthy() {
		bodyEnd := elseIdx
		if bodyEnd == -1 {
			bodyEnd = thenIdx
		}
		if err := vm.run(tokens, start, bodyEnd); err != nil {
			return 0, err
		}
	} else if elseIdx != -1 {
		if err := vm.run(tokens, elseIdx+1, thenIdx); err != nil {
			return 0, err
		}
	}
	return thenIdx + 1, nil
}

// execEach handles: <list> each <body> end
// Binds "it" to each element.
func (vm *VM) execEach(tokens []Token, start, end int) (int, error) {
	list := vm.pop()
	if list.IsNil() {
		// Skip to end
		depth := 0
		for j := start; j < end; j++ {
			if tokens[j].Kind == TokWord {
				switch tokens[j].Text {
				case "if", "each":
					depth++
				case "then", "end":
					// "then" closes "if", "end" closes "each"
					if tokens[j].Text == "end" && depth == 0 {
						return j + 1, nil
					}
					depth--
				}
			}
		}
		return end, nil
	}

	// Find matching end
	endIdx := -1
	depth := 0
	for j := start; j < end; j++ {
		if tokens[j].Kind == TokWord {
			switch tokens[j].Text {
			case "if", "each":
				depth++
			case "then", "end":
				// "then" closes "if", "end" closes "each"
				if tokens[j].Text == "end" && depth == 0 {
					endIdx = j
					goto found
				}
				depth--
			}
		}
	}
found:
	if endIdx == -1 {
		return end, fmt.Errorf("malformed each/end")
	}

	// Snapshot the list to avoid mutation during iteration
	items := list.AsList()
	snapshot := make([]Value, len(items))
	copy(snapshot, items)

	oldIt := vm.env["it"]
	for _, item := range snapshot {
		vm.env["it"] = item
		if err := vm.run(tokens, start, endIdx); err != nil {
			return 0, err
		}
	}
	vm.env["it"] = oldIt

	return endIdx + 1, nil
}

func (vm *VM) execWord(word string) error {
	fn, ok := builtins[word]
	if ok {
		return fn(vm)
	}
	return fmt.Errorf("unknown word: %s", word)
}

// Stack operations

func (vm *VM) push(v Value) {
	vm.stack = append(vm.stack, v)
}

func (vm *VM) pop() Value {
	if len(vm.stack) == 0 {
		return Nil()
	}
	v := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return v
}

func (vm *VM) peek() Value {
	if len(vm.stack) == 0 {
		return Nil()
	}
	return vm.stack[len(vm.stack)-1]
}

func (vm *VM) depth() int {
	return len(vm.stack)
}

// AbortError signals that a heuristic wants to abort the current task.
type AbortError struct{}

func (e *AbortError) Error() string { return "AbortTask" }

// IsAbort checks if an error is an AbortError.
func IsAbort(err error) bool {
	_, ok := err.(*AbortError)
	return ok
}
