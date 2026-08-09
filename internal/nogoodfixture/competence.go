package nogoodfixture

import (
	"math/rand/v2"

	"github.com/chazu/nous/internal/nogoodfixturecore"
)

type CompetenceMutation string

const (
	NoMutation          CompetenceMutation = ""
	DuplicateCompletion CompetenceMutation = "duplicate-completion"
	CrossDecision       CompetenceMutation = "cross-decision"
	StaleTarget         CompetenceMutation = "stale-target"
)

type CompetenceCase struct {
	Task
	Kind         string
	WantProposal bool
	Mutation     CompetenceMutation
}

// DevelopmentCompetence returns the public construction/certification panel.
// Protected validation construction is owned by the experiment guard.
func DevelopmentCompetence() ([]CompetenceCase, error) {
	cases := make([]CompetenceCase, 0, 8)
	for ordinal := 0; ordinal < 8; ordinal++ {
		seed := 831101 + ordinal
		constructed, err := nogoodfixturecore.ConstructCompetence(ordinal, func(purpose string) *rand.Rand {
			return stream("competence-development", seed, 0, purpose)
		})
		if err != nil {
			return nil, err
		}
		cohort := NearMiss
		if constructed.WantProposal {
			cohort = Reusable
		}
		cases = append(cases, CompetenceCase{
			Task: Task{
				Panel: "competence-development", Ordinal: ordinal, Seed: seed, Cohort: cohort,
				ProblemJSON: constructed.ProblemJSON, Decision: constructed.Decision,
			},
			Kind: constructed.Kind, WantProposal: constructed.WantProposal,
			Mutation: CompetenceMutation(constructed.Mutation),
		})
	}
	return cases, nil
}
