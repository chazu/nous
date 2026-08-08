package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

// CausalTaskScope is the narrow hidden-free task capability used by the v2
// causal builtins. The adapter registry is keyed by the exact top-level VM
// pointer, so copied stores and nested child VMs do not inherit authority.
type CausalTaskScope interface {
	Valid(name, slot string) bool
	Begin(name, slot string) error
	Operation(name, operation string, arguments ...string) (any, error)
	End(name, slot string) error
}

type causalTaskRegistration struct {
	id    uint64
	scope CausalTaskScope
}

var causalTaskScopes = struct {
	sync.RWMutex
	byVM map[*VM]causalTaskRegistration
}{byVM: make(map[*VM]causalTaskRegistration)}

var causalTaskRegistrationSequence atomic.Uint64

// RegisterCausalTaskScope binds scope to one exact VM and returns an
// idempotent revoker. A live binding cannot be replaced.
func RegisterCausalTaskScope(vm *VM, scope CausalTaskScope) (func(), error) {
	if vm == nil || scope == nil {
		return nil, fmt.Errorf("nil causal VM or task scope")
	}
	id := causalTaskRegistrationSequence.Add(1)
	causalTaskScopes.Lock()
	if _, exists := causalTaskScopes.byVM[vm]; exists {
		causalTaskScopes.Unlock()
		return nil, fmt.Errorf("causal task scope already registered")
	}
	causalTaskScopes.byVM[vm] = causalTaskRegistration{id: id, scope: scope}
	causalTaskScopes.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			causalTaskScopes.Lock()
			if current, exists := causalTaskScopes.byVM[vm]; exists && current.id == id {
				delete(causalTaskScopes.byVM, vm)
			}
			causalTaskScopes.Unlock()
		})
	}, nil
}

func causalTaskScopeFor(vm *VM) (CausalTaskScope, error) {
	causalTaskScopes.RLock()
	registration, ok := causalTaskScopes.byVM[vm]
	causalTaskScopes.RUnlock()
	if !ok {
		return nil, fmt.Errorf("child-vm-unauthorized: causal VM has no task capability scope")
	}
	return registration.scope, nil
}

// ProbeCausalChildTaskDenial executes the exact task-valid operation in a real
// nested VM. It is a fixed denial probe; callers cannot supply a program.
func ProbeCausalChildTaskDenial(vm *VM, name, slot string) error {
	if vm == nil || name == "" || slot == "" {
		return fmt.Errorf("invalid causal child denial probe")
	}
	child := vm.childVM()
	_, err := child.Execute(fmt.Sprintf("%q %q causal-v2-task-valid?", name, slot))
	return err
}

func init() {
	registerVocabularyWords("causal", map[string]builtinFn{
		"causal-actions":                          bCausalActions,
		"causal-profile-valid?":                   bCausalProfileValid,
		"causal-task-valid?":                      bCausalTaskValid,
		"causal-partition-json":                   bCausalPartitionJSON,
		"causal-feature-json":                     bCausalFeatureJSON,
		"causal-better?":                          bCausalBetter,
		"causal-equal-score?":                     bCausalEqualScore,
		"causal-artifact-name":                    bCausalArtifactName,
		"causal-response-outcome":                 bCausalResponseOutcome,
		"causal-filter":                           bCausalFilter,
		"causal-action-cost":                      bCausalActionCost,
		"causal-terminal":                         bCausalTerminal,
		"causal-set-digest":                       bCausalSetDigest,
		"causal-transcript-digest":                bCausalTranscriptDigest,
		"causal-code-less?":                       bCausalCodeLess,
		"causal-v2-task-valid?":                   bCausalV2TaskValid,
		"causal-v2-begin-task":                    bCausalV2BeginTask,
		"causal-v2-end-task":                      bCausalV2EndTask,
		"causal-v2-actions":                       bCausalV2Actions,
		"causal-v2-prepare-proposal":              bCausalV2PrepareProposal,
		"causal-v2-materialize-cache":             bCausalV2MaterializeCache,
		"causal-v2-materialize-proposal":          bCausalV2MaterializeProposal,
		"causal-v2-materialize-partition":         bCausalV2MaterializePartition,
		"causal-v2-materialize-score":             bCausalV2MaterializeScore,
		"causal-v2-better?":                       bCausalV2Better,
		"causal-v2-equal-score?":                  bCausalV2EqualScore,
		"causal-v2-materialize-tie":               bCausalV2MaterializeTie,
		"causal-v2-materialize-selection":         bCausalV2MaterializeSelection,
		"causal-v2-materialize-authorization":     bCausalV2MaterializeAuthorization,
		"causal-v2-materialize-awaiting-snapshot": bCausalV2MaterializeAwaitingSnapshot,
		"causal-v2-prepare-update":                bCausalV2PrepareUpdate,
		"causal-v2-eliminated":                    bCausalV2Eliminated,
		"causal-v2-materialize-elimination":       bCausalV2MaterializeElimination,
		"causal-v2-materialize-posterior":         bCausalV2MaterializePosterior,
		"causal-v2-materialize-consumption":       bCausalV2MaterializeConsumption,
		"causal-v2-materialize-transcript":        bCausalV2MaterializeTranscript,
		"causal-v2-materialize-next-snapshot":     bCausalV2MaterializeNextSnapshot,
		"causal-v2-terminal?":                     bCausalV2Terminal,
		"causal-v2-materialize-terminal":          bCausalV2MaterializeTerminal,
		"causal-v2-finish-update":                 bCausalV2FinishUpdate,
		"causal-v2-finalize-zero":                 bCausalV2FinalizeZero,
	})
}

func bCausalV2TaskValid(vm *VM) error {
	slot := vm.pop().AsString()
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	vm.push(BoolVal(scope.Valid(name, slot)))
	return nil
}

func bCausalV2BeginTask(vm *VM) error {
	slot := vm.pop().AsString()
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	if err := scope.Begin(name, slot); err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func bCausalV2EndTask(vm *VM) error {
	slot := vm.pop().AsString()
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	if err := scope.End(name, slot); err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func causalV2NoArg(vm *VM, operation string) error {
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	value, err := scope.Operation(name, operation)
	if err != nil {
		return err
	}
	if boolean, ok := value.(bool); ok {
		vm.push(BoolVal(boolean))
	} else {
		vm.push(BoolVal(true))
	}
	return nil
}

func causalV2OneArg(vm *VM, operation string) error {
	argument := vm.pop().AsString()
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	_, err = scope.Operation(name, operation, argument)
	if err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func bCausalV2PrepareProposal(vm *VM) error { return causalV2NoArg(vm, "prepare-proposal") }

func bCausalV2Actions(vm *VM) error {
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	value, err := scope.Operation(name, "actions")
	if err != nil {
		return err
	}
	items, ok := value.([]string)
	if !ok {
		return fmt.Errorf("causal actions returned non-list")
	}
	vm.push(stringValues(items))
	return nil
}

func bCausalV2MaterializeCache(vm *VM) error     { return causalV2OneArg(vm, "materialize-cache") }
func bCausalV2MaterializeProposal(vm *VM) error  { return causalV2OneArg(vm, "materialize-proposal") }
func bCausalV2MaterializePartition(vm *VM) error { return causalV2OneArg(vm, "materialize-partition") }
func bCausalV2MaterializeScore(vm *VM) error     { return causalV2OneArg(vm, "materialize-score") }
func bCausalV2MaterializeTie(vm *VM) error       { return causalV2OneArg(vm, "materialize-tie") }
func bCausalV2MaterializeSelection(vm *VM) error { return causalV2OneArg(vm, "materialize-selection") }
func bCausalV2MaterializeElimination(vm *VM) error {
	return causalV2OneArg(vm, "materialize-elimination")
}

func causalV2Compare(vm *VM, operation string) error {
	name := vm.pop().AsString()
	right := vm.pop().AsString()
	left := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	value, err := scope.Operation(name, operation, left, right)
	if err != nil {
		return err
	}
	boolean, ok := value.(bool)
	if !ok {
		return fmt.Errorf("%s returned non-bool", operation)
	}
	vm.push(BoolVal(boolean))
	return nil
}
func bCausalV2Better(vm *VM) error     { return causalV2Compare(vm, "better") }
func bCausalV2EqualScore(vm *VM) error { return causalV2Compare(vm, "equal-score") }

func bCausalV2MaterializeAuthorization(vm *VM) error {
	return causalV2NoArg(vm, "materialize-authorization")
}
func bCausalV2MaterializeAwaitingSnapshot(vm *VM) error {
	return causalV2NoArg(vm, "materialize-awaiting-snapshot")
}
func bCausalV2PrepareUpdate(vm *VM) error        { return causalV2NoArg(vm, "prepare-update") }
func bCausalV2MaterializePosterior(vm *VM) error { return causalV2NoArg(vm, "materialize-posterior") }
func bCausalV2MaterializeConsumption(vm *VM) error {
	return causalV2NoArg(vm, "materialize-consumption")
}
func bCausalV2MaterializeTranscript(vm *VM) error { return causalV2NoArg(vm, "materialize-transcript") }
func bCausalV2MaterializeNextSnapshot(vm *VM) error {
	return causalV2NoArg(vm, "materialize-next-snapshot")
}
func bCausalV2Terminal(vm *VM) error            { return causalV2NoArg(vm, "terminal") }
func bCausalV2MaterializeTerminal(vm *VM) error { return causalV2NoArg(vm, "materialize-terminal") }
func bCausalV2FinishUpdate(vm *VM) error        { return causalV2NoArg(vm, "finish-update") }
func bCausalV2FinalizeZero(vm *VM) error        { return causalV2NoArg(vm, "finalize-zero") }
func bCausalV2Eliminated(vm *VM) error {
	name := vm.pop().AsString()
	scope, err := causalTaskScopeFor(vm)
	if err != nil {
		return err
	}
	value, err := scope.Operation(name, "eliminated")
	if err != nil {
		return err
	}
	items, ok := value.([]string)
	if !ok {
		return fmt.Errorf("eliminated returned non-list")
	}
	vm.push(stringValues(items))
	return nil
}
func bCausalCodeLess(vm *VM) error {
	b, a := vm.pop().AsString(), vm.pop().AsString()
	vm.push(BoolVal(a < b))
	return nil
}

func causalExperiment(vm *VM, name string) (*unitLike, error) { return nil, nil }

// unitLike is intentionally unused; causal builtins access the shared store
// only to obtain explicit descriptor fields and never oracle state.
type unitLike struct{}

func stringValues(items []string) Value {
	values := make([]Value, len(items))
	for i, item := range items {
		values[i] = StringVal(item)
	}
	return ListVal(values)
}

func bCausalActions(vm *VM) error {
	codes := make([]string, 0, 6)
	for _, a := range causal.Actions() {
		codes = append(codes, a.Code())
	}
	vm.push(stringValues(codes))
	return nil
}

func causalDescriptor(vm *VM, name string) ([]string, string, bool) {
	u := vm.Store.Get(name)
	if u == nil || !vm.Store.IsA(name, "CausalExperiment") {
		return nil, "", false
	}
	posterior := u.GetStrings("posterior")
	if len(posterior) < 1 || len(posterior) > causal.MaximumPool {
		return nil, "", false
	}
	for _, code := range posterior {
		if _, e := causal.Parse(code); e != nil {
			return nil, "", false
		}
	}
	rule := u.GetString("acquisitionCode")
	if rule != "lexical-fixed" && rule != "uniform-random" && rule != "dynamic-optimal" {
		if _, e := causal.ParseRule(rule); e != nil {
			return nil, "", false
		}
	}
	return posterior, rule, true
}
func bCausalProfileValid(vm *VM) error {
	u := vm.Store.Get(vm.pop().AsString())
	valid := false
	if u != nil {
		_, _, valid = causalDescriptor(vm, u.Name)
		valid = valid && u.GetString("profileVersion") == causal.ProfileVersion && u.GetString("experimentVersion") == causal.ExperimentV1 && u.GetString("profileDigest") != ""
	}
	vm.push(BoolVal(valid))
	return nil
}
func bCausalTaskValid(vm *VM) error {
	kind := vm.pop().AsString()
	slot := vm.pop().AsString()
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	valid := false
	if u != nil {
		_, _, valid = causalDescriptor(vm, name)
		if kind == "propose" {
			valid = valid && slot == u.GetString("proposeTaskSlot") && u.GetString("state") == "ready"
		} else if kind == "update" {
			valid = valid && slot == u.GetString("updateTaskSlot") && u.GetString("state") == "response-present"
		} else {
			valid = false
		}
	}
	vm.push(BoolVal(valid))
	return nil
}

func actionCost(vm *VM, experiment, action string) (int, error) {
	u := vm.Store.Get(experiment)
	a, e := causal.ParseAction(action)
	if e != nil || u == nil {
		return 0, fmt.Errorf("invalid action cost context")
	}
	cost := u.GetInt(fmt.Sprintf("cost%d", a.Variable))
	if cost < 1 || cost > 100 {
		return 0, fmt.Errorf("invalid action cost")
	}
	return cost, nil
}
func actionFeatures(vm *VM, experiment, action string) (causal.Features, error) {
	posterior, _, ok := causalDescriptor(vm, experiment)
	if !ok {
		return causal.Features{}, fmt.Errorf("invalid descriptor")
	}
	cost, e := actionCost(vm, experiment, action)
	if e != nil {
		return causal.Features{}, e
	}
	u := vm.Store.Get(experiment)
	repeated := false
	for _, used := range u.GetStrings("consumedActions") {
		if used == action {
			repeated = true
		}
	}
	return causal.FeaturesFor(posterior, action, cost, repeated)
}
func bCausalPartitionJSON(vm *VM) error {
	experiment := vm.pop().AsString()
	action := vm.pop().AsString()
	posterior, _, ok := causalDescriptor(vm, experiment)
	if !ok {
		return fmt.Errorf("invalid causal descriptor")
	}
	cells, e := causal.Partition(posterior, action)
	if e != nil {
		return e
	}
	b, _ := json.Marshal(cells)
	vm.push(StringVal(string(b)))
	return nil
}
func bCausalFeatureJSON(vm *VM) error {
	experiment := vm.pop().AsString()
	action := vm.pop().AsString()
	f, e := actionFeatures(vm, experiment, action)
	if e != nil {
		return e
	}
	wire := struct {
		E, W int
		H    string
		C, R int
	}{f.ExpectedNumerator, f.Worst, f.EntropyProduct.String(), f.Cost, f.Repeat}
	b, _ := json.Marshal(wire)
	vm.push(StringVal(string(b)))
	return nil
}

func causalCompare(vm *VM, experiment, a, b string) (int, error) {
	posterior, ruleCode, ok := causalDescriptor(vm, experiment)
	if !ok {
		return 0, fmt.Errorf("invalid descriptor")
	}
	if a == b {
		return 0, nil
	}
	u := vm.Store.Get(experiment)
	forced := u.GetString("forcedAction")
	if ruleCode == "uniform-random" || ruleCode == "dynamic-optimal" {
		if a == forced {
			return -1, nil
		}
		if b == forced {
			return 1, nil
		}
		return strings.Compare(a, b), nil
	}
	fa, e := actionFeatures(vm, experiment, a)
	if e != nil {
		return 0, e
	}
	fb, e := actionFeatures(vm, experiment, b)
	if e != nil {
		return 0, e
	}
	if ruleCode == "lexical-fixed" {
		if fa.Repeat != fb.Repeat {
			if fa.Repeat < fb.Repeat {
				return -1, nil
			}
			return 1, nil
		}
		return strings.Compare(a, b), nil
	}
	rule, e := causal.ParseRule(ruleCode)
	if e != nil {
		return 0, e
	}
	cmp := causal.Compare(rule, len(posterior), fa, fb)
	if cmp == 0 {
		return strings.Compare(a, b), nil
	}
	return cmp, nil
}
func causalScoreEqual(vm *VM, experiment, a, b string) (bool, error) {
	posterior, ruleCode, ok := causalDescriptor(vm, experiment)
	if !ok {
		return false, fmt.Errorf("invalid descriptor")
	}
	if ruleCode == "uniform-random" || ruleCode == "dynamic-optimal" {
		return a == b, nil
	}
	fa, e := actionFeatures(vm, experiment, a)
	if e != nil {
		return false, e
	}
	fb, e := actionFeatures(vm, experiment, b)
	if e != nil {
		return false, e
	}
	if ruleCode == "lexical-fixed" {
		return fa.Repeat == fb.Repeat, nil
	}
	rule, e := causal.ParseRule(ruleCode)
	if e != nil {
		return false, e
	}
	return causal.Compare(rule, len(posterior), fa, fb) == 0, nil
}
func bCausalBetter(vm *VM) error {
	experiment := vm.pop().AsString()
	best := vm.pop().AsString()
	candidate := vm.pop().AsString()
	cmp, e := causalCompare(vm, experiment, candidate, best)
	if e != nil {
		return e
	}
	vm.push(BoolVal(cmp < 0))
	return nil
}
func bCausalEqualScore(vm *VM) error {
	experiment := vm.pop().AsString()
	best := vm.pop().AsString()
	candidate := vm.pop().AsString()
	equal, e := causalScoreEqual(vm, experiment, candidate, best)
	if e != nil {
		return e
	}
	vm.push(BoolVal(equal))
	return nil
}
func bCausalArtifactName(vm *VM) error {
	semantic := vm.pop().AsString()
	kind := vm.pop().AsString()
	experiment := vm.pop().AsString()
	step := 0
	if descriptor := vm.Store.Get(experiment); descriptor != nil {
		step = descriptor.GetInt("step")
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s", experiment, step, kind, semantic)))
	vm.push(StringVal("Causal." + kind + "." + hex.EncodeToString(sum[:8])))
	return nil
}
func bCausalResponseOutcome(vm *VM) error {
	name := vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil || !vm.Store.IsA(name, "CausalTeacherResult") {
		return fmt.Errorf("missing causal teacher result")
	}
	outcome := u.GetString("outcome")
	if len(outcome) != 3 {
		return fmt.Errorf("invalid teacher outcome")
	}
	vm.push(StringVal(outcome))
	return nil
}
func bCausalFilter(vm *VM) error {
	outcome := vm.pop().AsString()
	action := vm.pop().AsString()
	experiment := vm.pop().AsString()
	posterior, _, ok := causalDescriptor(vm, experiment)
	if !ok {
		return fmt.Errorf("invalid descriptor")
	}
	next, e := causal.Filter(posterior, action, outcome)
	if e != nil {
		return e
	}
	vm.push(stringValues(next))
	return nil
}
func bCausalActionCost(vm *VM) error {
	experiment := vm.pop().AsString()
	action := vm.pop().AsString()
	cost, e := actionCost(vm, experiment, action)
	if e != nil {
		return e
	}
	vm.push(IntVal(cost))
	return nil
}
func bCausalTerminal(vm *VM) error {
	experiment := vm.pop().AsString()
	posteriorValue := vm.pop()
	posterior, ok := strictStringList(posteriorValue)
	if !ok {
		return fmt.Errorf("invalid posterior")
	}
	u := vm.Store.Get(experiment)
	terminal := ""
	if len(posterior) == 1 {
		terminal = "identified"
	} else if causal.CompleteClass(u.GetStrings("initialPosterior"), posterior) {
		terminal = "equivalence"
	} else if u.GetInt("actionCount") >= causal.MaximumActions {
		terminal = "budget-exhausted"
	}
	vm.push(StringVal(terminal))
	return nil
}
func bCausalSetDigest(vm *VM) error {
	items, ok := strictStringList(vm.pop())
	if !ok {
		return fmt.Errorf("invalid set")
	}
	sort.Strings(items)
	digest, e := causal.Digest("causal-hypothesis-set/v1", items)
	if e != nil {
		return e
	}
	vm.push(StringVal(digest))
	return nil
}
func bCausalTranscriptDigest(vm *VM) error {
	experiment := vm.pop().AsString()
	u := vm.Store.Get(experiment)
	if u == nil {
		return fmt.Errorf("missing experiment")
	}
	material := struct {
		Previous, Action, Outcome string
		Step, Cost, Count         int
		Posterior                 []string
	}{u.GetString("transcriptDigest"), u.GetString("selectedAction"), u.GetString("lastOutcome"), u.GetInt("step"), u.GetInt("totalCost"), u.GetInt("actionCount"), u.GetStrings("posterior")}
	digest, e := causal.Digest("causal-transcript-entry/v1", material)
	if e != nil {
		return e
	}
	vm.push(StringVal(digest))
	return nil
}
