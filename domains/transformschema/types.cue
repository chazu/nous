package domains

units: [
	{
		name: "TransformationSchemaVocabulary"
		worth: 700
		isA: ["Vocabulary", "Anything"]
		dslExtension: "transformschema"
		english: "Bounded typed forests, concrete edits, and role-parameterized transformation schemas"
	},
	{name: "TransformTrainingCase", worth: 650, isA: ["Anything"]},
	{name: "TransformConcreteProgram", worth: 650, isA: ["Anything"]},
	{name: "TransformPartialCandidate", worth: 550, isA: ["Anything"]},
	{name: "TransformCounterexample", worth: 550, isA: ["Anything"]},
	{name: "TransformEvidenceBarrier", worth: 650, isA: ["Anything"]},
	{name: "TransformSchemaArtifact", worth: 800, isA: ["Anything"]},
]
