units: [
	{
		name:  "Anything"
		worth: 500
		isA: []
	},
	{
		name:  "Heuristic"
		worth: 800
		isA: ["Anything"]
	},
	{
		name:  "Vocabulary"
		worth: 500
		isA: ["Anything"]
		english: "A domain vocabulary that may select a scoped DSL extension"
	},
	{
		name:  "ContextualCredit"
		worth: 0
		isA: ["Anything"]
		english: "Compact context, subject, and role-specific credit record"
	},
	{
		name:  "Slot"
		worth: 300
		isA: ["Anything"]
	},
	{
		name:  "Op"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "BinaryOp"
		worth: 500
		isA: ["Op", "Anything"]
	},
	{
		name:  "UnaryOp"
		worth: 500
		isA: ["Op", "Anything"]
	},
	{
		name:  "Pred"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "BinaryPred"
		worth: 500
		isA: ["Pred", "Anything"]
	},
	{
		name:  "UnaryPred"
		worth: 500
		isA: ["Pred", "Anything"]
	},
	{
		name:  "ProtoConjec"
		worth: 400
		isA: ["Anything"]
		english: "Structured conjecture created by a heuristic; carries kind, about, evidence, status"
	},
]
