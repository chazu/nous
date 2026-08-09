package dsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func init() {
	registerVocabularyWords("transformschema", map[string]builtinFn{
		"ts-forest-valid?":      bTSForestValid,
		"ts-schema-apply":       bTSSchemaApply,
		"ts-program-apply":      bTSProgramApply,
		"ts-refine":             bTSRefine,
		"ts-candidate-allocate": bTSCandidateAllocate,
		"ts-output-compare":     bTSOutputCompare,
		"ts-digest":             bTSDigest,
		"ts-node-facts":         bTSNodeFacts,
		"ts-parent-facts":       bTSParentFacts,
		"ts-target":             bTSTarget,
		"ts-make-edit":          bTSMakeEdit,
		"ts-make-program":       bTSMakeProgram,
		"ts-program-edits":      bTSProgramEdits,
		"ts-make-schema":        bTSMakeSchema,
		"ts-close-stage":        bTSCloseStage,
		"ts-freeze-schema":      bTSFreezeSchema,
		"ts-eq":                 bTSEqual,
		"ts-id-eq":              bTSIDEqual,
		"ts-id-set-eq":          bTSIDSetEqual,
		"ts-factor-result":      bTSFactorResult,
		"ts-meter":              bTSMeter,
	})
}

func bTSEqual(vm *VM) error {
	right, left := vm.pop(), vm.pop()
	leftBytes, leftOK := transformComparisonAtom(left)
	rightBytes, rightOK := transformComparisonAtom(right)
	if !leftOK || !rightOK {
		vm.push(BoolVal(false))
		return nil
	}
	if transformAtomKind(leftBytes) != transformAtomKind(rightBytes) {
		if err := recordTransform(vm, "compare", "invalid-input", 3, [][]byte{leftBytes, rightBytes}, nil); err != nil {
			return err
		}
		vm.push(BoolVal(false))
		return nil
	}
	equal := bytes.Equal(leftBytes, rightBytes)
	outcome := "false"
	if equal {
		outcome = "true"
	}
	if err := recordTransform(vm, "compare", outcome, 3, [][]byte{leftBytes, rightBytes}, [][]byte{transformAtom("boolean", equal)}); err != nil {
		return err
	}
	vm.push(BoolVal(equal))
	return nil
}

func bTSIDEqual(vm *VM) error {
	return typedTransformEqual(vm, "id")
}

func bTSIDSetEqual(vm *VM) error {
	return typedTransformEqual(vm, "id-set")
}

func typedTransformEqual(vm *VM, kind string) error {
	right, left, forestValue := vm.pop(), vm.pop(), vm.pop()
	if forestValue.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	forest, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	if err != nil {
		vm.push(BoolVal(false))
		return nil
	}
	canonicalForest, _ := forest.CanonicalJSON()
	forestDigest := transformDigest(canonicalForest)
	encode := func(value Value) ([]byte, bool) {
		switch kind {
		case "id":
			if value.Kind() != VInt || value.AsInt() < -1 || value.AsInt() >= transformschema.MaxNodes {
				return nil, false
			}
			return boundedTransformComparisonAtom("scoped-id", []any{forestDigest, value.AsInt()})
		case "id-set":
			if value.Kind() != VList {
				return nil, false
			}
			serial, ok := tsSerializable(value).([]any)
			if !ok || len(serial) > 6 {
				return nil, false
			}
			ids := make([]int, len(serial))
			for index, item := range serial {
				id, ok := item.(int)
				if !ok || id < 0 || id >= transformschema.MaxNodes || index > 0 && id <= ids[index-1] {
					return nil, false
				}
				ids[index] = id
			}
			return boundedTransformComparisonAtom("scoped-id-set", []any{forestDigest, ids})
		}
		return nil, false
	}
	leftBytes, leftOK := encode(left)
	rightBytes, rightOK := encode(right)
	if !leftOK || !rightOK {
		vm.push(BoolVal(false))
		return nil
	}
	equal := bytes.Equal(leftBytes, rightBytes)
	outcome := "false"
	if equal {
		outcome = "true"
	}
	if err := recordTransform(vm, "compare", outcome, 3, [][]byte{leftBytes, rightBytes}, [][]byte{transformAtom("boolean", equal)}); err != nil {
		return err
	}
	vm.push(BoolVal(equal))
	return nil
}

func transformAtomKind(data []byte) string {
	var wire []json.RawMessage
	var kind string
	if json.Unmarshal(data, &wire) != nil || len(wire) != 3 || json.Unmarshal(wire[1], &kind) != nil {
		return ""
	}
	return kind
}

func bTSFactorResult(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VBool {
		vm.push(BoolVal(false))
		return nil
	}
	result := value.AsBool()
	resultBytes := transformAtom("boolean", result)
	outcome := "false"
	if result {
		outcome = "true"
	}
	if err := recordTransform(vm, "compare", outcome, 3, [][]byte{resultBytes, transformAtom("boolean", true)}, [][]byte{resultBytes}); err != nil {
		return err
	}
	if err := recordTransform(vm, "evidence-link", "attached", 10, [][]byte{resultBytes}, nil); err != nil {
		return err
	}
	vm.push(BoolVal(result))
	return nil
}

func transformComparisonAtom(value Value) ([]byte, bool) {
	switch value.Kind() {
	case VString:
		return boundedTransformComparisonAtom("enum", value.AsString())
	case VInt:
		return boundedTransformComparisonAtom("count", value.AsInt())
	case VBool:
		return boundedTransformComparisonAtom("boolean", value.AsBool())
	case VNil:
		return boundedTransformComparisonAtom("enum", nil)
	case VList:
		serial := tsSerializable(value)
		if values, ok := serial.([]any); ok {
			ids := make([]int, len(values))
			idSet := true
			for index, item := range values {
				id, ok := item.(int)
				if !ok || id < 0 || id >= transformschema.MaxNodes || len(values) > 6 || index > 0 && id <= ids[index-1] {
					idSet = false
					break
				}
				ids[index] = id
			}
			if idSet {
				return boundedTransformComparisonAtom("id-set", ids)
			}
		}
		return boundedTransformComparisonAtom("scalar", serial)
	default:
		return nil, false
	}
}

func boundedTransformComparisonAtom(kind string, value any) ([]byte, bool) {
	atom := transformAtom(kind, value)
	if len(atom) <= 128 {
		return atom, true
	}
	preimage, err := json.Marshal([]any{"transform-comparison-preimage/v1", kind, value})
	if err != nil {
		return nil, false
	}
	return transformAtom("digest", transformDigest(preimage)), true
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
	records              []TransformMeterRecord
	work                 int64
	applications         int
	reserved             bool
	reservedPhase        string
	reservedStartWork    int64
	reservedMaximumWork  int64
	applicationCommitted bool
	trainingPositive     bool
	waitingForComparison bool
}

var ErrTransformBudgetExhausted = errors.New("transformation application budget exhausted")

const transformLifecycleWorkCap int64 = 12000

var transformLifecycleCharges = [12]int64{1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1}

var transformMeters = struct {
	sync.Mutex
	items map[string]*transformMeter
}{items: map[string]*transformMeter{}}

func RegisterTransformMeter(token string) error {
	return RegisterTransformMeterWithRecords(token, nil)
}

func RegisterTransformMeterWithRecords(token string, records []TransformMeterRecord) error {
	if token == "" {
		return errors.New("empty transformation meter token")
	}
	transformMeters.Lock()
	defer transformMeters.Unlock()
	if transformMeters.items[token] != nil {
		return errors.New("duplicate transformation meter token")
	}
	cloned := make([]TransformMeterRecord, len(records))
	for i, record := range records {
		cloned[i] = record
		cloned[i].Inputs = cloneTransformBytes(record.Inputs)
		cloned[i].Outputs = cloneTransformBytes(record.Outputs)
	}
	meter := &transformMeter{records: cloned}
	for _, record := range cloned {
		if record.Category >= uint8(len(transformLifecycleCharges)) {
			return errors.New("invalid transformation meter category")
		}
		meter.work += transformLifecycleCharges[record.Category]
		if record.Operation == "schema-application" || record.Operation == "replay-application" {
			meter.applications++
		}
	}
	if meter.work >= transformLifecycleWorkCap || meter.applications > 48 {
		return errors.New("preloaded transformation meter exceeds lifecycle budget")
	}
	transformMeters.items[token] = meter
	return nil
}

func cloneTransformBytes(values [][]byte) [][]byte {
	out := make([][]byte, len(values))
	for i := range values {
		out[i] = slices.Clone(values[i])
	}
	return out
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

// RecordTransformProgramBatchVerification is an orchestration-only adapter.
// It is intentionally not registered as a CUE word: the closed acquisition
// barrier serializes its four promoted rows without returning the batch to the
// policy VM.
func RecordTransformProgramBatchVerification(token string, batch []byte) error {
	transformMeters.Lock()
	m := transformMeters.items[token]
	transformMeters.Unlock()
	if m == nil {
		return errors.New("unknown transformation meter")
	}
	if _, err := transformfixturecore.ParseProgramBatch(batch); err != nil {
		return err
	}
	m.Lock()
	defer m.Unlock()
	charge := transformLifecycleCharges[11]
	if m.reserved || m.work+charge >= transformLifecycleWorkCap {
		return errors.New("program batch verification outside closed acquisition budget")
	}
	m.records = append(m.records, TransformMeterRecord{Category: 11, Operation: "verify", Phase: "acquire", Outcome: "verified", Inputs: [][]byte{slices.Clone(batch)}, Outputs: [][]byte{transformAtom("boolean", true)}})
	m.work += charge
	return nil
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
	phase := map[string]string{"tsAcquire": "acquire", "tsRefine": "target", "tsClose": "training-validate", "tsHeldout": "heldout"}[vm.CurrentTask.SlotName]
	if vm.CurrentTask.SlotName == "tsEvaluateFactor" {
		phase = current.GetString("stage")
	}
	if phase == "" {
		return errors.New("unknown transformation semantic phase")
	}
	return recordTransformAtPhase(vm, phase, operation, outcome, category, inputs, outputs)
}

func recordTransformAtPhase(vm *VM, phase, operation, outcome string, category int, inputs, outputs [][]byte) error {
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
	token := experiment.GetString("meterToken")
	transformMeters.Lock()
	m := transformMeters.items[token]
	transformMeters.Unlock()
	if m == nil {
		return errors.New("unknown transformation meter")
	}
	subject, object := "", ""
	if len(inputs) > 0 {
		subject = transformDigest(inputs[0])
	}
	if len(outputs) > 0 {
		object = transformDigest(outputs[0])
	}
	m.Lock()
	defer m.Unlock()
	if category < 0 || category >= len(transformLifecycleCharges) {
		return errors.New("invalid transformation meter category")
	}
	charge := transformLifecycleCharges[category]
	if m.work+charge >= transformLifecycleWorkCap {
		return ErrTransformBudgetExhausted
	}
	if m.reserved && m.work+charge > m.reservedStartWork+m.reservedMaximumWork {
		return errors.New("transformation application exceeded reserved work")
	}
	if operation == "schema-application" || operation == "replay-application" {
		if !m.reserved || m.reservedPhase != phase || m.applicationCommitted {
			return errors.New("unreserved transformation application")
		}
		m.applicationCommitted = true
		m.waitingForComparison = phase == "training-validate" && m.trainingPositive && outcome == "applied"
		m.applications++
	}
	m.records = append(m.records, TransformMeterRecord{uint8(category), operation, subject, object, outcome, phase, cloneTransformBytes(inputs), cloneTransformBytes(outputs)})
	m.work += charge
	endpoint := operation == "evidence-link" && !m.waitingForComparison
	if operation == "output-compare" && m.waitingForComparison && transformOutputComparisonIsEndpoint(inputs) {
		endpoint = true
	}
	if endpoint && m.reserved && m.applicationCommitted {
		m.reserved = false
		m.reservedPhase = ""
		m.reservedStartWork = 0
		m.reservedMaximumWork = 0
		m.applicationCommitted = false
		m.trainingPositive = false
		m.waitingForComparison = false
	}
	return nil
}

func transformOutputComparisonIsEndpoint(inputs [][]byte) bool {
	if len(inputs) != 3 {
		return false
	}
	forest, err := transformschema.ParseForest(inputs[1])
	if err != nil || len(forest.Nodes) == 0 {
		return false
	}
	var atom []any
	if err := json.Unmarshal(inputs[2], &atom); err != nil || len(atom) != 3 || atom[0] != "transform-atom/v1" || atom[1] != "id" {
		return false
	}
	idValue, ok := atom[2].(float64)
	if !ok {
		return false
	}
	maxID := forest.Nodes[0].ID
	for _, node := range forest.Nodes[1:] {
		if node.ID > maxID {
			maxID = node.ID
		}
	}
	return int(idValue) == maxID && idValue == float64(int(idValue))
}

func reserveTransformApplication(vm *VM, maximumWork int64, trainingPositive bool) error {
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
	phase := map[string]string{"tsAcquire": "acquire", "tsRefine": "target", "tsClose": "training-validate", "tsHeldout": "heldout"}[vm.CurrentTask.SlotName]
	if phase == "" {
		return errors.New("unknown transformation application phase")
	}
	transformMeters.Lock()
	m := transformMeters.items[experiment.GetString("meterToken")]
	transformMeters.Unlock()
	if m == nil {
		return errors.New("unknown transformation meter")
	}
	m.Lock()
	defer m.Unlock()
	applicationCap := 48
	if phase != "heldout" {
		applicationCap = 40
	}
	if m.reserved {
		return errors.New("nested transformation application reservation")
	}
	if maximumWork <= 0 || m.applications >= applicationCap || m.work+maximumWork >= transformLifecycleWorkCap {
		return ErrTransformBudgetExhausted
	}
	m.reserved = true
	m.reservedPhase = phase
	m.reservedStartWork = m.work
	m.reservedMaximumWork = maximumWork
	m.applicationCommitted = false
	m.trainingPositive = trainingPositive
	m.waitingForComparison = false
	return nil
}

func bTSCloseStage(vm *VM) error {
	stageValue, experimentValue := vm.pop(), vm.pop()
	if stageValue.Kind() != VString || experimentValue.Kind() != VString || vm.Store == nil || vm.CurrentTask == nil {
		vm.push(BoolVal(false))
		return nil
	}
	stage := stageValue.AsString()
	stageIndex := map[string]int{"target": 0, "anchor": 1, "scope": 2, "old-guard": 3, "locality": 4}[stage]
	if stageIndex == 0 && stage != "target" {
		vm.push(BoolVal(false))
		return nil
	}
	experiment := vm.Store.Get(experimentValue.AsString())
	if experiment == nil || experiment.Name != vm.CurrentTask.UnitName {
		vm.push(BoolVal(false))
		return nil
	}
	for _, name := range experiment.GetStrings("candidateUnits") {
		candidate := vm.Store.Get(name)
		if candidate != nil && candidate.GetString("stage") == stage && (candidate.GetString("status") == "pending" || candidate.GetString("evidenceUnit") == "") {
			vm.push(BoolVal(false))
			return nil
		}
	}
	closureSlot := "meteredClosure." + stage
	if experiment.GetBool(closureSlot + ".done") {
		vm.push(BoolVal(experiment.GetBool(closureSlot + ".valid")))
		return nil
	}
	type alternative struct {
		partial, result, status string
	}
	var alternatives []alternative
	parentDigest := ""
	survivorDigest := ""
	valid := true
	for _, name := range experiment.GetStrings("candidateUnits") {
		candidate := vm.Store.Get(name)
		if candidate == nil || candidate.GetString("stage") != stage {
			continue
		}
		partial := []byte(candidate.GetString("partial"))
		parsed, err := transformschema.ParsePartial(partial)
		if err != nil || parsed.Stage != stageIndex+1 {
			valid = false
			continue
		}
		parent := vm.Store.Get(candidate.GetString("parentCandidate"))
		if parent == nil {
			valid = false
			continue
		}
		parentPartial := []byte(parent.GetString("partial"))
		parsedParent, err := transformschema.ParsePartial(parentPartial)
		if err != nil || parsedParent.Stage != stageIndex {
			valid = false
			continue
		}
		thisParent := transformDigest(parentPartial)
		if parentDigest == "" {
			parentDigest = thisParent
		} else if parentDigest != thisParent {
			valid = false
		}
		evidence := vm.Store.Get(candidate.GetString("evidenceUnit"))
		matched := evidence != nil && evidence.GetBool("matched")
		status := "counterexample"
		if oneOfTransformString(candidate.GetString("disposition"), "ablated-ineligible", "redundant-noncanonical") {
			status = candidate.GetString("disposition")
			matched = false
		} else if candidate.GetString("status") == "survivor" && matched {
			status = "survivor"
			survivorDigest = transformDigest(partial)
		} else if candidate.GetString("status") == "survivor" || matched {
			valid = false
		}
		result := transformAtom("boolean", matched)
		alternatives = append(alternatives, alternative{transformDigest(partial), transformDigest(result), status})
	}
	slices.SortFunc(alternatives, func(a, b alternative) int { return bytes.Compare([]byte(a.partial), []byte(b.partial)) })
	wantAlternatives := []int{3, 3, 2, 2, 2}[stageIndex]
	if len(alternatives) != wantAlternatives || survivorDigest == "" {
		valid = false
	}
	rows := make([]any, len(alternatives))
	for index, alternative := range alternatives {
		rows[index] = []any{alternative.partial, alternative.result, alternative.status}
	}
	closure, _ := json.Marshal([]any{"transform-closure/v1", stage, parentDigest, rows, survivorDigest})
	outcome := "verified"
	if !valid {
		outcome = "rejected"
	}
	if err := recordTransformAtPhase(vm, "freeze", "verify", outcome, 11, [][]byte{closure}, [][]byte{transformAtom("boolean", valid)}); err != nil {
		return err
	}
	experiment.Set(closureSlot+".valid", valid)
	experiment.Set(closureSlot+".done", true)
	vm.push(BoolVal(valid))
	return nil
}

func oneOfTransformString(value string, options ...string) bool {
	return slices.Contains(options, value)
}

func bTSFreezeSchema(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	data := []byte(value.AsString())
	if _, err := transformschema.ParseSchema(data); err != nil {
		if recordErr := recordTransformAtPhase(vm, "freeze", "verify", "rejected", 11, [][]byte{data}, [][]byte{transformAtom("boolean", false)}); recordErr != nil {
			return recordErr
		}
		vm.push(Nil())
		return nil
	}
	if err := recordTransformAtPhase(vm, "freeze", "verify", "verified", 11, [][]byte{data}, [][]byte{transformAtom("boolean", true)}); err != nil {
		return err
	}
	vm.push(value)
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
	kindValue, schemaValue, forestValue := vm.pop(), vm.pop(), vm.pop()
	if kindValue.Kind() != VString || schemaValue.Kind() != VString || forestValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	kind := kindValue.AsString()
	if kind != "positive" && kind != "abstain" {
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
	maximumWork := int64(68)
	if kind == "positive" {
		maximumWork = 80
	}
	if err := reserveTransformApplication(vm, maximumWork, kind == "positive"); err != nil {
		if errors.Is(err, ErrTransformBudgetExhausted) {
			vm.push(ListVal([]Value{StringVal("budget-exhausted"), StringVal("")}))
			return nil
		}
		return err
	}
	r, _ := s.Apply(f)
	output := ""
	if r.Output != nil {
		b, _ := r.Output.CanonicalJSON()
		output = string(b)
	}
	if err := recordSchemaApplication(vm, f, s, r, []byte(forestValue.AsString()), []byte(schemaValue.AsString()), []byte(output)); err != nil {
		return err
	}
	vm.push(ListVal([]Value{StringVal(r.Terminal), StringVal(output)}))
	return nil
}

func ExecuteTransformSchemaApplication(vm *VM, forestBytes, schemaBytes []byte) (string, []byte, error) {
	f, err := transformschema.ParseForest(forestBytes)
	if err != nil {
		return "invalid-input", nil, err
	}
	s, err := transformschema.ParseSchema(schemaBytes)
	if err != nil {
		return "invalid-input", nil, err
	}
	if err := reserveTransformApplication(vm, 68, false); err != nil {
		if errors.Is(err, ErrTransformBudgetExhausted) {
			return "budget-exhausted", nil, nil
		}
		return "invalid-input", nil, err
	}
	r, err := s.Apply(f)
	if err != nil {
		return "invalid-input", nil, err
	}
	var output []byte
	if r.Output != nil {
		output, _ = r.Output.CanonicalJSON()
	}
	if err := recordSchemaApplication(vm, f, s, r, forestBytes, schemaBytes, output); err != nil {
		return "invalid-input", nil, err
	}
	return r.Terminal, output, nil
}

func recordSchemaApplication(vm *VM, f transformschema.Forest, s transformschema.Schema, r transformschema.Result, forestBytes, schemaBytes, outputBytes []byte) error {
	if vm.CurrentTask == nil {
		return nil
	}
	start := transformMeterLength(vm)
	byID := map[int]transformschema.Node{}
	var requests, definitions, references []transformschema.Node
	for _, node := range f.Nodes {
		byID[node.ID] = node
		switch node.Kind {
		case "request":
			requests = append(requests, node)
		case "definition":
			definitions = append(definitions, node)
		case "reference":
			references = append(references, node)
		}
		nodeFacts, _ := json.Marshal([]any{"transform-node-facts/v1", node.Kind, node.Value, node.From, node.To})
		if err := recordTransform(vm, "node", "ok", 0, [][]byte{forestBytes, transformAtom("id", node.ID)}, [][]byte{nodeFacts}); err != nil {
			return err
		}
	}
	var guards []bool
	predicate := func(selector string, subject []byte, result bool) error {
		outcome := "false"
		if result {
			outcome = "true"
		}
		guards = append(guards, result)
		return recordTransform(vm, "schema-predicate", outcome, 8, [][]byte{forestBytes, schemaBytes, transformAtom("selector", selector), subject}, [][]byte{transformAtom("boolean", result)})
	}
	finish := func(requestID, definitionID int, referenceIDs []int, editDigests []string) error {
		if referenceIDs == nil {
			referenceIDs = []int{}
		}
		if editDigests == nil {
			editDigests = []string{}
		}
		outputDigest := ""
		if len(outputBytes) != 0 {
			outputDigest = transformDigest(outputBytes)
		}
		resultWire, _ := json.Marshal([]any{"transform-result/v1", r.Terminal, outputDigest})
		last := transformMeterLength(vm) + 1
		certificate, _ := json.Marshal([]any{"transform-certificate/v1", transformDigest(schemaBytes), transformDigest(forestBytes), requestID, definitionID, referenceIDs, guards, editDigests, outputDigest, r.Terminal, start, last})
		application, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(resultWire), json.RawMessage(certificate)})
		if err := recordTransform(vm, "schema-application", r.Terminal, 11, [][]byte{forestBytes, schemaBytes}, [][]byte{application}); err != nil {
			return err
		}
		return recordTransform(vm, "evidence-link", "attached", 10, [][]byte{resultWire}, nil)
	}
	if err := predicate("request-count", transformAtom("count", len(requests)), len(requests) == 1); err != nil {
		return err
	}
	if len(requests) != 1 {
		return finish(-1, -1, nil, nil)
	}
	request := requests[0]
	requestParent, _ := json.Marshal([]any{"transform-parent-facts/v1", request.Parent, request.Key})
	if err := recordTransform(vm, "parent", "ok", 1, [][]byte{forestBytes, transformAtom("id", request.ID)}, [][]byte{requestParent}); err != nil {
		return err
	}
	if s.Anchor == "request-target" {
		if err := recordTransform(vm, "target", "ok", 2, [][]byte{forestBytes, transformAtom("id", request.ID)}, [][]byte{transformAtom("id", request.Target)}); err != nil {
			return err
		}
	}
	var candidates []transformschema.Node
	for _, candidate := range definitions {
		if s.Anchor == "request-target" && candidate.ID == request.Target || s.Anchor == "from-value" && candidate.Value == request.From || s.Anchor == "first-local" && candidate.Parent == request.Parent {
			candidates = append(candidates, candidate)
			if s.Anchor == "first-local" {
				break
			}
		}
	}
	if err := predicate("anchor-candidate", transformAtom("count", len(candidates)), len(candidates) == 1); err != nil {
		return err
	}
	if s.Anchor != "request-target" {
		candidateID := -1
		if len(candidates) != 0 {
			candidateID = candidates[0].ID
		}
		if err := predicate("anchor-candidate", transformAtom("id", candidateID), len(candidates) == 1); err != nil {
			return err
		}
	}
	if len(candidates) != 1 {
		return finish(request.ID, -1, nil, nil)
	}
	definition := candidates[0]
	definitionParent, _ := json.Marshal([]any{"transform-parent-facts/v1", definition.Parent, definition.Key})
	if err := recordTransform(vm, "parent", "ok", 1, [][]byte{forestBytes, transformAtom("id", definition.ID)}, [][]byte{definitionParent}); err != nil {
		return err
	}
	if s.Anchor == "first-local" {
		if err := recordTransform(vm, "parent", "ok", 1, [][]byte{forestBytes, transformAtom("id", definition.ID)}, [][]byte{definitionParent}); err != nil {
			return err
		}
	}
	local := s.Locality == "none" || definition.Parent == request.Parent
	if err := predicate("anchor-locality", transformAtom("id", definition.ID), local); err != nil {
		return err
	}
	if !local {
		return finish(request.ID, definition.ID, nil, nil)
	}
	var selectedReferences []transformschema.Node
	for _, reference := range references {
		parent, _ := json.Marshal([]any{"transform-parent-facts/v1", reference.Parent, reference.Key})
		if err := recordTransform(vm, "parent", "ok", 1, [][]byte{forestBytes, transformAtom("id", reference.ID)}, [][]byte{parent}); err != nil {
			return err
		}
		if err := recordTransform(vm, "target", "ok", 2, [][]byte{forestBytes, transformAtom("id", reference.ID)}, [][]byte{transformAtom("id", reference.Target)}); err != nil {
			return err
		}
		targetMatch := reference.Target == definition.ID
		scopeMatch := s.ReferenceScope == "global" || reference.Parent == request.Parent
		guardMatch := s.OldGuard == "any" || reference.Value == request.From
		if err := predicate("reference-target", transformAtom("id", reference.ID), targetMatch); err != nil {
			return err
		}
		if err := predicate("reference-scope", transformAtom("id", reference.ID), scopeMatch); err != nil {
			return err
		}
		if err := predicate("reference-old-guard", transformAtom("id", reference.ID), guardMatch); err != nil {
			return err
		}
		if (s.Targets == "references" || s.Targets == "definition+references") && targetMatch && scopeMatch && guardMatch {
			selectedReferences = append(selectedReferences, reference)
		}
	}
	var edits []transformschema.Edit
	if s.Targets == "definition" || s.Targets == "definition+references" {
		edits = append(edits, transformschema.Edit{Target: definition.ID, Value: request.To})
	}
	for _, reference := range selectedReferences {
		edits = append(edits, transformschema.Edit{Target: reference.ID, Value: request.To})
	}
	slices.SortFunc(edits, func(a, b transformschema.Edit) int { return a.Target - b.Target })
	expansionOK := len(edits) >= 1 && len(edits) <= transformschema.MaxEdits
	if err := predicate("expansion-bound", transformAtom("count", len(edits)), expansionOK); err != nil {
		return err
	}
	if !expansionOK {
		return finish(request.ID, definition.ID, nil, nil)
	}
	editDigests := make([]string, len(edits))
	noOp := false
	for i, edit := range edits {
		editWire, _ := json.Marshal([]any{"set-value/v1", edit.Target, edit.Value})
		editDigests[i] = transformDigest(editWire)
		thisNoOp := byID[edit.Target].Value == edit.Value
		if err := predicate("edit-no-op", editWire, !thisNoOp); err != nil {
			return err
		}
		noOp = noOp || thisNoOp
	}
	if noOp {
		return finish(request.ID, definition.ID, nil, editDigests)
	}
	current := f
	for i, edit := range edits {
		editWire, _ := json.Marshal([]any{"set-value/v1", edit.Target, edit.Value})
		if editDigests[i] != transformDigest(editWire) {
			return errors.New("edit digest changed after no-op validation")
		}
		status, _ := json.Marshal([]any{"transform-edit-status/v1", "valid", editDigests[i]})
		currentBytes, _ := current.CanonicalJSON()
		if err := recordTransform(vm, "edit-validate", "valid", 6, [][]byte{currentBytes, editWire}, [][]byte{status}); err != nil {
			return err
		}
		next, err := (transformschema.Program{Edits: []transformschema.Edit{edit}}).Apply(current)
		if err != nil {
			return err
		}
		nextBytes, _ := next.CanonicalJSON()
		if err := recordTransform(vm, "edit-apply", "applied", 7, [][]byte{currentBytes, editWire}, [][]byte{nextBytes}); err != nil {
			return err
		}
		current = next
	}
	referenceIDs := make([]int, len(selectedReferences))
	for index, reference := range selectedReferences {
		referenceIDs[index] = reference.ID
	}
	return finish(request.ID, definition.ID, referenceIDs, editDigests)
}

func bTSOutputCompare(vm *VM) error {
	rightValue, leftValue := vm.pop(), vm.pop()
	if leftValue.Kind() != VString || rightValue.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	leftBytes, rightBytes := []byte(leftValue.AsString()), []byte(rightValue.AsString())
	left, leftErr := transformschema.ParseForest(leftBytes)
	right, rightErr := transformschema.ParseForest(rightBytes)
	if leftErr != nil || rightErr != nil || len(left.Nodes) != len(right.Nodes) {
		if err := recordTransform(vm, "output-compare", "invalid-input", 9, [][]byte{leftBytes, rightBytes, transformAtom("id", -1)}, nil); err != nil {
			return err
		}
		vm.push(BoolVal(false))
		return nil
	}
	equal := true
	for i := range left.Nodes {
		leftNode, _ := json.Marshal(left.Nodes[i])
		rightNode, _ := json.Marshal(right.Nodes[i])
		nodeEqual := bytes.Equal(leftNode, rightNode)
		outcome := "different"
		if nodeEqual {
			outcome = "equal"
		}
		if err := recordTransform(vm, "output-compare", outcome, 9, [][]byte{leftBytes, rightBytes, transformAtom("id", left.Nodes[i].ID)}, [][]byte{transformAtom("boolean", nodeEqual)}); err != nil {
			return err
		}
		equal = equal && nodeEqual
	}
	vm.push(BoolVal(equal))
	return nil
}

func CompareTransformOutputs(vm *VM, leftBytes, rightBytes []byte) (bool, error) {
	left, leftErr := transformschema.ParseForest(leftBytes)
	right, rightErr := transformschema.ParseForest(rightBytes)
	if leftErr != nil || rightErr != nil || len(left.Nodes) != len(right.Nodes) {
		if err := recordTransform(vm, "output-compare", "invalid-input", 9, [][]byte{leftBytes, rightBytes, transformAtom("id", -1)}, nil); err != nil {
			return false, err
		}
		return false, nil
	}
	equal := true
	for i := range left.Nodes {
		leftNode, _ := json.Marshal(left.Nodes[i])
		rightNode, _ := json.Marshal(right.Nodes[i])
		nodeEqual := bytes.Equal(leftNode, rightNode)
		outcome := "different"
		if nodeEqual {
			outcome = "equal"
		}
		if err := recordTransform(vm, "output-compare", outcome, 9, [][]byte{leftBytes, rightBytes, transformAtom("id", left.Nodes[i].ID)}, [][]byte{transformAtom("boolean", nodeEqual)}); err != nil {
			return false, err
		}
		equal = equal && nodeEqual
	}
	return equal, nil
}

func transformMeterLength(vm *VM) int {
	if vm.CurrentTask == nil || vm.Store == nil {
		return 0
	}
	current := vm.Store.Get(vm.CurrentTask.UnitName)
	if current == nil {
		return 0
	}
	experiment := current
	if name := current.GetString("experiment"); name != "" {
		experiment = vm.Store.Get(name)
	}
	if experiment == nil {
		return 0
	}
	transformMeters.Lock()
	m := transformMeters.items[experiment.GetString("meterToken")]
	transformMeters.Unlock()
	if m == nil {
		return 0
	}
	m.Lock()
	defer m.Unlock()
	return len(m.records)
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
	current := f
	for _, edit := range p.Edits {
		forestBytes, _ := current.CanonicalJSON()
		editWire, _ := json.Marshal([]any{"set-value/v1", edit.Target, edit.Value})
		editDigest := transformDigest(editWire)
		status, _ := json.Marshal([]any{"transform-edit-status/v1", "valid", editDigest})
		if err := recordTransform(vm, "edit-validate", "valid", 6, [][]byte{forestBytes, editWire}, [][]byte{status}); err != nil {
			return err
		}
		next, err := (transformschema.Program{Edits: []transformschema.Edit{edit}}).Apply(current)
		if err != nil {
			return err
		}
		nextBytes, _ := next.CanonicalJSON()
		if err := recordTransform(vm, "edit-apply", "applied", 7, [][]byte{forestBytes, editWire}, [][]byte{nextBytes}); err != nil {
			return err
		}
		current = next
	}
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
