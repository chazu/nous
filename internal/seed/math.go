// Package seed provides domain loaders that populate a UnitStore with
// initial concepts and heuristics. Each domain is a function that takes
// a fresh store. Heuristics are loaded separately so they can be shared
// across domains or swapped independently.
package seed

import "github.com/chazu/nous/internal/unit"

// LoadMath loads a set-theory/number-theory domain into the store,
// mirroring EURISKO's initial concept space but with concrete data
// and executable definitions.
func LoadMath(s *unit.Store) {
	// === Meta-structure ===
	put(s, "Anything", 500, nil)
	put(s, "Heuristic", 800, []string{"Anything"})
	put(s, "Slot", 300, []string{"Anything"})

	// === Type hierarchy ===
	put(s, "MathConcept", 500, []string{"Anything"})
	put(s, "MathObj", 500, []string{"MathConcept", "Anything"})
	put(s, "MathOp", 500, []string{"MathConcept", "Anything"})
	put(s, "MathPred", 500, []string{"MathConcept", "Anything"})
	put(s, "Op", 500, []string{"Anything"})
	put(s, "BinaryOp", 500, []string{"Op", "MathOp", "Anything"})
	put(s, "UnaryOp", 500, []string{"Op", "MathOp", "Anything"})
	put(s, "Pred", 500, []string{"Anything"})
	put(s, "BinaryPred", 500, []string{"Pred", "MathPred", "Anything"})
	put(s, "UnaryPred", 500, []string{"Pred", "MathPred", "Anything"})

	// === Structures ===
	structure := put(s, "Structure", 600, []string{"MathObj", "Anything"})
	structure.Set("specializations", []string{"Set", "List", "Bag"})

	set := put(s, "Set", 700, []string{"Structure", "MathObj", "Anything"})
	set.Set("english", "An unordered collection with no duplicate elements")
	set.Set("specializations", []string{"EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens"})

	list := put(s, "List", 600, []string{"Structure", "MathObj", "Anything"})
	list.Set("english", "An ordered collection that may contain duplicates")
	list.Set("specializations", []string{"SortedList"})

	put(s, "Bag", 500, []string{"Structure", "MathObj", "Anything"})

	// === Concrete sets (with actual data) ===
	emptySet := put(s, "EmptySet", 400, []string{"Set", "Structure", "MathObj", "Anything"})
	emptySet.Set("english", "The set with no elements")
	emptySet.Set("data", []int{})

	numbersTo20 := put(s, "SetOfNumbers", 600, []string{"Set", "Structure", "MathObj", "Anything"})
	numbersTo20.Set("english", "The integers from 1 to 20")
	numbersTo20.Set("data", makeRange(1, 21))
	numbersTo20.Set("specializations", []string{"SetOfPrimes", "SetOfEvens", "SetOfOdds"})

	primeSet := put(s, "SetOfPrimes", 600, []string{"Set", "SetOfNumbers", "Structure", "MathObj", "Anything"})
	primeSet.Set("english", "Prime numbers up to 20")
	primeSet.Set("data", []int{2, 3, 5, 7, 11, 13, 17, 19})
	primeSet.Set("defn", `# Filter: keep only primes
		each it prime? if it then end make-set`)
	primeSet.Set("generalizations", []string{"SetOfNumbers"})

	evenSet := put(s, "SetOfEvens", 600, []string{"Set", "SetOfNumbers", "Structure", "MathObj", "Anything"})
	evenSet.Set("english", "Even numbers up to 20")
	evenSet.Set("data", []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20})
	evenSet.Set("generalizations", []string{"SetOfNumbers"})

	oddSet := put(s, "SetOfOdds", 500, []string{"Set", "SetOfNumbers", "Structure", "MathObj", "Anything"})
	oddSet.Set("english", "Odd numbers up to 20")
	oddSet.Set("data", []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19})
	oddSet.Set("generalizations", []string{"SetOfNumbers"})

	put(s, "SortedList", 400, []string{"List", "Structure", "MathObj", "Anything"})

	// === Number types ===
	number := put(s, "Number", 600, []string{"MathObj", "Anything"})
	number.Set("specializations", []string{"EvenNum", "OddNum", "PrimeNum", "PerfectNum", "SquareNum"})

	evenNum := put(s, "EvenNum", 400, []string{"Number", "MathObj", "Anything"})
	evenNum.Set("defn", `even?`)
	evenNum.Set("examples", []int{2, 4, 6, 8, 10})

	oddNum := put(s, "OddNum", 400, []string{"Number", "MathObj", "Anything"})
	oddNum.Set("defn", `odd?`)
	oddNum.Set("examples", []int{1, 3, 5, 7, 9})

	primeNum := put(s, "PrimeNum", 600, []string{"Number", "MathObj", "Anything"})
	primeNum.Set("english", "A number with no divisors other than 1 and itself")
	primeNum.Set("defn", `prime?`)
	primeNum.Set("examples", []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29})

	perfectNum := put(s, "PerfectNum", 500, []string{"Number", "MathObj", "Anything"})
	perfectNum.Set("english", "A number equal to the sum of its proper divisors")
	perfectNum.Set("examples", []int{6, 28, 496})

	squareNum := put(s, "SquareNum", 400, []string{"Number", "MathObj", "Anything"})
	squareNum.Set("examples", []int{1, 4, 9, 16, 25, 36})

	put(s, "TruthValue", 400, []string{"MathObj", "Anything"})

	// === Operations with executable definitions ===
	// defn programs consume their args from the stack

	setUnion := put(s, "SetUnion", 600, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	setUnion.Set("domain", []string{"Set", "Set"})
	setUnion.Set("range", []string{"Set"})
	setUnion.Set("english", "Combine two sets, keeping all elements")
	setUnion.Set("defn", `set-union`)
	setUnion.Set("examples", []map[string]any{
		{"args": []int{1, 2, 3}, "args2": []int{3, 4, 5}, "result": []int{1, 2, 3, 4, 5}},
		{"args": []int{1, 2}, "args2": []int{1, 2}, "result": []int{1, 2}},
	})

	setIntersect := put(s, "SetIntersect", 600, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	setIntersect.Set("domain", []string{"Set", "Set"})
	setIntersect.Set("range", []string{"Set"})
	setIntersect.Set("english", "Elements common to both sets")
	setIntersect.Set("defn", `set-intersect`)
	setIntersect.Set("examples", []map[string]any{
		{"args": []int{1, 2, 3}, "args2": []int{2, 3, 4}, "result": []int{2, 3}},
		{"args": []int{1, 2}, "args2": []int{3, 4}, "result": []int{}},
	})

	setDiff := put(s, "SetDifference", 500, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	setDiff.Set("domain", []string{"Set", "Set"})
	setDiff.Set("range", []string{"Set"})
	setDiff.Set("english", "Elements in first set but not second")
	setDiff.Set("defn", `set-diff`)
	setDiff.Set("examples", []map[string]any{
		{"args": []int{1, 2, 3, 4}, "args2": []int{2, 4}, "result": []int{1, 3}},
	})

	// Unary operations
	divisorsOp := put(s, "DivisorsOf", 500, []string{"UnaryOp", "Op", "MathOp", "Anything"})
	divisorsOp.Set("domain", []string{"Number"})
	divisorsOp.Set("range", []string{"Set"})
	divisorsOp.Set("english", "All divisors of a number")
	divisorsOp.Set("defn", `divisors`)
	divisorsOp.Set("examples", []map[string]any{
		{"args": 12, "result": []int{1, 2, 3, 4, 6, 12}},
		{"args": 7, "result": []int{1, 7}},
		{"args": 1, "result": []int{1}},
	})

	// Predicates with executable definitions
	memberOfPred := put(s, "MemberOf", 500, []string{"BinaryPred", "Pred", "MathPred", "Anything"})
	memberOfPred.Set("domain", []string{"Number", "Set"})
	memberOfPred.Set("range", []string{"TruthValue"})
	memberOfPred.Set("defn", `swap set-member?`)

	subsetOfPred := put(s, "SubsetOf", 500, []string{"BinaryPred", "Pred", "MathPred", "Anything"})
	subsetOfPred.Set("domain", []string{"Set", "Set"})
	subsetOfPred.Set("range", []string{"TruthValue"})
	subsetOfPred.Set("defn", `set-subset?`)

	setEqualPred := put(s, "SetEqual", 500, []string{"BinaryPred", "Pred", "MathPred", "Anything"})
	setEqualPred.Set("domain", []string{"Set", "Set"})
	setEqualPred.Set("range", []string{"TruthValue"})
	setEqualPred.Set("defn", `set-equal?`)

	// GCD as a binary operation
	gcdOp := put(s, "GCD", 500, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	gcdOp.Set("domain", []string{"Number", "Number"})
	gcdOp.Set("range", []string{"Number"})
	gcdOp.Set("english", "Greatest common divisor of two numbers")
	gcdOp.Set("defn", `gcd`)
	gcdOp.Set("examples", []map[string]any{
		{"args": 12, "args2": 8, "result": 4},
		{"args": 7, "args2": 13, "result": 1},
	})

	// === Higher-order concepts ===
	compose := put(s, "Compose", 600, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	compose.Set("domain", []string{"Op", "Op"})
	compose.Set("range", []string{"Op"})
	compose.Set("english", "Apply one operation after another")

	restrict := put(s, "Restrict", 500, []string{"BinaryOp", "Op", "MathOp", "Anything"})
	restrict.Set("domain", []string{"Op", "Pred"})
	restrict.Set("range", []string{"Op"})
	restrict.Set("english", "Apply an operation only when a predicate is satisfied")

	// === Conjectures (to be discovered or verified) ===
	// Pre-seed one known conjecture as an example of the type
	goldbach := put(s, "Conjecture", 500, []string{"MathConcept", "Anything"})
	goldbach.Set("specializations", []string{"GoldbachConjecture"})

	gb := put(s, "GoldbachConjecture", 400, []string{"Conjecture", "MathConcept", "Anything"})
	gb.Set("english", "Every even number greater than 2 is the sum of two primes")
	gb.Set("status", "unverified")

	// === Interesting relationships to discover ===
	// These are NOT pre-seeded — the system should find them:
	// - Primes intersected with Evens = {2} (only even prime)
	// - DivisorsOf applied to primes always gives {1, n}
	// - SetUnion is commutative, SetDifference is not
	// - SetIntersect(A, A) = A (idempotent)
	// - SetUnion(A, EmptySet) = A (identity element)
	// - |DivisorsOf(n)| is odd iff n is a perfect square

	_ = setUnion
	_ = setIntersect
	_ = setDiff
	_ = divisorsOp
	_ = memberOfPred
	_ = subsetOfPred
	_ = setEqualPred
	_ = gcdOp
	_ = compose
	_ = restrict
	_ = goldbach
	_ = gb
}

func put(s *unit.Store, name string, worth int, isA []string) *unit.Unit {
	u := unit.New(name)
	u.SetWorth(worth)
	if isA != nil {
		u.Set("isA", isA)
	}
	s.Put(u)
	return u
}

func makeRange(start, end int) []int {
	r := make([]int, end-start)
	for i := range r {
		r[i] = start + i
	}
	return r
}
