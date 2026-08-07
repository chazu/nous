package domains

units: [
	{
		name: "H-EnumerateMemoryOneStrategies"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Enumerate every deterministic memory-one strategy declared by a valid game experiment"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "IteratedGameExperiment" isa?
			"CurUnit" @ "IteratedGameExperiment" != and
			"CurUnit" @ "generationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ "generationComplete" get-slot nil = and
			"CurUnit" @ game-experiment-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ game-generate-experiment drop
			"""#
	},
	{
		name: "H-EvaluateGameStrategy"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Evaluate one generated strategy against the complete declared game profile"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "gameExperiment" get-slot "experiment" !
			"experiment" @ nil !=
			"experiment" @ game-experiment-valid? and
			"experiment" @ "evaluationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ "evaluatedGameProfile" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ game-evaluate-candidate drop
			"""#
	},
	{
		name: "H-SelectGameStrategyFrontier"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Select the nondominated strategy frontier after the complete evidence barrier"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "IteratedGameExperiment" isa?
			"CurUnit" @ "IteratedGameExperiment" != and
			"CurUnit" @ "finalizationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ game-ready-to-finalize? and
			"""#
		thenCompute: #"""
			"CurUnit" @ game-finalize-experiment drop
			"""#
	},
]
