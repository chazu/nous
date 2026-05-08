units: [
	{
		name:    "H-FindExamples"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Collect instances of a concept from the store"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "examples" =
			"""#
		thenCompute: #"""
			"CurUnit" @ examples
			"found" !
			"found" @ list-length 0 >
			if
				"found" @ "CurUnit" @ "examples" set-slot
			then
			"""#
		thenPrintToUser: #"""
			"Found examples of " "CurUnit" @ concat print
			"""#
	},
	{
		name:    "H-RunOnExamples"
		worth:   750
		isA: ["Heuristic", "Anything"]
		english: "Run operations on concrete data to generate examples"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			# Track how many results we've created (cap at 5 per firing)
			0 "created" !
			"ArgU" @ "domain" get-slot first "domType1" !

			# Collect data-bearing sources (up to 4)
			"domType1" @ examples
			each
				it "src1" !
				"src1" @ "data" get-slot nil !=
				"src1" @ "creditors" get-slot nil =
				and
				"created" @ 5 <
				and
				if
					# Binary ops: pair with one other source
					"ArgU" @ "BinaryOp" isa?
					if
						"domType1" @ examples
						each
							it "src2" !
							"src2" @ "data" get-slot nil !=
							"src2" @ "creditors" get-slot nil =
							and
							"src1" @ "src2" @ !=
							and
							"created" @ 5 <
							and
							if
								"src1" @ "data" get-slot
								"src2" @ "data" get-slot
								"ArgU" @ apply-op
								"result" !
								"result" @ nil !=
								if
									"ArgU" @ "-on-" concat "src1" @ concat "-" concat "src2" @ concat
									"resultName" !
									"resultName" @ unit-exists? not
									if
										# Use the Op's range first element as the result parent so
										# Number-valued ops (GCD) create Number units and Set-valued
										# ops (SetUnion, DivisorsOf) create Set units. Without this,
										# every result is typed as Set and H-CheckExtremes treats
										# int data as an empty set (AsList(int) = nil).
										"resultName" @ "ArgU" @ "range" get-slot first create-unit
										"resultUnit" !
										"result" @ "resultUnit" @ "data" set-slot
										"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
										"created" @ 1 + "created" !
										# Phase 7.3: record full I/O applic on the op so H8/H10/H15/H20
										# can read args and outputs later.
										"ArgU" @ "src1" @ "src2" @ 2 list-of "resultUnit" @ record-applic
										"Applied " "ArgU" @ concat ": " concat "src1" @ concat " x " concat "src2" @ concat print
										# Schedule examination so H-CheckExtremes/H-Conjecture/
										# H-BoostInteresting can inspect data without waiting for
										# unit-focus (which worth-500 results rarely win).
										300 "resultUnit" @ "data" "Examine new application result" add-task
									then
								then
							then
						end
					then

					# Unary ops
					"ArgU" @ "UnaryOp" isa?
					"created" @ 5 <
					and
					if
						"src1" @ "data" get-slot
						"ArgU" @ apply-op
						"result" !
						"result" @ nil !=
						if
							"ArgU" @ "-on-" concat "src1" @ concat
							"resultName" !
							"resultName" @ unit-exists? not
							if
								# Use range for result type (see binary branch note).
								"resultName" @ "ArgU" @ "range" get-slot first create-unit
								"resultUnit" !
								"result" @ "resultUnit" @ "data" set-slot
								"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
								"created" @ 1 + "created" !
								# Phase 7.3: unary applic with single arg.
								"ArgU" @ "src1" @ 1 list-of "resultUnit" @ record-applic
								"Applied " "ArgU" @ concat ": " concat "src1" @ concat print
								300 "resultUnit" @ "data" "Examine new application result" add-task
							then
						then
					then
				then
			end
			"""#
	},
	{
		name:    "H-CheckExtremes"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Examine extreme cases of sets"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "data" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot "theData" !

			"theData" @ set-size 0 =
			if
				"ArgU" @ " is empty" concat print
			then

			"theData" @ set-size 1 =
			if
				"ArgU" @ " is a singleton: {" concat "theData" @ first concat "}" concat print
				"ArgU" @ "worth" get-slot 700 <
				if
					"ArgU" @ "worth" get-slot 100 + "ArgU" @ "worth" set-slot
				then
			then
			"""#
	},
	{
		name:    "H-Specialize"
		worth:   650
		isA: ["Heuristic", "Anything"]
		english: "Specialize operations by narrowing domain types"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil !=
			and
			"ArgU" @ "defn" get-slot nil !=
			and
			"ArgU" @ "restrictedTo" get-slot nil =
			and
			"ArgU" @ "specTaskAdded" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot
			each
				it "domType" !
				"domType" @ "specializations" get-slot nil !=
				if
					"domType" @ "specializations" get-slot
					each
						it "specType" !
						"ArgU" @ "-on-" concat "specType" @ concat
						unit-exists? not
						if
							600 "ArgU" @ "domain" "domType" @ "specType" @ add-spec-task
							"Specialization task: " "ArgU" @ concat " domain " concat "domType" @ concat " -> " concat "specType" @ concat print
						then
					end
				then
			end
			true "ArgU" @ "specTaskAdded" set-slot
			"""#
	},
	{
		name:    "H-Conjecture"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Compare sets to find equalities and subset relationships"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "data" get-slot nil !=
			and
			"ArgU" @ "data" get-slot is-list?
			"ArgU" @ "data" get-slot list-length 1 >=
			and and
			"ArgU" @ "OPair-False" starts-with? not
			"ArgU" @ "OPair-True" starts-with? not
			and and
			"ArgU" @ "OPair-Bag-ex" starts-with? not
			"ArgU" @ "OPair-BagOfTallies" starts-with? not
			and and
			"ArgU" @ "BestSubset-on-OPair" starts-with? not
			"ArgU" @ "GoodSubset-on-OPair" starts-with? not
			and and
			"""#
		thenCompute: #"""
			"Set" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "data" get-slot nil !=
				and
				# Quality gate: skip comparisons with trivially-named units
				# (type-error results like OPair-False-True, Bag-ex-tally-*)
				"other" @ "OPair-False" starts-with? not
				"other" @ "OPair-True" starts-with? not
				and and
				"other" @ "OPair-Bag-ex" starts-with? not
				"other" @ "OPair-BagOfTallies" starts-with? not
				and and
				# Both sides must have non-empty set data
				"ArgU" @ "data" get-slot is-list?
				"ArgU" @ "data" get-slot list-length 1 >=
				and
				"other" @ "data" get-slot is-list?
				"other" @ "data" get-slot list-length 1 >=
				and
				and and
				if
					"ArgU" @ "data" get-slot
					"other" @ "data" get-slot
					set-equal?
					if
						"SetEqual" "ArgU" @ "other" @ 2 list-of
							"ArgU" @ " = " concat "other" @ concat
							"H-Conjecture" make-protoconjec drop
						"CONJECTURE: " "ArgU" @ concat " = " concat "other" @ concat print
						# If ArgU is machine-created and other is not, ArgU is redundant
						"ArgU" @ "creditors" get-slot nil !=
						"other" @ "creditors" get-slot nil =
						and
						if
							# Penalize the redundant copy
							"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
							"Penalized redundant " "ArgU" @ concat " (= " concat "other" @ concat ")" concat print
						then
						# If both are machine-created, penalize the one with lower worth
						"ArgU" @ "creditors" get-slot nil !=
						"other" @ "creditors" get-slot nil !=
						and
						if
							"ArgU" @ "worth" get-slot "other" @ "worth" get-slot <=
							if
								"ArgU" @ "worth" get-slot 150 - "ArgU" @ "worth" set-slot
							then
						then
					then

					"ArgU" @ "data" get-slot
					"other" @ "data" get-slot
					set-subset?
					"ArgU" @ "data" get-slot "other" @ "data" get-slot set-equal? not
					and
					if
						"SubsetOf" "ArgU" @ "other" @ 2 list-of
							"ArgU" @ " ⊂ " concat "other" @ concat
							"H-Conjecture" make-protoconjec drop
						"CONJECTURE: " "ArgU" @ concat " ⊂ " concat "other" @ concat print
					then
				then
			end
			"""#
	},
	{
		name:    "H-ExploreSlots"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "Add tasks to explore empty important slots"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Heuristic" isa? not
			"ArgU" @ "Slot" isa? not
			and
			"ArgU" @ "explored" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "examples" get-slot nil =
			if
				400 "ArgU" @ "examples" "Unit needs examples" add-task
			then
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil =
			and
			if
				350 "ArgU" @ "domain" "Operation needs domain defined" add-task
			then
			true "ArgU" @ "explored" set-slot
			"""#
	},
	{
		name:    "H-KillWorthless"
		worth:   800
		isA: ["Heuristic", "Anything"]
		english: "Kill units with very low Worth that were machine-created"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "worth" get-slot 100 <
			"ArgU" @ "creditors" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"Killing worthless unit: " "ArgU" @ concat print
			"ArgU" @ kill-unit
			"""#
	},
	{
		name:    "H-BoostInteresting"
		worth:   650
		isA: ["Heuristic", "Anything"]
		english: "Boost worth of operations that produce surprising results"
		overallRecord: {successes: 0, failures: 0}
		// Gated on Set: set-size operates on lists, so Number-typed data (ints)
		// would report size 0 and trigger the false-empty branch.
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "creditors" get-slot nil !=
			and
			"ArgU" @ "data" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot set-size 0 =
			if
				"Interesting: " "ArgU" @ concat " produced empty result" concat print
				"ArgU" @ "creditors" get-slot
				each
					it "cred" !
					"cred" @ "worth" get-slot 50 + "cred" @ "worth" set-slot
				end
			then

			"ArgU" @ "data" get-slot set-size 1 =
			if
				"Interesting: " "ArgU" @ concat " is singleton {" concat "ArgU" @ "data" get-slot first concat "}" concat print
				"ArgU" @ "creditors" get-slot
				each
					it "cred" !
					"cred" @ "worth" get-slot 75 + "cred" @ "worth" set-slot
				end
			then
			"""#
	},
	{
		name:    "H-PenalizeTrivial"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Penalize machine-created units with trivial (empty) data"
		overallRecord: {successes: 0, failures: 0}
		// Gated on Set: set-size on a scalar returns 0, which would penalize
		// every Number-typed result as trivially empty.
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Set" isa?
			"ArgU" @ "creditors" get-slot nil !=
			and
			"ArgU" @ "data" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot set-size 0 =
			if
				# Empty result — likely trivial (e.g., intersect with empty set)
				"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
				"Trivial (empty): " "ArgU" @ concat print
			then
			"""#
	},
	{
		name:    "H-AnalyzeApplics"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Inspect applics for type-skewed success patterns and propose specializations"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Heuristic" isa?
			"ArgU" @ "H-AnalyzeApplics" !=
			and
			"""#
		ifTrulyRelevant: #"""
			"ArgU" @ applics-success-ratio "ratio" !
			"ratio" @ 0.3 >=
			"ratio" @ 0.7 <=
			and
			"ArgU" @ get-applics nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ analyze-and-specialize
			"""#
	},
]
