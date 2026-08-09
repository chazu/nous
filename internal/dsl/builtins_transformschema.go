package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func init() {
	registerVocabularyWords("transformschema", map[string]builtinFn{
		"ts-forest-valid?":      bTSForestValid,
		"ts-schema-apply":       bTSSchemaApply,
		"ts-program-apply":      bTSProgramApply,
		"ts-refine":             bTSRefine,
		"ts-candidate-allocate": bTSCandidateAllocate,
		"ts-digest":             bTSDigest,
		"ts-node-facts":         bTSNodeFacts,
		"ts-parent-facts":       bTSParentFacts,
		"ts-target":             bTSTarget,
		"ts-make-edit":          bTSMakeEdit,
		"ts-make-program":       bTSMakeProgram,
		"ts-program-edits":      bTSProgramEdits,
		"ts-make-schema":        bTSMakeSchema,
		"ts-meter":              bTSMeter,
	})
}

func bTSCandidateAllocate(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	data := []byte(value.AsString())
	if _, err := transformschema.ParsePartial(data); err != nil {
		if _, schemaErr := transformschema.ParseSchema(data); schemaErr != nil {
			if recordErr := recordTransform(vm, "candidate-allocate", "rejected", 4, [][]byte{data}, nil); recordErr != nil {
				return recordErr
			}
			vm.push(Nil())
			return nil
		}
	}
	if err := recordTransform(vm, "candidate-allocate", "allocated", 4, [][]byte{data}, [][]byte{data}); err != nil {
		return err
	}
	vm.push(value)
	return nil
}

type TransformMeterRecord struct {
	Category  uint8
	Operation string
	Subject   string
	Object    string
	Outcome   string
	Phase     string
	Inputs    [][]byte
	Outputs   [][]byte
}

type transformMeter struct {
	sync.Mutex
	records []TransformMeterRecord
}

var transformMeters = struct {
	sync.Mutex
	items map[string]*transformMeter
}{items: map[string]*transformMeter{}}

func RegisterTransformMeter(token string) error {
	if token == "" {
		return errors.New("empty transformation meter token")
	}
	transformMeters.Lock()
	defer transformMeters.Unlock()
	if transformMeters.items[token] != nil {
		return errors.New("duplicate transformation meter token")
	}
	transformMeters.items[token] = &transformMeter{}
	return nil
}

func UnregisterTransformMeter(token string) {
	transformMeters.Lock()
	delete(transformMeters.items, token)
	transformMeters.Unlock()
}

func TransformMeterSnapshot(token string) ([]TransformMeterRecord, error) {
	transformMeters.Lock()
	m := transformMeters.items[token]
	transformMeters.Unlock()
	if m == nil {
		return nil, errors.New("unknown transformation meter")
	}
	m.Lock()
	defer m.Unlock()
	return slices.Clone(m.records), nil
}

func chargeTransform(token, operation, subject, object, outcome string, category int) error {
	if category < 0 || category > 11 || operation == "" || subject == "" || object == "" || outcome == "" {
		return errors.New("invalid transformation meter record")
	}
	transformMeters.Lock()
	m := transformMeters.items[token]
	transformMeters.Unlock()
	if m == nil {
		return errors.New("unknown transformation meter")
	}
	m.Lock()
	defer m.Unlock()
	m.records = append(m.records, TransformMeterRecord{Category: uint8(category), Operation: operation, Subject: subject, Object: object, Outcome: outcome})
	return nil
}

func recordTransform(vm *VM, operation, outcome string, category int, inputs, outputs [][]byte) error {
	if vm.CurrentTask == nil || vm.Store == nil {
		return nil
	}
	current := vm.Store.Get(vm.CurrentTask.UnitName)
	if current == nil {
		return nil
	}
	experiment := current
	if name := current.GetString("experiment"); name != "" {
		experiment = vm.Store.Get(name)
	}
	if experiment == nil || experiment.GetString("meterToken") == "" {
		return nil
	}
	phase := map[string]string{"tsAcquire": "acquire", "tsRefine": "target", "tsClose": "training-validate"}[vm.CurrentTask.SlotName]
	if vm.CurrentTask.SlotName == "tsEvaluateFactor" {
		phase = current.GetString("stage")
	}
	if phase == "" {
		return errors.New("unknown transformation semantic phase")
	}
	token := experiment.GetString("meterToken")
	transformMeters.Lock()
	m := transformMeters.items[token]
	transformMeters.Unlock()
	if m == nil {
		return errors.New("unknown transformation meter")
	}
	clone := func(values [][]byte) [][]byte {
		out := make([][]byte, len(values))
		for i := range values {
			out[i] = slices.Clone(values[i])
		}
		return out
	}
	subject, object := "", ""
	if len(inputs) > 0 {
		subject = transformDigest(inputs[0])
	}
	if len(outputs) > 0 {
		object = transformDigest(outputs[0])
	}
	m.Lock()
	m.records = append(m.records, TransformMeterRecord{uint8(category), operation, subject, object, outcome, phase, clone(inputs), clone(outputs)})
	m.Unlock()
	return nil
}

func transformDigest(value []byte) string {
	d := sha256.Sum256(value)
	return hex.EncodeToString(d[:])
}

func transformAtom(kind string, value any) []byte {
	b, _ := json.Marshal([]any{"transform-atom/v1", kind, value})
	return b
}

func ChargeTransformMeter(token, operation, subject, object, outcome string, category int) error {
	return chargeTransform(token, operation, subject, object, outcome, category)
}

func bTSMeter(vm *VM) error {
	outcome, object, subject, operation, category, token := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if outcome.Kind() != VString || object.Kind() != VString || subject.Kind() != VString || operation.Kind() != VString || category.Kind() != VInt || token.Kind() != VString {
		return fmt.Errorf("invalid transformation meter operands token=%v category=%v operation=%v subject=%v object=%v outcome=%v", token.Kind(), category.Kind(), operation.Kind(), subject.Kind(), object.Kind(), outcome.Kind())
	}
	if err := chargeTransform(token.AsString(), operation.AsString(), subject.AsString(), object.AsString(), outcome.AsString(), category.AsInt()); err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func bTSForestValid(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	_, err := transformschema.ParseForest([]byte(value.AsString()))
	vm.push(BoolVal(err == nil))
	return nil
}

func bTSSchemaApply(vm *VM) error {
	schemaValue, forestValue := vm.pop(), vm.pop()
	if schemaValue.Kind() != VString || forestValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	f, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	s, err := transformschema.ParseSchema([]byte(schemaValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	r, _ := s.Apply(f)
	output := ""
	if r.Output != nil {
		b, _ := r.Output.CanonicalJSON()
		output = string(b)
	}
	resultWire, _ := json.Marshal([]any{"transform-result/v1", r.Terminal, transformDigest([]byte(output))})
	if output == "" {
		resultWire, _ = json.Marshal([]any{"transform-result/v1", r.Terminal, ""})
	}
	applicationWire, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(resultWire), nil})
	if err := recordTransform(vm, "schema-application", r.Terminal, 11, [][]byte{[]byte(forestValue.AsString()), []byte(schemaValue.AsString())}, [][]byte{applicationWire}); err != nil {
		return err
	}
	vm.push(ListVal([]Value{StringVal(r.Terminal), StringVal(output)}))
	return nil
}

func bTSProgramApply(vm *VM) error {
	programValue, forestValue := vm.pop(), vm.pop()
	if programValue.Kind() != VString || forestValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	f, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	p, err := transformschema.ParseProgram([]byte(programValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	out, err := p.Apply(f)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	b, _ := out.CanonicalJSON()
	vm.push(StringVal(string(b)))
	return nil
}

func bTSRefine(vm *VM) error {
	value, partialValue := vm.pop(), vm.pop()
	if value.Kind() != VString || partialValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	p, err := transformschema.ParsePartial([]byte(partialValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	next, err := p.Refine(value.AsString())
	if err != nil {
		vm.push(Nil())
		return nil
	}
	encoded, err := next.CanonicalJSON()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordTransform(vm, "refine", "refined", 5, [][]byte{[]byte(partialValue.AsString()), transformAtom("enum", value.AsString())}, [][]byte{encoded}); err != nil {
		return err
	}
	vm.push(StringVal(string(encoded)))
	return nil
}

func bTSDigest(vm *VM) error {
	v := vm.pop()
	var material any
	switch v.Kind() {
	case VString:
		material = v.AsString()
	case VInt:
		material = v.AsInt()
	case VBool:
		material = v.AsBool()
	case VList:
		material = tsSerializable(v)
	case VNil:
		material = nil
	default:
		vm.push(Nil())
		return nil
	}
	b, err := json.Marshal(material)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	d := sha256.Sum256(b)
	vm.push(StringVal(hex.EncodeToString(d[:])))
	return nil
}

func tsForestAndID(vm *VM) (transformschema.Forest, int, bool) {
	idValue, forestValue := vm.pop(), vm.pop()
	if idValue.Kind() != VInt || forestValue.Kind() != VString {
		return transformschema.Forest{}, 0, false
	}
	f, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	return f, idValue.AsInt(), err == nil
}

func tsFindNode(f transformschema.Forest, id int) (transformschema.Node, bool) {
	for _, n := range f.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return transformschema.Node{}, false
}

func bTSNodeFacts(vm *VM) error {
	f, id, ok := tsForestAndID(vm)
	forest, _ := f.CanonicalJSON()
	inputs := [][]byte{forest, transformAtom("id", id)}
	n, found := tsFindNode(f, id)
	if !ok || !found {
		if err := recordTransform(vm, "node", "invalid-input", 0, inputs, nil); err != nil {
			return err
		}
		vm.push(Nil())
		return nil
	}
	output, _ := json.Marshal([]any{"transform-node-facts/v1", n.Kind, n.Value, n.From, n.To})
	if err := recordTransform(vm, "node", "ok", 0, inputs, [][]byte{output}); err != nil {
		return err
	}
	vm.push(ListVal([]Value{StringVal(n.Kind), StringVal(n.Value), StringVal(n.From), StringVal(n.To)}))
	return nil
}

func bTSParentFacts(vm *VM) error {
	f, id, ok := tsForestAndID(vm)
	forest, _ := f.CanonicalJSON()
	inputs := [][]byte{forest, transformAtom("id", id)}
	n, found := tsFindNode(f, id)
	if !ok || !found || n.Parent < 0 {
		outcome := "absent"
		if !ok || !found {
			outcome = "invalid-input"
		}
		if err := recordTransform(vm, "parent", outcome, 1, inputs, nil); err != nil {
			return err
		}
		vm.push(Nil())
		return nil
	}
	output, _ := json.Marshal([]any{"transform-parent-facts/v1", n.Parent, n.Key})
	if err := recordTransform(vm, "parent", "ok", 1, inputs, [][]byte{output}); err != nil {
		return err
	}
	vm.push(ListVal([]Value{IntVal(n.Parent), StringVal(n.Key)}))
	return nil
}

func bTSTarget(vm *VM) error {
	f, id, ok := tsForestAndID(vm)
	forest, _ := f.CanonicalJSON()
	inputs := [][]byte{forest, transformAtom("id", id)}
	n, found := tsFindNode(f, id)
	if !ok || !found || n.Target < 0 {
		outcome := "absent"
		if !ok || !found {
			outcome = "invalid-input"
		}
		if err := recordTransform(vm, "target", outcome, 2, inputs, nil); err != nil {
			return err
		}
		vm.push(Nil())
		return nil
	}
	output := transformAtom("id", n.Target)
	if err := recordTransform(vm, "target", "ok", 2, inputs, [][]byte{output}); err != nil {
		return err
	}
	vm.push(IntVal(n.Target))
	return nil
}

func bTSMakeEdit(vm *VM) error {
	value, id := vm.pop(), vm.pop()
	if value.Kind() != VString || id.Kind() != VInt {
		vm.push(Nil())
		return nil
	}
	p := transformschema.Program{Edits: []transformschema.Edit{{Target: id.AsInt(), Value: value.AsString()}}}
	b, err := p.CanonicalJSON()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	var wire []any
	if json.Unmarshal(b, &wire) != nil {
		vm.push(Nil())
		return nil
	}
	edit, _ := json.Marshal(wire[1].([]any)[0])
	vm.push(StringVal(string(edit)))
	return nil
}

func bTSMakeProgram(vm *VM) error {
	list := vm.pop()
	if list.Kind() != VList {
		vm.push(Nil())
		return nil
	}
	var edits []transformschema.Edit
	for _, value := range list.AsList() {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
		var row []any
		if json.Unmarshal([]byte(value.AsString()), &row) != nil || len(row) != 3 || row[0] != "set-value/v1" {
			vm.push(Nil())
			return nil
		}
		id, idOK := row[1].(float64)
		literal, literalOK := row[2].(string)
		if !idOK || !literalOK || id != float64(int(id)) {
			vm.push(Nil())
			return nil
		}
		edits = append(edits, transformschema.Edit{Target: int(id), Value: literal})
	}
	slices.SortFunc(edits, func(a, b transformschema.Edit) int { return a.Target - b.Target })
	b, err := (transformschema.Program{Edits: edits}).CanonicalJSON()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(string(b)))
	return nil
}

func bTSProgramEdits(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	p, err := transformschema.ParseProgram([]byte(value.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	rows := make([]Value, len(p.Edits))
	for i, edit := range p.Edits {
		rows[i] = ListVal([]Value{IntVal(edit.Target), StringVal(edit.Value)})
	}
	vm.push(ListVal(rows))
	return nil
}

func bTSMakeSchema(vm *VM) error {
	locality, guard, scope, targets, anchor := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{anchor, targets, scope, guard, locality}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	s := transformschema.Schema{Anchor: values[0].AsString(), Targets: values[1].AsString(), ReferenceScope: values[2].AsString(), OldGuard: values[3].AsString(), Locality: values[4].AsString()}
	b, err := s.CanonicalJSON()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(string(b)))
	return nil
}

func tsSerializable(v Value) any {
	if v.Kind() != VList {
		return nil
	}
	values := v.AsList()
	out := make([]any, len(values))
	for i, value := range values {
		switch value.Kind() {
		case VString:
			out[i] = value.AsString()
		case VInt:
			out[i] = value.AsInt()
		case VBool:
			out[i] = value.AsBool()
		case VList:
			out[i] = tsSerializable(value)
		default:
			out[i] = nil
		}
	}
	return out
}
