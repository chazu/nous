// Phase 5.5 (deferred tail) — Concrete List/Bag operations as first-class
// units. EURISKO parity: List family (Insert/Delete/Delete1/Union/Intersect/
// Difference/Equal) plus the analogous Bag family with multiset semantics.
//
// Defns compose the new list/bag DSL builtins in internal/dsl/builtins.go.
// For Insert/Delete ops, domain is [Anything, List|Bag] (elem first, struc
// second); apply-op pushes arg1 then arg2, so stack is [elem, struc] — hence
// the `swap` prefix before the list-primitive which expects (struc elem).
units: [
	// ---------------- List family ----------------
	{
		name:    "ListInsert"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "List"]
		range: ["List"]
		english: "Append elem to the end of list"
		defn:    #"""
			swap list-append
			"""#
		examples: [
			{args: 7, args2: [1, 2, 3], result: [1, 2, 3, 7]},
			{args: 1, args2: [], result: [1]},
		]
	},
	{
		name:    "ListDelete"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "List"]
		range: ["List"]
		english: "Remove every occurrence of elem from list"
		defn:    #"""
			swap list-remove-all
			"""#
		examples: [
			{args: 2, args2: [1, 2, 3, 2], result: [1, 3]},
			{args: 9, args2: [1, 2, 3], result: [1, 2, 3]},
		]
	},
	{
		name:    "ListDelete1"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "List"]
		range: ["List"]
		english: "Remove the first occurrence of elem from list"
		defn:    #"""
			swap list-remove-one
			"""#
		examples: [
			{args: 2, args2: [1, 2, 3, 2], result: [1, 3, 2]},
			{args: 9, args2: [1, 2, 3], result: [1, 2, 3]},
		]
	},
	{
		name:    "ListUnion"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["List", "List"]
		range: ["List"]
		english: "Concatenate list1 and list2 (EURISKO FastAlg = APPEND)"
		defn:    #"""
			list-concat
			"""#
		examples: [
			{args: [1, 2, 3], args2: [3, 4], result: [1, 2, 3, 3, 4]},
			{args: [1, 2], args2: [], result: [1, 2]},
		]
	},
	{
		name:    "ListIntersect"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["List", "List"]
		range: ["List"]
		english: "Elements of list1 that appear in list2, preserving list1's order and duplicates"
		defn:    #"""
			list-intersect-ord
			"""#
		examples: [
			{args: [1, 2, 3, 2], args2: [2, 4], result: [2, 2]},
			{args: [1, 2], args2: [3, 4], result: []},
		]
	},
	{
		name:    "ListDifference"
		worth:   500
		isA: ["BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["List", "List"]
		range: ["List"]
		english: "Elements of list1 not in list2, preserving list1's order and duplicates"
		defn:    #"""
			list-diff-ord
			"""#
		examples: [
			{args: [1, 2, 3, 2], args2: [2], result: [1, 3]},
			{args: [1, 2], args2: [3, 4], result: [1, 2]},
		]
	},
	{
		name:    "ListEqual"
		worth:   500
		isA: ["BinaryPred", "Pred", "BinaryOp", "Op", "ListOp", "MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["List", "List"]
		range: ["TruthValue"]
		english: "True iff the two lists are position-wise equal"
		defn:    #"""
			list-equal?
			"""#
		examples: [
			{args: [1, 2, 3], args2: [1, 2, 3], result: true},
			{args: [1, 2], args2: [2, 1], result: false},
		]
	},

	// ---------------- Bag family ----------------
	{
		name:    "BagInsert"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "Bag"]
		range: ["Bag"]
		english: "Insert elem into bag (append — order is immaterial for bags)"
		defn:    #"""
			swap list-append
			"""#
		examples: [
			{args: 7, args2: [1, 2, 3], result: [1, 2, 3, 7]},
			{args: 1, args2: [1], result: [1, 1]},
		]
	},
	{
		name:    "BagDelete"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "Bag"]
		range: ["Bag"]
		english: "Remove every copy of elem from bag"
		defn:    #"""
			swap list-remove-all
			"""#
		examples: [
			{args: 2, args2: [1, 2, 3, 2], result: [1, 3]},
			{args: 9, args2: [1, 2, 3], result: [1, 2, 3]},
		]
	},
	{
		name:    "BagDelete1"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Anything", "Bag"]
		range: ["Bag"]
		english: "Remove one copy of elem from bag"
		defn:    #"""
			swap list-remove-one
			"""#
		examples: [
			{args: 2, args2: [1, 2, 3, 2], result: [1, 3, 2]},
			{args: 9, args2: [1, 2, 3], result: [1, 2, 3]},
		]
	},
	{
		name:    "BagUnion"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Bag", "Bag"]
		range: ["Bag"]
		english: "Multiset union — counts add (equivalent to list concatenation)"
		defn:    #"""
			list-concat
			"""#
		examples: [
			{args: [1, 2, 2], args2: [2, 3], result: [1, 2, 2, 2, 3]},
			{args: [], args2: [1], result: [1]},
		]
	},
	{
		name:    "BagIntersect"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Bag", "Bag"]
		range: ["Bag"]
		english: "Multiset intersect — count of each elem is min(count in B1, count in B2)"
		defn:    #"""
			bag-intersect
			"""#
		examples: [
			{args: [1, 2, 2, 3], args2: [2, 2, 4], result: [2, 2]},
			{args: [1, 2], args2: [3, 4], result: []},
		]
	},
	{
		name:    "BagDifference"
		worth:   500
		isA: ["BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Bag", "Bag"]
		range: ["Bag"]
		english: "Multiset difference — count is max(0, count in B1 - count in B2)"
		defn:    #"""
			bag-diff
			"""#
		examples: [
			{args: [1, 2, 2, 3], args2: [2, 3], result: [1, 2]},
			{args: [1, 2], args2: [3, 4], result: [1, 2]},
		]
	},
	{
		name:    "BagEqual"
		worth:   500
		isA: ["BinaryPred", "Pred", "BinaryOp", "Op", "BagOp", "MultEleStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain: ["Bag", "Bag"]
		range: ["TruthValue"]
		english: "True iff the two bags contain the same elements with the same multiplicities"
		defn:    #"""
			bag-equal?
			"""#
		examples: [
			{args: [1, 2, 2], args2: [2, 1, 2], result: true},
			{args: [1, 2], args2: [1, 1, 2], result: false},
		]
	},
]
