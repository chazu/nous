package domains

units: [
	{
		name: "H-TransformAcquireConcretePrograms"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Materialize locally observed concrete edit programs before any schema generalization"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "tsAcquire" =
			"CurUnit" @ "TransformLearningExperiment" isa? and
			"CurUnit" @ "tsAcquired" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "meterToken" get-slot "meter" !
			0 list-of "programs" !
			0 "positiveCount" !
			"TransformTrainingCase" examples
			each
				it "example" !
				"example" @ "TransformTrainingCase" !=
				"example" @ "experiment" get-slot "experiment" @ = and
				"example" @ "kind" get-slot "positive" = and
				if
					"positiveCount" @ 1 + "positiveCount" !
					"example" @ "before" get-slot "before" !
					"example" @ "after" get-slot "after" !
					0 list-of "edits" !
					12 iota each
						it "nodeID" !
						"before" @ "nodeID" @ ts-node-facts "beforeFacts" !
						"after" @ "nodeID" @ ts-node-facts "afterFacts" !
						"meter" @ 0 "node" "example" @ "beforeFacts" @ ts-digest "ok" ts-meter drop
						"meter" @ 0 "node" "example" @ "afterFacts" @ ts-digest "ok" ts-meter drop
						"beforeFacts" @ nil != "afterFacts" @ nil != and
						if
							"beforeFacts" @ 0 list-get "kind" !
							"afterFacts" @ 0 list-get "kind" @ =
							"kind" @ "definition" = "kind" @ "reference" = or and
							"before" @ "nodeID" @ ts-parent-facts "after" @ "nodeID" @ ts-parent-facts = and
							"before" @ "nodeID" @ ts-target "after" @ "nodeID" @ ts-target = and
							"beforeFacts" @ 1 list-get "afterFacts" @ 1 list-get != and
							if
								"nodeID" @ "afterFacts" @ 1 list-get ts-make-edit "edit" !
								"edit" @ nil !=
								if "edits" @ "edit" @ list-append "edits" ! then
							then
						then
					end
					"edits" @ ts-make-program "program" !
					"program" @ nil !=
					if
						"before" @ "program" @ ts-program-apply "actual" !
						"actual" @ "after" @ =
						if
							"TS.Program." "example" @ concat "programName" !
							"programName" @ "TransformConcreteProgram" create-unit drop
							"program" @ "programName" @ "program" set-slot
							"example" @ "programName" @ "example" set-slot
							"experiment" @ "programName" @ "experiment" set-slot
							"H-TransformAcquireConcretePrograms" "programName" @ "creditors" set-slot
							"programs" @ "programName" @ list-append "programs" !
						then
					then
				then
			end
			"programs" @ "experiment" @ "programUnits" set-slot
			"programs" @ list-length 4 = "positiveCount" @ 4 = and
			if
				700 "experiment" @ "tsRefine" "Materialize schema factor alternatives" add-task
			else
				"no-discovery" "experiment" @ "terminal" set-slot
			then
			true "experiment" @ "tsAcquired" set-slot
			"""#
	},
	{
		name: "H-TransformRefineSchemaFactors"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Traverse target, anchor, scope, old-guard, and locality alternatives while retaining counterexamples"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "tsRefine" =
			"CurUnit" @ "TransformLearningExperiment" isa? and
			"CurUnit" @ "tsRefined" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			0 list-of "candidates" !
			"target" "definition" "target" "references" "target" "definition+references"
			"anchor" "request-target" "anchor" "from-value" "anchor" "first-local"
			"scope" "local" "scope" "global"
			"old-guard" "equals-from" "old-guard" "any"
			"locality" "required" "locality" "none"
			24 list-of "pairs" !
			12 iota each
				it "index" !
				"pairs" @ "index" @ 2 * list-get "stage" !
				"pairs" @ "index" @ 2 * 1 + list-get "value" !
				"TS.Candidate." "experiment" @ concat "." concat "stage" @ concat "." concat "value" @ concat "candidate" !
				"candidate" @ "TransformPartialCandidate" create-unit drop
				"experiment" @ "candidate" @ "experiment" set-slot
				"stage" @ "candidate" @ "stage" set-slot
				"value" @ "candidate" @ "value" set-slot
				"pending" "candidate" @ "status" set-slot
				"H-TransformRefineSchemaFactors" "candidate" @ "creditors" set-slot
				"candidates" @ "candidate" @ list-append "candidates" !
			end
			"candidates" @ "experiment" @ "candidateUnits" set-slot
			700 "experiment" @ "tsClose" "Evaluate explicit factor alternatives" add-task
			true "experiment" @ "tsRefined" set-slot
			"""#
	},
	{
		name: "H-TransformCloseEvidenceBarriers"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Promote a schema only after every factor alternative has an attached result and exactly one survivor"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "tsClose" =
			"CurUnit" @ "TransformLearningExperiment" isa? and
			"CurUnit" @ "tsClosed" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"awaiting-factor-evidence" "experiment" @ "terminal" set-slot
			true "experiment" @ "tsClosed" set-slot
			"""#
	},
]
