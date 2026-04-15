units: [
	{
		name:   "MemberOf"
		worth:  500
		isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Number", "Set"]
		range: ["TruthValue"]
		defn:   #"""
			swap set-member?
			"""#
	},
	{
		name:   "SubsetOf"
		worth:  500
		isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-subset?
			"""#
	},
	{
		name:   "SetEqual"
		worth:  500
		isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-equal?
			"""#
	},
]
