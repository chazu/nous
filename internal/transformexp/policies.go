package transformexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/transformbaseline"
	"github.com/chazu/nous/internal/transformfixturecore"
	"github.com/chazu/nous/internal/unit"
)

type Policy string

const (
	NousRefine      Policy = "nous-refine"
	PositiveLGG     Policy = "positive-lgg"
	BoundedPBE      Policy = "bounded-pbe"
	RandomPBE       Policy = "random-pbe"
	ConcreteReplay  Policy = "concrete-replay"
	NoEqualityGuard Policy = "no-equality-guard"
)

var empiricalPolicies = []Policy{NousRefine, PositiveLGG, BoundedPBE, RandomPBE, ConcreteReplay, NoEqualityGuard}

type PolicyOutcome struct {
	Policy                Policy
	Terminal              string
	Schema                []byte
	Applications          int
	HeldoutCorrect        int
	HeldoutTotal          int
	FalseApplications     int
	TrainingWork          int
	TasksPopped           int
	Transcript            TransformTranscriptBundle
	HeldoutStoreUnchanged bool
	acquisition           *acquisitionRun
}

func executePolicy(domainsDir string, c curriculum, policy Policy) (PolicyOutcome, error) {
	out := PolicyOutcome{Policy: policy, HeldoutTotal: len(c.Expected)}
	switch policy {
	case NousRefine, NoEqualityGuard:
		var configure func(*unit.Store)
		if policy == NoEqualityGuard {
			configure = func(store *unit.Store) { store.Get("H-TransformEvaluateFactor").Set("ablateEquality", true) }
		}
		run, err := runAcquisitionConfigured(domainsDir, c.Training, policyToken(c, policy), configure)
		if err != nil {
			return out, err
		}
		out.Terminal, out.TasksPopped, out.TrainingWork = run.Terminal, run.TasksPopped, len(run.MeterRecords)
		out.acquisition = &run
		if run.Artifact != "" {
			out.Schema = []byte(run.Store.Get(run.Artifact).GetString("schema"))
			out.Applications = 8
		}
	case PositiveLGG, ConcreteReplay:
		run, err := runAcquisition(domainsDir, c.Training, policyToken(c, policy))
		if err != nil {
			return out, err
		}
		out.TasksPopped, out.TrainingWork = run.TasksPopped, len(run.MeterRecords)
		batch, err := programBatch(run)
		if err != nil {
			return out, err
		}
		if policy == PositiveLGG {
			learned, err := transformbaseline.PositiveLGG(c.Training, batch)
			if err != nil {
				return out, err
			}
			out.Terminal, out.Schema, out.Applications = learned.Terminal, learned.Schema, 4
			if out.Terminal == "completed" {
				out, err = scoreTrainingNegatives(c, out)
				if err != nil {
					return out, err
				}
			}
		} else {
			out.Terminal = "completed"
			return scoreReplay(c, batch, out)
		}
	case BoundedPBE, RandomPBE:
		var learned transformbaseline.Result
		var err error
		if policy == BoundedPBE {
			learned, err = transformbaseline.BoundedPBE(c.Training)
		} else {
			a, b := policySeed(c, policy)
			learned, err = transformbaseline.RandomPBE(c.Training, a, b)
		}
		if err != nil {
			return out, err
		}
		out.Terminal, out.Schema, out.Applications = learned.Terminal, learned.Schema, learned.Applications
	default:
		return out, fmt.Errorf("unknown policy %q", policy)
	}
	if out.Terminal != "completed" || len(out.Schema) == 0 {
		return out, nil
	}
	return scoreSchema(c, out)
}

func scoreSchema(c curriculum, out PolicyOutcome) (PolicyOutcome, error) {
	if out.Policy == NousRefine || out.Policy == NoEqualityGuard {
		return scoreProductionSchema(c, out)
	}
	heldout, err := transformfixturecore.ParseHeldout(c.Heldout)
	if err != nil {
		return out, err
	}
	expected := expectedByToken(c)
	for _, test := range heldout.Cases {
		var application transformbaseline.Application
		application, err = transformbaseline.ApplySchema(test.Before, out.Schema)
		if err != nil {
			return out, err
		}
		out.Applications++
		truth := expected[test.Token]
		if correctApplication(application, truth) {
			out.HeldoutCorrect++
		}
		if truth.Terminal == "abstain" && application.Terminal == "applied" {
			out.FalseApplications++
		}
	}
	return out, nil
}

func scoreProductionSchema(c curriculum, out PolicyOutcome) (PolicyOutcome, error) {
	if out.acquisition == nil {
		return out, errors.New("production heldout is missing frozen acquisition")
	}
	run := *out.acquisition
	out.acquisition = nil
	if run.Terminal != "completed" || run.Artifact == "" || !bytes.Equal([]byte(run.Store.Get(run.Artifact).GetString("schema")), out.Schema) {
		return out, errors.New("production heldout acquisition did not reproduce frozen schema")
	}
	heldout, err := transformfixturecore.ParseHeldout(c.Heldout)
	if err != nil {
		return out, err
	}
	experiment := run.Store.Get(run.Root).GetString("experiment")
	experimentUnit := run.Store.Get(experiment)
	storeBefore, err := run.Store.CanonicalJSON()
	if err != nil {
		return out, err
	}
	oldMeter := experimentUnit.GetString("meterToken")
	meterToken := "tsm:heldout:" + policyToken(c, out.Policy)
	if err := dsl.RegisterTransformMeterWithRecords(meterToken, run.MeterRecords); err != nil {
		return out, err
	}
	defer dsl.UnregisterTransformMeter(meterToken)
	experimentUnit.Set("meterToken", meterToken)
	defer experimentUnit.Set("meterToken", oldMeter)
	vm := dsl.NewVM(run.Store, agenda.New(), nil)
	vm.CurrentTask = &agenda.Task{UnitName: experiment, SlotName: "tsHeldout"}
	expected := expectedByToken(c)
	for _, test := range heldout.Cases {
		terminal, output, executeErr := dsl.ExecuteTransformSchemaApplication(vm, test.Before, out.Schema)
		if executeErr != nil {
			return out, fmt.Errorf("heldout adapter execution: %v", executeErr)
		}
		application := transformbaseline.Application{Terminal: terminal, Output: output}
		out.Applications++
		truth := expected[test.Token]
		if truth.Terminal == "applied" && application.Terminal == "applied" {
			_, compareErr := dsl.CompareTransformOutputs(vm, application.Output, truth.Output)
			if compareErr != nil {
				return out, fmt.Errorf("heldout output comparison: %v", compareErr)
			}
		}
		if correctApplication(application, truth) {
			out.HeldoutCorrect++
		}
		if truth.Terminal == "abstain" && application.Terminal == "applied" {
			out.FalseApplications++
		}
	}
	records, err := dsl.TransformMeterSnapshot(meterToken)
	if err != nil {
		return out, err
	}
	run.MeterRecords = records
	manifest := policyManifestDigest(c, out.Policy)
	out.Transcript, err = transcriptFromAcquisition(run, c.Ordinal, out.Policy, caseToken(c.Seed, "policy-"+string(out.Policy), 0), manifest)
	experimentUnit.Set("meterToken", oldMeter)
	storeAfter, storeErr := run.Store.CanonicalJSON()
	if storeErr != nil {
		return out, storeErr
	}
	out.HeldoutStoreUnchanged = bytes.Equal(storeBefore, storeAfter)
	return out, err
}

func scoreReplay(c curriculum, batch []byte, out PolicyOutcome) (PolicyOutcome, error) {
	training, err := transformfixturecore.ParseTraining(c.Training)
	if err != nil {
		return out, err
	}
	for _, test := range training.Cases {
		if test.Kind != "abstain" {
			continue
		}
		application, replayErr := transformbaseline.Replay(batch, test.Token, test.Before)
		if replayErr != nil {
			return out, replayErr
		}
		out.Applications++
		if len(application.Terminal) < 8 || application.Terminal[:8] != "abstain/" {
			out.FalseApplications++
		}
	}
	heldout, err := transformfixturecore.ParseHeldout(c.Heldout)
	if err != nil {
		return out, err
	}
	expected := expectedByToken(c)
	for _, test := range heldout.Cases {
		application, err := transformbaseline.Replay(batch, test.Token, test.Before)
		if err != nil {
			return out, err
		}
		out.Applications++
		if correctApplication(application, expected[test.Token]) {
			out.HeldoutCorrect++
		}
	}
	return out, nil
}

func scoreTrainingNegatives(c curriculum, out PolicyOutcome) (PolicyOutcome, error) {
	training, err := transformfixturecore.ParseTraining(c.Training)
	if err != nil {
		return out, err
	}
	for _, test := range training.Cases {
		if test.Kind != "abstain" {
			continue
		}
		application, err := transformbaseline.ApplySchema(test.Before, out.Schema)
		if err != nil {
			return out, err
		}
		out.Applications++
		if application.Terminal == "applied" {
			out.FalseApplications++
		}
	}
	return out, nil
}

func correctApplication(application transformbaseline.Application, truth expectedCase) bool {
	if truth.Terminal == "applied" {
		return application.Terminal == "applied" && bytes.Equal(application.Output, truth.Output)
	}
	return len(application.Terminal) >= 8 && application.Terminal[:8] == "abstain/"
}

func expectedByToken(c curriculum) map[string]expectedCase {
	out := make(map[string]expectedCase, len(c.Expected))
	for _, expected := range c.Expected {
		out[expected.Token] = expected
	}
	return out
}

func programBatch(run acquisitionRun) ([]byte, error) {
	batch := transformfixturecore.ProgramBatch{}
	for _, name := range run.Programs {
		program := run.Store.Get(name)
		example := run.Store.Get(program.GetString("example"))
		before := []byte(example.GetString("before"))
		digest := sha256.Sum256(before)
		batch.Rows = append(batch.Rows, transformfixturecore.ProgramRow{
			Token:        example.GetString("token"),
			BeforeDigest: hex.EncodeToString(digest[:]),
			Program:      []byte(program.GetString("program")),
		})
	}
	return batch.CanonicalJSON()
}

func policyToken(c curriculum, policy Policy) string {
	return fmt.Sprintf("%s-%03d", policy, c.Ordinal)
}

func policySeed(c curriculum, policy Policy) (uint64, uint64) {
	d := sha256.Sum256(mustJSON([]any{"part3/transform-schema/v1", "random-policy", c.Seed, policy}))
	return binary.BigEndian.Uint64(d[:8]), binary.BigEndian.Uint64(d[8:16])
}

func policyManifestDigest(c curriculum, policy Policy) string {
	training := sha256.Sum256(c.Training)
	heldout := sha256.Sum256(c.Heldout)
	preimage := mustJSON([]any{"transform-policy-manifest/v1", "transform-schema/v1", "transform-lifecycle-events/v1", "safe", policy, caseToken(c.Seed, "policy-"+string(policy), 0), hex.EncodeToString(training[:]), hex.EncodeToString(heldout[:]), "", []int{12000, 50000, 48, 2000, 20000}})
	return digestBytes(preimage)
}
