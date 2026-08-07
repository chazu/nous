units: [
	{
		name:  "H-RunGraphOps"
		worth: 800
		isA: ["Heuristic", "Anything"]
		english: "Apply build-graph operations to known graph examples"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "GraphOp" isa?
			"ArgU" @ "defn" get-slot nil != and
			"ArgU" @ "explored" get-slot nil = and
			"""#
		thenCompute: #"""
			0 "created" !
			"BuildGraph" examples
			each
				it "left" !
				"left" @ "data" get-slot nil !=
				if
					"BuildGraph" examples
					each
						it "right" !
						"right" @ "data" get-slot nil !=
						"left" @ "right" @ != and
						"created" @ 4 < and
						if
							"left" @ "data" get-slot
							"right" @ "data" get-slot
							"ArgU" @ apply-op "result" !
							"ArgU" @ "-on-" concat "left" @ concat "-" concat "right" @ concat "resultName" !
							"resultName" @ unit-exists? not
							if
								"resultName" @ "BuildGraph" create-unit "resultUnit" !
								"result" @ "resultUnit" @ "data" set-slot
								"H-RunGraphOps" "resultUnit" @ "creditors" set-slot
								"ArgU" @ "left" @ "right" @ 2 list-of "resultUnit" @ record-applic
								"created" @ 1 + "created" !
							then
						then
					end
				then
			end
			true "ArgU" @ "explored" set-slot
			"""#
	},
	{
		name:  "H-ConjectureGraphEquality"
		worth: 700
		isA: ["Heuristic", "Anything"]
		english: "Propose equality conjectures for extensionally equal build graphs"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "BuildGraph" isa?
			"ArgU" @ "data" get-slot nil != and
			"""#
		thenCompute: #"""
			"BuildGraph" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "data" get-slot nil != and
				if
					"ArgU" @ "data" get-slot "other" @ "data" get-slot "SameBuildGraph" apply-op
					if
						"BuildGraphEqual" "ArgU" @ "other" @ 2 list-of
						"ArgU" @ " has the same edges as " concat "other" @ concat
						"H-ConjectureGraphEquality" make-protoconjec drop
					then
				then
			end
			"""#
	},
]
