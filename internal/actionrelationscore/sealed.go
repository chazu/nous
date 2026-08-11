package actionrelationscore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/actionrelationcap"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationfixturecore"
)

const maximumSealedPanelBytes = int64(1024 * 1024 * 1024)

type sealedAttemptWire struct {
	Context           actionrelationfixture.DrawContext
	Training          []actionrelationfixturecore.Case
	TrainingAuthority actionrelationfixture.TrainingAuthority
	Curriculum        actionrelationfixture.Curriculum
	Truth             actionrelationfixture.CurriculumTruth
	Ledger            actionrelationfixture.AttemptLedger
	AttemptLedgers    []actionrelationfixture.AttemptLedger
	Fixture           actionrelationfixture.CurriculumFixture
}

type sealedPanelWire struct {
	Version   string
	Panel     string
	Authority string
	Attempts  []sealedAttemptWire
	Fixture   actionrelationfixture.PanelFixture
}

// SealedPanel is an opaque, supervisor-built fixture. Policy execution can
// reopen it without retaining a constructor token or locked seed root.
type SealedPanel struct {
	panel     string
	authority string
	attempts  []actionrelationfixture.GeneratedAttempt
	fixture   actionrelationfixture.PanelFixture
	canonical []byte
	digest    string
}

func PrepareDevelopmentPanel() (SealedPanel, error) {
	attempts, fixture, err := actionrelationfixture.GenerateDevelopmentPanel()
	if err != nil {
		return SealedPanel{}, err
	}
	return sealPanel(attempts, fixture)
}

// PrepareProtectedPanel is the sole scoring-layer caller of the protected
// fixture constructor. The returned value carries no constructor capability.
func PrepareProtectedPanel(token actionrelationcap.Token) (SealedPanel, error) {
	attempts, fixture, err := actionrelationfixture.GenerateProtectedPanel(token)
	if err != nil {
		return SealedPanel{}, err
	}
	return sealPanel(attempts, fixture)
}

func (s SealedPanel) Panel() string { return s.panel }

func (s SealedPanel) Authority() string { return s.authority }

func (s SealedPanel) Fixture() actionrelationfixture.PanelFixture { return s.fixture }

func (s SealedPanel) Canonical() []byte { return slices.Clone(s.canonical) }

func (s SealedPanel) Digest() string { return s.digest }

func sealPanel(attempts []actionrelationfixture.GeneratedAttempt, fixture actionrelationfixture.PanelFixture) (SealedPanel, error) {
	if err := actionrelationfixture.VerifyGeneratedPanel(attempts, fixture); err != nil {
		return SealedPanel{}, err
	}
	wire := sealedPanelWire{Version: "actionrelation-sealed-panel/v1", Panel: fixture.Panel, Authority: fixture.Authority, Fixture: fixture, Attempts: make([]sealedAttemptWire, len(attempts))}
	for index, attempt := range attempts {
		wire.Attempts[index] = sealedAttemptWire{
			Context: attempt.Context, Training: attempt.Training, TrainingAuthority: attempt.TrainingAuthority,
			Curriculum: attempt.Curriculum, Truth: attempt.Truth, Ledger: attempt.Ledger,
			AttemptLedgers: attempt.AttemptLedgers, Fixture: attempt.Fixture,
		}
	}
	canonical, err := json.Marshal(wire)
	if err != nil || int64(len(canonical)) > maximumSealedPanelBytes {
		return SealedPanel{}, fmt.Errorf("sealed panel exceeds temporary input cap")
	}
	return SealedPanel{panel: fixture.Panel, authority: fixture.Authority, attempts: attempts, fixture: fixture, canonical: canonical, digest: sealedDigest(canonical)}, nil
}

func ParseSealedPanel(reader io.Reader, size int64, digest string) (SealedPanel, error) {
	if size < 1 || size > maximumSealedPanelBytes || len(digest) != 64 {
		return SealedPanel{}, fmt.Errorf("invalid sealed panel envelope")
	}
	data, err := io.ReadAll(io.LimitReader(reader, size+1))
	if err != nil || int64(len(data)) != size || sealedDigest(data) != digest {
		return SealedPanel{}, fmt.Errorf("sealed panel bytes differ from supervisor authority")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire sealedPanelWire
	if err := decoder.Decode(&wire); err != nil || decoder.Decode(&struct{}{}) != io.EOF || wire.Version != "actionrelation-sealed-panel/v1" {
		return SealedPanel{}, fmt.Errorf("invalid sealed panel wire")
	}
	attempts := make([]actionrelationfixture.GeneratedAttempt, len(wire.Attempts))
	for index, encoded := range wire.Attempts {
		attempts[index] = actionrelationfixture.GeneratedAttempt{
			Context: encoded.Context, Training: encoded.Training, TrainingAuthority: encoded.TrainingAuthority,
			Curriculum: encoded.Curriculum, Truth: encoded.Truth, Ledger: encoded.Ledger,
			AttemptLedgers: encoded.AttemptLedgers, Fixture: encoded.Fixture,
		}
		normalizeAttemptSeeds(&attempts[index])
	}
	if wire.Panel != wire.Fixture.Panel || wire.Authority != wire.Fixture.Authority || actionrelationfixture.VerifyGeneratedPanel(attempts, wire.Fixture) != nil {
		return SealedPanel{}, fmt.Errorf("sealed panel authority does not reconstruct")
	}
	return SealedPanel{panel: wire.Panel, authority: wire.Authority, attempts: attempts, fixture: wire.Fixture, canonical: data, digest: digest}, nil
}

func normalizeAttemptSeeds(attempt *actionrelationfixture.GeneratedAttempt) {
	normalizeContextSeed(&attempt.Context)
	normalizeContextSeed(&attempt.Curriculum.Draws.Context)
	normalizeContextSeed(&attempt.Ledger.Context)
	normalizeContextSeed(&attempt.Ledger.Draws.Context)
	for index := range attempt.AttemptLedgers {
		normalizeContextSeed(&attempt.AttemptLedgers[index].Context)
		normalizeContextSeed(&attempt.AttemptLedgers[index].Draws.Context)
	}
}

func normalizeContextSeed(context *actionrelationfixture.DrawContext) {
	if value, ok := context.CurriculumSeed.(float64); ok && value >= 0 && value == float64(int(value)) {
		context.CurriculumSeed = int(value)
	}
}

func sealedDigest(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}
