package nogoodexp

import (
	"math/rand/v2"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodfixturecore"
)

func validationCompetence() ([]nogoodfixture.CompetenceCase, error) {
	cases := make([]nogoodfixture.CompetenceCase, 0, 16)
	for ordinal := 0; ordinal < 16; ordinal++ {
		seed := 831201 + ordinal
		constructed, err := nogoodfixturecore.ConstructCompetence(ordinal, func(purpose string) *rand.Rand {
			return protectedFixtureStream("competence-validation", seed, 0, purpose)
		})
		if err != nil {
			return nil, err
		}
		cohort := nogoodfixture.NearMiss
		if constructed.WantProposal {
			cohort = nogoodfixture.Reusable
		}
		cases = append(cases, nogoodfixture.CompetenceCase{
			Task: nogoodfixture.Task{
				Panel: "competence-validation", Ordinal: ordinal, Seed: seed, Cohort: cohort,
				ProblemJSON: constructed.ProblemJSON, Decision: constructed.Decision,
			},
			Kind: constructed.Kind, WantProposal: constructed.WantProposal,
			Mutation: nogoodfixture.CompetenceMutation(constructed.Mutation),
		})
	}
	return cases, nil
}
