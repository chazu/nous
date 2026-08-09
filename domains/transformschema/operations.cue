package domains

units: [
	{
		name: "ValidTransformForest"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "ts-forest-valid?"
	},
	{
		name: "ApplyTransformSchema"
		worth: 600
		isA: ["BinaryOp", "Op", "Anything"]
		domain: ["Anything", "Anything"]
		range: ["Anything"]
		arity: 2
		defn: "ts-schema-apply"
	},
	{
		name: "ApplyConcreteTransformProgram"
		worth: 600
		isA: ["BinaryOp", "Op", "Anything"]
		domain: ["Anything", "Anything"]
		range: ["Anything"]
		arity: 2
		defn: "ts-program-apply"
	},
]
