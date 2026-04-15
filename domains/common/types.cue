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
		name:  "Slot"
		worth: 300
		isA: ["Anything"]
	},
	{
		name:  "MathConcept"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "MathObj"
		worth: 500
		isA: ["MathConcept", "Anything"]
	},
	{
		name:  "MathOp"
		worth: 500
		isA: ["MathConcept", "Anything"]
	},
	{
		name:  "MathPred"
		worth: 500
		isA: ["MathConcept", "Anything"]
	},
	{
		name:  "Op"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "BinaryOp"
		worth: 500
		isA: ["Op", "MathOp", "Anything"]
	},
	{
		name:  "UnaryOp"
		worth: 500
		isA: ["Op", "MathOp", "Anything"]
	},
	{
		name:  "Pred"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "BinaryPred"
		worth: 500
		isA: ["Pred", "MathPred", "Anything"]
	},
	{
		name:  "UnaryPred"
		worth: 500
		isA: ["Pred", "MathPred", "Anything"]
	},
]
