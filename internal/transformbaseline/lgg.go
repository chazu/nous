package transformbaseline

import (
	"bytes"
	"slices"

	"github.com/chazu/nous/internal/transformfixturecore"
)

type lggObservation struct {
	forest          forest
	request         node
	definition      node
	editedKinds     map[string]bool
	editedReference []int
}

func positiveLGG(trainingBytes, programBatchBytes []byte) (Result, []Event, error) {
	training, err := transformfixturecore.ParseTraining(trainingBytes)
	if err != nil {
		return Result{}, nil, err
	}
	batch, err := transformfixturecore.ParseProgramBatch(programBatchBytes)
	if err != nil {
		return Result{}, nil, err
	}
	if err := validateProgramBatch(training, batch); err != nil {
		return Result{}, nil, err
	}
	positives := map[string]transformfixturecore.TrainingCase{}
	for _, c := range training.Cases {
		if c.Kind == "positive" {
			positives[c.Token] = c
		}
	}
	observations := make([]lggObservation, 0, 4)
	for _, row := range batch.Rows {
		caseValue := positives[row.Token]
		f, err := parseForest(caseValue.Before)
		if err != nil {
			return Result{}, nil, err
		}
		program, err := parseProgram(row.Program)
		if err != nil {
			return Result{}, nil, err
		}
		observation := lggObservation{forest: f, editedKinds: map[string]bool{}}
		for _, n := range f.nodes {
			if n.kind == "request" {
				observation.request = n
			}
		}
		definitionID := -1
		for _, edit := range program {
			n := f.nodes[edit.target]
			observation.editedKinds[n.kind] = true
			if n.kind == "definition" {
				definitionID = n.id
			}
			if n.kind == "reference" {
				definitionID = n.target
				observation.editedReference = append(observation.editedReference, n.id)
			}
		}
		if definitionID < 0 {
			return Result{}, nil, errInvalid
		}
		observation.definition = f.nodes[definitionID]
		slices.Sort(observation.editedReference)
		observations = append(observations, observation)
	}
	var events []Event
	targetChoices := []string{"definition", "references", "definition+references"}
	var targetExact []string
	for _, choice := range targetChoices {
		exact := true
		for _, observation := range observations {
			match := choice == "definition" && observation.editedKinds["definition"] && !observation.editedKinds["reference"] || choice == "references" && observation.editedKinds["reference"] && !observation.editedKinds["definition"] || choice == "definition+references" && observation.editedKinds["definition"] && observation.editedKinds["reference"]
			events = append(events, lggComparison("target", choice, match))
			exact = exact && match
		}
		if exact {
			targetExact = append(targetExact, choice)
		}
	}
	if len(targetExact) == 0 {
		return Result{Terminal: "no-discovery"}, events, nil
	}
	target := cheapest(targetExact, map[string]int{"definition": 1, "references": 1, "definition+references": 2}, targetChoices)
	anchorChoices := []string{"request-target", "from-value", "first-local"}
	var anchorExact []string
	for _, choice := range anchorChoices {
		exact := true
		for _, observation := range observations {
			selected := -1
			switch choice {
			case "request-target":
				selected = observation.request.target
			case "from-value":
				for _, n := range observation.forest.nodes {
					if n.kind == "definition" && n.value == observation.request.from {
						if selected != -1 {
							selected = -2
							break
						}
						selected = n.id
					}
				}
			case "first-local":
				for _, n := range observation.forest.nodes {
					if n.kind == "definition" && n.parent == observation.request.parent {
						selected = n.id
						break
					}
				}
			}
			match := selected == observation.definition.id
			events = append(events, lggComparison("anchor", choice, match))
			exact = exact && match
		}
		if exact {
			anchorExact = append(anchorExact, choice)
		}
	}
	if len(anchorExact) == 0 {
		return Result{Terminal: "no-discovery"}, events, nil
	}
	anchor := cheapest(anchorExact, map[string]int{"request-target": 1, "from-value": 2, "first-local": 3}, anchorChoices)
	scope, guard := "local", "any"
	if target != "definition" {
		scopeChoices, guardChoices := []string{"local", "global"}, []string{"equals-from", "any"}
		var scopeExact []string
		for _, scopeChoice := range scopeChoices {
			scopeMatches := false
			for _, guardChoice := range guardChoices {
				exact := lggReferenceProjectionExact(observations, scopeChoice, guardChoice, &events, "scope")
				scopeMatches = scopeMatches || exact
			}
			if scopeMatches {
				scopeExact = append(scopeExact, scopeChoice)
			}
		}
		if len(scopeExact) == 0 {
			return Result{Terminal: "no-discovery"}, events, nil
		}
		scope = cheapest(scopeExact, map[string]int{"local": 1, "global": 2}, scopeChoices)
		var guardExact []string
		for _, guardChoice := range guardChoices {
			if lggReferenceProjectionExact(observations, scope, guardChoice, &events, "old-guard") {
				guardExact = append(guardExact, guardChoice)
			}
		}
		if len(guardExact) == 0 {
			return Result{Terminal: "no-discovery"}, events, nil
		}
		guard = cheapest(guardExact, map[string]int{"equals-from": 2, "any": 1}, guardChoices)
	}
	learned := schema{anchor, target, scope, guard, "none"}
	schemaBytes := encodeSchema(learned)
	result := Result{Terminal: "completed", Schema: schemaBytes, Applications: 4, Ties: [][]byte{schemaBytes}}
	for _, c := range training.Cases {
		if c.Kind != "positive" {
			continue
		}
		application, applicationEvents, err := ApplySchemaMetered(c.Before, schemaBytes, "training-validate")
		if err != nil || application.Terminal != "applied" {
			return Result{}, nil, errInvalid
		}
		events = append(events, applicationEvents...)
		_, comparisonEvents, err := CompareOutputsMetered(application.Output, c.After, "training-validate")
		if err != nil || !bytes.Equal(application.Output, c.After) {
			return Result{}, nil, errInvalid
		}
		events = append(events, comparisonEvents...)
	}
	return result, events, nil
}

func lggReferenceProjectionExact(observations []lggObservation, scope, guard string, events *[]Event, phase string) bool {
	exact := true
	for _, observation := range observations {
		var predicted []int
		for _, n := range observation.forest.nodes {
			if n.kind == "reference" && n.target == observation.definition.id && (scope == "global" || n.parent == observation.request.parent) && (guard == "any" || n.value == observation.request.from) {
				predicted = append(predicted, n.id)
			}
		}
		slices.Sort(predicted)
		match := slices.Equal(predicted, observation.editedReference)
		*events = append(*events, lggComparison(phase, scope+"/"+guard, match))
		exact = exact && match
	}
	return exact
}

func lggComparison(phase, value string, result bool) Event {
	return Event{3, "compare", phase, map[bool]string{true: "true", false: "false"}[result], [][]byte{baselineAtom("enum", value), baselineAtom("boolean", true)}, [][]byte{baselineAtom("boolean", result)}}
}

func cheapest(values []string, costs map[string]int, order []string) string {
	best := values[0]
	for _, value := range values[1:] {
		if costs[value] < costs[best] || costs[value] == costs[best] && slices.Index(order, value) < slices.Index(order, best) {
			best = value
		}
	}
	return best
}
