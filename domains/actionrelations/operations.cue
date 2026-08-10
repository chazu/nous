package domains

units: [
	{
		name: "ValidFiniteActionState"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "ar-state-valid?"
	},
	{
		name: "ValidFiniteAction"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "ar-action-valid?"
	},
]
