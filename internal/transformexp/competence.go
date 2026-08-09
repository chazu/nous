package transformexp

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/transformbaseline"
	"github.com/chazu/nous/internal/transformoracle"
	"github.com/chazu/nous/internal/unit"
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
		{"anchor-bindings", []any{"three", "unique", "absent", "ambiguous"}, func() bool {
			for _, anchor := range []string{"request-target", "from-value", "first-local"} {
				if apply(base, transformschema.Schema{anchor, "definition", "local", "any", "required"}).Terminal != "applied" {
					return false
				}
			}
			absent := cloneBase()
			absent.Nodes[2].From = "c"
			ambiguous := cloneBase()
			ambiguous.Nodes = append(ambiguous.Nodes, transformschema.Node{ID: 4, Kind: "definition", Parent: 0, Key: "e", Value: "a", Target: -1})
			return apply(absent, transformschema.Schema{"from-value", "definition", "local", "any", "required"}).Terminal == "abstain/anchor" && apply(ambiguous, transformschema.Schema{"from-value", "definition", "local", "any", "required"}).Terminal == "abstain/anchor"
		}},
		{"expansion-modes", []any{"definition", "references", "combined"}, func() bool {
			definition := apply(base, transformschema.Schema{"request-target", "definition", "local", "any", "required"})
			references := apply(base, transformschema.Schema{"request-target", "references", "local", "any", "required"})
			combined := apply(base, transformschema.Schema{"request-target", "definition+references", "local", "any", "required"})
			return len(definition.Certificate.Edits) == 1 && len(references.Certificate.Edits) == 1 && len(combined.Certificate.Edits) == 2
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
		{"noop-locality", []any{"no-op", "required", "none"}, func() bool {
			noop := cloneBase()
			noop.Nodes[2].To = "a"
			remote := cloneBase()
			remote.Nodes = append(remote.Nodes, transformschema.Node{ID: 4, Kind: "group", Parent: -1, Target: -1})
			remote.Nodes[1].Parent = 4
			required := apply(remote, transformschema.Schema{"request-target", "definition", "local", "any", "required"})
			none := apply(remote, transformschema.Schema{"request-target", "definition", "local", "any", "none"})
			return apply(noop, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Terminal == "abstain/no-op" && required.Terminal == "abstain/locality" && none.Terminal == "applied"
		}},
		{"expansion-boundaries", []any{0, 1, 4, 5}, func() bool {
			zero := cloneBase()
			zero.Nodes = zero.Nodes[:3]
			four := cloneBase()
			four.Nodes = append(four.Nodes, transformschema.Node{ID: 4, Kind: "reference", Parent: 0, Key: "s", Value: "a", Target: 1}, transformschema.Node{ID: 5, Kind: "reference", Parent: 0, Key: "t", Value: "a", Target: 1})
			five := transformschema.Forest{Nodes: slices.Clone(four.Nodes)}
			five.Nodes = append(five.Nodes, transformschema.Node{ID: 6, Kind: "reference", Parent: 0, Key: "u", Value: "a", Target: 1})
			combined := transformschema.Schema{"request-target", "definition+references", "local", "any", "required"}
			return apply(zero, transformschema.Schema{"request-target", "references", "local", "any", "required"}).Terminal == "abstain/expansion" && len(apply(base, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Certificate.Edits) == 1 && len(apply(four, combined).Certificate.Edits) == 4 && apply(five, combined).Terminal == "abstain/expansion"
		}},
		{"alpha-child-order", []any{"reordered", "canonical"}, func() bool {
			left, _ := base.CanonicalJSON()
			rightForest := cloneBase()
			slices.Reverse(rightForest.Nodes)
			right, _ := rightForest.CanonicalJSON()
			return bytes.Equal(left, right)
		}},
		{"occupied-aliases", []any{"occupied", "alternate-aliases"}, func() bool {
			forest := cloneBase()
			forest.Nodes[1].Key, forest.Nodes[2].Key, forest.Nodes[3].Key = "service", "change", "usage"
			forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 4, Kind: "decoy", Parent: 0, Key: "occupied", Value: "c", Target: -1})
			return apply(forest, transformschema.Schema{"request-target", "definition+references", "local", "any", "required"}).Terminal == "applied"
		}},
		{"primitive-edit-rejection", []any{"zero", "five", "duplicate-target", "no-op"}, func() bool {
			duplicate := transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {1, "c"}}}
			noOp := transformschema.Program{Edits: []transformschema.Edit{{1, "a"}}}
			_, noOpErr := noOp.Apply(base)
			five := transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {2, "b"}, {3, "b"}, {4, "b"}, {5, "b"}}}
			return (transformschema.Program{}).Validate() != nil && five.Validate() != nil && duplicate.Validate() != nil && noOpErr != nil
		}},
		{"wrong-context-corruption", []any{"two-requests", "remote", "digest"}, func() bool {
			forest := cloneBase()
			forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 4, Kind: "request", Parent: 0, Key: "x", From: "a", To: "b", Target: 1})
			schema := transformschema.Schema{"request-target", "definition", "local", "any", "required"}
			encoded, _ := schema.CanonicalJSON()
			schema.Locality = "none"
			mutated, _ := schema.CanonicalJSON()
			return apply(forest, transformschema.Schema{"request-target", "definition", "local", "any", "required"}).Terminal == "abstain/request-count" && digestBytes(encoded) != digestBytes(mutated)
		}},
		{"concrete-program-recovery", []any{"local-facts", "no-pair-helper"}, func() bool {
			program := transformschema.Program{Edits: []transformschema.Edit{{Target: 1, Value: "b"}, {Target: 3, Value: "b"}}}
			programBytes, err := program.CanonicalJSON()
			if err != nil {
				return false
			}
			parsed, err := transformschema.ParseProgram(programBytes)
			output, applyErr := parsed.Apply(base)
			want := apply(base, transformschema.Schema{"request-target", "definition+references", "local", "any", "required"})
			outputBytes, _ := output.CanonicalJSON()
			wantBytes, _ := want.Output.CanonicalJSON()
			return err == nil && applyErr == nil && bytes.Equal(outputBytes, wantBytes)
		}},
		{"ties-evidence-barriers", []any{"mdl-tie", "five-stages"}, func() bool {
			left := transformschema.Schema{"request-target", "definition", "local", "equals-from", "none"}
			right := transformschema.Schema{"request-target", "references", "global", "any", "none"}
			partial := transformschema.Partial{}
			for _, value := range []string{"definition", "request-target", "local", "any", "required"} {
				var err error
				partial, err = partial.Refine(value)
				if err != nil {
					return false
				}
			}
			return schemaDescription(left) == schemaDescription(right) && partial.Stage == 5
		}},
		{"application-prefixes", []any{"standalone", "driver", "byte-identical"}, func() bool {
			forestBytes, _ := base.CanonicalJSON()
			schemaBytes, _ := (transformschema.Schema{"request-target", "definition+references", "local", "any", "required"}).CanonicalJSON()
			_, standalone, err := transformbaseline.ApplySchemaMeteredAt(forestBytes, schemaBytes, "heldout", 0)
			if err != nil {
				return false
			}
			store := unit.NewStore()
			experiment := unit.New("TransformCompetenceExperiment")
			experiment.Set("meterToken", "transform-competence-prefix")
			store.Put(experiment)
			if dsl.RegisterTransformMeter("transform-competence-prefix") != nil {
				return false
			}
			vm := dsl.NewVM(store, agenda.New(), nil)
			vm.CurrentTask = &agenda.Task{UnitName: experiment.Name, SlotName: "tsHeldout"}
			_, _, executeErr := dsl.ExecuteTransformSchemaApplication(vm, forestBytes, schemaBytes)
			records, snapshotErr := dsl.TransformMeterSnapshot("transform-competence-prefix")
			dsl.UnregisterTransformMeter("transform-competence-prefix")
			driver := baselineEventsFromTransformMeter(records)
			standalonePrefix, driverPrefix := len(standalone), len(driver)
			for index, event := range standalone {
				if event.Operation == "schema-application" {
					standalonePrefix = index
					break
				}
			}
			for index, event := range driver {
				if event.Operation == "schema-application" {
					driverPrefix = index
					break
				}
			}
			return executeErr == nil && snapshotErr == nil && standalonePrefix == driverPrefix && bytes.Equal(mustJSON(standalone[:standalonePrefix]), mustJSON(driver[:driverPrefix]))
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
