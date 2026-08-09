package domains

units: [
	{
		name: "NG-H-ConsiderPrune"
		worth: 800
		isA: ["Heuristic", "Anything"]
		english: "Populate and test blocked-pair schemas, then consider occurrence-specific prune requests"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "ngStart" =
			"CurSlot" @ "ngRefine" = or
			"CurSlot" @ "ngEvaluate" = or
			"CurSlot" @ "ngSelect" = or
			"CurSlot" @ "ngPromote" = or
			"CurSlot" @ "ngConsiderPrune" = or
			"CurUnit" @ unit-exists? and
			"""#
		thenCompute: #"""
			"CurSlot" @ "ngStart" =
			if
				"CurUnit" @ "experiment" !
				"experiment" @ "ngStarted" get-slot nil =
				if
					"candidate" "mask:0" ng-artifact-name "root" !
					"root" @ "NogoodCandidate" create-unit drop
					0 "root" @ "mask" set-slot
					0 "root" @ "refinementDepth" set-slot
					"experiment" @ "root" @ "experiment" set-slot
					800 "root" @ "ngRefine" "Refine the root mask by one bit" add-task
					true "experiment" @ "ngStarted" set-slot
				then
			then

			"CurSlot" @ "ngRefine" =
			if
				"CurUnit" @ "candidate" !
				"candidate" @ "ngRefined" get-slot nil =
				if
					"candidate" @ "experiment" get-slot "experiment" !
					"candidate" @ "mask" get-slot "mask" !
					3 iota each
						it "bit" !
						"mask" @ "bit" @ ng-refine-mask "childMask" !
						"childMask" @ nil !=
						if
							"mask:" "childMask" @ concat "childSemantic" !
							"candidate" "childSemantic" @ ng-artifact-name "child" !
							"child" @ unit-exists? "existed" !
							"child" @ "NogoodCandidate" create-unit drop
							"childMask" @ "child" @ "mask" set-slot
							"candidate" @ "child" @ "refinedFrom" add-to-slot
							"experiment" @ "child" @ "experiment" set-slot
							"refinement:" "candidate" @ concat ":" concat "child" @ concat "edgeSemantic" !
							"refinement" "edgeSemantic" @ ng-artifact-name "edge" !
							"edge" @ unit-exists? not
							if
								"edge" @ "NogoodRefinement" create-unit drop
								"candidate" @ "edge" @ "parent" set-slot
								"child" @ "edge" @ "child" set-slot
								"bit" @ "edge" @ "addedBit" set-slot
							then
							"existed" @ not
							if
								800 "child" @ "ngRefine" "Refine one candidate mask by one bit" add-task
							then
						then
					end
					700 "candidate" @ "ngEvaluate" "Evaluate candidate against all public training examples" add-task
					true "candidate" @ "ngRefined" set-slot
				then
			then

			"CurSlot" @ "ngEvaluate" =
			if
				"CurUnit" @ "candidate" !
				"candidate" @ "ngEvaluated" get-slot nil =
				if
					"candidate" @ "experiment" get-slot "experiment" !
					"candidate" @ "mask" get-slot "mask" !
					0 "exampleCount" ! 0 "matchCount" ! 0 "badClaims" ! 0 list-of "evidenceUnits" !
					"experiment" @ "trainingExamples" get-slot each
						it "example" !
						"exampleCount" @ 1 + "exampleCount" !
						"example" @ "problem" get-slot "problem" !
						"example" @ "decisionVariable" get-slot "anchor" !
						"example" @ "decisionColor" get-slot "blocked" !
						false "bindingFound" ! 0 "bx" ! 0 "by" ! 0 "escape" ! 0 "only" !
						3 iota each
							it "tryX" !
							3 iota each
								it "tryY" !
								4 iota each
									it "tryEscape" !
									4 iota each
										it "tryOnly" !
										"problem" @ "anchor" @ "blocked" @ "anchor" @ "tryX" @ "tryY" @ "blocked" @ "tryEscape" @ "tryOnly" @ ng-guard-matches?
										if
											true "bindingFound" ! "tryX" @ "bx" ! "tryY" @ "by" ! "tryEscape" @ "escape" ! "tryOnly" @ "only" !
										then
									end
								end
							end
						end
						"binding:" "candidate" @ concat ":" concat "example" @ concat "bindingSemantic" !
						"binding" "bindingSemantic" @ ng-artifact-name "binding" !
						"binding" @ "NogoodBinding" create-unit drop
						"candidate" @ "binding" @ "candidate" set-slot "example" @ "binding" @ "example" set-slot
						"bindingFound" @ "binding" @ "guardMatched" set-slot
						"anchor" @ "binding" @ "anchor" set-slot "bx" @ "binding" @ "x" set-slot "by" @ "binding" @ "y" set-slot
						"blocked" @ "binding" @ "blocked" set-slot "escape" @ "binding" @ "escape" set-slot "only" @ "binding" @ "only" set-slot
						false "matches" ! true "allConflict" ! 0 "completionCount" ! 0 list-of "results" !
						"bindingFound" @
						if
							"problem" @ "mask" @ "anchor" @ "bx" @ "by" @ "blocked" @ "escape" @ "only" @ ng-mask-matches? "matches" !
						then
						"matches" @
						if
							"matchCount" @ 1 + "matchCount" !
							"problem" @ "anchor" @ "bx" @ ng-edge-has? not "xKeepsBlocked" !
							"problem" @ "anchor" @ "by" @ ng-edge-has? not "yKeepsBlocked" !
							4 iota each
								it "xColor" !
								4 iota each
									it "yColor" !
									"problem" @ "bx" @ "xColor" @ ng-domain-has?
									"problem" @ "by" @ "yColor" @ ng-domain-has? and
									"xColor" @ "blocked" @ != "xKeepsBlocked" @ or and
									"yColor" @ "blocked" @ != "yKeepsBlocked" @ or and
									if
										"completionCount" @ 1 + "completionCount" !
										"problem" @ "mask" @ "anchor" @ "bx" @ "by" @ "blocked" @ "escape" @ "only" @ "xColor" @ "yColor" @ ng-completion-conflicts? "conflict" !
										"result:" "candidate" @ concat ":" concat "example" @ concat ":" concat "xColor" @ concat ":" concat "yColor" @ concat "resultSemantic" !
										"result" "resultSemantic" @ ng-artifact-name "result" !
										"result" @ "NogoodResult" create-unit drop
										"binding" @ "result" @ "binding" set-slot "xColor" @ "result" @ "xColor" set-slot "yColor" @ "result" @ "yColor" set-slot "conflict" @ "result" @ "conflict" set-slot
										"results" @ "result" @ list-append "results" !
										"conflict" @ not if false "allConflict" ! then
									then
								end
							end
							"allConflict" @ not if "badClaims" @ 1 + "badClaims" ! then
						then
						"evidence:" "candidate" @ concat ":" concat "example" @ concat "evidenceSemantic" !
						"evidence" "evidenceSemantic" @ ng-artifact-name "evidence" !
						"evidence" @ "NogoodEvidence" create-unit drop
						"candidate" @ "evidence" @ "candidate" set-slot "example" @ "evidence" @ "example" set-slot "binding" @ "evidence" @ "binding" set-slot
						"matches" @ "evidence" @ "matches" set-slot "completionCount" @ "evidence" @ "completionCount" set-slot "allConflict" @ "evidence" @ "allConflict" set-slot "results" @ "evidence" @ "results" set-slot
						"evidenceUnits" @ "evidence" @ list-append "evidenceUnits" !
					end
					"exampleCount" @ 4 = "matchCount" @ 1 = and "badClaims" @ 0 = and "exact" !
					"exampleCount" @ "candidate" @ "exampleCount" set-slot "matchCount" @ "candidate" @ "matchCount" set-slot "badClaims" @ "candidate" @ "badClaimCount" set-slot
					"evidenceUnits" @ "candidate" @ "evidenceUnits" set-slot "exact" @ "candidate" @ "trainingExact" set-slot true "candidate" @ "evidenceComplete" set-slot true "candidate" @ "ngEvaluated" set-slot
					600 "experiment" @ "ngSelect" "Select only after the complete candidate evidence barrier" add-task
				then
			then

			"CurSlot" @ "ngSelect" =
			if
				"CurUnit" @ "experiment" !
				"experiment" @ "selectionUnit" get-slot nil =
				if
					0 "candidateCount" ! 0 "completeCount" ! 0 list-of "exactCandidates" !
					"NogoodCandidate" examples each
						it "candidate" !
						"candidate" @ "NogoodCandidate" !=
						if
							"candidateCount" @ 1 + "candidateCount" !
							"candidate" @ "evidenceComplete" get-slot true = if "completeCount" @ 1 + "completeCount" ! then
							"candidate" @ "trainingExact" get-slot true = if "exactCandidates" @ "candidate" @ list-append "exactCandidates" ! then
						then
					end
					"candidateCount" @ 8 = "completeCount" @ 8 = and
					if
						"exactCandidates" @ list-length 1 =
						if
							"exactCandidates" @ 0 list-get "selected" !
							"selection" "unique-training-exact" ng-artifact-name "selection" !
							"selection" @ "NogoodSelection" create-unit drop
							"selected" @ "selection" @ "selectedCandidate" set-slot "exactCandidates" @ "selection" @ "ties" set-slot "candidateCount" @ "selection" @ "candidateCount" set-slot "completeCount" @ "selection" @ "completeCount" set-slot
							"selection" @ "experiment" @ "selectionUnit" set-slot
							600 "experiment" @ "ngPromote" "Prove the selected schema over all injective substitutions" add-task
						else
							"exactCandidates" @ list-length 0 = if "no-promotable-artifact" else "ambiguous" then "experiment" @ "terminal" set-slot
						then
					then
				then
			then

			"CurSlot" @ "ngPromote" =
			if
				"CurUnit" @ "experiment" !
				"experiment" @ "artifactUnit" get-slot nil =
				if
					"experiment" @ "selectionUnit" get-slot "selection" !
					"selection" @ "selectedCandidate" get-slot "selected" !
					"selected" @ "mask" get-slot "mask" !
					0 "proofCount" ! 0 "conflictCount" ! 0 list-of "proofs" !
					"experiment" @ "promotionCases" get-slot each
						it "case" !
						"case" @ "problem" get-slot "problem" !
						"case" @ "anchor" get-slot "anchor" ! "case" @ "x" get-slot "x" ! "case" @ "y" get-slot "y" !
						"case" @ "blocked" get-slot "blocked" ! "case" @ "escape" get-slot "escape" ! "case" @ "only" get-slot "only" !
						"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "only" @ "only" @ "only" @ ng-completion-conflicts? "conflict" !
						"proof:" "case" @ concat "proofSemantic" ! "promotion-proof" "proofSemantic" @ ng-artifact-name "proof" !
						"proof" @ "NogoodPromotionProof" create-unit drop "case" @ "proof" @ "case" set-slot "mask" @ "proof" @ "mask" set-slot "conflict" @ "proof" @ "conflict" set-slot
						"proofs" @ "proof" @ list-append "proofs" ! "proofCount" @ 1 + "proofCount" ! "conflict" @ if "conflictCount" @ 1 + "conflictCount" ! then
					end
					"mask" @ 7 = "proofCount" @ 24 = and "conflictCount" @ 24 = and
					if
						"artifact" "blocked-pair/v1:mask:7" ng-artifact-name "artifact" !
						"artifact" @ "NogoodArtifact" create-unit drop
						"blocked-pair/v1" "artifact" @ "schemaVersion" set-slot "blocked-pair-guard/v1" "artifact" @ "guardVersion" set-slot 7 "artifact" @ "mask" set-slot
						"selection" @ "artifact" @ "selection" set-slot "proofs" @ "artifact" @ "promotionProofs" set-slot "proofCount" @ "artifact" @ "promotionProofCount" set-slot
						"part3/nogoods/v1" "artifact" @ "provenance" set-slot true "artifact" @ "frozen" set-slot
						"artifact" @ "experiment" @ "artifactUnit" set-slot "promoted" "experiment" @ "terminal" set-slot
					else
						"no-promotable-artifact" "experiment" @ "terminal" set-slot
					then
				then
			then

			"CurSlot" @ "ngConsiderPrune" =
			if
				"CurUnit" @ "request" !
				"request" @ "dispositionUnit" get-slot nil =
				if
					"request" @ "problem" get-slot "problem" !
					"request" @ "decisionVariable" get-slot "anchor" !
					"request" @ "decisionColor" get-slot "blocked" !
					0 "anchorDomainCount" ! 0 "escape" !
					4 iota each
						it "color" !
						"problem" @ "anchor" @ "color" @ ng-domain-has?
						if
							"anchorDomainCount" @ 1 + "anchorDomainCount" !
							"color" @ "blocked" @ != if "color" @ "escape" ! then
						then
					end
					0 list-of "roleCandidates" !
					"anchorDomainCount" @ 2 = "problem" @ "anchor" @ "blocked" @ ng-domain-has? and
					if
						8 iota each
							it "variable" !
							"variable" @ "anchor" @ !=
							if
								0 "domainCount" ! 0 "only" !
								4 iota each
									it "color" !
									"problem" @ "variable" @ "color" @ ng-domain-has?
									if
										"domainCount" @ 1 + "domainCount" !
										"color" @ "blocked" @ != if "color" @ "only" ! then
									then
								end
								"domainCount" @ 2 = "problem" @ "variable" @ "blocked" @ ng-domain-has? and
								if
									"role:" "request" @ concat ":" concat "variable" @ concat "roleSemantic" !
									"role" "roleSemantic" @ ng-artifact-name "role" !
									"role" @ "NogoodRoleCandidate" create-unit drop
									"request" @ "role" @ "request" set-slot "variable" @ "role" @ "variable" set-slot "only" @ "role" @ "only" set-slot
									"roleCandidates" @ "role" @ list-append "roleCandidates" !
								then
							then
						end
					then
					0 "applicableCount" ! "" "applicableArtifact" ! "" "applicableBinding" ! "" "applicableCompletion" ! "" "applicableCertificate" !
					"roleCandidates" @ list-length iota each
						it "leftIndex" !
						"roleCandidates" @ list-length iota each
							it "rightIndex" !
							"leftIndex" @ "rightIndex" @ <
							if
								"roleCandidates" @ "leftIndex" @ list-get "leftRole" ! "roleCandidates" @ "rightIndex" @ list-get "rightRole" !
								"leftRole" @ "variable" get-slot "x" ! "rightRole" @ "variable" get-slot "y" !
								"leftRole" @ "only" get-slot "leftOnly" ! "rightRole" @ "only" get-slot "rightOnly" !
								"pair:" "request" @ concat ":" concat "x" @ concat ":" concat "y" @ concat "pairSemantic" !
								"pair" "pairSemantic" @ ng-artifact-name "pair" ! "pair" @ "NogoodPairProposal" create-unit drop
								"request" @ "pair" @ "request" set-slot "leftRole" @ "pair" @ "leftRole" set-slot "rightRole" @ "pair" @ "rightRole" set-slot
								"leftOnly" @ "rightOnly" @ = "leftOnly" @ "escape" @ != and "pairGuard" ! "pairGuard" @ "pair" @ "guardMatched" set-slot
								"pairGuard" @
								if
									"binding:" "pair" @ concat "bindingSemantic" ! "binding" "bindingSemantic" @ ng-artifact-name "binding" !
									"binding" @ "NogoodBinding" create-unit drop
									"request" @ "binding" @ "request" set-slot "anchor" @ "binding" @ "anchor" set-slot "x" @ "binding" @ "x" set-slot "y" @ "binding" @ "y" set-slot
									"blocked" @ "binding" @ "blocked" set-slot "escape" @ "binding" @ "escape" set-slot "leftOnly" @ "binding" @ "only" set-slot
									"NogoodArtifact" examples each
										it "artifact" !
										"artifact" @ "NogoodArtifact" !=
										if
											"artifact" @ "mask" get-slot "mask" !
											"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ ng-mask-matches?
											"artifact" @ "authoritative" get-slot true = and
											if
												"completion:" "binding" @ concat ":" concat "artifact" @ concat "completionSemantic" ! "completion" "completionSemantic" @ ng-artifact-name "completion" !
												"completion" @ "NogoodCompletion" create-unit drop "binding" @ "completion" @ "binding" set-slot "leftOnly" @ "completion" @ "xColor" set-slot "leftOnly" @ "completion" @ "yColor" set-slot
												"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ "leftOnly" @ "leftOnly" @ ng-completion-conflicts? "conflict" ! "conflict" @ "completion" @ "conflict" set-slot
												"certificate:" "completion" @ concat "certificateSemantic" ! "certificate" "certificateSemantic" @ ng-artifact-name "certificate" !
												"certificate" @ "NogoodCertificate" create-unit drop
												"artifact" @ "certificate" @ "artifact" set-slot "binding" @ "certificate" @ "binding" set-slot "completion" @ "certificate" @ "completion" set-slot
												"problem" @ "mask" @ "anchor" @ "blocked" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ "leftOnly" @ "leftOnly" @ "conflict" @ ng-certificate-valid? "valid" !
												"valid" @ "certificate" @ "valid" set-slot
												"valid" @
												if
													"applicableCount" @ 1 + "applicableCount" ! "artifact" @ "applicableArtifact" ! "binding" @ "applicableBinding" ! "completion" @ "applicableCompletion" ! "certificate" @ "applicableCertificate" !
												then
											then
										then
									end
								then
							then
						end
					end
					"disposition:" "request" @ concat "dispositionSemantic" ! "disposition" "dispositionSemantic" @ ng-artifact-name "disposition" !
					"disposition" @ "NogoodDisposition" create-unit drop
					"request" @ "disposition" @ "request" set-slot "request" @ "requestDigest" get-slot "disposition" @ "requestDigest" set-slot
					"applicableCount" @ 0 = if "resume" else "applicableCount" @ 1 = if "propose-prune" else "bridge-invalid" then then "status" !
					"status" @ "disposition" @ "status" set-slot "applicableCount" @ "disposition" @ "applicableCount" set-slot
					"applicableArtifact" @ "disposition" @ "artifact" set-slot "applicableBinding" @ "disposition" @ "binding" set-slot "applicableCompletion" @ "disposition" @ "completion" set-slot "applicableCertificate" @ "disposition" @ "certificate" set-slot
					true "disposition" @ "sealed" set-slot
					"disposition" @ "request" @ "dispositionUnit" set-slot
				then
			then
			"""#
	},
]
