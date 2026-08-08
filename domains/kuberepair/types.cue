package domains

units: [
	{name: "ProgramSynthesisVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "programsynth"},
	{name: "KubeRepairVocabulary", worth: 700, isA: ["Vocabulary", "Anything"], dslExtension: "kuberepair"},
	{name: "BoundedProgramSynthesisExperiment", worth: 700, isA: ["Anything"]},
	{name: "SynthesizedProgram", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "KubeRepairValue", worth: 650, isA: ["Anything"]},
	{name: "KubeAtomicEdit", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "KubeRepairCandidate", worth: 650, isA: ["SynthesizedProgram", "UnaryOp", "Op", "Anything"]},
	{name: "KubeRepairExample", worth: 650, isA: ["Anything"]},
	{name: "KubeRepairResult", worth: 500, isA: ["KubeRepairValue", "Anything"]},
	{name: "KubeRepairObservation", worth: 450, isA: ["Anything"]},
	{name: "KubeRepairEvidence", worth: 500, isA: ["Anything"]},
	{name: "KubeRepairSelectionEvidence", worth: 500, isA: ["Anything"]},
	{name: "KubeRepairSchema", worth: 750, isA: ["Anything"]},
	{name: "KubeRepairProbe", worth: 450, isA: ["KubeRepairValue", "Anything"]},
	{name: "KubeRepairSimplificationPair", worth: 450, isA: ["Anything"]},
	{name: "KubeRepairSimplificationExecutionObservation", worth: 400, isA: ["Anything"]},
	{name: "KubeRepairSimplificationExecutionResult", worth: 450, isA: ["KubeRepairValue", "Anything"]},
	{name: "KubeRepairSimplificationComparisonObservation", worth: 400, isA: ["Anything"]},
	{name: "KubeRepairSimplificationEvidence", worth: 500, isA: ["Anything"]},
	{name: "KubeRepairSimplificationSchema", worth: 750, isA: ["Anything"]},
	{name: "KubeRepairFeature", worth: 650, isA: ["Anything"]},
	{name: "KubeRepairRelation", worth: 650, isA: ["Anything"]},
	{
		name: "KubeRepairExperiment"
		worth: 700
		isA: ["BoundedProgramSynthesisExperiment", "Anything"]
		experimentKey: "kubernetes-selector-reference/seed/v1"
		primitiveCategory: "KubeAtomicEdit"
		candidateCategory: "KubeRepairCandidate"
		exampleCategory: "KubeRepairExample"
		valueCategory: "KubeRepairValue"
		resultCategory: "KubeRepairResult"
		observationCategory: "KubeRepairObservation"
		evidenceCategory: "KubeRepairEvidence"
		selectionEvidenceCategory: "KubeRepairSelectionEvidence"
		promotedSchemaCategory: "KubeRepairSchema"
		inputSlot: "input"
		expectedSlot: "expected"
		resultValueSlot: "data"
		inputValidator: "ValidKubeBundle"
		outputValidator: "ValidKubeRepairValue"
		comparator: "KubeValueMatches"
		primitiveSemanticSlot: "semanticOpcode"
		maxLength: 3
		minCorpus: 1
		primitiveCap: 8
		exampleCap: 1
		probeCap: 1
		candidateCap: 600
		simplificationComparisonCap: 4096
		synthesisMethod: "ordered-bound-atomic-edits-up-to-3/v1"
		creditContext: "kubernetes-selector-reference/atomic-edits/v1"
		synthesisTaskSlot: "boundedKubeRepairSynthesis"
		evaluationTaskSlot: "boundedKubeRepairEvaluation"
		finalizationTaskSlot: "boundedKubeRepairFinalization"
		simplificationTaskSlot: "boundedKubeRepairSimplification"
		synthesisPriority: 800
		evaluationPriority: 700
		finalizationPriority: 600
		simplificationPriority: 550
		probeCategory: "KubeRepairProbe"
		probeInputSlot: "data"
		simplificationProgramLength: 2
		simplificationPairCategory: "KubeRepairSimplificationPair"
		simplificationExecutionObservationCategory: "KubeRepairSimplificationExecutionObservation"
		simplificationExecutionResultCategory: "KubeRepairSimplificationExecutionResult"
		simplificationComparisonObservationCategory: "KubeRepairSimplificationComparisonObservation"
		simplificationEvidenceCategory: "KubeRepairSimplificationEvidence"
		simplificationSchemaCategory: "KubeRepairSimplificationSchema"
		probeSetVersion: "kubernetes-selector-reference/seed-probe/v1"
		initialTasks: [{priority: 800, slot: "boundedKubeRepairSynthesis", reason: "Enumerate bound atomic Kubernetes repair plans"}]
	},
	{
		name: "KubeRepairSeedExample"
		worth: 650
		isA: ["KubeRepairExample", "Anything"]
		input: #"{"version":"kubernetes-bundle/v1","bundle":{"namespace":"delta","deployment":{"name":"orbit","selector":[{"key":"app","value":"api"}],"template":{"labels":[{"key":"app","value":"broken"}],"containers":[{"name":"server","ports":[{"name":"health","number":9090},{"name":"web","number":8080}],"readiness":{"path":"/ready","port":{"kind":"name","name":"web"}}}]}},"service":{"name":"gateway","selector":[{"key":"app","value":"api"}],"servicePort":{"name":"https","port":443,"targetPort":{"kind":"name","name":"stale"}}},"pods":[{"name":"other","labels":[{"key":"app","value":"decoy"}],"containers":[{"name":"other","ports":[{"name":"web","number":7070}]}]}],"protected":["{\"kind\":\"declared-port\",\"resource\":\"orbit\",\"container\":\"server\",\"port\":\"health\"}","{\"kind\":\"declared-port\",\"resource\":\"orbit\",\"container\":\"server\",\"port\":\"web\"}","{\"kind\":\"deployment-label\",\"resource\":\"orbit\",\"key\":\"app\"}","{\"kind\":\"readiness-port\",\"resource\":\"orbit\",\"container\":\"server\"}","{\"kind\":\"service-label\",\"resource\":\"gateway\",\"key\":\"app\"}"]}}"#
		expected: #"{"version":"kubernetes-intent-handle/v1","handle":"912ef51d8fd1e5a23373f4e103758514ff1c6e666050d13c7322957025f96fb7"}"#
	},
	{
		name: "KubeRepairSeedProbe"
		worth: 450
		isA: ["KubeRepairProbe", "KubeRepairValue", "Anything"]
		data: #"{"version":"kubernetes-bundle/v1","bundle":{"namespace":"delta","deployment":{"name":"orbit","selector":[{"key":"app","value":"api"}],"template":{"labels":[{"key":"app","value":"broken"}],"containers":[{"name":"server","ports":[{"name":"health","number":9090},{"name":"web","number":8080}],"readiness":{"path":"/ready","port":{"kind":"name","name":"web"}}}]}},"service":{"name":"gateway","selector":[{"key":"app","value":"api"}],"servicePort":{"name":"https","port":443,"targetPort":{"kind":"name","name":"stale"}}},"pods":[{"name":"other","labels":[{"key":"app","value":"decoy"}],"containers":[{"name":"other","ports":[{"name":"web","number":7070}]}]}],"protected":["{\"kind\":\"declared-port\",\"resource\":\"orbit\",\"container\":\"server\",\"port\":\"health\"}","{\"kind\":\"declared-port\",\"resource\":\"orbit\",\"container\":\"server\",\"port\":\"web\"}","{\"kind\":\"deployment-label\",\"resource\":\"orbit\",\"key\":\"app\"}","{\"kind\":\"readiness-port\",\"resource\":\"orbit\",\"container\":\"server\"}","{\"kind\":\"service-label\",\"resource\":\"gateway\",\"key\":\"app\"}"]}}"#
	},
]
