package dsl

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ccRuleCount        = 40
	ccSeedCount        = 12
	ccCertificateCount = ccRuleCount * ccSeedCount
)

func init() {
	registerVocabularyWords("causalcurriculum", map[string]builtinFn{
		"cc-task-charge":                  bCCTaskCharge,
		"cc-initialize-charge":            bCCInitializeCharge,
		"cc-rule-root":                    bCCRuleRoot,
		"cc-refine-rule":                  bCCRefineRule,
		"cc-materialize-rule":             bCCMaterializeRule,
		"cc-admit":                        bCCAdmit,
		"cc-require-matrix":               bCCRequireMatrix,
		"cc-materialize-aggregate":        bCCMaterializeAggregate,
		"cc-require-aggregates":           bCCRequireAggregates,
		"cc-better?":                      bCCBetter,
		"cc-exact-tie?":                   bCCExactTie,
		"cc-materialize-tie":              bCCMaterializeTie,
		"cc-materialize-selection":        bCCMaterializeSelection,
		"cc-materialize-transcript-event": bCCMaterializeTranscriptEvent,
		"cc-require-terminal":             bCCRequireTerminal,
	})
}

type causalCurriculumMeter struct {
	mu     sync.Mutex
	counts causalv2.Counter
}

var causalCurriculumMeters = struct {
	sync.Mutex
	items map[string]*causalCurriculumMeter
}{items: make(map[string]*causalCurriculumMeter)}

// RegisterCausalCurriculumMeter creates the verifier-owned capability behind
// an opaque token. The DSL can charge it, but cannot inspect or replace it.
func RegisterCausalCurriculumMeter(token string) error {
	if token == "" {
		return errors.New("empty causal curriculum meter token")
	}
	causalCurriculumMeters.Lock()
	defer causalCurriculumMeters.Unlock()
	if _, exists := causalCurriculumMeters.items[token]; exists {
		return errors.New("duplicate causal curriculum meter token")
	}
	causalCurriculumMeters.items[token] = &causalCurriculumMeter{}
	return nil
}

func UnregisterCausalCurriculumMeter(token string) {
	causalCurriculumMeters.Lock()
	delete(causalCurriculumMeters.items, token)
	causalCurriculumMeters.Unlock()
}

func CausalCurriculumMeterSnapshot(token string) (causalv2.Counter, error) {
	causalCurriculumMeters.Lock()
	meter := causalCurriculumMeters.items[token]
	causalCurriculumMeters.Unlock()
	if meter == nil {
		return causalv2.Counter{}, errors.New("unknown causal curriculum meter capability")
	}
	meter.mu.Lock()
	defer meter.mu.Unlock()
	return meter.counts, nil
}

func ccRuntime(vm *VM, name string) (*unit.Unit, error) {
	runtime := vm.Store.Get(name)
	if runtime == nil || !vm.Store.IsA(name, "CausalCurriculumCursor") {
		return nil, fmt.Errorf("invalid causal curriculum cursor %q", name)
	}
	if failure := runtime.GetString("failure"); failure != "" {
		return nil, errors.New(failure)
	}
	return runtime, nil
}

func ccFail(runtime *unit.Unit, err error) error {
	if runtime != nil && runtime.GetString("failure") == "" {
		runtime.Set("failure", err.Error())
	}
	return err
}

func ccCounts(runtime *unit.Unit) (causalv2.Counter, error) {
	counts, err := CausalCurriculumMeterSnapshot(runtime.GetString("meterToken"))
	if err != nil {
		return counts, ccFail(runtime, err)
	}
	return counts, nil
}

func ccCharge(runtime *unit.Unit, field int, amount int64) error {
	if amount < 0 {
		return ccFail(runtime, errors.New("negative causal curriculum charge"))
	}
	causalCurriculumMeters.Lock()
	meter := causalCurriculumMeters.items[runtime.GetString("meterToken")]
	causalCurriculumMeters.Unlock()
	if meter == nil {
		return ccFail(runtime, errors.New("unknown causal curriculum meter capability"))
	}
	meter.mu.Lock()
	defer meter.mu.Unlock()
	values := meter.counts.Counts()
	values[field] += amount
	values[14] = 0
	for i := 0; i < 12; i++ {
		values[14] += values[i]
	}
	counter := causalv2.CounterFromCounts(values)
	if err := counter.Validate(); err != nil {
		return ccFail(runtime, err)
	}
	manifest := causalv2.PreregisteredManifest()
	if counter.TotalWork > int64(manifest.CurriculumSemanticWorkCap) || counter.AttributedUnits > int64(manifest.CurriculumAttributedUnitCap) || counter.EngineCycles > int64(manifest.CurriculumEngineCycleCap) {
		return ccFail(runtime, errors.New("causal curriculum cap exceeded"))
	}
	meter.counts = counter
	return nil
}

func bCCTaskCharge(vm *VM) error {
	runtime, err := ccRuntime(vm, vm.pop().AsString())
	if err != nil {
		return err
	}
	if err := ccCharge(runtime, 12, 1); err != nil {
		return err
	}
	vm.push(BoolVal(true))
	return nil
}

func bCCInitializeCharge(vm *VM) error {
	runtime, err := ccRuntime(vm, vm.pop().AsString())
	if err != nil {
		return err
	}
	if runtime.GetBool("initialChargesApplied") {
		return ccFail(runtime, errors.New("central initial charges already applied"))
	}
	seedUnits := append([]string{runtime.GetString("descriptorUnit")}, runtime.GetStrings("certificateUnits")...)
	if len(seedUnits) != 1+ccCertificateCount {
		return ccFail(runtime, errors.New("seeded central artifact matrix is incomplete"))
	}
	for _, name := range seedUnits {
		if vm.Store.Get(name) == nil {
			return ccFail(runtime, errors.New("seeded central artifact is absent"))
		}
		if err := ccCharge(runtime, 5, 1); err != nil {
			return err
		}
		if err := ccCharge(runtime, 13, 1); err != nil {
			return err
		}
	}
	var profile any
	if err := json.Unmarshal([]byte(runtime.GetString("centralProfileBytes")), &profile); err != nil {
		return ccFail(runtime, errors.New("central profile bytes are absent"))
	}
	var chargeProfileFields func(any) error
	chargeProfileFields = func(value any) error {
		switch typed := value.(type) {
		case map[string]any:
			for _, child := range typed {
				if err := chargeProfileFields(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := chargeProfileFields(child); err != nil {
					return err
				}
			}
		default:
			return ccCharge(runtime, 7, 1)
		}
		return nil
	}
	if err := chargeProfileFields(profile); err != nil {
		return err
	}
	runtime.Set("initialChargesApplied", true)
	vm.push(BoolVal(true))
	return nil
}

func bCCRuleRoot(vm *VM) error {
	vm.push(StringVal(""))
	return nil
}

// bCCRefineRule performs exactly one grammar refinement. It exposes only the
// atomic choices for the next dimension; CUE owns the traversal that enumerates
// complete rules.
func bCCRefineRule(vm *VM) error {
	partial := vm.pop().AsString()
	var next []string
	switch {
	case partial == "":
		for _, primary := range []string{"C", "E", "H", "R", "W"} {
			next = append(next, "P="+primary)
		}
	case strings.HasPrefix(partial, "P=") && !strings.Contains(partial, ";"):
		if len(partial) != 3 || !strings.Contains("CEHRW", partial[2:]) {
			return errors.New("invalid primary rule refinement")
		}
		for _, mode := range []string{"gain", "raw"} {
			next = append(next, partial+";M="+mode)
		}
	case strings.Count(partial, ";") == 1:
		parts := strings.Split(partial, ";")
		if len(parts) != 2 || len(parts[0]) != 3 || !strings.HasPrefix(parts[0], "P=") || !strings.Contains("CEHRW", parts[0][2:]) || (parts[1] != "M=gain" && parts[1] != "M=raw") {
			return errors.New("invalid mode rule refinement")
		}
		primary := parts[0][2:]
		for _, secondary := range []string{"C", "E", "H", "R", "W"} {
			if secondary != primary {
				next = append(next, partial+";S="+secondary)
			}
		}
	default:
		return errors.New("complete or invalid rule cannot be refined")
	}
	values := make([]Value, len(next))
	for i, refinement := range next {
		values[i] = StringVal(refinement)
	}
	vm.push(ListVal(values))
	return nil
}

func ccMaterialize(vm *VM, runtime *unit.Unit, kind, category string, payload any) (*unit.Unit, error) {
	chargeIndex := runtime.GetInt("nextChargeIndex")
	artifact, err := causalv2.NewArtifact(
		runtime.GetString("centralProfileDigest"), runtime.GetString("trainingKey"), 0,
		kind, payload, chargeIndex,
	)
	if err != nil {
		return nil, ccFail(runtime, err)
	}
	encoded, err := causalv2.CanonicalJSON(artifact)
	if err != nil {
		return nil, ccFail(runtime, err)
	}
	name := artifact.Name()
	if existing := vm.Store.Get(name); existing != nil {
		if existing.GetBool("sealed") && existing.GetString("artifactBytes") == string(encoded) {
			return existing, nil
		}
		return nil, ccFail(runtime, fmt.Errorf("occupied central artifact name %q", name))
	}
	u := unit.New(name)
	u.Set("isA", []string{category, "CausalCurriculumArtifact", "CausalArtifact", "Anything"})
	u.Set("sealed", true)
	u.Set("artifactBytes", string(encoded))
	u.Set("artifactDigest", artifact.ArtifactDigest)
	u.Set("semanticKey", artifact.SemanticKey)
	u.Set("kind", artifact.Kind)
	u.Set("scope", artifact.Scope)
	u.Set("step", artifact.Step)
	u.Set("chargeIndex", artifact.ChargeIndex)
	u.Set("runtime", runtime.Name)
	vm.Store.Put(u)
	runtime.Set("nextChargeIndex", chargeIndex+1)
	if err := ccCharge(runtime, 5, 1); err != nil {
		return nil, err
	}
	if err := ccCharge(runtime, 13, 1); err != nil {
		return nil, err
	}
	return u, nil
}

func bCCMaterializeRule(vm *VM) error {
	runtimeName := vm.pop().AsString()
	code := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	if runtime.GetString("phase") != "initializing" {
		return ccFail(runtime, errors.New("rule materialization outside initialization"))
	}
	if _, err := causal.ParseRule(code); err != nil {
		return ccFail(runtime, err)
	}
	u, err := ccMaterialize(vm, runtime, "central-rule", "CausalCentralRuleArtifact", causalv2.CentralRulePayload{RuleCode: code})
	if err != nil {
		return err
	}
	u.Set("ruleCode", code)
	vm.push(StringVal(u.Name))
	return nil
}

func ccArtifact(vm *VM, name, kind string) (causalv2.Artifact, *unit.Unit, error) {
	u := vm.Store.Get(name)
	if u == nil || !u.GetBool("sealed") {
		return causalv2.Artifact{}, u, fmt.Errorf("missing or unsealed artifact %q", name)
	}
	artifact, err := causalv2.VerifyArtifact([]byte(u.GetString("artifactBytes")))
	if err != nil {
		return artifact, u, err
	}
	if artifact.Name() != name || artifact.Kind != kind || artifact.ChargeIndex != u.GetInt("chargeIndex") {
		return artifact, u, errors.New("central artifact envelope/store mismatch")
	}
	return artifact, u, nil
}

func bCCAdmit(vm *VM) error {
	runtimeName := vm.pop().AsString()
	certificateUnitName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	if runtime.GetString("phase") != "admitting" {
		return ccFail(runtime, errors.New("certificate admission outside admitting phase"))
	}
	artifact, certificateUnit, err := ccArtifact(vm, certificateUnitName, "certificate")
	if err != nil {
		return ccFail(runtime, err)
	}
	index := certificateUnit.GetInt("matrixIndex")
	if index != runtime.GetInt("nextAdmission") || index < 0 || index >= ccCertificateCount {
		return ccFail(runtime, errors.New("certificate admission order mismatch"))
	}
	payload, err := causalv2.StrictDecode[causalv2.CertificatePayload](artifact.Payload)
	if err != nil {
		return ccFail(runtime, err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(payload.CertificateBytes)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != payload.CertificateBytes {
		return ccFail(runtime, errors.New("noncanonical certificate base64url"))
	}
	episodeBytes := []byte(certificateUnit.GetString("episodeBytes"))
	certificate, _, err := causalv2.VerifyApplicationCertificateForEpisode(decoded, episodeBytes)
	if err != nil {
		return ccFail(runtime, err)
	}
	manifest := causalv2.PreregisteredManifest()
	wantSeed := manifest.TrainingSeeds.Start + int64(index/ccRuleCount)*manifest.TrainingSeeds.Step
	ruleUnits := runtime.GetStrings("ruleUnits")
	if len(ruleUnits) != ccRuleCount {
		return ccFail(runtime, errors.New("curriculum rule grammar is incomplete"))
	}
	wantRuleUnit := vm.Store.Get(ruleUnits[index%ccRuleCount])
	if wantRuleUnit == nil {
		return ccFail(runtime, errors.New("curriculum rule grammar unit is absent"))
	}
	wantRule := wantRuleUnit.GetString("ruleCode")
	if certificate.Seed != wantSeed || certificate.RuleCode != wantRule || !certificate.AllCapsValid || certificate.OracleDisagreements != 0 {
		return ccFail(runtime, errors.New("certificate is outside the exact valid 40x12 matrix"))
	}
	key := fmt.Sprintf("%s\x00%d", certificate.RuleCode, certificate.Seed)
	for _, admitted := range runtime.GetStrings("admittedKeys") {
		if admitted == key {
			return ccFail(runtime, errors.New("duplicate curriculum rule/seed certificate"))
		}
	}
	certificateFields := []any{
		certificate.Seed, certificate.ProfileDigest, certificate.FixtureDigest, certificate.RuleCode,
		certificate.Score, certificate.Terminal, certificate.Cost, certificate.PosteriorDigest,
		certificate.TranscriptDigest, certificate.OracleAgreements, certificate.OracleDisagreements,
		certificate.AllCapsValid, certificate.EpisodeReportDigest, certificate.CertificateDigest,
	}
	for range certificateFields {
		if err := ccCharge(runtime, 7, 1); err != nil {
			return err
		}
	}
	applicationPayload := causalv2.ApplicationPayload{
		Seed: certificate.Seed, RuleCode: certificate.RuleCode, CertificateDigest: certificate.CertificateDigest,
		Score: certificate.Score, Terminal: certificate.Terminal, Cost: certificate.Cost,
	}
	application, err := ccMaterialize(vm, runtime, "application", "CausalCentralApplicationArtifact", applicationPayload)
	if err != nil {
		return err
	}
	application.Set("seed", int(certificate.Seed))
	application.Set("ruleCode", certificate.RuleCode)
	application.Set("certificateDigest", certificate.CertificateDigest)
	application.Set("certificateBytes", string(decoded))
	application.Set("score", certificate.Score)
	application.Set("terminal", certificate.Terminal)
	application.Set("cost", certificate.Cost)
	application.Set("sourceCertificateUnit", certificateUnitName)
	applications := append(runtime.GetStrings("applicationUnits"), application.Name)
	runtime.Set("applicationUnits", applications)
	if runtime.GetBool("creditEnabled") {
		if err := ccCharge(runtime, 3, 1); err != nil {
			return err
		}
		credit, err := ccMaterialize(vm, runtime, "credit", "CausalCentralCreditArtifact", causalv2.CreditPayload{
			ApplicationArtifactDigest: application.GetString("artifactDigest"), Delta: manifest.InvalidOrExhaustedScore - certificate.Score,
		})
		if err != nil {
			return err
		}
		credit.Set("applicationUnit", application.Name)
		credit.Set("ruleCode", certificate.RuleCode)
		credit.Set("seed", int(certificate.Seed))
		credit.Set("delta", manifest.InvalidOrExhaustedScore-certificate.Score)
		application.Set("creditUnit", credit.Name)
		runtime.Set("creditUnits", append(runtime.GetStrings("creditUnits"), credit.Name))
	}
	runtime.Set("admittedKeys", append(runtime.GetStrings("admittedKeys"), key))
	runtime.Set("nextAdmission", index+1)
	vm.push(BoolVal(true))
	return nil
}

func bCCRequireMatrix(vm *VM) error {
	runtime, err := ccRuntime(vm, vm.pop().AsString())
	if err != nil {
		return err
	}
	if runtime.GetInt("nextAdmission") != ccCertificateCount || len(runtime.GetStrings("admittedKeys")) != ccCertificateCount || len(runtime.GetStrings("applicationUnits")) != ccCertificateCount {
		return ccFail(runtime, errors.New("incomplete curriculum application matrix"))
	}
	wantCredits := 0
	if runtime.GetBool("creditEnabled") {
		wantCredits = ccCertificateCount
	}
	if len(runtime.GetStrings("creditUnits")) != wantCredits {
		return ccFail(runtime, errors.New("curriculum credit cardinality mismatch"))
	}
	vm.push(BoolVal(true))
	return nil
}

func bCCMaterializeAggregate(vm *VM) error {
	runtimeName := vm.pop().AsString()
	ruleUnitName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	if runtime.GetString("phase") != "aggregating" {
		return ccFail(runtime, errors.New("aggregate outside aggregating phase"))
	}
	_, ruleUnit, err := ccArtifact(vm, ruleUnitName, "central-rule")
	if err != nil {
		return ccFail(runtime, err)
	}
	code := ruleUnit.GetString("ruleCode")
	var certificates []causalv2.ApplicationCertificate
	aggregate := causalv2.RuleAggregatePayload{Code: code}
	for _, applicationName := range runtime.GetStrings("applicationUnits") {
		application := vm.Store.Get(applicationName)
		if application == nil || application.GetString("ruleCode") != code {
			continue
		}
		certificate, verifyErr := causalv2.VerifyApplicationCertificate([]byte(application.GetString("certificateBytes")))
		if verifyErr != nil {
			return ccFail(runtime, verifyErr)
		}
		certificates = append(certificates, certificate)
		if err := ccCharge(runtime, 3, 1); err != nil {
			return err
		}
		aggregate.Applications++
		aggregate.TotalScore += certificate.Score
		aggregate.TotalCost += certificate.Cost
		switch certificate.Terminal {
		case "identified":
			aggregate.Identified++
		case "equivalence":
			aggregate.Equivalence++
		case "budget-exhausted":
			aggregate.BudgetExhausted++
		}
		if runtime.GetBool("creditEnabled") {
			credit := vm.Store.Get(application.GetString("creditUnit"))
			if credit == nil {
				return ccFail(runtime, errors.New("application lacks credit artifact"))
			}
			aggregate.Worth += credit.GetInt("delta")
		}
	}
	if len(certificates) != ccSeedCount {
		return ccFail(runtime, fmt.Errorf("rule %s applications=%d, want %d", code, len(certificates), ccSeedCount))
	}
	aggregate.ApplicationDigest, err = causalv2.Digest(causalv2.RuleApplicationsDomain, certificates)
	if err != nil {
		return ccFail(runtime, err)
	}
	u, err := ccMaterialize(vm, runtime, "aggregate", "CausalCentralAggregateArtifact", aggregate)
	if err != nil {
		return err
	}
	u.Set("ruleCode", aggregate.Code)
	u.Set("applications", aggregate.Applications)
	u.Set("totalScore", aggregate.TotalScore)
	u.Set("totalCost", aggregate.TotalCost)
	u.Set("identified", aggregate.Identified)
	u.Set("equivalence", aggregate.Equivalence)
	u.Set("budgetExhausted", aggregate.BudgetExhausted)
	u.Set("worthScore", aggregate.Worth)
	u.Set("applicationDigest", aggregate.ApplicationDigest)
	runtime.Set("aggregateUnits", append(runtime.GetStrings("aggregateUnits"), u.Name))
	runtime.Set("nextAggregate", runtime.GetInt("nextAggregate")+1)
	vm.push(StringVal(u.Name))
	return nil
}

func bCCRequireAggregates(vm *VM) error {
	runtime, err := ccRuntime(vm, vm.pop().AsString())
	if err != nil {
		return err
	}
	if runtime.GetInt("nextAggregate") != ccRuleCount || len(runtime.GetStrings("aggregateUnits")) != ccRuleCount {
		return ccFail(runtime, errors.New("incomplete curriculum aggregate matrix"))
	}
	vm.push(BoolVal(true))
	return nil
}

func ccAggregateUnit(vm *VM, name string) (*unit.Unit, error) {
	aggregate := vm.Store.Get(name)
	if aggregate == nil || !vm.Store.IsA(name, "CausalCentralAggregateArtifact") {
		return nil, fmt.Errorf("invalid central aggregate %q", name)
	}
	return aggregate, nil
}

func bCCBetter(vm *VM) error {
	runtimeName := vm.pop().AsString()
	bestName := vm.pop().AsString()
	candidateName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	candidate, err := ccAggregateUnit(vm, candidateName)
	if err != nil {
		return ccFail(runtime, err)
	}
	best, err := ccAggregateUnit(vm, bestName)
	if err != nil {
		return ccFail(runtime, err)
	}
	if err := ccCharge(runtime, 3, 1); err != nil {
		return err
	}
	better := candidate.GetInt("worthScore") > best.GetInt("worthScore") ||
		(candidate.GetInt("worthScore") == best.GetInt("worthScore") &&
			(candidate.GetInt("budgetExhausted") < best.GetInt("budgetExhausted") ||
				(candidate.GetInt("budgetExhausted") == best.GetInt("budgetExhausted") && candidate.GetString("ruleCode") < best.GetString("ruleCode"))))
	vm.push(BoolVal(better))
	return nil
}

func bCCExactTie(vm *VM) error {
	runtimeName := vm.pop().AsString()
	bestName := vm.pop().AsString()
	candidateName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	candidate, err := ccAggregateUnit(vm, candidateName)
	if err != nil {
		return ccFail(runtime, err)
	}
	best, err := ccAggregateUnit(vm, bestName)
	if err != nil {
		return ccFail(runtime, err)
	}
	if err := ccCharge(runtime, 3, 1); err != nil {
		return err
	}
	vm.push(BoolVal(candidate.GetInt("worthScore") == best.GetInt("worthScore") && candidate.GetInt("budgetExhausted") == best.GetInt("budgetExhausted")))
	return nil
}

func bCCMaterializeTie(vm *VM) error {
	runtimeName := vm.pop().AsString()
	aggregateName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	_, aggregate, err := ccArtifact(vm, aggregateName, "aggregate")
	if err != nil {
		return ccFail(runtime, err)
	}
	u, err := ccMaterialize(vm, runtime, "central-tie", "CausalCentralTieArtifact", causalv2.CentralTiePayload{
		RuleCode: aggregate.GetString("ruleCode"), AggregateArtifactDigest: aggregate.GetString("artifactDigest"),
	})
	if err != nil {
		return err
	}
	u.Set("ruleCode", aggregate.GetString("ruleCode"))
	u.Set("aggregateUnit", aggregateName)
	runtime.Set("tieUnits", append(runtime.GetStrings("tieUnits"), u.Name))
	vm.push(StringVal(u.Name))
	return nil
}

func bCCMaterializeSelection(vm *VM) error {
	runtimeName := vm.pop().AsString()
	tieValues := vm.pop().AsList()
	selected := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	if _, err := causal.ParseRule(selected); err != nil {
		return ccFail(runtime, err)
	}
	tieDigests := make([]string, len(tieValues))
	for i, value := range tieValues {
		_, tie, verifyErr := ccArtifact(vm, value.AsString(), "central-tie")
		if verifyErr != nil {
			return ccFail(runtime, verifyErr)
		}
		tieDigests[i] = tie.GetString("artifactDigest")
	}
	if len(tieDigests) == 0 {
		return ccFail(runtime, errors.New("selection has no exact ties"))
	}
	u, err := ccMaterialize(vm, runtime, "central-selection", "CausalCentralSelectionArtifact", causalv2.CentralSelectionPayload{SelectedRule: selected, TieArtifactDigests: tieDigests})
	if err != nil {
		return err
	}
	u.Set("selectedRule", selected)
	u.Set("tieUnits", valueStrings(tieValues))
	runtime.Set("selectionUnit", u.Name)
	runtime.Set("selectedRule", selected)
	vm.push(StringVal(u.Name))
	return nil
}

func valueStrings(values []Value) []string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = value.AsString()
	}
	return items
}

func bCCMaterializeTranscriptEvent(vm *VM) error {
	runtimeName := vm.pop().AsString()
	subjectUnitName := vm.pop().AsString()
	runtime, err := ccRuntime(vm, runtimeName)
	if err != nil {
		return err
	}
	index := runtime.GetInt("transcriptIndex")
	subject := vm.Store.Get(subjectUnitName)
	if subject == nil || !subject.GetBool("sealed") {
		return ccFail(runtime, errors.New("central transcript subject is absent or unsealed"))
	}
	kind := "admission"
	if index >= ccCertificateCount && index < ccCertificateCount+ccRuleCount {
		kind = "aggregate"
	} else if index == ccCertificateCount+ccRuleCount {
		kind = "selection"
	} else if index < 0 || index > ccCertificateCount+ccRuleCount {
		return ccFail(runtime, errors.New("central transcript index out of range"))
	}
	wantSubjectKind := map[string]string{"admission": "credit", "aggregate": "aggregate", "selection": "central-selection"}[kind]
	if subject.GetString("kind") != wantSubjectKind {
		return ccFail(runtime, errors.New("central transcript subject kind mismatch"))
	}
	counts, err := ccCounts(runtime)
	if err != nil {
		return err
	}
	before := counts.TotalWork
	if err := ccCharge(runtime, 6, 16); err != nil {
		return err
	}
	afterCharge, err := ccCounts(runtime)
	if err != nil {
		return err
	}
	event := causalv2.CentralTranscriptEvent{
		EventVersion: "causal-central-transcript/v2", Index: index,
		PreviousDigest: runtime.GetString("lastTranscriptDigest"), Kind: kind,
		SubjectArtifactDigest: subject.GetString("artifactDigest"), WorkBefore: before,
		WorkAfter: afterCharge.TotalWork + 1,
	}
	if index == 0 {
		event.PreviousDigest = causalv2.ZeroDigest
	}
	event.EventDigest, err = causalv2.Digest(causalv2.CentralTranscriptEventDomain, event)
	if err != nil {
		return ccFail(runtime, err)
	}
	u, err := ccMaterialize(vm, runtime, "transcript", "CausalCentralTranscriptArtifact", event)
	if err != nil {
		return err
	}
	u.Set("eventIndex", index)
	u.Set("eventKind", kind)
	u.Set("eventDigest", event.EventDigest)
	u.Set("subjectUnit", subjectUnitName)
	runtime.Set("transcriptUnits", append(runtime.GetStrings("transcriptUnits"), u.Name))
	runtime.Set("lastTranscriptDigest", event.EventDigest)
	runtime.Set("transcriptIndex", index+1)
	vm.push(StringVal(u.Name))
	return nil
}

func bCCRequireTerminal(vm *VM) error {
	runtime, err := ccRuntime(vm, vm.pop().AsString())
	if err != nil {
		return err
	}
	if runtime.GetBool("creditEnabled") {
		if runtime.GetString("selectionUnit") == "" || runtime.GetString("selectedRule") == "" || runtime.GetInt("transcriptIndex") != 521 {
			return ccFail(runtime, errors.New("credited curriculum lacks selection or complete transcript"))
		}
	} else if len(runtime.GetStrings("creditUnits")) != 0 || runtime.GetString("selectionUnit") != "" || !runtime.GetBool("unresolved") || runtime.GetInt("transcriptIndex") != 0 {
		return ccFail(runtime, errors.New("no-credit curriculum did not remain unresolved"))
	}
	counts, err := ccCounts(runtime)
	if err != nil {
		return err
	}
	if counts.AttributedUnits > 2083 {
		return ccFail(runtime, errors.New("central conservative ledger exceeded 2083 artifacts"))
	}
	vm.push(BoolVal(true))
	return nil
}
