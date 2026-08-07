// Package configrepair implements bounded typed-configuration and repair
// semantics without depending on Nous units, the DSL, or the engine.
package configrepair

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	MaxConfigRecords = 32
	MaxConfigBytes   = 4096
	MaxSchemaRecords = 64
	MaxSchemaBytes   = 8192
	MaxRecordBytes   = 256
	MaxKeyBytes      = 64
	MaxValueBytes    = 128
	MaxPlanSize      = 3

	CreditContext   = "configuration/repair-subsets-up-to-3/v1"
	SynthesisMethod = "repair-subsets-up-to-3/v1"
)

var (
	keyPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	valuePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	intPattern   = regexp.MustCompile(`^(0|-[1-9][0-9]*|[1-9][0-9]*)$`)
)

type Config struct {
	Values map[string]string
}

type FieldKind int

const (
	StringField FieldKind = iota
	BoolField
	IntField
)

type Field struct {
	Kind    FieldKind
	Minimum int
	Maximum int
}

type ConstraintKind int

const (
	EqualIf ConstraintKind = iota
	MinimumIf
)

type Constraint struct {
	Kind        ConstraintKind
	GuardKey    string
	GuardValue  string
	TargetKey   string
	TargetValue string
	Minimum     int
}

type Schema struct {
	Fields      map[string]Field
	Required    map[string]bool
	Protected   map[string]bool
	Constraints []Constraint
}

type Repair struct {
	Key   string
	Value string
}

func ValidConfig(data []string) bool {
	_, err := ParseConfig(data)
	return err == nil
}

func ParseConfig(data []string) (Config, error) {
	if len(data) > MaxConfigRecords {
		return Config{}, fmt.Errorf("configuration has more than %d records", MaxConfigRecords)
	}
	if encodedSize(data) > MaxConfigBytes {
		return Config{}, fmt.Errorf("configuration exceeds %d bytes", MaxConfigBytes)
	}
	config := Config{Values: make(map[string]string, len(data))}
	for _, record := range data {
		if len(record) > MaxRecordBytes {
			return Config{}, fmt.Errorf("configuration record exceeds %d bytes", MaxRecordBytes)
		}
		parts := strings.Split(record, "=")
		if len(parts) != 2 || !ValidRepair(Repair{Key: parts[0], Value: parts[1]}) {
			return Config{}, fmt.Errorf("invalid configuration record %q", record)
		}
		if _, exists := config.Values[parts[0]]; exists {
			return Config{}, fmt.Errorf("duplicate configuration key %q", parts[0])
		}
		config.Values[parts[0]] = parts[1]
	}
	return config, nil
}

func (c Config) Canonical() []string {
	keys := make([]string, 0, len(c.Values))
	for key := range c.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, len(keys))
	for index, key := range keys {
		out[index] = key + "=" + c.Values[key]
	}
	return out
}

func Canonicalize(data []string) ([]string, error) {
	config, err := ParseConfig(data)
	if err != nil {
		return nil, err
	}
	return config.Canonical(), nil
}

func ValidSchema(data []string) bool {
	_, err := ParseSchema(data)
	return err == nil
}

func ParseSchema(data []string) (Schema, error) {
	if len(data) > MaxSchemaRecords {
		return Schema{}, fmt.Errorf("schema has more than %d records", MaxSchemaRecords)
	}
	if encodedSize(data) > MaxSchemaBytes {
		return Schema{}, fmt.Errorf("schema exceeds %d bytes", MaxSchemaBytes)
	}
	schema := Schema{
		Fields:    make(map[string]Field),
		Required:  make(map[string]bool),
		Protected: make(map[string]bool),
	}
	var rawConstraints []string
	for _, record := range data {
		if len(record) > MaxRecordBytes {
			return Schema{}, fmt.Errorf("schema record exceeds %d bytes", MaxRecordBytes)
		}
		parts := strings.Split(record, ":")
		if len(parts) < 2 {
			return Schema{}, fmt.Errorf("invalid schema record %q", record)
		}
		switch parts[0] {
		case "field":
			if err := parseField(parts, schema.Fields); err != nil {
				return Schema{}, err
			}
		case "required":
			if len(parts) != 2 || !validKey(parts[1]) || schema.Required[parts[1]] {
				return Schema{}, fmt.Errorf("invalid or duplicate required record %q", record)
			}
			schema.Required[parts[1]] = true
		case "protected":
			if len(parts) != 2 || !validKey(parts[1]) || schema.Protected[parts[1]] {
				return Schema{}, fmt.Errorf("invalid or duplicate protected record %q", record)
			}
			schema.Protected[parts[1]] = true
		case "eq-if", "min-if":
			if len(parts) != 2 {
				return Schema{}, fmt.Errorf("invalid constraint record %q", record)
			}
			rawConstraints = append(rawConstraints, record)
		default:
			return Schema{}, fmt.Errorf("unknown schema record %q", record)
		}
	}
	for key := range schema.Required {
		if _, exists := schema.Fields[key]; !exists {
			return Schema{}, fmt.Errorf("required key %q has no field", key)
		}
	}
	for key := range schema.Protected {
		if _, exists := schema.Fields[key]; !exists {
			return Schema{}, fmt.Errorf("protected key %q has no field", key)
		}
	}
	seenConstraints := make(map[string]bool)
	for _, record := range rawConstraints {
		constraint, key, err := parseConstraint(record, schema.Fields)
		if err != nil {
			return Schema{}, err
		}
		if seenConstraints[key] {
			return Schema{}, fmt.Errorf("duplicate constraint %q", record)
		}
		seenConstraints[key] = true
		schema.Constraints = append(schema.Constraints, constraint)
	}
	return schema, nil
}

func parseField(parts []string, fields map[string]Field) error {
	if len(parts) < 3 || !validKey(parts[1]) {
		return fmt.Errorf("invalid field record %q", strings.Join(parts, ":"))
	}
	if _, exists := fields[parts[1]]; exists {
		return fmt.Errorf("duplicate field %q", parts[1])
	}
	var field Field
	switch parts[2] {
	case "string":
		if len(parts) != 3 {
			return fmt.Errorf("invalid string field %q", parts[1])
		}
		field.Kind = StringField
	case "bool":
		if len(parts) != 3 {
			return fmt.Errorf("invalid bool field %q", parts[1])
		}
		field.Kind = BoolField
	case "int":
		if len(parts) != 5 {
			return fmt.Errorf("invalid int field %q", parts[1])
		}
		minimum, err := parseCanonicalInt(parts[3])
		if err != nil {
			return fmt.Errorf("invalid minimum for %q", parts[1])
		}
		maximum, err := parseCanonicalInt(parts[4])
		if err != nil || minimum > maximum {
			return fmt.Errorf("invalid maximum for %q", parts[1])
		}
		field = Field{Kind: IntField, Minimum: minimum, Maximum: maximum}
	default:
		return fmt.Errorf("unknown field kind %q", parts[2])
	}
	fields[parts[1]] = field
	return nil
}

func parseConstraint(record string, fields map[string]Field) (Constraint, string, error) {
	parts := strings.SplitN(record, ":", 2)
	assignments := strings.Split(parts[1], ",")
	if len(assignments) != 2 {
		return Constraint{}, "", fmt.Errorf("invalid constraint %q", record)
	}
	guardKey, guardValue, ok := splitAssignment(assignments[0])
	if !ok {
		return Constraint{}, "", fmt.Errorf("invalid guard in %q", record)
	}
	targetKey, targetValue, ok := splitAssignment(assignments[1])
	if !ok {
		return Constraint{}, "", fmt.Errorf("invalid target in %q", record)
	}
	guardField, guardExists := fields[guardKey]
	targetField, targetExists := fields[targetKey]
	if !guardExists || !targetExists || !validFieldValue(guardValue, guardField) {
		return Constraint{}, "", fmt.Errorf("unknown or invalid guard in %q", record)
	}
	constraint := Constraint{GuardKey: guardKey, GuardValue: guardValue, TargetKey: targetKey}
	switch parts[0] {
	case "eq-if":
		if !validFieldValue(targetValue, targetField) {
			return Constraint{}, "", fmt.Errorf("invalid equality target in %q", record)
		}
		constraint.Kind = EqualIf
		constraint.TargetValue = targetValue
		key := fmt.Sprintf("eq|%s|%s|%s|%s", guardKey, guardValue, targetKey, targetValue)
		return constraint, key, nil
	case "min-if":
		if targetField.Kind != IntField {
			return Constraint{}, "", fmt.Errorf("minimum target is not int in %q", record)
		}
		minimum, err := parseCanonicalInt(targetValue)
		if err != nil || minimum < targetField.Minimum || minimum > targetField.Maximum {
			return Constraint{}, "", fmt.Errorf("invalid minimum target in %q", record)
		}
		constraint.Kind = MinimumIf
		constraint.Minimum = minimum
		key := fmt.Sprintf("min|%s|%s|%s|%d", guardKey, guardValue, targetKey, minimum)
		return constraint, key, nil
	default:
		return Constraint{}, "", fmt.Errorf("unknown constraint %q", record)
	}
}

func Satisfies(configData, schemaData []string) (bool, error) {
	config, err := ParseConfig(configData)
	if err != nil {
		return false, err
	}
	schema, err := ParseSchema(schemaData)
	if err != nil {
		return false, err
	}
	for key, value := range config.Values {
		field, exists := schema.Fields[key]
		if !exists || !validFieldValue(value, field) {
			return false, nil
		}
	}
	for key := range schema.Required {
		if _, exists := config.Values[key]; !exists {
			return false, nil
		}
	}
	for _, constraint := range schema.Constraints {
		if config.Values[constraint.GuardKey] != constraint.GuardValue {
			continue
		}
		targetValue, exists := config.Values[constraint.TargetKey]
		if !exists {
			return false, nil
		}
		switch constraint.Kind {
		case EqualIf:
			if targetValue != constraint.TargetValue {
				return false, nil
			}
		case MinimumIf:
			value, err := parseCanonicalInt(targetValue)
			if err != nil || value < constraint.Minimum {
				return false, nil
			}
		}
	}
	return true, nil
}

func ValidRepair(repair Repair) bool {
	return validKey(repair.Key) && len(repair.Value) > 0 && len(repair.Value) <= MaxValueBytes &&
		valuePattern.MatchString(repair.Value)
}

func Apply(configData []string, repair Repair) ([]string, error) {
	config, err := ParseConfig(configData)
	if err != nil {
		return nil, err
	}
	if !ValidRepair(repair) {
		return nil, fmt.Errorf("invalid repair")
	}
	config.Values[repair.Key] = repair.Value
	return config.Canonical(), nil
}

func ValidPlan(repairs []Repair) bool {
	if len(repairs) == 0 || len(repairs) > MaxPlanSize {
		return false
	}
	keys := make(map[string]bool, len(repairs))
	for _, repair := range repairs {
		if !ValidRepair(repair) || keys[repair.Key] {
			return false
		}
		keys[repair.Key] = true
	}
	return true
}

func ApplyPlan(configData []string, repairs []Repair) ([]string, error) {
	if !ValidPlan(repairs) {
		return nil, fmt.Errorf("invalid repair plan")
	}
	result := append([]string(nil), configData...)
	var err error
	for _, repair := range repairs {
		result, err = Apply(result, repair)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func PreservesProtected(beforeData, afterData, schemaData []string) (bool, error) {
	before, err := ParseConfig(beforeData)
	if err != nil {
		return false, err
	}
	after, err := ParseConfig(afterData)
	if err != nil {
		return false, err
	}
	schema, err := ParseSchema(schemaData)
	if err != nil {
		return false, err
	}
	for key := range schema.Protected {
		beforeValue, beforeExists := before.Values[key]
		afterValue, afterExists := after.Values[key]
		field := schema.Fields[key]
		if (beforeExists && !validFieldValue(beforeValue, field)) || (afterExists && !validFieldValue(afterValue, field)) {
			return false, fmt.Errorf("protected key %q has invalid typed value", key)
		}
		if beforeExists != afterExists || beforeValue != afterValue {
			return false, nil
		}
	}
	return true, nil
}

func ChangedKeys(beforeData, afterData []string) (int, error) {
	before, err := ParseConfig(beforeData)
	if err != nil {
		return 0, err
	}
	after, err := ParseConfig(afterData)
	if err != nil {
		return 0, err
	}
	keys := make(map[string]bool, len(before.Values)+len(after.Values))
	for key := range before.Values {
		keys[key] = true
	}
	for key := range after.Values {
		keys[key] = true
	}
	changed := 0
	for key := range keys {
		beforeValue, beforeExists := before.Values[key]
		afterValue, afterExists := after.Values[key]
		if beforeExists != afterExists || beforeValue != afterValue {
			changed++
		}
	}
	return changed, nil
}

// DecisionKey returns bounded semantic assignment identity. It intentionally
// uses raw lexical assignments because type comes from an applied schema.
func DecisionKey(repairs []Repair) (string, error) {
	if !ValidPlan(repairs) {
		return "", fmt.Errorf("invalid repair plan")
	}
	ordered := append([]Repair(nil), repairs...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Key != ordered[j].Key {
			return ordered[i].Key < ordered[j].Key
		}
		return ordered[i].Value < ordered[j].Value
	})
	material := struct {
		Method  string      `json:"method"`
		Repairs [][2]string `json:"repairs"`
	}{Method: SynthesisMethod}
	for _, repair := range ordered {
		material.Repairs = append(material.Repairs, [2]string{repair.Key, repair.Value})
	}
	encoded, _ := json.Marshal(material)
	digest := sha256.Sum256(encoded)
	return "sha256:v1:" + hex.EncodeToString(digest[:]), nil
}

func validKey(key string) bool {
	return len(key) > 0 && len(key) <= MaxKeyBytes && keyPattern.MatchString(key)
}

func validFieldValue(value string, field Field) bool {
	if len(value) == 0 || len(value) > MaxValueBytes || !valuePattern.MatchString(value) {
		return false
	}
	switch field.Kind {
	case StringField:
		return true
	case BoolField:
		return value == "true" || value == "false"
	case IntField:
		parsed, err := parseCanonicalInt(value)
		return err == nil && parsed >= field.Minimum && parsed <= field.Maximum
	default:
		return false
	}
}

func parseCanonicalInt(value string) (int, error) {
	if !intPattern.MatchString(value) {
		return 0, fmt.Errorf("non-canonical integer")
	}
	parsed, err := strconv.ParseInt(value, 10, 0)
	return int(parsed), err
}

func splitAssignment(value string) (string, string, bool) {
	parts := strings.Split(value, "=")
	if len(parts) != 2 || !validKey(parts[0]) || len(parts[1]) == 0 ||
		len(parts[1]) > MaxValueBytes || !valuePattern.MatchString(parts[1]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func encodedSize(records []string) int {
	total := 0
	for _, record := range records {
		total += len(record)
	}
	return total
}
