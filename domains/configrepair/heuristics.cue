package domains

units: [
	{
		name:  "H-ComposeConfigurationRepairPlans"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Construct every valid unordered repair subset of size one through three"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "PrimitiveConfigurationRepair" isa?
			"ArgU" @ "PrimitiveConfigurationRepair" != and
			"ArgU" @ "repairKey" get-slot "ArgU" @ "repairValue" get-slot config-repair-valid? and
			"ArgU" @ "defn" get-slot nil != and
			"ArgU" @ "configPlansGenerated" get-slot nil = and
			"""#
		thenCompute: #"""
			"ArgU" @ "first" !
			0 list-of "plans" !
			"first" @ 1 list-of "plans" @ swap list-append "plans" !
			"PrimitiveConfigurationRepair" examples
			each
				it "second" !
				"second" @ "PrimitiveConfigurationRepair" !=
				"second" @ "repairKey" get-slot "second" @ "repairValue" get-slot config-repair-valid? and
				"second" @ "defn" get-slot nil != and
				"first" @ "second" @ config-name-less? and
				if
					"first" @ "second" @ 2 list-of "plans" @ swap list-append "plans" !
					"PrimitiveConfigurationRepair" examples
					each
						it "third" !
						"third" @ "PrimitiveConfigurationRepair" !=
						"third" @ "repairKey" get-slot "third" @ "repairValue" get-slot config-repair-valid? and
						"third" @ "defn" get-slot nil != and
						"second" @ "third" @ config-name-less? and
						if
							"first" @ "second" @ "third" @ 3 list-of "plans" @ swap list-append "plans" !
						then
					end
				then
			end

			"plans" @
			each
				it "components" !
				"components" @ config-plan-valid?
				if
					"components" @ config-plan-name "program" !
					"program" @ "CompositeConfigurationRepair" create-unit drop
					"CompositeConfigurationRepair" "UnaryOp" "Op" "Anything" 4 list-of "program" @ "isA" set-slot
					"Configuration" 1 list-of "program" @ "domain" set-slot
					"Configuration" 1 list-of "program" @ "range" set-slot
					1 "program" @ "arity" set-slot
					"components" @ "program" @ "components" set-slot
					"components" @ config-plan-defn "program" @ "defn" set-slot
					"repair-subsets-up-to-3/v1" "program" @ "synthesisMethod" set-slot
					"H-ComposeConfigurationRepairPlans" 1 list-of "components" @ list-concat "program" @ "creditors" set-slot
					"configuration/repair-subsets-up-to-3/v1" "program" @ "creditContext" set-slot
					"components" @ config-decision-key "program" @ "creditDecision" set-slot
					"synthesis" 1 list-of "roles" !
					"components" @
					each
						"roles" @ "repair" list-append "roles" !
					end
					"roles" @ "program" @ "creditRoles" set-slot
					700 "program" @ "configurationRepairEvaluation" "Evaluate synthesized configuration repair plan" add-task
				then
			end
			true "first" @ "configPlansGenerated" set-slot
			"""#
	},
	{
		name:  "H-EvaluateConfigurationRepairPlans"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Evaluate repair plans for schema satisfaction, protected intent, and idempotence"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "configurationRepairEvaluation" =
			"CurUnit" @ "CompositeConfigurationRepair" isa? and
			"CurUnit" @ "evaluatedConfigurationRepairCorpus" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "program" !
			0 "corpusSize" !
			0 "evaluated" !
			0 "support" !
			0 "constraintFailures" !
			0 "intentFailures" !
			0 "invalidApplications" !
			0 "idempotenceFailures" !
			0 "changedKeyTotal" !
			0 list-of "examplesSeen" !
			0 list-of "results" !
			0 list-of "observations" !
			"ConfigurationRepairExample" examples
			each
				it "example" !
				"example" @ "ConfigurationRepairExample" !=
				if
					"corpusSize" @ 1 + "corpusSize" !
					"evaluated" @ 1 + "evaluated" !
					"examplesSeen" @ "example" @ list-append "examplesSeen" !
					"example" @ "configuration" get-slot "input" !
					"example" @ "schema" get-slot "schemaName" !
					"schemaName" @ "data" get-slot "schema" !
					nil "actual" !
					nil "secondApplication" !
					false "applicationValid" !
					false "schemaSatisfied" !
					false "intentPreserved" !
					false "idempotent" !
					false "outcome" !
					0 "changedCount" !
					"" "resultName" !
					"invalid-application" "status" !

					"input" @ config-valid?
					"schema" @ config-schema-valid? and
					if
						"input" @ "program" @ apply-op "actual" !
						"actual" @ nil !=
						if
							true "applicationValid" !
							"Result" "program" @ "example" @ config-artifact-name "resultName" !
							"resultName" @ "ConfigurationRepairResult" create-unit drop
							"actual" @ "resultName" @ "data" set-slot
							"program" @ "resultName" @ "program" set-slot
							"example" @ "resultName" @ "example" set-slot
							"schemaName" @ "resultName" @ "schema" set-slot
							"H-EvaluateConfigurationRepairPlans" "resultName" @ "creditors" set-slot
							"results" @ "resultName" @ list-append "results" !
							"program" @ "example" @ 1 list-of "resultName" @ record-applic

							"actual" @ "schema" @ config-satisfies? "schemaSatisfied" !
							"input" @ "actual" @ "schema" @ config-preserves-protected? "intentPreserved" !
							"input" @ "actual" @ config-changed-count "changedCount" !
							"changedKeyTotal" @ "changedCount" @ + "changedKeyTotal" !
							"actual" @ "program" @ apply-op "secondApplication" !
							"secondApplication" @ nil !=
							if
								"actual" @ "secondApplication" @ = "idempotent" !
							then
						then
					then

					"applicationValid" @ "schemaSatisfied" @ and
					"intentPreserved" @ and
					"idempotent" @ and "outcome" !
					"applicationValid" @ not
					if
						"invalidApplications" @ 1 + "invalidApplications" !
					else
						"idempotent" @ not
						if
							"idempotenceFailures" @ 1 + "idempotenceFailures" !
						then
						"schemaSatisfied" @ not
						if
							"constraintFailures" @ 1 + "constraintFailures" !
						then
						"intentPreserved" @ not
						if
							"intentFailures" @ 1 + "intentFailures" !
						then
					then
					"outcome" @
					if
						"support" @ 1 + "support" !
					then

					"applicationValid" @ not
					if
						"invalid-application" "status" !
					else
						"idempotent" @ not
						if
							"non-idempotent" "status" !
						else
							"schemaSatisfied" @ not "intentPreserved" @ not and
							if
								"constraint-and-intent-failure" "status" !
							else
								"schemaSatisfied" @ not
								if
									"constraint-failure" "status" !
								else
									"intentPreserved" @ not
									if
										"intent-failure" "status" !
									else
										"success" "status" !
									then
								then
							then
						then
					then

					"Observation" "program" @ "example" @ config-artifact-name "observationName" !
					"observationName" @ "ConfigurationRepairObservation" create-unit drop
					"program" @ "observationName" @ "program" set-slot
					"example" @ "observationName" @ "example" set-slot
					"schemaName" @ "observationName" @ "schema" set-slot
					"input" @ "observationName" @ "input" set-slot
					"actual" @ "observationName" @ "actual" set-slot
					"applicationValid" @ "observationName" @ "applicationValid" set-slot
					"schemaSatisfied" @ "observationName" @ "schemaSatisfied" set-slot
					"intentPreserved" @ "observationName" @ "intentPreserved" set-slot
					"idempotent" @ "observationName" @ "idempotent" set-slot
					"outcome" @ "observationName" @ "outcome" set-slot
					"changedCount" @ "observationName" @ "changedCount" set-slot
					"status" @ "observationName" @ "status" set-slot
					"resultName" @ "observationName" @ "resultUnit" set-slot
					"H-EvaluateConfigurationRepairPlans" "observationName" @ "creditors" set-slot
					"observations" @ "observationName" @ list-append "observations" !
				then
			end

			"Evidence" "program" @ "" config-artifact-name "evidenceName" !
			"evidenceName" @ "ConfigurationRepairEvidence" create-unit drop
			"program" @ "evidenceName" @ "program" set-slot
			"examplesSeen" @ "evidenceName" @ "trainingExamples" set-slot
			"results" @ "evidenceName" @ "resultUnits" set-slot
			"observations" @ "evidenceName" @ "observations" set-slot
			"corpusSize" @ "evidenceName" @ "corpusSize" set-slot
			"evaluated" @ "evidenceName" @ "evaluatedCount" set-slot
			"support" @ "evidenceName" @ "supportCount" set-slot
			"constraintFailures" @ "evidenceName" @ "constraintFailureCount" set-slot
			"intentFailures" @ "evidenceName" @ "intentFailureCount" set-slot
			"invalidApplications" @ "evidenceName" @ "invalidApplicationCount" set-slot
			"idempotenceFailures" @ "evidenceName" @ "idempotenceFailureCount" set-slot
			"changedKeyTotal" @ "evidenceName" @ "changedKeyTotal" set-slot
			"exhaustive-training-corpus/v1" "evidenceName" @ "comparisonMethod" set-slot
			"H-EvaluateConfigurationRepairPlans" "evidenceName" @ "creditors" set-slot

			"evidenceName" @ "program" @ "evidenceUnit" set-slot
			"corpusSize" @ "program" @ "corpusSize" set-slot
			"evaluated" @ "program" @ "evaluatedCount" set-slot
			"support" @ "program" @ "supportCount" set-slot
			"constraintFailures" @ "program" @ "constraintFailureCount" set-slot
			"intentFailures" @ "program" @ "intentFailureCount" set-slot
			"invalidApplications" @ "program" @ "invalidApplicationCount" set-slot
			"idempotenceFailures" @ "program" @ "idempotenceFailureCount" set-slot
			"changedKeyTotal" @ "program" @ "changedKeyTotal" set-slot

			"corpusSize" @ 4 >=
			"evaluated" @ "corpusSize" @ = and
			"support" @ "corpusSize" @ = and
			"constraintFailures" @ 0 = and
			"intentFailures" @ 0 = and
			"invalidApplications" @ 0 = and
			"idempotenceFailures" @ 0 = and
			if
				800 "program" @ "worth" set-slot
				"Schema" "program" @ "" config-artifact-name "schemaResultName" !
				"schemaResultName" @ "ConfigurationRepairSchema" create-unit drop
				"program" @ "schemaResultName" @ "program" set-slot
				"program" @ "components" get-slot "schemaResultName" @ "components" set-slot
				"evidenceName" @ "schemaResultName" @ "evidenceUnit" set-slot
				"support" @ "schemaResultName" @ "supportCount" set-slot
				800 "schemaResultName" @ "worth" set-slot
				800 "schemaResultName" @ "creationWorth" set-slot
				800 "schemaResultName" @ "lastRewardedWorth" set-slot
				"H-EvaluateConfigurationRepairPlans" "schemaResultName" @ "creditors" set-slot
				"ConfigurationRepairPlanSatisfiesCorpus"
				"program" @ 1 list-of
				"Synthesized configuration repair plan satisfies every schema while preserving protected intent"
				"H-EvaluateConfigurationRepairPlans" make-protoconjec "conjectureName" !
				"evidenceName" @ 1 list-of "conjectureName" @ "evidence" set-slot
				"exhaustive-training-corpus/v1" "conjectureName" @ "comparisonMethod" set-slot
			else
				300 "program" @ "worth" set-slot
			then
			true "program" @ "evaluatedConfigurationRepairCorpus" set-slot
			"""#
	},
]
