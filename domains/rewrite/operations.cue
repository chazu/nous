package domains

units: [
	{
		name: "ValidRewriteString"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn: "rewrite-valid?"
	},
	{
		name: "ValidRewriteRule"
		worth: 500
		isA: ["BinaryPred", "Pred", "Op", "Anything"]
		domain: ["RewriteString", "RewriteString"]
		range: ["Anything"]
		arity: 2
		defn: "rewrite-rule-valid?"
	},
	{
		name: "RewriteOpKappa"
		worth: 600
		isA: ["PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"]
		domain: ["RewriteString"]
		range: ["RewriteString"]
		arity: 1
		rewriteLeft: "ab"
		rewriteRight: "x"
		defn: "\"ab\" \"x\" rewrite-replace-all"
	},
	{
		name: "RewriteOpLambda"
		worth: 600
		isA: ["PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"]
		domain: ["RewriteString"]
		range: ["RewriteString"]
		arity: 1
		rewriteLeft: "xc"
		rewriteRight: "y"
		defn: "\"xc\" \"y\" rewrite-replace-all"
	},
	{
		name: "RewriteOpMu"
		worth: 600
		isA: ["PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"]
		domain: ["RewriteString"]
		range: ["RewriteString"]
		arity: 1
		rewriteLeft: "bc"
		rewriteRight: "z"
		defn: "\"bc\" \"z\" rewrite-replace-all"
	},
	{
		name: "RewriteOpNu"
		worth: 600
		isA: ["PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"]
		domain: ["RewriteString"]
		range: ["RewriteString"]
		arity: 1
		rewriteLeft: "x"
		rewriteRight: "q"
		defn: "\"x\" \"q\" rewrite-replace-all"
	},
]
