// Logical operations as first-class units (Phase 5.8).
// Follows EURISKO's LogicOp family: And, Or, Not, Implies, TheFirstOf, TheSecondOf.
// Domains restricted to TruthValue (tighter than EURISKO's Anything) except the
// polymorphic projections TheFirstOf/TheSecondOf. Generalizations chain mirrors
// EURISKO wiring for discovery heuristics.
units: [
	{
		name:    "LogicOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: logical operations over truth values"
	},
	{
		name:  "True"
		worth: 400
		isA: ["TruthValue", "Anything"]
		data: true
	},
	{
		name:  "False"
		worth: 400
		isA: ["TruthValue", "Anything"]
		data: false
	},
	{
		name:    "Not"
		worth:   500
		isA: ["UnaryOp", "Op", "LogicOp", "UnaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue"]
		range:   ["TruthValue"]
		arity:   1
		english: "Logical negation of a truth value"
		defn: #"""
			not
			"""#
		examples: [
			{args: true, result: false},
			{args: false, result: true},
		]
	},
	{
		name:    "And"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Logical conjunction of two truth values"
		generalizations: ["TheFirstOf", "TheSecondOf", "Or"]
		defn: #"""
			and
			"""#
		examples: [
			{args: true, args2: true, result: true},
			{args: true, args2: false, result: false},
			{args: false, args2: false, result: false},
		]
	},
	{
		name:    "Or"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Logical disjunction of two truth values"
		defn: #"""
			or
			"""#
		examples: [
			{args: true, args2: false, result: true},
			{args: false, args2: false, result: false},
			{args: true, args2: true, result: true},
		]
	},
	{
		name:    "Implies"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Material implication: (not x) or y"
		defn: #"""
			swap not swap or
			"""#
		examples: [
			{args: true, args2: true, result: true},
			{args: true, args2: false, result: false},
			{args: false, args2: false, result: true},
		]
	},
	{
		name:    "TheFirstOf"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["Anything", "Anything"]
		range:   ["Anything"]
		arity:   2
		english: "Polymorphic projection returning the first argument"
		generalizations: ["Or"]
		defn: #"""
			drop
			"""#
	},
	{
		name:    "TheSecondOf"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["Anything", "Anything"]
		range:   ["Anything"]
		arity:   2
		english: "Polymorphic projection returning the second argument"
		generalizations: ["Or"]
		defn: #"""
			swap drop
			"""#
	},
]
