units: [
	{
		name: "ValidProtocol"
		worth: 500
		isA: ["UnaryPred", "ProtocolPred", "Pred", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["Anything"]
		arity: 1
		english: "True when the value is a valid protocol encoding"
		defn: #"""
			protocol-valid?
			"""#
	},
	{
		name: "ReachableProtocolStates"
		worth: 500
		isA: ["UnaryOp", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["StateList"]
		arity: 1
		defn: #"""
			protocol-reachable-states
			"""#
	},
	{
		name: "RejectingTrapProtocolStates"
		worth: 600
		isA: ["UnaryOp", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["StateList"]
		arity: 1
		defn: #"""
			protocol-rejecting-trap-states
			"""#
	},
	{
		name: "CanonicalizeProtocol"
		worth: 550
		isA: ["ProtocolTransform", "UnaryOp", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["Protocol"]
		arity: 1
		defn: #"""
			protocol-canonicalize
			"""#
	},
	{
		name: "RemoveUnreachableProtocolStates"
		worth: 650
		isA: ["ProtocolTransform", "UnaryOp", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["Protocol"]
		arity: 1
		defn: #"""
			protocol-remove-unreachable
			"""#
	},
	{
		name: "DropFirstProtocolTransition"
		worth: 500
		isA: ["ProtocolTransform", "UnaryOp", "Op", "Anything"]
		domain: ["Protocol"]
		range: ["Protocol"]
		arity: 1
		defn: #"""
			protocol-drop-first-transition
			"""#
	},
	{
		name: "ProtocolAcceptsTrace"
		worth: 550
		isA: ["BinaryPred", "ProtocolPred", "Pred", "Op", "Anything"]
		domain: ["Protocol", "Trace"]
		range: ["Anything"]
		arity: 2
		defn: #"""
			protocol-accepts?
			"""#
	},
	{
		name: "EquivalentProtocols"
		worth: 650
		isA: ["ProtocolRelation", "BinaryPred", "ProtocolPred", "Pred", "Op", "Anything"]
		domain: ["Protocol", "Protocol"]
		range: ["Anything"]
		arity: 2
		defn: #"""
			protocol-equivalent?
			"""#
	},
	{
		name: "SameProtocolEncoding"
		worth: 500
		isA: ["ProtocolRelation", "BinaryPred", "ProtocolPred", "Pred", "Op", "Anything"]
		domain: ["Protocol", "Protocol"]
		range: ["Anything"]
		arity: 2
		defn: #"""
			protocol-same-encoding?
			"""#
	},
]
