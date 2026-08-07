package domains

units: [
	{
		name: "ValidGameStrategy"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["GameStrategy"]
		range: ["Anything"]
		arity: 1
		defn: #"""
			game-strategy-valid?
			"""#
	},
	{
		name: "ValidIteratedGameExperiment"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["IteratedGameExperiment"]
		range: ["Anything"]
		arity: 1
		defn: #"""
			game-experiment-valid?
			"""#
	},
	{
		name: "CompleteIteratedGameExperiment"
		worth: 600
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["IteratedGameExperiment"]
		range: ["Anything"]
		arity: 1
		defn: #"""
			game-experiment-complete?
			"""#
	},
]
