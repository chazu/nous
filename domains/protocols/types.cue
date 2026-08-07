units: [
	{
		name:  "ProtocolVocabulary"
		worth: 700
		isA: ["Vocabulary", "Anything"]
		dslExtension: "protocol"
		english: "Finite-state protocol vocabulary and selected runtime words"
	},
	{
		name:  "Protocol"
		worth: 700
		isA: ["Anything"]
		english: "A partial deterministic accepted-trace automaton"
	},
	{
		name:  "ProtocolTrainingExample"
		worth: 650
		isA: ["Protocol", "Anything"]
		english: "A protocol visible to the relation-discovery trial"
	},
	{
		name:  "ProtocolTransform"
		worth: 650
		isA: ["UnaryOp", "Op", "Anything"]
	},
	{
		name:  "ProtocolPred"
		worth: 600
		isA: ["Pred", "Op", "Anything"]
	},
	{
		name:  "ProtocolRelation"
		worth: 650
		isA: ["BinaryPred", "ProtocolPred", "Pred", "Op", "Anything"]
	},
	{
		name:  "ProtocolRelationCandidate"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "ProtocolRelationSchema"
		worth: 750
		isA: ["Anything"]
	},
	{
		name:  "ProtocolEvidence"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "ProtocolRelationObservation"
		worth: 400
		isA: ["ProtocolEvidence", "Anything"]
	},
	{
		name:  "StateList"
		worth: 400
		isA: ["Anything"]
	},
	{
		name:  "Trace"
		worth: 400
		isA: ["Anything"]
	},
	{
		name:  "TrainingMachineAlpha"
		worth: 700
		isA: ["ProtocolTrainingExample", "Protocol", "Anything"]
		initialTasks: [{priority: 700, slot: "protocolAnalysis", reason: "Protocol integration control"}]
		data: [
			"state:a0", "state:a1", "state:a2", "state:orphan",
			"event:finish", "event:go", "start:a0", "accept:a2",
			"trans:a0,go>a1", "trans:a1,finish>a2", "trans:orphan,go>orphan",
		]
	},
	{
		name:  "TrainingMachineBeta"
		worth: 700
		isA: ["ProtocolTrainingExample", "Protocol", "Anything"]
		initialTasks: [{priority: 700, slot: "protocolAnalysis", reason: "Protocol integration control"}]
		data: [
			"state:b0", "state:b1", "event:open", "event:reset",
			"start:b0", "accept:b1", "trans:b0,open>b1", "trans:b1,reset>b0",
		]
	},
	{
		name:  "TrainingMachineGamma"
		worth: 700
		isA: ["ProtocolTrainingExample", "Protocol", "Anything"]
		initialTasks: [{priority: 700, slot: "protocolAnalysis", reason: "Protocol integration control"}]
		data: [
			"state:c0", "state:c1", "state:c2", "state:failed", "state:unused",
			"event:advance", "event:fail", "event:finish", "start:c0", "accept:c2",
			"trans:c0,advance>c1", "trans:c1,fail>failed", "trans:c1,finish>c2",
			"trans:failed,fail>failed", "trans:unused,advance>unused",
		]
	},
	{
		name:  "AcceptedAlphaTrace"
		worth: 400
		isA: ["Trace", "Anything"]
		data: ["go", "finish"]
	},
	{
		name:  "RejectedAlphaTrace"
		worth: 400
		isA: ["Trace", "Anything"]
		data: ["finish"]
	},
]
