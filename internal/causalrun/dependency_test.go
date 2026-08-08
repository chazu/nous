package causalrun

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProductionDependencyAndCapabilityBoundary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"internal/causaloracle", "internal/causalexp", "generator"}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, data, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			for _, denied := range forbidden {
				if strings.Contains(path, denied) {
					t.Fatalf("%s imports forbidden package %s", file, path)
				}
			}
		}
	}

	runnerType := reflect.TypeOf(Runner{})
	for index := 0; index < runnerType.NumField(); index++ {
		field := runnerType.Field(index)
		if strings.Contains(strings.ToLower(field.Name), "hidden") || typeContainsHidden(field.Type, map[reflect.Type]bool{}) {
			t.Fatalf("runner has hidden-bearing field %s %s", field.Name, field.Type)
		}
	}
	teacherType := reflect.TypeOf((*Teacher)(nil)).Elem()
	if teacherType.NumMethod() != 1 || teacherType.Method(0).Name != "Respond" {
		t.Fatalf("teacher capability methods=%v", teacherType.NumMethod())
	}
	_ = ast.File{} // keep the dependency test intentionally AST-backed.
}

func TestDependencyEvidenceRecursesAndChecksMethodGraph(t *testing.T) {
	evidence, err := AuditDependencyBoundary(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	direct, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	directProduction := 0
	for _, file := range direct {
		if !strings.HasSuffix(file, "_test.go") {
			directProduction++
		}
	}
	if evidence.Files <= directProduction {
		t.Fatalf("recursive audit files=%d, direct files=%d; subdirectories were skipped", evidence.Files, directProduction)
	}
	if len(evidence.Forbidden) != 0 {
		t.Fatalf("forbidden dependency/method graph: %v", evidence.Forbidden)
	}
	if len(evidence.RunnerMethods) == 0 || len(evidence.TeacherMethods) != 1 || evidence.TeacherMethods[0] != "Respond" || evidence.MethodEdges == 0 || evidence.Lookups == 0 {
		t.Fatalf("incomplete method-graph evidence: %+v", evidence)
	}
}

func typeContainsHidden(value reflect.Type, seen map[reflect.Type]bool) bool {
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Slice || value.Kind() == reflect.Array || value.Kind() == reflect.Map {
		value = value.Elem()
	}
	if seen[value] {
		return false
	}
	seen[value] = true
	if strings.Contains(strings.ToLower(value.Name()), "hidden") {
		return true
	}
	if value.Kind() != reflect.Struct {
		return false
	}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		if strings.Contains(strings.ToLower(field.Name), "hidden") || typeContainsHidden(field.Type, seen) {
			return true
		}
	}
	return false
}
