package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/chazu/nous/internal/unit"
	nogoods "github.com/chazu/nous/internal/vocab/nogoods"
)

// The nogood extension deliberately exposes only bounded, one-object
// operations. Candidate population, role enumeration, evidence aggregation,
// and promotion remain visible in the CUE heuristic program.
func init() {
	registerVocabularyWords("nogoods", map[string]builtinFn{
		"ng-problem-valid?":        bNGProblemValid,
		"ng-semantic-key":          bNGSemanticKey,
		"ng-domain-has?":           bNGDomainHas,
		"ng-domain-values":         bNGDomainValues,
		"ng-edge-has?":             bNGEdgeHas,
		"ng-refine-mask":           bNGRefineMask,
		"ng-guard-matches?":        bNGGuardMatches,
		"ng-mask-matches?":         bNGMaskMatches,
		"ng-completion-conflicts?": bNGCompletionConflicts,
		"ng-certificate-valid?":    bNGCertificateValid,
		"ng-artifact-name":         bNGArtifactName,
		"ng-digest-list":           bNGDigestList,
		"ng-digest-record":         bNGDigestRecord,
		"ng-unit-set-digest":       bNGUnitSetDigest,
		"ng-meter":                 bNGMeter,
	})
}

// NogoodMeterRecord is emitted at the semantic operation site by the CUE
// heuristic. The verifier owns the opaque capability and only exposes an
// append operation to the vocabulary.
type NogoodMeterRecord struct {
	Category  uint8
	Operation string
	Subject   string
	Object    string
	Outcome   string
}

type nogoodMeter struct {
	mu      sync.Mutex
	records []NogoodMeterRecord
}

var nogoodMeters = struct {
	sync.Mutex
	items map[string]*nogoodMeter
}{items: make(map[string]*nogoodMeter)}

func RegisterNogoodMeter(token string) error {
	if token == "" {
		return errors.New("empty nogood meter token")
	}
	nogoodMeters.Lock()
	defer nogoodMeters.Unlock()
	if _, exists := nogoodMeters.items[token]; exists {
		return errors.New("duplicate nogood meter token")
	}
	nogoodMeters.items[token] = &nogoodMeter{}
	return nil
}

func UnregisterNogoodMeter(token string) {
	nogoodMeters.Lock()
	delete(nogoodMeters.items, token)
	nogoodMeters.Unlock()
}

func NogoodMeterSnapshot(token string) ([]NogoodMeterRecord, error) {
	nogoodMeters.Lock()
	meter := nogoodMeters.items[token]
	nogoodMeters.Unlock()
	if meter == nil {
		return nil, errors.New("unknown nogood meter capability")
	}
	meter.mu.Lock()
	defer meter.mu.Unlock()
	return append([]NogoodMeterRecord(nil), meter.records...), nil
}

func ngCharge(token, operation, subject, object, outcome string, category int) error {
	if category < 1 || category > 12 || operation == "" || subject == "" || object == "" || outcome == "" {
		return errors.New("invalid nogood meter event")
	}
	nogoodMeters.Lock()
	meter := nogoodMeters.items[token]
	nogoodMeters.Unlock()
	if meter == nil {
		return errors.New("unknown nogood meter capability")
	}
	meter.mu.Lock()
	defer meter.mu.Unlock()
	meter.records = append(meter.records, NogoodMeterRecord{Category: uint8(category), Operation: operation, Subject: subject, Object: object, Outcome: outcome})
	return nil
}

// ChargeNogoodMeter records adapter-side semantic operations against the same
// verifier-owned capability used by the CUE heuristic.
func ChargeNogoodMeter(token, operation, subject, object, outcome string, category int) error {
	return ngCharge(token, operation, subject, object, outcome, category)
}

func bNGMeter(vm *VM) error {
	outcome, outcomeOK := ngString(vm.pop())
	objectValue := vm.pop()
	object, objectOK := ngString(objectValue)
	subject, subjectOK := ngString(vm.pop())
	operation, operationOK := ngString(vm.pop())
	category, categoryOK := ngInt(vm.pop())
	token, tokenOK := ngString(vm.pop())
	if !outcomeOK || !objectOK || !subjectOK || !operationOK || !categoryOK || !tokenOK {
		return fmt.Errorf("invalid nogood meter operands token=%v category=%v operation=%q/%v subject=%q/%v object-kind=%v outcome=%v", tokenOK, categoryOK, operation, operationOK, subject, subjectOK, objectValue.Kind(), outcomeOK)
	}
	if err := ngCharge(token, operation, subject, object, outcome, category); err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func bNGDigestRecord(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VList || len(value.AsList()) > 64 {
		vm.push(Nil())
		return nil
	}
	encoded, err := json.Marshal(ngSerializable(value))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	digest := sha256.Sum256(encoded)
	vm.push(StringVal(hex.EncodeToString(digest[:])))
	return nil
}

func ngSerializable(value Value) any {
	switch value.Kind() {
	case VNil:
		return nil
	case VBool:
		return value.AsBool()
	case VInt:
		return value.AsInt()
	case VFloat:
		return value.AsFloat()
	case VString:
		return value.AsString()
	case VList:
		items := value.AsList()
		out := make([]any, len(items))
		for index, item := range items {
			out[index] = ngSerializable(item)
		}
		return out
	default:
		return nil
	}
}

func bNGUnitSetDigest(vm *VM) error {
	names, ok := strictStringList(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	digest, err := UnitSetDigest(vm.Store, names)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(digest))
	return nil
}

// UnitSetDigest commits the authoritative contents of an already materialized
// ordered reference set. The two self-referential digest slots are excluded.
func UnitSetDigest(store *unit.Store, names []string) (string, error) {
	type record struct {
		Name  string         `json:"name"`
		Slots map[string]any `json:"slots"`
	}
	records := make([]record, 0, len(names))
	for _, name := range names {
		u := store.Get(name)
		if u == nil {
			return "", fmt.Errorf("missing referenced unit %q", name)
		}
		slots := make(map[string]any, len(u.Slots))
		for key, value := range u.Slots {
			if key != "referencedUnitSetDigest" && key != "barrierDigest" && key != "dispositionUnit" {
				slots[key] = value
			}
		}
		records = append(records, record{Name: name, Slots: slots})
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func bNGDigestList(vm *VM) error {
	values, ok := strictStringList(vm.pop())
	if !ok || len(values) > 64 {
		vm.push(Nil())
		return nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	digest := sha256.Sum256(encoded)
	vm.push(StringVal(hex.EncodeToString(digest[:])))
	return nil
}

func ngString(value Value) (string, bool) {
	if value.Kind() != VString {
		return "", false
	}
	return value.AsString(), true
}

func ngInt(value Value) (int, bool) {
	if value.Kind() != VInt {
		return 0, false
	}
	return value.AsInt(), true
}

func ngProblem(value Value) (nogoods.Problem, bool) {
	text, ok := ngString(value)
	if !ok {
		return nogoods.Problem{}, false
	}
	problem, err := nogoods.ParseProblem([]byte(text))
	return problem, err == nil
}

func bNGProblemValid(vm *VM) error {
	_, ok := ngProblem(vm.pop())
	vm.push(BoolVal(ok))
	return nil
}

func bNGSemanticKey(vm *VM) error {
	problem, ok := ngProblem(vm.pop())
	if !ok {
		vm.push(Nil())
		return nil
	}
	key, err := problem.SemanticKey()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(key))
	return nil
}

func bNGDomainHas(vm *VM) error {
	color, colorOK := ngInt(vm.pop())
	variable, variableOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	vm.push(BoolVal(problemOK && variableOK && colorOK && problem.DomainContains(variable, color)))
	return nil
}

func bNGDomainValues(vm *VM) error {
	variable, variableOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	if !problemOK || !variableOK || variable < 0 || variable >= len(problem.Variables) {
		vm.push(Nil())
		return nil
	}
	values := make([]Value, len(problem.Variables[variable].Domain))
	for index, color := range problem.Variables[variable].Domain {
		values[index] = IntVal(color)
	}
	vm.push(ListVal(values))
	return nil
}

func bNGEdgeHas(vm *VM) error {
	right, rightOK := ngInt(vm.pop())
	left, leftOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	vm.push(BoolVal(problemOK && leftOK && rightOK && problem.EdgePresent(left, right)))
	return nil
}

func bNGRefineMask(vm *VM) error {
	bit, bitOK := ngInt(vm.pop())
	mask, maskOK := ngInt(vm.pop())
	if !bitOK || !maskOK || mask < 0 {
		vm.push(Nil())
		return nil
	}
	refined, err := nogoods.RefineMask(nogoods.Mask(mask), bit)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(IntVal(int(refined)))
	return nil
}

func popNGBinding(vm *VM) (nogoods.Binding, bool) {
	only, onlyOK := ngInt(vm.pop())
	escape, escapeOK := ngInt(vm.pop())
	blocked, blockedOK := ngInt(vm.pop())
	y, yOK := ngInt(vm.pop())
	x, xOK := ngInt(vm.pop())
	anchor, anchorOK := ngInt(vm.pop())
	return nogoods.Binding{Anchor: anchor, X: x, Y: y, Blocked: blocked, Escape: escape, Only: only},
		anchorOK && xOK && yOK && blockedOK && escapeOK && onlyOK
}

func bNGGuardMatches(vm *VM) error {
	binding, bindingOK := popNGBinding(vm)
	decisionColor, colorOK := ngInt(vm.pop())
	decisionVariable, variableOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	decision := nogoods.Literal{Variable: decisionVariable, Color: decisionColor}
	vm.push(BoolVal(problemOK && variableOK && colorOK && bindingOK && nogoods.GuardMatches(problem, decision, binding)))
	return nil
}

func bNGMaskMatches(vm *VM) error {
	binding, bindingOK := popNGBinding(vm)
	mask, maskOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	vm.push(BoolVal(problemOK && maskOK && mask >= 0 && bindingOK && nogoods.MaskMatches(problem, nogoods.Mask(mask), binding)))
	return nil
}

func bNGCompletionConflicts(vm *VM) error {
	yColor, yColorOK := ngInt(vm.pop())
	xColor, xColorOK := ngInt(vm.pop())
	binding, bindingOK := popNGBinding(vm)
	mask, maskOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	if !problemOK || !maskOK || mask < 0 || !bindingOK || !xColorOK || !yColorOK {
		vm.push(Nil())
		return nil
	}
	conflict, err := nogoods.EvaluateCompletion(problem, nogoods.Mask(mask), binding, nogoods.Completion{XColor: xColor, YColor: yColor})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(BoolVal(conflict))
	return nil
}

func bNGCertificateValid(vm *VM) error {
	conflictValue := vm.pop()
	yColor, yColorOK := ngInt(vm.pop())
	xColor, xColorOK := ngInt(vm.pop())
	binding, bindingOK := popNGBinding(vm)
	decisionColor, colorOK := ngInt(vm.pop())
	decisionVariable, variableOK := ngInt(vm.pop())
	mask, maskOK := ngInt(vm.pop())
	problem, problemOK := ngProblem(vm.pop())
	if !problemOK || !maskOK || mask < 0 || !variableOK || !colorOK || !bindingOK || !xColorOK || !yColorOK || conflictValue.Kind() != VBool {
		vm.push(BoolVal(false))
		return nil
	}
	record := nogoods.CertificateRecord{
		SchemaVersion: nogoods.SchemaVersion,
		Mask:          nogoods.Mask(mask),
		Binding:       binding,
		Decision:      nogoods.Literal{Variable: decisionVariable, Color: decisionColor},
		Completion:    nogoods.Completion{XColor: xColor, YColor: yColor},
		Conflict:      conflictValue.AsBool(),
	}
	vm.push(BoolVal(nogoods.ValidateCertificateRecord(problem, record) == nil))
	return nil
}

// ng-artifact-name canonicalizes one already-materialized semantic identity;
// it does not search for, select, or populate an artifact.
func bNGArtifactName(vm *VM) error {
	semantic, semanticOK := ngString(vm.pop())
	kind, kindOK := ngString(vm.pop())
	if !semanticOK || !kindOK || semantic == "" || kind == "" || len(semantic) > 4096 || len(kind) > 128 {
		vm.push(Nil())
		return nil
	}
	digest := sha256.Sum256([]byte(nogoods.SchemaVersion + "\x00" + kind + "\x00" + semantic))
	vm.push(StringVal(fmt.Sprintf("NG.%s.%s", kind, hex.EncodeToString(digest[:12]))))
	return nil
}
