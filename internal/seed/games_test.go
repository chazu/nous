package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	gamevocab "github.com/chazu/nous/internal/vocab/game"
)

const gameTestCycles = 500

func loadGames(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "games"); err != nil {
		t.Fatal(err)
	}
	return store
}

func runGamesStore(t *testing.T, store *unit.Store, mutate bool) (*unit.Store, *engine.Engine, *agenda.Agenda) {
	t.Helper()
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = gameTestCycles
	eng.MutConfig.Enabled = mutate
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store, eng, ag
}

func gameComplete(t *testing.T, eng *engine.Engine) bool {
	t.Helper()
	value, err := eng.VM.Execute(`"MemoryOnePDProfileA" game-experiment-complete?`)
	if err != nil || value.Kind() != dsl.VBool {
		t.Fatalf("completion = (%v,%v)", value, err)
	}
	return value.AsBool()
}

func TestGamesVocabularyCompletesPreregisteredMatrix(t *testing.T) {
	store, eng, ag := runGamesStore(t, loadGames(t), false)
	if !gameComplete(t, eng) || ag.Len() != 0 {
		t.Fatalf("complete/agenda = %v/%d", gameComplete(t, eng), ag.Len())
	}
	experiment := store.Get("MemoryOnePDProfileA")
	if len(experiment.GetStrings("candidateUnits")) != 32 || experiment.GetInt("evaluatedCandidateCount") != 32 {
		t.Fatalf("candidate state = %d/%d", len(experiment.GetStrings("candidateUnits")), experiment.GetInt("evaluatedCandidateCount"))
	}
	counts := gameArtifactCounts(store)
	if counts["candidate"] != 32 || counts["result"] != 192 || counts["observation"] != 192 || counts["evidence"] != 32 || counts["selection"] != 1 || counts["schema"] != 14 || counts["conjecture"] != 14 {
		t.Fatalf("artifact counts = %v", counts)
	}
	wantFrontier := []string{"DDDCC", "DCDDC", "DCDCC", "DCCDD", "DCCDC", "DCCCD", "DCCCC", "CDDDC", "CDDCD", "CCDDD", "CCDCD", "CCDCC", "CCCCD", "CCCCC"}
	wantScalar := []string{"DDDDD", "DDCDD", "DCDDD", "DCCDD"}
	if got := gameActions(store, experiment.GetStrings("frontierUnits")); !reflect.DeepEqual(got, wantFrontier) {
		t.Fatalf("frontier = %v", got)
	}
	if got := gameActions(store, experiment.GetStrings("scalarLeaderUnits")); !reflect.DeepEqual(got, wantScalar) {
		t.Fatalf("scalar = %v", got)
	}
	for _, hidden := range []string{"Pavlov", "RepeatOwnAction", "SuspiciousTFT", "GameHeldOutCase"} {
		if store.Has(hidden) {
			t.Fatalf("training store contains held-out fixture label %s", hidden)
		}
	}
}

func TestGameVocabularyWordsAreScopedAndInheritedByApplyOp(t *testing.T) {
	base := dsl.NewVM(unit.NewStore(), agenda.New(), nil)
	if _, err := base.Execute(`"MemoryOnePDProfileA" game-experiment-valid?`); err == nil {
		t.Fatal("unselected VM exposed game word")
	}
	store := loadGames(t)
	vm := dsl.NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
	value, err := vm.Execute(`"initial:C" "after-CC:C" "after-CD:D" "after-DC:C" "after-DD:D" 5 list-of "ValidGameStrategy" apply-op`)
	if err != nil || !value.AsBool() {
		t.Fatalf("child VM game word = (%v,%v)", value, err)
	}
}

func TestGameTasksAreIdempotentAfterCompletion(t *testing.T) {
	store, eng, _ := runGamesStore(t, loadGames(t), false)
	before := gameSemanticSnapshot(t, store)
	experiment := store.Get("MemoryOnePDProfileA")
	eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: experiment.Name, SlotName: experiment.GetString("generationTaskSlot")})
	eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: experiment.GetStrings("candidateUnits")[0], SlotName: experiment.GetString("evaluationTaskSlot")})
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: experiment.Name, SlotName: experiment.GetString("finalizationTaskSlot")})
	after := gameSemanticSnapshot(t, store)
	if string(before) != string(after) {
		t.Fatal("repeated game tasks changed completed game semantics")
	}
}

func gameSemanticSnapshot(t *testing.T, store *unit.Store) []byte {
	t.Helper()
	snapshot := map[string]map[string]any{}
	for _, name := range store.All() {
		u := store.Get(name)
		if name != "MemoryOnePDProfileA" && u.GetString("gameExperiment") != "MemoryOnePDProfileA" {
			continue
		}
		slots := map[string]any{}
		for slot, value := range u.Slots {
			slots[slot] = value
		}
		snapshot[name] = slots
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestGamePreselectionBarrierRejectsCorruption(t *testing.T) {
	controls := map[string]func(*unit.Store){
		"missing": func(store *unit.Store) {
			store.Delete(firstGameArtifact(store, "result"))
		},
		"forged": func(store *unit.Store) {
			store.Get(firstGameArtifact(store, "result")).Set("candidateScore", 999)
		},
		"duplicate": func(store *unit.Store) {
			cloneGameUnit(store, firstGameArtifact(store, "result"), "ForgedDuplicateGameResult")
		},
		"extra": func(store *unit.Store) {
			u := unit.New("ForgedExtraGameResult")
			u.Set("gameExperiment", "MemoryOnePDProfileA")
			u.Set("profileKey", store.Get("MemoryOnePDProfileA").GetString("profileKey"))
			u.Set("artifactKind", "result")
			u.Set("semanticKey", "extra")
			store.Put(u)
		},
		"unknown-kind": func(store *unit.Store) {
			u := unit.New("ForgedUnknownGameArtifact")
			u.Set("gameExperiment", "MemoryOnePDProfileA")
			u.Set("profileKey", store.Get("MemoryOnePDProfileA").GetString("profileKey"))
			u.Set("artifactKind", "unknown")
			store.Put(u)
		},
		"wrong-category": func(store *unit.Store) {
			store.Get(firstGameArtifact(store, "result")).Set("isA", []string{"Anything"})
		},
		"application-link": func(store *unit.Store) {
			candidate := store.Get(store.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[0])
			applications := candidate.Get("applics").([]map[string]any)
			applications[0]["output"] = "forged"
		},
	}
	for name, corrupt := range controls {
		t.Run(name, func(t *testing.T) {
			store := loadGames(t)
			vm := dsl.NewVM(store, agenda.New(), nil)
			if value, err := vm.Execute(`"MemoryOnePDProfileA" game-generate-experiment`); err != nil || !value.AsBool() {
				t.Fatalf("generate = (%v,%v)", value, err)
			}
			for _, candidate := range store.Get("MemoryOnePDProfileA").GetStrings("candidateUnits") {
				if value, err := vm.Execute(fmt.Sprintf("%q game-evaluate-candidate", candidate)); err != nil || !value.AsBool() {
					t.Fatalf("evaluate %s = (%v,%v)", candidate, value, err)
				}
			}
			if value, _ := vm.Execute(`"MemoryOnePDProfileA" game-ready-to-finalize?`); !value.AsBool() {
				t.Fatal("uncorrupted preselection state was not ready")
			}
			corrupt(store)
			if value, _ := vm.Execute(`"MemoryOnePDProfileA" game-ready-to-finalize?`); value.AsBool() {
				t.Fatal("corrupted preselection state passed verifier")
			}
		})
	}
}

func TestGamePostselectionVerifierRejectsCorruption(t *testing.T) {
	controls := map[string]func(*unit.Store){
		"schema":     func(store *unit.Store) { store.Get(firstGameArtifact(store, "schema")).Set("candidateCode", 99) },
		"conjecture": func(store *unit.Store) { store.Get(firstGameArtifact(store, "conjecture")).Set("statement", "forged") },
		"selection": func(store *unit.Store) {
			store.Get(firstGameArtifact(store, "selection")).Set("frontierCodes", []int{0})
		},
		"worth": func(store *unit.Store) {
			store.Get(store.Get("MemoryOnePDProfileA").GetStrings("frontierUnits")[0]).SetWorth(799)
		},
		"extra-selection": func(store *unit.Store) {
			cloneGameUnit(store, firstGameArtifact(store, "selection"), "ForgedExtraGameSelection")
		},
	}
	for name, corrupt := range controls {
		t.Run(name, func(t *testing.T) {
			store, eng, _ := runGamesStore(t, loadGames(t), false)
			if !gameComplete(t, eng) {
				t.Fatal("control did not complete")
			}
			corrupt(store)
			if gameComplete(t, eng) {
				t.Fatal("post-selection corruption passed verifier")
			}
		})
	}
}

func TestGameSemanticAllocationSurvivesOccupiedNames(t *testing.T) {
	store := loadGames(t)
	occupied := []string{
		"GameStrategy-DDDDD",
		"GameMatch-DDDDD-dc8eb505ffb3",
		"GameObservation-DDDDD-dc8eb505ffb3",
		"GameEvidence-DDDDD",
		"GameSelection-a546aa6e4374",
		"GameSchema-DDDCC",
		"GameConjecture-DDDCC",
	}
	for _, name := range occupied {
		u := unit.New(name)
		u.Set("sentinel", "preserve")
		store.Put(u)
	}
	store, eng, _ := runGamesStore(t, store, false)
	if !gameComplete(t, eng) {
		t.Fatal("occupied names prevented completion")
	}
	for _, name := range occupied {
		if store.Get(name).GetString("sentinel") != "preserve" {
			t.Fatalf("occupied unit %s was overwritten", name)
		}
	}
}

func TestGameExperimentSurvivesOpaqueFixtureAliases(t *testing.T) {
	store := loadGames(t)
	opponentAliases := map[string]string{"GameOpponentAllC": "Opponent-Z", "GameOpponentAllD": "Opponent-Q", "GameOpponentTFT": "Opponent-X", "GameOpponentAlternator": "Opponent-M"}
	for oldName, newName := range opponentAliases {
		renameGameUnit(store, oldName, newName)
	}
	caseAliases := map[string]string{"GameCaseAllC": "Case-5", "GameCaseAllD": "Case-1", "GameCaseTFT": "Case-9", "GameCaseAlternator": "Case-2", "GameCaseSelf": "Case-7", "GameCasePerturbedTFT": "Case-3"}
	for oldName, newName := range caseAliases {
		renameGameUnit(store, oldName, newName)
	}
	for _, caseName := range caseAliases {
		caseUnit := store.Get(caseName)
		if opponent := caseUnit.GetString("opponent"); opponent != "" {
			caseUnit.Set("opponent", opponentAliases[opponent])
		}
	}
	experiment := store.Get("MemoryOnePDProfileA")
	for index, name := range experiment.GetStrings("evaluationCases") {
		experiment.GetStrings("evaluationCases")[index] = caseAliases[name]
	}
	store, eng, _ := runGamesStore(t, store, false)
	if !gameComplete(t, eng) {
		t.Fatal("opaque aliases changed semantic experiment")
	}
}

func TestGameRuntimeAlternateDescriptorUsesSameHeuristics(t *testing.T) {
	store := loadGames(t)
	opponentAliases := map[string]string{"GameOpponentAllC": "AltOpponent-A", "GameOpponentAllD": "AltOpponent-D", "GameOpponentTFT": "AltOpponent-T", "GameOpponentAlternator": "AltOpponent-X"}
	for oldName, newName := range opponentAliases {
		renameGameUnit(store, oldName, newName)
	}
	caseAliases := map[string]string{"GameCaseAllC": "AltCase-A", "GameCaseAllD": "AltCase-D", "GameCaseTFT": "AltCase-T", "GameCaseAlternator": "AltCase-X", "GameCaseSelf": "AltCase-S", "GameCasePerturbedTFT": "AltCase-P"}
	for oldName, newName := range caseAliases {
		renameGameUnit(store, oldName, newName)
	}
	for _, caseName := range caseAliases {
		caseUnit := store.Get(caseName)
		if opponent := caseUnit.GetString("opponent"); opponent != "" {
			caseUnit.Set("opponent", opponentAliases[opponent])
		}
	}
	experiment := store.Get("MemoryOnePDProfileA")
	experiment.Set("reward", 4)
	experiment.Set("rounds", 20)
	experiment.Set("evaluationCases", []string{"AltCase-X", "AltCase-T", "AltCase-A", "AltCase-S", "AltCase-P"})
	store.Get("AltCase-P").Set("candidateFlips", []int{3})
	store.Get("AltCase-P").Set("opponentFlips", []int{7})
	experiment.Set("profileKey", gameProfileKey(t, store))
	store, eng, _ := runGamesStore(t, store, false)
	if !gameComplete(t, eng) {
		t.Fatal("runtime alternate descriptor did not complete")
	}
	counts := gameArtifactCounts(store)
	if counts["result"] != 160 || counts["observation"] != 160 {
		t.Fatalf("alternate matrix = %v", counts)
	}
}

func TestGameCaseReorderingChangesIdentityButNotSemanticSelection(t *testing.T) {
	baseStore, baseEngine, _ := runGamesStore(t, loadGames(t), false)
	if !gameComplete(t, baseEngine) {
		t.Fatal("base did not complete")
	}
	reordered := loadGames(t)
	experiment := reordered.Get("MemoryOnePDProfileA")
	experiment.Set("evaluationCases", []string{"GameCaseAlternator", "GameCaseTFT", "GameCaseAllD", "GameCaseAllC", "GameCaseSelf", "GameCasePerturbedTFT"})
	experiment.Set("profileKey", gameProfileKey(t, reordered))
	reordered, reorderedEngine, _ := runGamesStore(t, reordered, false)
	if !gameComplete(t, reorderedEngine) {
		t.Fatal("reordered descriptor did not complete")
	}
	if baseStore.Get("MemoryOnePDProfileA").GetString("profileKey") == reordered.Get("MemoryOnePDProfileA").GetString("profileKey") {
		t.Fatal("case order did not change profile identity")
	}
	for code := 0; code < 32; code++ {
		base := baseStore.Get(baseStore.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[code])
		other := reordered.Get(reordered.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[code])
		for _, slot := range []string{"trainingTotal", "trainingWorst", "selfScore", "perturbationScore", "paretoFrontier"} {
			if !reflect.DeepEqual(base.Get(slot), other.Get(slot)) {
				t.Fatalf("code %d slot %s changed under reordering", code, slot)
			}
		}
	}
	for first := 0; first < 32; first++ {
		for second := 0; second < 32; second++ {
			baseFirst := baseStore.Get(baseStore.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[first])
			baseSecond := baseStore.Get(baseStore.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[second])
			otherFirst := reordered.Get(reordered.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[first])
			otherSecond := reordered.Get(reordered.Get("MemoryOnePDProfileA").GetStrings("candidateUnits")[second])
			for _, classSlot := range []string{"behaviorClass", "objectiveClass"} {
				baseSame := baseFirst.GetString(classSlot) == baseSecond.GetString(classSlot)
				otherSame := otherFirst.GetString(classSlot) == otherSecond.GetString(classSlot)
				if baseSame != otherSame {
					t.Fatalf("codes %d/%d partition %s changed under reordering", first, second, classSlot)
				}
			}
		}
	}
}

func TestGameDescriptorRejectsMalformedCombinations(t *testing.T) {
	controls := map[string]func(*unit.Store){
		"invalid-payoffs": func(store *unit.Store) { store.Get("MemoryOnePDProfileA").Set("reward", 5) },
		"duplicate-category": func(store *unit.Store) {
			store.Get("MemoryOnePDProfileA").Set("resultCategory", "GameMatchObservation")
		},
		"duplicate-task-slot": func(store *unit.Store) {
			experiment := store.Get("MemoryOnePDProfileA")
			experiment.Set("evaluationTaskSlot", experiment.GetString("generationTaskSlot"))
		},
		"inverted-priorities": func(store *unit.Store) { store.Get("MemoryOnePDProfileA").Set("evaluationPriority", 900) },
		"duplicate-case": func(store *unit.Store) {
			experiment := store.Get("MemoryOnePDProfileA")
			experiment.Set("evaluationCases", append(experiment.GetStrings("evaluationCases"), "GameCaseAllC"))
		},
		"invalid-flips": func(store *unit.Store) { store.Get("GameCasePerturbedTFT").Set("candidateFlips", []int{10, 10}) },
		"malformed-opponent": func(store *unit.Store) {
			store.Get("GameOpponentTFT").Set("data", []string{"initial:C"})
		},
		"stale-profile-key": func(store *unit.Store) { store.Get("MemoryOnePDProfileA").Set("rounds", 61) },
		"case-cap":          func(store *unit.Store) { store.Get("MemoryOnePDProfileA").Set("caseCap", 5) },
	}
	for name, mutate := range controls {
		t.Run(name, func(t *testing.T) {
			store := loadGames(t)
			mutate(store)
			vm := dsl.NewVM(store, agenda.New(), nil)
			value, err := vm.Execute(`"MemoryOnePDProfileA" game-experiment-valid?`)
			if err != nil || value.AsBool() {
				t.Fatalf("malformed descriptor accepted: value=%v err=%v", value, err)
			}
		})
	}
}

func TestGameDescriptorIgnoresUnlistedCategoryInjection(t *testing.T) {
	store := loadGames(t)
	extra := unit.New("InjectedUnlistedGameCase")
	extra.Set("isA", []string{"GameEvaluationCase", "Anything"})
	extra.Set("axis", "training")
	extra.Set("opponent", "GameOpponentAllD")
	extra.Set("candidateFlips", []int{})
	extra.Set("opponentFlips", []int{})
	store.Put(extra)
	store, eng, _ := runGamesStore(t, store, false)
	if !gameComplete(t, eng) || gameArtifactCounts(store)["result"] != 192 {
		t.Fatal("unlisted category member changed authoritative case matrix")
	}
}

func TestGameObjectiveAblationAgreesWithIndependentFrontier(t *testing.T) {
	store, eng, _ := runGamesStore(t, loadGames(t), false)
	if !gameComplete(t, eng) {
		t.Fatal("control did not complete")
	}
	objectives := map[int]gamevocab.Objectives{}
	for _, name := range store.Get("MemoryOnePDProfileA").GetStrings("candidateUnits") {
		candidate := store.Get(name)
		objectives[candidate.GetInt("semanticCode")] = gamevocab.Objectives{
			TrainingTotal: candidate.GetInt("trainingTotal"),
			TrainingWorst: candidate.GetInt("trainingWorst"),
			SelfScore:     candidate.GetInt("selfScore"),
			PerturbScore:  0,
		}
	}
	production := gamevocab.Frontier(objectives)
	oracle := independentGameFrontier(objectives)
	if !reflect.DeepEqual(production, oracle) {
		t.Fatalf("ablated production/oracle frontier = %v/%v", production, oracle)
	}
}

func independentGameFrontier(objectives map[int]gamevocab.Objectives) []int {
	var frontier []int
	for code := 0; code < gamevocab.StrategyCount; code++ {
		dominated := false
		for other := 0; other < gamevocab.StrategyCount; other++ {
			if other == code {
				continue
			}
			a, b := objectives[other], objectives[code]
			atLeast := a.TrainingTotal >= b.TrainingTotal && a.TrainingWorst >= b.TrainingWorst && a.SelfScore >= b.SelfScore && a.PerturbScore >= b.PerturbScore
			strict := a.TrainingTotal > b.TrainingTotal || a.TrainingWorst > b.TrainingWorst || a.SelfScore > b.SelfScore || a.PerturbScore > b.PerturbScore
			if atLeast && strict {
				dominated = true
				break
			}
		}
		if !dominated {
			frontier = append(frontier, code)
		}
	}
	return frontier
}

func TestGameStoresAreDeterministicWithMutationModes(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		first, firstEngine, _ := runGamesStore(t, loadGames(t), mutate)
		second, secondEngine, _ := runGamesStore(t, loadGames(t), mutate)
		if !gameComplete(t, firstEngine) || !gameComplete(t, secondEngine) {
			t.Fatalf("mutation=%v did not complete", mutate)
		}
		firstJSON, _ := first.CanonicalJSON()
		secondJSON, _ := second.CanonicalJSON()
		if string(firstJSON) != string(secondJSON) {
			t.Fatalf("mutation=%v stores differ", mutate)
		}
	}
}

func gameArtifactCounts(store *unit.Store) map[string]int {
	counts := map[string]int{}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("gameExperiment") == "MemoryOnePDProfileA" {
			counts[u.GetString("artifactKind")]++
		}
	}
	return counts
}

func gameActions(store *unit.Store, names []string) []string {
	result := make([]string, len(names))
	for index, name := range names {
		result[index] = store.Get(name).GetString("actions")
	}
	return result
}

func firstGameArtifact(store *unit.Store, kind string) string {
	for _, name := range store.All() {
		if store.Get(name).GetString("artifactKind") == kind {
			return name
		}
	}
	return ""
}

func cloneGameUnit(store *unit.Store, source, target string) {
	original := store.Get(source)
	clone := unit.New(target)
	for slot, value := range original.Slots {
		clone.Set(slot, value)
	}
	store.Put(clone)
}

func renameGameUnit(store *unit.Store, oldName, newName string) {
	original := store.Delete(oldName)
	alias := unit.New(newName)
	for slot, value := range original.Slots {
		alias.Set(slot, value)
	}
	store.Put(alias)
}

func gameProfileKey(t *testing.T, store *unit.Store) string {
	t.Helper()
	experiment := store.Get("MemoryOnePDProfileA")
	profile := gamevocab.Profile{
		ExperimentKey: experiment.GetString("experimentKey"), ComparisonMethod: experiment.GetString("comparisonMethod"),
		Payoffs: gamevocab.Payoffs{Temptation: experiment.GetInt("temptation"), Reward: experiment.GetInt("reward"), Punishment: experiment.GetInt("punishment"), Sucker: experiment.GetInt("sucker")}, Rounds: experiment.GetInt("rounds"),
	}
	for _, caseName := range experiment.GetStrings("evaluationCases") {
		caseUnit := store.Get(caseName)
		candidateFlips, _ := caseUnit.Get("candidateFlips").([]int)
		opponentFlips, _ := caseUnit.Get("opponentFlips").([]int)
		gameCase := gamevocab.Case{Axis: gamevocab.Axis(caseUnit.GetString("axis")), Self: caseUnit.GetBool("self"), CandidateFlip: candidateFlips, OpponentFlip: opponentFlips}
		if !gameCase.Self {
			strategy, err := gamevocab.ParseStrategy(store.Get(caseUnit.GetString("opponent")).GetStrings("data"))
			if err != nil {
				t.Fatal(err)
			}
			gameCase.Opponent = strategy
		}
		profile.Cases = append(profile.Cases, gameCase)
	}
	key, err := profile.Key()
	if err != nil {
		encoded, _ := json.Marshal(profile)
		t.Fatalf("profile key: %v (%s)", err, encoded)
	}
	return key
}
