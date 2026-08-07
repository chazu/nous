package domains

units: [
	{
		name:  "ValidIntegerStackInput"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["Anything"]
		arity: 1
		defn:  "stack-input-valid?"
	},
	{
		name:  "ValidIntegerStack"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["Anything"]
		arity: 1
		defn:  "stack-valid?"
	},
	{
		name:  "EqualIntegerStacks"
		worth: 500
		isA: ["BinaryPred", "Pred", "Op", "Anything"]
		domain: ["IntegerStack", "IntegerStack"]
		range: ["Anything"]
		arity: 2
		defn:  "stack-equal?"
	},
	{
		name:  "StackOpKappa"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "dup"
		defn:           "\"dup\" stack-exec-op"
	},
	{
		name:  "StackOpLambda"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "swap"
		defn:           "\"swap\" stack-exec-op"
	},
	{
		name:  "StackOpMu"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "drop"
		defn:           "\"drop\" stack-exec-op"
	},
	{
		name:  "StackOpNu"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "over"
		defn:           "\"over\" stack-exec-op"
	},
	{
		name:  "StackOpXi"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "add"
		defn:           "\"add\" stack-exec-op"
	},
	{
		name:  "StackOpOmicron"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "mul"
		defn:           "\"mul\" stack-exec-op"
	},
	{
		name:  "StackOpPi"
		worth: 500
		isA: ["StackInstruction", "UnaryOp", "Op", "Anything"]
		domain: ["IntegerStack"]
		range: ["IntegerStack"]
		arity:          1
		semanticOpcode: "double"
		defn:           "\"double\" stack-exec-op"
	},
]
