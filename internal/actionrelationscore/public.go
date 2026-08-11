package actionrelationscore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationfixturecore"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type PublicWorld struct {
	State   actionrelations.State
	Actions []actionrelations.Action
}

type PublicCurriculum struct {
	Curriculum int
	Training   []actionrelationfixturecore.PublicCase
	Worlds     []PublicWorld
}

type publicPanelWire struct {
	Version       string
	Panel         string
	Authority     string
	FixtureDigest string
	Curricula     []PublicCurriculum
}

// PublicPanel is the only descriptor passed to a policy process. Its type has
// no fields for scorer truth, relation labels, latent family/guard, generator
// ledgers, accepted-attempt metadata, strata, or seed authority.
type PublicPanel struct {
	panel         string
	authority     string
	fixtureDigest string
	curricula     []PublicCurriculum
	canonical     []byte
	digest        string
}

func buildPublicPanel(attempts []actionrelationfixture.GeneratedAttempt, fixture actionrelationfixture.PanelFixture) (PublicPanel, error) {
	if actionrelationfixture.VerifyGeneratedPanel(attempts, fixture) != nil {
		return PublicPanel{}, fmt.Errorf("invalid private panel authority")
	}
	curricula := make([]PublicCurriculum, len(attempts))
	for index, attempt := range attempts {
		training, err := actionrelationfixturecore.PublicCases(attempt.Training)
		if err != nil {
			return PublicPanel{}, err
		}
		worlds := make([]PublicWorld, len(attempt.Curriculum.Worlds))
		for world, view := range attempt.Curriculum.Worlds {
			worlds[world] = PublicWorld{State: view.State, Actions: slices.Clone(view.Actions)}
		}
		curricula[index] = PublicCurriculum{Curriculum: index, Training: training, Worlds: worlds}
	}
	return sealPublicPanel(publicPanelWire{Version: "actionrelation-public-policy-panel/v1", Panel: fixture.Panel, Authority: fixture.Authority, FixtureDigest: fixture.Digest, Curricula: curricula})
}

func sealPublicPanel(wire publicPanelWire) (PublicPanel, error) {
	if err := verifyPublicPanelWire(wire); err != nil {
		return PublicPanel{}, err
	}
	canonical, err := json.Marshal(wire)
	if err != nil || int64(len(canonical)) > maximumSealedPanelBytes {
		return PublicPanel{}, fmt.Errorf("public panel exceeds input cap")
	}
	return PublicPanel{panel: wire.Panel, authority: wire.Authority, fixtureDigest: wire.FixtureDigest, curricula: wire.Curricula, canonical: canonical, digest: sealedDigest(canonical)}, nil
}

func ParsePublicPanel(reader io.Reader, size int64, digest string) (PublicPanel, error) {
	if size < 1 || size > maximumSealedPanelBytes || len(digest) != 64 {
		return PublicPanel{}, fmt.Errorf("invalid public panel envelope")
	}
	data, err := io.ReadAll(io.LimitReader(reader, size+1))
	if err != nil || int64(len(data)) != size || sealedDigest(data) != digest {
		return PublicPanel{}, fmt.Errorf("public panel bytes differ from supervisor authority")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire publicPanelWire
	if err := decoder.Decode(&wire); err != nil || decoder.Decode(&struct{}{}) != io.EOF || verifyPublicPanelWire(wire) != nil {
		return PublicPanel{}, fmt.Errorf("invalid public panel wire")
	}
	want, _ := json.Marshal(wire)
	if !bytes.Equal(want, data) {
		return PublicPanel{}, fmt.Errorf("noncanonical public panel wire")
	}
	return PublicPanel{panel: wire.Panel, authority: wire.Authority, fixtureDigest: wire.FixtureDigest, curricula: wire.Curricula, canonical: data, digest: digest}, nil
}

func verifyPublicPanelWire(wire publicPanelWire) error {
	want := map[string]int{"development": 16, "validation": 24, "locked": 32}[wire.Panel]
	if wire.Version != "actionrelation-public-policy-panel/v1" || want == 0 || len(wire.Curricula) != want || !digestText(wire.FixtureDigest) || wire.Authority == "" {
		return fmt.Errorf("invalid public panel authority")
	}
	for index, curriculum := range wire.Curricula {
		if curriculum.Curriculum != index || len(curriculum.Training) != actionrelationfixturecore.TrainingCount || len(curriculum.Worlds) != 6 {
			return fmt.Errorf("invalid public curriculum %d", index)
		}
		for ordinal, testCase := range curriculum.Training {
			if testCase.Ordinal != ordinal {
				return fmt.Errorf("invalid public training ordinal")
			}
			if _, err := actionrelations.ParseState(testCase.State); err != nil {
				return err
			}
			a, aErr := actionrelations.ParseOccurrence(testCase.AOccurrence)
			b, bErr := actionrelations.ParseOccurrence(testCase.BOccurrence)
			if aErr != nil || bErr != nil || a == b {
				return fmt.Errorf("invalid public training pair")
			}
		}
		for _, public := range curriculum.Worlds {
			if _, err := (actionrelations.World{State: public.State, Actions: public.Actions}).Normalize(); err != nil {
				return fmt.Errorf("invalid public utility world: %w", err)
			}
		}
	}
	return nil
}

func (p PublicPanel) Panel() string                 { return p.panel }
func (p PublicPanel) Authority() string             { return p.authority }
func (p PublicPanel) FixtureDigest() string         { return p.fixtureDigest }
func (p PublicPanel) Curricula() []PublicCurriculum { return slices.Clone(p.curricula) }
func (p PublicPanel) Canonical() []byte             { return slices.Clone(p.canonical) }
func (p PublicPanel) Digest() string                { return p.digest }
