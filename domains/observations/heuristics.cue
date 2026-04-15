units: [
	{
		name:    "H-FindScopeHotspots"
		worth:   700
		isA: ["Heuristic", "Anything"]
		english: "Find scopes with multiple observations and flag them as hotspots"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Observation" isa?
			"""#
		thenCompute: #"""
			# Get this observation's repo
			"ArgU" @ "scope" get-slot "thisRepo" !
			"thisRepo" @ nil =
			if
				# No repo set, skip
			else
				# Count observations with same repo
				0 "count" !
				"Observation" examples
				each
					it "scope" get-slot "thisRepo" @ =
					if
						"count" @ 1 + "count" !
					then
				end

				# If 2+ observations share this repo, create hotspot
				"count" @ 2 >=
				if
					"Hotspot-" "thisRepo" @ concat "hsName" !
					"hsName" @ unit-exists? not
					if
						"hsName" @ "ScopeHotspot" create-unit
						"hsName" @ "scope" "thisRepo" @ set-slot
						"hsName" @ "observation_count" "count" @ set-slot
						"hsName" @ "worth" 600 set-slot
					else
						# Update count on existing hotspot
						"hsName" @ "observation_count" "count" @ set-slot
					then
				then
			then
			"""#
		thenPrintToUser: #"""
			"thisRepo" @ nil !=
			"count" @ 2 >=
			and
			if
				"Repo hotspot: " "thisRepo" @ concat " (" concat "count" @ concat " observations)" concat print
			then
			"""#
	},
	{
		name:    "H-CorroborateObstacles"
		worth:   650
		isA: ["Heuristic", "Anything"]
		english: "Boost worth of obstacles corroborated by multiple sources"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Observation" isa?
			"ArgU" @ "kind" get-slot "obstacle" =
			and
			"""#
		thenCompute: #"""
			"ArgU" @ "scope" get-slot "thisRepo" !
			"ArgU" @ "source" get-slot "thisSrc" !
			0 "otherSources" !

			"Observation" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "kind" get-slot "obstacle" =
				and
				"other" @ "scope" get-slot "thisRepo" @ =
				and
				"other" @ "source" get-slot "thisSrc" @ !=
				and
				if
					"otherSources" @ 1 + "otherSources" !
				then
			end

			# Corroborated: boost worth
			"otherSources" @ 0 >
			if
				"ArgU" @ "worth" get-slot 100 + 1000 min
				"ArgU" @ "worth" rot set-slot
			then
			"""#
		thenPrintToUser: #"""
			"otherSources" @ 0 >
			if
				"ArgU" @ " corroborated by " concat "otherSources" @ concat " other source(s)" concat print
			then
			"""#
	},
	{
		name:    "H-ConjectureFromPatterns"
		worth:   600
		isA: ["Heuristic", "Anything"]
		english: "When the same observation kind appears across scopes, propose a systemic conjecture"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Observation" isa?
			"""#
		thenCompute: #"""
			"ArgU" @ "kind" get-slot "thisKind" !
			"ArgU" @ "description" get-slot "thisDesc" !

			# Count distinct repos with same kind
			0 "repoCount" !
			"Observation" examples
			each
				it "kind" get-slot "thisKind" @ =
				if
					"repoCount" @ 1 + "repoCount" !
				then
			end

			# 3+ repos with same kind = systemic
			"repoCount" @ 3 >=
			if
				"Conjecture-systemic-" "thisKind" @ concat "cjName" !
				"cjName" @ unit-exists? not
				if
					"cjName" @ "Conjecture" create-unit
					"cjName" @ "kind" "thisKind" @ set-slot
					"cjName" @ "observation_count" "repoCount" @ set-slot
					"cjName" @ "english" "Systemic issue: " "thisKind" @ concat " observed across " concat "repoCount" @ concat " repos" concat set-slot
					"cjName" @ "worth" 700 set-slot
				then
			then
			"""#
		thenPrintToUser: #"""
			"repoCount" @ 3 >=
			if
				"Systemic " "thisKind" @ concat " across " concat "repoCount" @ concat " repos" concat print
			then
			"""#
	},
	{
		name:    "H-BoostCorroborated"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "Boost worth of observations corroborated by any kind, not just obstacles"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Observation" isa?
			"""#
		thenCompute: #"""
			"ArgU" @ "description" get-slot "thisDesc" !
			"ArgU" @ "source" get-slot "thisSrc" !
			0 "corroborators" !

			"Observation" examples
			each
				it "other" !
				"other" @ "ArgU" @ !=
				"other" @ "description" get-slot "thisDesc" @ =
				and
				"other" @ "source" get-slot "thisSrc" @ !=
				and
				if
					"corroborators" @ 1 + "corroborators" !
				then
			end

			"corroborators" @ 0 >
			if
				"ArgU" @ "worth" get-slot 50 + 1000 min
				"ArgU" @ "worth" rot set-slot
			then
			"""#
	},
	{
		name:    "H-PenalizeStaleObservations"
		worth:   400
		isA: ["Heuristic", "Anything"]
		english: "Decay uncorroborated observations"
		overallRecord: {successes: 0, failures: 0}
		ifPotentiallyRelevant: #"""
			"ArgU" @ "Observation" isa?
			"""#
		thenCompute: #"""
			"ArgU" @ "status" get-slot "raw" =
			if
				# Still raw — slight worth decay
				"ArgU" @ "worth" get-slot 10 - 0 max
				"ArgU" @ "worth" rot set-slot
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
