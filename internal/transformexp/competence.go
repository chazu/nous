package transformexp

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/transformoracle"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

type CompetenceReport struct {
	Forests             int  `json:"forests"`
	SchemaApplications  int  `json:"schema_applications"`
	ProgramApplications int  `json:"program_applications"`
	Microcases          int  `json:"microcases"`
	Passed              bool `json:"passed"`
}

func runTransformCompetence() (CompetenceReport, error) {
	report := CompetenceReport{}
	for _, forest := range tinyTransformForests() {
		report.Forests++
		forestBytes, err := forest.CanonicalJSON()
		if err != nil {
			return report, err
		}
		for _, schema := range transformschema.Schemas() {
			schemaBytes, _ := schema.CanonicalJSON()
			production, productionErr := schema.Apply(forest)
			oracle, oracleErr := transformoracle.Apply(forestBytes, schemaBytes)
			report.SchemaApplications++
			if (productionErr != nil) != (oracleErr != nil) || production.Terminal != oracle.Terminal {
				return report, fmt.Errorf("schema disagreement forest=%s schema=%s production=%s/%v oracle=%s/%v", forestBytes, schemaBytes, production.Terminal, productionErr, oracle.Terminal, oracleErr)
			}
			var productionOutput []byte
			if production.Output != nil {
				productionOutput, _ = production.Output.CanonicalJSON()
			}
			if !bytes.Equal(productionOutput, oracle.Output) {
				return report, fmt.Errorf("schema output disagreement forest=%s schema=%s", forestBytes, schemaBytes)
			}
		}
		for _, program := range tinyTransformPrograms(forest) {
			programBytes, _ := program.CanonicalJSON()
			production, productionErr := program.Apply(forest)
			oracle, oracleErr := transformoracle.ApplyProgram(forestBytes, programBytes)
			report.ProgramApplications++
			if (productionErr != nil) != (oracleErr != nil) {
				return report, fmt.Errorf("program terminal disagreement forest=%s program=%s", forestBytes, programBytes)
			}
			productionBytes, _ := production.CanonicalJSON()
			if !bytes.Equal(productionBytes, oracle) {
				return report, fmt.Errorf("program output disagreement forest=%s program=%s", forestBytes, programBytes)
			}
		}
	}
	microcases, err := runTransformMicrocases()
	if err != nil {
		return report, err
	}
	report.Microcases = len(microcases) / 2
	report.Passed = report.Forests == 351 && report.SchemaApplications == 25272 && report.ProgramApplications == 7020 && report.Microcases == 14
	if !report.Passed {
		return report, fmt.Errorf("competence cardinality mismatch: %+v", report)
	}
	return report, nil
}

func runTransformMicrocases() (map[string][]byte, error) {
	type check struct {
		name  string
		input any
		pass  func() bool
	}
	base := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "d", Value: "a", Target: -1},
		{ID: 2, Kind: "request", Parent: 0, Key: "q", From: "a", To: "b", Target: 1},
		{ID: 3, Kind: "reference", Parent: 0, Key: "r", Value: "a", Target: 1},
	}}
	apply := func(f transformschema.Forest, s transformschema.Schema) transformschema.Result {
		result, _ := s.Apply(f)
		return result
	}
	cloneBase := func() transformschema.Forest { return transformschema.Forest{Nodes: slices.Clone(base.Nodes)} }
	checks := []check{
		{"schema-roundtrip", []any{"all", 72}, func() bool {
			seen := map[string]bool{}
			for _, schema := range transformschema.Schemas() {
				encoded, err := schema.CanonicalJSON()
				if err != nil {
					return false
				}
				if _, err := transformschema.ParseSchema(encoded); err != nil || seen[string(encoded)] {
					return false
				}
				seen[string(encoded)] = true
			}
			return len(seen) == 72
		}},
		{"anchor-request-target", []any{"request-target", "unique"}, func() bool {
			return apply(base, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Terminal == "applied"
		}},
		{"anchor-from-value", []any{"from-value", "unique"}, func() bool {
			return apply(base, transformschema.Schema{"from-value", "definition", "local", "any", "required"}).Terminal == "applied"
		}},
		{"anchor-first-local", []any{"first-local", "first"}, func() bool {
			return apply(base, transformschema.Schema{"first-local", "definition", "local", "any", "required"}).Terminal == "applied"
		}},
		{"expansion-targets", []any{"definition+references", 2}, func() bool {
			result := apply(base, transformschema.Schema{"request-target", "definition+references", "local", "any", "required"})
			return result.Terminal == "applied" && len(result.Certificate.Edits) == 2
		}},
		{"reference-scope", []any{"local", "global"}, func() bool {
			forest := cloneBase()
			forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 4, Kind: "group", Parent: -1, Target: -1}, transformschema.Node{ID: 5, Kind: "reference", Parent: 4, Key: "s", Value: "a", Target: 1})
			local := apply(forest, transformschema.Schema{"request-target", "references", "local", "any", "none"})
			global := apply(forest, transformschema.Schema{"request-target", "references", "global", "any", "none"})
			return len(local.Certificate.Edits) == 1 && len(global.Certificate.Edits) == 2
		}},
		{"old-guard", []any{"equals-from", "any"}, func() bool {
			forest := cloneBase()
			forest.Nodes[3].Value = "c"
			equal := apply(forest, transformschema.Schema{"request-target", "references", "local", "equals-from", "required"})
			any := apply(forest, transformschema.Schema{"request-target", "references", "local", "any", "required"})
			return equal.Terminal == "abstain/expansion" && any.Terminal == "applied"
		}},
		{"locality-noop", []any{"locality", "no-op"}, func() bool {
			forest := cloneBase()
			forest.Nodes[2].To = "a"
			return apply(forest, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Terminal == "abstain/no-op"
		}},
		{"edit-boundaries", []any{0, 1, 4, 5}, func() bool {
			return (transformschema.Program{}).Validate() != nil && (transformschema.Program{Edits: []transformschema.Edit{{1, "b"}}}).Validate() == nil && (transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {2, "b"}, {3, "b"}, {4, "b"}}}).Validate() == nil && (transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {2, "b"}, {3, "b"}, {4, "b"}, {5, "b"}}}).Validate() != nil
		}},
		{"alpha-child-order", []any{"reordered", "canonical"}, func() bool {
			left, _ := base.CanonicalJSON()
			rightForest := cloneBase()
			slices.Reverse(rightForest.Nodes)
			right, _ := rightForest.CanonicalJSON()
			return bytes.Equal(left, right)
		}},
		{"program-invalid", []any{"duplicate-target", "no-op"}, func() bool {
			duplicate := transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {1, "c"}}}
			noOp := transformschema.Program{Edits: []transformschema.Edit{{1, "a"}}}
			_, noOpErr := noOp.Apply(base)
			return duplicate.Validate() != nil && noOpErr != nil
		}},
		{"wrong-context", []any{"two-requests", "abstain"}, func() bool {
			forest := cloneBase()
			forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 4, Kind: "request", Parent: 0, Key: "x", From: "a", To: "b", Target: 1})
			return apply(forest, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Terminal == "abstain/request-count"
		}},
		{"oracle-parity", []any{"production", "oracle"}, func() bool {
			forestBytes, _ := base.CanonicalJSON()
			schema := transformschema.Schema{"request-target", "definition+references", "local", "any", "required"}
			schemaBytes, _ := schema.CanonicalJSON()
			production := apply(base, schema)
			oracle, err := transformoracle.Apply(forestBytes, schemaBytes)
			productionBytes, _ := production.Output.CanonicalJSON()
			return err == nil && production.Terminal == oracle.Terminal && bytes.Equal(productionBytes, oracle.Output)
		}},
		{"evidence-corruption", []any{"frozen-digest", "mutated-schema"}, func() bool {
			schema := transformschema.Schema{"request-target", "definition", "local", "any", "required"}
			original, _ := schema.CanonicalJSON()
			commitment := digestBytes(original)
			schema.Locality = "none"
			mutated, _ := schema.CanonicalJSON()
			return commitment != digestBytes(mutated)
		}},
	}
	files := map[string][]byte{}
	for _, item := range checks {
		input := mustJSON([]any{"transform-competence-input/v1", item.name, item.input})
		if !item.pass() {
			return nil, fmt.Errorf("competence microcase failed: %s", item.name)
		}
		result := mustJSON([]any{"transform-competence-result/v1", item.name, "passed", digestBytes(input)})
		files["competence/cases/"+item.name+"-input.json"] = input
		files["competence/cases/"+item.name+"-result.json"] = result
	}
	return files, nil
}

func tinyTransformForests() []transformschema.Forest {
	values := []string{"a", "b", "c"}
	var forests []transformschema.Forest
	for _, definition := range values {
		for _, from := range values {
			for _, to := range values {
				for references := 0; references <= 2; references++ {
					combinations := 1
					for range references {
						combinations *= len(values)
					}
					for combination := 0; combination < combinations; combination++ {
						forest := transformschema.Forest{Nodes: []transformschema.Node{
							{ID: 0, Kind: "group", Parent: -1, Target: -1},
							{ID: 1, Kind: "definition", Parent: 0, Key: "d", Value: definition, Target: -1},
							{ID: 2, Kind: "request", Parent: 0, Key: "q", From: from, To: to, Target: 1},
						}}
						n := combination
						for index := 0; index < references; index++ {
							forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 3 + index, Kind: "reference", Parent: 0, Key: []string{"r", "s"}[index], Value: values[n%len(values)], Target: 1})
							n /= len(values)
						}
						forests = append(forests, forest)
					}
				}
			}
		}
	}
	return forests
}

func tinyTransformPrograms(forest transformschema.Forest) []transformschema.Program {
	values := []string{"a", "b", "c"}
	var editable []transformschema.Node
	for _, node := range forest.Nodes {
		if node.Kind == "definition" || node.Kind == "reference" {
			editable = append(editable, node)
		}
	}
	var programs []transformschema.Program
	var visit func(int, []transformschema.Edit)
	visit = func(index int, edits []transformschema.Edit) {
		if index == len(editable) {
			if len(edits) > 0 {
				programs = append(programs, transformschema.Program{Edits: append([]transformschema.Edit(nil), edits...)})
			}
			return
		}
		visit(index+1, edits)
		node := editable[index]
		for _, value := range values {
			if value != node.Value {
				visit(index+1, append(edits, transformschema.Edit{Target: node.ID, Value: value}))
			}
		}
	}
	visit(0, nil)
	return programs
}
