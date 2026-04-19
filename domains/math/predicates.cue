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
		worth:  700
		isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-subset?
			"""#
	},
	{
		name:   "SetEqual"
		worth:  700
		isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set", "Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-equal?
			"""#
	},
	{
		name:   "IsEmpty"
		worth:  700
		isA: ["UnaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-size 0 =
			"""#
	},
	{
		name:   "IsSingleton"
		worth:  700
		isA: ["UnaryPred", "Pred", "MathPred", "Anything"]
		domain: ["Set"]
		range: ["TruthValue"]
		defn:   #"""
			set-size 1 =
			"""#
	},
	{
		name:   "AlwaysT"
		worth:  500
		isA: ["UnaryPred", "Pred", "ConstantPred", "Anything"]
		domain: ["Anything"]
		range: ["TruthValue"]
		defn:   #"""
			drop true
			"""#
	},
	{
		name:   "AlwaysNIL"
		worth:  500
		isA: ["UnaryPred", "Pred", "ConstantPred", "Anything"]
		domain: ["Anything"]
		range: ["TruthValue"]
		defn:   #"""
			drop false
			"""#
	},
]
