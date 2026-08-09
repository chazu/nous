package transformexp

import (
	"bytes"
	"fmt"

	"github.com/chazu/nous/internal/transformoracle"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

type CompetenceReport struct {
	Forests             int  `json:"forests"`
	SchemaApplications  int  `json:"schema_applications"`
	ProgramApplications int  `json:"program_applications"`
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
	report.Passed = report.Forests == 351 && report.SchemaApplications == 25272 && report.ProgramApplications == 7020
	if !report.Passed {
		return report, fmt.Errorf("competence cardinality mismatch: %+v", report)
	}
	return report, nil
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
