package causalexpv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
)

func gitOutput(repoRoot string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func rootedDependencyProof(repoRoot string, summary causalrun.DependencyEvidence) (causalv2.DependencyProof, error) {
	headBytes, err := gitOutput(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return causalv2.DependencyProof{}, err
	}
	return rootedDependencyProofAt(repoRoot, strings.TrimSpace(string(headBytes)), summary)
}

// preflightDependencyProof reconstructs and validates the complete
// commit-rooted proof before any protected attempt record is created.
func preflightDependencyProof(repoRoot, head string) error {
	summary, err := causalrun.AuditDependencyBoundary(repoRoot)
	if err != nil {
		return fmt.Errorf("audit dependency boundary: %w", err)
	}
	first, err := rootedDependencyProofAt(repoRoot, head, summary)
	if err != nil {
		return fmt.Errorf("construct dependency proof: %w", err)
	}
	second, err := rootedDependencyProofAt(repoRoot, head, summary)
	if err != nil {
		return fmt.Errorf("reconstruct dependency proof: %w", err)
	}
	if err := causalv2.VerifyDependencyProof(first); err != nil {
		return fmt.Errorf("verify dependency proof: %w", err)
	}
	if err := causalv2.VerifyDependencyProof(second); err != nil {
		return fmt.Errorf("verify reconstructed dependency proof: %w", err)
	}
	firstBytes, err := causalv2.CanonicalJSON(first)
	if err != nil {
		return err
	}
	secondBytes, err := causalv2.CanonicalJSON(second)
	if err != nil {
		return err
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		return errors.New("dependency proof reconstruction is not byte-identical")
	}
	wantPaths, err := completeTrackedDependencyPaths(repoRoot, head)
	if err != nil {
		return err
	}
	gotPaths := make([]string, len(first.Files))
	for index, file := range first.Files {
		gotPaths[index] = file.Path
	}
	if !slices.Equal(gotPaths, wantPaths) {
		return errors.New("dependency proof does not cover the complete tracked Go and causal CUE tree")
	}
	return nil
}

func completeTrackedDependencyPaths(repoRoot, head string) ([]string, error) {
	listing, err := gitOutput(repoRoot, "ls-tree", "-rz", "--full-tree", head)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, raw := range bytes.Split(listing, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		meta, nameBytes, ok := bytes.Cut(raw, []byte{'\t'})
		if !ok {
			return nil, errors.New("invalid git ls-tree record")
		}
		parts := strings.Fields(string(meta))
		if len(parts) != 3 {
			return nil, errors.New("invalid git ls-tree metadata")
		}
		mode, kind, name := parts[0], parts[1], string(nameBytes)
		regular := kind == "blob" && (mode == "100644" || mode == "100755")
		inScope := strings.HasSuffix(name, ".go") || strings.HasPrefix(name, "domains/causal/") && strings.HasSuffix(name, ".cue")
		if regular && inScope {
			paths = append(paths, name)
		}
	}
	if !sort.StringsAreSorted(paths) {
		return nil, errors.New("tracked dependency paths are not globally sorted")
	}
	for index := 1; index < len(paths); index++ {
		if paths[index-1] == paths[index] {
			return nil, fmt.Errorf("tracked dependency path is duplicated at index %d: %q", index, paths[index])
		}
	}
	return paths, nil
}

func rootedDependencyProofAt(repoRoot, head string, summary causalrun.DependencyEvidence) (causalv2.DependencyProof, error) {
	listing, err := gitOutput(repoRoot, "ls-tree", "-rz", "--full-tree", head)
	if err != nil {
		return causalv2.DependencyProof{}, err
	}
	proof := causalv2.DependencyProof{AuditedCommit: head, AuditedRoots: []string{"."}, Files: []causalv2.DependencyFile{}, RunnerMethods: []string{}, RunnerFields: []causalv2.RunnerField{}, TeacherMethods: []string{}, Forbidden: append([]string{}, summary.Forbidden...)}
	goSources := map[string][]byte{}
	for _, raw := range bytes.Split(listing, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		meta, nameBytes, ok := bytes.Cut(raw, []byte{'\t'})
		if !ok {
			return proof, errors.New("invalid git ls-tree record")
		}
		parts := strings.Fields(string(meta))
		if len(parts) != 3 {
			return proof, errors.New("invalid git ls-tree metadata")
		}
		mode, kind, object, name := parts[0], parts[1], parts[2], string(nameBytes)
		if mode == "120000" || kind == "commit" {
			proof.Forbidden = append(proof.Forbidden, "forbidden-tree-entry:"+name)
			continue
		}
		if kind != "blob" || (!strings.HasSuffix(name, ".go") && !(strings.HasPrefix(name, "domains/causal/") && strings.HasSuffix(name, ".cue"))) {
			continue
		}
		content, err := gitOutput(repoRoot, "cat-file", "blob", object)
		if err != nil {
			return proof, err
		}
		sum := sha256.Sum256(content)
		file := causalv2.DependencyFile{Path: name, SourceSHA256: hex.EncodeToString(sum[:]), Imports: []string{}, ExportedFunctionParameters: []causalv2.DependencyParameter{}}
		if strings.HasSuffix(name, ".go") {
			goSources[name] = content
		}
		proof.Files = append(proof.Files, file)
	}
	typeEvidence, err := sourceDependencyMetadata(goSources)
	if err != nil {
		return proof, err
	}
	for index := range proof.Files {
		metadata, ok := typeEvidence.Files[proof.Files[index].Path]
		if !ok {
			continue
		}
		proof.Files[index].Imports = metadata.Imports
		proof.Files[index].ExportedFunctionParameters = metadata.Parameters
	}
	proof.RunnerMethods = typeEvidence.RunnerMethods
	proof.TeacherMethods = typeEvidence.TeacherMethods
	proof.RunnerFields = typeEvidence.RunnerFields
	for _, field := range proof.RunnerFields {
		if field.HiddenBearing {
			proof.Forbidden = append(proof.Forbidden, "runner-hidden-bearing-field:"+field.Name)
		}
	}
	for _, field := range proof.RunnerFields {
		hidden := strings.Contains(strings.ToLower(field.Name), "hidden") || strings.Contains(strings.ToLower(field.Type), "hidden")
		if hidden != field.HiddenBearing {
			return proof, errors.New("runner hidden-bearing classification mismatch")
		}
	}
	proof.Lookups = int64(len(proof.Files) + len(proof.RunnerMethods) + len(proof.RunnerFields) + len(proof.TeacherMethods))
	for _, file := range proof.Files {
		proof.Lookups += int64(len(file.Imports) + len(file.ExportedFunctionParameters))
	}
	diff := exec.Command("git", "-C", repoRoot, "diff", "--quiet", PlanCommit+".."+head, "--", "internal/engine", "internal/agenda", "internal/dsl/vm.go")
	if err := diff.Run(); err != nil {
		proof.Forbidden = append(proof.Forbidden, "forbidden-core-path-diff")
	}
	slices.Sort(proof.Forbidden)
	return proof, nil
}

type dependencyFileMetadata struct {
	Imports    []string
	Parameters []causalv2.DependencyParameter
}

type dependencyTypeEvidence struct {
	Files          map[string]dependencyFileMetadata
	RunnerMethods  []string
	RunnerFields   []causalv2.RunnerField
	TeacherMethods []string
}

type parsedDependencyFile struct {
	name string
	file *ast.File
}

type placeholderImporter struct {
	packages map[string]*types.Package
}

func (value placeholderImporter) Import(importPath string) (*types.Package, error) {
	if imported, ok := value.packages[importPath]; ok {
		return imported, nil
	}
	return nil, fmt.Errorf("dependency type importer: unknown import %q", importPath)
}

func sourceDependencyMetadata(sources map[string][]byte) (dependencyTypeEvidence, error) {
	result := dependencyTypeEvidence{Files: map[string]dependencyFileMetadata{}, RunnerMethods: []string{}, RunnerFields: []causalv2.RunnerField{}, TeacherMethods: []string{}}
	fset := token.NewFileSet()
	groups := map[string][]parsedDependencyFile{}
	for filename, content := range sources {
		parsed, err := parser.ParseFile(fset, filename, content, 0)
		if err != nil {
			return result, err
		}
		imports := make([]string, len(parsed.Imports))
		for index, item := range parsed.Imports {
			value, err := strconv.Unquote(item.Path.Value)
			if err != nil {
				return result, err
			}
			imports[index] = value
		}
		slices.Sort(imports)
		result.Files[filename] = dependencyFileMetadata{Imports: imports, Parameters: []causalv2.DependencyParameter{}}
		group := path.Dir(filename) + "\x00" + parsed.Name.Name
		groups[group] = append(groups[group], parsedDependencyFile{name: filename, file: parsed})
	}
	qualifier := func(pkg *types.Package) string { return pkg.Path() }
	for _, group := range groups {
		slices.SortFunc(group, func(a, b parsedDependencyFile) int { return strings.Compare(a.name, b.name) })
		packagePath := modulePackagePath(path.Dir(group[0].name))
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		checkerFiles := make([]*ast.File, len(group))
		for index := range group {
			checkerFiles[index] = group[index].file
		}
		config := types.Config{
			Importer:         dependencyImporter(group),
			IgnoreFuncBodies: true,
			Error:            func(error) {},
		}
		checked, _ := config.Check(packagePath, fset, checkerFiles, info)
		for _, parsed := range group {
			metadata := result.Files[parsed.name]
			for _, declaration := range parsed.file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || !function.Name.IsExported() || function.Type.Params == nil {
					continue
				}
				qualified := packagePath + "." + function.Name.Name
				if function.Recv != nil && len(function.Recv.List) > 0 {
					receiver := info.TypeOf(function.Recv.List[0].Type)
					if receiver == nil || receiver == types.Typ[types.Invalid] {
						return result, fmt.Errorf("dependency type metadata: unresolved receiver for %s.%s", parsed.name, function.Name.Name)
					}
					qualified = packagePath + ".(" + types.TypeString(receiver, qualifier) + ")." + function.Name.Name
				}
				parameterIndex := 0
				for _, field := range function.Type.Params.List {
					parameterType := info.TypeOf(field.Type)
					if parameterType == nil || parameterType == types.Typ[types.Invalid] {
						return result, fmt.Errorf("dependency type metadata: unresolved parameter for %s.%s", parsed.name, function.Name.Name)
					}
					count := max(len(field.Names), 1)
					for range count {
						metadata.Parameters = append(metadata.Parameters, causalv2.DependencyParameter{Function: qualified, ParameterIndex: parameterIndex, Type: types.TypeString(parameterType, qualifier)})
						parameterIndex++
					}
				}
			}
			slices.SortFunc(metadata.Parameters, compareDependencyParameters)
			result.Files[parsed.name] = metadata
		}
		if path.Dir(group[0].name) == "internal/causalrun" && group[0].file.Name.Name == "causalrun" {
			var err error
			result.RunnerMethods, result.RunnerFields, result.TeacherMethods, err = causalrunTypeEvidence(checked, qualifier)
			if err != nil {
				return result, err
			}
		}
	}
	if result.RunnerMethods == nil || result.RunnerFields == nil || result.TeacherMethods == nil {
		return result, errors.New("dependency type metadata: causalrun API not found")
	}
	return result, nil
}

func modulePackagePath(directory string) string {
	if directory == "." {
		return "github.com/chazu/nous"
	}
	return "github.com/chazu/nous/" + directory
}

func compareDependencyParameters(a, b causalv2.DependencyParameter) int {
	if compared := strings.Compare(a.Function, b.Function); compared != 0 {
		return compared
	}
	if a.ParameterIndex < b.ParameterIndex {
		return -1
	}
	if a.ParameterIndex > b.ParameterIndex {
		return 1
	}
	return strings.Compare(a.Type, b.Type)
}

func dependencyImporter(files []parsedDependencyFile) types.Importer {
	packages := map[string]*types.Package{}
	fallback := importer.Default()
	for _, parsed := range files {
		aliases := map[string]string{}
		for _, item := range parsed.file.Imports {
			importPath, err := strconv.Unquote(item.Path.Value)
			if err != nil || item.Name != nil && (item.Name.Name == "_" || item.Name.Name == ".") {
				continue
			}
			alias := importPackageName(importPath)
			if item.Name != nil {
				alias = item.Name.Name
			}
			aliases[alias] = importPath
			if _, exists := packages[importPath]; exists {
				continue
			}
			if imported, err := fallback.Import(importPath); err == nil {
				packages[importPath] = imported
			} else {
				packages[importPath] = types.NewPackage(importPath, importPackageName(importPath))
			}
		}
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath, ok := aliases[identifier.Name]
			if !ok {
				return true
			}
			pkg := packages[importPath]
			if pkg.Scope().Lookup(selector.Sel.Name) == nil && !pkg.Complete() {
				name := types.NewTypeName(token.NoPos, pkg, selector.Sel.Name, nil)
				types.NewNamed(name, types.NewStruct(nil, nil), nil)
				pkg.Scope().Insert(name)
			}
			return true
		})
	}
	for _, pkg := range packages {
		if !pkg.Complete() {
			pkg.MarkComplete()
		}
	}
	return placeholderImporter{packages: packages}
}

func importPackageName(importPath string) string {
	name := path.Base(importPath)
	if len(name) > 1 && name[0] == 'v' {
		if _, err := strconv.Atoi(name[1:]); err == nil {
			return path.Base(path.Dir(importPath))
		}
	}
	return name
}

func causalrunTypeEvidence(pkg *types.Package, qualifier types.Qualifier) ([]string, []causalv2.RunnerField, []string, error) {
	runnerObject := pkg.Scope().Lookup("Runner")
	teacherObject := pkg.Scope().Lookup("Teacher")
	if runnerObject == nil || teacherObject == nil {
		return nil, nil, nil, errors.New("dependency type metadata: Runner or Teacher missing")
	}
	runnerNamed, ok := types.Unalias(runnerObject.Type()).(*types.Named)
	if !ok {
		return nil, nil, nil, errors.New("dependency type metadata: Runner is not named")
	}
	runnerStruct, ok := runnerNamed.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, nil, errors.New("dependency type metadata: Runner is not a struct")
	}
	runnerMethods := methodSignatures(types.NewMethodSet(types.NewPointer(runnerNamed)), qualifier)
	teacherMethods := methodSignatures(types.NewMethodSet(teacherObject.Type()), qualifier)
	runnerFields := make([]causalv2.RunnerField, 0, runnerStruct.NumFields())
	for index := 0; index < runnerStruct.NumFields(); index++ {
		field := runnerStruct.Field(index)
		fieldType := types.TypeString(field.Type(), qualifier)
		hidden := strings.Contains(strings.ToLower(field.Name()), "hidden") || strings.Contains(strings.ToLower(fieldType), "hidden")
		runnerFields = append(runnerFields, causalv2.RunnerField{Name: field.Name(), Type: fieldType, HiddenBearing: hidden})
	}
	slices.SortFunc(runnerFields, func(a, b causalv2.RunnerField) int { return strings.Compare(a.Name, b.Name) })
	return runnerMethods, runnerFields, teacherMethods, nil
}

func methodSignatures(set *types.MethodSet, qualifier types.Qualifier) []string {
	result := make([]string, 0, set.Len())
	for index := 0; index < set.Len(); index++ {
		method := set.At(index).Obj().(*types.Func)
		result = append(result, method.Name()+types.TypeString(method.Type(), qualifier))
	}
	slices.Sort(result)
	return result
}
