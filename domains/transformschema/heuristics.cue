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
			0 list-of "programs" !
			0 "positiveCount" !
			"TransformTrainingCase" examples
			each
				it "example" !
				"example" @ "TransformTrainingCase" !=
				"example" @ "experiment" get-slot "experiment" @ ts-eq and
				"example" @ "kind" get-slot "positive" ts-eq and
				if
					"positiveCount" @ 1 + "positiveCount" !
					"example" @ "before" get-slot "before" !
					"example" @ "after" get-slot "after" !
					0 list-of "edits" !
					12 iota each
						it "nodeID" !
						"before" @ "nodeID" @ ts-node-facts "beforeFacts" !
						"after" @ "nodeID" @ ts-node-facts "afterFacts" !
						"beforeFacts" @ nil ts-eq not "afterFacts" @ nil ts-eq not and
						if
							"beforeFacts" @ 0 list-get "kind" !
							"afterFacts" @ 0 list-get "kind" @ ts-eq
							"kind" @ "definition" ts-eq "kind" @ "reference" ts-eq or and
							"before" @ "nodeID" @ ts-parent-facts "after" @ "nodeID" @ ts-parent-facts ts-eq and
							"before" @ "nodeID" @ ts-target "after" @ "nodeID" @ ts-target ts-eq and
							"beforeFacts" @ 1 list-get "afterFacts" @ 1 list-get ts-eq not and
							if
								"nodeID" @ "afterFacts" @ 1 list-get ts-make-edit "edit" !
								"edit" @ nil ts-eq not
								if "edits" @ "edit" @ list-append "edits" ! then
							then
						then
					end
					"edits" @ ts-make-program "program" !
					"program" @ nil ts-eq not
					if
						"before" @ "program" @ ts-program-apply "actual" !
					"actual" @ "after" @ ts-output-compare
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
			"programs" @ list-length 4 ts-eq "positiveCount" @ 4 ts-eq and
			if
				"H-TransformAcquireConcretePrograms" "acquisitionOnly" get-slot true =
				if true "experiment" @ "tsAcquisitionClosed" set-slot
				else 700 "experiment" @ "tsRefine" "Materialize schema factor alternatives" add-task then
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
			"TransformRootPartial" "root" !
			"root" @ "partial" get-slot ts-candidate-allocate drop
			"experiment" @ "root" @ "experiment" set-slot
			"H-TransformRefineSchemaFactors" "root" @ "creditors" set-slot
			0 list-of "candidates" !
			"candidates" @ "root" @ list-append "candidates" !
			"root" @ "experiment" @ "rootCandidate" set-slot
			"target" "experiment" @ "currentStage" set-slot
			0 list-of "edges" !
			"edges" @ "experiment" @ "edgeUnits" set-slot
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
				700 "priority" !
				"stage" @ "target" = if 900 "priority" ! then
				"stage" @ "anchor" = if 850 "priority" ! then
				"stage" @ "scope" = if 800 "priority" ! then
				"stage" @ "old-guard" = if 750 "priority" ! then
				"stage" @ "target" = if "priority" @ "candidate" @ "tsEvaluateFactor" "Evaluate one explicit factor alternative" add-task then
			end
			"candidates" @ "experiment" @ "candidateUnits" set-slot
			true "experiment" @ "tsRefined" set-slot
			"""#
	},
	{
		name: "H-TransformEvaluateFactor"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Evaluate one factor alternative from explicit local observations and acquired programs"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "tsEvaluateFactor" =
			"CurUnit" @ "TransformPartialCandidate" isa? and
			"CurUnit" @ "status" get-slot "pending" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "candidate" !
			"candidate" @ "experiment" get-slot "experiment" !
			"candidate" @ "stage" get-slot "stage" !
			"candidate" @ "value" get-slot "value" !
			true "exact" !
			"experiment" @ "rootCandidate" get-slot "parent" !
			"stage" @ "target" !=
			if
				"target" "anchor" "scope" "old-guard" 4 list-of "predecessors" !
				"anchor" "scope" "old-guard" "locality" 4 list-of "successors" !
				4 iota each
					it "stageIndex" !
					"successors" @ "stageIndex" @ list-get "stage" @ ts-eq
					if
						"predecessors" @ "stageIndex" @ list-get "predecessor" !
						"experiment" @ "candidateUnits" get-slot each
							it "other" !
							"other" @ "stage" get-slot "predecessor" @ ts-eq "other" @ "status" get-slot "survivor" ts-eq and
							if "other" @ "parent" ! then
						end
					then
				end
			then
			"parent" @ "partial" get-slot "value" @ ts-refine "childPartial" !
			"childPartial" @ nil ts-eq if false "exact" ! then
			"childPartial" @ ts-candidate-allocate drop
			"childPartial" @ "candidate" @ "partial" set-slot
			"parent" @ "candidate" @ "parentCandidate" set-slot
			"TS.Edge." "candidate" @ concat "edge" !
			"edge" @ "TransformRefinementEdge" create-unit drop
			"experiment" @ "edge" @ "experiment" set-slot
			"parent" @ "edge" @ "parentCandidate" set-slot
			"candidate" @ "edge" @ "childCandidate" set-slot
			"stage" @ "edge" @ "stage" set-slot
			"value" @ "edge" @ "value" set-slot
			"parent" @ "partial" get-slot "edge" @ "parentPartial" set-slot
			"childPartial" @ "edge" @ "childPartial" set-slot
			"H-TransformEvaluateFactor" "edge" @ "creditors" set-slot
			"edge" @ "candidate" @ "refinementEdge" set-slot
			"experiment" @ "edgeUnits" get-slot "edge" @ list-append "experiment" @ "edgeUnits" set-slot
			"stage" @ "target" ts-eq
			if
				"experiment" @ "programUnits" get-slot each
					it "programUnit" !
					"programUnit" @ "program" get-slot ts-program-edits "edits" !
					"programUnit" @ "example" get-slot "before" get-slot "before" !
					false "hasDefinition" ! false "hasReference" !
					"edits" @ each
						it 0 list-get "editID" !
						"before" @ "editID" @ ts-node-facts 0 list-get "kind" !
						"kind" @ "definition" ts-eq if true "hasDefinition" ! then
						"kind" @ "reference" ts-eq if true "hasReference" ! then
					end
					"value" @ "definition" ts-eq "hasDefinition" @ "hasReference" @ not and and
					"value" @ "references" ts-eq "hasReference" @ "hasDefinition" @ not and and or
					"value" @ "definition+references" ts-eq "hasDefinition" @ "hasReference" @ and and or
					"caseExact" !
					"caseExact" @ not if false "exact" ! then
				end
			else
				"stage" @ "anchor" ts-eq
				if
					"experiment" @ "programUnits" get-slot each
						it "programUnit" !
						"programUnit" @ "program" get-slot ts-program-edits "edits" !
						"programUnit" @ "example" get-slot "before" get-slot "before" !
						nil "requestID" ! nil "definitionID" !
						12 iota each
							it "nodeID" !
							"before" @ "nodeID" @ ts-node-facts "facts" !
							"facts" @ nil !=
							if "facts" @ 0 list-get "request" ts-eq if "nodeID" @ "requestID" ! then then
						end
						"edits" @ each
							it 0 list-get "editID" !
							"before" @ "editID" @ ts-node-facts 0 list-get "kind" !
							"kind" @ "definition" ts-eq if "editID" @ "definitionID" ! then
							"kind" @ "reference" ts-eq if "before" @ "editID" @ ts-target "definitionID" ! then
						end
						false "caseExact" !
						"value" @ "request-target" ts-eq
						if "before" @ "requestID" @ ts-target "definitionID" @ ts-eq "caseExact" ! then
						"value" @ "from-value" ts-eq
						if
							"before" @ "requestID" @ ts-node-facts 2 list-get "fromValue" !
							0 "matchingDefinitions" ! nil "matchingID" !
							12 iota each
								it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" !
								"facts" @ nil != if "facts" @ 0 list-get "definition" ts-eq "facts" @ 1 list-get "fromValue" @ ts-eq and if "matchingDefinitions" @ 1 + "matchingDefinitions" ! "nodeID" @ "matchingID" ! then then
							end
							"matchingDefinitions" @ 1 ts-eq "matchingID" @ "definitionID" @ ts-eq and "caseExact" !
						then
						"value" @ "first-local" ts-eq
						if
							"before" @ "requestID" @ ts-parent-facts 0 list-get "requestParent" !
							nil "firstID" !
							12 iota each
								it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" !
								"facts" @ nil != if "facts" @ 0 list-get "definition" ts-eq "firstID" @ nil ts-eq and if "before" @ "nodeID" @ ts-parent-facts 0 list-get "requestParent" @ ts-eq if "nodeID" @ "firstID" ! then then then
							end
							"firstID" @ "definitionID" @ ts-eq "caseExact" !
						then
						"caseExact" @ not if false "exact" ! then
					end
				else
					"stage" @ "scope" ts-eq
					if
						"" "selectedTarget" !
						"experiment" @ "candidateUnits" get-slot each
							it "other" !
							"other" @ "stage" get-slot "target" ts-eq "other" @ "status" get-slot "survivor" ts-eq and if "other" @ "value" get-slot "selectedTarget" ! then
						end
						"selectedTarget" @ "definition" ts-eq
						if
							"value" @ "local" ts-eq "exact" !
							"exact" @ not if "redundant-noncanonical" "candidate" @ "disposition" set-slot then
						else
							0 "equalsMatches" ! 0 "anyMatches" !
							"experiment" @ "programUnits" get-slot each
								it "programUnit" !
								"programUnit" @ "program" get-slot ts-program-edits "edits" !
								"programUnit" @ "example" get-slot "before" get-slot "before" !
								nil "requestID" ! nil "definitionID" ! 0 list-of "editedReferences" !
								12 iota each it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" ! "facts" @ nil != if "facts" @ 0 list-get "request" ts-eq if "nodeID" @ "requestID" ! then then end
								"edits" @ each
									it 0 list-get "editID" ! "before" @ "editID" @ ts-node-facts 0 list-get "kind" !
									"kind" @ "definition" ts-eq if "editID" @ "definitionID" ! then
									"kind" @ "reference" ts-eq if "before" @ "editID" @ ts-target "definitionID" ! "editedReferences" @ "editID" @ list-append "editedReferences" ! then
								end
								"before" @ "requestID" @ ts-parent-facts 0 list-get "requestParent" !
								"before" @ "requestID" @ ts-node-facts 2 list-get "fromValue" !
								0 list-of "equalsPredicted" ! 0 list-of "anyPredicted" !
								12 iota each
									it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" !
									"facts" @ nil != if "facts" @ 0 list-get "reference" ts-eq if
										"before" @ "nodeID" @ ts-target "definitionID" @ ts-eq
										"value" @ "global" ts-eq "before" @ "nodeID" @ ts-parent-facts 0 list-get "requestParent" @ ts-eq or and
										if
											"anyPredicted" @ "nodeID" @ list-append "anyPredicted" !
											"facts" @ 1 list-get "fromValue" @ ts-eq if "equalsPredicted" @ "nodeID" @ list-append "equalsPredicted" ! then
										then
									then then
								end
								"equalsPredicted" @ "editedReferences" @ ts-eq if "equalsMatches" @ 1 + "equalsMatches" ! then
								"anyPredicted" @ "editedReferences" @ ts-eq if "anyMatches" @ 1 + "anyMatches" ! then
							end
							"equalsMatches" @ 4 ts-eq "anyMatches" @ 4 ts-eq or "exact" !
						then
					else
						"stage" @ "old-guard" ts-eq
						if
							"" "selectedTarget" ! "" "selectedScope" !
							"experiment" @ "candidateUnits" get-slot each
								it "other" !
								"other" @ "status" get-slot "survivor" ts-eq if
									"other" @ "stage" get-slot "target" ts-eq if "other" @ "value" get-slot "selectedTarget" ! then
									"other" @ "stage" get-slot "scope" ts-eq if "other" @ "value" get-slot "selectedScope" ! then
								then
							end
							"selectedTarget" @ "definition" ts-eq
							if
								"value" @ "any" ts-eq "exact" !
								"exact" @ not if "redundant-noncanonical" "candidate" @ "disposition" set-slot then
							else
								0 "matches" !
								"experiment" @ "programUnits" get-slot each
									it "programUnit" !
									"programUnit" @ "program" get-slot ts-program-edits "edits" !
									"programUnit" @ "example" get-slot "before" get-slot "before" !
									nil "requestID" ! nil "definitionID" ! 0 list-of "editedReferences" !
									12 iota each it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" ! "facts" @ nil != if "facts" @ 0 list-get "request" ts-eq if "nodeID" @ "requestID" ! then then end
									"edits" @ each
										it 0 list-get "editID" ! "before" @ "editID" @ ts-node-facts 0 list-get "kind" !
										"kind" @ "definition" ts-eq if "editID" @ "definitionID" ! then
										"kind" @ "reference" ts-eq if "before" @ "editID" @ ts-target "definitionID" ! "editedReferences" @ "editID" @ list-append "editedReferences" ! then
									end
									"before" @ "requestID" @ ts-parent-facts 0 list-get "requestParent" !
									"before" @ "requestID" @ ts-node-facts 2 list-get "fromValue" !
									0 list-of "predicted" !
									12 iota each
										it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" !
										"facts" @ nil != if "facts" @ 0 list-get "reference" ts-eq if
											"before" @ "nodeID" @ ts-target "definitionID" @ ts-eq
											"selectedScope" @ "global" ts-eq "before" @ "nodeID" @ ts-parent-facts 0 list-get "requestParent" @ ts-eq or and
											"value" @ "any" ts-eq "facts" @ 1 list-get "fromValue" @ ts-eq or and
											if "predicted" @ "nodeID" @ list-append "predicted" ! then
										then then
									end
									"predicted" @ "editedReferences" @ ts-eq if "matches" @ 1 + "matches" ! then
								end
								"matches" @ 4 ts-eq "exact" !
							then
						else
							"stage" @ "locality" ts-eq
							if
								false "sawWrongContext" !
								"TransformTrainingCase" examples each
									it "example" !
									"example" @ "TransformTrainingCase" != "example" @ "experiment" get-slot "experiment" @ ts-eq and "example" @ "kind" get-slot "abstain" ts-eq and
									if
										"example" @ "before" get-slot "before" ! nil "requestID" ! 0 "requestCount" !
										12 iota each it "nodeID" ! "before" @ "nodeID" @ ts-node-facts "facts" ! "facts" @ nil != if "facts" @ 0 list-get "request" ts-eq if "requestCount" @ 1 + "requestCount" ! "nodeID" @ "requestID" ! then then end
										"requestCount" @ 1 ts-eq
										if
											"before" @ "requestID" @ ts-target "definitionID" !
											"before" @ "requestID" @ ts-parent-facts 0 list-get
											"before" @ "definitionID" @ ts-parent-facts 0 list-get !=
											if true "sawWrongContext" ! then
										then
									then
								end
								"value" @ "required" ts-eq "sawWrongContext" @ and "exact" !
							else
								false "exact" !
							then
						then
					then
				then
			then
			"H-TransformEvaluateFactor" "ablateEquality" get-slot true ts-eq
			"stage" @ "old-guard" ts-eq and "value" @ "equals-from" ts-eq and
			if false "exact" ! "ablated-ineligible" "candidate" @ "disposition" set-slot then
			"exact" @ ts-factor-result "exact" !
			"exact" @ if "survivor" else "rejected" then "candidate" @ "status" set-slot
			"TS.Evidence." "candidate" @ concat "evidence" !
			"evidence" @ "TransformFactorEvidence" create-unit drop
			"experiment" @ "evidence" @ "experiment" set-slot
			"candidate" @ "evidence" @ "candidate" set-slot
			"stage" @ "evidence" @ "stage" set-slot
			"value" @ "evidence" @ "value" set-slot
			"exact" @ "evidence" @ "matched" set-slot
			"H-TransformEvaluateFactor" "evidence" @ "creditors" set-slot
			"evidence" @ "candidate" @ "evidenceUnit" set-slot
			"H-TransformEvaluateFactor" "candidate" @ "creditors" set-slot
			600 "experiment" @ "tsClose" "Close factor barriers when every result exists" add-task
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
			"experiment" @ "currentStage" get-slot "currentStage" !
			0 "stagePending" !
			"experiment" @ "candidateUnits" get-slot each
				it "stageCandidate" !
				"stageCandidate" @ "stage" get-slot "currentStage" @ =
				"stageCandidate" @ "status" get-slot "pending" = and
				if "stagePending" @ 1 + "stagePending" ! then
			end
			"stagePending" @ 0 =
			if
				"experiment" @ "currentStage" @ ts-close-stage "stageClosed" !
				"stageClosed" @ "currentStage" @ "locality" != and
				if
					"target" "anchor" "scope" "old-guard" 4 list-of "predecessors" !
					"anchor" "scope" "old-guard" "locality" 4 list-of "successors" !
					"" "nextStage" !
					4 iota each
						it "stageIndex" !
						"predecessors" @ "stageIndex" @ list-get "currentStage" @ =
						if "successors" @ "stageIndex" @ list-get "nextStage" ! then
					end
					"nextStage" @ "experiment" @ "currentStage" set-slot
					"experiment" @ "candidateUnits" get-slot each
						it "nextCandidate" !
						"nextCandidate" @ "stage" get-slot "nextStage" @ =
						if 700 "nextCandidate" @ "tsEvaluateFactor" "Evaluate one explicit factor alternative" add-task then
					end
				then
			then
			0 "pending" !
			0 "targetCount" ! 0 "anchorCount" ! 0 "scopeCount" ! 0 "guardCount" ! 0 "localityCount" !
			"" "target" ! "" "anchor" ! "" "scope" ! "" "guard" ! "" "locality" !
			"experiment" @ "candidateUnits" get-slot each
				it "candidate" !
				"candidate" @ "status" get-slot "pending" = if "pending" @ 1 + "pending" ! then
				"candidate" @ "status" get-slot "survivor" =
				if
					"candidate" @ "stage" get-slot "target" = if "targetCount" @ 1 + "targetCount" ! "candidate" @ "value" get-slot "target" ! then
					"candidate" @ "stage" get-slot "anchor" = if "anchorCount" @ 1 + "anchorCount" ! "candidate" @ "value" get-slot "anchor" ! then
					"candidate" @ "stage" get-slot "scope" = if "scopeCount" @ 1 + "scopeCount" ! "candidate" @ "value" get-slot "scope" ! then
					"candidate" @ "stage" get-slot "old-guard" = if "guardCount" @ 1 + "guardCount" ! "candidate" @ "value" get-slot "guard" ! then
					"candidate" @ "stage" get-slot "locality" = if "localityCount" @ 1 + "localityCount" ! "candidate" @ "value" get-slot "locality" ! then
				then
			end
			"pending" @ 0 = "targetCount" @ 1 = and "anchorCount" @ 1 = and "scopeCount" @ 1 = and "guardCount" @ 1 = and "localityCount" @ 1 = and
			true "closuresValid" !
			"target" "anchor" "scope" "old-guard" "locality" 5 list-of each
				it "closureStage" !
				"experiment" @ "closureStage" @ ts-close-stage "closuresValid" @ and "closuresValid" !
			end
			"closuresValid" @ and
			if
				"anchor" @ "target" @ "scope" @ "guard" @ "locality" @ ts-make-schema ts-freeze-schema "schema" !
				true "valid" !
				false "budgetExhausted" !
				"TransformTrainingCase" examples each
					it "example" !
					"example" @ "TransformTrainingCase" != "example" @ "experiment" get-slot "experiment" @ = and
					if
						"example" @ "before" get-slot "schema" @ "example" @ "kind" get-slot ts-schema-apply "result" !
						"result" @ nil =
						if false "valid" !
					else
							"result" @ 0 list-get "terminal" ! "result" @ 1 list-get "output" !
							"terminal" @ "budget-exhausted" =
							if true "budgetExhausted" ! false "valid" ! then
							"example" @ "kind" get-slot "positive" =
							if "terminal" @ "applied" = "output" @ "example" @ "after" get-slot ts-output-compare and not if false "valid" ! then
							else
								"abstain/request-count" "abstain/anchor" "abstain/locality" "abstain/expansion" "abstain/no-op" 5 list-of "abstentions" !
								"abstentions" @ "terminal" @ list-contains not if false "valid" ! then
							then
						then
					then
				end
				"budgetExhausted" @
				if
					"budget-exhausted" "experiment" @ "terminal" set-slot
				else
				"valid" @
				if
					0 list-of "barriers" !
					"target" "anchor" "scope" "old-guard" "locality" 5 list-of each
						it "stage" ! "TS.Barrier." "experiment" @ concat "." concat "stage" @ concat "barrier" !
						"barrier" @ "TransformEvidenceBarrier" create-unit drop
						"experiment" @ "barrier" @ "experiment" set-slot
						"stage" @ "barrier" @ "stage" set-slot
						true "barrier" @ "sealed" set-slot
						"H-TransformCloseEvidenceBarriers" "barrier" @ "creditors" set-slot
						"barriers" @ "barrier" @ list-append "barriers" !
					end
					"TS.Artifact." "experiment" @ concat "artifact" !
					"artifact" @ "TransformSchemaArtifact" create-unit drop
					"schema" @ "artifact" @ "schema" set-slot
					"barriers" @ "artifact" @ "evidenceBarriers" set-slot
					"experiment" @ "candidateUnits" get-slot "artifact" @ "candidateUnits" set-slot
					"H-TransformCloseEvidenceBarriers" "artifact" @ "creditors" set-slot
					"artifact" @ "experiment" @ "artifactUnit" set-slot
					"completed" "experiment" @ "terminal" set-slot
				else
					"no-discovery" "experiment" @ "terminal" set-slot
				then
				then
				true "experiment" @ "tsClosed" set-slot
			then
			"""#
	},
]
