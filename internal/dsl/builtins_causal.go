package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

func init() {
	registerVocabularyWords("causal", map[string]builtinFn{
		"causal-actions":           bCausalActions,
		"causal-profile-valid?":    bCausalProfileValid,
		"causal-task-valid?":       bCausalTaskValid,
		"causal-partition-json":    bCausalPartitionJSON,
		"causal-feature-json":      bCausalFeatureJSON,
		"causal-better?":           bCausalBetter,
		"causal-equal-score?":      bCausalEqualScore,
		"causal-artifact-name":     bCausalArtifactName,
		"causal-response-outcome":  bCausalResponseOutcome,
		"causal-filter":            bCausalFilter,
		"causal-action-cost":       bCausalActionCost,
		"causal-terminal":          bCausalTerminal,
		"causal-set-digest":        bCausalSetDigest,
		"causal-transcript-digest": bCausalTranscriptDigest,
		"causal-code-less?":        bCausalCodeLess,
	})
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
