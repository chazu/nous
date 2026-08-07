package dsl

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	programvocab "github.com/chazu/nous/internal/vocab/programsynth"
)

const (
	synthMaxNameBytes       = 512
	synthMaxDefnBytes       = 4096
	synthMaxCandidates      = 600
	synthMaxExamples        = 16
	synthMaxPrimitives      = 8
	synthMaxProbes          = 16
	synthMaxComparisons     = 4096
	synthDescriptorCategory = "BoundedProgramSynthesisExperiment"
)

type synthDescriptor struct {
	name, experimentKey, primitiveCategory, candidateCategory    string
	exampleCategory, valueCategory, resultCategory               string
	observationCategory, evidenceCategory, schemaCategory        string
	selectionEvidenceCategory                                    string
	inputSlot, expectedSlot, resultValueSlot                     string
	inputValidator, outputValidator, comparator                  string
	semanticSlot, method, creditContext                          string
	synthesisTaskSlot, evaluationTaskSlot, finalizationTaskSlot  string
	simplificationTaskSlot, probeCategory, probeInputSlot        string
	simplificationPairCategory, simplificationExecObsCategory    string
	simplificationExecResultCategory, simplificationCmpCategory  string
	simplificationEvidenceCategory, simplificationSchemaCategory string
	probeSetVersion                                              string
	maxLength, minCorpus, primitiveCap, exampleCap               int
	probeCap, candidateCap, comparisonCap                        int
	synthesisPriority, evaluationPriority, finalizationPriority  int
	simplificationPriority, simplificationLength                 int
}

func init() {
	registerVocabularyWords("programsynth", map[string]builtinFn{
		"synth-experiment-valid?":          bSynthExperimentValid,
		"synth-candidate-count":            bSynthCandidateCount,
		"synth-sequence-valid?":            bSynthSequenceValid,
		"synth-program-name":               bSynthProgramName,
		"synth-program-defn":               bSynthProgramDefn,
		"synth-semantic-sequence":          bSynthSemanticSequence,
		"synth-decision-key":               bSynthDecisionKey,
		"synth-credit-roles":               bSynthCreditRoles,
		"synth-artifact-name":              bSynthArtifactName,
		"synth-find-execution-observation": bSynthFindExecutionObservation,
		"synth-ready-to-finalize?":         bSynthReadyToFinalize,
	})
}

func synthString(u interface{ GetString(string) string }, slot string) string {
	return u.GetString(slot)
}

func readSynthDescriptor(vm *VM, name string) (synthDescriptor, bool) {
	u := vm.Store.Get(name)
	if u == nil || name == synthDescriptorCategory || !vm.Store.IsA(name, synthDescriptorCategory) {
		return synthDescriptor{}, false
	}
	d := synthDescriptor{
		name:                             name,
		experimentKey:                    synthString(u, "experimentKey"),
		primitiveCategory:                synthString(u, "primitiveCategory"),
		candidateCategory:                synthString(u, "candidateCategory"),
		exampleCategory:                  synthString(u, "exampleCategory"),
		valueCategory:                    synthString(u, "valueCategory"),
		resultCategory:                   synthString(u, "resultCategory"),
		observationCategory:              synthString(u, "observationCategory"),
		evidenceCategory:                 synthString(u, "evidenceCategory"),
		selectionEvidenceCategory:        synthString(u, "selectionEvidenceCategory"),
		schemaCategory:                   synthString(u, "promotedSchemaCategory"),
		inputSlot:                        synthString(u, "inputSlot"),
		expectedSlot:                     synthString(u, "expectedSlot"),
		resultValueSlot:                  synthString(u, "resultValueSlot"),
		inputValidator:                   synthString(u, "inputValidator"),
		outputValidator:                  synthString(u, "outputValidator"),
		comparator:                       synthString(u, "comparator"),
		semanticSlot:                     synthString(u, "primitiveSemanticSlot"),
		method:                           synthString(u, "synthesisMethod"),
		creditContext:                    synthString(u, "creditContext"),
		synthesisTaskSlot:                synthString(u, "synthesisTaskSlot"),
		evaluationTaskSlot:               synthString(u, "evaluationTaskSlot"),
		finalizationTaskSlot:             synthString(u, "finalizationTaskSlot"),
		simplificationTaskSlot:           synthString(u, "simplificationTaskSlot"),
		probeCategory:                    synthString(u, "probeCategory"),
		probeInputSlot:                   synthString(u, "probeInputSlot"),
		simplificationPairCategory:       synthString(u, "simplificationPairCategory"),
		simplificationExecObsCategory:    synthString(u, "simplificationExecutionObservationCategory"),
		simplificationExecResultCategory: synthString(u, "simplificationExecutionResultCategory"),
		simplificationCmpCategory:        synthString(u, "simplificationComparisonObservationCategory"),
		simplificationEvidenceCategory:   synthString(u, "simplificationEvidenceCategory"),
		simplificationSchemaCategory:     synthString(u, "simplificationSchemaCategory"),
		probeSetVersion:                  synthString(u, "probeSetVersion"),
		maxLength:                        u.GetInt("maxLength"), minCorpus: u.GetInt("minCorpus"),
		primitiveCap: u.GetInt("primitiveCap"), exampleCap: u.GetInt("exampleCap"),
		probeCap: u.GetInt("probeCap"), candidateCap: u.GetInt("candidateCap"),
		comparisonCap:          u.GetInt("simplificationComparisonCap"),
		synthesisPriority:      u.GetInt("synthesisPriority"),
		evaluationPriority:     u.GetInt("evaluationPriority"),
		finalizationPriority:   u.GetInt("finalizationPriority"),
		simplificationPriority: u.GetInt("simplificationPriority"),
		simplificationLength:   u.GetInt("simplificationProgramLength"),
	}
	return d, true
}

func nonemptyBounded(values ...string) bool {
	for _, value := range values {
		if value == "" || len(value) > synthMaxNameBytes {
			return false
		}
	}
	return true
}

func defnOnly(vm *VM, name string) bool {
	u := vm.Store.Get(name)
	if u == nil || u.GetString("defn") == "" || len(u.GetString("defn")) > synthMaxDefnBytes {
		return false
	}
	for _, slot := range []string{"fastAlg", "alg", "fastDefn", "unitizedDefn", "iterativeDefn", "recursiveDefn", "compiledDefn"} {
		if u.GetString(slot) != "" {
			return false
		}
	}
	return true
}

func synthCategoryMembers(vm *VM, category string) []string {
	var names []string
	for _, name := range vm.Store.Examples(category) {
		if name != category {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func synthExecuteDefn(vm *VM, opName string, args ...Value) (Value, bool) {
	u := vm.Store.Get(opName)
	if u == nil || !defnOnly(vm, opName) {
		return Nil(), false
	}
	sub := vm.childVM()
	for key, value := range vm.env {
		sub.env[key] = value
	}
	sub.stack = append(sub.stack, args...)
	result, err := subExecute(sub, u.GetString("defn"))
	return result, err == nil
}

func exactUnaryEndomorphism(vm *VM, name, valueCategory string) bool {
	u := vm.Store.Get(name)
	return u != nil && defnOnly(vm, name) && u.GetInt("arity") == 1 &&
		sameStrings(u.GetStrings("domain"), []string{valueCategory}) &&
		sameStrings(u.GetStrings("range"), []string{valueCategory}) &&
		vm.Store.IsA(name, "UnaryOp")
}

func exactUnaryValidator(vm *VM, name, valueCategory string) bool {
	u := vm.Store.Get(name)
	return u != nil && defnOnly(vm, name) && u.GetInt("arity") == 1 &&
		sameStrings(u.GetStrings("domain"), []string{valueCategory}) &&
		(vm.Store.IsA(name, "UnaryOp") || vm.Store.IsA(name, "UnaryPred"))
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validSynthDescriptor(vm *VM, d synthDescriptor, validateCorpus bool) bool {
	stringsRequired := []string{
		d.experimentKey, d.primitiveCategory, d.candidateCategory, d.exampleCategory,
		d.valueCategory, d.resultCategory, d.observationCategory, d.evidenceCategory,
		d.selectionEvidenceCategory, d.schemaCategory, d.inputSlot, d.expectedSlot, d.resultValueSlot,
		d.inputValidator, d.outputValidator, d.comparator, d.semanticSlot,
		d.method, d.creditContext, d.synthesisTaskSlot, d.evaluationTaskSlot,
		d.finalizationTaskSlot, d.simplificationTaskSlot, d.probeCategory,
		d.probeInputSlot, d.simplificationPairCategory, d.simplificationExecObsCategory,
		d.simplificationExecResultCategory, d.simplificationCmpCategory,
		d.simplificationEvidenceCategory, d.simplificationSchemaCategory, d.probeSetVersion,
	}
	if !nonemptyBounded(stringsRequired...) || len(d.method) > programvocab.MaxMethodBytes || len(d.creditContext) > 256 {
		return false
	}
	categories := []string{
		d.primitiveCategory, d.candidateCategory, d.exampleCategory, d.resultCategory,
		d.observationCategory, d.evidenceCategory, d.selectionEvidenceCategory, d.schemaCategory, d.probeCategory,
		d.simplificationPairCategory, d.simplificationExecObsCategory,
		d.simplificationExecResultCategory, d.simplificationCmpCategory,
		d.simplificationEvidenceCategory, d.simplificationSchemaCategory,
	}
	seenCategories := map[string]bool{}
	for _, category := range append(categories, d.valueCategory) {
		if !vm.Store.Has(category) || seenCategories[category] {
			return false
		}
		seenCategories[category] = true
	}
	if d.maxLength < 1 || d.maxLength > programvocab.MaxProgramLength || d.minCorpus < 1 ||
		d.primitiveCap < 1 || d.primitiveCap > synthMaxPrimitives || d.exampleCap < d.minCorpus || d.exampleCap > synthMaxExamples ||
		d.probeCap < 1 || d.probeCap > synthMaxProbes || d.candidateCap < 1 || d.candidateCap > synthMaxCandidates ||
		d.comparisonCap < 1 || d.comparisonCap > synthMaxComparisons || d.simplificationLength != 2 ||
		!(d.synthesisPriority > d.evaluationPriority && d.evaluationPriority > d.finalizationPriority && d.finalizationPriority > d.simplificationPriority) {
		return false
	}
	taskSlots := []string{d.synthesisTaskSlot, d.evaluationTaskSlot, d.finalizationTaskSlot, d.simplificationTaskSlot}
	sort.Strings(taskSlots)
	for index := 1; index < len(taskSlots); index++ {
		if taskSlots[index] == taskSlots[index-1] {
			return false
		}
	}
	if !exactUnaryValidator(vm, d.inputValidator, d.valueCategory) || !exactUnaryValidator(vm, d.outputValidator, d.valueCategory) ||
		!defnOnly(vm, d.comparator) || vm.Store.Get(d.comparator).GetInt("arity") != 2 ||
		!sameStrings(vm.Store.Get(d.comparator).GetStrings("domain"), []string{d.valueCategory, d.valueCategory}) ||
		!(vm.Store.IsA(d.comparator, "BinaryOp") || vm.Store.IsA(d.comparator, "BinaryPred")) {
		return false
	}
	primitives := synthCategoryMembers(vm, d.primitiveCategory)
	if len(primitives) == 0 || len(primitives) > d.primitiveCap {
		return false
	}
	semanticSeen := map[string]bool{}
	for _, name := range primitives {
		if len(name) > synthMaxNameBytes || !exactUnaryEndomorphism(vm, name, d.valueCategory) || vm.Store.IsA(name, d.candidateCategory) {
			return false
		}
		semantic := vm.Store.Get(name).GetString(d.semanticSlot)
		if !programvocab.ValidSemanticKey(semantic) || semanticSeen[semantic] {
			return false
		}
		semanticSeen[semantic] = true
	}
	candidateCount := 0
	power := 1
	for length := 1; length <= d.maxLength; length++ {
		power *= len(primitives)
		candidateCount += power
	}
	if candidateCount > d.candidateCap {
		return false
	}
	examples := synthCategoryMembers(vm, d.exampleCategory)
	probes := synthCategoryMembers(vm, d.probeCategory)
	if len(examples) < d.minCorpus || len(examples) > d.exampleCap || len(probes) == 0 || len(probes) > d.probeCap {
		return false
	}
	if len(primitives)*len(primitives)*len(primitives)*len(probes) > d.comparisonCap {
		return false
	}
	if !validateCorpus {
		return true
	}
	for _, name := range examples {
		u := vm.Store.Get(name)
		input, inputOK := synthExecuteDefn(vm, d.inputValidator, anyToValue(u.Get(d.inputSlot)))
		expected, expectedOK := synthExecuteDefn(vm, d.outputValidator, anyToValue(u.Get(d.expectedSlot)))
		if !inputOK || !expectedOK || input.Kind() != VBool || expected.Kind() != VBool || !input.AsBool() || !expected.AsBool() {
			return false
		}
	}
	for _, name := range probes {
		result, ok := synthExecuteDefn(vm, d.inputValidator, anyToValue(vm.Store.Get(name).Get(d.probeInputSlot)))
		if !ok || result.Kind() != VBool || !result.AsBool() {
			return false
		}
	}
	return true
}

func bSynthExperimentValid(vm *VM) error {
	name, ok := strictString(vm.pop())
	d, descriptorOK := readSynthDescriptor(vm, name)
	vm.push(BoolVal(ok && descriptorOK && validSynthDescriptor(vm, d, true)))
	return nil
}

func synthCandidateCount(vm *VM, d synthDescriptor) int {
	n := len(synthCategoryMembers(vm, d.primitiveCategory))
	total, power := 0, 1
	for length := 1; length <= d.maxLength; length++ {
		power *= n
		total += power
	}
	return total
}

func bSynthCandidateCount(vm *VM) error {
	name, ok := strictString(vm.pop())
	d, descriptorOK := readSynthDescriptor(vm, name)
	if !ok || !descriptorOK || !validSynthDescriptor(vm, d, true) {
		vm.push(Nil())
		return nil
	}
	vm.push(IntVal(synthCandidateCount(vm, d)))
	return nil
}

func synthComponents(vm *VM, experiment string, value Value) (synthDescriptor, []string, []string, []string, bool) {
	d, ok := readSynthDescriptor(vm, experiment)
	if !ok || !validSynthDescriptor(vm, d, false) {
		return d, nil, nil, nil, false
	}
	names, ok := strictStringList(value)
	if !ok || len(names) == 0 || len(names) > d.maxLength {
		return d, nil, nil, nil, false
	}
	semantic := make([]string, len(names))
	definitions := make([]string, len(names))
	for index, name := range names {
		if len(name) > synthMaxNameBytes || name == d.primitiveCategory || !vm.Store.IsA(name, d.primitiveCategory) || !exactUnaryEndomorphism(vm, name, d.valueCategory) {
			return d, nil, nil, nil, false
		}
		semantic[index] = vm.Store.Get(name).GetString(d.semanticSlot)
		definitions[index] = vm.Store.Get(name).GetString("defn")
	}
	if !programvocab.ValidSequence(semantic) {
		return d, nil, nil, nil, false
	}
	return d, names, semantic, definitions, true
}

func bSynthSequenceValid(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	_, _, _, _, valid := synthComponents(vm, experiment, components)
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(BoolVal(valid))
	}
	return nil
}

func synthEncode(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }

func freshSynthName(vm *VM, base string) string {
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

func bSynthProgramName(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	d, names, _, _, valid := synthComponents(vm, experiment, components)
	if !ok || !valid {
		vm.push(Nil())
		return nil
	}
	parts := []string{"SynthProgram", synthEncode(d.experimentKey), fmt.Sprintf("%d", len(names))}
	for _, name := range names {
		parts = append(parts, synthEncode(name))
	}
	vm.push(StringVal(freshSynthName(vm, strings.Join(parts, "."))))
	return nil
}

func bSynthProgramDefn(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	_, _, _, definitions, valid := synthComponents(vm, experiment, components)
	if !ok || !valid {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(strings.Join(definitions, " ")))
	return nil
}

func bSynthSemanticSequence(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	_, _, semantic, _, valid := synthComponents(vm, experiment, components)
	if !ok || !valid {
		vm.push(Nil())
		return nil
	}
	vm.push(stringListValue(semantic))
	return nil
}

func bSynthDecisionKey(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	d, _, semantic, _, valid := synthComponents(vm, experiment, components)
	if !ok || !valid {
		vm.push(Nil())
		return nil
	}
	decision, err := programvocab.DecisionKey(d.method, semantic)
	if err != nil {
		vm.push(Nil())
	} else {
		vm.push(StringVal(decision))
	}
	return nil
}

func bSynthCreditRoles(vm *VM) error {
	components := vm.pop()
	experiment, ok := strictString(vm.pop())
	_, names, _, _, valid := synthComponents(vm, experiment, components)
	if !ok || !valid {
		vm.push(Nil())
		return nil
	}
	roles := []string{"synthesis"}
	for index := range names {
		roles = append(roles, fmt.Sprintf("step-%d", index+1))
	}
	vm.push(stringListValue(roles))
	return nil
}

func bSynthArtifactName(vm *VM) error {
	example, exampleOK := strictString(vm.pop())
	program, programOK := strictString(vm.pop())
	kind, kindOK := strictString(vm.pop())
	experiment, experimentOK := strictString(vm.pop())
	d, descriptorOK := readSynthDescriptor(vm, experiment)
	if !exampleOK || !programOK || !kindOK || !experimentOK || !descriptorOK || program == "" || kind == "" {
		vm.push(Nil())
		return nil
	}
	base := strings.Join([]string{"SynthArtifact", synthEncode(d.experimentKey), synthEncode(kind), synthEncode(program), synthEncode(example)}, ".")
	vm.push(StringVal(freshSynthName(vm, base)))
	return nil
}

// bSynthFindExecutionObservation returns the unique execution observation for
// an experiment, subject program, and probe. Nil is returned for absent or
// ambiguous evidence so simplification comparison cannot silently pick one.
func bSynthFindExecutionObservation(vm *VM) error {
	probe, probeOK := strictString(vm.pop())
	subject, subjectOK := strictString(vm.pop())
	experiment, experimentOK := strictString(vm.pop())
	d, descriptorOK := readSynthDescriptor(vm, experiment)
	if !probeOK || !subjectOK || !experimentOK || !descriptorOK ||
		!validSynthDescriptor(vm, d, false) || vm.Store.Get(subject) == nil ||
		vm.Store.Get(probe) == nil || !vm.Store.IsA(probe, d.probeCategory) {
		vm.push(Nil())
		return nil
	}
	found := ""
	for _, name := range synthCategoryMembers(vm, d.simplificationExecObsCategory) {
		u := vm.Store.Get(name)
		if u.GetString("synthesisExperiment") == experiment &&
			u.GetString("subjectProgram") == subject && u.GetString("probe") == probe {
			if found != "" {
				vm.push(Nil())
				return nil
			}
			found = name
		}
	}
	if found == "" {
		vm.push(Nil())
	} else {
		vm.push(StringVal(found))
	}
	return nil
}

func bSynthReadyToFinalize(vm *VM) error {
	name, ok := strictString(vm.pop())
	d, descriptorOK := readSynthDescriptor(vm, name)
	if !ok || !descriptorOK || !validSynthDescriptor(vm, d, true) {
		vm.push(BoolVal(false))
		return nil
	}
	u := vm.Store.Get(name)
	candidates := u.GetStrings("candidateUnits")
	expected := u.GetInt("expectedCandidateCount")
	if !u.GetBool("generationComplete") || u.GetBool("finalizationComplete") || expected != synthCandidateCount(vm, d) || len(candidates) != expected || u.GetInt("evaluatedCandidateCount") != expected {
		vm.push(BoolVal(false))
		return nil
	}
	seen := map[string]bool{}
	seenDecisions := map[string]bool{}
	for _, candidate := range candidates {
		program := vm.Store.Get(candidate)
		decision := ""
		if program != nil {
			decision = program.GetString("creditDecision")
		}
		if seen[candidate] || seenDecisions[decision] || program == nil || !validSynthFinalCandidate(vm, d, name, candidate) {
			vm.push(BoolVal(false))
			return nil
		}
		seen[candidate] = true
		seenDecisions[decision] = true
	}
	vm.push(BoolVal(true))
	return nil
}

func validSynthFinalCandidate(vm *VM, d synthDescriptor, experiment, candidate string) bool {
	program := vm.Store.Get(candidate)
	components := program.GetStrings("components")
	_, names, semantics, definitions, valid := synthComponents(vm, experiment, stringListValue(components))
	if !valid || !sameStrings(names, components) || program.GetString("synthesisExperiment") != experiment ||
		!vm.Store.IsA(candidate, d.candidateCategory) || !program.GetBool("evaluatedProgramCorpus") ||
		program.GetInt("programLength") != len(components) ||
		!sameStrings(program.GetStrings("semanticSequence"), semantics) ||
		program.GetString("defn") != strings.Join(definitions, " ") ||
		program.GetString("synthesisMethod") != d.method || program.GetString("creditContext") != d.creditContext {
		return false
	}
	decision, err := programvocab.DecisionKey(d.method, semantics)
	if err != nil || program.GetString("creditDecision") != decision {
		return false
	}
	wantCreditors := append([]string{"H-EnumerateBoundedPrograms"}, components...)
	wantRoles := []string{"synthesis"}
	for index := range components {
		wantRoles = append(wantRoles, fmt.Sprintf("step-%d", index+1))
	}
	if !sameStrings(program.GetStrings("creditors"), wantCreditors) || !sameStrings(program.GetStrings("creditRoles"), wantRoles) {
		return false
	}

	evidenceName := program.GetString("evidenceUnit")
	evidence := vm.Store.Get(evidenceName)
	if evidence == nil || !vm.Store.IsA(evidenceName, d.evidenceCategory) ||
		evidence.GetString("synthesisExperiment") != experiment || evidence.GetString("program") != candidate {
		return false
	}
	matchingEvidence := 0
	for _, name := range synthCategoryMembers(vm, d.evidenceCategory) {
		u := vm.Store.Get(name)
		if u.GetString("synthesisExperiment") == experiment && u.GetString("program") == candidate {
			matchingEvidence++
		}
	}
	corpus, evaluated := program.GetInt("corpusSize"), program.GetInt("evaluatedCount")
	support, failures, invalid := program.GetInt("supportCount"), program.GetInt("failureCount"), program.GetInt("invalidCount")
	observations, results, examples := evidence.GetStrings("observations"), evidence.GetStrings("resultUnits"), evidence.GetStrings("trainingExamples")
	if matchingEvidence != 1 || corpus != len(synthCategoryMembers(vm, d.exampleCategory)) || evaluated != corpus ||
		support < 0 || failures < 0 || invalid < 0 || support+failures+invalid != corpus ||
		len(observations) != corpus || len(examples) != corpus || len(results) != support+failures ||
		evidence.GetInt("corpusSize") != corpus || evidence.GetInt("evaluatedCount") != evaluated ||
		evidence.GetInt("supportCount") != support || evidence.GetInt("failureCount") != failures ||
		evidence.GetInt("invalidCount") != invalid ||
		program.GetBool("exactProgram") != (support == corpus && failures == 0 && invalid == 0) {
		return false
	}
	seenExamples, seenObservations, seenResults := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, example := range examples {
		if seenExamples[example] || example == d.exampleCategory || !vm.Store.IsA(example, d.exampleCategory) {
			return false
		}
		seenExamples[example] = true
	}
	resultExamples := map[string]bool{}
	for _, resultName := range results {
		result := vm.Store.Get(resultName)
		example := ""
		if result != nil {
			example = result.GetString("example")
		}
		if seenResults[resultName] || result == nil || !vm.Store.IsA(resultName, d.resultCategory) ||
			result.GetString("synthesisExperiment") != experiment || result.GetString("program") != candidate ||
			!seenExamples[example] || resultExamples[example] {
			return false
		}
		seenResults[resultName] = true
		resultExamples[example] = true
	}
	observedExamples := map[string]bool{}
	observedSupport, observedFailures, observedInvalid := 0, 0, 0
	for _, observationName := range observations {
		observation := vm.Store.Get(observationName)
		example := ""
		if observation != nil {
			example = observation.GetString("example")
		}
		if seenObservations[observationName] || observedExamples[example] || observation == nil || !vm.Store.IsA(observationName, d.observationCategory) ||
			observation.GetString("synthesisExperiment") != experiment || observation.GetString("program") != candidate || !seenExamples[example] {
			return false
		}
		seenObservations[observationName], observedExamples[example] = true, true
		resultName := observation.GetString("resultUnit")
		resultMatchesExample := seenResults[resultName] && vm.Store.Get(resultName).GetString("example") == example
		exampleUnit := vm.Store.Get(example)
		input := anyToValue(exampleUnit.Get(d.inputSlot))
		expected := anyToValue(exampleUnit.Get(d.expectedSlot))
		if !synthCompare(vm, d.comparator, input, anyToValue(observation.Get("input"))) ||
			!synthCompare(vm, d.comparator, expected, anyToValue(observation.Get("expected"))) {
			return false
		}
		actual, executed := synthExecuteDefn(vm, candidate, input)
		validActual, validated := synthExecuteDefn(vm, d.outputValidator, actual)
		defined := executed && validated && validActual.Kind() == VBool && validActual.AsBool()
		if !defined {
			if observation.GetString("status") != "semantic-nil" || observation.GetBool("outcome") || resultName != "" || resultExamples[example] || observation.Get("actual") != nil {
				return false
			}
			observedInvalid++
			continue
		}
		if !resultMatchesExample || !resultExamples[example] ||
			!synthCompare(vm, d.comparator, actual, anyToValue(observation.Get("actual"))) ||
			!synthCompare(vm, d.comparator, actual, anyToValue(vm.Store.Get(resultName).Get(d.resultValueSlot))) {
			return false
		}
		matches := synthCompare(vm, d.comparator, actual, expected)
		if matches {
			if observation.GetString("status") != "match" || !observation.GetBool("outcome") {
				return false
			}
			observedSupport++
		} else {
			if observation.GetString("status") != "mismatch" || observation.GetBool("outcome") {
				return false
			}
			observedFailures++
		}
	}
	if observedSupport != support || observedFailures != failures || observedInvalid != invalid {
		return false
	}
	return true
}

func synthCompare(vm *VM, comparator string, left, right Value) bool {
	result, ok := synthExecuteDefn(vm, comparator, left, right)
	return ok && result.Kind() == VBool && result.AsBool()
}
