package domains

units: [
	{
		name: "AR-H-AllocateGuardSpace"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Traverse the frozen 451-guard refinement tree through explicit one-literal extensions"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arAllocate" =
			"CurUnit" @ "ActionRelationExperiment" isa? and
			"CurUnit" @ "guardSpaceAllocated" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "pattern" get-slot "pattern" !
			0 list-of "candidates" ! 0 list-of "edges" !
			0 "ordinal" !
			"AR.Candidate." "experiment" @ concat ".0" concat "rootRequest" !
			"pattern" @ "rootRequest" @ ar-guard-root "rootResult" !
			"rootResult" @ 0 list-get "rootGuard" !
			"rootResult" @ 1 list-get "rootName" !
			"experiment" @ "rootName" @ "experiment" set-slot
			"candidates" @ "rootName" @ list-append "candidates" !
			"read-write-disjoint" "primary-same" "secondary-same"
			"a-primary-b-secondary" "a-secondary-b-primary" "argument-equal"
			"argument-opposite" "symbol-equal" "shared-value-zero"
			"shared-value-max" "a-primary-zero" "a-primary-max"
			"b-primary-zero" "b-primary-max" "combined-adds-in-bounds"
			15 list-of "atoms" !
			"experiment" @ "scope" get-slot "no-guard" = if 0 else 15 then "atomCount" !
			false true 2 list-of "polarities" !
			0 list-of "oneLiteralCandidates" !
			"atomCount" @ iota each
				it "atomIndex" !
				"polarities" @ each
					it "polarity" !
					"ordinal" @ 1 + "ordinal" !
					"atoms" @ "atomIndex" @ list-get "atom" !
					"AR.Edge." "experiment" @ concat "." concat "edges" @ list-length concat "edgeRequest" !
					"rootGuard" @ "atom" @ "polarity" @ "edges" @ list-length "edgeRequest" @ ar-guard-extend "extendResult" !
					"extendResult" @ 0 list-get "guard" !
					"extendResult" @ 1 list-get "edge" !
					"AR.Candidate." "experiment" @ concat "." concat "ordinal" @ concat "candidateRequest" !
					"pattern" @ "guard" @ "rootName" @ "ordinal" @ "candidateRequest" @ ar-candidate-allocate "candidate" !
					"experiment" @ "candidate" @ "experiment" set-slot
					"candidates" @ "candidate" @ list-append "candidates" !
					"oneLiteralCandidates" @ "candidate" @ list-append "oneLiteralCandidates" !
					"experiment" @ "edge" @ "experiment" set-slot
					"rootName" @ "edge" @ "parentCandidate" set-slot
					"candidate" @ "edge" @ "childCandidate" set-slot
					"edges" @ "edge" @ list-append "edges" !
				end
			end
			"atomCount" @ iota each
				it "leftIndex" !
				"atomCount" @ iota each
					it "rightIndex" !
					"rightIndex" @ "leftIndex" @ >
					if
						"polarities" @ each
							it "leftPolarity" !
							"polarities" @ each
								it "rightPolarity" !
								"ordinal" @ 1 + "ordinal" !
								"leftIndex" @ 2 * "leftPolarity" @ if 1 else 0 then + "parentIndex" !
								"oneLiteralCandidates" @ "parentIndex" @ list-get "parent" !
								"atoms" @ "rightIndex" @ list-get "atom" !
								"AR.Edge." "experiment" @ concat "." concat "edges" @ list-length concat "edgeRequest" !
								"parent" @ "guard" get-slot "atom" @ "rightPolarity" @ "edges" @ list-length "edgeRequest" @ ar-guard-extend "extendResult" !
								"extendResult" @ 0 list-get "guard" !
								"extendResult" @ 1 list-get "edge" !
								"AR.Candidate." "experiment" @ concat "." concat "ordinal" @ concat "candidateRequest" !
								"pattern" @ "guard" @ "parent" @ "ordinal" @ "candidateRequest" @ ar-candidate-allocate "candidate" !
								"experiment" @ "candidate" @ "experiment" set-slot
								"candidates" @ "candidate" @ list-append "candidates" !
								"experiment" @ "edge" @ "experiment" set-slot
								"parent" @ "edge" @ "parentCandidate" set-slot
								"candidate" @ "edge" @ "childCandidate" set-slot
								"edges" @ "edge" @ list-append "edges" !
							end
						end
					then
				end
			end
			"candidates" @ "experiment" @ "candidateUnits" set-slot
			"edges" @ "experiment" @ "edgeUnits" set-slot
			"rootName" @ "experiment" @ "rootCandidate" set-slot
			"experiment" @ "scope" get-slot "no-guard" =
			if "ordinal" @ 0 = "candidates" @ list-length 1 = and "edges" @ list-length 0 = and
			else "ordinal" @ 450 = "candidates" @ list-length 451 = and "edges" @ list-length 450 = and then
			if true "experiment" @ "guardSpaceAllocated" set-slot
			else "no-discovery" "experiment" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-AssembleTrainingObservations"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Construct every training observation through visible applicability, transition, and complete-state equality rows"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arObserve" =
			"CurUnit" @ "ActionRelationExperiment" isa? and
			"CurUnit" @ "observationsAssembled" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			0 list-of "observations" !
			"ActionRelationTrainingCase" examples each
				it "case" !
				"case" @ "ActionRelationTrainingCase" !=
				"case" @ "experiment" get-slot "experiment" @ = and
				if
					"case" @ "state" get-slot "state" !
					"case" @ "aOccurrence" get-slot "a" !
					"case" @ "bOccurrence" get-slot "b" !
					"AR.Facts.A." "case" @ concat "aFactsRequest" !
					"AR.Facts.B." "case" @ concat "bFactsRequest" !
					"state" @ "a" @ "aFactsRequest" @ ar-action-facts "aFacts" !
					"state" @ "b" @ "bFactsRequest" @ ar-action-facts "bFacts" !
					"AR.App.A.Initial." "case" @ concat "aAppRequest" !
					"AR.App.B.Initial." "case" @ concat "bAppRequest" !
					"state" @ "a" @ "aAppRequest" @ ar-applicable? "aApp" !
					"state" @ "b" @ "bAppRequest" @ ar-applicable? "bApp" !
					"AR.Transition.A.Initial." "case" @ concat "aTransitionRequest" !
					"AR.Transition.B.Initial." "case" @ concat "bTransitionRequest" !
					"AR.State.AfterA." "case" @ concat "afterARequest" !
					"AR.State.AfterB." "case" @ concat "afterBRequest" !
					"state" @ "a" @ "aApp" @ "aTransitionRequest" @ "afterARequest" @ ar-apply "aInitialResult" !
					"state" @ "b" @ "bApp" @ "bTransitionRequest" @ "afterBRequest" @ ar-apply "bInitialResult" !
					"aInitialResult" @ 0 list-get "aInitial" !
					"bInitialResult" @ 0 list-get "bInitial" !
					"aInitialResult" @ 1 list-get "afterAUnit" !
					"bInitialResult" @ 1 list-get "afterBUnit" !
					"" "bAfterA" ! "" "aAfterB" ! "" "equality" !
					"" "abState" ! "" "baState" !
					"afterAUnit" @ "" !=
					if
						"afterAUnit" @ "state" get-slot "afterA" !
						"AR.App.B.AfterA." "case" @ concat "crossAppRequest" !
						"afterA" @ "b" @ "crossAppRequest" @ ar-applicable? "crossApp" !
						"AR.Transition.B.AfterA." "case" @ concat "crossTransitionRequest" !
						"AR.State.AB." "case" @ concat "abRequest" !
						"afterA" @ "b" @ "crossApp" @ "crossTransitionRequest" @ "abRequest" @ ar-apply "crossResult" !
						"crossResult" @ 0 list-get "bAfterA" !
						"crossResult" @ 1 list-get "abUnit" !
						"abUnit" @ "" != if "abUnit" @ "state" get-slot "abState" ! then
					then
					"afterBUnit" @ "" !=
					if
						"afterBUnit" @ "state" get-slot "afterB" !
						"AR.App.A.AfterB." "case" @ concat "crossAppRequest" !
						"afterB" @ "a" @ "crossAppRequest" @ ar-applicable? "crossApp" !
						"AR.Transition.A.AfterB." "case" @ concat "crossTransitionRequest" !
						"AR.State.BA." "case" @ concat "baRequest" !
						"afterB" @ "a" @ "crossApp" @ "crossTransitionRequest" @ "baRequest" @ ar-apply "crossResult" !
						"crossResult" @ 0 list-get "aAfterB" !
						"crossResult" @ 1 list-get "baUnit" !
						"baUnit" @ "" != if "baUnit" @ "state" get-slot "baState" ! then
					then
					"abState" @ "" != "baState" @ "" != and
					if
						"AR.Equality." "case" @ concat "equalityRequest" !
						"abState" @ "baState" @ "equalityRequest" @ ar-state-equal? "equality" !
					then
					"AR.Observation." "case" @ concat "observationRequest" !
					"state" @ "a" @ "b" @ "aInitial" @ "bInitial" @ "bAfterA" @ "aAfterB" @ "equality" @ "case" @ "label" get-slot "observationRequest" @ ar-observation-assemble "observation" !
					"observation" @ nil !=
					if
						"experiment" @ "observation" @ "experiment" set-slot
						"case" @ "observation" @ "trainingCase" set-slot
						"aFacts" @ "observation" @ "aFacts" set-slot
						"bFacts" @ "observation" @ "bFacts" set-slot
						"observations" @ "observation" @ list-append "observations" !
					then
				then
			end
			"observations" @ "experiment" @ "observationUnits" set-slot
			"observations" @ list-length "experiment" @ "expectedObservationCount" get-slot =
			if true "experiment" @ "observationsAssembled" set-slot
			else "no-discovery" "experiment" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-EvaluateGuardSpace"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Evaluate every normalized guard against every committed observation through explicit signed-literal rows"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arEvaluate" =
			"CurUnit" @ "ActionRelationExperiment" isa? and
			"CurUnit" @ "guardSpaceAllocated" get-slot true = and
			"CurUnit" @ "observationsAssembled" get-slot true = and
			"CurUnit" @ "guardsEvaluated" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			0 list-of "allResults" ! 0 list-of "allLiteralRows" !
			"experiment" @ "candidateUnits" get-slot each
				it "candidate" !
				"candidate" @ "guard" get-slot "guard" !
				"candidate" @ "atoms" get-slot "atoms" !
				"candidate" @ "polarities" get-slot "polarities" !
				0 list-of "candidateResults" !
				"experiment" @ "observationUnits" get-slot each
					it "observation" !
					0 list-of "literalRows" !
					"atoms" @ list-length iota each
						it "literalIndex" !
						"AR.Literal." "candidate" @ concat "." concat "observation" @ concat "." concat "literalIndex" @ concat "literalRequest" !
						"guard" @ "observation" @ "observation" @ "aFacts" get-slot "observation" @ "bFacts" get-slot "atoms" @ "literalIndex" @ list-get "polarities" @ "literalIndex" @ list-get "literalRequest" @ ar-guard-match "literalRow" !
						"literalRows" @ "literalRow" @ list-append "literalRows" !
						"allLiteralRows" @ "literalRow" @ list-append "allLiteralRows" !
					end
					"AR.GuardResult." "candidate" @ concat "." concat "observation" @ concat "resultRequest" !
					"guard" @ "observation" @ "literalRows" @ "resultRequest" @ ar-guard-result "guardResult" !
					"candidateResults" @ "guardResult" @ list-append "candidateResults" !
					"allResults" @ "guardResult" @ list-append "allResults" !
				end
				"candidateResults" @ "candidate" @ "guardResults" set-slot
			end
			"allResults" @ "experiment" @ "guardResultUnits" set-slot
			"allLiteralRows" @ "experiment" @ "literalRowUnits" set-slot
			"allResults" @ list-length "experiment" @ "candidateUnits" get-slot list-length "experiment" @ "observationUnits" get-slot list-length * =
			if true "experiment" @ "guardsEvaluated" set-slot
			else "no-discovery" "experiment" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-FinalizeGuardSearch"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Finalize all candidates and retain every tied optimum before evidence-bound closure"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arFinalize" =
			"CurUnit" @ "ActionRelationExperiment" isa? and
			"CurUnit" @ "guardsEvaluated" get-slot true = and
			"CurUnit" @ "candidatesFinalized" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			0 list-of "candidateResults" !
			"experiment" @ "candidateUnits" get-slot each
				it "candidate" !
				"AR.CandidateResult." "candidate" @ concat "request" !
				"candidate" @ "candidate" @ "guardResults" get-slot "experiment" @ "viewEvidenceRoot" get-slot "request" @ ar-candidate-result "result" !
				"result" @ "candidate" @ "candidateResult" set-slot
				"candidateResults" @ "result" @ list-append "candidateResults" !
			end
			-1 "maxPositive" ! 3 "minLiterals" !
			"candidateResults" @ each
				it "result" !
				"result" @ "eligible" get-slot
				if
					"result" @ "positiveCoverage" get-slot "positive" !
					"result" @ "literalCount" get-slot "literals" !
					"positive" @ "maxPositive" @ >
					if "positive" @ "maxPositive" ! "literals" @ "minLiterals" !
					else
						"positive" @ "maxPositive" @ = "literals" @ "minLiterals" @ < and
						if "literals" @ "minLiterals" ! then
					then
				then
			end
			0 list-of "winners" !
			"candidateResults" @ each
				it "result" !
				"result" @ "eligible" get-slot
				"result" @ "positiveCoverage" get-slot "maxPositive" @ = and
				"result" @ "literalCount" get-slot "minLiterals" @ = and
				if "winners" @ "result" @ list-append "winners" ! then
			end
			"candidateResults" @ "experiment" @ "candidateResultUnits" set-slot
			"winners" @ "experiment" @ "winnerResultUnits" set-slot
			true "experiment" @ "candidatesFinalized" set-slot
			"""#
	},
	{
		name: "AR-H-CloseGuardSearch"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Close the complete evidence-bound barrier and freeze canonical relations"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arClose" =
			"CurUnit" @ "ActionRelationExperiment" isa? and
			"CurUnit" @ "candidatesFinalized" get-slot true = and
			"CurUnit" @ "evidenceRootsReady" get-slot true = and
			"CurUnit" @ "guardSearchClosed" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "experiment" !
			"experiment" @ "candidateResultUnits" get-slot
			"experiment" @ "winnerResultUnits" get-slot
			"experiment" @ "candidateLeafDigests" get-slot
			"experiment" @ "edgeTableRoot" get-slot
			"experiment" @ "evaluationTableRoots" get-slot
			"experiment" @ "winnerLeafDigests" get-slot
			"AR.Barrier." "experiment" @ concat ar-close-guard-search "barrier" !
			"barrier" @ nil !=
			if
				"barrier" @ "experiment" @ "guardSearchBarrier" set-slot
				"barrier" @ "experiment" @ "semanticTrainingRoot" get-slot "AR.Artifact." "experiment" @ concat ar-freeze-relation "artifact" !
				"artifact" @ nil !=
				if
					"artifact" @ "experiment" @ "artifactUnit" set-slot
					"completed" "experiment" @ "terminal" set-slot
					true "experiment" @ "guardSearchClosed" set-slot
				else "no-discovery" "experiment" @ "terminal" set-slot then
			else "no-discovery" "experiment" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-CertifyLocalDiamondInitial"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Execute the two initial arms of one supplied local diamond"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arCertifyInitial" =
			"CurUnit" @ "ActionRelationCertificateRequest" isa? and
			"CurUnit" @ "initialTerminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "state" get-slot "state" !
			"request" @ "aOccurrence" get-slot "a" !
			"request" @ "bOccurrence" get-slot "b" !
			"Cert.App.A." "request" @ concat "aAppRequest" !
			"Cert.App.B." "request" @ concat "bAppRequest" !
			"state" @ "a" @ "aAppRequest" @ ar-applicable? "aApp" !
			"state" @ "b" @ "bAppRequest" @ ar-applicable? "bApp" !
			"state" @ "a" @ "aApp" @ "Cert.Transition.A." "request" @ concat "Cert.State.AfterA." "request" @ concat ar-apply "aResult" !
			"state" @ "b" @ "bApp" @ "Cert.Transition.B." "request" @ concat "Cert.State.AfterB." "request" @ concat ar-apply "bResult" !
			"aResult" @ 0 list-get "aInitial" ! "aResult" @ 1 list-get "afterAUnit" !
			"bResult" @ 0 list-get "bInitial" ! "bResult" @ 1 list-get "afterBUnit" !
			"aInitial" @ "request" @ "aInitial" set-slot
			"bInitial" @ "request" @ "bInitial" set-slot
			"afterAUnit" @ "request" @ "afterAUnit" set-slot
			"afterBUnit" @ "request" @ "afterBUnit" set-slot
			"completed" "request" @ "initialTerminal" set-slot
			"""#
	},
	{
		name: "AR-H-CertifyLocalDiamondCross"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Execute the two crossed arms of one supplied local diamond"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arCertifyCross" =
			"CurUnit" @ "ActionRelationCertificateRequest" isa? and
			"CurUnit" @ "initialTerminal" get-slot "completed" = and
			"CurUnit" @ "crossTerminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "aOccurrence" get-slot "a" !
			"request" @ "bOccurrence" get-slot "b" !
			"request" @ "afterAUnit" get-slot "afterAUnit" !
			"request" @ "afterBUnit" get-slot "afterBUnit" !
			"" "bAfterA" ! "" "aAfterB" ! "" "abUnit" ! "" "baUnit" !
			"afterAUnit" @ "" != "afterBUnit" @ "" != and
			if
				"afterAUnit" @ "state" get-slot "afterA" !
				"afterBUnit" @ "state" get-slot "afterB" !
				"afterA" @ "b" @ "Cert.App.BAfterA." "request" @ concat ar-applicable? "bCrossApp" !
				"afterB" @ "a" @ "Cert.App.AAfterB." "request" @ concat ar-applicable? "aCrossApp" !
				"afterA" @ "b" @ "bCrossApp" @ "Cert.Transition.BAfterA." "request" @ concat "Cert.State.AB." "request" @ concat ar-apply "bCrossResult" !
				"afterB" @ "a" @ "aCrossApp" @ "Cert.Transition.AAfterB." "request" @ concat "Cert.State.BA." "request" @ concat ar-apply "aCrossResult" !
				"bCrossResult" @ 0 list-get "bAfterA" ! "bCrossResult" @ 1 list-get "abUnit" !
				"aCrossResult" @ 0 list-get "aAfterB" ! "aCrossResult" @ 1 list-get "baUnit" !
			then
			"bAfterA" @ "request" @ "bAfterA" set-slot
			"aAfterB" @ "request" @ "aAfterB" set-slot
			"abUnit" @ "request" @ "abUnit" set-slot
			"baUnit" @ "request" @ "baUnit" set-slot
			"completed" "request" @ "crossTerminal" set-slot
			"""#
	},
	{
		name: "AR-H-CertifyLocalDiamondEquality"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Compare the two completed local-diamond arms when both exist"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arCertifyEquality" =
			"CurUnit" @ "ActionRelationCertificateRequest" isa? and
			"CurUnit" @ "crossTerminal" get-slot "completed" = and
			"CurUnit" @ "equalityTerminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "abUnit" get-slot "abUnit" !
			"request" @ "baUnit" get-slot "baUnit" !
			"" "equality" !
			"abUnit" @ "" != "baUnit" @ "" != and
			if
				"abUnit" @ "state" get-slot "abState" !
				"baUnit" @ "state" get-slot "baState" !
				"abState" @ "baState" @ "Cert.Equality." "request" @ concat ar-state-equal? "equality" !
			then
			"equality" @ "request" @ "equality" set-slot
			"completed" "request" @ "equalityTerminal" set-slot
			"""#
	},
	{
		name: "AR-H-CertifyLocalDiamondAssemble"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Assemble one local-diamond attempt after its charged operation root is closed"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arCertifyAssemble" =
			"CurUnit" @ "ActionRelationCertificateRequest" isa? and
			"CurUnit" @ "equalityTerminal" get-slot "completed" = and
			"CurUnit" @ "certificateTerminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "state" get-slot
			"request" @ "aOccurrence" get-slot
			"request" @ "bOccurrence" get-slot
			"request" @ "witness" get-slot
			"request" @ "aInitial" get-slot
			"request" @ "bInitial" get-slot
			"request" @ "bAfterA" get-slot
			"request" @ "aAfterB" get-slot
			"request" @ "equality" get-slot
			"request" @ "aOccurrence" get-slot
			"request" @ "operationRoot" get-slot
			"AR.CertificateAttempt." "request" @ concat ar-certificate-assemble "attempt" !
			"attempt" @ nil !=
			if
				"attempt" @ "request" @ "certificateAttemptUnit" set-slot
				"attempt" @ "certificateUnit" get-slot "request" @ "certificateUnit" set-slot
				"attempt" @ "result" get-slot "request" @ "certificateTerminal" set-slot
			else "failed" "request" @ "certificateTerminal" set-slot then
			"""#
	},
	{
		name: "AR-H-MatchGuardedRelations"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Match every retained relation through explicit applicability, fact, literal, and unanimous-use rows"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arMatch" =
			"CurUnit" @ "ActionRelationMatchRequest" isa? and
			"CurUnit" @ "matchTerminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "state" get-slot "state" !
			"request" @ "aOccurrence" get-slot "a" !
			"request" @ "bOccurrence" get-slot "b" !
			"request" @ "artifactUnit" get-slot "artifact" !
			"state" @ "a" @ "Match.Facts.A." "request" @ concat ar-action-facts "aFacts" !
			"state" @ "b" @ "Match.Facts.B." "request" @ concat ar-action-facts "bFacts" !
			"aFacts" @ "request" @ "aFacts" set-slot
			"bFacts" @ "request" @ "bFacts" set-slot
			"state" @ "a" @ "Match.App.A." "request" @ concat ar-applicable? "aApp" !
			"state" @ "b" @ "Match.App.B." "request" @ concat ar-applicable? "bApp" !
			0 list-of "matchRows" !
			"artifact" @ "relationUnits" get-slot each
				it "relation" !
				"relation" @ "guard" get-slot "guard" !
				"relation" @ "atoms" get-slot "atoms" !
				"relation" @ "polarities" get-slot "polarities" !
				0 list-of "literalRows" !
				"atoms" @ list-length iota each
					it "literalIndex" !
					"Match.Literal." "request" @ concat "." concat "relation" @ concat "." concat "literalIndex" @ concat "literalRequest" !
					"guard" @ "request" @ "aFacts" @ "bFacts" @ "atoms" @ "literalIndex" @ list-get "polarities" @ "literalIndex" @ list-get "literalRequest" @ ar-guard-match "literalRow" !
					"literalRows" @ "literalRow" @ list-append "literalRows" !
				end
				"Match.Relation." "request" @ concat "." concat "relation" @ concat "matchRequest" !
				"relation" @ "state" @ "a" @ "b" @ "aFacts" @ "bFacts" @ "aApp" @ "bApp" @ "literalRows" @ "matchRequest" @ ar-pattern-match "matchRow" !
				"matchRows" @ "matchRow" @ list-append "matchRows" !
			end
			"artifact" @ "matchRows" @ "Match.Barrier." "request" @ concat ar-close-relation-use "barrierResult" !
			"barrierResult" @ nil !=
			if
				"barrierResult" @ 0 list-get "request" @ "useBarrier" set-slot
				"barrierResult" @ 1 list-get "request" @ "matched" set-slot
				"completed" "request" @ "matchTerminal" set-slot
			else "failed" "request" @ "matchTerminal" set-slot then
			"matchRows" @ "request" @ "matchRows" set-slot
			"""#
	},
	{
		name: "AR-H-SearchApplicable"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Evaluate one explicitly supplied search-node occurrence"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arSearchApplicable" =
			"CurUnit" @ "ActionRelationSearchRequest" isa? and
			"CurUnit" @ "terminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "worldDigest" get-slot
			"request" @ "policy" get-slot
			"request" @ "nodeUnit" get-slot
			"request" @ "state" get-slot
			"request" @ "occurrence" get-slot
			"AR.Search.Applicability." "request" @ concat ar-search-applicable? "row" !
			"row" @ nil !=
			if
				"row" @ "request" @ "resultRow" set-slot
				"completed" "request" @ "terminal" set-slot
			else "failed" "request" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-StaticFootprint"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Evaluate one explicitly supplied oriented static footprint pair"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arStaticFootprint" =
			"CurUnit" @ "ActionStaticFootprintRequest" isa? and
			"CurUnit" @ "terminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "state" get-slot "state" !
			"request" @ "aOccurrence" get-slot "a" !
			"request" @ "bOccurrence" get-slot "b" !
			"state" @ "a" @ "AR.Static.Facts.A." "request" @ concat ar-action-facts "aFacts" !
			"state" @ "b" @ "AR.Static.Facts.B." "request" @ concat ar-action-facts "bFacts" !
			"request" @ "worldDigest" get-slot
			"request" @ "nodeUnit" get-slot
			"state" @ "a" @ "b" @ "aFacts" @ "bFacts" @
			"AR.Static.Footprint." "request" @ concat ar-static-footprint? "row" !
			"row" @ nil !=
			if
				"row" @ "request" @ "resultRow" set-slot
				"completed" "request" @ "terminal" set-slot
			else "failed" "request" @ "terminal" set-slot then
			"""#
	},
	{
		name: "AR-H-CertificateCacheFinalize"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Finalize one supplied certificate-cache miss after its proof range closes"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "arCacheFinalize" =
			"CurUnit" @ "ActionCertificateCacheRequest" isa? and
			"CurUnit" @ "terminal" get-slot nil = and
			"""#
		thenCompute: #"""
			"CurUnit" @ "request" !
			"request" @ "worldDigest" get-slot
			"request" @ "policy" get-slot
			"request" @ "state" get-slot
			"request" @ "aOccurrence" get-slot
			"request" @ "bOccurrence" get-slot
			"request" @ "missLookupCallID" get-slot
			"request" @ "attemptUnit" get-slot
			"request" @ "operationRoot" get-slot
			"AR.CertificateCache." "request" @ concat ar-cache-finalize "row" !
			"row" @ nil !=
			if
				"row" @ "request" @ "resultRow" set-slot
				"completed" "request" @ "terminal" set-slot
			else "failed" "request" @ "terminal" set-slot then
			"""#
	},
]
