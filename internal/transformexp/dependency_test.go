package transformexp

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestTransformationDependencyBoundary(t *testing.T) {
	rules := map[string][]string{
		"../vocab/transformschema": {"internal/transformexp", "internal/transformbaseline", "internal/transformoracle"},
		"../dsl":                   {"internal/transformexp", "internal/transformbaseline", "internal/transformoracle"},
		"../transformfixturecore":  {"internal/transformexp", "internal/transformbaseline", "internal/transformoracle", "internal/vocab/transformschema", "internal/dsl"},
		"../transformbaseline":     {"internal/transformexp", "internal/transformoracle", "internal/vocab/transformschema", "internal/dsl", "internal/engine"},
		"../transformoracle":       {"github.com/chazu/nous/internal/"},
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
}

func TestTransformationProductionSourcesContainNoPanelAnswers(t *testing.T) {
	production := []string{
		"../vocab/transformschema",
		"../dsl/builtins_transformschema.go",
		"../transformfixturecore",
		"../transformbaseline",
		"../transformoracle",
		"../../domains/transformschema",
	}
	for _, path := range production {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		var files []string
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), ".cue")) && !strings.HasSuffix(entry.Name(), "_test.go") {
					files = append(files, filepath.Join(path, entry.Name()))
				}
			}
		} else {
			files = []string{path}
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"841001", "842001", "locked-curriculum", "familySchemas", "latentSchema", "acceptedAttempt", "heldoutExpected"} {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("%s contains forbidden panel/answer term %q", file, forbidden)
				}
			}
		}
	}
}

func TestTransformationImplementationChangesStayWithinFrozenPaths(t *testing.T) {
	allowedPrefixes := []string{
		"cmd/nous", "domains/transformschema/", "internal/dsl/builtins_transformschema", "internal/transformbaseline/",
		"internal/transformexp/", "internal/transformfixturecore/", "internal/transformoracle/", "internal/vocab/transformschema/",
		"docs/transformation-schema", "docs/vocabulary-research-part-3",
	}
	planCommit := PlanCommit
	command := testGitCommand(t, "diff", "--name-only", planCommit+"..HEAD")
	scanner := bufio.NewScanner(strings.NewReader(command))
	for scanner.Scan() {
		path := scanner.Text()
		if path == "" {
			continue
		}
		allowed := slices.ContainsFunc(allowedPrefixes, func(prefix string) bool { return strings.HasPrefix(path, prefix) })
		if !allowed {
			t.Fatalf("implementation changed frozen-out path %s", path)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func testGitCommand(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = "../.."
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(output)
}
