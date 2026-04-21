// Phase 5.7 — Choice operations as first-class units.
// EURISKO: RandomChoose / RandomSubset (roots) with Good* and Best*
// specializations. nous already has `random-choice` / `random-subset` DSL
// builtins (Phase 2). Here we expose them as units, plus Good/Best variants
// that delegate to the random parents (matching EURISKO's convention —
// unless a unit has its own FastAlg, inference falls back to a parent op).
units: [
	{
		name:    "RandomChoose"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Anything"]
		arity:   1
		english: "Return a random element of a set"
		defn: #"""
			random-choice
			"""#
		specializations: ["GoodChoose", "BestChoose"]
	},
	{
		name:    "RandomSubset"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Set"]
		arity:   1
		english: "Return a random subset of a set"
		defn: #"""
			random-subset
			"""#
		specializations: ["GoodSubset", "BestSubset"]
	},
	{
		name:    "GoodChoose"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Anything"]
		arity:   1
		english: "Return a high-worth element of a set (delegates to RandomChoose when no goodness metric is available)"
		generalizations: ["RandomChoose"]
		specializations: ["BestChoose"]
		defn: #"""
			random-choice
			"""#
	},
	{
		name:    "BestChoose"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Anything"]
		arity:   1
		english: "Return the highest-worth element of a set (delegates to RandomChoose absent a goodness metric)"
		generalizations: ["GoodChoose", "RandomChoose"]
		defn: #"""
			random-choice
			"""#
	},
	{
		name:    "GoodSubset"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Set"]
		arity:   1
		english: "Return a subset of high-worth elements (delegates to RandomSubset absent a goodness metric)"
		generalizations: ["RandomSubset"]
		specializations: ["BestSubset"]
		defn: #"""
			random-subset
			"""#
	},
	{
		name:    "BestSubset"
		worth:   500
		isA: ["UnaryOp", "Op", "SetOp", "StrucOp", "MathOp", "MathConcept", "Anything"]
		domain:  ["Set"]
		range:   ["Set"]
		arity:   1
		english: "Return the best-worth subset (delegates to RandomSubset absent a goodness metric)"
		generalizations: ["GoodSubset", "RandomSubset"]
		defn: #"""
			random-subset
			"""#
	},
]
