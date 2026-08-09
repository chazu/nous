package transformoracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type Observation struct {
	Token    string
	Terminal string
	Output   []byte
}

type PolicyAudit struct {
	TrainingExact bool
	ProgramsExact bool
	Heldout       []Observation
}

type ScoreAudit struct {
	Correct           int
	CorrectBits       byte
	FalseApplications int
}

type AcceptanceAudit struct {
	Applications int
	Work         int64
	MatrixSHA256 string
	Accepted     bool
}

type TerminalAudit struct {
	TrainingCases int
	HeldoutCases  int
	Valid         bool
}

func AuditTerminal(trainingBytes, heldoutBytes []byte, terminal string) (TerminalAudit, error) {
	if terminal != "no-discovery" && terminal != "budget-exhausted" {
		return TerminalAudit{}, ErrInvalid
	}
	training, err := auditTraining(trainingBytes)
	if err != nil {
		return TerminalAudit{}, err
	}
	heldout, err := auditHeldout(heldoutBytes)
	if err != nil {
		return TerminalAudit{}, err
	}
	return TerminalAudit{len(training), len(heldout), len(training) == 8 && len(heldout) == 8}, nil
}

type fixtureCase struct {
	token, kind   string
	before, after []byte
}

type batchRow struct {
	token, beforeDigest string
	program             []byte
}

func AuditAcceptance(trainingBytes, heldoutBytes, scorerBytes []byte) (AcceptanceAudit, error) {
	training, err := auditTraining(trainingBytes)
	if err != nil {
		return AcceptanceAudit{}, err
	}
	heldout, err := auditHeldout(heldoutBytes)
	if err != nil {
		return AcceptanceAudit{}, err
	}
	value, err := decode(scorerBytes)
	if err != nil {
		return AcceptanceAudit{}, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 6 || row[0] != "transform-scorer-curriculum/v1" {
		return AcceptanceAudit{}, ErrInvalid
	}
	latent, err := json.Marshal(row[4])
	if err != nil {
		return AcceptanceAudit{}, err
	}
	if _, err := parseSchema(latent); err != nil {
		return AcceptanceAudit{}, err
	}
	expectedRows, ok := row[5].([]any)
	if !ok || len(expectedRows) != 8 || len(training) != 8 || len(heldout) != 8 {
		return AcceptanceAudit{}, ErrInvalid
	}
	expected := map[string]Observation{}
	previous := ""
	for _, raw := range expectedRows {
		fields, ok := raw.([]any)
		if !ok || len(fields) != 3 {
			return AcceptanceAudit{}, ErrInvalid
		}
		token, tokenOK := fields[0].(string)
		terminal, terminalOK := fields[1].(string)
		if !tokenOK || !terminalOK || token <= previous || !oneOf(terminal, "applied", "abstain") {
			return AcceptanceAudit{}, ErrInvalid
		}
		var output []byte
		if fields[2] != nil {
			output, err = json.Marshal(fields[2])
			if err != nil {
				return AcceptanceAudit{}, err
			}
		}
		expected[token] = Observation{token, terminal, output}
		previous = token
	}
	anchors := []string{"request-target", "from-value", "first-local"}
	targets := []string{"definition", "references", "definition+references"}
	scopes := []string{"local", "global"}
	guards := []string{"equals-from", "any"}
	localities := []string{"required", "none"}
	best := int(^uint(0) >> 1)
	var winners [][]byte
	var matrixRows []any
	applications := 0
	latentHeldoutExact := false
	for _, anchor := range anchors {
		for _, target := range targets {
			for _, scope := range scopes {
				for _, guard := range guards {
					for _, locality := range localities {
						schemaBytes, _ := json.Marshal([]any{"transform-schema/v1", anchor, target, scope, guard, locality})
						trainingExact, heldoutExact := true, true
						var outcomes []any
						for _, test := range training {
							result, applyErr := Apply(test.before, schemaBytes)
							if applyErr != nil {
								return AcceptanceAudit{}, applyErr
							}
							applications++
							matches := test.kind == "positive" && result.Terminal == "applied" && bytes.Equal(result.Output, test.after) || test.kind == "abstain" && len(result.Terminal) >= 8 && result.Terminal[:8] == "abstain/"
							trainingExact = trainingExact && matches
							outcomes = append(outcomes, []any{test.token, result.Terminal, auditDigest(result.Output), matches})
						}
						for _, test := range heldout {
							result, applyErr := Apply(test.before, schemaBytes)
							if applyErr != nil {
								return AcceptanceAudit{}, applyErr
							}
							applications++
							truth, exists := expected[test.token]
							matches := exists && (truth.Terminal == "applied" && result.Terminal == "applied" && bytes.Equal(result.Output, truth.Output) || truth.Terminal == "abstain" && len(result.Terminal) >= 8 && result.Terminal[:8] == "abstain/")
							heldoutExact = heldoutExact && matches
							outcomes = append(outcomes, []any{test.token, result.Terminal, auditDigest(result.Output), matches})
						}
						matrixRows = append(matrixRows, []any{auditDigest(schemaBytes), trainingExact, heldoutExact, outcomes})
						if bytes.Equal(schemaBytes, latent) {
							latentHeldoutExact = heldoutExact
						}
						if trainingExact {
							cost := auditSchemaDescription(anchor, target, scope, guard, locality)
							if cost < best {
								best = cost
								winners = [][]byte{schemaBytes}
							} else if cost == best {
								winners = append(winners, schemaBytes)
							}
						}
					}
				}
			}
		}
	}
	if applications != 72*16 {
		return AcceptanceAudit{}, ErrInvalid
	}
	matrix, _ := json.Marshal([]any{"transform-generator-acceptance-matrix/v1", matrixRows})
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, scorerBytes) {
		return AcceptanceAudit{}, ErrInvalid
	}
	return AcceptanceAudit{applications, 109161, auditDigest(matrix), len(winners) == 1 && bytes.Equal(winners[0], latent) && latentHeldoutExact}, nil
}

func auditDigest(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func auditSchemaDescription(anchor, target, scope, guard, locality string) int {
	cost := map[string]int{"request-target": 1, "from-value": 2, "first-local": 3, "definition": 1, "references": 1, "definition+references": 2, "local": 1, "global": 2, "equals-from": 2, "any": 1, "required": 2, "none": 1}
	return cost[anchor] + cost[target] + cost[scope] + cost[guard] + cost[locality]
}

func AuditPolicy(trainingBytes, heldoutBytes, schemaBytes, batchBytes []byte) (PolicyAudit, error) {
	training, err := auditTraining(trainingBytes)
	if err != nil {
		return PolicyAudit{}, err
	}
	heldout, err := auditHeldout(heldoutBytes)
	if err != nil {
		return PolicyAudit{}, err
	}
	audit := PolicyAudit{TrainingExact: true, ProgramsExact: true}
	var rows []batchRow
	if len(batchBytes) != 0 {
		rows, err = auditBatch(batchBytes)
		if err != nil {
			return PolicyAudit{}, err
		}
		positives := map[string]fixtureCase{}
		for _, test := range training {
			if test.kind == "positive" {
				positives[test.token] = test
			}
		}
		for _, row := range rows {
			test, ok := positives[row.token]
			digest := sha256.Sum256(test.before)
			output, applyErr := ApplyProgram(test.before, row.program)
			if !ok || hex.EncodeToString(digest[:]) != row.beforeDigest || applyErr != nil || !bytes.Equal(output, test.after) {
				audit.ProgramsExact = false
			}
		}
		if len(rows) != len(positives) {
			audit.ProgramsExact = false
		}
	}
	if len(schemaBytes) != 0 {
		if _, err := parseSchema(schemaBytes); err != nil {
			return PolicyAudit{}, err
		}
		for _, test := range training {
			result, applyErr := Apply(test.before, schemaBytes)
			if applyErr != nil || test.kind == "positive" && (result.Terminal != "applied" || !bytes.Equal(result.Output, test.after)) || test.kind == "abstain" && !(len(result.Terminal) >= 8 && result.Terminal[:8] == "abstain/") {
				audit.TrainingExact = false
			}
		}
		for _, test := range heldout {
			result, applyErr := Apply(test.before, schemaBytes)
			if applyErr != nil {
				return PolicyAudit{}, applyErr
			}
			audit.Heldout = append(audit.Heldout, Observation{test.token, result.Terminal, result.Output})
		}
		return audit, nil
	}
	if len(rows) == 0 {
		return PolicyAudit{}, ErrInvalid
	}
	for _, test := range heldout {
		terminal := "abstain/replay-miss"
		var output []byte
		digest := sha256.Sum256(test.before)
		for _, row := range rows {
			if row.beforeDigest == hex.EncodeToString(digest[:]) {
				output, err = ApplyProgram(test.before, row.program)
				if err != nil {
					return PolicyAudit{}, err
				}
				terminal = "applied"
				break
			}
		}
		audit.Heldout = append(audit.Heldout, Observation{test.token, terminal, output})
	}
	return audit, nil
}

func AuditScore(scorerBytes []byte, observations []Observation) (ScoreAudit, error) {
	value, err := decode(scorerBytes)
	if err != nil {
		return ScoreAudit{}, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 6 || row[0] != "transform-scorer-curriculum/v1" {
		return ScoreAudit{}, ErrInvalid
	}
	latent, err := json.Marshal(row[4])
	if err != nil {
		return ScoreAudit{}, err
	}
	if _, err := parseSchema(latent); err != nil {
		return ScoreAudit{}, err
	}
	expectedRows, ok := row[5].([]any)
	if !ok || len(expectedRows) != 8 || len(observations) != 8 {
		return ScoreAudit{}, ErrInvalid
	}
	expected := map[string]Observation{}
	previous := ""
	for _, raw := range expectedRows {
		values, ok := raw.([]any)
		if !ok || len(values) != 3 {
			return ScoreAudit{}, ErrInvalid
		}
		token, tokenOK := values[0].(string)
		terminal, terminalOK := values[1].(string)
		if !tokenOK || !terminalOK || token <= previous {
			return ScoreAudit{}, ErrInvalid
		}
		var output []byte
		if values[2] != nil {
			output, err = json.Marshal(values[2])
			if err != nil {
				return ScoreAudit{}, err
			}
		}
		expected[token] = Observation{token, terminal, output}
		previous = token
	}
	audit := ScoreAudit{}
	previous = ""
	for index, observation := range observations {
		truth, exists := expected[observation.Token]
		if !exists || observation.Token <= previous {
			return ScoreAudit{}, ErrInvalid
		}
		correct := truth.Terminal == "applied" && observation.Terminal == "applied" && bytes.Equal(truth.Output, observation.Output) || truth.Terminal == "abstain" && len(observation.Terminal) >= 8 && observation.Terminal[:8] == "abstain/"
		if correct {
			audit.Correct++
			audit.CorrectBits |= 1 << index
		}
		if truth.Terminal == "abstain" && observation.Terminal == "applied" {
			audit.FalseApplications++
		}
		previous = observation.Token
	}
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, scorerBytes) {
		return ScoreAudit{}, ErrInvalid
	}
	return audit, nil
}

func auditTraining(data []byte) ([]fixtureCase, error) {
	value, err := decode(data)
	if err != nil {
		return nil, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 3 || row[0] != "transform-policy-curriculum/v1" || row[1] != auditProfileDigest() {
		return nil, ErrInvalid
	}
	caseRows, ok := row[2].([]any)
	if !ok || len(caseRows) != 8 {
		return nil, ErrInvalid
	}
	values := make([]fixtureCase, len(caseRows))
	positives, abstentions := 0, 0
	previous := ""
	for index, raw := range caseRows {
		fields, ok := raw.([]any)
		if !ok || len(fields) != 4 {
			return nil, ErrInvalid
		}
		values[index].token, ok = fields[0].(string)
		if !ok || values[index].token <= previous {
			return nil, ErrInvalid
		}
		values[index].kind, ok = fields[1].(string)
		if !ok {
			return nil, ErrInvalid
		}
		values[index].before, _ = json.Marshal(fields[2])
		if _, err := parseForest(values[index].before); err != nil {
			return nil, err
		}
		switch values[index].kind {
		case "positive":
			positives++
			if fields[3] == nil {
				return nil, ErrInvalid
			}
			values[index].after, _ = json.Marshal(fields[3])
			if _, err := parseForest(values[index].after); err != nil {
				return nil, err
			}
		case "abstain":
			abstentions++
			if fields[3] != nil {
				return nil, ErrInvalid
			}
		default:
			return nil, ErrInvalid
		}
		previous = values[index].token
	}
	canonical, _ := json.Marshal(value)
	if positives != 4 || abstentions != 4 || !bytes.Equal(canonical, data) {
		return nil, ErrInvalid
	}
	return values, nil
}

func auditHeldout(data []byte) ([]fixtureCase, error) {
	value, err := decode(data)
	if err != nil {
		return nil, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 3 || row[0] != "transform-heldout-inputs/v1" || row[1] != auditProfileDigest() {
		return nil, ErrInvalid
	}
	caseRows, ok := row[2].([]any)
	if !ok || len(caseRows) != 8 {
		return nil, ErrInvalid
	}
	values := make([]fixtureCase, len(caseRows))
	previous := ""
	for index, raw := range caseRows {
		fields, ok := raw.([]any)
		if !ok || len(fields) != 2 {
			return nil, ErrInvalid
		}
		values[index].token, ok = fields[0].(string)
		if !ok || values[index].token <= previous {
			return nil, ErrInvalid
		}
		values[index].before, _ = json.Marshal(fields[1])
		if _, err := parseForest(values[index].before); err != nil {
			return nil, err
		}
		previous = values[index].token
	}
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, data) {
		return nil, ErrInvalid
	}
	return values, nil
}

func auditBatch(data []byte) ([]batchRow, error) {
	value, err := decode(data)
	if err != nil {
		return nil, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 2 || row[0] != "transform-program-batch/v1" {
		return nil, ErrInvalid
	}
	rows, ok := row[1].([]any)
	if !ok || len(rows) != 4 {
		return nil, ErrInvalid
	}
	values := make([]batchRow, len(rows))
	previous := ""
	for index, raw := range rows {
		fields, ok := raw.([]any)
		if !ok || len(fields) != 3 {
			return nil, ErrInvalid
		}
		values[index].token, ok = fields[0].(string)
		if !ok || values[index].token <= previous {
			return nil, ErrInvalid
		}
		values[index].beforeDigest, ok = fields[1].(string)
		if !ok || len(values[index].beforeDigest) != 64 {
			return nil, ErrInvalid
		}
		values[index].program, _ = json.Marshal(fields[2])
		if _, err := ApplyProgram([]byte{}, values[index].program); err == nil {
			// ApplyProgram must reject the deliberately absent forest; program
			// structure is checked later against its actual training forest.
			return nil, ErrInvalid
		}
		previous = values[index].token
	}
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, data) {
		return nil, ErrInvalid
	}
	sort.Slice(values, func(i, j int) bool { return values[i].token < values[j].token })
	return values, nil
}

func auditProfileDigest() string {
	preimage, _ := json.Marshal([]any{"transform-profile/v1", "typed-reference-forest/v1", "set-scalar-from-request/v1", "anchor-target-scope-old-guard-locality/v1", "transform-lifecycle-events/v2", 12, 4, 72, 48, 12000})
	digest := sha256.Sum256(preimage)
	return hex.EncodeToString(digest[:])
}
