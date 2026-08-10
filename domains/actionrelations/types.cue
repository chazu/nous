package domains

units: [
	{
		name: "GuardedActionRelationVocabulary"
		worth: 700
		isA: ["Vocabulary", "Anything"]
		dslExtension: "actionrelations"
		english: "Bounded actions, guarded commutativity relations, and certified sleep-set evidence"
	},
	{name: "ActionRelationTrainingCase", worth: 650, isA: ["Anything"]},
	{name: "ActionRelationObservation", worth: 650, isA: ["Anything"]},
	{name: "ActionGuardCandidate", worth: 550, isA: ["Anything"]},
	{name: "ActionGuardRefinement", worth: 500, isA: ["Anything"]},
	{name: "ActionGuardSearchBarrier", worth: 650, isA: ["Anything"]},
	{name: "GuardedActionRelation", worth: 700, isA: ["Anything"]},
	{name: "GuardedActionArtifact", worth: 800, isA: ["Anything"]},
	{name: "ActionRelationCertificate", worth: 700, isA: ["Anything"]},
	{name: "ActionRelationSearchNode", worth: 600, isA: ["Anything"]},
	{name: "ActionRelationExperiment", worth: 700, isA: ["Anything"]},
]
