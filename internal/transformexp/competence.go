package transformexp

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/transformbaseline"
	"github.com/chazu/nous/internal/transformfixturecore"
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

func runTransformCompetence(domainsDir string) (CompetenceReport, error) {
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
	microcases, err := runTransformMicrocases(domainsDir)
	if err != nil {
		return report, err
	}
	report.Microcases = len(microcases) / 2
	oracleUniverse, oracleErr := transformoracle.AuditCompetenceUniverse()
	if oracleErr != nil {
		return report, oracleErr
	}
	report.Passed = report.Forests == 351 && report.SchemaApplications == 25272 && report.ProgramApplications == 7020 && report.Microcases == 14 && oracleUniverse.Forests == report.Forests && oracleUniverse.SchemaApplications == report.SchemaApplications && oracleUniverse.ProgramApplications == report.ProgramApplications
	if !report.Passed {
		return report, fmt.Errorf("competence cardinality mismatch: %+v", report)
	}
	return report, nil
}

func runTransformMicrocases(domainsDir string) (map[string][]byte, error) {
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
			remote := cloneBase()
			remote.Nodes = append(remote.Nodes, transformschema.Node{ID: 4, Kind: "group", Parent: -1, Target: -1})
			remote.Nodes[1].Parent = 4
			firstLocal := apply(ambiguous, transformschema.Schema{"first-local", "definition", "local", "any", "required"})
			return apply(absent, transformschema.Schema{"from-value", "definition", "local", "any", "required"}).Terminal == "abstain/anchor" &&
				apply(ambiguous, transformschema.Schema{"from-value", "definition", "local", "any", "required"}).Terminal == "abstain/anchor" &&
				apply(remote, transformschema.Schema{"first-local", "definition", "local", "any", "required"}).Terminal == "abstain/anchor" &&
				firstLocal.Terminal == "applied" && firstLocal.Certificate.DefinitionID == 1
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
			c, err := makeCurriculum(0, 8, 992004)
			if err != nil {
				return false
			}
			training, err := transformfixturecore.ParseTraining(c.Training)
			if err != nil {
				return false
			}
			occupied := make([]string, 0, 4)
			for _, example := range training.Cases {
				if example.Kind == "positive" {
					occupied = append(occupied, "TS.Program.TS.Example.competence-occupied."+example.Token)
				}
			}
			run, err := runAcquisitionConfigured(domainsDir, c.Training, "competence-occupied", func(store *unit.Store) {
				for _, name := range occupied {
					placeholder := unit.New(name)
					placeholder.Set("descriptorAlias", "already-occupied")
					store.Put(placeholder)
				}
			})
			return err == nil && len(run.Programs) == 4 && run.Terminal == "completed" && apply(forest, transformschema.Schema{"request-target", "definition+references", "local", "any", "required"}).Terminal == "applied"
		}},
		{"primitive-edit-rejection", []any{"zero", "five", "duplicate-target", "no-op"}, func() bool {
			duplicate := transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {1, "c"}}}
			noOp := transformschema.Program{Edits: []transformschema.Edit{{1, "a"}}}
			_, noOpErr := noOp.Apply(base)
			five := transformschema.Program{Edits: []transformschema.Edit{{1, "b"}, {2, "b"}, {3, "b"}, {4, "b"}, {5, "b"}}}
			return (transformschema.Program{}).Validate() != nil && five.Validate() != nil && duplicate.Validate() != nil && noOpErr != nil
		}},
		{"wrong-context-corruption", []any{"two-requests", "remote", "reducer-rejection"}, func() bool {
			forest := cloneBase()
			forest.Nodes = append(forest.Nodes, transformschema.Node{ID: 4, Kind: "request", Parent: 0, Key: "x", From: "a", To: "b", Target: 1})
			zeroRequest := cloneBase()
			zeroRequest.Nodes = zeroRequest.Nodes[:2]
			schema := transformschema.Schema{"request-target", "definition", "local", "any", "required"}
			c, err := makeCurriculum(0, 8, 992001)
			if err != nil {
				return false
			}
			run, err := runAcquisition(domainsDir, c.Training, "competence-corruption")
			if err != nil {
				return false
			}
			bundle, err := transcriptFromAcquisition(run, 0, NousRefine, "0123456789abcdef", digestBytes([]byte("competence-corruption-manifest")))
			if err != nil || len(bundle.Raw) < 2 {
				return false
			}
			corrupt := bytes.Clone(bundle.Raw)
			corrupt[len(corrupt)-2] ^= 1
			_, reduceErr := reduceTransformTranscriptWithTraining(corrupt, bundle.Objects, digestBytes([]byte("competence-corruption-manifest")), c.Training)
			return apply(zeroRequest, schema).Terminal == "abstain/request-count" && apply(forest, schema).Terminal == "abstain/request-count" && reduceErr != nil
		}},
		{"concrete-program-recovery", []any{"ordinary-heuristic", "four-promoted", "exact-replay"}, func() bool {
			c, err := makeCurriculum(0, 8, 992002)
			if err != nil {
				return false
			}
			run, err := runAcquisitionConfigured(domainsDir, c.Training, "competence-programs", func(store *unit.Store) {
				store.Get("H-TransformAcquireConcretePrograms").Set("acquisitionOnly", true)
			})
			if err != nil || len(run.Programs) != 4 {
				return false
			}
			batch, err := programBatch(run)
			if err != nil {
				return false
			}
			audit, err := transformoracle.AuditPolicy(c.Training, c.Heldout, nil, batch)
			return err == nil && audit.ProgramsExact
		}},
		{"ties-evidence-barriers", []any{"complete-minimum-tier", "five-closures", "five-barriers"}, func() bool {
			partial := transformschema.Partial{}
			for _, value := range []string{"definition", "request-target", "local", "any", "required"} {
				var err error
				partial, err = partial.Refine(value)
				if err != nil {
					return false
				}
			}
			c, err := makeCurriculum(0, 8, 992003)
			if err != nil {
				return false
			}
			run, err := runAcquisition(domainsDir, c.Training, "competence-barriers")
			if err != nil {
				return false
			}
			closures, barriers := 0, 0
			for _, record := range run.MeterRecords {
				if record.Operation == "verify" && record.Phase == "freeze" && len(record.Inputs) == 1 && objectVersion(record.Inputs[0], "transform-closure/v1") {
					closures++
				}
			}
			for _, name := range run.Store.All() {
				if strings.HasPrefix(name, "TS.Barrier.") {
					barriers++
				}
			}
			left, _ := (transformschema.Schema{"request-target", "definition", "local", "equals-from", "none"}).CanonicalJSON()
			right, _ := (transformschema.Schema{"request-target", "references", "global", "any", "none"}).CanonicalJSON()
			higher, _ := (transformschema.Schema{"first-local", "definition+references", "global", "equals-from", "required"}).CanonicalJSON()
			tier, tierErr := transformbaseline.MinimumDescriptionTier([][]byte{higher, right, left})
			oracleTier, oracleTierErr := transformoracle.AuditMinimumDescriptionTier([][]byte{higher, right, left})
			_, pbeErr := transformbaseline.BoundedPBE(c.Training)
			completeTier := pbeErr == nil && tierErr == nil && oracleTierErr == nil && len(tier) == 2 && bytes.Equal(tier[0], left) && bytes.Equal(tier[1], right) && slices.EqualFunc(tier, oracleTier, bytes.Equal)
			return completeTier && partial.Stage == 5 && barriers == 5 && closures == 5
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
			malformedForest := append(bytes.Clone(forestBytes), byte('x'))
			malformedSchema := []byte(`["transform-schema/v1","request-target"]`)
			_, forestErr := transformschema.ParseForest(malformedForest)
			_, schemaErr := transformschema.ParseSchema(malformedSchema)
			return executeErr == nil && snapshotErr == nil && standalonePrefix == driverPrefix && bytes.Equal(mustJSON(standalone[:standalonePrefix]), mustJSON(driver[:driverPrefix])) && forestErr != nil && schemaErr != nil
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
