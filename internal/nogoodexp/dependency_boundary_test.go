package nogoodexp

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestVocabularyDependencyBoundaries(t *testing.T) {
	tests := []struct {
		directory string
		allowed   map[string]bool
	}{
		{"../vocab/nogoods", map[string]bool{}},
		{"../nogoodoracle", map[string]bool{}},
		{"../nogoodbaseline", map[string]bool{}},
		{"../nogoodfixturecore", map[string]bool{"github.com/chazu/nous/internal/vocab/nogoods": true}},
		{"../nogoodfixture", map[string]bool{
			"github.com/chazu/nous/internal/vocab/nogoods":     true,
			"github.com/chazu/nous/internal/nogoodfixturecore": true,
		}},
	}
	for _, test := range tests {
		t.Run(test.directory, func(t *testing.T) {
			entries, err := os.ReadDir(test.directory)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
					continue
				}
				path := filepath.Join(test.directory, entry.Name())
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, spec := range file.Imports {
					name, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						t.Fatal(err)
					}
					if strings.HasPrefix(name, "github.com/chazu/nous/") && !test.allowed[name] {
						t.Fatalf("%s imports forbidden repository package %s", path, name)
					}
				}
			}
		})
	}
}

func TestPartTwoAuthorityTokensAreAbsentFromNogoodProduction(t *testing.T) {
	directories := []string{".", "../nogoodbaseline", "../nogoodfixture", "../nogoodfixturecore", "../nogoodoracle", "../vocab/nogoods"}
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(string(data))
			for _, forbidden := range []string{".git/nous-attempts", "part2/", "part-2/", "causalexp", "active-causal-diagnosis"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("%s contains forbidden prior-program token %q", path, forbidden)
				}
			}
		}
	}
}
