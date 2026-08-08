package domains

units: [
	{
		name: "H-RI-Start"
		worth: 750
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "start" ri-task-valid?
			"CurUnit" @ "riStarted" get-slot nil = and
			"CurUnit" @ "experimentProfileKey" get-slot nil =
			"CurUnit" @ ri-profile-valid? or and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "rootName" get-slot "rootBase" !
			"root:" "experiment" @ "stage" get-slot concat "rootSemantic" !
			"experiment" @ "experiment" @ "stage" get-slot "partial" "rootSemantic" @ "rootBase" @ ri-artifact-name "root" !
			"root" @ "experiment" @ "rootName" set-slot
			"root" @ unit-exists? not
			if
				"root" @ "experiment" @ "partialCategory" get-slot create-unit drop
				"root" @ "experiment" @ "experiment" @ "stage" get-slot "partial" "rootSemantic" @ ri-envelope drop
				"----" "root" @ "partial" set-slot
				"experiment" @ "root" @ "experiment" set-slot
				"experiment" @ "stage" get-slot "root" @ "stage" set-slot
				"experiment" @ "refinementPriority" get-slot "root" @ "experiment" @ "refineTaskSlot" get-slot "Refine one typed field" add-task
			then
			"experiment" @ "experiment" @ "stage" get-slot "start" "root" @ 1 ri-record-action drop
			true "experiment" @ "riStarted" set-slot
			"""#
	},
	{
		name: "H-RI-Refine"
		worth: 750
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "refine" ri-task-valid?
			"CurUnit" @ "riRefined" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "parent" !
			"parent" @ "experiment" get-slot "experiment" !
			"parent" @ "partial" get-slot ri-refine-one
			each
				it "childPartial" !
				"RI.Partial." "experiment" @ concat "." concat "experiment" @ "stage" get-slot concat "." concat "childPartial" @ concat "childBase" !
				"partial:" "childPartial" @ concat "childSemantic" !
				"experiment" @ "experiment" @ "stage" get-slot "partial" "childSemantic" @ "childBase" @ ri-artifact-name "childName" !
				"childName" @ unit-exists? not
				if
					"childName" @ "experiment" @ "partialCategory" get-slot create-unit drop
					"childName" @ "experiment" @ "experiment" @ "stage" get-slot "partial" "childSemantic" @ ri-envelope drop
					"childPartial" @ "childName" @ "partial" set-slot
					"experiment" @ "childName" @ "experiment" set-slot
					"experiment" @ "stage" get-slot "childName" @ "stage" set-slot
					"parent" @ "childName" @ "refinedFrom" set-slot
				then
				"RI.Refinement." "parent" @ concat "." concat "childName" @ concat "edgeBase" !
				"refinement:" "parent" @ concat ":" concat "childName" @ concat "edgeSemantic" !
				"experiment" @ "experiment" @ "stage" get-slot "refinement" "edgeSemantic" @ "edgeBase" @ ri-artifact-name "edge" !
				"edge" @ unit-exists? not
				if
					"edge" @ "experiment" @ "refinementCategory" get-slot create-unit drop
					"edge" @ "experiment" @ "experiment" @ "stage" get-slot "refinement" "edgeSemantic" @ ri-envelope drop
					"experiment" @ "edge" @ "experiment" set-slot
					"experiment" @ "stage" get-slot "edge" @ "stage" set-slot
					"parent" @ "edge" @ "parent" set-slot
					"childName" @ "edge" @ "child" set-slot
				then
				"experiment" @ "experiment" @ "stage" get-slot "refine" "childName" @ "childPartial" @ ri-refinement-work ri-record-action drop
				"childPartial" @ ri-complete-code "code" !
				"code" @ nil =
				if
					"experiment" @ "refinementPriority" get-slot "childName" @ "experiment" @ "refineTaskSlot" get-slot "Refine one typed field" add-task
				else
					"RI.Candidate." "experiment" @ concat "." concat "experiment" @ "stage" get-slot concat "." concat "code" @ concat "candidateBase" !
					"candidate:" "code" @ concat "candidateSemantic" !
					"experiment" @ "experiment" @ "stage" get-slot "candidate" "candidateSemantic" @ "candidateBase" @ ri-artifact-name "candidate" !
					"candidate" @ unit-exists? not
					if
						"candidate" @ "experiment" @ "candidateCategory" get-slot create-unit drop
						"candidate" @ "experiment" @ "experiment" @ "stage" get-slot "candidate" "candidateSemantic" @ ri-envelope drop
						"code" @ "candidate" @ "definitionCode" set-slot
						"experiment" @ "candidate" @ "experiment" set-slot
						"experiment" @ "stage" get-slot "candidate" @ "stage" set-slot
						"childName" @ "candidate" @ "completedBy" set-slot
						"code" @ "experiment" @ "queue" get-slot ri-queue-rank "rank" !
						"rank" @ nil =
						if
							true "candidate" @ "ineligible" set-slot
						else
							"rank" @ "candidate" @ "queueRank" set-slot
							"experiment" @ "evaluationPriority" get-slot "rank" @ - "candidate" @ "experiment" @ "evaluateTaskSlot" get-slot "Evaluate explicit Horn definition" add-task
						then
					then
				then
			end
			true "parent" @ "riRefined" set-slot
			"""#
	},
	{
		name: "H-RI-Evaluate"
		worth: 750
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "evaluate" ri-task-valid?
			"CurUnit" @ "riEvaluated" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "candidate" !
			"candidate" @ "experiment" get-slot "experiment" !
			"experiment" @ "terminal" get-slot nil =
			if
				"candidate" @ "definitionCode" get-slot "code" !
				"experiment" @ "reuseMode" get-slot "naive-direct" !=
				"experiment" @ "reuseMode" get-slot "uniform-random" != and
				if
					"experiment" @ "constraintCategory" get-slot examples
					each
						it "constraint" !
						"constraint" @ "experiment" @ "constraintCategory" get-slot !=
						"constraint" @ "experiment" get-slot "experiment" @ = and
						"constraint" @ "stage" get-slot "candidate" @ "stage" get-slot = and
						if
							"constraint" @ "direction" get-slot "too-general" =
							if
								"code" @ "constraint" @ "failedCode" get-slot ri-structural-subsumes? "pruned" !
								"code" @ "constraint" @ "failedCode" get-slot ri-structural-work "thetaWork" !
							else
								"constraint" @ "failedCode" get-slot "code" @ ri-structural-subsumes? "pruned" !
								"constraint" @ "failedCode" get-slot "code" @ ri-structural-work "thetaWork" !
							then
							"RI.Comparison." "constraint" @ concat "." concat "candidate" @ concat "comparisonBase" !
							"comparison:" "constraint" @ concat ":" concat "candidate" @ concat "comparisonSemantic" !
							"experiment" @ "candidate" @ "stage" get-slot "comparison" "comparisonSemantic" @ "comparisonBase" @ ri-artifact-name "comparison" !
							"comparison" @ unit-exists? not
							if
								"comparison" @ "experiment" @ "comparisonCategory" get-slot create-unit drop
								"comparison" @ "experiment" @ "candidate" @ "stage" get-slot "comparison" "comparisonSemantic" @ ri-envelope drop
								"experiment" @ "comparison" @ "experiment" set-slot
								"candidate" @ "stage" get-slot "comparison" @ "stage" set-slot
								"constraint" @ "comparison" @ "constraint" set-slot
								"candidate" @ "comparison" @ "candidate" set-slot
								"pruned" @ "comparison" @ "subsumes" set-slot
								"thetaWork" @ "comparison" @ "thetaWork" set-slot
							then
							"experiment" @ "candidate" @ "stage" get-slot "comparison" "comparison" @ "thetaWork" @ ri-record-action drop
						then
					end
				then
				"code" @ "experiment" @ "facts" get-slot ri-evaluation "evaluation" !
				"evaluation" @ ri-evaluation-signature "signature" !
				"evaluation" @ ri-evaluation-work "fixedWork" !
				0 "support" ! 0 "failures" ! 0 "falsePositive" ! 0 "falseNegative" ! 0 "seen" !
				"experiment" @ "experimentProfileKey" get-slot nil =
				if "RuleInductionExample" examples else "experiment" @ "stage1CorpusUnit" get-slot "candidate" @ "stage" get-slot "stage2" = if drop "experiment" @ "stage2CorpusUnit" get-slot then "examples" get-slot then
				each
					it "example" !
					"example" @ "experiment" @ "candidate" @ "stage" get-slot ri-example-for?
					if
						"seen" @ 1 + "seen" !
						"signature" @ "example" @ ri-example-x "example" @ ri-example-y ri-signature-has? "actual" !
						"actual" @ "example" @ ri-example-positive? = "outcome" !
						"outcome" @
						if
							"support" @ 1 + "support" !
						else
							"failures" @ 1 + "failures" !
							"actual" @ if "falsePositive" @ 1 + "falsePositive" ! else "falseNegative" @ 1 + "falseNegative" ! then
						then
						"RI.Result." "candidate" @ concat "." concat "example" @ concat "resultBase" !
						"result:" "candidate" @ concat ":" concat "example" @ concat "resultSemantic" !
						"experiment" @ "candidate" @ "stage" get-slot "result" "resultSemantic" @ "resultBase" @ ri-artifact-name "result" !
						"result" @ unit-exists? not
						if
							"result" @ "experiment" @ "resultCategory" get-slot create-unit drop
							"result" @ "experiment" @ "candidate" @ "stage" get-slot "result" "resultSemantic" @ ri-envelope drop
							"experiment" @ "result" @ "experiment" set-slot
							"candidate" @ "result" @ "candidate" set-slot
							"example" @ "result" @ "example" set-slot
							"actual" @ "result" @ "actual" set-slot
							"outcome" @ "result" @ "outcome" set-slot
						then
						"RI.Observation." "candidate" @ concat "." concat "example" @ concat "observationBase" !
						"observation:" "candidate" @ concat ":" concat "example" @ concat "observationSemantic" !
						"experiment" @ "candidate" @ "stage" get-slot "observation" "observationSemantic" @ "observationBase" @ ri-artifact-name "observation" !
						"observation" @ unit-exists? not
						if
							"observation" @ "experiment" @ "observationCategory" get-slot create-unit drop
							"observation" @ "experiment" @ "candidate" @ "stage" get-slot "observation" "observationSemantic" @ ri-envelope drop
							"experiment" @ "observation" @ "experiment" set-slot
							"candidate" @ "observation" @ "candidate" set-slot
							"example" @ "observation" @ "example" set-slot
							"result" @ "observation" @ "result" set-slot
							"actual" @ "observation" @ "actual" set-slot
							"outcome" @ "observation" @ "outcome" set-slot
						then
					then
				end
				"RI.Evidence." "candidate" @ concat "evidenceBase" !
				"evidence:" "candidate" @ concat "evidenceSemantic" !
				"experiment" @ "candidate" @ "stage" get-slot "evidence" "evidenceSemantic" @ "evidenceBase" @ ri-artifact-name "evidence" !
				"evidence" @ unit-exists? not
				if
					"evidence" @ "experiment" @ "evidenceCategory" get-slot create-unit drop
					"evidence" @ "experiment" @ "candidate" @ "stage" get-slot "evidence" "evidenceSemantic" @ ri-envelope drop
					"experiment" @ "evidence" @ "experiment" set-slot
					"candidate" @ "evidence" @ "candidate" set-slot
					"seen" @ "evidence" @ "exampleCount" set-slot
					"support" @ "evidence" @ "supportCount" set-slot
					"failures" @ "evidence" @ "failureCount" set-slot
				then
				"signature" @ "candidate" @ "signature" set-slot
				"fixedWork" @ "candidate" @ "fixedWork" set-slot
				"seen" @ "candidate" @ "exampleCount" set-slot
				"support" @ "candidate" @ "supportCount" set-slot
				"failures" @ "candidate" @ "failureCount" set-slot
				"falsePositive" @ "candidate" @ "falsePositiveCount" set-slot
				"falseNegative" @ "candidate" @ "falseNegativeCount" set-slot
				true "candidate" @ "riEvaluated" set-slot
				"experiment" @ "candidate" @ "stage" get-slot "evaluation" "candidate" @ "fixedWork" @ "seen" @ + ri-record-action drop
				"experiment" @ "budgetExceeded" get-slot true = "candidate" @ "stage" get-slot "stage2" = and
				if
					"budget-exhausted" "experiment" @ "terminal" set-slot
					"experiment" @ "stage2" "termination" "budget-exhausted" 1 ri-record-action drop
					"experiment" @ "stage2" "terminal" "budget-exhausted" "" ri-record-decision drop
					"experiment" @ "stage2" "decision" "experiment" @ "stage2TerminalUnit" get-slot 0 ri-record-action drop
					"experiment" @ ri-experiment-complete? "experiment" @ "experimentComplete" set-slot
				else
				"failures" @ 0 = "seen" @ 4 >= and
				if
					"candidate" @ ri-ready-to-select?
					if
						800 "candidate" @ "worth" set-slot
						"experiment" @ "reuseMode" get-slot "lff-task-local-invention" =
						"experiment" @ "reuseMode" get-slot "shared-recomputed" = "candidate" @ "stage" get-slot "stage1" = and or
						if
							"RI.LocalLibrary." "experiment" @ concat "." concat "candidate" @ "stage" get-slot concat "localLibraryBase" !
							"local-library:" "candidate" @ "stage" get-slot concat ":" concat "code" @ concat "localLibrarySemantic" !
							"experiment" @ "candidate" @ "stage" get-slot "library" "localLibrarySemantic" @ "localLibraryBase" @ ri-artifact-name "localLibrary" !
							"localLibrary" @ unit-exists? not
							if
								"localLibrary" @ "experiment" @ "libraryCategory" get-slot create-unit drop
								"localLibrary" @ "experiment" @ "candidate" @ "stage" get-slot "library" "localLibrarySemantic" @ ri-envelope drop
								"experiment" @ "localLibrary" @ "experiment" set-slot
								"candidate" @ "stage" get-slot "localLibrary" @ "stage" set-slot
								"code" @ "localLibrary" @ "definitionCode" set-slot
								"signature" @ "localLibrary" @ "materializedRelation" set-slot
								"candidate" @ "localLibrary" @ "learnedFrom" set-slot
								true "localLibrary" @ "taskLocal" set-slot
								"experiment" @ "reuseMode" get-slot "shared-recomputed" = "localLibrary" @ "discardedAtBoundary" set-slot
							then
							"RI.LocalProvenance." "experiment" @ concat "." concat "candidate" @ "stage" get-slot concat "localProvenanceBase" !
							"local-provenance:" "candidate" @ "stage" get-slot concat ":" concat "code" @ concat "localProvenanceSemantic" !
							"experiment" @ "candidate" @ "stage" get-slot "provenance" "localProvenanceSemantic" @ "localProvenanceBase" @ ri-artifact-name "localProvenance" !
							"localProvenance" @ unit-exists? not
							if
								"localProvenance" @ "experiment" @ "provenanceCategory" get-slot create-unit drop
								"localProvenance" @ "experiment" @ "candidate" @ "stage" get-slot "provenance" "localProvenanceSemantic" @ ri-envelope drop
								"experiment" @ "localProvenance" @ "experiment" set-slot
								"candidate" @ "stage" get-slot "localProvenance" @ "stage" set-slot
								"localLibrary" @ "localProvenance" @ "library" set-slot
								"candidate" @ "localProvenance" @ "candidate" set-slot
								"code" @ "localProvenance" @ "definitionCode" set-slot
							then
							"experiment" @ "reuseMode" get-slot "lff-task-local-invention" =
							if
								"RI.LocalProjection." "experiment" @ concat "." concat "candidate" @ "stage" get-slot concat "localProjectionBase" !
								"local-projection:" "candidate" @ "stage" get-slot concat ":" concat "code" @ concat "localProjectionSemantic" !
								"experiment" @ "candidate" @ "stage" get-slot "projection" "localProjectionSemantic" @ "localProjectionBase" @ ri-artifact-name "localProjection" !
								"localProjection" @ unit-exists? not
								if
									"localProjection" @ "experiment" @ "projectionCategory" get-slot create-unit drop
									"localProjection" @ "experiment" @ "candidate" @ "stage" get-slot "projection" "localProjectionSemantic" @ ri-envelope drop
									"experiment" @ "localProjection" @ "experiment" set-slot
									"candidate" @ "stage" get-slot "localProjection" @ "stage" set-slot
									"localLibrary" @ "localProjection" @ "library" set-slot
									"candidate" @ "localProjection" @ "candidate" set-slot
									"code" @ "localProjection" @ "definitionCode" set-slot
									"signature" @ "localProjection" @ "materializedRelation" set-slot
									true "localProjection" @ "taskLocal" set-slot
								then
							then
							"experiment" @ "candidate" @ "stage" get-slot "local-invention" "localLibrary" @ 1 ri-record-action drop
						then
						"candidate" @ "stage" get-slot "stage1" =
						if
							"experiment" @ "reuseMode" get-slot "shared-library" =
							"experiment" @ "reuseMode" get-slot "shared-inlined" = or
							if
								"RI.Library." "experiment" @ concat "libraryBase" !
								"library:" "code" @ concat "librarySemantic" !
								"experiment" @ "stage1" "library" "librarySemantic" @ "libraryBase" @ ri-artifact-name "library" !
								"library" @ unit-exists? not
								if
									"library" @ "experiment" @ "libraryCategory" get-slot create-unit drop
									"library" @ "experiment" @ "stage1" "library" "librarySemantic" @ ri-envelope drop
									"experiment" @ "library" @ "experiment" set-slot
									"stage1" "library" @ "stage" set-slot
									"code" @ "library" @ "definitionCode" set-slot
									"signature" @ "library" @ "materializedRelation" set-slot
									"candidate" @ "library" @ "learnedFrom" set-slot
									true "library" @ "frozen" set-slot
								then
								"RI.Provenance." "experiment" @ concat "provenanceBase" !
								"provenance:" "code" @ concat "provenanceSemantic" !
								"experiment" @ "stage1" "provenance" "provenanceSemantic" @ "provenanceBase" @ ri-artifact-name "provenance" !
								"provenance" @ unit-exists? not
								if
									"provenance" @ "experiment" @ "provenanceCategory" get-slot create-unit drop
									"provenance" @ "experiment" @ "stage1" "provenance" "provenanceSemantic" @ ri-envelope drop
									"experiment" @ "provenance" @ "experiment" set-slot
									"library" @ "provenance" @ "library" set-slot
									"candidate" @ "provenance" @ "candidate" set-slot
									"code" @ "provenance" @ "definitionCode" set-slot
								then
								"library" @ "experiment" @ "libraryUnit" set-slot
								"provenance" @ "experiment" @ "provenanceUnit" set-slot
							then
							"code" @ "experiment" @ "frozenCode" set-slot
							"signature" @ "experiment" @ "frozenSignature" set-slot
							"experiment" @ "stage1" "promotion" "candidate" @ 1 ri-record-action drop
							"experiment" @ "stage1" "selection" "code" @ "candidate" @ ri-record-decision drop
							"experiment" @ "stage1" "decision" "experiment" @ "stage1SelectionUnit" get-slot 0 ri-record-action drop
							"awaiting-stage-2" "experiment" @ "terminal" set-slot
							"experiment" @ "stage1" "termination" "awaiting-stage-2" 1 ri-record-action drop
							"experiment" @ "stage1" "terminal" "awaiting-stage-2" "code" @ ri-record-decision drop
							"experiment" @ "stage1" "decision" "experiment" @ "stage1TerminalUnit" get-slot 0 ri-record-action drop
						else
							"code" @ "experiment" @ "selectedCode" set-slot
							"identified" "experiment" @ "terminal" set-slot
							"experiment" @ "stage2" "promotion" "candidate" @ 1 ri-record-action drop
							"experiment" @ "stage2" "selection" "code" @ "candidate" @ ri-record-decision drop
							"experiment" @ "stage2" "decision" "experiment" @ "stage2SelectionUnit" get-slot 0 ri-record-action drop
							"experiment" @ "stage2" "termination" "identified" 1 ri-record-action drop
							"experiment" @ "stage2" "terminal" "identified" "code" @ ri-record-decision drop
							"experiment" @ "stage2" "decision" "experiment" @ "stage2TerminalUnit" get-slot 0 ri-record-action drop
							"experiment" @ ri-experiment-complete? "experiment" @ "experimentComplete" set-slot
						then
					else
						"invalid" "experiment" @ "terminal" set-slot
					then
				else
					300 "candidate" @ "worth" set-slot
					"experiment" @ "reuseMode" get-slot "naive-direct" !=
					"experiment" @ "reuseMode" get-slot "uniform-random" != and
					if
						"falsePositive" @ 0 >
						if
							"RI.Constraint." "candidate" @ concat ".too-general" concat "constraintBase" !
							"constraint:too-general:" "code" @ concat "constraintSemantic" !
							"experiment" @ "candidate" @ "stage" get-slot "constraint" "constraintSemantic" @ "constraintBase" @ ri-artifact-name "constraint" !
							"constraint" @ unit-exists? not
							if
								"constraint" @ "experiment" @ "constraintCategory" get-slot create-unit drop
								"constraint" @ "experiment" @ "candidate" @ "stage" get-slot "constraint" "constraintSemantic" @ ri-envelope drop
								"experiment" @ "constraint" @ "experiment" set-slot
								"candidate" @ "stage" get-slot "constraint" @ "stage" set-slot
								"code" @ "constraint" @ "failedCode" set-slot
								"too-general" "constraint" @ "direction" set-slot
							then
							"experiment" @ "candidate" @ "stage" get-slot "constraint" "constraint" @ 1 ri-record-action drop
						then
						"falseNegative" @ 0 >
						if
							"RI.Constraint." "candidate" @ concat ".too-specific" concat "constraintBase" !
							"constraint:too-specific:" "code" @ concat "constraintSemantic" !
							"experiment" @ "candidate" @ "stage" get-slot "constraint" "constraintSemantic" @ "constraintBase" @ ri-artifact-name "constraint" !
							"constraint" @ unit-exists? not
							if
								"constraint" @ "experiment" @ "constraintCategory" get-slot create-unit drop
								"constraint" @ "experiment" @ "candidate" @ "stage" get-slot "constraint" "constraintSemantic" @ ri-envelope drop
								"experiment" @ "constraint" @ "experiment" set-slot
								"candidate" @ "stage" get-slot "constraint" @ "stage" set-slot
								"code" @ "constraint" @ "failedCode" set-slot
								"too-specific" "constraint" @ "direction" set-slot
							then
							"experiment" @ "candidate" @ "stage" get-slot "constraint" "constraint" @ 1 ri-record-action drop
						then
					then
					"candidate" @ "queueRank" get-slot 1 + "experiment" @ "queue" get-slot list-length =
					if
						"candidate" @ "stage" get-slot "stage2" =
						if
							"no-solution" "experiment" @ "terminal" set-slot
							"experiment" @ "stage2" "termination" "no-solution" 1 ri-record-action drop
							"experiment" @ "stage2" "terminal" "no-solution" "" ri-record-decision drop
							"experiment" @ "stage2" "decision" "experiment" @ "stage2TerminalUnit" get-slot 0 ri-record-action drop
							"experiment" @ ri-experiment-complete? "experiment" @ "experimentComplete" set-slot
						then
					then
				then
				then
			then
			"""#
	},
	{
		name: "H-RI-Continue"
		worth: 750
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "continue" ri-task-valid?
			"CurUnit" @ "terminal" get-slot "awaiting-stage-2" = and
			"CurUnit" @ "stage" get-slot "stage2" = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "libraryUnit" get-slot "library" !
			"library" @ "materializedRelation" get-slot "signature" !
			"RI.Projection." "experiment" @ concat "projectionBase" !
			"projection:" "experiment" @ "frozenCode" get-slot concat "projectionSemantic" !
			"experiment" @ "stage2" "candidate" "projectionSemantic" @ "projectionBase" @ ri-artifact-name "projection" !
			0 "projectionWork" !
				"experiment" @ "reuseMode" get-slot "shared-inlined" =
			if
				"experiment" @ "frozenCode" get-slot "experiment" @ "facts" get-slot ri-evaluation "evaluation" !
				"evaluation" @ ri-evaluation-signature "signature" !
				"evaluation" @ ri-evaluation-work "projectionWork" !
			then
			0 "support" ! 0 "failures" ! 0 "falsePositive" ! 0 "falseNegative" ! 0 "seen" !
			"experiment" @ "experimentProfileKey" get-slot nil = if "RuleInductionExample" examples else "experiment" @ "stage2CorpusUnit" get-slot "examples" get-slot then
			each
				it "example" !
				"example" @ "experiment" @ "stage2" ri-example-for?
				if
					"seen" @ 1 + "seen" !
					"signature" @ "example" @ ri-example-x "example" @ ri-example-y ri-signature-has? "actual" !
					"actual" @ "example" @ ri-example-positive? = "outcome" !
					"outcome" @
					if
						"support" @ 1 + "support" !
					else
						"failures" @ 1 + "failures" !
						"actual" @ if "falsePositive" @ 1 + "falsePositive" ! else "falseNegative" @ 1 + "falseNegative" ! then
					then
					"RI.Result." "projection" @ concat "." concat "example" @ concat "resultBase" !
					"result:" "projection" @ concat ":" concat "example" @ concat "resultSemantic" !
					"experiment" @ "stage2" "result" "resultSemantic" @ "resultBase" @ ri-artifact-name "result" !
					"result" @ unit-exists? not
					if
						"result" @ "experiment" @ "resultCategory" get-slot create-unit drop
						"result" @ "experiment" @ "stage2" "result" "resultSemantic" @ ri-envelope drop
						"experiment" @ "result" @ "experiment" set-slot
						"projection" @ "result" @ "candidate" set-slot
						"example" @ "result" @ "example" set-slot
						"actual" @ "result" @ "actual" set-slot
						"outcome" @ "result" @ "outcome" set-slot
					then
					"RI.Observation." "projection" @ concat "." concat "example" @ concat "observationBase" !
					"observation:" "projection" @ concat ":" concat "example" @ concat "observationSemantic" !
					"experiment" @ "stage2" "observation" "observationSemantic" @ "observationBase" @ ri-artifact-name "observation" !
					"observation" @ unit-exists? not
					if
						"observation" @ "experiment" @ "observationCategory" get-slot create-unit drop
						"observation" @ "experiment" @ "stage2" "observation" "observationSemantic" @ ri-envelope drop
						"experiment" @ "observation" @ "experiment" set-slot
						"projection" @ "observation" @ "candidate" set-slot
						"example" @ "observation" @ "example" set-slot
						"result" @ "observation" @ "result" set-slot
						"actual" @ "observation" @ "actual" set-slot
						"outcome" @ "observation" @ "outcome" set-slot
					then
				then
			end
			"projection" @ unit-exists? not
			if
				"projection" @ "experiment" @ "candidateCategory" get-slot create-unit drop
				"projection" @ "experiment" @ "stage2" "candidate" "projectionSemantic" @ ri-envelope drop
				"experiment" @ "projection" @ "experiment" set-slot
				"stage2" "projection" @ "stage" set-slot
				"experiment" @ "frozenCode" get-slot "projection" @ "definitionCode" set-slot
				"signature" @ "projection" @ "signature" set-slot
				"projectionWork" @ "projection" @ "fixedWork" set-slot
				"seen" @ "projection" @ "exampleCount" set-slot
				"support" @ "projection" @ "supportCount" set-slot
				"failures" @ "projection" @ "failureCount" set-slot
				"falsePositive" @ "projection" @ "falsePositiveCount" set-slot
				"falseNegative" @ "projection" @ "falseNegativeCount" set-slot
				true "projection" @ "projection" set-slot
				true "projection" @ "riEvaluated" set-slot
			then
			"projection" @ "experiment" @ "projectionUnit" set-slot
			"RI.Evidence." "projection" @ concat "evidenceBase" !
			"evidence:" "projection" @ concat "evidenceSemantic" !
			"experiment" @ "stage2" "evidence" "evidenceSemantic" @ "evidenceBase" @ ri-artifact-name "evidence" !
			"evidence" @ unit-exists? not
			if
				"evidence" @ "experiment" @ "evidenceCategory" get-slot create-unit drop
				"evidence" @ "experiment" @ "stage2" "evidence" "evidenceSemantic" @ ri-envelope drop
				"experiment" @ "evidence" @ "experiment" set-slot
				"projection" @ "evidence" @ "candidate" set-slot
				"seen" @ "evidence" @ "exampleCount" set-slot
				"support" @ "evidence" @ "supportCount" set-slot
				"failures" @ "evidence" @ "failureCount" set-slot
			then
			"experiment" @ "stage2" "evaluation" "projection" @ "projectionWork" @ "seen" @ + ri-record-action drop
			"failures" @ 0 = "seen" @ 4 >= and
			if
				"projection" @ ri-ready-to-select?
				if
					true "experiment" @ "usedFrozenLibrary" set-slot
					"experiment" @ "frozenCode" get-slot "experiment" @ "selectedCode" set-slot
					"identified" "experiment" @ "terminal" set-slot
					"experiment" @ "stage2" "promotion" "projection" @ 1 ri-record-action drop
					"experiment" @ "stage2" "selection" "experiment" @ "frozenCode" get-slot "projection" @ ri-record-decision drop
					"experiment" @ "stage2" "decision" "experiment" @ "stage2SelectionUnit" get-slot 0 ri-record-action drop
					"experiment" @ "stage2" "termination" "identified" 1 ri-record-action drop
					"experiment" @ "stage2" "terminal" "identified" "experiment" @ "frozenCode" get-slot ri-record-decision drop
					"experiment" @ "stage2" "decision" "experiment" @ "stage2TerminalUnit" get-slot 0 ri-record-action drop
					"experiment" @ ri-experiment-complete? "experiment" @ "experimentComplete" set-slot
				else
					"invalid" "experiment" @ "terminal" set-slot
				then
			else
				true "experiment" @ "fellBack" set-slot
				"experiment" @ "stage2" "fallback" "projection" @ 1 ri-record-action drop
				nil "experiment" @ "terminal" set-slot
				nil "experiment" @ "riStarted" set-slot
				"experiment" @ "refinementPriority" get-slot "experiment" @ "experiment" @ "startTaskSlot" get-slot "Start stage-two local fallback" add-task
			then
			"""#
	},
]
