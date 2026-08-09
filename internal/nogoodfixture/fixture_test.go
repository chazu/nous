package nogoodfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodoracle"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func TestVariablePositionGoldenVectors(t *testing.T) {
	tests := []struct {
		panel   string
		root    any
		ordinal int
		n       int
		digest  string
		seeds   [2]uint64
		first   [4]uint64
		want    []int
	}{
		{"training", 831001, 0, 3, "ac5eb4be74b192caba90677858e7ca28c346c1329bfe16347b864600b1669f0d", [2]uint64{12420563552428987082, 13443358654286252584}, [4]uint64{16196928853732314818, 13964589984287698613, 14828555489593947842, 3864106448264698449}, []int{0, 1, 2}},
		{"development", 832001, 0, 8, "7ee119fb7276549805e9faaa614a7049671b62ab1466d6a66417e274b279e2a6", [2]uint64{9142617286286660760, 426147249446875209}, [4]uint64{1213905441346112375, 1699159744660973649, 1395889987714557996, 13902222507069901990}, []int{5, 2, 1, 4, 3, 6, 0, 7}},
		{"validation", 833001, 0, 8, "c6ae755044795c25e589d32a073dc98744347a48470364294121d4445b8aca9e", [2]uint64{14316509253064023077, 16539983283958434183}, [4]uint64{4803874459439516066, 14275222390261356384, 10176726368503672072, 10929098418971052156}, []int{6, 0, 1, 4, 7, 3, 5, 2}},
		{"locked", "0000000000000000000000000000000000000000000000000000000000000000", 0, 8, "d1eae2c49127eaedc67e0983f6395fd5ece5fd712a4b44a8598253a172644af1", [2]uint64{15126151632354011885, 14302879928951594965}, [4]uint64{14781793926225946851, 10566838600647845555, 7622181505994308792, 3324590857268697477}, []int{5, 1, 6, 7, 0, 2, 4, 3}},
	}
	for _, test := range tests {
		encoded, err := json.Marshal([]any{SeedAuthority, test.panel, test.root, test.ordinal, "variable-positions"})
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(encoded)
		seeds := [2]uint64{binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])}
		if hex.EncodeToString(digest[:]) != test.digest || seeds != test.seeds {
			t.Fatalf("%s digest/seeds = %x/%v", test.panel, digest, seeds)
		}
		rawSource := rand.New(rand.NewPCG(seeds[0], seeds[1]))
		var first [4]uint64
		for index := range first {
			first[index] = rawSource.Uint64()
		}
		if first != test.first {
			t.Fatalf("%s first raw values = %v", test.panel, first)
		}
		got := permutation(test.n, stream(test.panel, test.root, test.ordinal, "variable-positions"))
		if !slices.Equal(got, test.want) {
			t.Fatalf("%s permutation = %v, want %v", test.panel, got, test.want)
		}
	}
}

func TestRandomControlGoldenVector(t *testing.T) {
	encoded, err := json.Marshal([]any{SeedAuthority, "training", 831001, 0, "random-control"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	seeds := [2]uint64{binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])}
	if string(encoded) != `["part3/nogoods/v1","training",831001,0,"random-control"]` || hex.EncodeToString(digest[:]) != "287e2fe3f24afab3f17a07eef3a485774c68135bdcc19933e358437c00777e5f" || seeds != [2]uint64{2917822264651741875, 17400228833170589047} {
		t.Fatalf("random-control authority drifted: %s %x %v", encoded, digest, seeds)
	}
	raw := rand.New(rand.NewPCG(seeds[0], seeds[1])).Uint64()
	mask := rand.New(rand.NewPCG(seeds[0], seeds[1])).Uint64N(7)
	if raw != 10321276913399045564 || mask != 3 || RandomControlMask() != 3 {
		t.Fatalf("random-control raw/mask = %d/%d", raw, mask)
	}
}

func TestTrainingIsOneFailureAndThreeSingleEdgeCounterexamples(t *testing.T) {
	tasks, err := Training()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 4 {
		t.Fatalf("training tasks = %d", len(tasks))
	}
	seenMissing := map[int]bool{}
	for ordinal, task := range tasks {
		if task.Ordinal != ordinal || task.Seed != 831001+ordinal || task.Panel != "training" {
			t.Fatalf("training manifest[%d] = %#v", ordinal, task)
		}
		problem, err := nogoods.ParseProblem(task.ProblemJSON)
		if err != nil {
			t.Fatal(err)
		}
		if len(problem.Variables) != 3 || len(problem.ColorAliases) != 4 {
			t.Fatalf("training shape[%d] = %d variables/%d colors", ordinal, len(problem.Variables), len(problem.ColorAliases))
		}
		result, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
		if err != nil {
			t.Fatal(err)
		}
		if result.Satisfiable != (ordinal > 0) {
			t.Fatalf("training[%d] satisfiable = %v", ordinal, result.Satisfiable)
		}
		if ordinal == 0 {
			if task.MissingBit != -1 {
				t.Fatalf("full training missing bit = %d", task.MissingBit)
			}
		} else {
			seenMissing[task.MissingBit] = true
		}
	}
	for bit := 0; bit < 3; bit++ {
		if !seenMissing[bit] {
			t.Fatalf("missing-edge training did not cover bit %d", bit)
		}
	}
}

func TestPromotionCasesCoverAllInjectiveColorSubstitutions(t *testing.T) {
	cases, err := PromotionCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 24 {
		t.Fatalf("promotion cases = %d", len(cases))
	}
	seen := map[[3]int]bool{}
	for ordinal, testCase := range cases {
		if testCase.Ordinal != ordinal {
			t.Fatalf("promotion ordinal[%d] = %d", ordinal, testCase.Ordinal)
		}
		roles := [3]int{testCase.Binding.Blocked, testCase.Binding.Escape, testCase.Binding.Only}
		if roles[0] == roles[1] || roles[0] == roles[2] || roles[1] == roles[2] || seen[roles] {
			t.Fatalf("invalid/duplicate promotion roles = %v", roles)
		}
		seen[roles] = true
		problem, err := nogoods.ParseProblem(testCase.ProblemJSON)
		if err != nil {
			t.Fatal(err)
		}
		conflict, err := nogoods.EvaluateCompletion(problem, nogoods.FullMask, testCase.Binding, testCase.Completion)
		if err != nil || !conflict {
			t.Fatalf("promotion[%d] conflict = %v, %v", ordinal, conflict, err)
		}
	}
}

func TestCompetencePanelsHaveFrozenCaseOrderAndDisjointAliases(t *testing.T) {
	wantKinds := []string{"full", "external-context", "missing-0", "missing-1", "missing-2", "wrong-decision", "pair-domain-three", "duplicate-completion", "missing-0-1", "missing-0-2", "missing-1-2", "anchor-domain-three", "unequal-pair-domains", "cross-decision", "stale-target", "color-position-audit"}
	for _, panel := range []string{"development"} {
		cases, err := DevelopmentCompetence()
		if err != nil {
			t.Fatal(err)
		}
		wantCount := 8
		if len(cases) != wantCount {
			t.Fatalf("%s competence cases = %d", panel, len(cases))
		}
		for ordinal, competenceCase := range cases {
			if competenceCase.Ordinal != ordinal || competenceCase.Kind != wantKinds[ordinal] {
				t.Fatalf("%s case %d = %#v", panel, ordinal, competenceCase)
			}
			if string(competenceCase.ProblemJSON) == "" || bytes.Contains(competenceCase.ProblemJSON, []byte("ta")) || bytes.Contains(competenceCase.ProblemJSON, []byte("tc")) {
				t.Fatalf("%s case %d leaked training aliases", panel, ordinal)
			}
		}
	}
}

func TestPanelCountsCellsAndOracleTruth(t *testing.T) {
	for _, panel := range []string{"development"} {
		tasks, err := DevelopmentPanel()
		if err != nil {
			t.Fatal(err)
		}
		wantTotal := 96
		if len(tasks) != wantTotal {
			t.Fatalf("%s tasks = %d", panel, len(tasks))
		}
		cells := map[[3]int]int{}
		for _, task := range tasks {
			result, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
			if err != nil {
				t.Fatalf("%s task %d: %v", panel, task.Ordinal, err)
			}
			wantSAT := task.Cohort == NearMiss || task.Cohort == Irrelevant
			if result.Satisfiable != wantSAT {
				t.Fatalf("%s task %d cohort %s satisfiable=%v", panel, task.Ordinal, task.Cohort, result.Satisfiable)
			}
			cohortIndex := 0
			switch task.Cohort {
			case NearMiss:
				cohortIndex = 1
			case Irrelevant:
				cohortIndex = 2
			case IndependentUnsat:
				cohortIndex = 3
			}
			bit := task.MissingBit
			cells[[3]int{cohortIndex, task.Template, bit}]++
		}
		multiplier := 1
		for template := 0; template < 4; template++ {
			if cells[[3]int{0, template, -1}] != 14*multiplier {
				t.Fatalf("%s reusable template %d count=%d", panel, template, cells[[3]int{0, template, -1}])
			}
			for bit := 0; bit < 3; bit++ {
				if cells[[3]int{1, template, bit}] != 2*multiplier {
					t.Fatalf("%s near template %d bit %d count=%d", panel, template, bit, cells[[3]int{1, template, bit}])
				}
			}
			if cells[[3]int{2, template, -1}] != 2*multiplier || cells[[3]int{3, template, -1}] != 2*multiplier {
				t.Fatalf("%s nonreusable template %d counts=%d/%d", panel, template, cells[[3]int{2, template, -1}], cells[[3]int{3, template, -1}])
			}
		}
	}
}
