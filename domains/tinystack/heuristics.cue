package domains

units: [
	{
		name:  "H-EnumerateBoundedPrograms"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Enumerate every ordered primitive sequence permitted by a bounded synthesis descriptor"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "BoundedProgramSynthesisExperiment" isa?
			"CurUnit" @ "BoundedProgramSynthesisExperiment" != and
			"CurUnit" @ "synthesisTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ "generationComplete" get-slot nil = and
			"CurUnit" @ synth-experiment-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "primitiveCategory" get-slot "primitiveCategory" !
			"experiment" @ "candidateCategory" get-slot "candidateCategory" !
			"experiment" @ "valueCategory" get-slot "valueCategory" !
			"experiment" @ "evaluationTaskSlot" get-slot "evaluationTaskSlot" !
			"experiment" @ "evaluationPriority" get-slot "evaluationPriority" !
			"experiment" @ "maxLength" get-slot "maxLength" !
			0 list-of "sequences" !

			"primitiveCategory" @ examples
			each
				it "first" !
				"first" @ "primitiveCategory" @ !=
				if
					"first" @ 1 list-of "sequences" @ swap list-append "sequences" !
					"maxLength" @ 2 >=
					if
						"primitiveCategory" @ examples
						each
							it "second" !
							"second" @ "primitiveCategory" @ !=
							if
								"first" @ "second" @ 2 list-of "sequences" @ swap list-append "sequences" !
								"maxLength" @ 3 >=
								if
									"primitiveCategory" @ examples
									each
										it "third" !
										"third" @ "primitiveCategory" @ !=
										if
											"first" @ "second" @ "third" @ 3 list-of "sequences" @ swap list-append "sequences" !
										then
									end
								then
							then
						end
					then
				then
			end

			0 list-of "candidateUnits" !
			"sequences" @
			each
				it "components" !
				"experiment" @ "components" @ synth-sequence-valid?
				if
					"experiment" @ "components" @ synth-program-name "program" !
					"program" @ "candidateCategory" @ create-unit drop
					"candidateCategory" @ "SynthesizedProgram" "UnaryOp" "Op" "Anything" 5 list-of "program" @ "isA" set-slot
					"valueCategory" @ 1 list-of "program" @ "domain" set-slot
					"valueCategory" @ 1 list-of "program" @ "range" set-slot
					1 "program" @ "arity" set-slot
					"components" @ "program" @ "components" set-slot
					"experiment" @ "components" @ synth-semantic-sequence "program" @ "semanticSequence" set-slot
					"experiment" @ "components" @ synth-program-defn "program" @ "defn" set-slot
					"components" @ list-length "program" @ "programLength" set-slot
					"experiment" @ "program" @ "synthesisExperiment" set-slot
					"experiment" @ "synthesisMethod" get-slot "program" @ "synthesisMethod" set-slot
					"experiment" @ "creditContext" get-slot "program" @ "creditContext" set-slot
					"experiment" @ "components" @ synth-decision-key "program" @ "creditDecision" set-slot
					"H-EnumerateBoundedPrograms" 1 list-of "components" @ list-concat "program" @ "creditors" set-slot
					"experiment" @ "components" @ synth-credit-roles "program" @ "creditRoles" set-slot
					"evaluationPriority" @ "program" @ "evaluationTaskSlot" @ "Evaluate bounded synthesized program" add-task
					"candidateUnits" @ "program" @ list-append "candidateUnits" !
				then
			end
			"candidateUnits" @ "experiment" @ "candidateUnits" set-slot
			"candidateUnits" @ list-length "experiment" @ "expectedCandidateCount" set-slot
			0 "experiment" @ "evaluatedCandidateCount" set-slot
			true "experiment" @ "generationComplete" set-slot
			"""#
	},
	{
		name:  "H-EvaluateBoundedProgram"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Evaluate one synthesized program against every example and retain observations"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "synthesisExperiment" get-slot "experiment" !
			"experiment" @ nil !=
			"experiment" @ "BoundedProgramSynthesisExperiment" isa? and
			"experiment" @ synth-experiment-valid? and
			"experiment" @ "generationComplete" get-slot true = and
			"experiment" @ "evaluationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ "evaluatedProgramCorpus" get-slot nil = and
			"CurUnit" @ "experiment" @ "candidateCategory" get-slot isa? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "program" !
			"program" @ "synthesisExperiment" get-slot "experiment" !
			"experiment" @ "exampleCategory" get-slot "exampleCategory" !
			"experiment" @ "resultCategory" get-slot "resultCategory" !
			"experiment" @ "observationCategory" get-slot "observationCategory" !
			"experiment" @ "evidenceCategory" get-slot "evidenceCategory" !
			"experiment" @ "inputSlot" get-slot "inputSlot" !
			"experiment" @ "expectedSlot" get-slot "expectedSlot" !
			"experiment" @ "resultValueSlot" get-slot "resultValueSlot" !
			"experiment" @ "inputValidator" get-slot "inputValidator" !
			"experiment" @ "outputValidator" get-slot "outputValidator" !
			"experiment" @ "comparator" get-slot "comparator" !
			0 "corpusSize" ! 0 "evaluated" ! 0 "support" ! 0 "failures" ! 0 "invalid" !
			0 list-of "examplesSeen" ! 0 list-of "results" ! 0 list-of "observations" !

			"exampleCategory" @ examples
			each
				it "example" !
				"example" @ "exampleCategory" @ !=
				if
					"corpusSize" @ 1 + "corpusSize" !
					"evaluated" @ 1 + "evaluated" !
					"examplesSeen" @ "example" @ list-append "examplesSeen" !
					"example" @ "inputSlot" @ get-slot "input" !
					"example" @ "expectedSlot" @ get-slot "expected" !
					nil "actual" ! false "outcome" ! "" "resultName" ! "invalid-input" "status" !
					"input" @ "inputValidator" @ apply-op "inputValid" !
					"expected" @ "outputValidator" @ apply-op "expectedValid" !
					"inputValid" @ true = "expectedValid" @ true = and
					if
						"input" @ "program" @ apply-op "actual" !
						"actual" @ "outputValidator" @ apply-op "actualValid" !
						"actualValid" @ true =
						if
							"experiment" @ "Result" "program" @ "example" @ synth-artifact-name "resultName" !
							"resultName" @ "resultCategory" @ create-unit drop
							"actual" @ "resultName" @ "resultValueSlot" @ set-slot
							"experiment" @ "resultName" @ "synthesisExperiment" set-slot
							"program" @ "resultName" @ "program" set-slot
							"example" @ "resultName" @ "example" set-slot
							"H-EvaluateBoundedProgram" "resultName" @ "creditors" set-slot
							"results" @ "resultName" @ list-append "results" !
							"program" @ "example" @ 1 list-of "resultName" @ record-applic
							"actual" @ "expected" @ "comparator" @ apply-op "outcome" !
							"outcome" @ true =
							if
								"support" @ 1 + "support" ! "match" "status" !
							else
								"failures" @ 1 + "failures" ! "mismatch" "status" !
							then
						else
							"invalid" @ 1 + "invalid" ! "semantic-nil" "status" !
						then
					else
						"invalid" @ 1 + "invalid" !
						"inputValid" @ true = if "invalid-expected" "status" ! then
					then
					"experiment" @ "Observation" "program" @ "example" @ synth-artifact-name "observationName" !
					"observationName" @ "observationCategory" @ create-unit drop
					"experiment" @ "observationName" @ "synthesisExperiment" set-slot
					"program" @ "observationName" @ "program" set-slot
					"example" @ "observationName" @ "example" set-slot
					"input" @ "observationName" @ "input" set-slot
					"expected" @ "observationName" @ "expected" set-slot
					"actual" @ "observationName" @ "actual" set-slot
					"outcome" @ "observationName" @ "outcome" set-slot
					"status" @ "observationName" @ "status" set-slot
					"resultName" @ "observationName" @ "resultUnit" set-slot
					"H-EvaluateBoundedProgram" "observationName" @ "creditors" set-slot
					"observations" @ "observationName" @ list-append "observations" !
				then
			end

			"experiment" @ "Evidence" "program" @ "" synth-artifact-name "evidenceName" !
			"evidenceName" @ "evidenceCategory" @ create-unit drop
			"experiment" @ "evidenceName" @ "synthesisExperiment" set-slot
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
			"H-EvaluateBoundedProgram" "evidenceName" @ "creditors" set-slot
			"evidenceName" @ "program" @ "evidenceUnit" set-slot
			"corpusSize" @ "program" @ "corpusSize" set-slot
			"evaluated" @ "program" @ "evaluatedCount" set-slot
			"support" @ "program" @ "supportCount" set-slot
			"failures" @ "program" @ "failureCount" set-slot
			"invalid" @ "program" @ "invalidCount" set-slot
			"support" @ "corpusSize" @ = "failures" @ 0 = and "invalid" @ 0 = and "program" @ "exactProgram" set-slot
			"support" @ "corpusSize" @ = "failures" @ 0 = and "invalid" @ 0 = and
			if 500 "program" @ "worth" set-slot else 300 "program" @ "worth" set-slot then
			true "program" @ "evaluatedProgramCorpus" set-slot

			"experiment" @ "evaluatedCandidateCount" get-slot 1 + "newEvaluated" !
			"newEvaluated" @ "experiment" @ "evaluatedCandidateCount" set-slot
			"newEvaluated" @ "experiment" @ "expectedCandidateCount" get-slot =
			"experiment" @ "finalizationScheduled" get-slot nil = and
			if
				true "experiment" @ "finalizationScheduled" set-slot
				"experiment" @ "finalizationPriority" get-slot "experiment" @ "experiment" @ "finalizationTaskSlot" get-slot "Select shortest exact synthesized programs" add-task
			then
			"""#
	},
	{
		name:  "H-SelectShortestExactProgram"
		worth: 750
		isA: ["Heuristic", "Anything"]
		english: "Select and promote every shortest exact program after the evaluation barrier"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "BoundedProgramSynthesisExperiment" isa?
			"CurUnit" @ "BoundedProgramSynthesisExperiment" != and
			"CurUnit" @ "finalizationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ synth-ready-to-finalize? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			0 list-of "exactCandidates" !
			"experiment" @ "maxLength" get-slot 1 + "minimumLength" !
			"experiment" @ "candidateUnits" get-slot
			each
				it "program" !
				"program" @ "exactProgram" get-slot true =
				if
					"exactCandidates" @ "program" @ list-append "exactCandidates" !
					"program" @ "programLength" get-slot "minimumLength" @ <
					if "program" @ "programLength" get-slot "minimumLength" ! then
				then
			end
			0 list-of "selectedPrograms" !
			"exactCandidates" @
			each
				it "program" !
				"program" @ "programLength" get-slot "minimumLength" @ =
				if
					"selectedPrograms" @ "program" @ list-append "selectedPrograms" !
				then
			end
			"exactCandidates" @
			each
				it "program" !
				"program" @ "programLength" get-slot "minimumLength" @ >
				if "selectedPrograms" @ "program" @ "dominatedBy" set-slot then
			end
			"exactCandidates" @ "experiment" @ "exactCandidates" set-slot
			"selectedPrograms" @ "experiment" @ "selectedPrograms" set-slot
			"exactCandidates" @ list-length 0 >
			if "minimumLength" @ "experiment" @ "minimumLength" set-slot then
			"selectedPrograms" @ list-length 0 =
			if "no-solution" "selectionStatus" ! else
				"selectedPrograms" @ list-length 1 =
				if "selected" "selectionStatus" ! else "co-minimal" "selectionStatus" ! then
			then
			"selectionStatus" @ "experiment" @ "selectionStatus" set-slot

			"experiment" @ "Selection" "experiment" @ "" synth-artifact-name "selectionEvidence" !
			"selectionEvidence" @ "experiment" @ "selectionEvidenceCategory" get-slot create-unit drop
			"experiment" @ "selectionEvidence" @ "synthesisExperiment" set-slot
			"exactCandidates" @ "selectionEvidence" @ "exactCandidates" set-slot
			"selectedPrograms" @ "selectionEvidence" @ "selectedPrograms" set-slot
			"minimumLength" @ "selectionEvidence" @ "minimumLength" set-slot
			"selectionStatus" @ "selectionEvidence" @ "selectionStatus" set-slot
			"H-SelectShortestExactProgram" "selectionEvidence" @ "creditors" set-slot
			"selectionEvidence" @ "experiment" @ "selectionEvidenceUnit" set-slot

			"selectedPrograms" @
			each
				it "program" !
				800 "program" @ "worth" set-slot
				"experiment" @ "SelectedSchema" "program" @ "" synth-artifact-name "schemaName" !
				"schemaName" @ "experiment" @ "promotedSchemaCategory" get-slot create-unit drop
				"experiment" @ "schemaName" @ "synthesisExperiment" set-slot
				"program" @ "schemaName" @ "program" set-slot
				"program" @ "components" get-slot "schemaName" @ "components" set-slot
				"program" @ "evidenceUnit" get-slot "schemaName" @ "evidenceUnit" set-slot
				800 "schemaName" @ "worth" set-slot 800 "schemaName" @ "creationWorth" set-slot 800 "schemaName" @ "lastRewardedWorth" set-slot
				"H-SelectShortestExactProgram" "schemaName" @ "creditors" set-slot
				"BoundedProgramSatisfiesCorpus" "program" @ 1 list-of "Shortest synthesized program satisfies every training example" "H-SelectShortestExactProgram" make-protoconjec "conjectureName" !
				"program" @ "evidenceUnit" get-slot 1 list-of "conjectureName" @ "evidence" set-slot
				"experiment" @ "conjectureName" @ "synthesisExperiment" set-slot
			end
			true "experiment" @ "finalizationComplete" set-slot
			true "experiment" @ "simplificationScheduled" set-slot
			"experiment" @ "simplificationPriority" get-slot "experiment" @ "experiment" @ "simplificationTaskSlot" get-slot "Compare bounded simplification candidates" add-task
			"""#
	},
	{
		name:  "H-CompareProgramSimplifications"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Compare synthesized two-step programs with primitives on a bounded partial-function probe set"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurUnit" @ "BoundedProgramSynthesisExperiment" isa?
			"CurUnit" @ "BoundedProgramSynthesisExperiment" != and
			"CurUnit" @ "simplificationTaskSlot" get-slot "CurSlot" @ = and
			"CurUnit" @ "finalizationComplete" get-slot true = and
			"CurUnit" @ "simplificationComplete" get-slot nil = and
			"CurUnit" @ synth-experiment-valid? and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "probeCategory" get-slot "probeCategory" !
			"experiment" @ "probeInputSlot" get-slot "probeInputSlot" !
			"experiment" @ "primitiveCategory" get-slot "primitiveCategory" !
			"experiment" @ "outputValidator" get-slot "outputValidator" !
			"experiment" @ "comparator" get-slot "comparator" !
			0 list-of "longPrograms" ! 0 list-of "primitives" ! 0 list-of "subjects" !
			"experiment" @ "candidateUnits" get-slot
			each
				it "program" !
				"program" @ "programLength" get-slot "experiment" @ "simplificationProgramLength" get-slot =
				if "longPrograms" @ "program" @ list-append "longPrograms" ! then
			end
			"primitiveCategory" @ examples
			each
				it "primitive" !
				"primitive" @ "primitiveCategory" @ !=
				if "primitives" @ "primitive" @ list-append "primitives" ! then
			end
			"longPrograms" @ "primitives" @ list-concat "subjects" !

			0 "executionObservationCount" ! 0 "executionResultCount" ! 0 "undefinedExecutionCount" !
			"subjects" @
			each
				it "subject" !
				"probeCategory" @ examples
				each
					it "probe" !
					"probe" @ "probeCategory" @ !=
					if
						"probe" @ "probeInputSlot" @ get-slot "input" !
						"input" @ "subject" @ apply-op "actual" !
						"actual" @ "outputValidator" @ apply-op true = "defined" !
						"" "resultName" !
						"defined" @
						if
							"experiment" @ "SimplificationResult" "subject" @ "probe" @ synth-artifact-name "resultName" !
							"resultName" @ "experiment" @ "simplificationExecutionResultCategory" get-slot create-unit drop
							"experiment" @ "resultName" @ "synthesisExperiment" set-slot
							"subject" @ "resultName" @ "subjectProgram" set-slot
							"probe" @ "resultName" @ "probe" set-slot
							"actual" @ "resultName" @ "experiment" @ "resultValueSlot" get-slot set-slot
							"H-CompareProgramSimplifications" "resultName" @ "creditors" set-slot
							"subject" @ "probe" @ 1 list-of "resultName" @ record-applic
							"executionResultCount" @ 1 + "executionResultCount" !
						else
							"undefinedExecutionCount" @ 1 + "undefinedExecutionCount" !
						then
						"experiment" @ "SimplificationExecution" "subject" @ "probe" @ synth-artifact-name "observationName" !
						"observationName" @ "experiment" @ "simplificationExecutionObservationCategory" get-slot create-unit drop
						"experiment" @ "observationName" @ "synthesisExperiment" set-slot
						"subject" @ "observationName" @ "subjectProgram" set-slot
						"probe" @ "observationName" @ "probe" set-slot
						"input" @ "observationName" @ "input" set-slot
						"actual" @ "observationName" @ "actual" set-slot
						"defined" @ "observationName" @ "defined" set-slot
						"resultName" @ "observationName" @ "resultUnit" set-slot
						"H-CompareProgramSimplifications" "observationName" @ "creditors" set-slot
						"executionObservationCount" @ 1 + "executionObservationCount" !
					then
				end
			end

			0 "pairCount" ! 0 "comparisonObservationCount" ! 0 "equivalentPairCount" ! 0 list-of "simplificationSchemas" !
			"longPrograms" @
			each
				it "longProgram" !
				"primitives" @
				each
					it "primitive" !
					"pairCount" @ 1 + "pairCount" !
					"experiment" @ "SimplificationPair" "longProgram" @ "primitive" @ synth-artifact-name "pairName" !
					"pairName" @ "experiment" @ "simplificationPairCategory" get-slot create-unit drop
					"experiment" @ "pairName" @ "synthesisExperiment" set-slot
					"longProgram" @ "pairName" @ "longProgram" set-slot
					"primitive" @ "pairName" @ "shortProgram" set-slot
					0 "bothDefined" ! 0 "bothUndefined" ! 0 "oneUndefined" ! 0 "definedMismatches" ! 0 "mismatches" ! 0 "support" ! 0 list-of "comparisons" !
					"probeCategory" @ examples
					each
						it "probe" !
						"probe" @ "probeCategory" @ !=
						if
							"experiment" @ "longProgram" @ "probe" @ synth-find-execution-observation "longObs" !
							"experiment" @ "primitive" @ "probe" @ synth-find-execution-observation "shortObs" !
							"longObs" @ "defined" get-slot "longDefined" !
							"shortObs" @ "defined" get-slot "shortDefined" !
							false "agreement" ! "one-undefined" "status" !
							"longDefined" @ true = "shortDefined" @ true = and
							if
								"bothDefined" @ 1 + "bothDefined" !
								"longObs" @ "actual" get-slot "shortObs" @ "actual" get-slot "comparator" @ apply-op "agreement" !
								"agreement" @ true = if "both-defined-match" "status" ! else "definedMismatches" @ 1 + "definedMismatches" ! "both-defined-mismatch" "status" ! then
							else
								"longDefined" @ false = "shortDefined" @ false = and
								if
									"bothUndefined" @ 1 + "bothUndefined" ! true "agreement" ! "both-undefined" "status" !
								else
									"oneUndefined" @ 1 + "oneUndefined" !
								then
							then
							"agreement" @ true = if "support" @ 1 + "support" ! else "mismatches" @ 1 + "mismatches" ! then
							"primitive" @ "." concat "probe" @ concat "comparisonKey" !
							"experiment" @ "SimplificationComparison" "longProgram" @ "comparisonKey" @ synth-artifact-name "comparisonName" !
							"comparisonName" @ "experiment" @ "simplificationComparisonObservationCategory" get-slot create-unit drop
							"experiment" @ "comparisonName" @ "synthesisExperiment" set-slot
							"pairName" @ "comparisonName" @ "pair" set-slot
							"probe" @ "comparisonName" @ "probe" set-slot
							"longObs" @ "comparisonName" @ "longExecutionObservation" set-slot
							"shortObs" @ "comparisonName" @ "shortExecutionObservation" set-slot
							"agreement" @ "comparisonName" @ "outcome" set-slot
							"status" @ "comparisonName" @ "status" set-slot
							"H-CompareProgramSimplifications" "comparisonName" @ "creditors" set-slot
							"comparisons" @ "comparisonName" @ list-append "comparisons" !
							"comparisonObservationCount" @ 1 + "comparisonObservationCount" !
						then
					end
					"mismatches" @ 0 = "oneUndefined" @ 0 = and "bothDefined" @ 3 >= and "equivalent" !
					"experiment" @ "SimplificationEvidence" "longProgram" @ "primitive" @ synth-artifact-name "evidenceName" !
					"evidenceName" @ "experiment" @ "simplificationEvidenceCategory" get-slot create-unit drop
					"experiment" @ "evidenceName" @ "synthesisExperiment" set-slot
					"pairName" @ "evidenceName" @ "pair" set-slot
					"longProgram" @ "evidenceName" @ "longProgram" set-slot
					"primitive" @ "evidenceName" @ "shortProgram" set-slot
					"comparisons" @ "evidenceName" @ "comparisonObservations" set-slot
					"bothDefined" @ "evidenceName" @ "bothDefinedCount" set-slot
					"bothUndefined" @ "evidenceName" @ "bothUndefinedCount" set-slot
					"oneUndefined" @ "evidenceName" @ "oneUndefinedCount" set-slot
					"definedMismatches" @ "evidenceName" @ "definedMismatchCount" set-slot
					"mismatches" @ "evidenceName" @ "mismatchCount" set-slot
					"support" @ "evidenceName" @ "supportCount" set-slot
					"equivalent" @ "evidenceName" @ "equivalent" set-slot
					"experiment" @ "probeSetVersion" get-slot "evidenceName" @ "comparisonMethod" set-slot
					"H-CompareProgramSimplifications" "evidenceName" @ "creditors" set-slot
					"evidenceName" @ "pairName" @ "evidenceUnit" set-slot
					"equivalent" @ "pairName" @ "equivalent" set-slot
					"equivalent" @
					if
						"equivalentPairCount" @ 1 + "equivalentPairCount" !
						"experiment" @ "SimplificationSchema" "longProgram" @ "primitive" @ synth-artifact-name "schemaName" !
						"schemaName" @ "experiment" @ "simplificationSchemaCategory" get-slot create-unit drop
						"experiment" @ "schemaName" @ "synthesisExperiment" set-slot
						"longProgram" @ "schemaName" @ "longProgram" set-slot
						"primitive" @ "schemaName" @ "shortProgram" set-slot
						"evidenceName" @ "schemaName" @ "evidenceUnit" set-slot
						800 "schemaName" @ "worth" set-slot 800 "schemaName" @ "creationWorth" set-slot 800 "schemaName" @ "lastRewardedWorth" set-slot
						"H-CompareProgramSimplifications" "schemaName" @ "creditors" set-slot
						"simplificationSchemas" @ "schemaName" @ list-append "simplificationSchemas" !
						"StackProgramSimplifiesToPrimitive" "longProgram" @ "primitive" @ 2 list-of "Two-step stack program is observationally equivalent to one primitive on the bounded probe set" "H-CompareProgramSimplifications" make-protoconjec "conjectureName" !
						"evidenceName" @ 1 list-of "conjectureName" @ "evidence" set-slot
						"experiment" @ "conjectureName" @ "synthesisExperiment" set-slot
					then
				end
			end
			"executionObservationCount" @ "experiment" @ "simplificationExecutionObservationCount" set-slot
			"executionResultCount" @ "experiment" @ "simplificationExecutionResultCount" set-slot
			"undefinedExecutionCount" @ "experiment" @ "simplificationUndefinedExecutionCount" set-slot
			"pairCount" @ "experiment" @ "simplificationPairCount" set-slot
			"comparisonObservationCount" @ "experiment" @ "simplificationComparisonObservationCount" set-slot
			"equivalentPairCount" @ "experiment" @ "simplificationEquivalentPairCount" set-slot
			"simplificationSchemas" @ "experiment" @ "simplificationSchemas" set-slot
			true "experiment" @ "simplificationComplete" set-slot
			"""#
	},
]
