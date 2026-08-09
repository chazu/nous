package transformexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/chazu/nous/internal/transformbaseline"
	"github.com/chazu/nous/internal/transformfixturecore"
	"github.com/chazu/nous/internal/unit"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
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
	Policy            Policy
	Terminal          string
	Schema            []byte
	Applications      int
	HeldoutCorrect    int
	HeldoutTotal      int
	FalseApplications int
	TrainingWork      int
	TasksPopped       int
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
	heldout, err := transformfixturecore.ParseHeldout(c.Heldout)
	if err != nil {
		return out, err
	}
	expected := expectedByToken(c)
	for _, test := range heldout.Cases {
		var application transformbaseline.Application
		if out.Policy == NousRefine || out.Policy == NoEqualityGuard {
			schema, parseErr := transformschema.ParseSchema(out.Schema)
			forest, forestErr := transformschema.ParseForest(test.Before)
			if parseErr != nil || forestErr != nil {
				return out, fmt.Errorf("production heldout decode")
			}
			result, applyErr := schema.Apply(forest)
			if applyErr != nil {
				return out, applyErr
			}
			application.Terminal = result.Terminal
			if result.Output != nil {
				application.Output, _ = result.Output.CanonicalJSON()
			}
		} else {
			application, err = transformbaseline.ApplySchema(test.Before, out.Schema)
			if err != nil {
				return out, err
			}
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
