package unit

import (
	"encoding/json"
	"reflect"
	"sort"
	"sync"
)

// Store holds all units in memory, keyed by name.
type Store struct {
	mu       sync.RWMutex
	units    map[string]*Unit
	inverses map[string]string // slot name -> inverse slot name
}

// NewStore creates an empty unit store.
func NewStore() *Store {
	return &Store{
		units:    make(map[string]*Unit),
		inverses: make(map[string]string),
	}
}

// Clone returns an independent in-memory copy of the Store. Unit names and
// slot value types are preserved, including the registered inverse relation
// table, but subsequent unit and collection mutations cannot affect the
// source Store.
func (s *Store) Clone() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := NewStore()
	for name, source := range s.units {
		target := New(name)
		for slot, value := range source.Slots {
			target.Slots[slot] = cloneSlotValue(reflect.ValueOf(value)).Interface()
		}
		clone.units[name] = target
	}
	for slot, inverse := range s.inverses {
		clone.inverses[slot] = inverse
	}
	return clone
}

func cloneSlotValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(cloneSlotValue(value.Elem()))
		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			result.SetMapIndex(cloneSlotValue(iterator.Key()), cloneSlotValue(iterator.Value()))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneSlotValue(value.Index(index)))
		}
		return result
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneSlotValue(value.Index(index)))
		}
		return result
	default:
		return value
	}
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
	sort.Strings(names)
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
	sort.Strings(result)
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
		"ifAboutToWorkOnTask",
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

// RegisterInverse registers a bidirectional inverse relationship between two slots.
func (s *Store) RegisterInverse(slot, inverse string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inverses[slot] = inverse
	s.inverses[inverse] = slot
}

// SetSlot sets a slot on a unit and maintains inverse relationships.
func (s *Store) SetSlot(unitName, slotName string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := s.units[unitName]
	if u == nil {
		return
	}
	u.Set(slotName, value)

	// Check for inverse maintenance
	invSlot, ok := s.inverses[slotName]
	if !ok {
		return
	}

	// Extract unit references from the value
	var refs []string
	switch v := value.(type) {
	case []string:
		refs = v
	case string:
		refs = []string{v}
	default:
		return
	}

	// Add unitName to the inverse slot on each referenced unit
	for _, ref := range refs {
		target := s.units[ref]
		if target == nil {
			continue
		}
		existing := target.GetStrings(invSlot)
		// Don't add duplicates
		found := false
		for _, e := range existing {
			if e == unitName {
				found = true
				break
			}
		}
		if !found {
			target.Set(invSlot, append(existing, unitName))
		}
	}
}

// Count returns the number of units in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.units)
}

// CanonicalJSON returns a deterministic JSON snapshot of every unit and slot.
// encoding/json sorts string map keys, including nested slot maps.
func (s *Store) CanonicalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]map[string]any, len(s.units))
	for name, u := range s.units {
		slots := make(map[string]any, len(u.Slots))
		for slot, value := range u.Slots {
			slots[slot] = value
		}
		snapshot[name] = slots
	}
	return json.Marshal(snapshot)
}
