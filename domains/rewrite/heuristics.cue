package domains

units: [
	{
		name: "H-ComposeRewritePrograms"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Construct every ordered pair of distinct primitive rewrite operations"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "PrimitiveRewriteOp" isa?
			"ArgU" @ "PrimitiveRewriteOp" != and
			"ArgU" @ "defn" get-slot nil != and
			"ArgU" @ "compositionsGenerated" get-slot nil = and
			"""#
		thenCompute: #"""
			"ArgU" @ "first" !
			"PrimitiveRewriteOp" examples
			each
				it "second" !
				"second" @ "PrimitiveRewriteOp" !=
				"second" @ "first" @ != and
				"second" @ "defn" get-slot nil != and
				if
					"first" @ "second" @ rewrite-compose-name "program" !
					"program" @ "CompositeRewriteOp" create-unit drop
					"CompositeRewriteOp" "UnaryOp" "Op" "Anything" 4 list-of "program" @ "isA" set-slot
					"RewriteString" 1 list-of "program" @ "domain" set-slot
					"RewriteString" 1 list-of "program" @ "range" set-slot
					1 "program" @ "arity" set-slot
					"first" @ "second" @ 2 list-of "program" @ "components" set-slot
					"first" @ "defn" get-slot " " concat "second" @ "defn" get-slot concat "program" @ "defn" set-slot
					"ordered-distinct-pairs/v1" "program" @ "synthesisMethod" set-slot
					"H-ComposeRewritePrograms" "first" @ "second" @ 3 list-of "program" @ "creditors" set-slot
					700 "program" @ "rewriteEvaluation" "Evaluate synthesized rewrite program" add-task
				then
			end
			true "first" @ "compositionsGenerated" set-slot
			"""#
	},
	{
		name: "H-EvaluateRewritePrograms"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Evaluate synthesized programs against the complete rewrite corpus"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "rewriteEvaluation" =
			"CurUnit" @ "CompositeRewriteOp" isa? and
			"CurUnit" @ "evaluatedRewriteCorpus" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "program" !
			0 "corpusSize" !
			0 "evaluated" !
			0 "support" !
			0 "failures" !
			0 "invalid" !
			0 list-of "examplesSeen" !
			0 list-of "results" !
			0 list-of "observations" !
			"RewriteTrainingExample" examples
			each
				it "example" !
				"example" @ "RewriteTrainingExample" !=
				if
					"corpusSize" @ 1 + "corpusSize" !
					"evaluated" @ 1 + "evaluated" !
					"examplesSeen" @ "example" @ list-append "examplesSeen" !
					"example" @ "input" get-slot "input" !
					"example" @ "expected" get-slot "expected" !
					nil "actual" !
					false "outcome" !
					"" "resultName" !
					"invalid-input" "status" !
					"input" @ rewrite-valid? "inputValid" !
					"expected" @ rewrite-valid? "expectedValid" !
					"inputValid" @ "expectedValid" @ and
					if
						"input" @ "program" @ apply-op "actual" !
						"actual" @ nil !=
						if
							"Result" "program" @ "example" @ rewrite-artifact-name "resultName" !
							"resultName" @ "RewriteStringResult" create-unit drop
							"actual" @ "resultName" @ "data" set-slot
							"program" @ "resultName" @ "program" set-slot
							"example" @ "resultName" @ "example" set-slot
							"H-EvaluateRewritePrograms" "resultName" @ "creditors" set-slot
							"results" @ "resultName" @ list-append "results" !
							"program" @ "example" @ 1 list-of "resultName" @ record-applic
							"actual" @ "expected" @ = "outcome" !
							"outcome" @
							if
								"match" "status" !
								"support" @ 1 + "support" !
							else
								"mismatch" "status" !
								"failures" @ 1 + "failures" !
							then
						else
							"semantic-nil" "status" !
							"invalid" @ 1 + "invalid" !
						then
					else
						"inputValid" @
						if
							"invalid-expected" "status" !
						else
							"expectedValid" @
							if
								"invalid-input" "status" !
							else
								"invalid-input-and-expected" "status" !
							then
						then
						"invalid" @ 1 + "invalid" !
					then
					"Observation" "program" @ "example" @ rewrite-artifact-name "observationName" !
					"observationName" @ "RewriteObservation" create-unit drop
					"program" @ "observationName" @ "program" set-slot
					"example" @ "observationName" @ "example" set-slot
					"input" @ "observationName" @ "input" set-slot
					"expected" @ "observationName" @ "expected" set-slot
					"actual" @ "observationName" @ "actual" set-slot
					"outcome" @ "observationName" @ "outcome" set-slot
					"status" @ "observationName" @ "status" set-slot
					"resultName" @ "observationName" @ "resultUnit" set-slot
					"H-EvaluateRewritePrograms" "observationName" @ "creditors" set-slot
					"observations" @ "observationName" @ list-append "observations" !
				then
			end

			"Evidence" "program" @ "" rewrite-artifact-name "evidenceName" !
			"evidenceName" @ "RewriteProgramEvidence" create-unit drop
			"program" @ "evidenceName" @ "program" set-slot
			"examplesSeen" @ "evidenceName" @ "trainingExamples" set-slot
			"results" @ "evidenceName" @ "resultUnits" set-slot
			"observations" @ "evidenceName" @ "observations" set-slot
			"corpusSize" @ "evidenceName" @ "corpusSize" set-slot
			"evaluated" @ "evidenceName" @ "evaluatedCount" set-slot
			"support" @ "evidenceName" @ "supportCount" set-slot
			"failures" @ "evidenceName" @ "failureCount" set-slot
			"invalid" @ "evidenceName" @ "invalidCount" set-slot
			"exhaustive-training-corpus/v1" "evidenceName" @ "comparisonMethod" set-slot
			"H-EvaluateRewritePrograms" "evidenceName" @ "creditors" set-slot

			"evidenceName" @ "program" @ "evidenceUnit" set-slot
			"corpusSize" @ "program" @ "corpusSize" set-slot
			"evaluated" @ "program" @ "evaluatedCount" set-slot
			"support" @ "program" @ "supportCount" set-slot
			"failures" @ "program" @ "failureCount" set-slot
			"invalid" @ "program" @ "invalidCount" set-slot

			"corpusSize" @ 3 >=
			"evaluated" @ "corpusSize" @ = and
			"support" @ "corpusSize" @ = and
			"failures" @ 0 = and
			"invalid" @ 0 = and
			if
				800 "program" @ "worth" set-slot
				"Schema" "program" @ "" rewrite-artifact-name "schemaName" !
				"schemaName" @ "RewriteProgramSchema" create-unit drop
				"program" @ "schemaName" @ "program" set-slot
				"program" @ "components" get-slot "schemaName" @ "components" set-slot
				"evidenceName" @ "schemaName" @ "evidenceUnit" set-slot
				"support" @ "schemaName" @ "supportCount" set-slot
				800 "schemaName" @ "worth" set-slot
				800 "schemaName" @ "creationWorth" set-slot
				800 "schemaName" @ "lastRewardedWorth" set-slot
				"H-EvaluateRewritePrograms" "schemaName" @ "creditors" set-slot
				"RewriteProgramSatisfiesCorpus"
				"program" @ 1 list-of
				"Synthesized rewrite program satisfies every training example"
				"H-EvaluateRewritePrograms" make-protoconjec "conjectureName" !
				"evidenceName" @ 1 list-of "conjectureName" @ "evidence" set-slot
				"exhaustive-training-corpus/v1" "conjectureName" @ "comparisonMethod" set-slot
			else
				300 "program" @ "worth" set-slot
			then
			true "program" @ "evaluatedRewriteCorpus" set-slot
			"""#
	},
]

