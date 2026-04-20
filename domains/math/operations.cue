units: [
	{
		name:    "SetUnion"
		worth:   600
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Set", "Set"]
		range: ["Set"]
		english: "Combine two sets, keeping all elements"
		defn:    #"""
			set-union
			"""#
		examples: [
			{args: [1, 2, 3], args2: [3, 4, 5], result: [1, 2, 3, 4, 5]},
			{args: [1, 2], args2: [1, 2], result: [1, 2]},
		]
	},
	{
		name:    "SetIntersect"
		worth:   600
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Set", "Set"]
		range: ["Set"]
		english: "Elements common to both sets"
		defn:    #"""
			set-intersect
			"""#
		examples: [
			{args: [1, 2, 3], args2: [2, 3, 4], result: [2, 3]},
			{args: [1, 2], args2: [3, 4], result: []},
		]
	},
	{
		name:    "SetDifference"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Set", "Set"]
		range: ["Set"]
		english: "Elements in first set but not second"
		defn:    #"""
			set-diff
			"""#
		examples: [
			{args: [1, 2, 3, 4], args2: [2, 4], result: [1, 3]},
		]
	},
	{
		name:    "DivisorsOf"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["Number"]
		range: ["Set"]
		english: "All divisors of a number"
		defn:    #"""
			divisors
			"""#
		examples: [
			{args: 12, result: [1, 2, 3, 4, 6, 12]},
			{args: 7, result: [1, 7]},
			{args: 1, result: [1]},
		]
	},
	{
		name:    "GCD"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Number", "Number"]
		range: ["Number"]
		english: "Greatest common divisor of two numbers"
		defn:    #"""
			gcd
			"""#
		examples: [
			{args: 12, args2: 8, result: 4},
			{args: 7, args2: 13, result: 1},
		]
	},
	{
		name:    "Compose"
		worth:   600
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Op", "Op"]
		range: ["Op"]
		english: "Apply one operation after another"
	},
	{
		name:    "Restrict"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["Op", "Pred"]
		range: ["Op"]
		english: "Apply an operation only when a predicate is satisfied"
	},
	{
		name:    "Add"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number", "Number"]
		range:   ["Number"]
		english: "Sum of two numbers"
		defn:    #"""
			+
			"""#
		examples: [
			{args: 2, args2: 3, result: 5},
			{args: 0, args2: 7, result: 7},
		]
	},
	{
		name:    "Multiply"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number", "Number"]
		range:   ["Number"]
		english: "Product of two numbers"
		defn:    #"""
			*
			"""#
		examples: [
			{args: 3, args2: 4, result: 12},
			{args: 1, args2: 9, result: 9},
		]
	},
	{
		name:    "Successor"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number"]
		range:   ["Number"]
		english: "Next integer after n"
		defn:    #"""
			1 +
			"""#
		examples: [
			{args: 0, result: 1},
			{args: 7, result: 8},
		]
	},
	{
		name:    "Square"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain:  ["Number"]
		range:   ["Number"]
		english: "n times n"
		defn:    #"""
			dup *
			"""#
		examples: [
			{args: 3, result: 9},
			{args: 5, result: 25},
		]
	},
	{
		name:    "OSetUnion"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["OSet"]
		english: "Combine two ordered sets, preserving first's order then appending second's novel elements"
		defn:    #"""
			oset-union
			"""#
		examples: [
			{args: [3, 1, 2], args2: [4, 2, 5], result: [3, 1, 2, 4, 5]},
			{args: [19, 17], args2: [2, 17], result: [19, 17, 2]},
		]
	},
	{
		name:    "OSetIntersect"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["OSet"]
		english: "Elements common to both ordered sets, in first's order"
		defn:    #"""
			oset-intersect
			"""#
		examples: [
			{args: [3, 1, 2, 4], args2: [4, 2], result: [2, 4]},
			{args: [19, 17, 13], args2: [13, 19], result: [19, 13]},
		]
	},
	{
		name:    "OSetInsert"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "Anything"]
		range: ["OSet"]
		english: "Append element to ordered set if not already present"
		defn:    #"""
			oset-insert
			"""#
		examples: [
			{args: [3, 1, 2], args2: 7, result: [3, 1, 2, 7]},
			{args: [3, 1, 2], args2: 1, result: [3, 1, 2]},
		]
	},
	{
		name:    "OSetDelete"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "Anything"]
		range: ["OSet"]
		english: "Remove element from ordered set, preserving remaining order"
		defn:    #"""
			oset-delete
			"""#
		examples: [
			{args: [3, 1, 2, 4], args2: 1, result: [3, 2, 4]},
			{args: [19, 17, 13], args2: 99, result: [19, 17, 13]},
		]
	},
	{
		name:    "OSetEqual"
		worth:   500
		isA: ["BinaryPred", "Pred", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["TruthValue"]
		english: "True iff two ordered sets contain the same elements in the same order"
		defn:    #"""
			oset-equal?
			"""#
		examples: [
			{args: [1, 2, 3], args2: [1, 2, 3], result: true},
			{args: [1, 2], args2: [2, 1], result: false},
		]
	},
]
