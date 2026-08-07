package domains

units: [
	{name: "IteratedGameVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "game"},
	{name: "IteratedGameExperiment", worth: 700, isA: ["Anything"]},
	{name: "GameStrategy", worth: 650, isA: ["Anything"]},
	{name: "GameStrategyCandidate", worth: 650, isA: ["GameStrategy", "Anything"]},
	{name: "GameOpponent", worth: 650, isA: ["GameStrategy", "Anything"]},
	{name: "GameEvaluationCase", worth: 650, isA: ["Anything"]},
	{name: "GameMatchResult", worth: 500, isA: ["Anything"]},
	{name: "GameMatchObservation", worth: 450, isA: ["Anything"]},
	{name: "GameStrategyEvidence", worth: 500, isA: ["Anything"]},
	{name: "GameSelectionEvidence", worth: 500, isA: ["Anything"]},
	{name: "GameStrategySchema", worth: 750, isA: ["Anything"]},
	{name: "GameStrategyConjecture", worth: 400, isA: ["Anything"]},

	{name: "GameOpponentAllC", worth: 600, isA: ["GameOpponent", "GameStrategy", "Anything"], data: ["initial:C", "after-CC:C", "after-CD:C", "after-DC:C", "after-DD:C"]},
	{name: "GameOpponentAllD", worth: 600, isA: ["GameOpponent", "GameStrategy", "Anything"], data: ["initial:D", "after-CC:D", "after-CD:D", "after-DC:D", "after-DD:D"]},
	{name: "GameOpponentTFT", worth: 600, isA: ["GameOpponent", "GameStrategy", "Anything"], data: ["initial:C", "after-CC:C", "after-CD:D", "after-DC:C", "after-DD:D"]},
	{name: "GameOpponentAlternator", worth: 600, isA: ["GameOpponent", "GameStrategy", "Anything"], data: ["initial:C", "after-CC:D", "after-CD:D", "after-DC:C", "after-DD:C"]},

	{name: "GameCaseAllC", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "training", opponent: "GameOpponentAllC", candidateFlips: [], opponentFlips: []},
	{name: "GameCaseAllD", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "training", opponent: "GameOpponentAllD", candidateFlips: [], opponentFlips: []},
	{name: "GameCaseTFT", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "training", opponent: "GameOpponentTFT", candidateFlips: [], opponentFlips: []},
	{name: "GameCaseAlternator", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "training", opponent: "GameOpponentAlternator", candidateFlips: [], opponentFlips: []},
	{name: "GameCaseSelf", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "self", self: true, candidateFlips: [], opponentFlips: []},
	{name: "GameCasePerturbedTFT", worth: 600, isA: ["GameEvaluationCase", "Anything"], axis: "perturbation", opponent: "GameOpponentTFT", candidateFlips: [10], opponentFlips: [20]},

	{
		name: "MemoryOnePDProfileA"
		worth: 700
		isA: ["IteratedGameExperiment", "Anything"]
		experimentKey: "game/memory-one-pd/profile-a/v1"
		comparisonMethod: "exhaustive-memory-one-profile/v1"
		creditContext: "game/memory-one-pd/profile-a/v1"
		profileKey: "sha256:v1:a546aa6e4374be3f4bd13e96e6eab698361db85471d79da435dd04e70123f0b4"
		temptation: 5
		reward: 3
		punishment: 1
		sucker: 0
		rounds: 60
		candidateCategory: "GameStrategyCandidate"
		opponentCategory: "GameOpponent"
		caseCategory: "GameEvaluationCase"
		resultCategory: "GameMatchResult"
		observationCategory: "GameMatchObservation"
		evidenceCategory: "GameStrategyEvidence"
		selectionCategory: "GameSelectionEvidence"
		schemaCategory: "GameStrategySchema"
		conjectureCategory: "GameStrategyConjecture"
		opponentStrategySlot: "data"
		evaluationCases: ["GameCaseAllC", "GameCaseAllD", "GameCaseTFT", "GameCaseAlternator", "GameCaseSelf", "GameCasePerturbedTFT"]
		candidateCap: 32
		caseCap: 16
		generationTaskSlot: "gameStrategyGeneration"
		evaluationTaskSlot: "gameStrategyEvaluation"
		finalizationTaskSlot: "gameStrategyFinalization"
		generationPriority: 800
		evaluationPriority: 700
		finalizationPriority: 600
		initialTasks: [{priority: 800, slot: "gameStrategyGeneration", reason: "Enumerate deterministic memory-one strategies"}]
	},
]
