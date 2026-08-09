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
				"experiment" @ "meterToken" get-slot "meter" !
				"experiment" @ "ngStarted" get-slot nil =
				if
					"candidate" "mask:0" ng-artifact-name "root" !
					"meter" @ 1 "candidate-proposal" "experiment" @ "root" @ "ok" ng-meter drop
					"meter" @ 12 "semantic-key-read" "experiment" @ "mask:0" "ok" ng-meter drop "meter" @ 12 "candidate-write" "experiment" @ "root" @ "ok" ng-meter drop
					"root" @ "NogoodCandidate" create-unit drop
					0 "root" @ "mask" set-slot
					0 "root" @ "refinementDepth" set-slot
					"experiment" @ "root" @ "experiment" set-slot
					800 "root" @ "ngRefine" "Refine the root mask by one bit" add-task
					"meter" @ 12 "agenda-enqueue" "root" @ "ngRefine" "ok" ng-meter drop
					true "experiment" @ "ngStarted" set-slot
				then
			then

			"CurSlot" @ "ngRefine" =
			if
				"CurUnit" @ "candidate" !
				"candidate" @ "ngRefined" get-slot nil =
				if
					"candidate" @ "experiment" get-slot "experiment" !
					"experiment" @ "meterToken" get-slot "meter" !
					"candidate" @ "mask" get-slot "mask" !
					3 iota each
						it "bit" !
						"mask" @ "bit" @ ng-refine-mask "childMask" !
						"childMask" @ nil !=
						if
							"mask:" "childMask" @ concat "childSemantic" !
							"meter" @ 1 "refinement-proposal" "candidate" @ "childSemantic" @ "ok" ng-meter drop "meter" @ 12 "semantic-key-read" "candidate" @ "childSemantic" @ "ok" ng-meter drop
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
								"meter" @ 12 "refinement-record-write" "candidate" @ "edge" @ "ok" ng-meter drop
								"edge" @ "NogoodRefinement" create-unit drop
								"candidate" @ "edge" @ "parent" set-slot
								"child" @ "edge" @ "child" set-slot
								"bit" @ "edge" @ "addedBit" set-slot
							then
							"existed" @ not
							if
								"meter" @ 12 "candidate-write" "candidate" @ "child" @ "ok" ng-meter drop
								800 "child" @ "ngRefine" "Refine one candidate mask by one bit" add-task
								"meter" @ 12 "agenda-enqueue" "child" @ "ngRefine" "ok" ng-meter drop
							then
						then
					end
					700 "candidate" @ "ngEvaluate" "Evaluate candidate against all public training examples" add-task
					"meter" @ 12 "agenda-enqueue" "candidate" @ "ngEvaluate" "ok" ng-meter drop
					true "candidate" @ "ngRefined" set-slot
				then
			then

			"CurSlot" @ "ngEvaluate" =
			if
				"CurUnit" @ "candidate" !
				"candidate" @ "ngEvaluated" get-slot nil =
				if
					"candidate" @ "experiment" get-slot "experiment" !
					"experiment" @ "meterToken" get-slot "meter" !
					"candidate" @ "mask" get-slot "mask" !
					0 "exampleCount" ! 0 "matchCount" ! 0 "badClaims" ! 0 "barrierCount" ! 0 list-of "evidenceUnits" !
					"experiment" @ "trainingExamples" get-slot each
						it "example" !
						"meter" @ 2 "problem-read" "example" @ "problem" "ok" ng-meter drop "meter" @ 2 "decision-variable-read" "example" @ "decisionVariable" "ok" ng-meter drop "meter" @ 2 "decision-color-read" "example" @ "decisionColor" "ok" ng-meter drop
						"exampleCount" @ 1 + "exampleCount" !
						"example" @ "problem" get-slot "problem" !
						"example" @ "decisionVariable" get-slot "anchor" !
						"example" @ "decisionColor" get-slot "blocked" !
						false "bindingFound" ! 0 "bx" ! 0 "by" ! 0 "escape" ! 0 "only" !
						"problem" @ "anchor" @ ng-domain-values "trainingAnchorDomain" !
						"trainingAnchorDomain" @ each it "trainingColor" ! "trainingColor" @ "blocked" @ != if "trainingColor" @ "escape" ! then end
						3 iota each
							it "tryX" !
							"tryX" @ "anchor" @ !=
							if
								"problem" @ "tryX" @ ng-domain-values "tryXDomain" ! 0 "tryXOnly" !
								"tryXDomain" @ each it "trainingColor" ! "trainingColor" @ "blocked" @ != if "trainingColor" @ "tryXOnly" ! then end
								3 iota each
									it "tryY" !
									"tryY" @ "tryX" @ > "tryY" @ "anchor" @ != and
									if
										"problem" @ "tryY" @ ng-domain-values "tryYDomain" ! 0 "tryYOnly" !
										"tryYDomain" @ each it "trainingColor" ! "trainingColor" @ "blocked" @ != if "trainingColor" @ "tryYOnly" ! then end
										"trainingAnchorDomain" @ list-length 2 = "tryXDomain" @ list-length 2 = and "tryYDomain" @ list-length 2 = and
										"tryXDomain" @ "blocked" @ list-contains and "tryYDomain" @ "blocked" @ list-contains and
										"tryXOnly" @ "tryYOnly" @ = and "tryXOnly" @ "escape" @ != and
										if true "bindingFound" ! "tryX" @ "bx" ! "tryY" @ "by" ! "tryXOnly" @ "only" ! then
									then
								end
							then
						end
						"binding:" "candidate" @ concat ":" concat "example" @ concat "bindingSemantic" !
						"binding" "bindingSemantic" @ ng-artifact-name "binding" !
						"binding" @ "NogoodBinding" create-unit drop
						"meter" @ 1 "binding-proposal" "candidate" @ "binding" @ "ok" ng-meter drop "meter" @ 12 "binding-write" "candidate" @ "binding" @ "ok" ng-meter drop
						"candidate" @ "binding" @ "candidate" set-slot "example" @ "binding" @ "example" set-slot
						"bindingFound" @ "binding" @ "guardMatched" set-slot
						"anchor" @ "binding" @ "anchor" set-slot "bx" @ "binding" @ "x" set-slot "by" @ "binding" @ "y" set-slot
						"blocked" @ "binding" @ "blocked" set-slot "escape" @ "binding" @ "escape" set-slot "only" @ "binding" @ "only" set-slot
						false "matches" ! true "allConflict" ! 0 "completionCount" ! 0 list-of "results" ! 0 list-of "expectedKeys" ! 0 list-of "actualKeys" !
						"bindingFound" @
						if
							"meter" @ 8 "artifact-read" "candidate" @ "mask" "ok" ng-meter drop "meter" @ 8 "mask-bit-check" "candidate" @ "bit:0" "ok" ng-meter drop "meter" @ 8 "mask-bit-check" "candidate" @ "bit:1" "ok" ng-meter drop "meter" @ 8 "mask-bit-check" "candidate" @ "bit:2" "ok" ng-meter drop
							"meter" @ 2 "edge-read" "example" @ "a-x" "ok" ng-meter drop "meter" @ 2 "edge-read" "example" @ "a-y" "ok" ng-meter drop "meter" @ 2 "edge-read" "example" @ "x-y" "ok" ng-meter drop
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
										"completion:" "xColor" @ concat ":" concat "yColor" @ concat "completionKey" !
										"meter" @ 9 "completion-construct" "binding" @ "completionKey" @ "ok" ng-meter drop "meter" @ 2 "domain-read-x" "binding" @ "completionKey" @ "ok" ng-meter drop "meter" @ 2 "domain-read-y" "binding" @ "completionKey" @ "ok" ng-meter drop
										"meter" @ 4 "inequality-check" "binding" @ "a-x" "ok" ng-meter drop "meter" @ 4 "inequality-check" "binding" @ "a-y" "ok" ng-meter drop "meter" @ 4 "inequality-check" "binding" @ "x-y" "ok" ng-meter drop
										"expectedKeys" @ "completionKey" @ list-append "expectedKeys" !
										"problem" @ "mask" @ "anchor" @ "bx" @ "by" @ "blocked" @ "escape" @ "only" @ "xColor" @ "yColor" @ ng-completion-conflicts? "conflict" !
										"result:" "candidate" @ concat ":" concat "example" @ concat ":" concat "xColor" @ concat ":" concat "yColor" @ concat "resultSemantic" !
										"result" "resultSemantic" @ ng-artifact-name "result" !
										"result" @ "NogoodResult" create-unit drop
										"meter" @ 12 "result-write" "binding" @ "result" @ "ok" ng-meter drop
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
						"meter" @ 12 "evidence-write" "candidate" @ "evidence" @ "ok" ng-meter drop
						"candidate" @ "evidence" @ "candidate" set-slot "example" @ "evidence" @ "example" set-slot "binding" @ "evidence" @ "binding" set-slot
						"matches" @ "evidence" @ "matches" set-slot "completionCount" @ "evidence" @ "completionCount" set-slot "allConflict" @ "evidence" @ "allConflict" set-slot "results" @ "evidence" @ "results" set-slot
						0 list-of "actualKeys" !
						"results" @ each
							it "storedResult" !
							"meter" @ 12 "result-read" "evidence" @ "storedResult" @ "ok" ng-meter drop
							"completion:" "storedResult" @ "xColor" get-slot concat ":" concat "storedResult" @ "yColor" get-slot concat "storedKey" !
							"actualKeys" @ "storedKey" @ list-append "actualKeys" !
						end
						"barrier:" "candidate" @ concat ":" concat "example" @ concat "barrierSemantic" ! "evidence-barrier" "barrierSemantic" @ ng-artifact-name "barrier" !
						"barrier" @ "NogoodEvidenceBarrier" create-unit drop
						"expectedKeys" @ each it "meterKey" ! "meter" @ 10 "expected-key-check" "barrier" @ "meterKey" @ "ok" ng-meter drop end
						"actualKeys" @ each it "meterKey" ! "meter" @ 10 "actual-key-check" "barrier" @ "meterKey" @ "ok" ng-meter drop end
						"meter" @ 10 "key-set-equality" "barrier" @ "expected-actual" "ok" ng-meter drop "meter" @ 10 "completion-count-check" "barrier" @ "completionCount" "ok" ng-meter drop "meter" @ 12 "barrier-write" "candidate" @ "barrier" @ "ok" ng-meter drop
						"candidate" @ "barrier" @ "candidate" set-slot "example" @ "barrier" @ "example" set-slot "expectedKeys" @ "barrier" @ "expectedKeys" set-slot "actualKeys" @ "barrier" @ "actualKeys" set-slot
						"expectedKeys" @ "actualKeys" @ list-equal? "completionCount" @ "actualKeys" @ list-length = and "barrierSealed" !
						"barrierSealed" @ "barrier" @ "sealed" set-slot "barrier" @ "evidence" @ "barrier" set-slot
						"barrierSealed" @ if "barrierCount" @ 1 + "barrierCount" ! then
						"evidenceUnits" @ "evidence" @ list-append "evidenceUnits" !
					end
					"exampleCount" @ 4 = "matchCount" @ 1 = and "badClaims" @ 0 = and "exact" !
					"exampleCount" @ "candidate" @ "exampleCount" set-slot "matchCount" @ "candidate" @ "matchCount" set-slot "badClaims" @ "candidate" @ "badClaimCount" set-slot
					"evidenceUnits" @ "candidate" @ "evidenceUnits" set-slot "barrierCount" @ "candidate" @ "barrierCount" set-slot "exact" @ "candidate" @ "trainingExact" set-slot "barrierCount" @ 4 = "candidate" @ "evidenceComplete" set-slot true "candidate" @ "ngEvaluated" set-slot
					600 "experiment" @ "ngSelect" "Select only after the complete candidate evidence barrier" add-task
					"meter" @ 12 "agenda-enqueue" "experiment" @ "ngSelect" "ok" ng-meter drop
				then
			then

			"CurSlot" @ "ngSelect" =
			if
				"CurUnit" @ "experiment" !
				"experiment" @ "meterToken" get-slot "meter" !
				"experiment" @ "selectionUnit" get-slot nil =
				if
					0 "candidateCount" ! 0 "completeCount" ! 0 list-of "exactCandidates" ! 0 list-of "actualMasks" !
					"NogoodCandidate" examples each
						it "candidate" !
						"candidate" @ "NogoodCandidate" !=
						if
							"meter" @ 8 "selection-evidence-read" "experiment" @ "candidate" @ "ok" ng-meter drop "meter" @ 12 "selection-comparison" "experiment" @ "candidate" @ "ok" ng-meter drop
							"candidateCount" @ 1 + "candidateCount" !
							"actualMasks" @ "candidate" @ "mask" get-slot list-append "actualMasks" !
							"candidate" @ "evidenceComplete" get-slot true = if "completeCount" @ 1 + "completeCount" ! then
							"candidate" @ "trainingExact" get-slot true = if "exactCandidates" @ "candidate" @ list-append "exactCandidates" ! then
						then
					end
					"actualMasks" @ sort "actualMasks" ! 8 iota "expectedMasks" !
					"selection-barrier" "mask-population-0-through-7" ng-artifact-name "selectionBarrier" !
					"selectionBarrier" @ "NogoodEvidenceBarrier" create-unit drop "expectedMasks" @ "selectionBarrier" @ "expectedKeys" set-slot "actualMasks" @ "selectionBarrier" @ "actualKeys" set-slot
					"expectedMasks" @ each it "meterKey" ! "mask:" "meterKey" @ concat "meterObject" ! "meter" @ 10 "expected-mask-check" "selectionBarrier" @ "meterObject" @ "ok" ng-meter drop end
					"actualMasks" @ each it "meterKey" ! "mask:" "meterKey" @ concat "meterObject" ! "meter" @ 10 "actual-mask-check" "selectionBarrier" @ "meterObject" @ "ok" ng-meter drop end
					"meter" @ 10 "mask-set-equality" "selectionBarrier" @ "expected-actual" "ok" ng-meter drop "meter" @ 10 "candidate-count-check" "selectionBarrier" @ "candidateCount" "ok" ng-meter drop "meter" @ 10 "complete-count-check" "selectionBarrier" @ "completeCount" "ok" ng-meter drop "meter" @ 12 "selection-barrier-write" "experiment" @ "selectionBarrier" @ "ok" ng-meter drop
					"expectedMasks" @ "actualMasks" @ list-equal? "populationSealed" ! "populationSealed" @ "selectionBarrier" @ "sealed" set-slot
					"candidateCount" @ 8 = "completeCount" @ 8 = and "populationSealed" @ and
					if
						"exactCandidates" @ list-length 1 =
						if
							"exactCandidates" @ 0 list-get "selected" !
							"selection" "unique-training-exact" ng-artifact-name "selection" !
							"selection" @ "NogoodSelection" create-unit drop
							"meter" @ 12 "tie-record-write" "selection" @ "exactCandidates" "ok" ng-meter drop "meter" @ 12 "selection-record-write" "experiment" @ "selection" @ "ok" ng-meter drop
							"selected" @ "selection" @ "selectedCandidate" set-slot "exactCandidates" @ "selection" @ "ties" set-slot "candidateCount" @ "selection" @ "candidateCount" set-slot "completeCount" @ "selection" @ "completeCount" set-slot
							"selectionBarrier" @ "selection" @ "barrier" set-slot
							"selection" @ "experiment" @ "selectionUnit" set-slot
							600 "experiment" @ "ngPromote" "Prove the selected schema over all injective substitutions" add-task
							"meter" @ 12 "agenda-enqueue" "experiment" @ "ngPromote" "ok" ng-meter drop
						else
							"exactCandidates" @ list-length 0 = if "no-promotable-artifact" else "ambiguous" then "experiment" @ "terminal" set-slot
						then
					then
				then
			then

			"CurSlot" @ "ngPromote" =
			if
				"CurUnit" @ "experiment" !
				"experiment" @ "meterToken" get-slot "meter" !
				"experiment" @ "artifactUnit" get-slot nil =
				if
					"experiment" @ "selectionUnit" get-slot "selection" !
					"selection" @ "selectedCandidate" get-slot "selected" !
					"selected" @ "mask" get-slot "mask" !
					0 "proofCount" ! 0 "conflictCount" ! 0 list-of "proofs" ! 0 list-of "expectedCases" ! 0 list-of "actualCases" !
					"experiment" @ "promotionCases" get-slot each
						it "case" !
						"meter" @ 9 "promotion-completion" "case" @ "only-only" "ok" ng-meter drop
						"expectedCases" @ "case" @ list-append "expectedCases" !
						"case" @ "problem" get-slot "problem" !
						"case" @ "anchor" get-slot "anchor" ! "case" @ "x" get-slot "x" ! "case" @ "y" get-slot "y" !
						"case" @ "blocked" get-slot "blocked" ! "case" @ "escape" get-slot "escape" ! "case" @ "only" get-slot "only" !
						"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "only" @ "only" @ "only" @ ng-completion-conflicts? "conflict" !
						"proof:" "case" @ concat "proofSemantic" ! "promotion-proof" "proofSemantic" @ ng-artifact-name "proof" !
						"proof" @ "NogoodPromotionProof" create-unit drop "case" @ "proof" @ "case" set-slot "mask" @ "proof" @ "mask" set-slot "conflict" @ "proof" @ "conflict" set-slot
						"meter" @ 12 "promotion-proof-write" "case" @ "proof" @ "ok" ng-meter drop
						"proofs" @ "proof" @ list-append "proofs" ! "proofCount" @ 1 + "proofCount" ! "conflict" @ if "conflictCount" @ 1 + "conflictCount" ! then
					end
					0 list-of "actualCases" !
					"proofs" @ each
						it "storedProof" ! "actualCases" @ "storedProof" @ "case" get-slot list-append "actualCases" !
					end
					"promotion-barrier" "all-injective-color-substitutions" ng-artifact-name "promotionBarrier" !
					"promotionBarrier" @ "NogoodEvidenceBarrier" create-unit drop "expectedCases" @ "promotionBarrier" @ "expectedKeys" set-slot "actualCases" @ "promotionBarrier" @ "actualKeys" set-slot
					"expectedCases" @ each it "meterKey" ! "meter" @ 10 "expected-promotion-check" "promotionBarrier" @ "meterKey" @ "ok" ng-meter drop end
					"actualCases" @ each it "meterKey" ! "meter" @ 10 "actual-promotion-check" "promotionBarrier" @ "meterKey" @ "ok" ng-meter drop end
					"meter" @ 10 "promotion-count-check" "promotionBarrier" @ "proofCount" "ok" ng-meter drop "meter" @ 10 "promotion-conflict-check" "promotionBarrier" @ "conflictCount" "ok" ng-meter drop "meter" @ 12 "promotion-barrier-write" "experiment" @ "promotionBarrier" @ "ok" ng-meter drop
					"expectedCases" @ "actualCases" @ list-equal? "proofCount" @ 24 = and "conflictCount" @ 24 = and "promotionSealed" ! "promotionSealed" @ "promotionBarrier" @ "sealed" set-slot
					"mask" @ 7 = "promotionSealed" @ and
					if
						"artifact" "blocked-pair/v1:mask:7" ng-artifact-name "artifact" !
						"artifact" @ "NogoodArtifact" create-unit drop
						"meter" @ 8 "artifact-freeze-write" "selection" @ "artifact" @ "ok" ng-meter drop "meter" @ 12 "provenance-write" "artifact" @ "part3/nogoods/v1" "ok" ng-meter drop "meter" @ 12 "boundary-write" "artifact" @ "promotionBarrier" @ "ok" ng-meter drop
						"blocked-pair/v1" "artifact" @ "schemaVersion" set-slot "blocked-pair-guard/v1" "artifact" @ "guardVersion" set-slot 7 "artifact" @ "mask" set-slot
						"selection" @ "artifact" @ "selection" set-slot "proofs" @ "artifact" @ "promotionProofs" set-slot "proofCount" @ "artifact" @ "promotionProofCount" set-slot
						"promotionBarrier" @ "artifact" @ "promotionBarrier" set-slot
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
					"request" @ "meterToken" get-slot "meter" !
					"request" @ "problem" get-slot "problem" !
					"request" @ "decisionVariable" get-slot "anchor" !
					"request" @ "decisionColor" get-slot "blocked" !
					"domain" "anchor" @ concat "anchorDomainSlot" ! "request" @ "anchorDomainSlot" @ get-slot "anchorDomain" !
					"meter" @ 2 "domain-read" "request" @ "anchorDomainSlot" @ "ok" ng-meter drop
					"anchorDomain" @ list-length "anchorDomainCount" ! 0 "escape" !
					"anchorDomain" @ each
						it "color" ! "color" @ "blocked" @ != if "color" @ "escape" ! then
					end
					0 "memoMatchCount" ! "" "applicableMemo" !
					"NogoodConcreteMemo" examples each
						it "memo" !
						"memo" @ "NogoodConcreteMemo" !=
						if
							"memo" @ "exactKey" get-slot "memoKey" ! "request" @ "concreteMemoKey" get-slot "requestMemoKey" !
							"memoKey" @ "requestMemoKey" @ = if "hit" else "miss" then "memoOutcome" !
							"meter" @ 12 "concrete-memo-lookup" "memo" @ "request" @ "memoOutcome" @ ng-meter drop
							"memoOutcome" @ "hit" = if "memoMatchCount" @ 1 + "memoMatchCount" ! "memo" @ "applicableMemo" ! then
						then
					end
					0 list-of "roleCandidates" !
					"anchorDomainCount" @ 2 = "anchorDomain" @ "blocked" @ list-contains and
					if
						8 iota each
							it "variable" !
							"variable" @ "anchor" @ !=
							if
								"domain" "variable" @ concat "domainSlot" ! "request" @ "domainSlot" @ get-slot "variableDomain" !
								"variable:" "variable" @ concat "meterObject" ! "meter" @ 2 "domain-size-check" "request" @ "domainSlot" @ "ok" ng-meter drop "meter" @ 2 "domain-membership-check" "request" @ "domainSlot" @ "ok" ng-meter drop "meter" @ 12 "role-visit-record" "request" @ "meterObject" @ "ok" ng-meter drop
								"variableDomain" @ list-length "domainCount" ! 0 "only" !
								"variableDomain" @ each
									it "color" ! "color" @ "blocked" @ != if "color" @ "only" ! then
								end
								"domainCount" @ 2 = "variableDomain" @ "blocked" @ list-contains and
								if
									"role:" "request" @ concat ":" concat "variable" @ concat "roleSemantic" !
									"role" "roleSemantic" @ ng-artifact-name "role" !
									"meter" @ 1 "role-candidate" "request" @ "role" @ "ok" ng-meter drop "meter" @ 12 "role-candidate-write" "request" @ "role" @ "ok" ng-meter drop
									"role" @ "NogoodRoleCandidate" create-unit drop
									"request" @ "role" @ "request" set-slot "variable" @ "role" @ "variable" set-slot "only" @ "role" @ "only" set-slot
									"roleCandidates" @ "role" @ list-append "roleCandidates" !
								then
							then
						end
					then
					0 "applicableCount" ! "" "applicableArtifact" ! "" "applicableBinding" ! "" "applicableCompletion" ! "" "applicableCertificate" ! "" "applicableBarrier" ! "" "applicableProposal" ! "" "applicableReferenceDigest" ! "" "applicableBarrierDigest" !
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
								"meter" @ 1 "pair-candidate" "request" @ "pair" @ "ok" ng-meter drop "meter" @ 2 "pair-only-equality" "leftRole" @ "rightRole" @ "ok" ng-meter drop "meter" @ 2 "pair-escape-inequality" "leftRole" @ "escape" "ok" ng-meter drop "meter" @ 12 "pair-record-write" "request" @ "pair" @ "ok" ng-meter drop
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
										"artifact-read" "mask-bit-0" "mask-bit-1" "mask-bit-2" "authority-read" "frozen-read" "schema-read" "guard-version-read" "artifact-digest-read" "evidence-digest-read" 10 list-of each it "meterOperation" ! "meter" @ 8 "meterOperation" @ "artifact" @ "binding" @ "ok" ng-meter drop end
										"edge-a-x" "edge-a-y" "edge-x-y" 3 list-of each it "meterObject" ! "meter" @ 2 "artifact-edge-read" "binding" @ "meterObject" @ "ok" ng-meter drop end
										"meter" @ 12 "artifact-match-read" "binding" @ "artifact" @ "ok" ng-meter drop "meter" @ 12 "artifact-match-record" "binding" @ "artifact" @ "ok" ng-meter drop
											"artifact" @ "mask" get-slot "mask" !
											"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ ng-mask-matches?
											"artifact" @ "authoritative" get-slot true = and
										if
												"completion:" "binding" @ concat ":" concat "artifact" @ concat "completionSemantic" ! "completion" "completionSemantic" @ ng-artifact-name "completion" !
												"meter" @ 9 "completion-construct" "binding" @ "completion" @ "ok" ng-meter drop "meter" @ 2 "completion-domain-read" "binding" @ "x" "ok" ng-meter drop "meter" @ 2 "completion-domain-read" "binding" @ "y" "ok" ng-meter drop
												"a-x" "a-y" "x-y" 3 list-of each it "meterObject" ! "meter" @ 4 "completion-inequality" "completion" @ "meterObject" @ "ok" ng-meter drop end "meter" @ 12 "completion-result-write" "binding" @ "completion" @ "ok" ng-meter drop
												"completion" @ "NogoodCompletion" create-unit drop "binding" @ "completion" @ "binding" set-slot "leftOnly" @ "completion" @ "xColor" set-slot "leftOnly" @ "completion" @ "yColor" set-slot
												"problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ "leftOnly" @ "leftOnly" @ ng-completion-conflicts? "conflict" ! "conflict" @ "completion" @ "conflict" set-slot
												"certificate:" "completion" @ concat "certificateSemantic" ! "certificate" "certificateSemantic" @ ng-artifact-name "certificate" !
												"certificate" @ "NogoodCertificate" create-unit drop
												"artifact" @ "certificate" @ "artifact" set-slot "binding" @ "certificate" @ "binding" set-slot "completion" @ "certificate" @ "completion" set-slot "request" @ "certificate" @ "request" set-slot
												"request" @ "requestDigest" get-slot "certificate" @ "requestDigest" set-slot "request" @ "targetDigest" get-slot "certificate" @ "targetDigest" set-slot "request" @ "decisionDigest" get-slot "certificate" @ "decisionDigest" set-slot
												"problem" @ "mask" @ "anchor" @ "blocked" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ "leftOnly" @ "leftOnly" @ "conflict" @ ng-certificate-valid? "valid" !
												"valid" @ "certificate" @ "valid" set-slot
												"valid" @
												if
													"meter" @ 10 "certificate-record" "completion" @ "certificate" @ "ok" ng-meter drop
													"predicate-problem" "predicate-guard" "predicate-mask" "predicate-conflict" "predicate-certificate" "predicate-authority" "predicate-frozen" "predicate-schema" "predicate-guard-version" "predicate-artifact-digest" "predicate-evidence-digest" "predicate-promotion-digest" "predicate-provenance-digest" "predicate-request-digest" "predicate-target-digest" "predicate-decision-digest" "predicate-assignment-digest" "predicate-reduced-domain-digest" 18 list-of "predicateKeys" !
													0 list-of "predicateResults" !
													"predicateResults" @ "problem" @ ng-problem-valid? list-append "predicateResults" !
													"predicateResults" @ "problem" @ "anchor" @ "blocked" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ ng-guard-matches? list-append "predicateResults" !
													"predicateResults" @ "problem" @ "mask" @ "anchor" @ "x" @ "y" @ "blocked" @ "escape" @ "leftOnly" @ ng-mask-matches? list-append "predicateResults" !
													"predicateResults" @ "conflict" @ list-append "predicateResults" ! "predicateResults" @ "valid" @ list-append "predicateResults" !
													"predicateResults" @ "artifact" @ "authoritative" get-slot true = list-append "predicateResults" ! "predicateResults" @ "artifact" @ "frozen" get-slot true = list-append "predicateResults" !
													"predicateResults" @ "artifact" @ "schemaVersion" get-slot "blocked-pair/v1" = list-append "predicateResults" ! "predicateResults" @ "artifact" @ "guardVersion" get-slot "blocked-pair-guard/v1" = list-append "predicateResults" !
													"predicateResults" @ "artifact" @ "artifactDigest" get-slot "request" @ "acceptedArtifactDigest" get-slot = list-append "predicateResults" ! "predicateResults" @ "artifact" @ "evidenceBoundaryDigest" get-slot "request" @ "acceptedEvidenceDigest" get-slot = list-append "predicateResults" ! "predicateResults" @ "artifact" @ "promotionDigest" get-slot "request" @ "acceptedPromotionDigest" get-slot = list-append "predicateResults" ! "predicateResults" @ "artifact" @ "provenanceDigest" get-slot "request" @ "acceptedProvenanceDigest" get-slot = list-append "predicateResults" !
													"predicateResults" @ "request" @ "requestDigest" get-slot "" != list-append "predicateResults" ! "predicateResults" @ "request" @ "targetDigest" get-slot "" != list-append "predicateResults" ! "predicateResults" @ "request" @ "decisionDigest" get-slot "" != list-append "predicateResults" ! "predicateResults" @ "request" @ "assignmentDigest" get-slot "" != list-append "predicateResults" ! "predicateResults" @ "request" @ "reducedDomainDigest" get-slot "" != list-append "predicateResults" !
															0 list-of "predicateOutcomeKeys" ! "predicateResults" @ each it if "true" else "false" then "predicateOutcomeKeys" @ swap list-append "predicateOutcomeKeys" ! end
															"predicateKeys" @ each it "meterPredicate" ! "meter" @ 10 "barrier-predicate-check" "certificate" @ "meterPredicate" @ "ok" ng-meter drop end
													"barrier:" "certificate" @ concat "barrierSemantic" ! "evidence-barrier" "barrierSemantic" @ ng-artifact-name "barrier" ! "barrier" @ "NogoodEvidenceBarrier" create-unit drop
													"request" @ "barrier" @ "request" set-slot "artifact" @ "barrier" @ "artifact" set-slot "binding" @ "barrier" @ "binding" set-slot "completion" @ "barrier" @ "completion" set-slot "certificate" @ "barrier" @ "certificate" set-slot
													"predicateKeys" @ "barrier" @ "predicateKeys" set-slot "predicateOutcomeKeys" @ "barrier" @ "predicateOutcomeKeys" set-slot
													"request" @ "requestDigest" get-slot "barrier" @ "requestDigest" set-slot "request" @ "targetDigest" get-slot "barrier" @ "targetDigest" set-slot "request" @ "decisionDigest" get-slot "barrier" @ "decisionDigest" set-slot "request" @ "assignmentDigest" get-slot "barrier" @ "assignmentDigest" set-slot "request" @ "immutableDomainDigest" get-slot "barrier" @ "immutableDomainDigest" set-slot "request" @ "reducedDomainDigest" get-slot "barrier" @ "reducedDomainDigest" set-slot "request" @ "exactConflictStoreDigest" get-slot "barrier" @ "exactConflictStoreDigest" set-slot
													"predicateResults" @ list-length 18 = "predicateResults" @ false list-contains not and "barrierSealed" ! "barrierSealed" @ "barrier" @ "sealed" set-slot
															"barrierSealed" @
															if
																"certificate-index-write" "agreement-result-read" "agreement-record-write" "expected-key-set-write" "actual-key-set-write" "sealed-barrier-write" 6 list-of each it "meterOperation" ! "meter" @ 12 "meterOperation" @ "certificate" @ "barrier" @ "ok" ng-meter drop end
														"proposal:" "barrier" @ concat "proposalSemantic" ! "prune-proposal" "proposalSemantic" @ ng-artifact-name "proposal" ! "proposal" @ "NogoodPruneProposal" create-unit drop
														"request" @ "proposal" @ "request" set-slot "artifact" @ "proposal" @ "artifact" set-slot "binding" @ "proposal" @ "binding" set-slot "completion" @ "proposal" @ "completion" set-slot "certificate" @ "proposal" @ "certificate" set-slot "barrier" @ "proposal" @ "barrier" set-slot
														"request" @ "artifact" @ "binding" @ "completion" @ "certificate" @ "barrier" @ "proposal" @ 7 list-of "referencedUnits" !
														"referencedUnits" @ ng-unit-set-digest "referenceDigest" ! "referenceDigest" @ "barrier" @ "referencedUnitSetDigest" set-slot "referenceDigest" @ "proposal" @ "referencedUnitSetDigest" set-slot
														"predicateKeys" @ "predicateOutcomeKeys" @ "referenceDigest" @ "request" @ "requestDigest" get-slot "request" @ "targetDigest" get-slot "request" @ "decisionDigest" get-slot "request" @ "assignmentDigest" get-slot "request" @ "immutableDomainDigest" get-slot "request" @ "reducedDomainDigest" get-slot "request" @ "exactConflictStoreDigest" get-slot 10 list-of ng-digest-record "barrierDigest" !
														"barrierDigest" @ "barrier" @ "barrierDigest" set-slot "barrierDigest" @ "proposal" @ "barrierDigest" set-slot
														"applicableCount" @ 1 + "applicableCount" ! "artifact" @ "applicableArtifact" ! "binding" @ "applicableBinding" ! "completion" @ "applicableCompletion" ! "certificate" @ "applicableCertificate" ! "barrier" @ "applicableBarrier" ! "proposal" @ "applicableProposal" ! "referenceDigest" @ "applicableReferenceDigest" ! "barrierDigest" @ "applicableBarrierDigest" !
													then
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
					"memoMatchCount" @ 1 = "applicableCount" @ 0 = and if "concrete-prune" else "memoMatchCount" @ 0 = if "applicableCount" @ 0 = if "resume" else "applicableCount" @ 1 = if "propose-prune" else "bridge-invalid" then then else "bridge-invalid" then then "status" !
					"status" @ "disposition" @ "status" set-slot "applicableCount" @ "disposition" @ "applicableCount" set-slot
					"status" @ "propose-prune" =
					if
						"disposition-write" "request-digest-check" "target-digest-check" "decision-digest-check" "assignment-digest-check" "artifact-digest-check" 6 list-of each it "meterOperation" ! "meter" @ 12 "meterOperation" @ "request" @ "disposition" @ "ok" ng-meter drop end
					else
						"disposition-write" "request-digest-check" "target-digest-check" "decision-digest-check" 4 list-of each it "meterOperation" ! "meter" @ 12 "meterOperation" @ "request" @ "disposition" @ "ok" ng-meter drop end
					then
					"applicableArtifact" @ "disposition" @ "artifact" set-slot "applicableBinding" @ "disposition" @ "binding" set-slot "applicableCompletion" @ "disposition" @ "completion" set-slot "applicableCertificate" @ "disposition" @ "certificate" set-slot "applicableBarrier" @ "disposition" @ "barrier" set-slot "applicableProposal" @ "disposition" @ "proposal" set-slot
					"applicableMemo" @ "disposition" @ "memo" set-slot
					"request" @ "targetDigest" get-slot "disposition" @ "targetDigest" set-slot "request" @ "decisionDigest" get-slot "disposition" @ "decisionDigest" set-slot
					"applicableReferenceDigest" @ "disposition" @ "referencedUnitSetDigest" set-slot "applicableBarrierDigest" @ "disposition" @ "barrierDigest" set-slot
					true "disposition" @ "sealed" set-slot
					"disposition" @ "request" @ "dispositionUnit" set-slot
				then
			then
			"""#
	},
]
