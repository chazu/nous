// Package unit defines the Unit type — a named entity with typed property slots.
// This is the direct equivalent of EURISKO's Interlisp property-list units.
package unit

import "fmt"

// Unit is a named entity with dynamic typed slots.
type Unit struct {
	Name  string
	Slots map[string]any
}

// New creates a unit with the given name and empty slots.
func New(name string) *Unit {
	return &Unit{
		Name:  name,
		Slots: make(map[string]any),
	}
}

// Get returns a slot value, or nil if not present.
func (u *Unit) Get(slot string) any {
	return u.Slots[slot]
}

// GetInt returns a slot as int, or 0.
func (u *Unit) GetInt(slot string) int {
	v, _ := u.Slots[slot].(int)
	return v
}

// GetFloat returns a slot as float64, or 0.
func (u *Unit) GetFloat(slot string) float64 {
	switch v := u.Slots[slot].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

// GetString returns a slot as string, or "".
func (u *Unit) GetString(slot string) string {
	v, _ := u.Slots[slot].(string)
	return v
}

// GetStrings returns a slot as []string, or nil.
func (u *Unit) GetStrings(slot string) []string {
	v, _ := u.Slots[slot].([]string)
	return v
}

// GetBool returns a slot as bool, or false.
func (u *Unit) GetBool(slot string) bool {
	v, _ := u.Slots[slot].(bool)
	return v
}

// GetMap returns a slot as map[string]any, or nil.
func (u *Unit) GetMap(slot string) map[string]any {
	v, _ := u.Slots[slot].(map[string]any)
	return v
}

// Set sets a slot value.
func (u *Unit) Set(slot string, value any) {
	u.Slots[slot] = value
}

// Has returns true if the slot exists (even if nil).
func (u *Unit) Has(slot string) bool {
	_, ok := u.Slots[slot]
	return ok
}

// Worth returns the unit's worth, or 0 if unset.
func (u *Unit) Worth() int {
	return u.GetInt("worth")
}

// SetWorth sets the unit's worth.
func (u *Unit) SetWorth(w int) {
	if w < 0 {
		w = 0
	}
	if w > 1000 {
		w = 1000
	}
	u.Slots["worth"] = w
}

// IsA returns the unit's isA list.
func (u *Unit) IsA() []string {
	return u.GetStrings("isA")
}

func (u *Unit) String() string {
	return fmt.Sprintf("Unit(%s worth:%d)", u.Name, u.Worth())
}
