package dsl

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"

	configvocab "github.com/chazu/nous/internal/vocab/configrepair"
)

func init() {
	registerVocabularyWords("configrepair", map[string]builtinFn{
		"config-valid?":               bConfigValid,
		"config-schema-valid?":        bConfigSchemaValid,
		"config-canonicalize":         bConfigCanonicalize,
		"config-satisfies?":           bConfigSatisfies,
		"config-set":                  bConfigSet,
		"config-preserves-protected?": bConfigPreservesProtected,
		"config-changed-count":        bConfigChangedCount,
		"config-repair-valid?":        bConfigRepairValid,
		"config-plan-valid?":          bConfigPlanValid,
		"config-name-less?":           bConfigNameLess,
		"config-plan-name":            bConfigPlanName,
		"config-plan-defn":            bConfigPlanDefn,
		"config-artifact-name":        bConfigArtifactName,
		"config-decision-key":         bConfigDecisionKey,
	})
}

func strictStringList(value Value) ([]string, bool) {
	if value.Kind() != VList {
		return nil, false
	}
	items := value.AsList()
	out := make([]string, len(items))
	for index, item := range items {
		if item.Kind() != VString {
			return nil, false
		}
		out[index] = item.AsString()
	}
	return out, true
}

func stringListValue(values []string) Value {
	out := make([]Value, len(values))
	for index, value := range values {
		out[index] = StringVal(value)
	}
	return ListVal(out)
}

func strictString(value Value) (string, bool) {
	if value.Kind() != VString {
		return "", false
	}
	return value.AsString(), true
}

func bConfigValid(vm *VM) error {
	data, ok := strictStringList(vm.pop())
	vm.push(BoolVal(ok && configvocab.ValidConfig(data)))
	return nil
}

func bConfigSchemaValid(vm *VM) error {
	data, ok := strictStringList(vm.pop())
	vm.push(BoolVal(ok && configvocab.ValidSchema(data)))
	return nil
}

func bConfigCanonicalize(vm *VM) error {
	data, ok := strictStringList(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	result, err := configvocab.Canonicalize(data)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(stringListValue(result))
	return nil
}

func bConfigSatisfies(vm *VM) error {
	schema, schemaOK := strictStringList(vm.pop())
	configuration, configOK := strictStringList(vm.pop())
	if !schemaOK || !configOK {
		vm.push(Nil())
		return nil
	}
	result, err := configvocab.Satisfies(configuration, schema)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(result))
	return nil
}

func bConfigSet(vm *VM) error {
	value, valueOK := strictString(vm.pop())
	key, keyOK := strictString(vm.pop())
	configuration, configOK := strictStringList(vm.pop())
	if !valueOK || !keyOK || !configOK {
		vm.push(Nil())
		return nil
	}
	result, err := configvocab.Apply(configuration, configvocab.Repair{Key: key, Value: value})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(stringListValue(result))
	return nil
}

func bConfigPreservesProtected(vm *VM) error {
	schema, schemaOK := strictStringList(vm.pop())
	after, afterOK := strictStringList(vm.pop())
	before, beforeOK := strictStringList(vm.pop())
	if !schemaOK || !afterOK || !beforeOK {
		vm.push(Nil())
		return nil
	}
	result, err := configvocab.PreservesProtected(before, after, schema)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(result))
	return nil
}

func bConfigChangedCount(vm *VM) error {
	after, afterOK := strictStringList(vm.pop())
	before, beforeOK := strictStringList(vm.pop())
	if !afterOK || !beforeOK {
		vm.push(Nil())
		return nil
	}
	result, err := configvocab.ChangedKeys(before, after)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(IntVal(result))
	return nil
}

func bConfigRepairValid(vm *VM) error {
	value, valueOK := strictString(vm.pop())
	key, keyOK := strictString(vm.pop())
	vm.push(BoolVal(valueOK && keyOK && configvocab.ValidRepair(configvocab.Repair{Key: key, Value: value})))
	return nil
}

func configComponents(vm *VM, value Value) ([]string, []configvocab.Repair, bool) {
	names, ok := strictStringList(value)
	if !ok || len(names) == 0 || len(names) > configvocab.MaxPlanSize {
		return nil, nil, false
	}
	repairs := make([]configvocab.Repair, len(names))
	seenNames := make(map[string]bool, len(names))
	for index, name := range names {
		if name == "" || len(name) > 512 || seenNames[name] {
			return nil, nil, false
		}
		seenNames[name] = true
		component := vm.Store.Get(name)
		if component == nil || !vm.Store.IsA(name, "PrimitiveConfigurationRepair") || name == "PrimitiveConfigurationRepair" {
			return nil, nil, false
		}
		keyRaw, keyOK := component.Get("repairKey").(string)
		valueRaw, valueOK := component.Get("repairValue").(string)
		if !keyOK || !valueOK {
			return nil, nil, false
		}
		repairs[index] = configvocab.Repair{Key: keyRaw, Value: valueRaw}
	}
	return names, repairs, true
}

func bConfigPlanValid(vm *VM) error {
	_, repairs, ok := configComponents(vm, vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(configvocab.ValidPlan(repairs)))
	return nil
}

func bConfigNameLess(vm *VM) error {
	right, rightOK := strictString(vm.pop())
	left, leftOK := strictString(vm.pop())
	if !rightOK || !leftOK || left == "" || right == "" || len(left) > 512 || len(right) > 512 {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(left < right))
	return nil
}

func encodeConfigIdentity(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func freshConfigName(vm *VM, base string) string {
	if !vm.Store.Has(base) {
		return base
	}
	for suffix := 1; ; suffix++ {
		candidate := fmt.Sprintf("%s-collision-%d", base, suffix)
		if !vm.Store.Has(candidate) {
			return candidate
		}
	}
}

func bConfigPlanName(vm *VM) error {
	names, repairs, ok := configComponents(vm, vm.pop())
	if !ok || !configvocab.ValidPlan(repairs) {
		vm.push(Nil())
		return nil
	}
	ordered := append([]string(nil), names...)
	sort.Strings(ordered)
	parts := []string{"ConfigPlan", fmt.Sprintf("%d", len(ordered))}
	for _, name := range ordered {
		parts = append(parts, encodeConfigIdentity(name))
	}
	vm.push(StringVal(freshConfigName(vm, strings.Join(parts, "."))))
	return nil
}

func bConfigPlanDefn(vm *VM) error {
	names, repairs, ok := configComponents(vm, vm.pop())
	if !ok || !configvocab.ValidPlan(repairs) {
		vm.push(Nil())
		return nil
	}
	type component struct {
		name   string
		repair configvocab.Repair
	}
	components := make([]component, len(names))
	for index := range names {
		components[index] = component{name: names[index], repair: repairs[index]}
	}
	sort.Slice(components, func(i, j int) bool { return components[i].name < components[j].name })
	parts := make([]string, len(components))
	for index, component := range components {
		parts[index] = strconv.Quote(component.repair.Key) + " " + strconv.Quote(component.repair.Value) + " config-set"
	}
	vm.push(StringVal(strings.Join(parts, " ")))
	return nil
}

func bConfigArtifactName(vm *VM) error {
	example, exampleOK := strictString(vm.pop())
	program, programOK := strictString(vm.pop())
	kind, kindOK := strictString(vm.pop())
	if !exampleOK || !programOK || !kindOK || program == "" || kind == "" {
		vm.push(Nil())
		return nil
	}
	base := "ConfigArtifact." + encodeConfigIdentity(kind) + "." +
		encodeConfigIdentity(program) + "." + encodeConfigIdentity(example)
	vm.push(StringVal(freshConfigName(vm, base)))
	return nil
}

func bConfigDecisionKey(vm *VM) error {
	_, repairs, ok := configComponents(vm, vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	decision, err := configvocab.DecisionKey(repairs)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(decision))
	return nil
}
