units: [
	{
		name:    "H6-Specialize"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Specialize a given slot of a given unit"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "specializations" =
			"SlotToChange" get-task-extra nil !=
			and
			"""#
		thenCompute: #"""
			"SlotToChange" get-task-extra "slot" !
			"SpecializeFrom" get-task-extra "from" !
			"SpecializeTo" get-task-extra "to" !

			# Guard: all extras must be non-nil
			"to" @ nil != "from" @ nil != and "slot" @ nil != and
			if
				"CurUnit" @ "-on-" concat "to" @ concat "specName" !
				"specName" @ unit-exists? not
				if
					"specName" @ "CurUnit" @ "isA" get-slot first create-unit "specUnit" !

					# Copy ALL slots from parent unchanged (domain stays as-is)
					"CurUnit" @ "defn" get-slot nil !=
					if "CurUnit" @ "defn" get-slot "specUnit" @ "defn" set-slot then
					"CurUnit" @ "range" get-slot nil !=
					if "CurUnit" @ "range" get-slot "specUnit" @ "range" set-slot then
					"CurUnit" @ "domain" get-slot nil !=
					if "CurUnit" @ "domain" get-slot "specUnit" @ "domain" set-slot then
					"CurUnit" @ "arity" get-slot nil !=
					if "CurUnit" @ "arity" get-slot "specUnit" @ "arity" set-slot then

					# Record the restriction (don't modify domain)
					"to" @ "specUnit" @ "restrictedTo" set-slot

					# Record slot-change provenance for Phase 3 HindSight (H12/H13/H14)
					"specUnit" @ "slot" @ "from" @ "to" @ record-slot-change

					# Set creditors and english
					"H-Specialize" "specUnit" @ "creditors" set-slot
					"Specialized " "CurUnit" @ concat " restricted to " concat "to" @ concat
					"specUnit" @ "english" set-slot

					# Link spec to parent via Specializations (inverse auto-wires
					# Generalizations on the spec). H8 walks the Generalizations
					# chain to propagate applicable arg tuples.
					"specUnit" @ "CurUnit" @ "specializations" add-to-slot

					600 "specUnit" @ "examples" "Specialized op needs testing" add-task
					"Created specialized: " "specName" @ concat print
				then
			then
			"""#
	},
	{
		name:    "H3-RandomSlot"
		worth:   101
		isA: ["Heuristic", "Anything"]
		english: "Randomly choose a slot to specialize"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "specializations" =
			"SlotToChange" get-task-extra nil =
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ criterial-slots random-choice "chosenSlot" !
			"chosenSlot" @ nil !=
			if
				"CurUnit" @ "chosenSlot" @ get-slot "curVal" !
				"curVal" @ nil !=
				if
					"curVal" @ random-choice "fromType" !
					"fromType" @ nil !=
					"fromType" @ "specializations" get-slot nil !=
					and
					if
						"fromType" @ "specializations" get-slot random-choice "toType" !
						600 "CurUnit" @ "chosenSlot" @ "fromType" @ "toType" @ add-spec-task
					then
				then
			then
			"""#
	},
	{
		name:    "H5-Criterial"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Choose criterial slots to specialize"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "specializations" =
			"SlotToChange" get-task-extra nil =
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ criterial-slots random-subset
			each
				it "chosenSlot" !
				"CurUnit" @ "chosenSlot" @ get-slot "curVal" !
				"curVal" @ nil !=
				if
					"curVal" @ random-choice "fromType" !
					"fromType" @ nil !=
					"fromType" @ "specializations" get-slot nil !=
					and
					if
						"fromType" @ "specializations" get-slot
						each
							it "toType" !
							600 "CurUnit" @ "chosenSlot" @ "fromType" @ "toType" @ add-spec-task
						end
					then
				then
			end
			"""#
	},
	{
		name:    "H16-Generalize"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "If operation has some good results, try generalizing"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "domain" get-slot nil !=
			and
			"ArgU" @ "genTaskAdded" get-slot nil =
			and
			"""#
		ifTrulyRelevant: #"""
			"ArgU" @ "-on-" concat "prefix" !
			false "hasResults" !
			"Anything" examples
			each
				it "name" !
				"name" @ "prefix" @ starts-with?
				if
					true "hasResults" !
				then
			end
			"hasResults" @
			"""#
		thenCompute: #"""
			500 "ArgU" @ "generalizations" "Operation has some good results, try generalizing" add-task
			true "ArgU" @ "genTaskAdded" set-slot
			"Generalization task added for " "ArgU" @ concat print
			"""#
	},
	{
		name:    "H17-ChooseGenSlots"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Choose slots to generalize"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "generalizations" =
			"SlotToChange" get-task-extra nil =
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ criterial-slots random-subset
			each
				it "chosenSlot" !
				"CurUnit" @ "chosenSlot" @ get-slot "curVal" !
				"curVal" @ nil !=
				if
					"curVal" @ random-choice "fromType" !
					"fromType" @ nil !=
					"fromType" @ "generalizations" get-slot nil !=
					and
					if
						"fromType" @ "generalizations" get-slot
						each
							it "toType" !
							500 "CurUnit" @ "chosenSlot" @ "fromType" @ "toType" @ add-gen-task
						end
					then
				then
			end
			"""#
	},
	{
		name:    "H18-Generalize"
		worth:   704
		isA: ["Heuristic", "Anything"]
		english: "Generalize a given slot of a given unit"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "generalizations" =
			"SlotToChange" get-task-extra nil !=
			and
			"""#
		thenCompute: #"""
			"SlotToChange" get-task-extra "slot" !
			"GeneralizeFrom" get-task-extra "from" !
			"GeneralizeTo" get-task-extra "to" !

			# Guard: all extras must be non-nil
			"to" @ nil != "from" @ nil != and "slot" @ nil != and
			if
			"CurUnit" @ "-gen-" concat "to" @ concat "genName" !
			"genName" @ unit-exists? not
			if
				"genName" @ "CurUnit" @ "isA" get-slot first create-unit "genUnit" !

				# Copy key slots from parent
				"CurUnit" @ "defn" get-slot nil !=
				if "CurUnit" @ "defn" get-slot "genUnit" @ "defn" set-slot then
				"CurUnit" @ "range" get-slot nil !=
				if "CurUnit" @ "range" get-slot "genUnit" @ "range" set-slot then
				"CurUnit" @ "domain" get-slot nil !=
				if "CurUnit" @ "domain" get-slot "genUnit" @ "domain" set-slot then
				"CurUnit" @ "arity" get-slot nil !=
				if "CurUnit" @ "arity" get-slot "genUnit" @ "arity" set-slot then

				# Apply the generalization
				"genUnit" @ "slot" @ "from" @ "to" @ replace-slot-value drop

				# Record slot-change provenance for Phase 3 HindSight (H12/H13/H14)
				"genUnit" @ "slot" @ "from" @ "to" @ record-slot-change

				# Set creditors and english
				"H18-Generalize" "genUnit" @ "creditors" set-slot
				"Generalized " "CurUnit" @ concat ": " concat "slot" @ concat " " concat "from" @ concat " -> " concat "to" @ concat
				"genUnit" @ "english" set-slot

				500 "genUnit" @ "examples" "Generalized op needs testing" add-task
				"Created generalized: " "genName" @ concat print
			then
			then
			"""#
	},
	{
		name:    "HAvoidIfWorking"
		worth:   700
		isA: ["Heuristic", "HAvoidRule", "HindSightRule", "Anything"]
		english: "If generalizing IfWorkingOnTask, abort 90% of the time — ifWorkingOnTask is the most dangerous slot to mutate"
		overallRecord: {successes: 0, failures: 0}
		ifAboutToWorkOnTask: #"""
			"CurSlot" @ "generalizations" =
			"SlotToChange" get-task-extra "ifWorkingOnTask" =
			and
			if
				10 random-int 0 != if abort then
			then
			false
			"""#
	},
	{
		name:    "H2-KillGarbageCreator"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "If a heuristic creates many mediocre units, punish it and kill its worst children"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Heuristic" isa?
			"ArgU" @ "H2-KillGarbageCreator" !=
			and
			"""#
		ifTrulyRelevant: #"""
			0 "childCount" !
			0 "mediocreCount" !
			"Anything" examples
			each
				it "child" !
				"child" @ "creditors" get-slot nil !=
				if
					"child" @ "creditors" get-slot "ArgU" @ list-contains
					if
						"childCount" @ 1 + "childCount" !
						"child" @ "worth" get-slot 400 <
						if
							"mediocreCount" @ 1 + "mediocreCount" !
						then
					then
				then
			end
			# 5+ children and 80%+ mediocre
			"childCount" @ 5 >=
			"mediocreCount" @ 5 * "childCount" @ 4 * >=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "worth" get-slot 3 / "ArgU" @ "worth" set-slot
			"H2: punished " "ArgU" @ concat " (prolific garbage creator)" concat print
			"Anything" examples
			each
				it "child" !
				"child" @ "creditors" get-slot nil !=
				if
					"child" @ "creditors" get-slot "ArgU" @ list-contains
					"child" @ "worth" get-slot 175 <=
					and
					if
						"child" @ kill-unit
					then
				then
			end
			"""#
	},
	{
		name:    "H19-EliminateDuplicates"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Kill machine-created units whose data duplicates an existing unit"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "creditors" get-slot nil !=
			"ArgU" @ "data" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "data" get-slot "myData" !
			"Set" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "data" get-slot nil !=
				and
				if
					"myData" @ "other" @ "data" get-slot set-equal?
					if
						"other" @ "creditors" get-slot nil =
						if
							"ArgU" @ "worth" get-slot 300 - "ArgU" @ "worth" set-slot
							"Duplicate of seed " "other" @ concat ": " concat "ArgU" @ concat print
						else
							"ArgU" @ "worth" get-slot "other" @ "worth" get-slot <=
							if
								"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
								"Duplicate: " "ArgU" @ concat " = " concat "other" @ concat print
							then
						then
					then
				then
			end
			"""#
	},
	{
		name:    "H24-Seeder"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "After examples of a category are populated, schedule a whyInt task so H24 can search for shared rare predicates"
		overallRecord: {successes: 0, failures: 0}
		ifFinishedWorkingOnTask: #"""
			"CurSlot" @ "examples" =
			if
				"CurUnit" @ "examples" get-slot "exs" !
				"exs" @ nil !=
				if
					"exs" @ list-length 4 >=
					"CurUnit" @ "whyIntScheduled" get-slot nil =
					and
					if
						400 "CurUnit" @ "whyInt" "Check if all examples share a rare predicate" add-task
						true "CurUnit" @ "whyIntScheduled" set-slot
						"H24-Seeder: scheduled whyInt for " "CurUnit" @ concat print
					then
				then
			then
			"""#
	},
	{
		name:    "H24"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "If a category has many examples, see if all satisfy the same rare unary predicate — that's what makes it interesting"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "examples" get-slot nil !=
			if
				"ArgU" @ "examples" get-slot list-length 4 >=
			else
				false
			then
			"""#
		ifWorkingOnTask: #"""
			"CurSlot" @ "whyInt" =
			"CurUnit" @ "examples" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "examples" get-slot "exs" !
			"UnaryPred" examples
			each
				it "pred" !
				# Filter: accept predicate if rarity unknown (never tested) OR freqT ≤ 0.3
				true "rare" !
				"pred" @ "rarity" get-slot "rar" !
				"rar" @ nil !=
				if
					"rar" @ first 0.3 >
					if false "rare" ! then
				then
				# Domain-type check: predicate's domain[0] must be a generalization
				# of the candidate examples' type (or Anything). Otherwise we'd
				# get IsEmpty matching Number via set-size coercion artifacts.
				"pred" @ "domain" get-slot first "predDomain" !
				"rare" @
				"predDomain" @ nil !=
				and
				if
					# Test predicate against every example with data of matching type
					true "allPass" !
					0 "testedCount" !
					"exs" @
					each
						it "ex" !
						"ex" @ unit-exists?
						"ex" @ "data" get-slot nil !=
						and
						"ex" @ "predDomain" @ isa?
						"predDomain" @ "Anything" =
						or
						and
						if
							"ex" @ "data" get-slot "pred" @ apply-op "res" !
							"testedCount" @ 1 + "testedCount" !
							"res" @
							if else false "allPass" ! then
						then
					end
					# Require we actually tested ≥4 examples (skipping data-less / type-mismatched ones)
					"allPass" @ "testedCount" @ 4 >= and
					if
						"pred" @ "ArgU" @ "whyInt" add-to-slot
						"H24: every example of " "ArgU" @ concat " satisfies " concat "pred" @ concat print
					then
				then
			end
			"""#
	},
	{
		name:    "H20"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Run this op on arg tuples recorded on sibling ops' applics — cross-pollination"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
			"ArgU" @ "isA" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			0 "created" !
			"ArgU" @ "isA" get-slot first "cat" !
			"cat" @ examples
			each
				it "sib" !
				"sib" @ "ArgU" @ !=
				"sib" @ "Op" isa?
				and
				"created" @ 3 <
				and
				if
					"sib" @ applics-args
					each
						it "argTup" !
						"created" @ 3 <
						if
							"argTup" @ "-" list-join "argsuffix" !
							"ArgU" @ "-on-" concat "argsuffix" @ concat "rname" !
							"rname" @ unit-exists? not
							if
								"argTup" @ "ArgU" @ apply-op-args "res" !
								"res" @ nil !=
								if
									"rname" @ "ArgU" @ "range" get-slot first create-unit "rUnit" !
									"res" @ "rUnit" @ "data" set-slot
									"H20" "rUnit" @ "creditors" set-slot
									"ArgU" @ "argTup" @ "rUnit" @ record-applic
									"created" @ 1 + "created" !
									"H20 cross-applied " "ArgU" @ concat " on " concat "sib" @ concat "'s args -> " concat "rUnit" @ concat print
								then
							then
						then
					end
				then
			end
			"""#
	},
	{
		name:    "H10"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Find examples of a type by extracting outputs from one op that ranges into it"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "examples" =
			"CurUnit" @ "isRangeOf" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ "isRangeOf" get-slot random-choice "op" !
			"op" @ nil !=
			if
				"op" @ applics-outputs
				each
					it "out" !
					"out" @ unit-exists?
					if "out" @ "CurUnit" @ "examples" add-to-slot then
				end
			then
			"""#
	},
	{
		name:    "H15"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Find examples of a type by extracting outputs from every op that ranges into it"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "examples" =
			"CurUnit" @ "isRangeOf" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ "isRangeOf" get-slot
			each
				it "op" !
				"op" @ applics-outputs
				each
					it "out" !
					"out" @ unit-exists?
					if "out" @ "CurUnit" @ "examples" add-to-slot then
				end
			end
			"""#
	},
	{
		name:    "H22"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "When examples of a unit are found, schedule a task to check which are unusually interesting"
		overallRecord: {successes: 0, failures: 0}
		ifFinishedWorkingOnTask: #"""
			"CurSlot" @ "examples" =
			"CurUnit" @ "interestingness" get-slot nil !=
			"CurUnit" @ "examples" get-slot nil !=
			and and
			if
				500 "CurUnit" @ "intExamples" "Check which examples are unusually interesting" add-task
			then
			"""#
	},
	{
		name:    "H23"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Find interesting examples by applying the unit's Interestingness predicate to each known example"
		overallRecord: {successes: 0, failures: 0}
		ifWorkingOnTask: #"""
			"CurSlot" @ "intExamples" =
			"CurUnit" @ "interestingness" get-slot nil !=
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ "examples" get-slot "exList" !
			"exList" @ nil !=
			if
				"exList" @
				each
					it "ex" !
					"CurUnit" @ "ex" @ is-interesting?
					if
						"ex" @ "CurUnit" @ "intExamples" add-to-slot
						"Interesting: " "ex" @ concat " is in intExamples of " concat "CurUnit" @ concat print
					then
				end
			then
			"""#
	},
	{
		name:    "H4"
		worth:   703
		isA: ["Heuristic", "Anything"]
		english: "If a new unit was synthesized, schedule a task to gather empirical data about it"
		overallRecord: {successes: 0, failures: 0}
		ifFinishedWorkingOnTask: #"""
			new-units
			each
				it "nu" !
				"nu" @ unit-exists?
				if
					"nu" @ "examples" get-slot nil =
					if
						500 "nu" @ "examples" "After synthesis, seek instances" add-task
					then
				then
			end
			"""#
	},
	{
		name:    "H19Criterial"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Kill newly-created units whose criterial slots duplicate an existing unit"
		overallRecord: {successes: 0, failures: 0}
		ifFinishedWorkingOnTask: #"""
			new-units
			each
				it "nu" !
				"nu" @ unit-exists?
				if
					"nu" @ "creditors" get-slot "H-Specialize" list-contains
					"nu" @ "creditors" get-slot "H18-Generalize" list-contains
					or
					"nu" @ "ProtoConjec" isa?
					or not
					"nu" @ criterial-slots "csList" !
					"csList" @ list-length 0 >
					"nu" @ "isA" get-slot nil !=
					and and
					if
						"nu" @ "isA" get-slot first "cat" !
						"cat" @ examples
						each
							it "cand" !
							"cand" @ "nu" @ !=
							"cand" @ unit-exists?
							and
							if
								true "allMatch" !
								"csList" @
								each
									it "cs" !
									"nu" @ "cs" @ get-slot
									"cand" @ "cs" @ get-slot
									=
									if else false "allMatch" ! then
								end
								"allMatch" @
								if
									"Duplicate on criterial slots, killing: " "nu" @ concat " (matches " concat "cand" @ concat ")" concat print
									"nu" @ kill-unit
								then
							then
						end
					then
				then
			end
			"""#
	},
	{
		name:    "H1"
		worth:   800
		isA: ["Heuristic", "Anything"]
		english: "If an op has >=5 applications and >=80% are failures, conjecture high failure rate and enqueue a specialization task"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ 5 applics-bad?
			and
			"""#
		thenCompute: #"""
			"HighFailureRate" "ArgU" @ 1 list-of
				"ArgU" @ " fails on most inputs" concat
				"H1" make-protoconjec drop
			600 "ArgU" @ "specializations" "H1: high failure rate" add-task
			"H1: flagged " "ArgU" @ concat " for specialization" concat print
			"""#
	},
	{
		name:    "H27"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "For an interesting unary predicate, define the category of domain elements that satisfy it"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "UnaryPred" isa?
			"ArgU" @ "worth" get-slot 600 >=
			"ArgU" @ "isAInt" get-slot nil !=
			or
			"ArgU" @ "rarity" get-slot nil !=
			"ArgU" @ "rarity" get-slot first 0.3 <
			and
			or
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot first "srcCat" !
			"srcCat" @ nil !=
			if
				"SatisfyingSetFor-" "ArgU" @ concat "resName" !
				"resName" @ unit-exists? not
				if
					"resName" @ "srcCat" @ "isA" get-slot first create-unit drop
					"srcCat" @ "resName" @ "generalizations" add-to-slot
					"ArgU" @ "resName" @ "defn" set-slot
					"H27" "resName" @ "creditors" set-slot
					"Satisfying " "srcCat" @ concat "s for " concat "ArgU" @ concat
					"resName" @ "english" set-slot

					"srcCat" @ "examples" get-slot
					each
						it "ex" !
						"ex" @ "data" get-slot "ArgU" @ apply-pred
						if
							"ex" @ "resName" @ "examples" add-to-slot
						then
					end

					"resName" @ "examples" get-slot list-length 4 >=
					if
						500 "resName" @ "whyInt" "H27: explore why this predicate set is interesting" add-task
					then

					"H27: created " "resName" @ concat print
				then
			then
			"""#
	},
	{
		name:    "H28"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "For an interesting unary predicate, define the category of domain elements that fail it"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "UnaryPred" isa?
			"ArgU" @ "worth" get-slot 600 >=
			"ArgU" @ "isAInt" get-slot nil !=
			or
			"ArgU" @ "rarity" get-slot nil !=
			"ArgU" @ "rarity" get-slot first 0.3 <
			and
			or
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot first "srcCat" !
			"srcCat" @ nil !=
			if
				"FailingSetFor-" "ArgU" @ concat "resName" !
				"resName" @ unit-exists? not
				if
					"resName" @ "srcCat" @ "isA" get-slot first create-unit drop
					"srcCat" @ "resName" @ "generalizations" add-to-slot
					"ArgU" @ "resName" @ "defn" set-slot
					"H28" "resName" @ "creditors" set-slot
					"Failing " "srcCat" @ concat "s for " concat "ArgU" @ concat
					"resName" @ "english" set-slot

					"srcCat" @ "examples" get-slot
					each
						it "ex" !
						"ex" @ "data" get-slot "ArgU" @ apply-pred not
						if
							"ex" @ "resName" @ "examples" add-to-slot
						then
					end

					"resName" @ "examples" get-slot list-length 4 >=
					if
						500 "resName" @ "whyInt" "H28: explore why this predicate-failing set is interesting" add-task
					then

					"H28: created " "resName" @ concat print
				then
			then
			"""#
	},
	{
		name:    "H-ExercisePreds"
		worth:   400
		isA: ["Heuristic", "Anything"]
		english: "Run each domain-matching unary/binary predicate on a category's examples to populate rarity (one-shot per category)"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "UnaryPred" isa? not
			"ArgU" @ "examples" get-slot nil !=
			and
			"ArgU" @ "examples" get-slot list-length 0 >
			and
			"ArgU" @ "predsExercised" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"UnaryPred" examples
			each
				it "p" !
				"p" @ "domain" get-slot nil !=
				if
					"p" @ "domain" get-slot first "pDom" !
					"ArgU" @ "pDom" @ isa?
					if
						"ArgU" @ "examples" get-slot
						each
							it "ex" !
							"ex" @ "data" get-slot nil !=
							if
								"ex" @ "data" get-slot "p" @ apply-pred drop
							then
						end
						"p" @ "predFocusScheduled" get-slot nil =
						if
							500 "p" @ "whyInt" "H-ExercisePreds: focus pred for H27/H28" add-task
							true "p" @ "predFocusScheduled" set-slot
						then
					then
				then
			end
			"BinaryPred" examples
			each
				it "p" !
				"p" @ "predFocusScheduled" get-slot nil =
				if
					500 "p" @ "whyInt" "H-ExercisePreds: focus binary pred for H25/H26" add-task
					true "p" @ "predFocusScheduled" set-slot
				then
			end
			true "ArgU" @ "predsExercised" set-slot
			"""#
	},
	{
		name:    "H25"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "For an interesting binary predicate, define the set of ordered pairs that satisfy it"
		overallRecord: {successes: 0, failures: 0}
		pairCap: 50
		ifPotentiallyRelevant: #"""
			"ArgU" @ "BinaryPred" isa?
			"ArgU" @ "worth" get-slot 600 >=
			"ArgU" @ "isAInt" get-slot nil !=
			or
			"ArgU" @ "rarity" get-slot nil !=
			"ArgU" @ "rarity" get-slot first 0.3 <
			and
			or
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot "doms" !
			"doms" @ nil !=
			"doms" @ list-length 2 >=
			and
			if
				"doms" @ first "srcA" !
				"doms" @ rest first "srcB" !
				"SatisfyingSetFor-" "ArgU" @ concat "resName" !
				"resName" @ unit-exists? not
				if
					"resName" @ "Set" create-unit drop
					"srcA" @ "resName" @ "generalizations" add-to-slot
					"ArgU" @ "resName" @ "defn" set-slot
					"H25" "resName" @ "creditors" set-slot
					"Satisfying pairs for " "ArgU" @ concat "resName" @ "english" set-slot

					"H25" "pairCap" get-slot "cap" !
					0 "count" !
					"srcA" @ "examples" get-slot
					each
						it "a" !
						"srcB" @ "examples" get-slot
						each
							it "b" !
							"count" @ "cap" @ <
							if
								"a" @ "data" get-slot "b" @ "data" get-slot "ArgU" @ apply-pred
								if
									"OPair-" "a" @ concat "-" concat "b" @ concat "pairName" !
									"pairName" @ unit-exists? not
									if
										"pairName" @ "OPair" create-unit drop
										"a" @ "b" @ 2 list-of "pairName" @ "data" set-slot
									then
									"pairName" @ "resName" @ "examples" add-to-slot
								then
								"count" @ 1 + "count" !
							then
						end
					end

					"resName" @ "examples" get-slot list-length 4 >=
					if
						500 "resName" @ "whyInt" "H25: explore why these pairs satisfy" add-task
					then

					"H25: created " "resName" @ concat print
				then
			then
			"""#
	},
	{
		name:    "H26"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "For an interesting binary predicate, define the set of ordered pairs that fail it"
		overallRecord: {successes: 0, failures: 0}
		pairCap: 50
		ifPotentiallyRelevant: #"""
			"ArgU" @ "BinaryPred" isa?
			"ArgU" @ "worth" get-slot 600 >=
			"ArgU" @ "isAInt" get-slot nil !=
			or
			"ArgU" @ "rarity" get-slot nil !=
			"ArgU" @ "rarity" get-slot first 0.3 <
			and
			or
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot "doms" !
			"doms" @ nil !=
			"doms" @ list-length 2 >=
			and
			if
				"doms" @ first "srcA" !
				"doms" @ rest first "srcB" !
				"FailingSetFor-" "ArgU" @ concat "resName" !
				"resName" @ unit-exists? not
				if
					"resName" @ "Set" create-unit drop
					"srcA" @ "resName" @ "generalizations" add-to-slot
					"ArgU" @ "resName" @ "defn" set-slot
					"H26" "resName" @ "creditors" set-slot
					"Failing pairs for " "ArgU" @ concat "resName" @ "english" set-slot

					"H26" "pairCap" get-slot "cap" !
					0 "count" !
					"srcA" @ "examples" get-slot
					each
						it "a" !
						"srcB" @ "examples" get-slot
						each
							it "b" !
							"count" @ "cap" @ <
							if
								"a" @ "data" get-slot "b" @ "data" get-slot "ArgU" @ apply-pred not
								if
									"OPair-" "a" @ concat "-" concat "b" @ concat "pairName" !
									"pairName" @ unit-exists? not
									if
										"pairName" @ "OPair" create-unit drop
										"a" @ "b" @ 2 list-of "pairName" @ "data" set-slot
									then
									"pairName" @ "resName" @ "examples" add-to-slot
								then
								"count" @ 1 + "count" !
							then
						end
					end

					"resName" @ "examples" get-slot list-length 4 >=
					if
						500 "resName" @ "whyInt" "H26: explore why these pairs fail" add-task
					then

					"H26: created " "resName" @ concat print
				then
			then
			"""#
	},
	{
		name:    "H8"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "Propagate applicable arg tuples from generalizations to this op using type-pred domain checks"
		overallRecord: {successes: 0, failures: 0}
		h8Cap: 3
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
			"ArgU" @ "domain" get-slot nil !=
			and
			"ArgU" @ "generalizations" get-slot nil !=
			and
			"ArgU" @ "h8Ran" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "domain" get-slot "myDomain" !
			"H8" "h8Cap" get-slot "cap" !
			0 "created" !
			"ArgU" @ "generalizations" get-slot
			each
				it "gen" !
				"gen" @ applics-args
				each
					it "argTuple" !
					"created" @ "cap" @ <
					if
						"argTuple" @ list-length "myDomain" @ list-length =
						if
							true "allMatch" !
							0 "i" !
							"argTuple" @
							each
								it "argName" !
								"myDomain" @ "i" @ list-get "expectedType" !
								"argName" @ unit-exists?
								"expectedType" @ nil !=
								and
								if
									"argName" @ "data" get-slot "expectedType" @ apply-pred not
									if false "allMatch" ! then
								else
									false "allMatch" !
								then
								"i" @ 1 + "i" !
							end
							"allMatch" @
							if
								"argTuple" @ "ArgU" @ apply-op-args "result" !
								"result" @ nil !=
								if
									"ArgU" @ "-on-" concat "argTuple" @ "-" list-join concat "resultName" !
									"resultName" @ unit-exists? not
									if
										"resultName" @ "ArgU" @ "range" get-slot first create-unit drop
										"result" @ "resultName" @ "data" set-slot
										"H8" "resultName" @ "creditors" set-slot
										"ArgU" @ "argTuple" @ "resultName" @ record-applic
										"created" @ 1 + "created" !
									then
								then
							then
						then
					then
				end
			end
			true "ArgU" @ "h8Ran" set-slot
			"created" @ 0 >
			if
				"H8: propagated " "created" @ concat " applics to " concat "ArgU" @ concat print
			then
			"""#
	},
	{
		name:    "H-Generate"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "If a unit has a generator and few examples, produce new instance units by running the generator"
		overallRecord: {successes: 0, failures: 0}
		generateCount: 10
		ifPotentiallyRelevant: #"""
			"ArgU" @ "generator" get-slot nil !=
			"ArgU" @ "generated" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"H-Generate" "generateCount" get-slot "cnt" !
			"ArgU" @ "cnt" @ run-generator "vals" !
			"vals" @ list-length 0 >
			if
				0 "i" !
				"vals" @
				each
					it "v" !
					"ArgU" @ "-gen-" concat "i" @ concat "instName" !
					"instName" @ unit-exists? not
					if
						"instName" @ "ArgU" @ create-unit drop
						"v" @ "instName" @ "data" set-slot
						"H-Generate" "instName" @ "creditors" set-slot
						"instName" @ "ArgU" @ "examples" add-to-slot
					then
					"i" @ 1 + "i" !
				end
				"H-Generate: produced " "vals" @ list-length concat " instances for " concat "ArgU" @ concat print
			then
			true "ArgU" @ "generated" set-slot
			"""#
	},
	{
		name:    "H-Transpose"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "Create transposed version of binary ops"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "BinaryOp" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
			"ArgU" @ "transposed" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"ArgU" @ transpose-op
			"newOp" !
			"newOp" @ nil !=
			if
				400 "newOp" @ "examples" "Examples for transposed op" add-task
				"Transposed " "ArgU" @ concat print
			then
			true "ArgU" @ "transposed" set-slot
			"""#
	},
	{
		name:    "H-Compose"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "Compose pairs of ops with matching range/domain"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "defn" get-slot nil !=
			and
			"ArgU" @ "composed" get-slot nil =
			and
			"""#
		thenCompute: #"""
			0 "composeCount" !
			"Op" examples
			each
				it "g" !
				"composeCount" @ 3 <
				if
					"ArgU" @ "g" @ compose-ops
					"newOp" !
					"newOp" @ nil !=
					if
						400 "newOp" @ "examples" "Examples for composed op" add-task
						"Composed " "ArgU" @ concat " . " concat "g" @ concat print
						"composeCount" @ 1 + "composeCount" !
					then
				then
			end
			true "ArgU" @ "composed" set-slot
			"""#
	},
	{
		name:    "H-SemanticDup"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "Kill ops whose observed applics are fully reproduced by a generalization"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Op" isa?
			"ArgU" @ "creditors" get-slot nil !=
			and
			"ArgU" @ "creditors" get-slot "H-Transpose" list-contains
			"ArgU" @ "creditors" get-slot "H-Compose" list-contains
			or
			and
			"ArgU" @ "generalizations" get-slot nil !=
			and
			"ArgU" @ "applics" get-slot nil !=
			and
			"ArgU" @ "applics" get-slot 3 >=
			and
			"ArgU" @ "semDupChecked" get-slot nil =
			and
			"""#
		thenCompute: #"""
			true "ArgU" @ "semDupChecked" set-slot
			true "active" !
			"ArgU" @ "generalizations" get-slot
			each
				it "parent" !
				"active" @
				if
					"ArgU" @ "parent" @ applics-redundant?
					if
						"H-SemanticDup: " "ArgU" @ concat " redundant vs " concat "parent" @ concat " — killing" concat print
						"ArgU" @ kill-unit
						false "active" !
					then
				then
			end
			"""#
	},
]
