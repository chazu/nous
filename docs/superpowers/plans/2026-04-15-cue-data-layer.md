# CUE Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move domain definitions from hardcoded Go into CUE files loaded at runtime, with embedded fallback.

**Architecture:** CUE files in `domains/` define units validated against a `#Unit` schema. A Go loader (`internal/cueload/`) reads them into `[]UnitDef` structs. `internal/seed/registry.go` creates units from those defs. `-domains-dir` flag enables filesystem loading for development; embedded files are the default.

**Tech Stack:** Go, `cuelang.org/go` v0.16.0 (already in go.mod), `//go:embed`

---

### Task 1: CUE Schema

Create the `#Unit` schema that all domain files validate against.

**Files:**
- Create: `domains/schema.cue`

- [ ] **Step 1: Create domains directory**

```bash
mkdir -p domains
```

- [ ] **Step 2: Write the schema**

Create `domains/schema.cue`:

```cue
package domains

#Unit: {
	name:  string
	worth: int & >=0 & <=1000

	isA: [...string]

	// Known slots with types
	english?:         string
	abbrev?:          string
	domain?:          [...string] | string
	range?:           [...string] | string
	arity?:           int
	defn?:            string
	data?:            [...int] | [...string]
	examples?:        _
	nonExamples?:     _
	generalizations?: [...string]
	specializations?: [...string]
	creditors?:       [...string]
	inverse?:         string
	status?:          string
	overallRecord?: {successes: int, failures: int}

	// Heuristic program slots (stack DSL strings)
	ifPotentiallyRelevant?:   string
	ifTrulyRelevant?:         string
	ifWorkingOnTask?:         string
	ifFinishedWorkingOnTask?: string
	thenCompute?:             string
	thenAddToAgenda?:         string
	thenDefineNewConcepts?:   string
	thenDeleteOldConcepts?:   string
	thenPrintToUser?:         string
	thenConjecture?:          string
	thenModifySlots?:         string

	// Open for novel slots
	...
}
```

- [ ] **Step 3: Commit**

```bash
git add domains/schema.cue
git commit -m "feat: add CUE schema for unit definitions"
```

---

### Task 2: CUE Loader Package

Go package that reads CUE files and returns unit definitions.

**Files:**
- Create: `internal/cueload/cueload.go`
- Create: `internal/cueload/cueload_test.go`
- Create: `internal/cueload/embed.go`

- [ ] **Step 1: Write the test**

Create `internal/cueload/cueload_test.go`:

```go
package cueload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDomainFromDir(t *testing.T) {
	// Create a temp directory with a CUE file
	dir := t.TempDir()
	cueContent := `
package test

import "domains"

units: [...domains.#Unit]

units: [{
	name:  "TestUnit"
	worth: 500
	isA:   ["Anything"]
	english: "A test unit"
}]
`
	if err := os.WriteFile(filepath.Join(dir, "test.cue"), []byte(cueContent), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(defs))
	}
	if defs[0].Name != "TestUnit" {
		t.Errorf("expected name TestUnit, got %s", defs[0].Name)
	}
	if defs[0].Worth != 500 {
		t.Errorf("expected worth 500, got %d", defs[0].Worth)
	}
	if len(defs[0].IsA) != 1 || defs[0].IsA[0] != "Anything" {
		t.Errorf("expected isA [Anything], got %v", defs[0].IsA)
	}
	english, ok := defs[0].Slots["english"]
	if !ok || english != "A test unit" {
		t.Errorf("expected english slot 'A test unit', got %v", english)
	}
}

func TestLoadDomainMultipleFiles(t *testing.T) {
	dir := t.TempDir()

	file1 := `
package test

import "domains"

units: [...domains.#Unit]

units: [{
	name:  "Unit1"
	worth: 400
	isA:   ["Anything"]
}]
`
	file2 := `
package test

import "domains"

units: [...domains.#Unit]

units: [{
	name:  "Unit2"
	worth: 600
	isA:   ["Anything"]
}]
`
	os.WriteFile(filepath.Join(dir, "a.cue"), []byte(file1), 0644)
	os.WriteFile(filepath.Join(dir, "b.cue"), []byte(file2), 0644)

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("expected 2 units from 2 files, got %d", len(defs))
	}
}

func TestLoadDomainWithDataSlot(t *testing.T) {
	dir := t.TempDir()
	cueContent := `
package test

import "domains"

units: [...domains.#Unit]

units: [{
	name:  "Primes"
	worth: 600
	isA:   ["Set"]
	data:  [2, 3, 5, 7]
	defn:  "each it prime? if it then end make-set"
}]
`
	os.WriteFile(filepath.Join(dir, "test.cue"), []byte(cueContent), 0644)

	defs, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(defs))
	}

	data, ok := defs[0].Slots["data"]
	if !ok {
		t.Fatal("expected data slot")
	}
	dataSlice, ok := data.([]int)
	if !ok {
		t.Fatalf("expected []int, got %T", data)
	}
	if len(dataSlice) != 4 {
		t.Errorf("expected 4 elements, got %d", len(dataSlice))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/cueload/ -v`
Expected: FAIL (package doesn't exist)

- [ ] **Step 3: Create the embed file for schema**

Create `internal/cueload/embed.go`:

```go
package cueload

import "embed"

//go:embed all:../../domains
var embeddedDomains embed.FS
```

Note: this embeds the entire `domains/` directory. The `all:` prefix includes files starting with `.` or `_`.

- [ ] **Step 4: Implement the loader**

Create `internal/cueload/cueload.go`:

```go
// Package cueload reads CUE domain files and returns unit definitions.
package cueload

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// UnitDef is a unit definition loaded from CUE.
type UnitDef struct {
	Name  string
	Worth int
	IsA   []string
	Slots map[string]any
}

// LoadDir reads all .cue files from a filesystem directory and returns unit definitions.
func LoadDir(dir string) ([]UnitDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading domain dir %s: %w", dir, err)
	}

	// Read the schema
	schemaSource, err := getSchemaSource()
	if err != nil {
		return nil, fmt.Errorf("reading schema: %w", err)
	}

	var allDefs []UnitDef
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cue") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		defs, err := parseUnits(schemaSource, string(data), entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

// LoadEmbedded reads CUE files from the embedded filesystem for a domain.
func LoadEmbedded(domainName string) ([]UnitDef, error) {
	dir := domainName
	entries, err := fs.ReadDir(embeddedDomains, dir)
	if err != nil {
		return nil, fmt.Errorf("reading embedded domain %s: %w", domainName, err)
	}

	schemaSource, err := getSchemaSource()
	if err != nil {
		return nil, fmt.Errorf("reading schema: %w", err)
	}

	var allDefs []UnitDef
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cue") {
			continue
		}
		data, err := fs.ReadFile(embeddedDomains, filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		defs, err := parseUnits(schemaSource, string(data), entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		allDefs = append(allDefs, defs...)
	}
	return allDefs, nil
}

func getSchemaSource() (string, error) {
	// Try embedded first
	data, err := fs.ReadFile(embeddedDomains, "schema.cue")
	if err != nil {
		return "", fmt.Errorf("schema.cue not found in embedded files: %w", err)
	}
	return string(data), nil
}

func parseUnits(schemaSource, fileSource, filename string) ([]UnitDef, error) {
	ctx := cuecontext.New()

	// Compile schema and file together
	// Strip the package declaration and import from the file -- we compile as a single scope
	combined := schemaSource + "\n" + stripPackageAndImport(fileSource)
	val := ctx.CompileString(combined, cue.Filename(filename))
	if val.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", val.Err())
	}

	unitsList := val.LookupPath(cue.ParsePath("units"))
	if unitsList.Err() != nil {
		return nil, fmt.Errorf("no units list: %w", unitsList.Err())
	}

	iter, err := unitsList.List()
	if err != nil {
		return nil, fmt.Errorf("units is not a list: %w", err)
	}

	var defs []UnitDef
	for iter.Next() {
		def, err := extractUnit(iter.Value())
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, nil
}

func stripPackageAndImport(source string) string {
	var lines []string
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			continue
		}
		if trimmed == `import "domains"` {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func extractUnit(v cue.Value) (UnitDef, error) {
	def := UnitDef{
		Slots: make(map[string]any),
	}

	// Extract name
	nameVal := v.LookupPath(cue.ParsePath("name"))
	if nameVal.Err() != nil {
		return def, fmt.Errorf("unit missing name: %w", nameVal.Err())
	}
	name, _ := nameVal.String()
	def.Name = name

	// Extract worth
	worthVal := v.LookupPath(cue.ParsePath("worth"))
	if worthVal.Err() == nil {
		w, _ := worthVal.Int64()
		def.Worth = int(w)
	}

	// Extract isA
	isaVal := v.LookupPath(cue.ParsePath("isA"))
	if isaVal.Err() == nil {
		iter, err := isaVal.List()
		if err == nil {
			for iter.Next() {
				s, _ := iter.Value().String()
				def.IsA = append(def.IsA, s)
			}
		}
	}

	// Extract all other fields as slots
	iter, _ := v.Fields(cue.All())
	for iter.Next() {
		label := iter.Selector().String()
		if label == "name" || label == "worth" || label == "isA" {
			continue
		}
		val := iter.Value()
		def.Slots[label] = cueToGo(val)
	}

	return def, nil
}

func cueToGo(v cue.Value) any {
	switch v.IncompleteKind() {
	case cue.StringKind:
		s, _ := v.String()
		return s
	case cue.IntKind:
		n, _ := v.Int64()
		return int(n)
	case cue.FloatKind:
		f, _ := v.Float64()
		return f
	case cue.BoolKind:
		b, _ := v.Bool()
		return b
	case cue.ListKind:
		return cueListToGo(v)
	case cue.StructKind:
		m := make(map[string]any)
		iter, _ := v.Fields(cue.All())
		for iter.Next() {
			m[iter.Selector().String()] = cueToGo(iter.Value())
		}
		return m
	default:
		s := fmt.Sprintf("%v", v)
		return s
	}
}

func cueListToGo(v cue.Value) any {
	iter, err := v.List()
	if err != nil {
		return nil
	}

	// Peek at first element to determine type
	var ints []int
	var strs []string
	var mixed []any
	allInt := true
	allStr := true

	for iter.Next() {
		elem := iter.Value()
		switch elem.IncompleteKind() {
		case cue.IntKind:
			n, _ := elem.Int64()
			ints = append(ints, int(n))
			mixed = append(mixed, int(n))
			allStr = false
		case cue.StringKind:
			s, _ := elem.String()
			strs = append(strs, s)
			mixed = append(mixed, s)
			allInt = false
		default:
			g := cueToGo(elem)
			mixed = append(mixed, g)
			allInt = false
			allStr = false
		}
	}

	if allInt && len(ints) > 0 {
		return ints
	}
	if allStr && len(strs) > 0 {
		return strs
	}
	return mixed
}
```

- [ ] **Step 5: Run tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/cueload/ -v`
Expected: Tests may need adjustment based on how CUE evaluates the combined source. Debug and fix until all 3 tests pass.

- [ ] **Step 6: Run all project tests to check for import issues**

Run: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: All pass

- [ ] **Step 7: Commit**

```bash
git add internal/cueload/
git commit -m "feat: add CUE loader package for domain definitions"
```

---

### Task 3: Common Domain CUE Files

Move base types (Anything, Heuristic, Slot, Op, Pred, etc.) from `math.go` into `domains/common/`.

**Files:**
- Create: `domains/common/types.cue`

- [ ] **Step 1: Create the common types file**

Create `domains/common/types.cue`:

```cue
package common

import "domains"

units: [...domains.#Unit]

units: [
	{name: "Anything", worth: 500, isA: []},
	{name: "Heuristic", worth: 800, isA: ["Anything"]},
	{name: "Slot", worth: 300, isA: ["Anything"]},
	{name: "MathConcept", worth: 500, isA: ["Anything"]},
	{name: "MathObj", worth: 500, isA: ["MathConcept", "Anything"]},
	{name: "MathOp", worth: 500, isA: ["MathConcept", "Anything"]},
	{name: "MathPred", worth: 500, isA: ["MathConcept", "Anything"]},
	{name: "Op", worth: 500, isA: ["Anything"]},
	{name: "BinaryOp", worth: 500, isA: ["Op", "MathOp", "Anything"]},
	{name: "UnaryOp", worth: 500, isA: ["Op", "MathOp", "Anything"]},
	{name: "Pred", worth: 500, isA: ["Anything"]},
	{name: "BinaryPred", worth: 500, isA: ["Pred", "MathPred", "Anything"]},
	{name: "UnaryPred", worth: 500, isA: ["Pred", "MathPred", "Anything"]},
]
```

- [ ] **Step 2: Verify it loads**

Write a quick test or run: `cd /Users/chazu/dev/go/nous && go test ./internal/cueload/ -run TestLoadDir -v` with a test pointing at `domains/common/`.

- [ ] **Step 3: Commit**

```bash
git add domains/common/
git commit -m "feat: add common type hierarchy in CUE"
```

---

### Task 4: Math Domain CUE Files

Move all math domain units from `internal/seed/math.go` into CUE files.

**Files:**
- Create: `domains/math/types.cue`
- Create: `domains/math/sets.cue`
- Create: `domains/math/operations.cue`
- Create: `domains/math/predicates.cue`
- Create: `domains/math/numbers.cue`
- Create: `domains/math/conjectures.cue`

- [ ] **Step 1: Create math types**

Create `domains/math/types.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:  "Structure"
		worth: 600
		isA:   ["MathObj", "Anything"]
		specializations: ["Set", "List", "Bag"]
	},
	{
		name:    "Set"
		worth:   700
		isA:     ["Structure", "MathObj", "Anything"]
		english: "An unordered collection with no duplicate elements"
		specializations: ["EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens"]
	},
	{
		name:    "List"
		worth:   600
		isA:     ["Structure", "MathObj", "Anything"]
		english: "An ordered collection that may contain duplicates"
		specializations: ["SortedList"]
	},
	{name: "Bag", worth: 500, isA: ["Structure", "MathObj", "Anything"]},
	{name: "SortedList", worth: 400, isA: ["List", "Structure", "MathObj", "Anything"]},
	{name: "TruthValue", worth: 400, isA: ["MathObj", "Anything"]},
	{
		name: "Conjecture"
		worth: 500
		isA: ["MathConcept", "Anything"]
		specializations: ["GoldbachConjecture"]
	},
]
```

- [ ] **Step 2: Create concrete sets**

Create `domains/math/sets.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:    "EmptySet"
		worth:   400
		isA:     ["Set", "Structure", "MathObj", "Anything"]
		english: "The set with no elements"
		data:    []
	},
	{
		name:    "SetOfNumbers"
		worth:   600
		isA:     ["Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20"
		data:    [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
		specializations: ["SetOfPrimes", "SetOfEvens", "SetOfOdds"]
	},
	{
		name:    "SetOfPrimes"
		worth:   600
		isA:     ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Prime numbers up to 20"
		data:    [2, 3, 5, 7, 11, 13, 17, 19]
		defn:    "each it prime? if it then end make-set"
		generalizations: ["SetOfNumbers"]
	},
	{
		name:    "SetOfEvens"
		worth:   600
		isA:     ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Even numbers up to 20"
		data:    [2, 4, 6, 8, 10, 12, 14, 16, 18, 20]
		generalizations: ["SetOfNumbers"]
	},
	{
		name:    "SetOfOdds"
		worth:   500
		isA:     ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Odd numbers up to 20"
		data:    [1, 3, 5, 7, 9, 11, 13, 15, 17, 19]
		generalizations: ["SetOfNumbers"]
	},
]
```

- [ ] **Step 3: Create operations**

Create `domains/math/operations.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:    "SetUnion"
		worth:   600
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Set", "Set"]
		range:   ["Set"]
		english: "Combine two sets, keeping all elements"
		defn:    "set-union"
	},
	{
		name:    "SetIntersect"
		worth:   600
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Set", "Set"]
		range:   ["Set"]
		english: "Elements common to both sets"
		defn:    "set-intersect"
	},
	{
		name:    "SetDifference"
		worth:   500
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Set", "Set"]
		range:   ["Set"]
		english: "Elements in first set but not second"
		defn:    "set-diff"
	},
	{
		name:    "DivisorsOf"
		worth:   500
		isA:     ["UnaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number"]
		range:   ["Set"]
		english: "All divisors of a number"
		defn:    "divisors"
	},
	{
		name:    "GCD"
		worth:   500
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number", "Number"]
		range:   ["Number"]
		english: "Greatest common divisor of two numbers"
		defn:    "gcd"
	},
	{
		name:    "Compose"
		worth:   600
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Op", "Op"]
		range:   ["Op"]
		english: "Apply one operation after another"
	},
	{
		name:    "Restrict"
		worth:   500
		isA:     ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Op", "Pred"]
		range:   ["Op"]
		english: "Apply an operation only when a predicate is satisfied"
	},
]
```

- [ ] **Step 4: Create predicates**

Create `domains/math/predicates.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:  "MemberOf"
		worth: 500
		isA:   ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Number", "Set"]
		range:  ["TruthValue"]
		defn:  "swap set-member?"
	},
	{
		name:  "SubsetOf"
		worth: 500
		isA:   ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range:  ["TruthValue"]
		defn:  "set-subset?"
	},
	{
		name:  "SetEqual"
		worth: 500
		isA:   ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range:  ["TruthValue"]
		defn:  "set-equal?"
	},
]
```

- [ ] **Step 5: Create number types**

Create `domains/math/numbers.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:  "Number"
		worth: 600
		isA:   ["MathObj", "Anything"]
		specializations: ["EvenNum", "OddNum", "PrimeNum", "PerfectNum", "SquareNum"]
	},
	{
		name:     "EvenNum"
		worth:    400
		isA:      ["Number", "MathObj", "Anything"]
		defn:     "even?"
		examples: [2, 4, 6, 8, 10]
	},
	{
		name:     "OddNum"
		worth:    400
		isA:      ["Number", "MathObj", "Anything"]
		defn:     "odd?"
		examples: [1, 3, 5, 7, 9]
	},
	{
		name:     "PrimeNum"
		worth:    600
		isA:      ["Number", "MathObj", "Anything"]
		english:  "A number with no divisors other than 1 and itself"
		defn:     "prime?"
		examples: [2, 3, 5, 7, 11, 13, 17, 19, 23, 29]
	},
	{
		name:     "PerfectNum"
		worth:    500
		isA:      ["Number", "MathObj", "Anything"]
		english:  "A number equal to the sum of its proper divisors"
		examples: [6, 28, 496]
	},
	{
		name:     "SquareNum"
		worth:    400
		isA:      ["Number", "MathObj", "Anything"]
		examples: [1, 4, 9, 16, 25, 36]
	},
]
```

- [ ] **Step 6: Create conjectures**

Create `domains/math/conjectures.cue`:

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:    "GoldbachConjecture"
		worth:   400
		isA:     ["Conjecture", "MathConcept", "Anything"]
		english: "Every even number greater than 2 is the sum of two primes"
		status:  "unverified"
	},
]
```

- [ ] **Step 7: Commit**

```bash
git add domains/math/
git commit -m "feat: migrate math domain data to CUE files"
```

---

### Task 5: Math Heuristics CUE File

Move all 12 heuristics from `internal/seed/heuristics.go` into CUE.

**Files:**
- Create: `domains/math/heuristics.cue`

- [ ] **Step 1: Create heuristics file**

Create `domains/math/heuristics.cue` with all 12 heuristics. Each heuristic's DSL programs use CUE raw strings (`#"""..."""#`) to avoid escaping issues with the stack DSL's `"string"` syntax.

```cue
package math

import "domains"

units: [...domains.#Unit]

units: [
	{
		name:    "H-FindExamples"
		worth:   700
		isA:     ["Heuristic", "Anything"]
		english: "Collect instances of a concept from the store"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #""" "CurSlot" @ "examples" = """#
		thenCompute: #"""
			"CurUnit" @ examples
			"found" !
			"found" @ list-length 0 >
			if
				"found" @ "CurUnit" @ "examples" set-slot
			then
		"""#
		thenPrintToUser: #"""
			"Found examples of " "CurUnit" @ concat print
		"""#
	},
	{
		name:    "H-RunOnExamples"
		worth:   750
		isA:     ["Heuristic", "Anything"]
		english: "Run operations on concrete data to generate examples"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			0 "created" !
			"ArgU" @ "domain" get-slot first "domType1" !

			"domType1" @ examples
			each
				it "src1" !
				"src1" @ "data" get-slot nil !=
				"created" @ 5 <
				and
				if
					"ArgU" @ "BinaryOp" isa?
					if
						"domType1" @ examples
						each
							it "src2" !
							"src2" @ "data" get-slot nil !=
							"src1" @ "src2" @ !=
							and
							"created" @ 5 <
							and
							if
								"src1" @ "data" get-slot
								"src2" @ "data" get-slot
								"ArgU" @ apply-op
								"result" !
								"result" @ nil !=
								if
									"ArgU" @ "-on-" concat "src1" @ concat "-" concat "src2" @ concat
									"resultName" !
									"resultName" @ unit-exists? not
									if
										"resultName" @ "Set" create-unit
										"resultUnit" !
										"result" @ "resultUnit" @ "data" set-slot
										"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
										"created" @ 1 + "created" !
										"Applied " "ArgU" @ concat ": " concat "src1" @ concat " x " concat "src2" @ concat print
									then
								then
							then
						end
					then

					"ArgU" @ "UnaryOp" isa?
					"created" @ 5 <
					and
					if
						"src1" @ "data" get-slot
						"ArgU" @ apply-op
						"result" !
						"result" @ nil !=
						if
							"ArgU" @ "-on-" concat "src1" @ concat
							"resultName" !
							"resultName" @ unit-exists? not
							if
								"resultName" @ "Set" create-unit
								"resultUnit" !
								"result" @ "resultUnit" @ "data" set-slot
								"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
								"created" @ 1 + "created" !
								"Applied " "ArgU" @ concat ": " concat "src1" @ concat print
							then
						then
					then
				then
			end
		"""#
	},
	{
		name:    "H-CheckExtremes"
		worth:   600
		isA:     ["Heuristic", "Anything"]
		english: "Examine extreme cases of sets"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "data" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot "theData" !

			"theData" @ set-size 0 =
			if
				"ArgU" @ " is empty" concat print
			then

			"theData" @ set-size 1 =
			if
				"ArgU" @ " is a singleton: {" concat "theData" @ first concat "}" concat print
				"ArgU" @ "worth" get-slot 700 <
				if
					"ArgU" @ "worth" get-slot 100 + "ArgU" @ "worth" set-slot
				then
			then
		"""#
	},
	{
		name:    "H-Specialize"
		worth:   650
		isA:     ["Heuristic", "Anything"]
		english: "Specialize operations by narrowing domain types"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil !=
			and
			"ArgU" @ "defn" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot
			each
				it "domType" !
				"domType" @ "specializations" get-slot
				each
					it "specType" !
					"ArgU" @ "-on-" concat "specType" @ concat
					"specName" !
					"specName" @ unit-exists? not
					if
						"specName" @ "ArgU" @ "isA" get-slot first create-unit
						"specUnit" !
						"ArgU" @ "defn" get-slot "specUnit" @ "defn" set-slot
						"H-Specialize" "specUnit" @ "creditors" set-slot
						"ArgU" @ "range" get-slot "specUnit" @ "range" set-slot
						"specType" @ "specUnit" @ "domain" set-slot
						600 "specUnit" @ "examples" "Specialized op needs testing" add-task
						"Specialized " "ArgU" @ concat " -> " concat "specName" @ concat print
					then
				end
			end
		"""#
	},
	{
		name:    "H-CheckDomain"
		worth:   550
		isA:     ["Heuristic", "Anything"]
		english: "If domain/range overlap, create self-composition"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil !=
			and
			"ArgU" @ "range" get-slot nil !=
			and
			"ArgU" @ "creditors" get-slot nil =
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "range" get-slot
			each
				it "rangeType" !
				"ArgU" @ "domain" get-slot
				each
					it "domType" !
					"domType" @ "rangeType" @ =
					if
						"SelfCompose-" "ArgU" @ pack-name
						"composeName" !
						"composeName" @ unit-exists? not
						if
							"composeName" @ "BinaryOp" create-unit
							"compUnit" !
							"H-CheckDomain" "compUnit" @ "creditors" set-slot
							"ArgU" @ "domain" get-slot "compUnit" @ "domain" set-slot
							"ArgU" @ "range" get-slot "compUnit" @ "range" set-slot
							600 "compUnit" @ "examples" "Self-composition needs examples" add-task
							"Created self-composition: " "composeName" @ concat print
						then
					then
				end
			end
		"""#
	},
	{
		name:    "H-Conjecture"
		worth:   700
		isA:     ["Heuristic", "Anything"]
		english: "Compare sets to find equalities and subset relationships"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "data" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"Set" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "data" get-slot nil !=
				and
				if
					"ArgU" @ "data" get-slot
					"other" @ "data" get-slot
					set-equal?
					if
						"CONJECTURE: " "ArgU" @ concat " = " concat "other" @ concat print
						"ArgU" @ "creditors" get-slot nil !=
						"other" @ "creditors" get-slot nil =
						and
						if
							"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
							"Penalized redundant " "ArgU" @ concat " (= " concat "other" @ concat ")" concat print
						then
						"ArgU" @ "creditors" get-slot nil !=
						"other" @ "creditors" get-slot nil !=
						and
						if
							"ArgU" @ "worth" get-slot "other" @ "worth" get-slot <=
							if
								"ArgU" @ "worth" get-slot 150 - "ArgU" @ "worth" set-slot
							then
						then
					then

					"ArgU" @ "data" get-slot
					"other" @ "data" get-slot
					set-subset?
					"ArgU" @ "data" get-slot "other" @ "data" get-slot set-equal? not
					and
					if
						"CONJECTURE: " "ArgU" @ concat " ⊂ " concat "other" @ concat print
					then
				then
			end
		"""#
	},
	{
		name:    "H-ExploreSlots"
		worth:   500
		isA:     ["Heuristic", "Anything"]
		english: "Add tasks to explore empty important slots"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Heuristic" isa? not
			"ArgU" @ "Slot" isa? not
			and
			"ArgU" @ "explored" get-slot nil =
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "examples" get-slot nil =
			if
				400 "ArgU" @ "examples" "Unit needs examples" add-task
			then
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil =
			and
			if
				350 "ArgU" @ "domain" "Operation needs domain defined" add-task
			then
			true "ArgU" @ "explored" set-slot
		"""#
	},
	{
		name:    "H-KillWorthless"
		worth:   800
		isA:     ["Heuristic", "Anything"]
		english: "Kill units with very low Worth that were machine-created"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "worth" get-slot 100 <
			"ArgU" @ "creditors" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"Killing worthless unit: " "ArgU" @ concat print
			"ArgU" @ kill-unit
		"""#
	},
	{
		name:    "H-BoostInteresting"
		worth:   650
		isA:     ["Heuristic", "Anything"]
		english: "Boost worth of operations that produce surprising results"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "creditors" get-slot nil !=
			"ArgU" @ "data" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot set-size 0 =
			if
				"Interesting: " "ArgU" @ concat " produced empty result" concat print
				"ArgU" @ "creditors" get-slot
				each
					it "cred" !
					"cred" @ "worth" get-slot 50 + "cred" @ "worth" set-slot
				end
			then

			"ArgU" @ "data" get-slot set-size 1 =
			if
				"Interesting: " "ArgU" @ concat " is singleton {" concat "ArgU" @ "data" get-slot first concat "}" concat print
				"ArgU" @ "creditors" get-slot
				each
					it "cred" !
					"cred" @ "worth" get-slot 75 + "cred" @ "worth" set-slot
				end
			then
		"""#
	},
	{
		name:    "H-PenalizeTrivial"
		worth:   600
		isA:     ["Heuristic", "Anything"]
		english: "Penalize machine-created units with trivial (empty) data"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "creditors" get-slot nil !=
			"ArgU" @ "data" get-slot nil !=
			and
		"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot set-size 0 =
			if
				"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
				"Trivial (empty): " "ArgU" @ concat print
			then
		"""#
	},
	{
		name:    "H-AnalyzeApplics"
		worth:   600
		isA:     ["Heuristic", "Anything"]
		english: "Inspect applics for type-skewed success patterns and propose specializations"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Heuristic" isa?
			"ArgU" @ "H-AnalyzeApplics" !=
			and
		"""#
		ifTrulyRelevant: #"""
			"ArgU" @ applics-success-ratio "ratio" !
			"ratio" @ 0.3 >=
			"ratio" @ 0.7 <=
			and
			"ArgU" @ get-applics nil !=
			and
		"""#
		thenCompute: #"""
			"ArgU" @ analyze-and-specialize
		"""#
	},
]
```

- [ ] **Step 2: Commit**

```bash
git add domains/math/heuristics.cue
git commit -m "feat: migrate math heuristics to CUE"
```

---

### Task 6: Observations Domain CUE Files

Move observation domain types and heuristics to CUE.

**Files:**
- Create: `domains/observations/types.cue`
- Create: `domains/observations/heuristics.cue`

- [ ] **Step 1: Create observation types**

Create `domains/observations/types.cue`:

```cue
package observations

import "domains"

units: [...domains.#Unit]

units: [
	{name: "Observation", worth: 500, isA: ["Anything"]},
	{name: "DerivedFact", worth: 400, isA: ["Anything"]},
	{name: "Conjecture", worth: 600, isA: ["Anything"]},
	{name: "ScopeHotspot", worth: 500, isA: ["DerivedFact", "Anything"]},
]
```

- [ ] **Step 2: Create observation heuristics**

Create `domains/observations/heuristics.cue` with all 6 observation heuristics (H-FindScopeHotspots, H-CorroborateObstacles, H-ConjectureFromPatterns, H-BoostCorroborated, H-PenalizeStaleObservations, H-AnalyzeApplics). Use the same raw string pattern as math heuristics. Copy the DSL programs verbatim from `internal/seed/observations.go`.

This file is large but mechanical -- each heuristic is a `#Unit` entry with the DSL programs as raw strings.

- [ ] **Step 3: Commit**

```bash
git add domains/observations/
git commit -m "feat: migrate observations domain to CUE"
```

---

### Task 7: Wire CUE Loader into Registry

Replace the Go seed loaders with CUE-based loading. Add `-domains-dir` flag.

**Files:**
- Modify: `internal/seed/registry.go`
- Modify: `cmd/nous/main.go`

- [ ] **Step 1: Rewrite registry.go**

Replace `internal/seed/registry.go`:

```go
package seed

import (
	"fmt"

	"github.com/chazu/nous/internal/cueload"
	"github.com/chazu/nous/internal/unit"
)

// DomainsDir is set by the CLI flag. Empty means use embedded.
var DomainsDir string

// LoadDomain loads common types then the named domain from CUE files.
func LoadDomain(s *unit.Store, name string) error {
	// Load common types first
	commonDefs, err := loadDomain("common")
	if err != nil {
		return fmt.Errorf("loading common: %w", err)
	}
	populateStore(s, commonDefs)

	// Load the requested domain
	domainDefs, err := loadDomain(name)
	if err != nil {
		return fmt.Errorf("loading domain %s: %w", name, err)
	}
	populateStore(s, domainDefs)

	return nil
}

func loadDomain(name string) ([]cueload.UnitDef, error) {
	if DomainsDir != "" {
		return cueload.LoadDir(DomainsDir + "/" + name)
	}
	return cueload.LoadEmbedded(name)
}

func populateStore(s *unit.Store, defs []cueload.UnitDef) {
	for _, def := range defs {
		u := unit.New(def.Name)
		u.SetWorth(def.Worth)
		if len(def.IsA) > 0 {
			u.Set("isA", def.IsA)
		}
		for k, v := range def.Slots {
			u.Set(k, v)
		}
		s.Put(u)
	}
}

// Available returns the list of known domain names.
func Available() string {
	return "math, observations"
}
```

- [ ] **Step 2: Add -domains-dir flag to main.go**

In `cmd/nous/main.go`, add the flag to `runCmd`:

```go
domainsDir := fs.String("domains-dir", "", "filesystem path to domains/ directory (overrides embedded)")
```

After `fs.Parse(args)`, set it:

```go
if *domainsDir != "" {
    seed.DomainsDir = *domainsDir
}
```

Remove the separate heuristic loading block (lines 62-67 in current main.go) -- heuristics are now loaded as part of the domain CUE files. The domain files include heuristics.

- [ ] **Step 3: Update the usage string**

Add `-domains-dir` to the usage text.

- [ ] **Step 4: Build and test with embedded domains**

```bash
cd /Users/chazu/dev/go/nous
go build -o nous ./cmd/nous
./nous run -domain math -cycles 10 -v 1
```

Expected: loads units from embedded CUE, runs normally.

- [ ] **Step 5: Test with filesystem domains**

```bash
./nous run -domain math -domains-dir ./domains -cycles 10 -v 1
```

Expected: identical output to step 4.

- [ ] **Step 6: Commit**

```bash
git add internal/seed/registry.go cmd/nous/main.go
git commit -m "feat: wire CUE loader into domain registry

LoadDomain now reads from CUE files (embedded or filesystem).
-domains-dir flag enables development without recompilation.
Heuristics load as part of domain CUE files."
```

---

### Task 8: Regression Test

Verify the CUE migration produces identical behavior to the Go seed code.

**Files:**
- Test: run existing tests + behavioral comparison

- [ ] **Step 1: Run all existing tests**

```bash
cd /Users/chazu/dev/go/nous && go test ./...
```

Expected: all pass. If any engine tests fail, the CUE data doesn't match the Go data. Fix the CUE files.

- [ ] **Step 2: Run 100-cycle comparison**

```bash
./nous run -domain math -cycles 100 -v 1 2>&1 | head -5
```

Expected: `nous: loaded 53 units (12 heuristics)` -- same counts as before.

- [ ] **Step 3: Verify conjectures appear**

```bash
./nous run -domain math -cycles 100 -v 1 2>&1 | grep "CONJECTURE"
```

Expected: Same conjectures as the Go-seed runs (SetOfEvens subset SetOfNumbers, SetOfPrimes subset SetOfNumbers, etc.)

- [ ] **Step 4: Commit test results confirmation**

If everything matches, no code changes needed. If fixes were required in previous steps, they should already be committed.

---

### Task 9: Remove Old Go Seed Files

Clean up the Go files that have been replaced by CUE.

**Files:**
- Delete: `internal/seed/math.go`
- Delete: `internal/seed/heuristics.go`
- Delete: `internal/seed/observations.go`

- [ ] **Step 1: Verify no imports remain**

```bash
grep -r "LoadMath\|LoadHeuristics\|LoadObservation" internal/ cmd/ --include="*.go"
```

Expected: only `registry.go` (which no longer calls them). If `engine_test.go` imports `seed.LoadMath` or `seed.LoadHeuristics`, update those tests to use `seed.LoadDomain(store, "math")` instead.

- [ ] **Step 2: Update engine tests**

The test helper `testEngine` in `engine_test.go` calls `seed.LoadMath(store)` and `seed.LoadHeuristics(store)`. Replace with:

```go
func testEngine(t *testing.T) (*Engine, *bytes.Buffer) {
	t.Helper()
	store := unit.NewStore()
	ag := agenda.New()
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	eng := New(store, ag)
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf
	eng.Verbosity = 2
	return eng, buf
}
```

- [ ] **Step 3: Delete old seed files**

```bash
rm internal/seed/math.go internal/seed/heuristics.go internal/seed/observations.go
```

- [ ] **Step 4: Run all tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove Go seed files replaced by CUE domains

math.go, heuristics.go, and observations.go content now lives in
domains/math/ and domains/observations/ CUE files."
```
