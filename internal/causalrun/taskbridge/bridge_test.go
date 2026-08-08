package taskbridge

import "testing"

func TestScopesAreUnforgeableByRuntimeName(t *testing.T) {
	left, right := NewScope(), NewScope()
	calls := 0
	if err := left.Register("predictable-name", func(slot string) bool { return slot == "task" }, func(string) error {
		calls++
		return nil
	}, func(string, ...string) (any, error) {
		calls++
		return true, nil
	}, func(string) error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if right.Valid("predictable-name", "task") {
		t.Fatal("runtime name crossed an unrelated capability scope")
	}
	if err := right.Begin("predictable-name", "task"); err == nil {
		t.Fatal("unrelated scope invoked a handler by guessed name")
	}
	if !left.Valid("predictable-name", "task") {
		t.Fatal("owning scope cannot see its handler")
	}
	if err := left.Begin("predictable-name", "task"); err != nil {
		t.Fatal(err)
	}
	if _, err := left.Operation("predictable-name", "operation"); err != nil {
		t.Fatal(err)
	}
	if err := left.End("predictable-name", "task"); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("handler calls=%d, want 3", calls)
	}
	left.Unregister("predictable-name")
	if left.Valid("predictable-name", "task") {
		t.Fatal("unregistered capability remains reachable")
	}
}
