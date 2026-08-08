package causalrun

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

// DependencyEvidence is a closed, read-only result suitable for retention by
// experiment controls. The audit accepts no predicate or callback from its
// caller, so the forbidden graph cannot be redefined by experiment code.
type DependencyEvidence struct {
	Files          int      `json:"files"`
	ImportEdges    int      `json:"import_edges"`
	MethodEdges    int      `json:"method_edges"`
	Lookups        int64    `json:"lookups"`
	RunnerMethods  []string `json:"runner_methods"`
	TeacherMethods []string `json:"teacher_methods"`
	Forbidden      []string `json:"forbidden"`
}

// AuditDependencyBoundary recursively audits the complete production
// causalrun tree and the exported Runner/Teacher method graph. It deliberately
// exposes evidence only; it does not expose a Runner, Store, VM, or callback.
func AuditDependencyBoundary(repoRoot string) (DependencyEvidence, error) {
	root := filepath.Join(repoRoot, "internal", "causalrun")
	evidence := DependencyEvidence{RunnerMethods: []string{}, TeacherMethods: []string{}, Forbidden: []string{}}
	deniedImports := []string{"internal/causaloracle", "internal/causalexp", "generator"}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		evidence.Files++
		evidence.Lookups++
		for _, imported := range parsed.Imports {
			evidence.ImportEdges++
			evidence.Lookups++
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, denied := range deniedImports {
				if strings.Contains(importPath, denied) {
					evidence.Forbidden = append(evidence.Forbidden, fmt.Sprintf("%s imports %s", relativePath(root, path), importPath))
				}
			}
		}
		// Exported root-package APIs may be typed, but must not accept raw
		// callback functions which would let experiment code redefine checks.
		if parsed.Name.Name == "causalrun" {
			for _, declaration := range parsed.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || !function.Name.IsExported() || function.Type.Params == nil {
					continue
				}
				evidence.Lookups++
				for _, field := range function.Type.Params.List {
					if containsFunctionType(field.Type) {
						evidence.Forbidden = append(evidence.Forbidden, fmt.Sprintf("%s accepts callback parameter", function.Name.Name))
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return DependencyEvidence{}, err
	}

	runnerType := reflect.TypeOf((*Runner)(nil))
	allowedRunner := []string{"AdvanceToTeacher", "ArtifactBytes", "Boundary", "Close", "Respond", "Result", "Run", "State"}
	for index := 0; index < runnerType.NumMethod(); index++ {
		name := runnerType.Method(index).Name
		evidence.RunnerMethods = append(evidence.RunnerMethods, name)
		evidence.MethodEdges++
		evidence.Lookups++
		if !slices.Contains(allowedRunner, name) {
			evidence.Forbidden = append(evidence.Forbidden, "Runner."+name)
		}
	}
	teacherType := reflect.TypeOf((*Teacher)(nil)).Elem()
	for index := 0; index < teacherType.NumMethod(); index++ {
		name := teacherType.Method(index).Name
		evidence.TeacherMethods = append(evidence.TeacherMethods, name)
		evidence.MethodEdges++
		evidence.Lookups++
	}
	if !slices.Equal(evidence.TeacherMethods, []string{"Respond"}) {
		evidence.Forbidden = append(evidence.Forbidden, "Teacher method graph is not exactly Respond")
	}
	for index := 0; index < runnerType.Elem().NumField(); index++ {
		field := runnerType.Elem().Field(index)
		evidence.Lookups++
		if strings.Contains(strings.ToLower(field.Name), "hidden") || dependencyTypeContainsHidden(field.Type, map[reflect.Type]bool{}) {
			evidence.Forbidden = append(evidence.Forbidden, "Runner hidden-bearing field "+field.Name)
		}
	}
	slices.Sort(evidence.Forbidden)
	return evidence, nil
}

func relativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return relative
}

func containsFunctionType(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.FuncType:
		return true
	case *ast.ArrayType:
		return containsFunctionType(value.Elt)
	case *ast.MapType:
		return containsFunctionType(value.Key) || containsFunctionType(value.Value)
	case *ast.StarExpr:
		return containsFunctionType(value.X)
	case *ast.Ellipsis:
		return containsFunctionType(value.Elt)
	case *ast.ChanType:
		return containsFunctionType(value.Value)
	default:
		return false
	}
}

func dependencyTypeContainsHidden(value reflect.Type, seen map[reflect.Type]bool) bool {
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
		if dependencyTypeContainsHidden(value.Field(index).Type, seen) {
			return true
		}
	}
	return false
}
