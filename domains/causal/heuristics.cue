package domains

units: [
	{
		name: "H-Causal-V2-Propose"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "causalV2Propose" =
			"CurUnit" @ "CurSlot" @ causal-v2-task-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "CurSlot" @ causal-v2-begin-task drop
			"CurUnit" @ causal-v2-prepare-proposal drop
			"CurUnit" @ causal-v2-actions "actions" !
			"actions" @ each "CurUnit" @ it causal-v2-materialize-cache drop end
			"actions" @ each "CurUnit" @ it causal-v2-materialize-proposal drop end
			"actions" @ each "CurUnit" @ it causal-v2-materialize-partition drop end
			"actions" @ each "CurUnit" @ it causal-v2-materialize-score drop end
			"" "best" !
			"actions" @
			each
				it "action" !
				"best" @ "" =
				if
					"action" @ "best" !
				else
					"action" @ "best" @ "CurUnit" @ causal-v2-better?
					if "action" @ "best" ! then
				then
			end
			"actions" @
			each
				it "action" !
				"action" @ "best" @ "CurUnit" @ causal-v2-equal-score?
				if "CurUnit" @ "action" @ causal-v2-materialize-tie drop then
			end
			"CurUnit" @ "best" @ causal-v2-materialize-selection drop
			"CurUnit" @ "CurSlot" @ causal-v2-end-task drop
			"""#
	},
	{
		name: "H-Causal-V2-Authorize"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "causalV2Authorize" =
			"CurUnit" @ "CurSlot" @ causal-v2-task-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "CurSlot" @ causal-v2-begin-task drop
			"CurUnit" @ causal-v2-materialize-authorization drop
			"CurUnit" @ causal-v2-materialize-awaiting-snapshot drop
			"CurUnit" @ "CurSlot" @ causal-v2-end-task drop
			"""#
	},
	{
		name: "H-Causal-V2-Update"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "causalV2Update" =
			"CurUnit" @ "CurSlot" @ causal-v2-task-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "CurSlot" @ causal-v2-begin-task drop
			"CurUnit" @ causal-v2-prepare-update drop
			"CurUnit" @ causal-v2-eliminated
			each "CurUnit" @ it causal-v2-materialize-elimination drop end
			"CurUnit" @ causal-v2-materialize-posterior drop
			"CurUnit" @ causal-v2-materialize-consumption drop
			"CurUnit" @ causal-v2-materialize-transcript drop
			"CurUnit" @ causal-v2-materialize-next-snapshot drop
			"CurUnit" @ causal-v2-terminal?
			if "CurUnit" @ causal-v2-materialize-terminal drop then
			"CurUnit" @ causal-v2-finish-update drop
			"CurUnit" @ "CurSlot" @ causal-v2-end-task drop
			"""#
	},
	{
		name: "H-Causal-V2-Finalize"
		worth: 900
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "causalV2Finalize" =
			"CurUnit" @ "CurSlot" @ causal-v2-task-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "CurSlot" @ causal-v2-begin-task drop
			"CurUnit" @ causal-v2-finalize-zero drop
			"CurUnit" @ "CurSlot" @ causal-v2-end-task drop
			"""#
	},
	{
		name: "H-Causal-Propose"
		worth: 800
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "propose" causal-task-valid?
			"CurUnit" @ causal-profile-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			causal-actions "actions" !
			"" "best" !
			"actions" @
			each
				it "action" !
				"experiment" @ "proposal" "action" @ causal-artifact-name "proposal" !
				"proposal" @ unit-exists? not
				if
					"proposal" @ "CausalProposal" create-unit drop
					"experiment" @ "proposal" @ "experiment" set-slot
					"action" @ "proposal" @ "action" set-slot
				then
				"experiment" @ "partition" "action" @ causal-artifact-name "partition" !
				"partition" @ unit-exists? not
				if
					"partition" @ "CausalPartition" create-unit drop
					"experiment" @ "partition" @ "experiment" set-slot
					"action" @ "partition" @ "action" set-slot
					"action" @ "experiment" @ causal-partition-json "partition" @ "cells" set-slot
				then
				"experiment" @ "score" "action" @ causal-artifact-name "score" !
				"score" @ unit-exists? not
				if
					"score" @ "CausalScore" create-unit drop
					"experiment" @ "score" @ "experiment" set-slot
					"action" @ "score" @ "action" set-slot
					"action" @ "experiment" @ causal-feature-json "score" @ "features" set-slot
				then
				"best" @ "" =
				if
					"action" @ "best" !
				else
					"action" @ "best" @ "experiment" @ causal-better?
					if "action" @ "best" ! then
				then
			end
			0 list-of "ties" !
			"actions" @
			each
				it "action" !
				"action" @ "best" @ "experiment" @ causal-equal-score?
				if
					"ties" @ "action" @ list-append "ties" !
					"experiment" @ "tie" "action" @ causal-artifact-name "tie" !
					"tie" @ unit-exists? not
					if
						"tie" @ "CausalTie" create-unit drop
						"experiment" @ "tie" @ "experiment" set-slot
						"action" @ "tie" @ "action" set-slot
					then
				then
			end
			"experiment" @ "selection" "best" @ causal-artifact-name "selection" !
			"selection" @ unit-exists? not
			if
				"selection" @ "CausalSelection" create-unit drop
				"experiment" @ "selection" @ "experiment" set-slot
				"best" @ "selection" @ "action" set-slot
				"ties" @ "selection" @ "ties" set-slot
			then
			"best" @ "experiment" @ "selectedAction" set-slot
			"ties" @ "experiment" @ "selectedTies" set-slot
			"selection" @ "experiment" @ "selectionUnit" set-slot
			"awaiting-teacher" "experiment" @ "state" set-slot
			"""#
	},
	{
		name: "H-Causal-Update"
		worth: 800
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "CurSlot" @ "update" causal-task-valid?
			"CurUnit" @ causal-profile-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "responseUnit" get-slot causal-response-outcome "outcome" !
			"experiment" @ "selectedAction" get-slot "action" !
			"experiment" @ "posterior" get-slot "before" !
			"experiment" @ "actionCount" get-slot 1 + "count" !
			"experiment" @ "totalCost" get-slot "action" @ "experiment" @ causal-action-cost + "cost" !
			"experiment" @ "action" @ "outcome" @ causal-filter "after" !
			"before" @
			each
				it "hypothesis" !
				"after" @ "hypothesis" @ list-contains not
				if
					"experiment" @ "elimination" "hypothesis" @ causal-artifact-name "elimination" !
					"elimination" @ unit-exists? not
					if
						"elimination" @ "CausalElimination" create-unit drop
						"experiment" @ "elimination" @ "experiment" set-slot
						"hypothesis" @ "elimination" @ "hypothesis" set-slot
					then
				then
			end
			"after" @ "experiment" @ "posterior" set-slot
			"outcome" @ "experiment" @ "lastOutcome" set-slot
			"count" @ "experiment" @ "actionCount" set-slot
			"count" @ "experiment" @ "step" set-slot
			"cost" @ "experiment" @ "totalCost" set-slot
			"experiment" @ "consumedActions" get-slot "action" @ list-append "experiment" @ "consumedActions" set-slot
			"experiment" @ causal-transcript-digest "digest" !
			"digest" @ "experiment" @ "transcriptDigest" set-slot
			"experiment" @ "posterior" "digest" @ causal-artifact-name "posteriorUnit" !
			"posteriorUnit" @ unit-exists? not
			if
				"posteriorUnit" @ "CausalPosterior" create-unit drop
				"after" @ "posteriorUnit" @ "hypotheses" set-slot
				"experiment" @ "posteriorUnit" @ "experiment" set-slot
			then
			"after" @ "experiment" @ causal-terminal "terminal" !
			"terminal" @ "" =
			if
				"ready" "experiment" @ "state" set-slot
			else
				"experiment" @ "terminal" "terminal" @ causal-artifact-name "terminalUnit" !
				"terminalUnit" @ unit-exists? not
				if
					"terminalUnit" @ "CausalTerminal" create-unit drop
					"terminal" @ "terminalUnit" @ "terminal" set-slot
					"after" @ "terminalUnit" @ "posterior" set-slot
					"experiment" @ "terminalUnit" @ "experiment" set-slot
				then
				"terminal" @ "experiment" @ "terminal" set-slot
				"terminal" "experiment" @ "state" set-slot
			then
			"""#
	},
	{
		name: "H-Causal-TrainAcquisitionRules"
		worth: 850
		isA: ["Heuristic", "Anything"]
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "causalTrain" =
			"CurUnit" @ "CausalTraining" isa? and
			"CurUnit" @ "creditEnabled" get-slot true = and
			"CurUnit" @ "selectedRule" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "training" !
			"" "bestCode" !
			-1 "bestCredit" !
			1000000 "bestExhaustions" !
			"CausalAcquisitionRule" examples
			each
				it "rule" !
				"rule" @ "CausalAcquisitionRule" !=
				"rule" @ "training" get-slot "training" @ = and
				if
					0 "totalCredit" ! 0 "exhaustions" ! 0 "applications" !
					"CausalApplication" examples
					each
						it "application" !
						"application" @ "CausalApplication" !=
						"application" @ "training" get-slot "training" @ = and
						"application" @ "ruleCode" get-slot "rule" @ "ruleCode" get-slot = and
						if
							"applications" @ 1 + "applications" !
							1001 "application" @ "score" get-slot - "delta" !
							"totalCredit" @ "delta" @ + "totalCredit" !
							"application" @ "terminal" get-slot "budget-exhausted" =
							if "exhaustions" @ 1 + "exhaustions" ! then
							"Causal.Credit." "application" @ concat "credit" !
							"credit" @ unit-exists? not
							if
								"credit" @ "CausalCreditDelta" create-unit drop
								"training" @ "credit" @ "training" set-slot
								"rule" @ "credit" @ "rule" set-slot
								"application" @ "credit" @ "application" set-slot
								"delta" @ "credit" @ "delta" set-slot
							then
						then
					end
					"applications" @ 12 =
					if
						"Causal.Aggregate." "rule" @ concat "aggregate" !
						"aggregate" @ unit-exists? not
						if
							"aggregate" @ "CausalRuleAggregate" create-unit drop
							"training" @ "aggregate" @ "training" set-slot
							"rule" @ "aggregate" @ "rule" set-slot
							"rule" @ "ruleCode" get-slot "aggregate" @ "ruleCode" set-slot
							"totalCredit" @ "aggregate" @ "totalCredit" set-slot
							"exhaustions" @ "aggregate" @ "exhaustions" set-slot
						then
						"bestCode" @ "" =
						if
							"rule" @ "ruleCode" get-slot "bestCode" ! "totalCredit" @ "bestCredit" ! "exhaustions" @ "bestExhaustions" !
						else
							"totalCredit" @ "bestCredit" @ >
							"totalCredit" @ "bestCredit" @ = "exhaustions" @ "bestExhaustions" @ < and or
							"totalCredit" @ "bestCredit" @ = "exhaustions" @ "bestExhaustions" @ = and "rule" @ "ruleCode" get-slot "bestCode" @ causal-code-less? and or
							if "rule" @ "ruleCode" get-slot "bestCode" ! "totalCredit" @ "bestCredit" ! "exhaustions" @ "bestExhaustions" ! then
						then
					then
				then
			end
			0 list-of "winnerTies" !
			"CausalRuleAggregate" examples
			each
				it "aggregate" !
				"aggregate" @ "CausalRuleAggregate" !=
				"aggregate" @ "training" get-slot "training" @ = and
				"aggregate" @ "totalCredit" get-slot "bestCredit" @ = and
				"aggregate" @ "exhaustions" get-slot "bestExhaustions" @ = and
				if "winnerTies" @ "aggregate" @ "ruleCode" get-slot list-append "winnerTies" ! then
			end
			"bestCode" @ "training" @ "selectedRule" set-slot
			"winnerTies" @ "training" @ "winnerTies" set-slot
			"bestCredit" @ "training" @ "selectedCredit" set-slot
			"""#
	},
]
