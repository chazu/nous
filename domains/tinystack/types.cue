package domains

units: [
	{name: "ProgramSynthesisVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "programsynth"},
	{name: "StackVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "stack"},
	{name: "BoundedProgramSynthesisExperiment", worth: 700, isA: ["Anything"]},
	{name: "SynthesizedProgram", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "IntegerStack", worth: 650, isA: ["Anything"]},
	{name: "StackInstruction", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "StackProgramCandidate", worth: 650, isA: ["SynthesizedProgram", "UnaryOp", "Op", "Anything"]},
	{name: "StackProgramExample", worth: 650, isA: ["Anything"]},
	{name: "StackProgramResult", worth: 500, isA: ["IntegerStack", "Anything"]},
	{name: "StackProgramObservation", worth: 450, isA: ["Anything"]},
	{name: "StackProgramEvidence", worth: 500, isA: ["Anything"]},
	{name: "StackSelectionEvidence", worth: 500, isA: ["Anything"]},
	{name: "StackProgramSchema", worth: 750, isA: ["Anything"]},
	{name: "StackSimplificationProbe", worth: 450, isA: ["IntegerStack", "Anything"]},
	{name: "StackSimplificationPair", worth: 450, isA: ["Anything"]},
	{name: "StackSimplificationExecutionObservation", worth: 400, isA: ["Anything"]},
	{name: "StackSimplificationExecutionResult", worth: 450, isA: ["IntegerStack", "Anything"]},
	{name: "StackSimplificationComparisonObservation", worth: 400, isA: ["Anything"]},
	{name: "StackSimplificationEvidence", worth: 500, isA: ["Anything"]},
	{name: "StackSimplificationSchema", worth: 750, isA: ["Anything"]},
	{
		name:  "StackSynthesisExperiment"
		worth: 700
		isA: ["BoundedProgramSynthesisExperiment", "Anything"]
		experimentKey:                               "stack/ordered-programs/v1"
		primitiveCategory:                           "StackInstruction"
		candidateCategory:                           "StackProgramCandidate"
		exampleCategory:                             "StackProgramExample"
		valueCategory:                               "IntegerStack"
		resultCategory:                              "StackProgramResult"
		observationCategory:                         "StackProgramObservation"
		evidenceCategory:                            "StackProgramEvidence"
		selectionEvidenceCategory:                   "StackSelectionEvidence"
		promotedSchemaCategory:                      "StackProgramSchema"
		inputSlot:                                   "input"
		expectedSlot:                                "expected"
		resultValueSlot:                             "data"
		inputValidator:                              "ValidIntegerStackInput"
		outputValidator:                             "ValidIntegerStack"
		comparator:                                  "EqualIntegerStacks"
		primitiveSemanticSlot:                       "semanticOpcode"
		maxLength:                                   3
		minCorpus:                                   4
		primitiveCap:                                8
		exampleCap:                                  16
		probeCap:                                    16
		candidateCap:                                600
		simplificationComparisonCap:                 4096
		synthesisMethod:                             "ordered-sequences-up-to-3/v1"
		creditContext:                               "stack/corpus-a/ordered-sequences-up-to-3/v1"
		synthesisTaskSlot:                           "boundedProgramSynthesis"
		evaluationTaskSlot:                          "boundedProgramEvaluation"
		finalizationTaskSlot:                        "boundedProgramFinalization"
		simplificationTaskSlot:                      "boundedProgramSimplification"
		synthesisPriority:                           800
		evaluationPriority:                          700
		finalizationPriority:                        600
		simplificationPriority:                      550
		probeCategory:                               "StackSimplificationProbe"
		probeInputSlot:                              "data"
		simplificationProgramLength:                 2
		simplificationPairCategory:                  "StackSimplificationPair"
		simplificationExecutionObservationCategory:  "StackSimplificationExecutionObservation"
		simplificationExecutionResultCategory:       "StackSimplificationExecutionResult"
		simplificationComparisonObservationCategory: "StackSimplificationComparisonObservation"
		simplificationEvidenceCategory:              "StackSimplificationEvidence"
		simplificationSchemaCategory:                "StackSimplificationSchema"
		probeSetVersion:                             "stack-probes-a/v1"
		initialTasks: [{priority: 800, slot: "boundedProgramSynthesis", reason: "Enumerate bounded stack programs"}]
	},
	{name: "StackExampleA", worth: 650, isA: ["StackProgramExample", "Anything"], input: [2, 3], expected: [2, 5]},
	{name: "StackExampleB", worth: 650, isA: ["StackProgramExample", "Anything"], input: [-1, 4], expected: [-1, 3]},
	{name: "StackExampleC", worth: 650, isA: ["StackProgramExample", "Anything"], input: [0, -2], expected: [0, -2]},
	{name: "StackExampleD", worth: 650, isA: ["StackProgramExample", "Anything"], input: [5, 0], expected: [5, 5]},
	{name: "StackProbeEmpty", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: []},
	{name: "StackProbeSingle", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: [2]},
	{name: "StackProbePair", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: [2, 3]},
	{name: "StackProbeTripleA", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: [-1, 4, 5]},
	{name: "StackProbePairZero", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: [0, -2]},
	{name: "StackProbeTripleB", worth: 450, isA: ["StackSimplificationProbe", "IntegerStack", "Anything"], data: [3, 0, 1]},
]
