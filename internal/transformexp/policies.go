package transformexp

import (
	"bytes"
	"crypto/sha256"
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
	HeldoutCorrectBits    byte
	HeldoutTotal          int
	FalseApplications     int
	NonmatchingWork       int64
	TrainingWork          int
	TasksPopped           int
	Transcript            TransformTranscriptBundle
	HeldoutStoreUnchanged bool
	acquisition           *acquisitionRun
	baselineEvents        []transformbaseline.Event
	heldoutObservations   []heldoutObservation
	frozenReplayBatch     []byte
}

type heldoutObservation struct {
	Token, Terminal string
	Output          []byte
	Work            int64
}

func executePolicy(domainsDir string, c policyCurriculum, policy Policy) (PolicyOutcome, error) {
	out := PolicyOutcome{Policy: policy, HeldoutTotal: 8}
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
		run, err := runAcquisitionConfigured(domainsDir, c.Training, policyToken(c, policy), func(store *unit.Store) {
			store.Get("H-TransformAcquireConcretePrograms").Set("acquisitionOnly", true)
		})
		if err != nil {
			return out, err
		}
		out.TasksPopped, out.TrainingWork = run.TasksPopped, len(run.MeterRecords)
		batch, err := programBatch(run)
		if err != nil {
			return out, err
		}
		if policy == PositiveLGG {
			learned, events, err := transformbaseline.PositiveLGGMetered(c.Training, batch)
			if err != nil {
				return out, err
			}
			out.baselineEvents = append(baselineEventsFromTransformMeter(run.MeterRecords), events...)
			out.Terminal, out.Schema, out.Applications = learned.Terminal, learned.Schema, 4
			if out.Terminal == "completed" {
				out, err = scoreTrainingNegatives(c, out)
				if err != nil {
					return out, err
				}
			}
		} else {
			out.Terminal = "completed"
			out.baselineEvents = baselineEventsFromTransformMeter(run.MeterRecords)
			return validateReplayTraining(c, batch, out)
		}
	case BoundedPBE, RandomPBE:
		var learned transformbaseline.Result
		var events []transformbaseline.Event
		var err error
		if policy == BoundedPBE {
			learned, events, err = transformbaseline.BoundedPBEMetered(c.Training)
		} else {
			a, b := policySeed(c, policy)
			learned, events, err = transformbaseline.RandomPBEMetered(c.Training, a, b)
		}
		if err != nil {
			return out, err
		}
		out.Terminal, out.Schema, out.Applications = learned.Terminal, learned.Schema, learned.Applications
		out.baselineEvents = events
	default:
		return out, fmt.Errorf("unknown policy %q", policy)
	}
	if out.Terminal != "completed" || len(out.Schema) == 0 {
		if out.acquisition != nil {
			run := *out.acquisition
			out.acquisition = nil
			transcript, transcriptErr := transcriptFromAcquisition(run, c.Ordinal, policy, c.PolicyTokens[policy], policyManifestDigest(c, policy))
			if transcriptErr != nil {
				return out, transcriptErr
			}
			out.Transcript = transcript
			out.TrainingWork = int(out.Transcript.Work)
		} else if len(out.baselineEvents) != 0 {
			transcript, transcriptErr := transcriptFromBaselineEvents(out.baselineEvents, c, policy, out.Terminal, nil)
			if transcriptErr != nil {
				return out, transcriptErr
			}
			out.Transcript = transcript
			out.TrainingWork = int(transcript.Work)
			out.baselineEvents = nil
		}
		return out, nil
	}
	return out, nil
}

func executeHeldoutInputs(c policyCurriculum, heldoutBytes []byte, out PolicyOutcome) (PolicyOutcome, error) {
	if out.Terminal != "completed" {
		return out, nil
	}
	if out.Policy == ConcreteReplay {
		return scoreReplayHeldout(c, heldoutBytes, out)
	}
	if len(out.Schema) == 0 {
		return out, nil
	}
	return scoreSchema(c, heldoutBytes, out)
}

func scoreSchema(c policyCurriculum, heldoutBytes []byte, out PolicyOutcome) (PolicyOutcome, error) {
	if out.Policy == NousRefine || out.Policy == NoEqualityGuard {
		return scoreProductionSchema(c, heldoutBytes, out)
	}
	heldout, err := transformfixturecore.ParseHeldout(heldoutBytes)
	if err != nil {
		return out, err
	}
	for _, test := range heldout.Cases {
		beforeEventCount := len(out.baselineEvents)
		application, events, applyErr := transformbaseline.ApplySchemaMetered(test.Before, out.Schema, "heldout")
		err = applyErr
		if err != nil {
			return out, err
		}
		out.baselineEvents = append(out.baselineEvents, events...)
		out.Applications++
		out.heldoutObservations = append(out.heldoutObservations, heldoutObservation{test.Token, application.Terminal, bytes.Clone(application.Output), baselineEventWork(out.baselineEvents[beforeEventCount:])})
	}
	if out.Policy == PositiveLGG || out.Policy == BoundedPBE || out.Policy == RandomPBE {
		out.Transcript, err = transcriptFromBaselineEvents(out.baselineEvents, c, out.Policy, out.Terminal, out.Schema)
		if err != nil {
			return out, err
		}
		out.TrainingWork = int(out.Transcript.Work)
		out.baselineEvents = nil
	}
	return out, nil
}

func scoreProductionSchema(c policyCurriculum, heldoutBytes []byte, out PolicyOutcome) (PolicyOutcome, error) {
	if out.acquisition == nil {
		return out, errors.New("production heldout is missing frozen acquisition")
	}
	run := *out.acquisition
	out.acquisition = nil
	if run.Terminal != "completed" || run.Artifact == "" || !bytes.Equal([]byte(run.Store.Get(run.Artifact).GetString("schema")), out.Schema) {
		return out, errors.New("production heldout acquisition did not reproduce frozen schema")
	}
	heldout, err := transformfixturecore.ParseHeldout(heldoutBytes)
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
	for _, test := range heldout.Cases {
		beforeRecords, snapshotErr := dsl.TransformMeterSnapshot(meterToken)
		if snapshotErr != nil {
			return out, snapshotErr
		}
		terminal, output, executeErr := dsl.ExecuteTransformSchemaApplication(vm, test.Before, out.Schema)
		if executeErr != nil {
			return out, fmt.Errorf("heldout adapter execution: %v", executeErr)
		}
		application := transformbaseline.Application{Terminal: terminal, Output: output}
		out.Applications++
		afterRecords, snapshotErr := dsl.TransformMeterSnapshot(meterToken)
		if snapshotErr != nil {
			return out, snapshotErr
		}
		beforeWork, _, workErr := transformMeterWork(beforeRecords)
		if workErr != nil {
			return out, workErr
		}
		afterWork, _, workErr := transformMeterWork(afterRecords)
		if workErr != nil {
			return out, workErr
		}
		out.heldoutObservations = append(out.heldoutObservations, heldoutObservation{test.Token, application.Terminal, bytes.Clone(application.Output), afterWork - beforeWork})
	}
	records, err := dsl.TransformMeterSnapshot(meterToken)
	if err != nil {
		return out, err
	}
	run.MeterRecords = records
	manifest := policyManifestDigest(c, out.Policy)
	out.Transcript, err = transcriptFromAcquisition(run, c.Ordinal, out.Policy, c.PolicyTokens[out.Policy], manifest)
	experimentUnit.Set("meterToken", oldMeter)
	storeAfter, storeErr := run.Store.CanonicalJSON()
	if storeErr != nil {
		return out, storeErr
	}
	out.HeldoutStoreUnchanged = bytes.Equal(storeBefore, storeAfter)
	out.TrainingWork = int(out.Transcript.Work)
	return out, err
}

func validateReplayTraining(c policyCurriculum, batch []byte, out PolicyOutcome) (PolicyOutcome, error) {
	training, err := transformfixturecore.ParseTraining(c.Training)
	if err != nil {
		return out, err
	}
	for _, test := range training.Cases {
		if test.Kind != "abstain" {
			continue
		}
		application, events, replayErr := transformbaseline.ReplayMetered(batch, test.Token, test.Before, "training-validate")
		if replayErr != nil {
			return out, replayErr
		}
		out.Applications++
		out.baselineEvents = append(out.baselineEvents, events...)
		if len(application.Terminal) < 8 || application.Terminal[:8] != "abstain/" {
			out.FalseApplications++
		}
	}
	out.frozenReplayBatch = bytes.Clone(batch)
	return out, nil
}

func scoreReplayHeldout(c policyCurriculum, heldoutBytes []byte, out PolicyOutcome) (PolicyOutcome, error) {
	if len(out.frozenReplayBatch) == 0 {
		return out, errors.New("replay heldout is missing frozen program batch")
	}
	heldout, err := transformfixturecore.ParseHeldout(heldoutBytes)
	if err != nil {
		return out, err
	}
	for _, test := range heldout.Cases {
		beforeEventCount := len(out.baselineEvents)
		application, events, err := transformbaseline.ReplayMetered(out.frozenReplayBatch, test.Token, test.Before, "heldout")
		if err != nil {
			return out, err
		}
		out.Applications++
		out.baselineEvents = append(out.baselineEvents, events...)
		out.heldoutObservations = append(out.heldoutObservations, heldoutObservation{test.Token, application.Terminal, bytes.Clone(application.Output), baselineEventWork(out.baselineEvents[beforeEventCount:])})
	}
	out.Transcript, err = transcriptFromBaselineEvents(out.baselineEvents, c, out.Policy, out.Terminal, out.frozenReplayBatch)
	if err != nil {
		return out, err
	}
	out.TrainingWork = int(out.Transcript.Work)
	out.baselineEvents = nil
	out.frozenReplayBatch = nil
	return out, nil
}

func scoreTrainingNegatives(c policyCurriculum, out PolicyOutcome) (PolicyOutcome, error) {
	training, err := transformfixturecore.ParseTraining(c.Training)
	if err != nil {
		return out, err
	}
	for _, test := range training.Cases {
		if test.Kind != "abstain" {
			continue
		}
		application, events, err := transformbaseline.ApplySchemaMetered(test.Before, out.Schema, "training-validate")
		if err != nil {
			return out, err
		}
		out.Applications++
		out.baselineEvents = append(out.baselineEvents, events...)
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

func expectedByToken(c scorerCurriculum) map[string]expectedCase {
	out := make(map[string]expectedCase, len(c.Expected))
	for _, expected := range c.Expected {
		out[expected.Token] = expected
	}
	return out
}

func scorePolicyOutcome(scorer scorerCurriculum, out PolicyOutcome) (PolicyOutcome, error) {
	truth := expectedByToken(scorer)
	out.HeldoutCorrect = 0
	out.HeldoutCorrectBits = 0
	out.FalseApplications = 0
	out.NonmatchingWork = 0
	previous := ""
	for index, observation := range out.heldoutObservations {
		if observation.Token <= previous {
			return out, fmt.Errorf("heldout observations are not in opaque-token order")
		}
		expected, ok := truth[observation.Token]
		if !ok {
			return out, fmt.Errorf("heldout observation has no sealed expectation")
		}
		application := transformbaseline.Application{Terminal: observation.Terminal, Output: observation.Output}
		if correctApplication(application, expected) {
			out.HeldoutCorrect++
			out.HeldoutCorrectBits |= 1 << index
		}
		if expected.Terminal == "abstain" {
			out.NonmatchingWork += observation.Work
			if observation.Terminal == "applied" {
				out.FalseApplications++
			}
		}
		previous = observation.Token
	}
	if len(out.heldoutObservations) != 0 && len(out.heldoutObservations) != len(scorer.Expected) {
		return out, fmt.Errorf("heldout observation count mismatch")
	}
	out.heldoutObservations = nil
	return out, nil
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

func policyToken(c policyCurriculum, policy Policy) string {
	return c.PolicyTokens[policy]
}

func policySeed(c policyCurriculum, policy Policy) (uint64, uint64) {
	value := c.PolicyRandomness[policy]
	return value[0], value[1]
}

func policyManifestDigest(c policyCurriculum, policy Policy) string {
	return digestBytes(policyManifestBytes(c, policy))
}

func policyManifestBytes(c policyCurriculum, policy Policy) []byte {
	training := sha256.Sum256(c.Training)
	queueDigest := digestBytes(policyQueueBytesFromView(c))
	return mustJSON([]any{"transform-policy-manifest/v1", "transform-schema/v1", "transform-lifecycle-events/v1", c.PanelCommitment, policy, c.PolicyTokens[policy], hex.EncodeToString(training[:]), c.HeldoutDigest, queueDigest, []int{12000, 50000, 48, 2000, 20000}})
}
