package ruleinductionexp

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func importsUnder(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil { t.Fatal(err) }
	var imports []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") { continue }
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, parser.ImportsOnly)
		if err != nil { t.Fatal(err) }
		for _, imported := range file.Imports { value, _ := strconv.Unquote(imported.Path.Value); imports = append(imports, value) }
	}
	return imports
}

func TestProductionAndOracleDependencyBoundaries(t *testing.T) {
	for _, imported := range importsUnder(t, "../vocab/ruleinduction") {
		for _, forbidden := range []string{"internal/unit", "internal/dsl", "internal/engine", "internal/ruleinductionexp", "internal/ruleinductionoracle"} {
			if strings.Contains(imported, forbidden) { t.Fatalf("production semantics imports %q", imported) }
		}
	}
	for _, imported := range importsUnder(t, "../ruleinductionoracle") {
		if strings.Contains(imported, "internal/vocab/ruleinduction") || strings.Contains(imported, "internal/dsl") { t.Fatalf("oracle imports production %q", imported) }
	}
	adapter, err := os.ReadFile("../dsl/builtins_ruleinduction.go")
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(adapter), "ruleinductionoracle") { t.Fatal("DSL adapter imports the oracle") }
}
