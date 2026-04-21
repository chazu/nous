units: [
	{
		name:    "OPair"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "An ordered pair of two elements"
	},
	{
		// Phase 5.2 tail — unordered 2-element structure.
		// EURISKO: isA (MathConcept MathObj Anything Category TypeOfStructure),
		// FastDefn = (EQ 2 (LENGTH s)), generalizations include MultEleStruc,
		// UnOrdStruc, Bag. nous models this as a Structure whose defn accepts
		// any 2-length list without regard to element order.
		name:    "Pair"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "An unordered pair of two elements"
		generalizations: ["Structure", "Bag", "Anything"]
		defn: #"""
			dup is-list? swap list-length 2 = and
			"""#
	},
	{
		// Phase 5.2 tail — UnaryOp on OPair that returns the pair with its
		// two elements swapped. EURISKO: FastAlg = (LIST (CADR p) (CAR p)).
		// In nous, data is [a,b]; defn reverses the list.
		name:    "ReverseOPair"
		worth:   500
		isA: ["UnaryOp", "Op", "ListOp", "OrdStrucOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["OPair"]
		range:   ["OPair"]
		arity:   1
		english: "Return the OPair with its two elements swapped"
		defn: #"""
			reverse
			"""#
		examples: [
			{args: [3, 7], result: [7, 3]},
			{args: [19, 2], result: [2, 19]},
		]
	},
]
