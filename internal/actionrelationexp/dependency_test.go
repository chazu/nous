package actionrelationexp

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestActionRelationDependencyAndHistoricalIsolation(t *testing.T) {
	rules := map[string][]string{
		"../vocab/actionrelations": {"github.com/chazu/nous/internal/", `"os"`, `"io/fs"`},
		"../actionrelationoracle":  {"github.com/chazu/nous/internal/"},
		"../actionrelationcap":     {"internal/actionrelationfixture", "internal/actionrelationscore", "internal/actionrelationrun"},
		"../actionrelationfixture": {"internal/actionrelationsearch", "internal/actionrelationutility", "internal/actionrelationacquire", "internal/actionrelationscore", "internal/actionrelationexp"},
	}
	for directory, forbidden := range rules {
		files, err := filepath.Glob(filepath.Join(directory, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, denied := range forbidden {
				if strings.Contains(string(data), denied) {
					t.Fatalf("%s crosses dependency boundary through %q", file, denied)
				}
			}
		}
	}
	for _, root := range []string{"../actionrelationacquire", "../actionrelationcap", "../actionrelationcertify", "../actionrelationcompetence", "../actionrelationexp", "../actionrelationfixture", "../actionrelationfixturecore", "../actionrelationledger", "../actionrelationmatch", "../actionrelationoracle", "../actionrelationrun", "../actionrelationscore", "../actionrelationsearch", "../actionrelationutility", "../actionrelationwire", "../vocab/actionrelations", "../../domains/actionrelations"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !(strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".cue")) || strings.HasSuffix(path, "_test.go") {
				return err
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, denied := range []string{"internal/transform", "internal/nogood", "internal/causal", "internal/vocab/protocol", ".nous/transform", ".nous/nogood", ".nous/causal"} {
				if strings.Contains(string(data), denied) {
					t.Fatalf("%s imports historical lane surface %q", path, denied)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestProtectedConstructorSurfaceHasOneGuardedProductionPath(t *testing.T) {
	type occurrence struct {
		needle string
		path   string
		count  int
	}
	wants := []occurrence{
		{needle: "actionrelationcap.Authorize(", path: "../actionrelationrun/protected.go", count: 1},
		{needle: "actionrelationscore.PrepareProtectedPanel(", path: "../actionrelationrun/protected.go", count: 1},
		{needle: "actionrelationfixture.GenerateProtectedPanel(", path: "../actionrelationscore/sealed.go", count: 1},
		{needle: "token.BeginConstruction()", path: "../actionrelationfixture/panel_authority.go", count: 1},
		{needle: "actionrelationscore.FinalizePolicyPanel(", path: "../actionrelationrun/isolated.go", count: 1},
		{needle: "actionrelationscore.ExecutePublicPanel(", path: "../actionrelationrun/isolated.go", count: 1},
		{needle: "actionrelationrun.ExecuteIsolatedPolicyWorker(", path: "../../cmd/nous/main.go", count: 1},
	}
	productionRoots := []string{"../actionrelationcap", "../actionrelationfixture", "../actionrelationrun", "../actionrelationscore", "../../cmd/nous"}
	for _, want := range wants {
		found := 0
		for _, root := range productionRoots {
			err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return err
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				count := strings.Count(string(data), want.needle)
				if count > 0 && filepath.Clean(path) != filepath.Clean(want.path) {
					t.Fatalf("protected surface %q appears in unexpected production source %s", want.needle, path)
				}
				found += count
				if strings.Contains(string(data), "go:linkname") {
					t.Fatalf("protected production source uses linkname: %s", path)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if found != want.count {
			t.Fatalf("protected surface %q count=%d want=%d", want.needle, found, want.count)
		}
	}
	for _, forbidden := range []string{
		"func BeginAttemptMeter(",
		"func SealAttemptLedger(",
		"func BuildCurriculumFromCatalogs(",
	} {
		for _, root := range productionRoots {
			err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return err
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("protected construction backdoor %q remains in %s", forbidden, path)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestActionRelationProductionSourcesContainNoPanelAnswers(t *testing.T) {
	paths := []string{"../vocab/actionrelations", "../actionrelationsearch", "../actionrelationutility", "../../domains/actionrelations"}
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !(strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".cue")) || strings.HasSuffix(path, "_test.go") {
				return err
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, denied := range []string{"851001", "852001", "development-public-v1", "validation-public-v1", "locked-curriculum", "latentGuard", "scorer-shards"} {
				if strings.Contains(string(data), denied) {
					t.Fatalf("%s contains forbidden panel/answer term %q", path, denied)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestScorerTruthReachabilityUsesOnlyIndependentOracleTransitions(t *testing.T) {
	data, err := os.ReadFile("../actionrelationfixture/truth.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "actionrelationoracle.Apply(") || !strings.Contains(text, "actionrelationoracle.Observe(") {
		t.Fatal("scorer truth does not use the independent oracle for transitions and labels")
	}
	for _, forbidden := range []string{"actionrelations.Apply(", "actionrelations.Applicable(", "actionrelations.ExecuteHistory(", "actionrelations.ParseState("} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("scorer truth reaches production semantics through %q", forbidden)
		}
	}
}

func TestActionRelationImplementationStaysWithinAcceptedSurface(t *testing.T) {
	allowed := []string{
		"cmd/nous/main.go", "docs/actionrelations-", "domains/actionrelations/", "internal/actionrelation", "internal/vocab/actionrelations/",
		"internal/dsl/builtins_actionrelations", "internal/seed/actionrelations", "mise.toml",
	}
	command := exec.Command("git", "diff", "--name-only", PlanCommit+"..HEAD")
	command.Dir = "../.."
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		path := scanner.Text()
		if path != "" && !slices.ContainsFunc(allowed, func(prefix string) bool { return strings.HasPrefix(path, prefix) }) {
			t.Fatalf("implementation changed frozen-out path %s", path)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
