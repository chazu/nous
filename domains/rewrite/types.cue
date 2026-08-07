package domains

units: [
	{
		name: "RewriteVocabulary"
		worth: 700
		isA: ["Vocabulary", "Anything"]
		dslExtension: "rewrite"
		english: "Bounded string-rewrite synthesis vocabulary"
	},
	{name: "RewriteString", worth: 650, isA: ["Anything"]},
	{name: "RewriteStringResult", worth: 500, isA: ["RewriteString", "Anything"]},
	{name: "RewriteTrainingExample", worth: 600, isA: ["Anything"]},
	{name: "PrimitiveRewriteOp", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "CompositeRewriteOp", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "RewriteProgramEvidence", worth: 500, isA: ["Anything"]},
	{name: "RewriteObservation", worth: 450, isA: ["Anything"]},
	{name: "RewriteProgramSchema", worth: 750, isA: ["Anything"]},
	{
		name: "RewriteExampleOne"
		worth: 600
		isA: ["RewriteTrainingExample", "Anything"]
		input: "abc"
		expected: "y"
	},
	{
		name: "RewriteExampleTwo"
		worth: 600
		isA: ["RewriteTrainingExample", "Anything"]
		input: "zabc"
		expected: "zy"
	},
	{
		name: "RewriteExampleThree"
		worth: 600
		isA: ["RewriteTrainingExample", "Anything"]
		input: "abcc"
		expected: "yc"
	},
	{
		name: "RewriteExampleFour"
		worth: 600
		isA: ["RewriteTrainingExample", "Anything"]
		input: "abcabc"
		expected: "yy"
	},
]

