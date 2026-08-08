package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/chazu/nous/internal/unit"
	rivocab "github.com/chazu/nous/internal/vocab/ruleinduction"
)

func init() {
	registerVocabularyWords("ruleinduction", map[string]builtinFn{
		"ri-partial-valid?":       bRIPartialValid,
		"ri-profile-valid?":       bRIProfileValid,
		"ri-task-valid?":          bRITaskValid,
		"ri-refine-one":           bRIRefineOne,
		"ri-refinement-work":      bRIRefinementWork,
		"ri-complete-code":        bRICompleteCode,
		"ri-signature":            bRISignature,
		"ri-fixed-work":           bRIFixedWork,
		"ri-evaluation":           bRIEvaluation,
		"ri-evaluation-signature": bRIEvaluationSignature,
		"ri-evaluation-work":      bRIEvaluationWork,
		"ri-signature-has?":       bRISignatureHas,
		"ri-example-for?":         bRIExampleFor,
		"ri-example-x":            bRIExampleX,
		"ri-example-y":            bRIExampleY,
		"ri-example-positive?":    bRIExamplePositive,
		"ri-structural-subsumes?": bRIStructuralSubsumes,
		"ri-structural-work":      bRIStructuralWork,
		"ri-semantic-key":         bRISemanticKey,
		"ri-artifact-name":        bRIArtifactName,
		"ri-envelope":             bRIEnvelope,
		"ri-record-action":        bRIRecordAction,
		"ri-record-decision":      bRIRecordDecision,
		"ri-queue-rank":           bRIQueueRank,
		"ri-ready-to-select?":     bRIReadyToSelect,
		"ri-stage-one-complete?":  bRIStageOneComplete,
		"ri-experiment-complete?": bRIExperimentComplete,
	})
}

func riExampleRecord(vm *VM, text string) (rivocab.Example, bool) {
	if u := vm.Store.Get(text); directRIMember(u, "RuleInductionExample") {
		return rivocab.Example{X: u.GetInt("x"), Y: u.GetInt("y"), Positive: u.GetBool("positive")}, true
	}
	parts := strings.Split(text, ":")
	if len(parts) != 3 {
		return rivocab.Example{}, false
	}
	x, xErr := strconv.Atoi(parts[0])
	y, yErr := strconv.Atoi(parts[1])
	positive, pErr := strconv.ParseBool(parts[2])
	_, xOK := rivocab.PairIndex(x, y)
	return rivocab.Example{X: x, Y: y, Positive: positive}, xErr == nil && yErr == nil && pErr == nil && xOK
}

func bRIExampleFor(vm *VM) error {
	stage, experiment, text := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	_, valid := riExampleRecord(vm, text)
	if u := vm.Store.Get(text); u != nil {
		valid = valid && u.GetString("experiment") == experiment && u.GetString("stage") == stage
	}
	vm.push(BoolVal(valid))
	return nil
}

func bRIExampleX(vm *VM) error {
	example, ok := riExampleRecord(vm, vm.pop().AsString())
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(IntVal(example.X))
	}
	return nil
}
func bRIExampleY(vm *VM) error {
	example, ok := riExampleRecord(vm, vm.pop().AsString())
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(IntVal(example.Y))
	}
	return nil
}
func bRIExamplePositive(vm *VM) error {
	example, ok := riExampleRecord(vm, vm.pop().AsString())
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(BoolVal(example.Positive))
	}
	return nil
}

func riProfile(vm *VM, name string) (rivocab.ExperimentProfile, bool) {
	experiment := vm.Store.Get(name)
	if experiment == nil || !directRIMember(experiment, "RuleInductionExperiment") {
		return rivocab.ExperimentProfile{}, false
	}
	p := rivocab.ExperimentProfile{
		ProfileVersion: experiment.GetString("profileVersion"), ExperimentVersion: experiment.GetString("experimentVersion"), GeneratorVersion: experiment.GetString("generatorVersion"), GrammarVersion: experiment.GetString("grammarVersion"), CostVersion: experiment.GetString("costVersion"), OracleVersion: experiment.GetString("oracleVersion"), ReportVersion: experiment.GetString("reportVersion"), BaselineVersion: experiment.GetString("baselineVersion"), StatisticsVersion: experiment.GetString("statisticsVersion"), QueueVersion: experiment.GetString("queueVersion"), CacheVersion: experiment.GetString("cacheVersion"), IntegrityContract: experiment.GetString("integrityContract"),
		Panel: experiment.GetString("panel"), Seed: int64(experiment.GetInt("seed")), Policy: experiment.GetString("policy"),
		Categories:       rivocab.CategoryBindings{Partial: experiment.GetString("partialCategory"), Refinement: experiment.GetString("refinementCategory"), Candidate: experiment.GetString("candidateCategory"), Result: experiment.GetString("resultCategory"), Observation: experiment.GetString("observationCategory"), Evidence: experiment.GetString("evidenceCategory"), Constraint: experiment.GetString("constraintCategory"), Comparison: experiment.GetString("comparisonCategory"), Prune: experiment.GetString("pruneCategory"), Library: experiment.GetString("libraryCategory"), Provenance: experiment.GetString("provenanceCategory"), Projection: experiment.GetString("projectionCategory"), Transcript: experiment.GetString("transcriptCategory"), Boundary: experiment.GetString("boundaryCategory"), Corpus: experiment.GetString("corpusCategory"), Selection: experiment.GetString("selectionCategory"), Terminal: experiment.GetString("terminalCategory")},
		Tasks:            rivocab.TaskBindings{Start: experiment.GetString("startTaskSlot"), Refine: experiment.GetString("refineTaskSlot"), Evaluate: experiment.GetString("evaluateTaskSlot"), Continue: experiment.GetString("continueTaskSlot")},
		ConstantBindings: experiment.GetStrings("constantBindings"), PredicateBindings: experiment.GetStrings("predicateBindings"), Metarules: experiment.GetStrings("metarules"), Stage1Queue: experiment.GetStrings("stage1Queue"), Stage2Queue: experiment.GetStrings("stage2Queue"),
		CandidateCap: experiment.GetInt("candidateCap"), EvaluationCap: experiment.GetInt("evaluationCap"), FixedPointStepCap: experiment.GetInt("fixedPointStepCap"), SemanticWorkCap: experiment.GetInt("semanticWorkCap"), EngineCycleCap: experiment.GetInt("engineCycleCap"), AttributedUnitCap: experiment.GetInt("attributedUnitCap"), ReportByteCap: experiment.GetInt("reportByteCap"), InitialPriority: experiment.GetInt("initialPriority"), RefinePriority: experiment.GetInt("refinementPriority"), EvaluatePriority: experiment.GetInt("evaluationPriority"), InitialReason: experiment.GetString("initialReason"),
	}
	key, err := p.Key()
	if err != nil || key != experiment.GetString("experimentProfileKey") {
		return rivocab.ExperimentProfile{}, false
	}
	categories := p.Categories.Ordered()
	for _, category := range categories {
		if vm.Store.Get(category) == nil {
			return rivocab.ExperimentProfile{}, false
		}
	}
	for first, a := range categories {
		for second, b := range categories {
			if first != second && vm.Store.IsA(a, b) {
				return rivocab.ExperimentProfile{}, false
			}
		}
	}
	return p, true
}

func bRIProfileValid(vm *VM) error {
	_, ok := riProfile(vm, vm.pop().AsString())
	vm.push(BoolVal(ok))
	return nil
}

func bRITaskValid(vm *VM) error {
	kind, slot, name := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	u := vm.Store.Get(name)
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}
	experimentName := u.GetString("experiment")
	if kind == "start" || kind == "continue" {
		experimentName = name
	}
	experiment := vm.Store.Get(experimentName)
	if experiment == nil || experiment.GetString("experimentProfileKey") == "" {
		fallbackCategory, fallbackSlot := map[string]string{"start": "RuleInductionExperiment", "continue": "RuleInductionExperiment", "refine": "RuleInductionPartial", "evaluate": "RuleInductionCandidate"}[kind], map[string]string{"start": "riStart", "continue": "riContinue", "refine": "riRefine", "evaluate": "riEvaluate"}[kind]
		vm.push(BoolVal(experimentName == "RuleInductionSeed" && directRIMember(u, fallbackCategory) && slot == fallbackSlot))
		return nil
	}
	category, taskSlot := "", ""
	switch kind {
	case "start":
		category, taskSlot = "RuleInductionExperiment", experiment.GetString("startTaskSlot")
	case "continue":
		category, taskSlot = "RuleInductionExperiment", experiment.GetString("continueTaskSlot")
	case "refine":
		category, taskSlot = experiment.GetString("partialCategory"), experiment.GetString("refineTaskSlot")
	case "evaluate":
		category, taskSlot = experiment.GetString("candidateCategory"), experiment.GetString("evaluateTaskSlot")
	}
	vm.push(BoolVal(category != "" && directRIMember(u, category) && slot == taskSlot))
	return nil
}

func riStageIdentity(experiment *unit.Unit, stage string) (int, string, bool) {
	switch stage {
	case "stage1":
		key := experiment.GetString("stage1ProfileKey")
		return 1, key, key != ""
	case "stage2":
		key := experiment.GetString("stage2ProfileKey")
		return 2, key, key != ""
	default:
		return 0, "", false
	}
}

func riArtifactIdentity(experiment *unit.Unit, stage, kind, semantic string) (int, string, string, bool) {
	stageIndex, stageKey, ok := riStageIdentity(experiment, stage)
	if !ok || kind == "" || semantic == "" {
		return 0, "", "", false
	}
	return stageIndex, stageKey, rivocab.ArtifactSemanticKey(kind, semantic), true
}

func riArtifactCategory(experiment *unit.Unit, kind string) string {
	return map[string]string{"partial": experiment.GetString("partialCategory"), "refinement": experiment.GetString("refinementCategory"), "candidate": experiment.GetString("candidateCategory"), "result": experiment.GetString("resultCategory"), "observation": experiment.GetString("observationCategory"), "evidence": experiment.GetString("evidenceCategory"), "constraint": experiment.GetString("constraintCategory"), "comparison": experiment.GetString("comparisonCategory"), "prune": experiment.GetString("pruneCategory"), "library": experiment.GetString("libraryCategory"), "provenance": experiment.GetString("provenanceCategory"), "projection": experiment.GetString("projectionCategory"), "transcript": experiment.GetString("transcriptCategory"), "boundary": experiment.GetString("boundaryCategory"), "corpus": experiment.GetString("corpusCategory"), "selection": experiment.GetString("selectionCategory"), "terminal": experiment.GetString("terminalCategory")}[kind]
}

func riAuthoritativeDigest(u *unit.Unit) string {
	slots := make(map[string]any, len(u.Slots))
	for slot, value := range u.Slots {
		if slot != "authoritativeDigest" {
			slots[slot] = value
		}
	}
	return rivocab.SemanticDigest(rivocab.ArtifactSnapshot{Name: u.Name, Slots: slots})
}

func riAuthoritativeValid(u *unit.Unit) bool {
	return u != nil && len(u.GetString("authoritativeDigest")) == 71 && u.GetString("authoritativeDigest") == riAuthoritativeDigest(u)
}

func riSealArtifacts(store *unit.Store, experiment *unit.Unit) {
	for _, name := range store.All() {
		u := store.Get(name)
		kind := u.GetString("artifactKind")
		if u.GetString("experiment") == experiment.Name && u.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && kind != "" && kind != "descriptor" && kind != "transcript" {
			u.Set("authoritativeDigest", riAuthoritativeDigest(u))
		}
	}
}

func bRIArtifactName(vm *VM) error {
	base, semantic, kind, stage, experimentName := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	experiment := vm.Store.Get(experimentName)
	if experiment == nil || experiment.GetString("experimentProfileKey") == "" {
		vm.push(StringVal(base))
		return nil
	}
	stageIndex, stageKey, semanticKey, ok := riArtifactIdentity(experiment, stage, kind, semantic)
	if !ok {
		vm.push(Nil())
		return nil
	}
	matches := func(u *unit.Unit) bool {
		return u != nil && directRIMember(u, riArtifactCategory(experiment, kind)) && u.GetString("experiment") == experimentName && u.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && u.GetInt("stageIndex") == stageIndex && u.GetString("stageProfileKey") == stageKey && u.GetString("artifactKind") == kind && u.GetString("semanticKey") == semanticKey
	}
	existing := ""
	for _, name := range vm.Store.All() {
		if matches(vm.Store.Get(name)) {
			if existing != "" || !riAuthoritativeValid(vm.Store.Get(name)) {
				vm.push(Nil())
				return nil
			}
			existing = name
		}
	}
	if existing != "" {
		vm.push(StringVal(existing))
		return nil
	}
	for collision := 0; collision <= 64; collision++ {
		name := base
		if collision > 0 {
			name = fmt.Sprintf("%s-collision-%d", base, collision)
		}
		if !vm.Store.Has(name) {
			vm.push(StringVal(name))
			return nil
		}
	}
	vm.push(Nil())
	return nil
}

func bRIEnvelope(vm *VM) error {
	semantic, kind, stage, experimentName, name := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	u, experiment := vm.Store.Get(name), vm.Store.Get(experimentName)
	if u == nil || experiment == nil {
		vm.push(BoolVal(false))
		return nil
	}
	if experiment.GetString("experimentProfileKey") == "" && experimentName == "RuleInductionSeed" {
		vm.push(BoolVal(true))
		return nil
	}
	stageIndex, stageKey, semanticKey, ok := riArtifactIdentity(experiment, stage, kind, semantic)
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	want := map[string]any{"experiment": experimentName, "experimentProfileKey": experiment.GetString("experimentProfileKey"), "stageIndex": stageIndex, "stageProfileKey": stageKey, "artifactKind": kind, "semanticKey": semanticKey}
	for slot, value := range want {
		if u.Has(slot) && !reflect.DeepEqual(u.Get(slot), value) {
			vm.push(BoolVal(false))
			return nil
		}
	}
	for slot, value := range want {
		u.Set(slot, value)
	}
	probes := 1
	if strings.HasSuffix(name, "-collision-1") {
		probes = 2
	}
	u.Set("allocationProbes", probes)
	vm.push(BoolVal(true))
	return nil
}

func riEnvelopeValid(experiment, artifact *unit.Unit, stage, kind, semantic string) bool {
	if experiment.GetString("experimentProfileKey") == "" && experiment.Name == "RuleInductionSeed" {
		return true
	}
	stageIndex, stageKey, semanticKey, ok := riArtifactIdentity(experiment, stage, kind, semantic)
	return ok && artifact.GetString("experiment") == experiment.Name && artifact.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && artifact.GetInt("stageIndex") == stageIndex && artifact.GetString("stageProfileKey") == stageKey && artifact.GetString("artifactKind") == kind && artifact.GetString("semanticKey") == semanticKey
}

func riStageArtifactDigest(store *unit.Store, experiment string, stage int) string {
	records := []rivocab.ArtifactSnapshot{}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") != experiment || u.GetInt("stageIndex") != stage || u.GetString("artifactKind") == "" || u.GetString("artifactKind") == "boundary" || u.GetString("artifactKind") == "descriptor" {
			continue
		}
		slots := make(map[string]any, len(u.Slots))
		for slot, value := range u.Slots {
			slots[slot] = value
		}
		records = append(records, rivocab.ArtifactSnapshot{Name: name, Slots: slots})
	}
	return rivocab.StageArtifactDigest(records)
}

func riActionDigest(experimentKey, stageKey, previous string, sequence, chargeIndex int, action, semantic string, domainWork int, artifactLink, artifactSetDigest string, workBefore, workAfter, remainingSemantic, remainingEvaluations, remainingFixedPoint, remainingUnits int) string {
	encoded, _ := json.Marshal([]any{experimentKey, stageKey, previous, sequence, chargeIndex, action, semantic, domainWork, artifactLink, artifactSetDigest, workBefore, workAfter, remainingSemantic, remainingEvaluations, remainingFixedPoint, remainingUnits})
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func riActionLinkValid(experiment, link *unit.Unit, action string) bool {
	if action == "termination" {
		return link == nil
	}
	if link == nil || link.GetString("experiment") != experiment.Name {
		return false
	}
	want := map[string]string{"start": "partial", "refine": "partial", "evaluation": "candidate", "comparison": "comparison", "constraint": "constraint", "local-invention": "library", "promotion": "candidate", "fallback": "candidate"}[action]
	if action == "decision" {
		return link.GetString("artifactKind") == "selection" || link.GetString("artifactKind") == "terminal"
	}
	return want != "" && link.GetString("artifactKind") == want
}

func riActionDomainWorkValid(link *unit.Unit, action string, domainWork int) bool {
	switch action {
	case "start", "constraint", "local-invention", "promotion", "termination", "fallback":
		return domainWork == 1
	case "decision":
		return domainWork == 0
	case "refine":
		partial, ok := riPartial(link.GetString("partial"))
		if !ok || partial.Bound() == 0 {
			return false
		}
		want := 2
		if partial.Bound() == len(partial.Fields) {
			want++
		}
		return domainWork == want
	case "evaluation":
		return link.GetBool("riEvaluated") && domainWork == link.GetInt("fixedWork")+link.GetInt("exampleCount")
	case "comparison":
		return domainWork == link.GetInt("thetaWork")
	default:
		return false
	}
}

func riBudgetUsage(store *unit.Store, experiment *unit.Unit) (evaluations, fixedPoint, attributed int) {
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") != experiment.Name || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || u.GetString("artifactKind") == "" {
			continue
		}
		attributed++
		if u.GetString("artifactKind") == "candidate" && u.GetBool("riEvaluated") {
			evaluations++
			fixedPoint += u.GetInt("fixedWork")
		}
	}
	return
}

func riBudgetsWithinCap(vm *VM, experiment *unit.Unit) bool {
	evaluations, fixedPoint, attributed := riBudgetUsage(vm.Store, experiment)
	return !experiment.GetBool("budgetExceeded") && riAuditedPrimaryWork(vm, experiment) <= experiment.GetInt("semanticWorkCap") && evaluations <= experiment.GetInt("evaluationCap") && fixedPoint <= experiment.GetInt("fixedPointStepCap") && attributed <= experiment.GetInt("attributedUnitCap")
}

func bRIRecordAction(vm *VM) error {
	domainWork, semantic, action, stage, experimentName := vm.pop().AsInt(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	experiment := vm.Store.Get(experimentName)
	if experiment == nil || domainWork < 0 || action == "" || semantic == "" {
		vm.push(BoolVal(false))
		return nil
	}
	if experiment.GetString("experimentProfileKey") == "" && experimentName == "RuleInductionSeed" {
		vm.push(BoolVal(true))
		return nil
	}
	stageIndex, stageKey, _, ok := riArtifactIdentity(experiment, stage, "transcript", "action:"+action+":"+semantic)
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	category := experiment.GetString("transcriptCategory")
	indexSlot, namesSlot := fmt.Sprintf("stage%dTranscriptActions", stageIndex), fmt.Sprintf("stage%dTranscriptUnits", stageIndex)
	actionIdentity := action + "\x00" + semantic
	identities, names := experiment.GetStrings(indexSlot), experiment.GetStrings(namesSlot)
	if len(identities) != len(names) {
		vm.push(BoolVal(false))
		return nil
	}
	for index, identity := range identities {
		if identity == actionIdentity {
			u := vm.Store.Get(names[index])
			vm.push(BoolVal(directRIMember(u, category) && u.GetInt("domainWork") == domainWork && riEnvelopeValid(experiment, u, stage, "transcript", "action:"+action+":"+semantic) && riTranscriptValid(vm, experiment)))
			return nil
		}
	}
	var link *unit.Unit
	if action != "termination" {
		link = vm.Store.Get(semantic)
	}
	if !riActionLinkValid(experiment, link, action) || !riActionDomainWorkValid(link, action, domainWork) {
		vm.push(BoolVal(false))
		return nil
	}
	seqSlot, digestSlot := fmt.Sprintf("stage%dTranscriptSeq", stageIndex), fmt.Sprintf("stage%dTranscriptDigest", stageIndex)
	sequence := experiment.GetInt(seqSlot) + 1
	chargeIndex := experiment.GetInt("transcriptChargeIndex") + 1
	previous := experiment.GetString(digestSlot)
	if previous == "" {
		previous = "genesis"
	}
	base := fmt.Sprintf("RI.Transcript.%s.stage%d.%04d", experimentName, stageIndex, sequence)
	name := base
	if vm.Store.Has(name) {
		name = base + "-collision-1"
		if vm.Store.Has(name) {
			vm.push(BoolVal(false))
			return nil
		}
	}
	u := unit.New(name)
	u.Set("isA", []string{category, "Anything"})
	u.Set("experiment", experimentName)
	u.Set("experimentProfileKey", experiment.GetString("experimentProfileKey"))
	u.Set("stageIndex", stageIndex)
	u.Set("stageProfileKey", stageKey)
	u.Set("artifactKind", "transcript")
	u.Set("semanticKey", rivocab.ArtifactSemanticKey("transcript", "action:"+action+":"+semantic))
	u.Set("sequence", sequence)
	u.Set("chargeIndex", chargeIndex)
	u.Set("previousDigest", previous)
	u.Set("action", action)
	u.Set("actionSemantic", semantic)
	u.Set("domainWork", domainWork)
	artifactLink := ""
	if link != nil {
		artifactLink = link.Name
	}
	u.Set("artifactLink", artifactLink)
	probes := 1
	if strings.HasSuffix(name, "-collision-1") {
		probes = 2
	}
	u.Set("allocationProbes", probes)
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, name)
	artifactUnits := []string{}
	for _, artifactName := range vm.Store.All() {
		artifact := vm.Store.Get(artifactName)
		if artifact.GetString("experiment") == experimentName && artifact.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && artifact.GetString("artifactKind") != "" && artifact.GetInt("chargeIndex") == 0 {
			artifact.Set("chargeIndex", chargeIndex)
			artifactUnits = append(artifactUnits, artifactName)
		}
	}
	sort.Strings(artifactUnits)
	if action == "evaluation" || action == "comparison" {
		link.Set("semanticChargeIndex", chargeIndex)
	}
	workBefore := experiment.GetInt("primaryWorkUsed")
	workAfter := riAuditedPrimaryWork(vm, experiment)
	evaluations, fixedPoint, attributed := riBudgetUsage(vm.Store, experiment)
	remainingSemantic := experiment.GetInt("semanticWorkCap") - workAfter
	remainingEvaluations := experiment.GetInt("evaluationCap") - evaluations
	remainingFixedPoint := experiment.GetInt("fixedPointStepCap") - fixedPoint
	remainingUnits := experiment.GetInt("attributedUnitCap") - attributed
	artifactSetDigest := rivocab.SemanticDigest(artifactUnits)
	digest := riActionDigest(experiment.GetString("experimentProfileKey"), stageKey, previous, sequence, chargeIndex, action, semantic, domainWork, artifactLink, artifactSetDigest, workBefore, workAfter, remainingSemantic, remainingEvaluations, remainingFixedPoint, remainingUnits)
	u.Set("artifactUnits", artifactUnits)
	u.Set("artifactSetDigest", artifactSetDigest)
	u.Set("workBefore", workBefore)
	u.Set("workAfter", workAfter)
	u.Set("remainingSemanticWork", remainingSemantic)
	u.Set("remainingEvaluations", remainingEvaluations)
	u.Set("remainingFixedPoint", remainingFixedPoint)
	u.Set("remainingAttributedUnits", remainingUnits)
	u.Set("prefixDigest", digest)
	experiment.Set(indexSlot, append(identities, actionIdentity))
	experiment.Set(namesSlot, append(names, name))
	experiment.Set(seqSlot, sequence)
	experiment.Set(digestSlot, digest)
	experiment.Set("transcriptChargeIndex", chargeIndex)
	experiment.Set("primaryWorkUsed", workAfter)
	if remainingSemantic < 0 || remainingEvaluations < 0 || remainingFixedPoint < 0 || remainingUnits < 0 {
		experiment.Set("budgetExceeded", true)
	}
	riSealArtifacts(vm.Store, experiment)
	vm.push(BoolVal(true))
	return nil
}

func riTranscriptValid(vm *VM, experiment *unit.Unit) bool {
	if experiment.Name == "RuleInductionSeed" && experiment.GetString("experimentProfileKey") == "" {
		return true
	}
	category := experiment.GetString("transcriptCategory")
	actions := []*unit.Unit{}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if directRIMember(u, category) && u.GetString("experiment") == experiment.Name {
			actions = append(actions, u)
		}
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].GetInt("chargeIndex") < actions[j].GetInt("chargeIndex") })
	stageSequence := [2]int{}
	stagePrevious := [2]string{"genesis", "genesis"}
	previousWork := 0
	for index, transcript := range actions {
		chargeIndex := index + 1
		stageIndex := transcript.GetInt("stageIndex")
		if stageIndex < 1 || stageIndex > 2 {
			return false
		}
		stageSequence[stageIndex-1]++
		stage := fmt.Sprintf("stage%d", stageIndex)
		semantic, action := transcript.GetString("actionSemantic"), transcript.GetString("action")
		var link *unit.Unit
		if transcript.GetString("artifactLink") != "" {
			link = vm.Store.Get(transcript.GetString("artifactLink"))
		}
		artifactUnits := transcript.GetStrings("artifactUnits")
		if transcript.GetInt("chargeIndex") != chargeIndex || transcript.GetInt("sequence") != stageSequence[stageIndex-1] || transcript.GetString("previousDigest") != stagePrevious[stageIndex-1] || transcript.GetInt("workBefore") != previousWork || !sort.StringsAreSorted(artifactUnits) || rivocab.SemanticDigest(artifactUnits) != transcript.GetString("artifactSetDigest") || !riActionLinkValid(experiment, link, action) || !riActionDomainWorkValid(link, action, transcript.GetInt("domainWork")) || !riEnvelopeValid(experiment, transcript, stage, "transcript", "action:"+action+":"+semantic) {
			return false
		}
		seenArtifact := map[string]bool{}
		for _, artifactName := range artifactUnits {
			artifact := vm.Store.Get(artifactName)
			if artifact == nil || seenArtifact[artifactName] || artifact.GetString("experiment") != experiment.Name || artifact.GetInt("chargeIndex") != chargeIndex {
				return false
			}
			seenArtifact[artifactName] = true
		}
		workAfter := riAuditedWorkThrough(vm, experiment, chargeIndex)
		evaluations, fixedPoint, attributed := riBudgetUsageThrough(vm.Store, experiment, chargeIndex)
		remainingSemantic := experiment.GetInt("semanticWorkCap") - workAfter
		remainingEvaluations := experiment.GetInt("evaluationCap") - evaluations
		remainingFixedPoint := experiment.GetInt("fixedPointStepCap") - fixedPoint
		remainingUnits := experiment.GetInt("attributedUnitCap") - attributed
		_, stageKey, _ := riStageIdentity(experiment, stage)
		want := riActionDigest(experiment.GetString("experimentProfileKey"), stageKey, stagePrevious[stageIndex-1], stageSequence[stageIndex-1], chargeIndex, action, semantic, transcript.GetInt("domainWork"), transcript.GetString("artifactLink"), transcript.GetString("artifactSetDigest"), previousWork, workAfter, remainingSemantic, remainingEvaluations, remainingFixedPoint, remainingUnits)
		if transcript.GetInt("workAfter") != workAfter || transcript.GetInt("remainingSemanticWork") != remainingSemantic || transcript.GetInt("remainingEvaluations") != remainingEvaluations || transcript.GetInt("remainingFixedPoint") != remainingFixedPoint || transcript.GetInt("remainingAttributedUnits") != remainingUnits || transcript.GetString("prefixDigest") != want {
			return false
		}
		previousWork = workAfter
		stagePrevious[stageIndex-1] = want
	}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") == experiment.Name && u.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && u.GetString("artifactKind") != "" && u.GetInt("chargeIndex") == 0 {
			return false
		}
	}
	return experiment.GetInt("transcriptChargeIndex") == len(actions) && experiment.GetInt("primaryWorkUsed") == previousWork && experiment.GetInt("stage1TranscriptSeq") == stageSequence[0] && experiment.GetInt("stage2TranscriptSeq") == stageSequence[1] && (stageSequence[0] == 0 || experiment.GetString("stage1TranscriptDigest") == stagePrevious[0]) && (stageSequence[1] == 0 || experiment.GetString("stage2TranscriptDigest") == stagePrevious[1])
}

func bRIRecordDecision(vm *VM) error {
	link, semantic, kind, stage, experimentName := vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString(), vm.pop().AsString()
	experiment := vm.Store.Get(experimentName)
	if experiment == nil || (kind != "selection" && kind != "terminal") || semantic == "" {
		vm.push(BoolVal(false))
		return nil
	}
	if experiment.GetString("experimentProfileKey") == "" && experimentName == "RuleInductionSeed" {
		vm.push(BoolVal(true))
		return nil
	}
	stageIndex, stageKey, semanticKey, ok := riArtifactIdentity(experiment, stage, kind, kind+":"+semantic)
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	category := experiment.GetString(kind + "Category")
	label := map[string]string{"selection": "Selection", "terminal": "Terminal"}[kind]
	base := fmt.Sprintf("RI.%s.%s.stage%d", label, experimentName, stageIndex)
	name := base
	if existing := vm.Store.Get(base); existing != nil {
		if existing.GetString("semanticKey") == semanticKey && existing.GetString("stageProfileKey") == stageKey && existing.GetString("link") == link {
			vm.push(BoolVal(true))
			return nil
		}
		name = base + "-collision-1"
		if vm.Store.Has(name) {
			vm.push(BoolVal(false))
			return nil
		}
	}
	u := unit.New(name)
	u.Set("isA", []string{category, "Anything"})
	u.Set("experiment", experimentName)
	u.Set("experimentProfileKey", experiment.GetString("experimentProfileKey"))
	u.Set("stageIndex", stageIndex)
	u.Set("stageProfileKey", stageKey)
	u.Set("artifactKind", kind)
	u.Set("semanticKey", semanticKey)
	u.Set("semantic", semantic)
	u.Set("link", link)
	u.Set("barrierDigest", experiment.GetString(fmt.Sprintf("stage%dTranscriptDigest", stageIndex)))
	probes := 1
	if strings.HasSuffix(name, "-collision-1") {
		probes = 2
	}
	u.Set("allocationProbes", probes)
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, name)
	experiment.Set(fmt.Sprintf("stage%d%sUnit", stageIndex, label), name)
	vm.push(BoolVal(true))
	return nil
}

func riDecisionArtifactsValid(vm *VM, experiment *unit.Unit) bool {
	if experiment.GetString("experimentProfileKey") == "" {
		return true
	}
	selectionCategory, terminalCategory := experiment.GetString("selectionCategory"), experiment.GetString("terminalCategory")
	var selections, terminals []*unit.Unit
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		if directRIMember(u, selectionCategory) {
			selections = append(selections, u)
		}
		if directRIMember(u, terminalCategory) {
			terminals = append(terminals, u)
		}
	}
	wantSelections, wantTerminals := 2, 2
	if experiment.GetString("stage2ProfileKey") == "" {
		wantSelections, wantTerminals = 1, 1
	}
	if experiment.GetString("terminal") == "no-solution" || experiment.GetString("terminal") == "budget-exhausted" {
		wantSelections = 1
	}
	if len(selections) != wantSelections || len(terminals) != wantTerminals {
		return false
	}
	for _, selection := range selections {
		stage := fmt.Sprintf("stage%d", selection.GetInt("stageIndex"))
		semantic := selection.GetString("semantic")
		if !riEnvelopeValid(experiment, selection, stage, "selection", "selection:"+semantic) || !riDigestRecordsAction(vm.Store, experiment.Name, selection.GetInt("stageIndex"), selection.GetString("barrierDigest"), "promotion") {
			return false
		}
		if selection.GetInt("stageIndex") == 1 && semantic != experiment.GetString("frozenCode") {
			return false
		}
		if selection.GetInt("stageIndex") == 2 && semantic != experiment.GetString("selectedCode") {
			return false
		}
	}
	for _, terminal := range terminals {
		stage := fmt.Sprintf("stage%d", terminal.GetInt("stageIndex"))
		semantic := terminal.GetString("semantic")
		if !riEnvelopeValid(experiment, terminal, stage, "terminal", "terminal:"+semantic) || !riDigestRecordsAction(vm.Store, experiment.Name, terminal.GetInt("stageIndex"), terminal.GetString("barrierDigest"), "termination") {
			return false
		}
		if terminal.GetInt("stageIndex") == 1 && semantic != "awaiting-stage-2" {
			return false
		}
		if terminal.GetInt("stageIndex") == 2 && semantic != experiment.GetString("terminal") {
			return false
		}
	}
	return true
}

func riDigestRecordsAction(store *unit.Store, experiment string, stage int, digest, action string) bool {
	matches := 0
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") == experiment && u.GetString("artifactKind") == "transcript" && u.GetInt("stageIndex") == stage && u.GetString("prefixDigest") == digest && u.GetString("action") == action {
			matches++
		}
	}
	return matches == 1
}

type riEvaluationRecord struct {
	Signature string       `json:"signature"`
	Work      rivocab.Work `json:"work"`
}

func parseRIEvaluation(value Value) (riEvaluationRecord, bool) {
	if value.Kind() != VString {
		return riEvaluationRecord{}, false
	}
	var record riEvaluationRecord
	err := json.Unmarshal([]byte(value.AsString()), &record)
	return record, err == nil && len(record.Signature) == rivocab.GroundPairCount
}

func bRIEvaluation(vm *VM) error {
	relation, work, ok := riEvaluate(vm)
	if !ok {
		vm.push(Nil())
		return nil
	}
	encoded, _ := json.Marshal(riEvaluationRecord{Signature: relation.Signature(), Work: work})
	vm.push(StringVal(string(encoded)))
	return nil
}

func bRIEvaluationSignature(vm *VM) error {
	record, ok := parseRIEvaluation(vm.pop())
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(StringVal(record.Signature))
	}
	return nil
}

func bRIEvaluationWork(vm *VM) error {
	record, ok := parseRIEvaluation(vm.pop())
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(IntVal(record.Work.FixedPointTotal()))
	}
	return nil
}

func riPartial(text string) (rivocab.Partial, bool) {
	if len(text) != 4 {
		return rivocab.Partial{}, false
	}
	p := rivocab.Partial{}
	for i := range text {
		if text[i] == '-' {
			p.Fields[i] = -1
			continue
		}
		if text[i] < '0' || text[i] > '9' {
			return rivocab.Partial{}, false
		}
		p.Fields[i] = int(text[i] - '0')
	}
	return p, p.Valid()
}

func riPartialText(p rivocab.Partial) string {
	b := make([]byte, 4)
	for i, field := range p.Fields {
		b[i] = '-'
		if field >= 0 {
			b[i] = byte('0' + field)
		}
	}
	return string(b)
}

func riDefinition(code string) (rivocab.Definition, bool) {
	if len(code) != 2 || code[0] < '0' || code[0] > '5' || code[1] < '0' || code[1] > '5' {
		return rivocab.Definition{}, false
	}
	clause := func(raw byte) rivocab.Clause {
		value := int(raw - '0')
		return rivocab.Clause{Kind: rivocab.ClauseKind(value / 3), Background: value % 3}
	}
	definition, err := rivocab.Normalize(rivocab.Definition{Clauses: [2]rivocab.Clause{clause(code[0]), clause(code[1])}})
	return definition, err == nil
}

func bRIPartialValid(vm *VM) error {
	_, ok := riPartial(vm.pop().AsString())
	vm.push(BoolVal(ok))
	return nil
}

func bRIRefineOne(vm *VM) error {
	partial, ok := riPartial(vm.pop().AsString())
	if !ok {
		vm.push(Nil())
		return nil
	}
	children, err := rivocab.RefineOne(partial)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	values := make([]Value, len(children))
	for index, child := range children {
		values[index] = StringVal(riPartialText(child))
	}
	vm.push(ListVal(values))
	return nil
}

func bRIRefinementWork(vm *VM) error {
	partial, ok := riPartial(vm.pop().AsString())
	if !ok || partial.Bound() == 0 {
		vm.push(Nil())
		return nil
	}
	work := 2 // one field bind and validation
	if partial.Bound() == len(partial.Fields) {
		work++
	} // clause-order comparison
	vm.push(IntVal(work))
	return nil
}

func bRICompleteCode(vm *VM) error {
	partial, ok := riPartial(vm.pop().AsString())
	if !ok {
		vm.push(Nil())
		return nil
	}
	definition, err := partial.Definition()
	if err != nil {
		vm.push(Nil())
		return nil
	}
	code, _ := definition.Code()
	vm.push(StringVal(code))
	return nil
}

func riBackground(value Value) ([rivocab.PredicateCount]rivocab.Relation, bool) {
	var background [rivocab.PredicateCount]rivocab.Relation
	for _, item := range value.AsList() {
		text := item.AsString()
		parts := make([]int, 3)
		part, start := 0, 0
		for index := 0; index <= len(text); index++ {
			if index < len(text) && text[index] != ':' {
				continue
			}
			if part >= 3 {
				return background, false
			}
			parsed, err := strconv.Atoi(text[start:index])
			if err != nil {
				return background, false
			}
			parts[part] = parsed
			part++
			start = index + 1
		}
		if part != 3 || parts[0] < 0 || parts[0] >= rivocab.PredicateCount || !background[parts[0]].Add(parts[1], parts[2]) {
			return background, false
		}
	}
	return background, true
}

func riEvaluate(vm *VM) (rivocab.Relation, rivocab.Work, bool) {
	facts, code := vm.pop(), vm.pop().AsString()
	definition, ok := riDefinition(code)
	if !ok {
		return rivocab.Relation{}, rivocab.Work{}, false
	}
	background, ok := riBackground(facts)
	if !ok {
		return rivocab.Relation{}, rivocab.Work{}, false
	}
	relation, work, err := rivocab.Evaluate(definition, background)
	return relation, work, err == nil
}

func bRISignature(vm *VM) error {
	relation, _, ok := riEvaluate(vm)
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(StringVal(relation.Signature()))
	}
	return nil
}

func bRIFixedWork(vm *VM) error {
	_, work, ok := riEvaluate(vm)
	if !ok {
		vm.push(Nil())
	} else {
		vm.push(IntVal(work.FixedPointTotal()))
	}
	return nil
}

func bRISignatureHas(vm *VM) error {
	y, x, signature := vm.pop().AsInt(), vm.pop().AsInt(), vm.pop().AsString()
	index, ok := rivocab.PairIndex(x, y)
	vm.push(BoolVal(ok && len(signature) == rivocab.GroundPairCount && signature[index] == '1'))
	return nil
}

func bRIStructuralSubsumes(vm *VM) error {
	specific, general := vm.pop().AsString(), vm.pop().AsString()
	g, gok := riDefinition(general)
	s, sok := riDefinition(specific)
	if !gok || !sok {
		vm.push(Nil())
		return nil
	}
	ok, _, err := rivocab.StructurallySubsumes(g, s)
	if err != nil {
		vm.push(Nil())
	} else {
		vm.push(BoolVal(ok))
	}
	return nil
}

func bRIStructuralWork(vm *VM) error {
	specific, general := vm.pop().AsString(), vm.pop().AsString()
	g, gok := riDefinition(general)
	s, sok := riDefinition(specific)
	if !gok || !sok {
		vm.push(Nil())
		return nil
	}
	_, work, err := rivocab.StructurallySubsumes(g, s)
	if err != nil {
		vm.push(Nil())
	} else {
		vm.push(IntVal(work))
	}
	return nil
}

func bRISemanticKey(vm *VM) error {
	value, kind := vm.pop().AsString(), vm.pop().AsString()
	sum := sha256.Sum256([]byte("rule-induction/v1\x00" + kind + "\x00" + value))
	vm.push(StringVal("sha256:" + hex.EncodeToString(sum[:])))
	return nil
}

func bRIQueueRank(vm *VM) error {
	queue, code := vm.pop().AsList(), vm.pop().AsString()
	for index, item := range queue {
		if item.Kind() == VString && item.AsString() == code {
			vm.push(IntVal(index))
			return nil
		}
	}
	vm.push(Nil())
	return nil
}

func directRIMember(u *unit.Unit, category string) bool {
	if u == nil || u.Name == category {
		return false
	}
	for _, parent := range u.GetStrings("isA") {
		if parent == category {
			return true
		}
	}
	return false
}

func riCandidateByCode(store *unit.Store, experiment, stage, code string) []*unit.Unit {
	var result []*unit.Unit
	category := "RuleInductionCandidate"
	if descriptor := store.Get(experiment); descriptor != nil && descriptor.GetString("candidateCategory") != "" {
		category = descriptor.GetString("candidateCategory")
	}
	for _, name := range store.All() {
		u := store.Get(name)
		if directRIMember(u, category) && u.GetString("experiment") == experiment && u.GetString("stage") == stage && u.GetString("definitionCode") == code && !u.GetBool("projection") {
			result = append(result, u)
		}
	}
	return result
}

func riDirectArtifacts(store *unit.Store, category, experiment, candidate, example string) []*unit.Unit {
	var result []*unit.Unit
	for _, name := range store.All() {
		u := store.Get(name)
		if directRIMember(u, category) && u.GetString("experiment") == experiment &&
			(candidate == "" || u.GetString("candidate") == candidate) &&
			(example == "" || u.GetString("example") == example) {
			result = append(result, u)
		}
	}
	return result
}

func riCandidateEvidenceValid(vm *VM, experiment, candidate *unit.Unit) bool {
	candidateCategory := experiment.GetString("candidateCategory")
	if candidateCategory == "" {
		candidateCategory = "RuleInductionCandidate"
	}
	if candidate == nil || !directRIMember(candidate, candidateCategory) || candidate.GetString("experiment") != experiment.Name || !candidate.GetBool("riEvaluated") {
		return false
	}
	candidateSemantic := "candidate:" + candidate.GetString("definitionCode")
	if candidate.GetBool("projection") {
		candidateSemantic = "projection:" + candidate.GetString("definitionCode")
	}
	if !riEnvelopeValid(experiment, candidate, candidate.GetString("stage"), "candidate", candidateSemantic) {
		return false
	}
	definition, ok := riDefinition(candidate.GetString("definitionCode"))
	if !ok {
		return false
	}
	background, ok := riBackground(ListVal(stringsToValues(experiment.GetStrings("facts"))))
	if !ok {
		return false
	}
	relation, work, err := rivocab.Evaluate(definition, background)
	if err != nil || candidate.GetString("signature") != relation.Signature() {
		return false
	}
	wantWork := work.FixedPointTotal()
	if candidate.GetBool("projection") && experiment.GetString("reuseMode") == "shared-library" {
		wantWork = 0
	}
	if candidate.GetInt("fixedWork") != wantWork {
		return false
	}
	examples, examplesOK := riCandidateExamples(vm, experiment, candidate.GetString("stage"))
	if !examplesOK {
		return false
	}
	seen, support, failures, falsePositive, falseNegative := 0, 0, 0, 0, 0
	for _, binding := range examples {
		example, name := binding.Example, binding.ID
		seen++
		actual := relation.Has(example.X, example.Y)
		outcome := actual == example.Positive
		if outcome {
			support++
		} else {
			failures++
			if actual {
				falsePositive++
			} else {
				falseNegative++
			}
		}
		resultCategory, observationCategory := experiment.GetString("resultCategory"), experiment.GetString("observationCategory")
		if resultCategory == "" {
			resultCategory = "RuleInductionResult"
		}
		if observationCategory == "" {
			observationCategory = "RuleInductionObservation"
		}
		results := riDirectArtifacts(vm.Store, resultCategory, experiment.Name, candidate.Name, name)
		observations := riDirectArtifacts(vm.Store, observationCategory, experiment.Name, candidate.Name, name)
		if len(results) != 1 || len(observations) != 1 || results[0].GetBool("actual") != actual || results[0].GetBool("outcome") != outcome ||
			observations[0].GetString("result") != results[0].Name || observations[0].GetBool("actual") != actual || observations[0].GetBool("outcome") != outcome ||
			!riEnvelopeValid(experiment, results[0], candidate.GetString("stage"), "result", "result:"+candidate.Name+":"+name) || !riEnvelopeValid(experiment, observations[0], candidate.GetString("stage"), "observation", "observation:"+candidate.Name+":"+name) {
			return false
		}
	}
	evidenceCategory := experiment.GetString("evidenceCategory")
	if evidenceCategory == "" {
		evidenceCategory = "RuleInductionEvidence"
	}
	evidence := riDirectArtifacts(vm.Store, evidenceCategory, experiment.Name, candidate.Name, "")
	return seen >= 4 && len(evidence) == 1 && riEnvelopeValid(experiment, evidence[0], candidate.GetString("stage"), "evidence", "evidence:"+candidate.Name) && candidate.GetInt("exampleCount") == seen && candidate.GetInt("supportCount") == support && candidate.GetInt("failureCount") == failures && candidate.GetInt("falsePositiveCount") == falsePositive && candidate.GetInt("falseNegativeCount") == falseNegative &&
		evidence[0].GetInt("exampleCount") == seen && evidence[0].GetInt("supportCount") == support && evidence[0].GetInt("failureCount") == failures
}

func riAllCandidateEvidenceValid(vm *VM, experiment *unit.Unit, requireProjection bool) bool {
	projectionCount := 0
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name || !directRIMember(u, experiment.GetString("candidateCategory")) {
			continue
		}
		if u.GetBool("projection") {
			projectionCount++
		}
		if _, ok := riDefinition(u.GetString("definitionCode")); !ok || u.GetString("stage") != "stage1" && u.GetString("stage") != "stage2" {
			return false
		}
		semantic := "candidate:" + u.GetString("definitionCode")
		if u.GetBool("projection") {
			semantic = "projection:" + u.GetString("definitionCode")
		}
		if !riEnvelopeValid(experiment, u, u.GetString("stage"), "candidate", semantic) {
			return false
		}
		if u.GetBool("riEvaluated") && !riCandidateEvidenceValid(vm, experiment, u) {
			return false
		}
	}
	wantProjection := 0
	if requireProjection && (experiment.GetString("reuseMode") == "shared-library" || experiment.GetString("reuseMode") == "shared-inlined") {
		wantProjection = 1
	}
	return projectionCount == wantProjection
}

func riLFFValid(vm *VM, experiment *unit.Unit) bool {
	constraintCategory, comparisonCategory, pruneCategory := experiment.GetString("constraintCategory"), experiment.GetString("comparisonCategory"), experiment.GetString("pruneCategory")
	var constraints, comparisons, prunes []*unit.Unit
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		switch {
		case directRIMember(u, constraintCategory):
			constraints = append(constraints, u)
		case directRIMember(u, comparisonCategory):
			comparisons = append(comparisons, u)
		case directRIMember(u, pruneCategory):
			prunes = append(prunes, u)
		}
	}
	if experiment.GetString("reuseMode") == "naive-direct" || experiment.GetString("reuseMode") == "uniform-random" {
		return len(constraints) == 0 && len(comparisons) == 0 && len(prunes) == 0
	}
	if len(prunes) != 0 {
		return false
	}
	for _, constraint := range constraints {
		code, stage, direction := constraint.GetString("failedCode"), constraint.GetString("stage"), constraint.GetString("direction")
		matches := riCandidateByCode(vm.Store, experiment.Name, stage, code)
		semantic := "constraint:" + direction + ":" + code
		if len(matches) != 1 || !riCandidateEvidenceValid(vm, experiment, matches[0]) || matches[0].GetInt("failureCount") == 0 || !riEnvelopeValid(experiment, constraint, stage, "constraint", semantic) ||
			(direction == "too-general" && matches[0].GetInt("falsePositiveCount") == 0) || (direction == "too-specific" && matches[0].GetInt("falseNegativeCount") == 0) || (direction != "too-general" && direction != "too-specific") {
			return false
		}
	}
	expected := map[string]bool{}
	for _, constraint := range constraints {
		failed := riCandidateByCode(vm.Store, experiment.Name, constraint.GetString("stage"), constraint.GetString("failedCode"))[0]
		for _, code := range experiment.GetStrings(map[string]string{"stage1": "stage1Queue", "stage2": "stage2Queue"}[constraint.GetString("stage")]) {
			matches := riCandidateByCode(vm.Store, experiment.Name, constraint.GetString("stage"), code)
			if len(matches) == 1 && matches[0].GetBool("riEvaluated") && matches[0].GetInt("queueRank") > failed.GetInt("queueRank") {
				expected[constraint.Name+"\x00"+matches[0].Name] = true
			}
		}
	}
	if len(comparisons) != len(expected) {
		return false
	}
	for _, comparison := range comparisons {
		constraint, candidate := vm.Store.Get(comparison.GetString("constraint")), vm.Store.Get(comparison.GetString("candidate"))
		if constraint == nil || candidate == nil || !expected[constraint.Name+"\x00"+candidate.Name] || comparison.GetString("stage") != candidate.GetString("stage") || !riEnvelopeValid(experiment, comparison, candidate.GetString("stage"), "comparison", "comparison:"+constraint.Name+":"+candidate.Name) {
			return false
		}
		general, specific := candidate.GetString("definitionCode"), constraint.GetString("failedCode")
		if constraint.GetString("direction") == "too-specific" {
			general, specific = constraint.GetString("failedCode"), candidate.GetString("definitionCode")
		}
		g, gok := riDefinition(general)
		s, sok := riDefinition(specific)
		subsumes, work, err := rivocab.StructurallySubsumes(g, s)
		if !gok || !sok || err != nil || comparison.GetBool("subsumes") != subsumes || comparison.GetInt("thetaWork") != work || subsumes {
			return false
		}
		delete(expected, constraint.Name+"\x00"+candidate.Name)
	}
	return len(expected) == 0
}

func riArtifactSetValid(vm *VM, experiment *unit.Unit) bool {
	if experiment.Name == "RuleInductionSeed" && experiment.GetString("experimentProfileKey") == "" {
		return true
	}
	p, ok := riProfile(vm, experiment.Name)
	if !ok {
		return false
	}
	categoryForKind := map[string]string{"partial": p.Categories.Partial, "refinement": p.Categories.Refinement, "candidate": p.Categories.Candidate, "result": p.Categories.Result, "observation": p.Categories.Observation, "evidence": p.Categories.Evidence, "constraint": p.Categories.Constraint, "comparison": p.Categories.Comparison, "prune": p.Categories.Prune, "library": p.Categories.Library, "provenance": p.Categories.Provenance, "projection": p.Categories.Projection, "transcript": p.Categories.Transcript, "boundary": p.Categories.Boundary, "corpus": p.Categories.Corpus, "selection": p.Categories.Selection, "terminal": p.Categories.Terminal}
	seen := map[string]bool{}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		kind := u.GetString("artifactKind")
		if kind == "" {
			if directRIMember(u, "RuleInductionExample") {
				continue
			}
			for _, category := range p.Categories.Ordered() {
				if directRIMember(u, category) {
					return false
				}
			}
			continue
		}
		if kind == "descriptor" {
			if name != experiment.Name || !directRIMember(u, "RuleInductionExperiment") || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || u.GetString("semanticKey") != rivocab.ArtifactSemanticKey("descriptor", "experiment-descriptor") || seen["descriptor"] {
				return false
			}
			seen["descriptor"] = true
			continue
		}
		if kind != "transcript" && !riAuthoritativeValid(u) {
			return false
		}
		category, known := categoryForKind[kind]
		stageIndex := u.GetInt("stageIndex")
		stageKey := experiment.GetString(map[int]string{1: "stage1ProfileKey", 2: "stage2ProfileKey"}[stageIndex])
		if !known || !directRIMember(u, category) || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || stageKey == "" || u.GetString("stageProfileKey") != stageKey || len(u.GetString("semanticKey")) != 71 {
			return false
		}
		identity := fmt.Sprintf("%d\x00%s\x00%s", stageIndex, kind, u.GetString("semanticKey"))
		if seen[identity] {
			return false
		}
		seen[identity] = true
	}
	return true
}

func riRefinementArtifactsValid(vm *VM, experiment *unit.Unit) bool {
	partialCategory, refinementCategory := experiment.GetString("partialCategory"), experiment.GetString("refinementCategory")
	expectedParent := map[string]string{riPartialText(rivocab.RootPartial()): ""}
	frontier := []rivocab.Partial{rivocab.RootPartial()}
	for len(frontier) > 0 {
		parent := frontier[0]
		frontier = frontier[1:]
		children, err := rivocab.RefineOne(parent)
		if err != nil {
			return false
		}
		for _, child := range children {
			childText := riPartialText(child)
			expectedParent[childText] = riPartialText(parent)
			if child.Bound() < len(child.Fields) {
				frontier = append(frontier, child)
			}
		}
	}
	partialsByStage := map[string]map[string]*unit.Unit{"stage1": {}, "stage2": {}}
	edgesByStage := map[string][]*unit.Unit{"stage1": {}, "stage2": {}}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		stage := u.GetString("stage")
		if directRIMember(u, partialCategory) {
			partialText := u.GetString("partial")
			if expectedParent[partialText] == "" && partialText != riPartialText(rivocab.RootPartial()) || partialsByStage[stage] == nil || partialsByStage[stage][partialText] != nil {
				return false
			}
			partialsByStage[stage][partialText] = u
		}
		if directRIMember(u, refinementCategory) {
			if edgesByStage[stage] == nil {
				return false
			}
			edgesByStage[stage] = append(edgesByStage[stage], u)
		}
	}
	if len(partialsByStage["stage1"]) != len(expectedParent) || len(partialsByStage["stage2"]) != 0 && len(partialsByStage["stage2"]) != len(expectedParent) {
		return false
	}
	for _, stage := range []string{"stage1", "stage2"} {
		partials := partialsByStage[stage]
		if len(partials) == 0 {
			continue
		}
		expectedEdges := map[string]bool{}
		for partialText, child := range partials {
			parentText := expectedParent[partialText]
			if parentText == "" {
				if child.GetString("refinedFrom") != "" || !riEnvelopeValid(experiment, child, stage, "partial", "root:"+stage) {
					return false
				}
				continue
			}
			parent := partials[parentText]
			if parent == nil || child.GetString("refinedFrom") != parent.Name || !riEnvelopeValid(experiment, child, stage, "partial", "partial:"+partialText) {
				return false
			}
			expectedEdges[parent.Name+"\x00"+child.Name] = true
		}
		if len(edgesByStage[stage]) != len(expectedEdges) {
			return false
		}
		for _, edge := range edgesByStage[stage] {
			key := edge.GetString("parent") + "\x00" + edge.GetString("child")
			if !expectedEdges[key] || !riEnvelopeValid(experiment, edge, stage, "refinement", "refinement:"+edge.GetString("parent")+":"+edge.GetString("child")) {
				return false
			}
			delete(expectedEdges, key)
		}
		if len(expectedEdges) != 0 {
			return false
		}
	}
	return true
}

func riLibraryArtifactsValid(vm *VM, experiment *unit.Unit) bool {
	libraryCategory := experiment.GetString("libraryCategory")
	provenanceCategory := experiment.GetString("provenanceCategory")
	projectionCategory := experiment.GetString("projectionCategory")
	var libraries, provenances, projections []*unit.Unit
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		if directRIMember(u, libraryCategory) {
			libraries = append(libraries, u)
		}
		if directRIMember(u, provenanceCategory) {
			provenances = append(provenances, u)
		}
		if directRIMember(u, projectionCategory) {
			projections = append(projections, u)
		}
	}
	wantLibraries := 0
	final := experiment.GetString("stage2ProfileKey") != ""
	switch experiment.GetString("reuseMode") {
	case "shared-library", "shared-inlined", "shared-recomputed":
		wantLibraries = 1
	case "lff-task-local-invention":
		wantLibraries = 1
		if final && experiment.GetString("terminal") == "identified" {
			wantLibraries = 2
		}
	}
	wantProjections := 0
	if experiment.GetString("reuseMode") == "lff-task-local-invention" {
		wantProjections = 1
		if final && experiment.GetString("terminal") == "identified" {
			wantProjections = 2
		}
	}
	if len(libraries) != wantLibraries || len(provenances) != wantLibraries || len(projections) != wantProjections {
		return false
	}
	background, ok := riBackground(ListVal(stringsToValues(experiment.GetStrings("facts"))))
	if !ok {
		return false
	}
	for _, library := range libraries {
		definition, ok := riDefinition(library.GetString("definitionCode"))
		if !ok {
			return false
		}
		relation, _, err := rivocab.Evaluate(definition, background)
		if err != nil || library.GetString("materializedRelation") != relation.Signature() {
			return false
		}
		stage := library.GetString("stage")
		semantic := "library:" + library.GetString("definitionCode")
		provenanceSemantic := "provenance:" + library.GetString("definitionCode")
		if library.GetBool("taskLocal") {
			semantic = "local-library:" + stage + ":" + library.GetString("definitionCode")
			provenanceSemantic = "local-provenance:" + stage + ":" + library.GetString("definitionCode")
		}
		if !riEnvelopeValid(experiment, library, stage, "library", semantic) {
			return false
		}
		matches := 0
		for _, provenance := range provenances {
			if provenance.GetString("library") == library.Name {
				matches++
				if provenance.GetString("candidate") != library.GetString("learnedFrom") || provenance.GetString("definitionCode") != library.GetString("definitionCode") || !riEnvelopeValid(experiment, provenance, stage, "provenance", provenanceSemantic) {
					return false
				}
			}
		}
		if matches != 1 {
			return false
		}
		if experiment.GetString("reuseMode") == "shared-recomputed" && !library.GetBool("discardedAtBoundary") {
			return false
		}
		if (experiment.GetString("reuseMode") == "shared-library" || experiment.GetString("reuseMode") == "shared-inlined") && (!library.GetBool("frozen") || library.Name != experiment.GetString("libraryUnit")) {
			return false
		}
	}
	seenProjection := map[string]bool{}
	for _, projection := range projections {
		stage := projection.GetString("stage")
		code := projection.GetString("definitionCode")
		library := vm.Store.Get(projection.GetString("library"))
		candidate := vm.Store.Get(projection.GetString("candidate"))
		definition, definitionOK := riDefinition(code)
		if stage != "stage1" && stage != "stage2" || seenProjection[stage] || library == nil || candidate == nil || !definitionOK {
			return false
		}
		relation, _, err := rivocab.Evaluate(definition, background)
		semantic := "local-projection:" + stage + ":" + code
		if err != nil || !projection.GetBool("taskLocal") || library.GetString("stage") != stage || !library.GetBool("taskLocal") || library.GetString("definitionCode") != code || candidate.GetString("stage") != stage || candidate.GetString("definitionCode") != code || projection.GetString("materializedRelation") != relation.Signature() || !riEnvelopeValid(experiment, projection, stage, "projection", semantic) {
			return false
		}
		seenProjection[stage] = true
	}
	return true
}

func riAuditedPrimaryWork(vm *VM, experiment *unit.Unit) int {
	return riAuditedPrimaryWorkAt(vm, experiment, 0)
}

func riAuditedWorkThrough(vm *VM, experiment *unit.Unit, chargeIndex int) int {
	return riAuditedPrimaryWorkAt(vm, experiment, chargeIndex)
}

func riAuditedPrimaryWorkAt(vm *VM, experiment *unit.Unit, maximumCharge int) int {
	if experiment.GetString("experimentProfileKey") == "" {
		return 0
	}
	charged := func(u *unit.Unit) bool {
		charge := u.GetInt("chargeIndex")
		return charge > 0 && (maximumCharge == 0 || charge <= maximumCharge)
	}
	total, artifacts, transcripts := 0, 0, 0
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || u.GetString("artifactKind") == "" || !charged(u) {
			continue
		}
		artifacts++
		switch u.GetString("artifactKind") {
		case "candidate":
			semanticCharge := u.GetInt("semanticChargeIndex")
			if u.GetBool("riEvaluated") && semanticCharge > 0 && (maximumCharge == 0 || semanticCharge <= maximumCharge) {
				total += u.GetInt("fixedWork") + u.GetInt("exampleCount") + 1
			}
		case "comparison":
			semanticCharge := u.GetInt("semanticChargeIndex")
			if semanticCharge > 0 && (maximumCharge == 0 || semanticCharge <= maximumCharge) {
				total += u.GetInt("thetaWork")
			}
		case "transcript":
			transcripts++
			switch u.GetString("action") {
			case "start", "refine", "constraint", "local-invention", "promotion", "termination", "fallback":
				total += u.GetInt("domainWork")
			}
		}
	}
	allocationProbes := 0
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") == experiment.Name && u.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && u.GetString("artifactKind") != "" && charged(u) {
			probes := u.GetInt("allocationProbes")
			if probes == 0 {
				probes = 1
			}
			allocationProbes += probes
		}
	}
	return total + allocationProbes + artifacts*32 + transcripts*16
}

func riBudgetUsageThrough(store *unit.Store, experiment *unit.Unit, maximumCharge int) (evaluations, fixedPoint, attributed int) {
	for _, name := range store.All() {
		u := store.Get(name)
		charge := u.GetInt("chargeIndex")
		if u.GetString("experiment") != experiment.Name || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || u.GetString("artifactKind") == "" || charge == 0 || charge > maximumCharge {
			continue
		}
		attributed++
		semanticCharge := u.GetInt("semanticChargeIndex")
		if u.GetString("artifactKind") == "candidate" && u.GetBool("riEvaluated") && semanticCharge > 0 && semanticCharge <= maximumCharge {
			evaluations++
			fixedPoint += u.GetInt("fixedWork")
		}
	}
	return
}

func stringsToValues(values []string) []Value {
	result := make([]Value, len(values))
	for index, value := range values {
		result[index] = StringVal(value)
	}
	return result
}

type riExampleBinding struct {
	ID      string
	Example rivocab.Example
}

func riCandidateExamples(vm *VM, experiment *unit.Unit, stage string) ([]riExampleBinding, bool) {
	if experiment.GetString("experimentProfileKey") != "" {
		corpusSlot := "stage1CorpusUnit"
		if stage == "stage2" {
			corpusSlot = "stage2CorpusUnit"
		}
		corpus := vm.Store.Get(experiment.GetString(corpusSlot))
		if corpus == nil {
			return nil, false
		}
		var result []riExampleBinding
		seen := map[string]bool{}
		for _, record := range corpus.GetStrings("examples") {
			example, ok := riExampleRecord(vm, record)
			if !ok || seen[record] {
				return nil, false
			}
			seen[record] = true
			result = append(result, riExampleBinding{ID: record, Example: example})
		}
		return result, len(result) >= 4 && len(result) <= 24
	}
	var records []string
	seen := map[string]bool{}
	var result []riExampleBinding
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if !directRIMember(u, "RuleInductionExample") || u.GetString("experiment") != experiment.Name || u.GetString("stage") != stage {
			continue
		}
		record := fmt.Sprintf("%d:%d:%t", u.GetInt("x"), u.GetInt("y"), u.GetBool("positive"))
		if seen[record] {
			return nil, false
		}
		seen[record] = true
		records = append(records, record)
		result = append(result, riExampleBinding{ID: name, Example: rivocab.Example{X: u.GetInt("x"), Y: u.GetInt("y"), Positive: u.GetBool("positive")}})
	}
	return result, len(records) >= 4 && len(records) <= 24
}

func riInputValid(vm *VM, experiment *unit.Unit, requireStage2 bool) bool {
	if experiment.Name == "RuleInductionSeed" && experiment.GetString("experimentProfileKey") == "" {
		return true
	}
	if _, ok := riProfile(vm, experiment.Name); !ok {
		return false
	}
	corpusCategory := experiment.GetString("corpusCategory")
	var stage1Corpus, stage2Corpus, boundary *unit.Unit
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if directRIMember(u, corpusCategory) && u.GetString("experiment") == experiment.Name {
			switch u.GetInt("stage") {
			case 1:
				if stage1Corpus != nil {
					return false
				}
				stage1Corpus = u
			case 2:
				if stage2Corpus != nil {
					return false
				}
				stage2Corpus = u
			default:
				return false
			}
		}
		if directRIMember(u, experiment.GetString("boundaryCategory")) && u.GetString("experiment") == experiment.Name {
			if boundary != nil {
				return false
			}
			boundary = u
		}
	}
	if stage1Corpus == nil {
		return false
	}
	stage1Examples := stage1Corpus.GetStrings("examples")
	_, stage1ExamplesOK := riCandidateExamples(vm, experiment, "stage1")
	if !stage1ExamplesOK || experiment.GetString("stage1CorpusUnit") != stage1Corpus.Name || !reflect.DeepEqual(stage1Corpus.GetStrings("facts"), experiment.GetStrings("facts")) || !riEnvelopeValid(experiment, stage1Corpus, "stage1", "corpus", "stage-1-corpus") {
		return false
	}
	stage1Profile := rivocab.StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: experiment.GetString("experimentProfileKey"), Stage: 1, FactDigest: rivocab.SemanticDigest(experiment.GetStrings("facts")), ExampleDigest: rivocab.SemanticDigest(stage1Examples)}
	stage1Key, err := stage1Profile.Key()
	if err != nil || stage1Key != experiment.GetString("stage1ProfileKey") || stage1Corpus.GetString("stageProfileKey") != stage1Key {
		return false
	}
	if !requireStage2 {
		return stage2Corpus == nil && boundary == nil
	}
	if stage2Corpus == nil {
		return false
	}
	stage2Examples := stage2Corpus.GetStrings("examples")
	_, stage2ExamplesOK := riCandidateExamples(vm, experiment, "stage2")
	if !stage2ExamplesOK || experiment.GetString("stage2CorpusUnit") != stage2Corpus.Name || boundary == nil || len(stage2Corpus.GetStrings("facts")) != 0 || stage2Corpus.GetString("boundary") != boundary.Name ||
		!riEnvelopeValid(experiment, stage2Corpus, "stage2", "corpus", "stage-2-corpus") || !riEnvelopeValid(experiment, boundary, "stage1", "boundary", "stage-1-boundary") || boundary.GetString("frozenCode") != experiment.GetString("frozenCode") || boundary.GetString("priorTerminal") != "awaiting-stage-2" || boundary.GetString("storeDigest") != riStageArtifactDigest(vm.Store, experiment.Name, 1) {
		return false
	}
	stage2Profile := rivocab.StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: experiment.GetString("experimentProfileKey"), Stage: 2, FactDigest: stage1Profile.FactDigest, ExampleDigest: rivocab.SemanticDigest(stage2Examples), PriorBoundaryDigest: boundary.GetString("storeDigest")}
	stage2Key, err := stage2Profile.Key()
	return err == nil && stage2Key == experiment.GetString("stage2ProfileKey") && stage2Corpus.GetString("stageProfileKey") == stage2Key
}

func bRIReadyToSelect(vm *VM) error {
	candidate := vm.Store.Get(vm.pop().AsString())
	if candidate == nil {
		vm.push(BoolVal(false))
		return nil
	}
	experiment := vm.Store.Get(candidate.GetString("experiment"))
	if experiment == nil || !riInputValid(vm, experiment, candidate.GetString("stage") == "stage2") || !riTranscriptValid(vm, experiment) || !riCandidateEvidenceValid(vm, experiment, candidate) || !riLFFValid(vm, experiment) || !riArtifactSetValid(vm, experiment) || !riRefinementArtifactsValid(vm, experiment) || !riBudgetsWithinCap(vm, experiment) || candidate.GetInt("failureCount") != 0 {
		vm.push(BoolVal(false))
		return nil
	}
	if candidate.GetBool("projection") {
		library := vm.Store.Get(experiment.GetString("libraryUnit"))
		category := experiment.GetString("libraryCategory")
		if category == "" {
			category = "RuleInductionLibrary"
		}
		vm.push(BoolVal(directRIMember(library, category) && library.GetBool("frozen") && library.GetString("definitionCode") == candidate.GetString("definitionCode") && library.GetString("materializedRelation") == candidate.GetString("signature")))
		return nil
	}
	queue := experiment.GetStrings("queue")
	if len(queue) != experiment.GetInt("candidateCap") {
		vm.push(BoolVal(false))
		return nil
	}
	stage, selectedCode := candidate.GetString("stage"), candidate.GetString("definitionCode")
	seen := map[string]bool{}
	for _, code := range queue {
		matches := riCandidateByCode(vm.Store, experiment.Name, stage, code)
		if len(matches) != 1 || seen[code] {
			vm.push(BoolVal(false))
			return nil
		}
		seen[code] = true
		if code == selectedCode {
			vm.push(BoolVal(matches[0].Name == candidate.Name))
			return nil
		}
		if !riCandidateEvidenceValid(vm, experiment, matches[0]) || matches[0].GetInt("failureCount") == 0 {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(false))
	return nil
}

func bRIStageOneComplete(vm *VM) error {
	experiment := vm.Store.Get(vm.pop().AsString())
	if experiment == nil || experiment.GetString("terminal") != "awaiting-stage-2" || experiment.GetString("stage2ProfileKey") != "" || experiment.GetString("frozenCode") == "" || !riInputValid(vm, experiment, false) || !riTranscriptValid(vm, experiment) || !riLFFValid(vm, experiment) || !riArtifactSetValid(vm, experiment) || !riRefinementArtifactsValid(vm, experiment) || !riAllCandidateEvidenceValid(vm, experiment, false) || !riLibraryArtifactsValid(vm, experiment) || !riDecisionArtifactsValid(vm, experiment) || !riBudgetsWithinCap(vm, experiment) {
		vm.push(BoolVal(false))
		return nil
	}
	queue := experiment.GetStrings("stage1Queue")
	seen := map[string]bool{}
	for _, code := range queue {
		matches := riCandidateByCode(vm.Store, experiment.Name, "stage1", code)
		if len(matches) != 1 || seen[code] {
			vm.push(BoolVal(false))
			return nil
		}
		seen[code] = true
		candidate := matches[0]
		if code == experiment.GetString("frozenCode") {
			vm.push(BoolVal(riCandidateEvidenceValid(vm, experiment, candidate) && candidate.GetInt("failureCount") == 0 && candidate.GetString("signature") == experiment.GetString("frozenSignature")))
			return nil
		}
		if !riCandidateEvidenceValid(vm, experiment, candidate) || candidate.GetInt("failureCount") == 0 {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(false))
	return nil
}

func riFallbackValid(vm *VM, experiment *unit.Unit) bool {
	shared := experiment.GetString("reuseMode") == "shared-library" || experiment.GetString("reuseMode") == "shared-inlined"
	var projection *unit.Unit
	projectionCount, fallbackActions := 0, 0
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		if directRIMember(u, experiment.GetString("candidateCategory")) && u.GetBool("projection") {
			projection, projectionCount = u, projectionCount+1
		}
		if u.GetString("artifactKind") == "transcript" && u.GetString("action") == "fallback" {
			fallbackActions++
		}
	}
	if !shared {
		return projectionCount == 0 && fallbackActions == 0 && !experiment.GetBool("fellBack") && !experiment.GetBool("usedFrozenLibrary")
	}
	if projectionCount != 1 || projection == nil || !riCandidateEvidenceValid(vm, experiment, projection) {
		return false
	}
	wantFallback := projection.GetInt("failureCount") > 0
	wantUse := !wantFallback
	wantFallbackActions := 0
	if wantFallback {
		wantFallbackActions = 1
	}
	if experiment.GetBool("fellBack") != wantFallback || experiment.GetBool("usedFrozenLibrary") != wantUse || fallbackActions != wantFallbackActions {
		return false
	}
	for _, name := range vm.Store.All() {
		u := vm.Store.Get(name)
		if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "transcript" && u.GetString("action") == "fallback" && u.GetString("artifactLink") != projection.Name {
			return false
		}
	}
	return true
}

func bRIExperimentComplete(vm *VM) error {
	experiment := vm.Store.Get(vm.pop().AsString())
	if experiment == nil {
		vm.push(BoolVal(false))
		return nil
	}
	if !riInputValid(vm, experiment, true) || !riTranscriptValid(vm, experiment) || !riLFFValid(vm, experiment) || !riArtifactSetValid(vm, experiment) || !riRefinementArtifactsValid(vm, experiment) || !riAllCandidateEvidenceValid(vm, experiment, true) || !riLibraryArtifactsValid(vm, experiment) || !riDecisionArtifactsValid(vm, experiment) || !riFallbackValid(vm, experiment) || !riBudgetsWithinCap(vm, experiment) {
		vm.push(BoolVal(false))
		return nil
	}
	if experiment.GetString("terminal") == "no-solution" {
		queue := experiment.GetStrings("queue")
		seen := map[string]bool{}
		ok := len(queue) == experiment.GetInt("candidateCap") && experiment.GetString("selectedCode") == ""
		for _, code := range queue {
			matches := riCandidateByCode(vm.Store, experiment.Name, "stage2", code)
			if len(matches) != 1 || seen[code] || !riCandidateEvidenceValid(vm, experiment, matches[0]) || matches[0].GetInt("failureCount") == 0 {
				ok = false
				break
			}
			seen[code] = true
		}
		vm.push(BoolVal(ok))
		return nil
	}
	ok := experiment.GetString("terminal") == "identified" && experiment.GetString("selectedCode") != ""
	if ok && experiment.GetBool("usedFrozenLibrary") {
		projection := vm.Store.Get(experiment.GetString("projectionUnit"))
		library := vm.Store.Get(experiment.GetString("libraryUnit"))
		provenance := vm.Store.Get(experiment.GetString("provenanceUnit"))
		libraryCategory, provenanceCategory := experiment.GetString("libraryCategory"), experiment.GetString("provenanceCategory")
		if libraryCategory == "" {
			libraryCategory = "RuleInductionLibrary"
		}
		if provenanceCategory == "" {
			provenanceCategory = "RuleInductionProvenance"
		}
		ok = projection != nil && projection.GetBool("projection") && riCandidateEvidenceValid(vm, experiment, projection) && projection.GetInt("failureCount") == 0 && projection.GetString("definitionCode") == experiment.GetString("selectedCode") &&
			directRIMember(library, libraryCategory) && library.GetBool("frozen") && library.GetString("definitionCode") == experiment.GetString("frozenCode") && library.GetString("materializedRelation") == experiment.GetString("frozenSignature") &&
			directRIMember(provenance, provenanceCategory) && provenance.GetString("library") == library.Name && provenance.GetString("definitionCode") == library.GetString("definitionCode")
	}
	if ok && !experiment.GetBool("usedFrozenLibrary") {
		matches := riCandidateByCode(vm.Store, experiment.Name, "stage2", experiment.GetString("selectedCode"))
		ok = len(matches) == 1 && riCandidateEvidenceValid(vm, experiment, matches[0]) && matches[0].GetInt("failureCount") == 0
	}
	vm.push(BoolVal(ok))
	return nil
}
