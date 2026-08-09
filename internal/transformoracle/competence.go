package transformoracle

import (
	"bytes"
	"encoding/json"
	"sort"
)

type CompetenceAudit struct {
	Forests, SchemaApplications, ProgramApplications int
}

func AuditMinimumDescriptionTier(candidates [][]byte) ([][]byte, error) {
	type ranked struct {
		encoded []byte
		value   Schema
	}
	values := make([]ranked, len(candidates))
	for index, encoded := range candidates {
		value, err := parseSchema(encoded)
		if err != nil {
			return nil, err
		}
		values[index] = ranked{append([]byte(nil), encoded...), value}
	}
	sort.Slice(values, func(i, j int) bool {
		left := auditSchemaDescription(values[i].value.Anchor, values[i].value.Targets, values[i].value.Scope, values[i].value.Guard, values[i].value.Locality)
		right := auditSchemaDescription(values[j].value.Anchor, values[j].value.Targets, values[j].value.Scope, values[j].value.Guard, values[j].value.Locality)
		if left != right {
			return left < right
		}
		return bytes.Compare(values[i].encoded, values[j].encoded) < 0
	})
	if len(values) == 0 {
		return nil, nil
	}
	minimum := auditSchemaDescription(values[0].value.Anchor, values[0].value.Targets, values[0].value.Scope, values[0].value.Guard, values[0].value.Locality)
	var result [][]byte
	for _, value := range values {
		if auditSchemaDescription(value.value.Anchor, value.value.Targets, value.value.Scope, value.value.Guard, value.value.Locality) != minimum {
			break
		}
		result = append(result, value.encoded)
	}
	return result, nil
}

// AuditCompetenceUniverse independently generates and executes the frozen tiny
// universe. It deliberately shares neither production forest/program types nor
// the transformexp enumerators.
func AuditCompetenceUniverse() (CompetenceAudit, error) {
	values := []string{"a", "b", "c"}
	result := CompetenceAudit{}
	for _, definition := range values {
		for _, from := range values {
			for _, to := range values {
				for references := 0; references <= 2; references++ {
					combinations := 1
					for range references {
						combinations *= len(values)
					}
					for combination := 0; combination < combinations; combination++ {
						nodes := []any{[]any{0, "group", -1, "", "", "", "", -1}, []any{1, "definition", 0, "d", definition, "", "", -1}, []any{2, "request", 0, "q", "", from, to, 1}}
						n := combination
						for index := 0; index < references; index++ {
							nodes = append(nodes, []any{3 + index, "reference", 0, []string{"r", "s"}[index], values[n%len(values)], "", "", 1})
							n /= len(values)
						}
						forest, _ := json.Marshal([]any{"typed-reference-forest/v1", nodes})
						result.Forests++
						for _, anchor := range []string{"request-target", "from-value", "first-local"} {
							for _, targets := range []string{"definition", "references", "definition+references"} {
								for _, scope := range []string{"local", "global"} {
									for _, guard := range []string{"equals-from", "any"} {
										for _, locality := range []string{"required", "none"} {
											schema, _ := json.Marshal([]any{"transform-schema/v1", anchor, targets, scope, guard, locality})
											if _, err := Apply(forest, schema); err != nil {
												return CompetenceAudit{}, err
											}
											result.SchemaApplications++
										}
									}
								}
							}
						}
						var editable [][2]any
						editable = append(editable, [2]any{1, definition})
						n = combination
						for index := 0; index < references; index++ {
							editable = append(editable, [2]any{3 + index, values[n%len(values)]})
							n /= len(values)
						}
						var visit func(int, []any) error
						visit = func(index int, edits []any) error {
							if index == len(editable) {
								if len(edits) == 0 {
									return nil
								}
								program, _ := json.Marshal([]any{"concrete-program/v1", edits})
								if _, err := ApplyProgram(forest, program); err != nil {
									return err
								}
								result.ProgramApplications++
								return nil
							}
							if err := visit(index+1, edits); err != nil {
								return err
							}
							for _, value := range values {
								if value != editable[index][1].(string) {
									if err := visit(index+1, append(edits, []any{"set-value/v1", editable[index][0], value})); err != nil {
										return err
									}
								}
							}
							return nil
						}
						if err := visit(0, nil); err != nil {
							return CompetenceAudit{}, err
						}
					}
				}
			}
		}
	}
	return result, nil
}
