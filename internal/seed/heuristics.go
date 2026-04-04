package seed

import "github.com/chazu/nous/internal/unit"

// LoadHeuristics loads the seed heuristic rules into the store.
// These are the nous equivalents of EURISKO's H1-H29.
// They are domain-independent — they operate on structural properties
// (isA, domain, range, defn, examples, data) that any domain can provide.
func LoadHeuristics(s *unit.Store) {
	hFindExamples(s)
	hRunOnExamples(s)
	hCheckExtremes(s)
	hSpecialize(s)
	hCheckDomain(s)
	hConjecture(s)
	hExploreSlots(s)
	hKillWorthless(s)
	hPenalizeTrivial(s)
	hBoostInteresting(s)
}

// H-FindExamples: "If working on examples for a concept, collect instances."
func hFindExamples(s *unit.Store) {
	h := putHeuristic(s, "H-FindExamples", 700)
	h.Set("english", "Collect instances of a concept from the store")

	h.Set("ifWorkingOnTask", `"CurSlot" @ "examples" =`)

	h.Set("thenCompute", `
		"CurUnit" @ examples
		"found" !
		"found" @ list-length 0 >
		if
			"found" @ "CurUnit" @ "examples" set-slot
		then
	`)

	h.Set("thenPrintToUser", `
		"Found examples of " "CurUnit" @ concat print
	`)
}

// H-RunOnExamples: "If an operation has a defn and its domain types have
// data slots, run the operation on the data to generate result examples."
func hRunOnExamples(s *unit.Store) {
	h := putHeuristic(s, "H-RunOnExamples", 750)
	h.Set("english", "Run operations on concrete data to generate examples")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Op" isa?
		"ArgU" @ "defn" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
		# Track how many results we've created (cap at 5 per firing)
		0 "created" !
		"ArgU" @ "domain" get-slot first "domType1" !

		# Collect data-bearing sources (up to 4)
		"domType1" @ examples
		each
			it "src1" !
			"src1" @ "data" get-slot nil !=
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
									"resultName" @ "Set" create-unit
									"resultUnit" !
									"result" @ "resultUnit" @ "data" set-slot
									"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
									"created" @ 1 + "created" !
									"Applied " "ArgU" @ concat ": " concat "src1" @ concat " x " concat "src2" @ concat print
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
							"resultName" @ "Set" create-unit
							"resultUnit" !
							"result" @ "resultUnit" @ "data" set-slot
							"H-RunOnExamples" "resultUnit" @ "creditors" set-slot
							"created" @ 1 + "created" !
							"Applied " "ArgU" @ concat ": " concat "src1" @ concat print
						then
					then
				then
			then
		end
	`)
}

// H-CheckExtremes: "When a set has data, check extreme elements."
func hCheckExtremes(s *unit.Store) {
	h := putHeuristic(s, "H-CheckExtremes", 600)
	h.Set("english", "Examine extreme cases of sets")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Set" isa?
		"ArgU" @ "data" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
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
	`)
}

// H-Specialize: "If an operation has a broad domain, try restricting it."
func hSpecialize(s *unit.Store) {
	h := putHeuristic(s, "H-Specialize", 650)
	h.Set("english", "Specialize operations by narrowing domain types")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Op" isa?
		"ArgU" @ "domain" get-slot nil !=
		and
		"ArgU" @ "defn" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
		"ArgU" @ "domain" get-slot
		each
			it "domType" !
			"domType" @ "specializations" get-slot
			each
				it "specType" !
				"ArgU" @ "-on-" concat "specType" @ concat
				"specName" !
				"specName" @ unit-exists? not
				if
					"specName" @ "ArgU" @ "isA" get-slot first create-unit
					"specUnit" !
					"ArgU" @ "defn" get-slot "specUnit" @ "defn" set-slot
					"H-Specialize" "specUnit" @ "creditors" set-slot
					"ArgU" @ "range" get-slot "specUnit" @ "range" set-slot
					"specType" @ "specUnit" @ "domain" set-slot
					600 "specUnit" @ "examples" "Specialized op needs testing" add-task
					"Specialized " "ArgU" @ concat " -> " concat "specName" @ concat print
				then
			end
		end
	`)
}

// H-CheckDomain: "If domain and range types overlap, create self-composition."
func hCheckDomain(s *unit.Store) {
	h := putHeuristic(s, "H-CheckDomain", 550)
	h.Set("english", "If domain/range overlap, create self-composition")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Op" isa?
		"ArgU" @ "domain" get-slot nil !=
		and
		"ArgU" @ "range" get-slot nil !=
		and
		"ArgU" @ "creditors" get-slot nil =
		and
	`)

	h.Set("thenCompute", `
		"ArgU" @ "range" get-slot
		each
			it "rangeType" !
			"ArgU" @ "domain" get-slot
			each
				it "domType" !
				"domType" @ "rangeType" @ =
				if
					"SelfCompose-" "ArgU" @ pack-name
					"composeName" !
					"composeName" @ unit-exists? not
					if
						"composeName" @ "BinaryOp" create-unit
						"compUnit" !
						"H-CheckDomain" "compUnit" @ "creditors" set-slot
						"ArgU" @ "domain" get-slot "compUnit" @ "domain" set-slot
						"ArgU" @ "range" get-slot "compUnit" @ "range" set-slot
						600 "compUnit" @ "examples" "Self-composition needs examples" add-task
						"Created self-composition: " "composeName" @ concat print
					then
				then
			end
		end
	`)
}

// H-Conjecture: "Compare data across sets to find equal or subset relationships."
func hConjecture(s *unit.Store) {
	h := putHeuristic(s, "H-Conjecture", 700)
	h.Set("english", "Compare sets to find equalities and subset relationships")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Set" isa?
		"ArgU" @ "data" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
		"Set" examples
		each
			it "other" !
			"other" @ "ArgU" @ !=
			"other" @ "data" get-slot nil !=
			and
			if
				"ArgU" @ "data" get-slot
				"other" @ "data" get-slot
				set-equal?
				if
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
					"CONJECTURE: " "ArgU" @ concat " ⊂ " concat "other" @ concat print
				then
			then
		end
	`)
}

// H-ExploreSlots: "Add tasks to fill empty important slots."
func hExploreSlots(s *unit.Store) {
	h := putHeuristic(s, "H-ExploreSlots", 500)
	h.Set("english", "Add tasks to explore empty important slots")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Heuristic" isa? not
		"ArgU" @ "Slot" isa? not
		and
		"ArgU" @ "explored" get-slot nil =
		and
	`)

	h.Set("thenCompute", `
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
	`)
}

// H-KillWorthless: "Kill machine-created units with very low Worth."
func hKillWorthless(s *unit.Store) {
	h := putHeuristic(s, "H-KillWorthless", 800)
	h.Set("english", "Kill units with very low Worth that were machine-created")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "worth" get-slot 100 <
		"ArgU" @ "creditors" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
		"Killing worthless unit: " "ArgU" @ concat print
		"ArgU" @ kill-unit
	`)
}

// H-BoostInteresting: "Boost worth of operations producing surprising results."
func hBoostInteresting(s *unit.Store) {
	h := putHeuristic(s, "H-BoostInteresting", 650)
	h.Set("english", "Boost worth of operations that produce surprising results")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "creditors" get-slot nil !=
		"ArgU" @ "data" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
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
	`)
}

// H-PenalizeTrivial: "If a result unit's data is empty or identical to its
// input, it's trivial — lower its worth."
func hPenalizeTrivial(s *unit.Store) {
	h := putHeuristic(s, "H-PenalizeTrivial", 600)
	h.Set("english", "Penalize machine-created units with trivial (empty) data")

	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "creditors" get-slot nil !=
		"ArgU" @ "data" get-slot nil !=
		and
	`)

	h.Set("thenCompute", `
		"ArgU" @ "data" get-slot set-size 0 =
		if
			# Empty result — likely trivial (e.g., intersect with empty set)
			"ArgU" @ "worth" get-slot 200 - "ArgU" @ "worth" set-slot
			"Trivial (empty): " "ArgU" @ concat print
		then
	`)
}

func putHeuristic(s *unit.Store, name string, worth int) *unit.Unit {
	h := unit.New(name)
	h.SetWorth(worth)
	h.Set("isA", []string{"Heuristic", "Anything"})
	h.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	s.Put(h)
	return h
}
