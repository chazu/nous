units: [
	{
		name: "H-ProtocolControlAnalysis"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Integration control: report rejecting traps in training protocols"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "protocolAnalysis" =
			"CurUnit" @ "ProtocolTrainingExample" isa? and
			"CurUnit" @ "protocolControlAnalyzed" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "data" get-slot "RejectingTrapProtocolStates" apply-op "traps" !
			"traps" @ nil !=
			"traps" @ list-length 0 > and
			if
				"Evidence-RejectingTraps-" "CurUnit" @ concat "evidenceName" !
				"evidenceName" @ unit-exists? not
				if
					"evidenceName" @ "ProtocolEvidence" create-unit drop
					"CurUnit" @ "evidenceName" @ "sourceProtocol" set-slot
					"traps" @ "evidenceName" @ "trapStates" set-slot
					"RejectingTrapProtocolStates/v1" "evidenceName" @ "comparisonMethod" set-slot
					"H-ProtocolControlAnalysis" "evidenceName" @ "creditors" set-slot
					"ProtocolRejectingTrap"
					"CurUnit" @ 1 list-of
					"CurUnit" @ " contains reachable states from which acceptance is impossible" concat
					"H-ProtocolControlAnalysis" make-protoconjec "conjectureName" !
					"evidenceName" @ 1 list-of "conjectureName" @ "evidence" set-slot
					"RejectingTrapProtocolStates/v1" "conjectureName" @ "comparisonMethod" set-slot
				then
			then
			true "CurUnit" @ "protocolControlAnalyzed" set-slot
			"""#
	},
	{
		name: "H-DiscoverProtocolRelations"
		worth: 800
		isA: ["Heuristic", "Anything"]
		english: "Test every protocol transform against every protocol relation over all training machines"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "ProtocolTransform" isa?
			"ArgU" @ "ProtocolTransform" != and
			"ArgU" @ "defn" get-slot nil != and
			"ArgU" @ "protocolRelationsExplored" get-slot nil = and
			"""#
		thenCompute: #"""
			"ArgU" @ "transform" !
			"ProtocolRelation" examples
			each
				it "relation" !
				"relation" @ "ProtocolRelation" !=
				"relation" @ "defn" get-slot nil != and
				if
					0 "support" !
					0 "failures" !
					0 list-of "subjects" !
					0 list-of "results" !
					0 list-of "observations" !
					"ProtocolTrainingExample" examples
					each
						it "subject" !
						"subject" @ "ProtocolTrainingExample" !=
						"subject" @ "data" get-slot nil != and
						if
							"subject" @ "data" get-slot "transform" @ apply-op "transformed" !
							"transformed" @ nil !=
							if
								"subjects" @ "subject" @ list-append "subjects" !
								"Result-" "transform" @ concat "-" concat "relation" @ concat "-" concat "subject" @ concat "resultName" !
								"resultName" @ unit-exists? not
								if
									"resultName" @ "Protocol" create-unit drop
									"transformed" @ "resultName" @ "data" set-slot
									"subject" @ "resultName" @ "derivedFrom" set-slot
									"transform" @ "resultName" @ "derivedBy" set-slot
									"H-DiscoverProtocolRelations" "resultName" @ "creditors" set-slot
								then
								"results" @ "resultName" @ list-append "results" !
								"transform" @ "subject" @ 1 list-of "resultName" @ record-applic
								"subject" @ "data" get-slot "transformed" @ "relation" @ apply-op "outcome" !
								"Observation-" "transform" @ concat "-" concat "relation" @ concat "-" concat "subject" @ concat "observationName" !
								"observationName" @ unit-exists? not
								if
									"observationName" @ "ProtocolRelationObservation" create-unit drop
									"subject" @ "observationName" @ "subjectProtocol" set-slot
									"resultName" @ "observationName" @ "resultProtocol" set-slot
									"transform" @ "observationName" @ "transform" set-slot
									"relation" @ "observationName" @ "relation" set-slot
									"outcome" @ "observationName" @ "outcome" set-slot
									"H-DiscoverProtocolRelations" "observationName" @ "creditors" set-slot
								then
								"observations" @ "observationName" @ list-append "observations" !
								"outcome" @ nil !=
								"outcome" @ and
								if
									"support" @ 1 + "support" !
								else
									"failures" @ 1 + "failures" !
								then
							then
						then
					end

					"Evidence-" "transform" @ concat "-" concat "relation" @ concat "evidenceName" !
					"evidenceName" @ unit-exists? not
					if
						"evidenceName" @ "ProtocolEvidence" create-unit drop
						"transform" @ "evidenceName" @ "transform" set-slot
						"relation" @ "evidenceName" @ "relation" set-slot
						"subjects" @ "evidenceName" @ "trainingSubjects" set-slot
						"results" @ "evidenceName" @ "resultUnits" set-slot
						"observations" @ "evidenceName" @ "relationObservations" set-slot
						"support" @ "evidenceName" @ "supportCount" set-slot
						"failures" @ "evidenceName" @ "failureCount" set-slot
						"exhaustive-transform-relation/v1" "evidenceName" @ "comparisonMethod" set-slot
						"H-DiscoverProtocolRelations" "evidenceName" @ "creditors" set-slot
					then

					"Candidate-" "transform" @ concat "-" concat "relation" @ concat "candidateName" !
					"candidateName" @ unit-exists? not
					if
						"candidateName" @ "ProtocolRelationCandidate" create-unit drop
						"transform" @ "candidateName" @ "transform" set-slot
						"relation" @ "candidateName" @ "relation" set-slot
						"evidenceName" @ "candidateName" @ "evidenceUnit" set-slot
						"support" @ "candidateName" @ "supportCount" set-slot
						"failures" @ "candidateName" @ "failureCount" set-slot
						500 "support" @ 50 * + "failures" @ 100 * - "candidateName" @ "worth" set-slot
						"H-DiscoverProtocolRelations" "candidateName" @ "creditors" set-slot
					then

					"support" @ 3 >= "failures" @ 0 = and
					if
						"Schema-" "transform" @ concat "-" concat "relation" @ concat "schemaName" !
						"schemaName" @ unit-exists? not
						if
							"schemaName" @ "ProtocolRelationSchema" create-unit drop
							"transform" @ "schemaName" @ "transform" set-slot
							"relation" @ "schemaName" @ "relation" set-slot
							"evidenceName" @ "schemaName" @ "evidenceUnit" set-slot
							"support" @ "schemaName" @ "supportCount" set-slot
							800 "schemaName" @ "worth" set-slot
							"H-DiscoverProtocolRelations" "schemaName" @ "creditors" set-slot
							"ProtocolTransformRelation"
							"transform" @ "relation" @ 2 list-of
							"relation" @ " holds between each training protocol and the output of " concat "transform" @ concat
							"H-DiscoverProtocolRelations" make-protoconjec "conjectureName" !
							"evidenceName" @ 1 list-of "conjectureName" @ "evidence" set-slot
							"exhaustive-transform-relation/v1" "conjectureName" @ "comparisonMethod" set-slot
						then
					then
				then
			end
			true "transform" @ "protocolRelationsExplored" set-slot
			"""#
	},
]
