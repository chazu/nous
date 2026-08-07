units: [
	{
		name:  "MergeBuildGraphs"
		worth: 700
		isA: ["BinaryOp", "GraphOp", "Op", "BuildConcept", "Anything"]
		domain: ["BuildGraph", "BuildGraph"]
		range: ["BuildGraph"]
		arity: 2
		english: "Union the edges of two build graphs"
		defn: #"""
			collection-union
			"""#
		examples: [
			{args: ["web>api"], args2: ["api>core"], result: ["web>api", "api>core"]},
		]
	},
	{
		name:  "CommonBuildEdges"
		worth: 600
		isA: ["BinaryOp", "GraphOp", "Op", "BuildConcept", "Anything"]
		domain: ["BuildGraph", "BuildGraph"]
		range: ["BuildGraph"]
		arity: 2
		english: "Find edges common to two build graphs"
		defn: #"""
			collection-intersect
			"""#
	},
	{
		name:  "SubtractBuildEdges"
		worth: 600
		isA: ["BinaryOp", "GraphOp", "Op", "BuildConcept", "Anything"]
		domain: ["BuildGraph", "BuildGraph"]
		range: ["BuildGraph"]
		arity: 2
		english: "Remove every edge in the second graph from the first"
		defn: #"""
			collection-diff
			"""#
	},
	{
		name:  "SameBuildGraph"
		worth: 500
		isA: ["BinaryPred", "GraphPred", "Pred", "Op", "BuildConcept", "Anything"]
		domain: ["BuildGraph", "BuildGraph"]
		range: ["Anything"]
		arity: 2
		english: "True when two build graphs have the same edges"
		defn: #"""
			collection-equal?
			"""#
	},
]
