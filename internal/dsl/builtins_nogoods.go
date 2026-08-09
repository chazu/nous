package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
		"ng-edge-has?":             bNGEdgeHas,
		"ng-refine-mask":           bNGRefineMask,
		"ng-guard-matches?":        bNGGuardMatches,
		"ng-mask-matches?":         bNGMaskMatches,
		"ng-completion-conflicts?": bNGCompletionConflicts,
		"ng-certificate-valid?":    bNGCertificateValid,
		"ng-artifact-name":         bNGArtifactName,
	})
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
