package actionrelationfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

const (
	GeneratorAttemptWorkCap = 1_000_000
	GeneratorCurriculumCap  = 32_000_000
)

type GeneratorPhase struct {
	Name      string
	StartWork int
	EndWork   int
	Predicate string
	Status    string
}

type AttemptLedger struct {
	Context   DrawContext
	Draws     DrawBlock
	Phases    []GeneratorPhase
	TotalWork int
	Terminal  string
	Canonical []byte
	Digest    string
}

var generatorPhaseVocabulary = []struct {
	name      string
	predicate string
}{
	{"draw-precommit", "exact-66-draws"},
	{"family-universe", "complete-family-universe"},
	{"identifiability", "identifiable"},
	{"training-selection", "exact-training-cores-and-views"},
	{"skeleton-catalogs", "three-fixed-strata-catalogs"},
	{"utility-presentation", "six-fixed-strata-worlds"},
	{"truth-sealing", "complete-scorer-truth"},
	{"evidence-preflight", "complete-evidence-preflight"},
}

func SealAttemptLedger(context DrawContext, draws DrawBlock, phases []GeneratorPhase, terminal string) (AttemptLedger, error) {
	ledger := AttemptLedger{Context: context, Draws: draws, Phases: slices.Clone(phases), Terminal: terminal}
	if len(phases) > 0 {
		ledger.TotalWork = phases[len(phases)-1].EndWork
	}
	canonical, err := attemptLedgerWire(ledger)
	if err != nil {
		return AttemptLedger{}, err
	}
	ledger.Canonical = canonical
	ledger.Digest = digestBytes(canonical)
	if err := VerifyAttemptLedger(ledger); err != nil {
		return AttemptLedger{}, err
	}
	return ledger, nil
}

func VerifyAttemptLedger(ledger AttemptLedger) error {
	if err := validateDrawContext(ledger.Context); err != nil || !equalDrawContexts(ledger.Draws.Context, ledger.Context) {
		return fmt.Errorf("invalid attempt-ledger context")
	}
	wantDraws, err := precommitDraws(ledger.Context)
	if err != nil || !equalDrawBlocks(ledger.Draws, wantDraws) {
		return fmt.Errorf("attempt ledger changed frozen draws")
	}
	if len(ledger.Phases) < 1 || len(ledger.Phases) > len(generatorPhaseVocabulary) {
		return fmt.Errorf("invalid attempt-ledger phase count")
	}
	previousEnd := 0
	failed := false
	for index, phase := range ledger.Phases {
		vocabulary := generatorPhaseVocabulary[index]
		if phase.Name != vocabulary.name || phase.Predicate != vocabulary.predicate || phase.StartWork != previousEnd || phase.EndWork < phase.StartWork || phase.EndWork > GeneratorAttemptWorkCap {
			return fmt.Errorf("invalid attempt-ledger phase %d", index)
		}
		if index == 0 && (phase.StartWork != 0 || phase.EndWork != 66 || phase.Status != "passed") {
			return fmt.Errorf("invalid draw-precommit phase")
		}
		if phase.Status != "passed" && phase.Status != "failed" {
			return fmt.Errorf("invalid attempt-ledger phase status")
		}
		if index > 0 && phase.Status == "passed" && phase.EndWork == phase.StartWork {
			return fmt.Errorf("passed generator phase has no charged predicate")
		}
		if failed {
			return fmt.Errorf("attempt ledger performed work after failure")
		}
		failed = phase.Status == "failed"
		previousEnd = phase.EndWork
	}
	complete := len(ledger.Phases) == len(generatorPhaseVocabulary) && !failed
	if complete && ledger.Terminal != "accepted" || !complete && ledger.Terminal != "rejected" {
		return fmt.Errorf("attempt-ledger terminal does not match phases")
	}
	if ledger.TotalWork != previousEnd || ledger.TotalWork < 66 || ledger.TotalWork > GeneratorAttemptWorkCap {
		return fmt.Errorf("invalid attempt-ledger work total")
	}
	wantCanonical, err := attemptLedgerWire(ledger)
	if err != nil || !bytes.Equal(wantCanonical, ledger.Canonical) || ledger.Digest != digestBytes(ledger.Canonical) {
		return fmt.Errorf("attempt ledger changed canonical wire")
	}
	if len(ledger.Canonical) > 65536 {
		return fmt.Errorf("attempt ledger does not fit kind 36")
	}
	return nil
}

func attemptLedgerWire(ledger AttemptLedger) ([]byte, error) {
	drawRows := make([]any, len(ledger.Draws.Draws))
	for index, draw := range ledger.Draws.Draws {
		drawRows[index] = json.RawMessage(draw.Canonical)
	}
	phaseRows := make([]any, len(ledger.Phases))
	for index, phase := range ledger.Phases {
		phaseRows[index] = []any{phase.Name, phase.StartWork, phase.EndWork, phase.Predicate, phase.Status}
	}
	return json.Marshal([]any{
		"action-generator-attempt-ledger/v2", ledger.Context.Panel, ledger.Context.Authority,
		ledger.Context.Curriculum, ledger.Context.CurriculumSeed, ledger.Context.Attempt,
		drawRows, ledger.Draws.Root, phaseRows, ledger.TotalWork, ledger.Terminal,
	})
}

func equalDrawBlocks(got, want DrawBlock) bool {
	if !equalDrawContexts(got.Context, want.Context) || got.Root != want.Root || len(got.Draws) != 66 || len(want.Draws) != 66 {
		return false
	}
	for index := range want.Draws {
		a, b := got.Draws[index], want.Draws[index]
		if a.Ordinal != b.Ordinal || a.Namespace != b.Namespace || a.Index != b.Index || a.U64 != b.U64 || a.U64Hex != b.U64Hex || a.Digest != b.Digest || !bytes.Equal(a.Canonical, b.Canonical) {
			return false
		}
	}
	return true
}

func equalDrawContexts(a, b DrawContext) bool {
	if a.Panel != b.Panel || a.Authority != b.Authority || a.Curriculum != b.Curriculum || a.Attempt != b.Attempt {
		return false
	}
	switch seed := a.CurriculumSeed.(type) {
	case int:
		other, ok := b.CurriculumSeed.(int)
		return ok && seed == other
	case string:
		other, ok := b.CurriculumSeed.(string)
		return ok && seed == other
	default:
		return false
	}
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
