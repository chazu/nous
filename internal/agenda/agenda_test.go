package agenda

import "testing"

func TestAgendaBasics(t *testing.T) {
	ag := New()

	if ag.Len() != 0 {
		t.Fatal("new agenda should be empty")
	}
	if ag.Pop() != nil {
		t.Fatal("Pop on empty should return nil")
	}

	ag.Push(&Task{Priority: 100, UnitName: "A", SlotName: "examples", Reasons: []string{"r1"}})
	ag.Push(&Task{Priority: 300, UnitName: "B", SlotName: "examples", Reasons: []string{"r2"}})
	ag.Push(&Task{Priority: 200, UnitName: "C", SlotName: "domain", Reasons: []string{"r3"}})

	if ag.Len() != 3 {
		t.Fatalf("expected 3 tasks, got %d", ag.Len())
	}

	// Should pop in priority order (highest first)
	t1 := ag.Pop()
	if t1.UnitName != "B" || t1.Priority != 300 {
		t.Errorf("expected B/300, got %s/%d", t1.UnitName, t1.Priority)
	}
	t2 := ag.Pop()
	if t2.UnitName != "C" || t2.Priority != 200 {
		t.Errorf("expected C/200, got %s/%d", t2.UnitName, t2.Priority)
	}
	t3 := ag.Pop()
	if t3.UnitName != "A" || t3.Priority != 100 {
		t.Errorf("expected A/100, got %s/%d", t3.UnitName, t3.Priority)
	}
}

func TestAgendaMerge(t *testing.T) {
	ag := New()

	ag.Push(&Task{Priority: 100, UnitName: "X", SlotName: "examples", Reasons: []string{"first"}})
	ag.Push(&Task{Priority: 200, UnitName: "X", SlotName: "examples", Reasons: []string{"second"}})

	// Should have merged into one task
	if ag.Len() != 1 {
		t.Fatalf("expected 1 task after merge, got %d", ag.Len())
	}

	task := ag.Pop()
	// Priority should be max(100,200) + 50 = 250
	if task.Priority != 250 {
		t.Errorf("expected priority 250 after merge, got %d", task.Priority)
	}
	if len(task.Reasons) != 2 {
		t.Errorf("expected 2 reasons after merge, got %d", len(task.Reasons))
	}
}

func TestAgendaMergeDifferentSlots(t *testing.T) {
	ag := New()

	ag.Push(&Task{Priority: 100, UnitName: "X", SlotName: "examples", Reasons: []string{"r1"}})
	ag.Push(&Task{Priority: 100, UnitName: "X", SlotName: "domain", Reasons: []string{"r2"}})

	// Different slots = different tasks, no merge
	if ag.Len() != 2 {
		t.Fatalf("expected 2 tasks (different slots), got %d", ag.Len())
	}
}

func TestAgendaPeek(t *testing.T) {
	ag := New()
	if ag.Peek() != nil {
		t.Fatal("Peek on empty should return nil")
	}

	ag.Push(&Task{Priority: 100, UnitName: "A", SlotName: "s"})
	ag.Push(&Task{Priority: 500, UnitName: "B", SlotName: "s"})

	p := ag.Peek()
	if p.UnitName != "B" {
		t.Errorf("Peek should return highest priority, got %s", p.UnitName)
	}
	// Peek shouldn't remove
	if ag.Len() != 2 {
		t.Error("Peek should not remove")
	}
}

func TestAgendaPriorityCapped(t *testing.T) {
	ag := New()

	ag.Push(&Task{Priority: 990, UnitName: "X", SlotName: "s", Reasons: []string{"r1"}})
	ag.Push(&Task{Priority: 990, UnitName: "X", SlotName: "s", Reasons: []string{"r2"}})

	task := ag.Pop()
	if task.Priority > 1000 {
		t.Errorf("priority should be capped at 1000, got %d", task.Priority)
	}
}
