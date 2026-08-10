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
			false true 2 list-of "polarities" !
			0 list-of "oneLiteralCandidates" !
			15 iota each
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
			15 iota each
				it "leftIndex" !
				15 iota each
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
			"ordinal" @ 450 = "candidates" @ list-length 451 = and "edges" @ list-length 450 = and
			if true "experiment" @ "guardSpaceAllocated" set-slot
			else "no-discovery" "experiment" @ "terminal" set-slot then
			"""#
	},
]
