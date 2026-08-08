// Package taskbridge is the narrow capability seam between one causal CUE VM
// and its hidden-free runner. It knows no fixture, profile, artifact, teacher,
// or runner type.
package taskbridge

import (
	"errors"
	"fmt"
	"sync"
)

type handler struct {
	valid     func(string) bool
	begin     func(string) error
	operation func(string, ...string) (any, error)
	end       func(string) error
}

// Scope is an unforgeable, VM-local bearer capability. A runtime name alone is
// deliberately insufficient to reach a handler: callers must possess the
// exact Scope installed on that runner's VM.
type Scope struct {
	sync.RWMutex
	handlers map[string]handler
}

func NewScope() *Scope { return &Scope{handlers: make(map[string]handler)} }

func (s *Scope) Register(name string, valid func(string) bool, begin func(string) error, operation func(string, ...string) (any, error), end func(string) error) error {
	if s == nil {
		return errors.New("nil causal task capability scope")
	}
	if name == "" || valid == nil || begin == nil || operation == nil || end == nil {
		return errors.New("invalid causal task capability")
	}
	s.Lock()
	defer s.Unlock()
	if _, exists := s.handlers[name]; exists {
		return fmt.Errorf("causal task capability %q already registered", name)
	}
	s.handlers[name] = handler{valid: valid, begin: begin, operation: operation, end: end}
	return nil
}

func (s *Scope) Unregister(name string) {
	if s == nil {
		return
	}
	s.Lock()
	defer s.Unlock()
	delete(s.handlers, name)
}

func (s *Scope) Valid(name, slot string) bool {
	if s == nil {
		return false
	}
	s.RLock()
	handler := s.handlers[name]
	s.RUnlock()
	return handler.valid != nil && handler.valid(slot)
}

func (s *Scope) Begin(name, slot string) error {
	if s == nil {
		return errors.New("causal VM has no task capability scope")
	}
	s.RLock()
	handler := s.handlers[name]
	s.RUnlock()
	if handler.begin == nil {
		return fmt.Errorf("no causal task capability for %q", name)
	}
	if !handler.valid(slot) {
		return fmt.Errorf("invalid causal task %s.%s", name, slot)
	}
	return handler.begin(slot)
}

func (s *Scope) Operation(name, operation string, arguments ...string) (any, error) {
	if s == nil {
		return nil, errors.New("causal VM has no task capability scope")
	}
	s.RLock()
	handler := s.handlers[name]
	s.RUnlock()
	if handler.operation == nil {
		return nil, fmt.Errorf("no causal task capability for %q", name)
	}
	return handler.operation(operation, arguments...)
}

func (s *Scope) End(name, slot string) error {
	if s == nil {
		return errors.New("causal VM has no task capability scope")
	}
	s.RLock()
	handler := s.handlers[name]
	s.RUnlock()
	if handler.end == nil {
		return fmt.Errorf("no causal task capability for %q", name)
	}
	return handler.end(slot)
}
