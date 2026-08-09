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
	{name: "TransformLearningExperiment", worth: 700, isA: ["Anything"]},
	{name: "TransformConcreteProgram", worth: 650, isA: ["Anything"]},
	{name: "TransformPartialCandidate", worth: 550, isA: ["Anything"]},
	{name: "TransformRefinementEdge", worth: 550, isA: ["Anything"]},
	{name: "TransformRootPartial", worth: 550, isA: ["TransformPartialCandidate", "Anything"], stage: "root", value: "", status: "root", partial: "[\"transform-partial/v1\",0,\"\",\"\",\"\",\"\",\"\"]"},
	{name: "TransformCounterexample", worth: 550, isA: ["Anything"]},
	{name: "TransformFactorEvidence", worth: 600, isA: ["Anything"]},
	{name: "TransformEvidenceBarrier", worth: 650, isA: ["Anything"]},
	{name: "TransformSchemaArtifact", worth: 800, isA: ["Anything"]},
]
