package nogoodexp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

type fixtureRecord struct {
	Ordinal  int    `json:"ordinal"`
	Problem  string `json:"problem"`
	Variable int    `json:"variable"`
	Color    int    `json:"color"`
}

func encodeFixtureBundle(panel string, tasks []nogoodfixture.Task) ([]byte, error) {
	want, _, err := fixturePanelShape(panel)
	if err != nil {
		return nil, err
	}
	if len(tasks) != want {
		return nil, fmt.Errorf("%s fixture bundle has %d tasks, want %d", panel, len(tasks), want)
	}
	records := make([]fixtureRecord, len(tasks))
	for index, task := range tasks {
		if task.Panel != panel || task.Ordinal != index {
			return nil, fmt.Errorf("%s fixture %d has noncanonical identity", panel, index)
		}
		problem, err := nogoods.ParseProblem(task.ProblemJSON)
		if err != nil {
			return nil, fmt.Errorf("%s fixture %d problem: %w", panel, index, err)
		}
		canonical, err := problem.CanonicalJSON()
		if err != nil || !bytes.Equal(canonical, task.ProblemJSON) {
			return nil, fmt.Errorf("%s fixture %d problem is not canonical", panel, index)
		}
		if task.Decision.Variable < 0 || task.Decision.Variable >= len(problem.Variables) || task.Decision.Color < 0 || task.Decision.Color >= len(problem.ColorAliases) || !problem.DomainContains(task.Decision.Variable, task.Decision.Color) {
			return nil, fmt.Errorf("%s fixture %d decision is invalid", panel, index)
		}
		records[index] = fixtureRecord{
			Ordinal: index, Problem: base64.RawURLEncoding.EncodeToString(task.ProblemJSON),
			Variable: task.Decision.Variable, Color: task.Decision.Color,
		}
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}
	if len(encoded) > FixtureBundleByteCap {
		return nil, fmt.Errorf("%s fixture bundle exceeds byte cap", panel)
	}
	return encoded, nil
}

func decodeFixtureBundle(panel string, encoded []byte) ([]nogoodfixture.Task, error) {
	want, counts, err := fixturePanelShape(panel)
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 || len(encoded) > FixtureBundleByteCap {
		return nil, fmt.Errorf("%s fixture bundle has invalid byte size", panel)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var records []fixtureRecord
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("decode %s fixture bundle: %w", panel, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode %s fixture bundle: %w", panel, err)
	}
	canonical, err := json.Marshal(records)
	if err != nil || !bytes.Equal(canonical, encoded) {
		return nil, fmt.Errorf("%s fixture bundle is not canonical", panel)
	}
	if len(records) != want {
		return nil, fmt.Errorf("%s fixture bundle has %d records, want %d", panel, len(records), want)
	}
	tasks := make([]nogoodfixture.Task, len(records))
	for index, record := range records {
		if record.Ordinal != index || record.Variable < 0 || record.Color < 0 {
			return nil, fmt.Errorf("%s fixture record %d has invalid numeric fields", panel, index)
		}
		problemJSON, err := base64.RawURLEncoding.DecodeString(record.Problem)
		if err != nil || base64.RawURLEncoding.EncodeToString(problemJSON) != record.Problem {
			return nil, fmt.Errorf("%s fixture record %d has noncanonical problem encoding", panel, index)
		}
		problem, err := nogoods.ParseProblem(problemJSON)
		if err != nil {
			return nil, fmt.Errorf("%s fixture record %d problem: %w", panel, index, err)
		}
		canonicalProblem, err := problem.CanonicalJSON()
		if err != nil || !bytes.Equal(canonicalProblem, problemJSON) {
			return nil, fmt.Errorf("%s fixture record %d problem is not canonical", panel, index)
		}
		if record.Variable >= len(problem.Variables) || record.Color >= len(problem.ColorAliases) || !problem.DomainContains(record.Variable, record.Color) {
			return nil, fmt.Errorf("%s fixture record %d decision is out of range", panel, index)
		}
		cohort, local := fixtureCohort(index, counts)
		task := nogoodfixture.Task{
			Panel: panel, Ordinal: index, Cohort: cohort, CohortOrdinal: local,
			Template: index % 4, MissingBit: -1, ProblemJSON: slices.Clone(problemJSON),
			Decision: nogoods.Literal{Variable: record.Variable, Color: record.Color},
		}
		if panel == "development" {
			task.Seed = 832001 + index
		} else if panel == "validation" {
			task.Seed = 833001 + index
		}
		if cohort == nogoodfixture.NearMiss {
			task.MissingBit = local % 3
		}
		tasks[index] = task
	}
	return tasks, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("trailing JSON value")
}

func fixturePanelShape(panel string) (int, [4]int, error) {
	switch panel {
	case "development":
		return DevelopmentTaskCount, [4]int{56, 24, 8, 8}, nil
	case "validation":
		return ValidationTaskCount, [4]int{112, 48, 16, 16}, nil
	case "locked":
		return LockedTaskCount, [4]int{312, 48, 12, 12}, nil
	default:
		return 0, [4]int{}, fmt.Errorf("unknown fixture panel %q", panel)
	}
}

func fixtureCohort(ordinal int, counts [4]int) (nogoodfixture.Cohort, int) {
	cohorts := [4]nogoodfixture.Cohort{
		nogoodfixture.Reusable, nogoodfixture.NearMiss,
		nogoodfixture.Irrelevant, nogoodfixture.IndependentUnsat,
	}
	start := 0
	for index, count := range counts {
		if ordinal < start+count {
			return cohorts[index], ordinal - start
		}
		start += count
	}
	return "", -1
}
