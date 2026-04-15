package cueload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSingleUnit(t *testing.T) {
	dir := t.TempDir()
	src := `
units: [{
	name:  "Addition"
	worth: 500
	isA:   ["MathOp", "BinaryOp"]
}]
`
	if err := os.WriteFile(filepath.Join(dir, "math.cue"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 unit, got %d", len(defs))
	}
	u := defs[0]
	if u.Name != "Addition" {
		t.Errorf("name = %q, want Addition", u.Name)
	}
	if u.Worth != 500 {
		t.Errorf("worth = %d, want 500", u.Worth)
	}
	if len(u.IsA) != 2 || u.IsA[0] != "MathOp" || u.IsA[1] != "BinaryOp" {
		t.Errorf("isA = %v, want [MathOp BinaryOp]", u.IsA)
	}
}

func TestLoadMultipleFiles(t *testing.T) {
	dir := t.TempDir()

	file1 := `units: [{name: "A", worth: 100, isA: ["X"]}]`
	file2 := `units: [{name: "B", worth: 200, isA: ["Y"]}, {name: "C", worth: 300, isA: []}]`

	os.WriteFile(filepath.Join(dir, "a.cue"), []byte(file1), 0644)
	os.WriteFile(filepath.Join(dir, "b.cue"), []byte(file2), 0644)

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 3 {
		t.Fatalf("want 3 units, got %d", len(defs))
	}

	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, want := range []string{"A", "B", "C"} {
		if !names[want] {
			t.Errorf("missing unit %q", want)
		}
	}
}

func TestLoadUnitWithSlots(t *testing.T) {
	dir := t.TempDir()
	src := `
units: [{
	name:  "Primes"
	worth: 700
	isA:   ["Concept"]
	data:  [2, 3, 5, 7, 11]
	defn:  "integers > 1 with no divisors other than 1 and themselves"
	overallRecord: {successes: 10, failures: 2}
	english: "prime numbers"
}]
`
	os.WriteFile(filepath.Join(dir, "primes.cue"), []byte(src), 0644)

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 unit, got %d", len(defs))
	}
	u := defs[0]

	// data should be []int
	data, ok := u.Slots["data"].([]int)
	if !ok {
		t.Fatalf("data type = %T, want []int", u.Slots["data"])
	}
	if len(data) != 5 || data[0] != 2 || data[4] != 11 {
		t.Errorf("data = %v, want [2 3 5 7 11]", data)
	}

	// defn should be string
	defn, ok := u.Slots["defn"].(string)
	if !ok || defn == "" {
		t.Errorf("defn = %v (%T), want non-empty string", u.Slots["defn"], u.Slots["defn"])
	}

	// overallRecord should be map[string]any
	rec, ok := u.Slots["overallRecord"].(map[string]any)
	if !ok {
		t.Fatalf("overallRecord type = %T, want map[string]any", u.Slots["overallRecord"])
	}
	if rec["successes"] != 10 || rec["failures"] != 2 {
		t.Errorf("overallRecord = %v, want {successes:10 failures:2}", rec)
	}

	// english should be string in slots
	eng, ok := u.Slots["english"].(string)
	if !ok || eng != "prime numbers" {
		t.Errorf("english = %v, want prime numbers", u.Slots["english"])
	}
}
