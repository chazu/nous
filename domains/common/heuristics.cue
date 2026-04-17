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
]
