package seed

import "github.com/chazu/nous/internal/unit"

// LoadObservationDomain sets up the type hierarchy for Mode 2 reasoning
// over pudl observations. This domain doesn't load concrete data — that
// comes from the pudl bridge at runtime. It loads types and heuristics.
func LoadObservationDomain(s *unit.Store) {
	// Type hierarchy
	putType(s, "Observation", 500, []string{"Anything"})
	putType(s, "DerivedFact", 400, []string{"Anything"})
	putType(s, "Conjecture", 600, []string{"Anything"})
	putType(s, "RepoHotspot", 500, []string{"DerivedFact", "Anything"})
}

// LoadObservationHeuristics loads Mode 2 heuristics for reasoning over observations.
func LoadObservationHeuristics(s *unit.Store) {
	hFindRepoHotspots(s)
	hCorroborateObstacles(s)
	hConjectureFromPatterns(s)
	hBoostCorroborated(s)
	hPenalizeStaleObservations(s)
}

// H-FindRepoHotspots: "If multiple observations target the same repo, that repo
// is a hotspot. Create a RepoHotspot unit."
func hFindRepoHotspots(s *unit.Store) {
	h := putHeuristic(s, "H-FindRepoHotspots", 700)
	h.Set("english", "Find repos with multiple observations and flag them as hotspots")

	// Fires when working on any Observation unit
	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Observation" isa?
	`)

	h.Set("thenCompute", `
		# Get this observation's repo
		"ArgU" @ "repo" get-slot "thisRepo" !
		"thisRepo" @ nil =
		if
			# No repo set, skip
		else
			# Count observations with same repo
			0 "count" !
			"Observation" examples
			each
				it "repo" get-slot "thisRepo" @ =
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
					"hsName" @ "RepoHotspot" create-unit
					"hsName" @ "repo" "thisRepo" @ set-slot
					"hsName" @ "observation_count" "count" @ set-slot
					"hsName" @ "worth" 600 set-slot
				else
					# Update count on existing hotspot
					"hsName" @ "observation_count" "count" @ set-slot
				then
			then
		then
	`)

	h.Set("thenPrintToUser", `
		"thisRepo" @ nil !=
		"count" @ 2 >=
		and
		if
			"Repo hotspot: " "thisRepo" @ concat " (" concat "count" @ concat " observations)" concat print
		then
	`)
}

// H-CorroborateObstacles: "If multiple sources reported the same obstacle kind
// for the same repo, boost its worth."
func hCorroborateObstacles(s *unit.Store) {
	h := putHeuristic(s, "H-CorroborateObstacles", 650)
	h.Set("english", "Boost worth of obstacles corroborated by multiple sources")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Observation" isa?
		"ArgU" @ "kind" get-slot "obstacle" =
		and
	`)

	h.Set("thenCompute", `
		"ArgU" @ "repo" get-slot "thisRepo" !
		"ArgU" @ "source" get-slot "thisSrc" !
		0 "otherSources" !

		"Observation" examples
		each
			it "other" !
			"other" @ "ArgU" @ !=
			"other" @ "kind" get-slot "obstacle" =
			and
			"other" @ "repo" get-slot "thisRepo" @ =
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
	`)

	h.Set("thenPrintToUser", `
		"otherSources" @ 0 >
		if
			"ArgU" @ " corroborated by " concat "otherSources" @ concat " other source(s)" concat print
		then
	`)
}

// H-ConjectureFromPatterns: "If we see the same kind of observation across
// multiple repos, conjecture that it's a systemic issue."
func hConjectureFromPatterns(s *unit.Store) {
	h := putHeuristic(s, "H-ConjectureFromPatterns", 600)
	h.Set("english", "When the same observation kind appears across repos, propose a systemic conjecture")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Observation" isa?
	`)

	h.Set("thenCompute", `
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
	`)

	h.Set("thenPrintToUser", `
		"repoCount" @ 3 >=
		if
			"Systemic " "thisKind" @ concat " across " concat "repoCount" @ concat " repos" concat print
		then
	`)
}

// H-BoostCorroborated: "Observations with multiple sources are worth more."
func hBoostCorroborated(s *unit.Store) {
	h := putHeuristic(s, "H-BoostCorroborated", 500)
	h.Set("english", "Boost worth of observations corroborated by any kind, not just obstacles")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Observation" isa?
	`)

	h.Set("thenCompute", `
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
	`)
}

// H-PenalizeStaleObservations: "Observations that haven't been corroborated
// or acted on lose worth over time."
func hPenalizeStaleObservations(s *unit.Store) {
	h := putHeuristic(s, "H-PenalizeStaleObservations", 400)
	h.Set("english", "Decay uncorroborated observations")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Observation" isa?
	`)

	h.Set("thenCompute", `
		"ArgU" @ "status" get-slot "raw" =
		if
			# Still raw — slight worth decay
			"ArgU" @ "worth" get-slot 10 - 0 max
			"ArgU" @ "worth" rot set-slot
		then
	`)
}

func putType(s *unit.Store, name string, worth int, isA []string) {
	u := unit.New(name)
	u.Set("isA", isA)
	u.Set("worth", worth)
	s.Put(u)
}
