package causalcurriculum

units: [
	{
		name: "H-Causal-Curriculum-Initialize"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccInitialize" =
			"CurUnit" @ "CausalCurriculumCursor" isa? and
			"CurUnit" @ "phase" get-slot "initializing" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "runtime" !
			"runtime" @ cc-task-charge drop
			"runtime" @ cc-initialize-charge drop
			0 list-of "ruleUnits" !
			cc-rule-root cc-refine-rule
			each
				it "primaryRule" !
				"primaryRule" @ cc-refine-rule
				each
					it "modeRule" !
					"modeRule" @ cc-refine-rule
					each
						it "runtime" @ cc-materialize-rule "ruleUnit" !
						"ruleUnits" @ "ruleUnit" @ list-append "ruleUnits" !
					end
				end
			end
			"ruleUnits" @ "runtime" @ "ruleUnits" set-slot
			"admitting" "runtime" @ "phase" set-slot
			"runtime" @ "certificateUnits" get-slot
			each 900 it "ccAdmit" "admit exact central certificate" add-task end
			"""#
	},
	{
		name: "H-Causal-Curriculum-Admit"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccAdmit" =
			"CurUnit" @ "CausalCentralCertificateArtifact" isa? and
			"CurUnit" @ "runtime" get-slot "runtime" !
			"runtime" @ "phase" get-slot "admitting" = and
			"""#
		thenCompute: #"""
			"runtime" @ cc-task-charge drop
			"CurUnit" @ "runtime" @ cc-admit drop
			"runtime" @ "nextAdmission" get-slot 480 =
			if 900 "runtime" @ "ccMatrix" "complete matrix barrier" add-task then
			"""#
	},
	{
		name: "H-Causal-Curriculum-Matrix-Barrier"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccMatrix" =
			"CurUnit" @ "CausalCurriculumCursor" isa? and
			"CurUnit" @ "phase" get-slot "admitting" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "runtime" !
			"runtime" @ cc-task-charge drop
			"runtime" @ cc-require-matrix drop
			"aggregating" "runtime" @ "phase" set-slot
			"runtime" @ "ruleUnits" get-slot
			each 900 it "ccAggregate" "aggregate exact rule applications" add-task end
			"""#
	},
	{
		name: "H-Causal-Curriculum-Aggregate"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccAggregate" =
			"CurUnit" @ "CausalCentralRuleArtifact" isa? and
			"CurUnit" @ "runtime" get-slot "runtime" !
			"runtime" @ "phase" get-slot "aggregating" = and
			"""#
		thenCompute: #"""
			"runtime" @ cc-task-charge drop
			"CurUnit" @ "runtime" @ cc-materialize-aggregate drop
			"runtime" @ "nextAggregate" get-slot 40 =
			if 900 "runtime" @ "ccAggregates" "complete aggregate barrier" add-task then
			"""#
	},
	{
		name: "H-Causal-Curriculum-Aggregate-Barrier"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccAggregates" =
			"CurUnit" @ "CausalCurriculumCursor" isa? and
			"CurUnit" @ "phase" get-slot "aggregating" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "runtime" !
			"runtime" @ cc-task-charge drop
			"runtime" @ cc-require-aggregates drop
			"selecting" "runtime" @ "phase" set-slot
			900 "runtime" @ "ccSelect" "central selection barrier" add-task
			"""#
	},
	{
		name: "H-Causal-Curriculum-Select"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccSelect" =
			"CurUnit" @ "CausalCurriculumCursor" isa? and
			"CurUnit" @ "phase" get-slot "selecting" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "runtime" !
			"runtime" @ cc-task-charge drop
			"runtime" @ "creditEnabled" get-slot
			if
				"" "best" !
				"runtime" @ "aggregateUnits" get-slot
				each
					it "candidate" !
					"best" @ "" =
					if
						"candidate" @ "best" !
					else
						"candidate" @ "best" @ "runtime" @ cc-better?
						if "candidate" @ "best" ! then
					then
				end
				0 list-of "ties" !
				"runtime" @ "aggregateUnits" get-slot
				each
					it "candidate" !
					"candidate" @ "best" @ "runtime" @ cc-exact-tie?
					if
						"candidate" @ "runtime" @ cc-materialize-tie "tie" !
						"ties" @ "tie" @ list-append "ties" !
					then
				end
				"best" @ "ruleCode" get-slot "ties" @ "runtime" @ cc-materialize-selection drop
			else
				true "runtime" @ "unresolved" set-slot
			then
			"transcribing" "runtime" @ "phase" set-slot
			900 "runtime" @ "ccTranscript" "central transcript barrier" add-task
			"""#
	},
	{
		name: "H-Causal-Curriculum-Transcript"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ccTranscript" =
			"CurUnit" @ "CausalCurriculumCursor" isa? and
			"CurUnit" @ "phase" get-slot "transcribing" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "runtime" !
			"runtime" @ cc-task-charge drop
			"runtime" @ "creditEnabled" get-slot
			if
				"runtime" @ "creditUnits" get-slot
				each it "runtime" @ cc-materialize-transcript-event drop end
				"runtime" @ "aggregateUnits" get-slot
				each it "runtime" @ cc-materialize-transcript-event drop end
				"runtime" @ "selectionUnit" get-slot "runtime" @ cc-materialize-transcript-event drop
			then
			"runtime" @ cc-require-terminal drop
			"terminal" "runtime" @ "phase" set-slot
			"""#
	},
]
