package domains

units: [
	{name: "RuleInductionVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "ruleinduction"},
	{name: "RuleInductionExperiment", worth: 700, isA: ["Anything"]},
	{name: "RuleInductionPartial", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionRefinement", worth: 450, isA: ["Anything"]},
	{name: "RuleInductionCandidate", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionResult", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionExample", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionObservation", worth: 450, isA: ["Anything"]},
	{name: "RuleInductionEvidence", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionLibrary", worth: 750, isA: ["Anything"]},
	{name: "RuleInductionProvenance", worth: 650, isA: ["Anything"]},
	{name: "RuleInductionProjection", worth: 600, isA: ["Anything"]},
	{name: "RuleInductionConstraint", worth: 450, isA: ["Anything"]},
	{name: "RuleInductionComparison", worth: 400, isA: ["Anything"]},
	{name: "RuleInductionPrune", worth: 400, isA: ["Anything"]},
	{name: "RuleInductionTranscript", worth: 500, isA: ["Anything"]},
	{name: "RuleInductionBoundary", worth: 700, isA: ["Anything"]},
	{name: "RuleInductionCorpus", worth: 700, isA: ["Anything"]},
	{name: "RuleInductionSelection", worth: 700, isA: ["Anything"]},
	{name: "RuleInductionTerminal", worth: 700, isA: ["Anything"]},
	{
		name: "RuleInductionSeed"
		worth: 700
		isA: ["RuleInductionExperiment", "Anything"]
		experimentKey: "rule-induction/seed/v1"
		stage: "stage1"
		reuseMode: "shared-library"
		partialCategory: "RuleInductionPartial"
		refinementCategory: "RuleInductionRefinement"
		candidateCategory: "RuleInductionCandidate"
		resultCategory: "RuleInductionResult"
		observationCategory: "RuleInductionObservation"
		evidenceCategory: "RuleInductionEvidence"
		libraryCategory: "RuleInductionLibrary"
		provenanceCategory: "RuleInductionProvenance"
		projectionCategory: "RuleInductionProjection"
		startTaskSlot: "riStart"
		refineTaskSlot: "riRefine"
		evaluateTaskSlot: "riEvaluate"
		continueTaskSlot: "riContinue"
		facts: ["0:0:1", "0:1:2", "0:1:3", "0:3:4", "1:5:6", "2:6:7"]
		queue: ["03", "01", "02", "04", "05", "12", "13", "14", "15", "23", "24", "25", "34", "35", "45"]
		rootName: "RI.Partial.RuleInductionSeed.stage1.root"
		refinementPriority: 900
		evaluationPriority: 800
		candidateCap: 15
		initialTasks: [{priority: 950, slot: "riStart", reason: "Start bounded rule induction"}]
	},
	{name: "RISeedPositiveOne", worth: 500, isA: ["RuleInductionExample", "Anything"], experiment: "RuleInductionSeed", stage: "stage1", x: 0, y: 4, positive: true},
	{name: "RISeedPositiveTwo", worth: 500, isA: ["RuleInductionExample", "Anything"], experiment: "RuleInductionSeed", stage: "stage1", x: 1, y: 4, positive: true},
	{name: "RISeedNegativeOne", worth: 500, isA: ["RuleInductionExample", "Anything"], experiment: "RuleInductionSeed", stage: "stage1", x: 2, y: 0, positive: false},
	{name: "RISeedNegativeTwo", worth: 500, isA: ["RuleInductionExample", "Anything"], experiment: "RuleInductionSeed", stage: "stage1", x: 4, y: 0, positive: false},
]
