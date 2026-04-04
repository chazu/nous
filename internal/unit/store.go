package unit

import "sync"

// Store holds all units in memory, keyed by name.
type Store struct {
	mu    sync.RWMutex
	units map[string]*Unit
}

// NewStore creates an empty unit store.
func NewStore() *Store {
	return &Store{units: make(map[string]*Unit)}
}

// Get returns a unit by name, or nil.
func (s *Store) Get(name string) *Unit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.units[name]
}

// Put adds or replaces a unit.
func (s *Store) Put(u *Unit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.units[u.Name] = u
}

// Delete removes a unit by name. Returns the deleted unit, or nil.
func (s *Store) Delete(name string) *Unit {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.units[name]
	delete(s.units, name)
	return u
}

// Has returns true if a unit exists.
func (s *Store) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.units[name]
	return ok
}

// All returns all unit names.
func (s *Store) All() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.units))
	for name := range s.units {
		names = append(names, name)
	}
	return names
}

// IsA checks if unitName is a kind of category, walking the isA chain transitively.
func (s *Store) IsA(unitName, category string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isA(unitName, category, make(map[string]bool))
}

func (s *Store) isA(unitName, category string, visited map[string]bool) bool {
	if unitName == category {
		return true
	}
	if visited[unitName] {
		return false
	}
	visited[unitName] = true

	u := s.units[unitName]
	if u == nil {
		return false
	}
	for _, parent := range u.GetStrings("isA") {
		if parent == category {
			return true
		}
		if s.isA(parent, category, visited) {
			return true
		}
	}
	for _, gen := range u.GetStrings("generalizations") {
		if gen == category {
			return true
		}
		if s.isA(gen, category, visited) {
			return true
		}
	}
	return false
}

// Examples returns all unit names that are instances of the given category.
func (s *Store) Examples(category string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []string
	for name := range s.units {
		if s.isA(name, category, make(map[string]bool)) {
			result = append(result, name)
		}
	}
	return result
}

// Generalizations returns the generalizations slot of a unit.
func (s *Store) Generalizations(name string) []string {
	u := s.Get(name)
	if u == nil {
		return nil
	}
	return u.GetStrings("generalizations")
}

// Specializations returns the specializations slot of a unit.
func (s *Store) Specializations(name string) []string {
	u := s.Get(name)
	if u == nil {
		return nil
	}
	return u.GetStrings("specializations")
}

// IfPartSlots returns the canonical ordering of condition slots.
func IfPartSlots() []string {
	return []string{
		"ifPotentiallyRelevant",
		"ifTrulyRelevant",
		"ifWorkingOnTask",
		"ifFinishedWorkingOnTask",
	}
}

// ThenPartSlots returns the canonical ordering of action slots.
func ThenPartSlots() []string {
	return []string{
		"thenCompute",
		"thenAddToAgenda",
		"thenDefineNewConcepts",
		"thenDeleteOldConcepts",
		"thenPrintToUser",
		"thenConjecture",
	}
}

// Count returns the number of units in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.units)
}
