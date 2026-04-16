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

			"CurUnit" @ "-on-" concat "to" @ concat "specName" !
			"specName" @ unit-exists? not
			if
				"specName" @ "CurUnit" @ "isA" get-slot first create-unit "specUnit" !

				# Copy key slots from parent
				"CurUnit" @ "defn" get-slot nil !=
				if "CurUnit" @ "defn" get-slot "specUnit" @ "defn" set-slot then
				"CurUnit" @ "range" get-slot nil !=
				if "CurUnit" @ "range" get-slot "specUnit" @ "range" set-slot then
				"CurUnit" @ "domain" get-slot nil !=
				if "CurUnit" @ "domain" get-slot "specUnit" @ "domain" set-slot then
				"CurUnit" @ "arity" get-slot nil !=
				if "CurUnit" @ "arity" get-slot "specUnit" @ "arity" set-slot then

				# Apply the specialization
				"specUnit" @ "slot" @ "from" @ "to" @ replace-slot-value drop

				# Set creditors and english
				"H-Specialize" "specUnit" @ "creditors" set-slot
				"Specialized " "CurUnit" @ concat ": " concat "slot" @ concat " " concat "from" @ concat " -> " concat "to" @ concat
				"specUnit" @ "english" set-slot

				600 "specUnit" @ "examples" "Specialized op needs testing" add-task
				"Created specialized: " "specName" @ concat print
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
			"""#
		ifTrulyRelevant: #"""
			"ArgU" @ applics-success-ratio 0.1 >
			"ArgU" @ applics-success-ratio 1.0 <
			and
			"""#
		thenCompute: #"""
			500 "ArgU" @ "generalizations" "Operation has some good results, try generalizing" add-task
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

				# Set creditors and english
				"H18-Generalize" "genUnit" @ "creditors" set-slot
				"Generalized " "CurUnit" @ concat ": " concat "slot" @ concat " " concat "from" @ concat " -> " concat "to" @ concat
				"genUnit" @ "english" set-slot

				500 "genUnit" @ "examples" "Generalized op needs testing" add-task
				"Created generalized: " "genName" @ concat print
			then
			"""#
	},
]
