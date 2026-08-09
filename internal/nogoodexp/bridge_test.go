package nogoodexp

import (
	"fmt"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func TestBridgeProfileDigestIsCommitted(t *testing.T) {
	execution, err := NewBridgeExecution("../../domains", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if execution.profileHash != committedBridgeProfileHash {
		t.Fatalf("profile hash = %s", execution.profileHash)
	}
	if len(execution.preflight) != 54 || execution.preflight[53].Operands[2].Number != 54 {
		t.Fatalf("profile preflight = %d events", len(execution.preflight))
	}
}

func TestBridgeInvalidAmbiguityIsAnError(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	problem := nogoods.Problem{
		Version: nogoods.ProblemVersion, ColorAliases: []string{"c0", "c1", "c2"},
		Variables: []nogoods.Variable{
			{Alias: "a", Domain: []int{0, 1}},
			{Alias: "x", Domain: []int{0, 2}},
			{Alias: "y", Domain: []int{0, 2}},
			{Alias: "z", Domain: []int{0, 2}},
		},
		Edges: []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 0, Right: 3}, {Left: 1, Right: 2}, {Left: 1, Right: 3}, {Left: 2, Right: 3}},
	}
	encoded, err := problem.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConsiderPrune("../../domains", encoded, nogoods.Literal{Variable: 0, Color: 0}, &artifact, &authority); err == nil {
		t.Fatal("ambiguous bridge result was not mechanically invalid")
	}
}

func TestCUEBridgeMatcherAgreesAcrossAllFourColorTwoValueDomains(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	execution, err := NewBridgeExecution("../../domains", &artifact, &authority)
	if err != nil {
		t.Fatal(err)
	}
	domains := [][]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	ordinal := 0
	for anchor := 0; anchor < 3; anchor++ {
		for left := range domains {
			for middle := range domains {
				for right := range domains {
					variableDomains := [][]int{domains[left], domains[middle], domains[right]}
					for _, blocked := range variableDomains[anchor] {
						variables := make([]nogoods.Variable, 3)
						for index := range variables {
							variables[index] = nogoods.Variable{Alias: fmt.Sprintf("v%d", index), Domain: slices.Clone(variableDomains[index])}
						}
						problem := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: []string{"c0", "c1", "c2", "c3"}, Variables: variables, Edges: []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}}}
						encoded, encodeErr := problem.CanonicalJSON()
						if encodeErr != nil {
							t.Fatal(encodeErr)
						}
						nonAnchor := []int{}
						for variable := 0; variable < 3; variable++ {
							if variable != anchor {
								nonAnchor = append(nonAnchor, variable)
							}
						}
						escape := otherColor(variableDomains[anchor], blocked)
						leftOnly := otherColor(variableDomains[nonAnchor[0]], blocked)
						rightOnly := otherColor(variableDomains[nonAnchor[1]], blocked)
						want := escape >= 0 && leftOnly >= 0 && rightOnly >= 0 && leftOnly == rightOnly && leftOnly != escape
						disposition, considerErr := execution.Consider(encoded, nogoods.Literal{Variable: anchor, Color: blocked})
						if considerErr != nil {
							t.Fatalf("case %d: %v", ordinal, considerErr)
						}
						if got := disposition.Status == "propose-prune"; got != want {
							t.Fatalf("case %d domains=%v anchor=%d blocked=%d proposal=%v want %v", ordinal, variableDomains, anchor, blocked, got, want)
						}
						ordinal++
					}
				}
			}
		}
	}
	if ordinal != 1296 {
		t.Fatalf("exhaustive cases = %d", ordinal)
	}
}

func TestCUEBridgeExhaustsFourColorSubstitutionsEdgeMasksAndAuthority(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	authorized, err := NewBridgeExecution("../../domains", &artifact, &authority)
	if err != nil {
		t.Fatal(err)
	}
	parsedOnly, err := NewBridgeExecution("../../domains", &artifact, nil)
	if err != nil {
		t.Fatal(err)
	}
	ordinal := 0
	for blocked := 0; blocked < 4; blocked++ {
		for escape := 0; escape < 4; escape++ {
			for only := 0; only < 4; only++ {
				if blocked == escape || blocked == only || escape == only {
					continue
				}
				for edgeMask := 0; edgeMask < 8; edgeMask++ {
					problem := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: []string{"c0", "c1", "c2", "c3"}, Variables: []nogoods.Variable{{Alias: "a", Domain: sortedTestPair(blocked, escape)}, {Alias: "x", Domain: sortedTestPair(blocked, only)}, {Alias: "y", Domain: sortedTestPair(blocked, only)}}}
					for bit, edge := range []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}} {
						if edgeMask&(1<<bit) != 0 {
							problem.Edges = append(problem.Edges, edge)
						}
					}
					encoded, encodeErr := problem.CanonicalJSON()
					if encodeErr != nil {
						t.Fatal(encodeErr)
					}
					decision := nogoods.Literal{Variable: 0, Color: blocked}
					for name, execution := range map[string]*BridgeExecution{"authorized": authorized, "parsed-only": parsedOnly} {
						disposition, considerErr := execution.Consider(encoded, decision)
						if considerErr != nil {
							t.Fatalf("case %d %s: %v", ordinal, name, considerErr)
						}
						want := name == "authorized" && edgeMask == 7
						if got := disposition.Status == "propose-prune"; got != want {
							t.Fatalf("case %d %s mask=%d proposal=%v want %v", ordinal, name, edgeMask, got, want)
						}
						ordinal++
					}
				}
			}
		}
	}
	if ordinal != 384 {
		t.Fatalf("authority/mask cases = %d", ordinal)
	}
}

func sortedTestPair(left, right int) []int {
	if left > right {
		left, right = right, left
	}
	return []int{left, right}
}

func otherColor(domain []int, blocked int) int {
	if !slices.Contains(domain, blocked) {
		return -1
	}
	for _, color := range domain {
		if color != blocked {
			return color
		}
	}
	return -1
}

func learnedArtifact(t *testing.T) (FrozenArtifact, ArtifactAuthority) {
	t.Helper()
	training, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	artifact, encoded, authority, err := FreezeArtifact(training)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseFrozenArtifact(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Digest != artifact.Digest {
		t.Fatal("artifact digest changed across freeze/load")
	}
	return decoded, authority
}

func TestLearnedArtifactProposesOnlyOnReusableDevelopmentCases(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, &artifact, &authority)
		if err != nil {
			t.Fatalf("task %d (%s): %v", task.Ordinal, task.Cohort, err)
		}
		want := "resume"
		if task.Cohort == nogoodfixture.Reusable {
			want = "propose-prune"
		}
		if disposition.Status != want {
			t.Fatalf("task %d (%s) disposition = %s, want %s", task.Ordinal, task.Cohort, disposition.Status, want)
		}
		if disposition.TasksPopped != 1 {
			t.Fatalf("task %d popped %d bridge tasks", task.Ordinal, disposition.TasksPopped)
		}
	}
}

func TestBridgeRejectsInvalidDecisionWithoutCreatingRequest(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range []nogoods.Literal{{Variable: -1, Color: 0}, {Variable: 99, Color: 0}, {Variable: tasks[0].Decision.Variable, Color: 99}} {
		if _, err := ConsiderPrune("../../domains", tasks[0].ProblemJSON, decision, &artifact, &authority); err == nil {
			t.Fatalf("accepted invalid decision %#v", decision)
		}
	}
}

func TestEmptyArtifactUsesSameBridgeAndNeverPrunes(t *testing.T) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks[:4] {
		disposition, err := ConsiderPrune("../../domains", task.ProblemJSON, task.Decision, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if disposition.Status != "resume" || disposition.TasksPopped != 1 {
			t.Fatalf("empty artifact disposition = %#v", disposition)
		}
	}
}

func TestConcreteMemoPrunesOnlyItsExactTrainingTupleThroughBridge(t *testing.T) {
	training, err := nogoodfixture.Training()
	if err != nil {
		t.Fatal(err)
	}
	key, err := concreteMemoKey(training[0].ProblemJSON, training[0].Decision)
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := NewConcreteMemoBridge("../../domains", key)
	if err != nil {
		t.Fatal(err)
	}
	hit, err := bridge.Consider(training[0].ProblemJSON, training[0].Decision)
	if err != nil || hit.Status != "concrete-prune" {
		t.Fatalf("exact memo hit = %#v, %v", hit, err)
	}
	miss, err := bridge.Consider(training[1].ProblemJSON, training[1].Decision)
	if err != nil || miss.Status != "resume" {
		t.Fatalf("exact memo miss = %#v, %v", miss, err)
	}
}

func TestParsedOrMutatedArtifactCannotSelfAuthorize(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	parsedOnly := artifact
	result, err := ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &parsedOnly, nil)
	if err != nil || result.Status != "resume" {
		t.Fatalf("parsed-only artifact = %s, %v", result.Status, err)
	}
	mutated := artifact
	mutated.Mask = 5
	mutated.Digest = artifactDigest(mutated)
	result, err = ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &mutated, &authority)
	if err != nil || result.Status != "resume" {
		t.Fatalf("mutated artifact = %s, %v", result.Status, err)
	}
	forged := artifact
	forged.PromotionProofs[1] = forged.PromotionProofs[0]
	forged.Digest = artifactDigest(forged)
	if err := forged.Validate(); err == nil {
		t.Fatal("duplicate self-hashed proof set was accepted")
	}
}
