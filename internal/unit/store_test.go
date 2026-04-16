package unit

import "testing"

func TestStoreBasics(t *testing.T) {
	s := NewStore()

	u := New("Foo")
	u.SetWorth(500)
	u.Set("isA", []string{"Bar"})
	s.Put(u)

	if !s.Has("Foo") {
		t.Fatal("expected Foo to exist")
	}
	if s.Has("Baz") {
		t.Fatal("expected Baz to not exist")
	}
	if s.Count() != 1 {
		t.Fatalf("expected 1 unit, got %d", s.Count())
	}

	got := s.Get("Foo")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Worth() != 500 {
		t.Errorf("worth: got %d, want 500", got.Worth())
	}
}

func TestStoreDelete(t *testing.T) {
	s := NewStore()
	s.Put(New("A"))
	s.Put(New("B"))

	deleted := s.Delete("A")
	if deleted == nil {
		t.Fatal("Delete returned nil")
	}
	if s.Has("A") {
		t.Fatal("A should be deleted")
	}
	if s.Count() != 1 {
		t.Fatalf("expected 1 unit, got %d", s.Count())
	}
}

func TestIsATransitive(t *testing.T) {
	s := NewStore()

	a := New("Animal")
	a.Set("isA", []string{})
	s.Put(a)

	m := New("Mammal")
	m.Set("isA", []string{"Animal"})
	s.Put(m)

	d := New("Dog")
	d.Set("isA", []string{"Mammal"})
	s.Put(d)

	if !s.IsA("Dog", "Dog") {
		t.Error("Dog should be a Dog")
	}
	if !s.IsA("Dog", "Mammal") {
		t.Error("Dog should be a Mammal")
	}
	if !s.IsA("Dog", "Animal") {
		t.Error("Dog should be an Animal (transitive)")
	}
	if s.IsA("Animal", "Dog") {
		t.Error("Animal should not be a Dog")
	}
}

func TestIsACycle(t *testing.T) {
	s := NewStore()

	a := New("A")
	a.Set("isA", []string{"B"})
	s.Put(a)

	b := New("B")
	b.Set("isA", []string{"A"})
	s.Put(b)

	// Should not infinite loop
	_ = s.IsA("A", "C")
}

func TestExamples(t *testing.T) {
	s := NewStore()

	s.Put(New("Shape"))

	c := New("Circle")
	c.Set("isA", []string{"Shape"})
	s.Put(c)

	sq := New("Square")
	sq.Set("isA", []string{"Shape"})
	s.Put(sq)

	tri := New("Triangle")
	tri.Set("isA", []string{"Shape"})
	s.Put(tri)

	examples := s.Examples("Shape")
	if len(examples) < 3 {
		t.Errorf("expected at least 3 examples of Shape, got %d", len(examples))
	}

	// Shape itself is also an example of Shape (IsA identity)
	found := false
	for _, e := range examples {
		if e == "Circle" {
			found = true
		}
	}
	if !found {
		t.Error("Circle should be an example of Shape")
	}
}

func TestUnitWorthClamped(t *testing.T) {
	u := New("Test")
	u.SetWorth(-10)
	if u.Worth() != 0 {
		t.Errorf("worth should be clamped to 0, got %d", u.Worth())
	}
	u.SetWorth(2000)
	if u.Worth() != 1000 {
		t.Errorf("worth should be clamped to 1000, got %d", u.Worth())
	}
}

// TestWorthClampedViaSet verifies worth is clamped even when written through
// the generic Set/SetSlot path (used by DSL set-slot). Previously these
// bypassed SetWorth's clamp and allowed worth values like -3700.
func TestWorthClampedViaSet(t *testing.T) {
	u := New("Test")
	u.Set("worth", -500)
	if u.Worth() != 0 {
		t.Errorf("Set(worth,-500) should clamp to 0, got %d", u.Worth())
	}
	u.Set("worth", 5000)
	if u.Worth() != 1000 {
		t.Errorf("Set(worth,5000) should clamp to 1000, got %d", u.Worth())
	}

	s := NewStore()
	u2 := New("Test2")
	s.Put(u2)
	s.SetSlot("Test2", "worth", -1000)
	if u2.Worth() != 0 {
		t.Errorf("SetSlot(Test2,worth,-1000) should clamp to 0, got %d", u2.Worth())
	}
}

func TestInverseMaintenance(t *testing.T) {
	s := NewStore()
	s.RegisterInverse("range", "isRangeOf")

	// Create target type
	set := New("Set")
	set.Set("isA", []string{"Anything"})
	s.Put(set)

	// Create operation with range
	op := New("SetUnion")
	op.Set("isA", []string{"Op"})
	s.Put(op)

	// Set range via SetSlot
	s.SetSlot("SetUnion", "range", []string{"Set"})

	// Verify inverse was maintained
	isRangeOf := s.Get("Set").GetStrings("isRangeOf")
	if len(isRangeOf) != 1 || isRangeOf[0] != "SetUnion" {
		t.Errorf("expected isRangeOf=[SetUnion], got %v", isRangeOf)
	}

	// Add another op with same range
	op2 := New("SetIntersect")
	s.Put(op2)
	s.SetSlot("SetIntersect", "range", []string{"Set"})

	isRangeOf = s.Get("Set").GetStrings("isRangeOf")
	if len(isRangeOf) != 2 {
		t.Errorf("expected 2 entries in isRangeOf, got %d", len(isRangeOf))
	}

	// No duplicates on repeated set
	s.SetSlot("SetUnion", "range", []string{"Set"})
	isRangeOf = s.Get("Set").GetStrings("isRangeOf")
	if len(isRangeOf) != 2 {
		t.Errorf("expected still 2 entries (no dup), got %d", len(isRangeOf))
	}
}

func TestInverseMaintenanceSingleString(t *testing.T) {
	s := NewStore()
	s.RegisterInverse("generalizations", "specializations")

	parent := New("Number")
	s.Put(parent)

	child := New("PrimeNum")
	s.Put(child)

	// Set generalizations as single string
	s.SetSlot("PrimeNum", "generalizations", "Number")

	specs := s.Get("Number").GetStrings("specializations")
	found := false
	for _, sp := range specs {
		if sp == "PrimeNum" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Number.specializations to contain PrimeNum, got %v", specs)
	}
}

func TestUnitSlotAccessors(t *testing.T) {
	u := New("Test")
	u.Set("name", "hello")
	u.Set("count", 42)
	u.Set("rate", 3.14)
	u.Set("active", true)
	u.Set("tags", []string{"a", "b"})
	u.Set("meta", map[string]any{"x": 1})

	if u.GetString("name") != "hello" {
		t.Error("GetString failed")
	}
	if u.GetInt("count") != 42 {
		t.Error("GetInt failed")
	}
	if u.GetFloat("rate") != 3.14 {
		t.Error("GetFloat failed")
	}
	if u.GetBool("active") != true {
		t.Error("GetBool failed")
	}
	if len(u.GetStrings("tags")) != 2 {
		t.Error("GetStrings failed")
	}
	if u.GetMap("meta") == nil {
		t.Error("GetMap failed")
	}
	if !u.Has("name") {
		t.Error("Has failed")
	}
	if u.Has("missing") {
		t.Error("Has should return false for missing slot")
	}
}
