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
	for _, root := range []string{"../actionrelationacquire", "../actionrelationcertify", "../actionrelationcompetence", "../actionrelationexp", "../actionrelationfixture", "../actionrelationfixturecore", "../actionrelationledger", "../actionrelationmatch", "../actionrelationoracle", "../actionrelationscore", "../actionrelationsearch", "../actionrelationutility", "../actionrelationwire", "../vocab/actionrelations", "../../domains/actionrelations"} {
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
