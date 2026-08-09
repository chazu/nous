package domains

units: [
	{
		name: "ValidNogoodProblem"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "ng-problem-valid?"
	},
	{
		name: "NogoodProblemSemanticKey"
		worth: 500
		isA: ["UnaryOp", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "ng-semantic-key"
	},
]
