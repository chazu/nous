package nogoodexp

import (
	"fmt"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodoracle"
	"github.com/chazu/nous/internal/unit"
)

type CompetenceOutcome struct {
	Ordinal            int    `json:"ordinal"`
	Kind               string `json:"kind"`
	Disposition        string `json:"disposition"`
	OracleSolutions    int    `json:"oracle_solutions"`
	CorruptionRejected bool   `json:"corruption_rejected"`
}

type CompetenceExecution struct {
	Panel    string              `json:"panel"`
	Artifact string              `json:"artifact"`
	Outcomes []CompetenceOutcome `json:"outcomes"`
}

func runCompetence(domainsDir, panel string) (CompetenceExecution, error) {
	training, err := RunTraining(domainsDir)
	if err != nil {
		return CompetenceExecution{}, err
	}
	artifact, _, authority, err := FreezeArtifact(training)
	if err != nil {
		return CompetenceExecution{}, err
	}
	var cases []nogoodfixture.CompetenceCase
	if panel == "development" {
		cases, err = nogoodfixture.DevelopmentCompetence()
	} else if panel == "validation" {
		cases, err = validationCompetence()
	} else {
		return CompetenceExecution{}, fmt.Errorf("unknown competence panel %q", panel)
	}
	if err != nil {
		return CompetenceExecution{}, err
	}
	execution := CompetenceExecution{Panel: panel, Artifact: artifact.Digest}
	for _, competenceCase := range cases {
		bridge, bridgeErr := NewBridgeExecution(domainsDir, &artifact, &authority)
		if bridgeErr != nil {
			return CompetenceExecution{}, bridgeErr
		}
		disposition, considerErr := bridge.Consider(competenceCase.ProblemJSON, competenceCase.Decision)
		if considerErr != nil {
			return CompetenceExecution{}, fmt.Errorf("competence %d %s: %w", competenceCase.Ordinal, competenceCase.Kind, considerErr)
		}
		wantStatus := "resume"
		if competenceCase.WantProposal {
			wantStatus = "propose-prune"
		}
		if disposition.Status != wantStatus {
			return CompetenceExecution{}, fmt.Errorf("competence %d %s disposition %s, want %s", competenceCase.Ordinal, competenceCase.Kind, disposition.Status, wantStatus)
		}
		if _, meterErr := bridgeTranscript(uint32(competenceCase.Ordinal), disposition); meterErr != nil {
			return CompetenceExecution{}, fmt.Errorf("competence %d %s meter audit: %w", competenceCase.Ordinal, competenceCase.Kind, meterErr)
		}
		oracle, oracleErr := nogoodoracle.Enumerate(competenceCase.ProblemJSON, nogoodoracle.Literal(competenceCase.Decision))
		if oracleErr != nil {
			return CompetenceExecution{}, oracleErr
		}
		outcome := CompetenceOutcome{Ordinal: competenceCase.Ordinal, Kind: competenceCase.Kind, Disposition: disposition.Status, OracleSolutions: len(oracle.Solutions)}
		if competenceCase.Mutation != nogoodfixture.NoMutation {
			if err := requireCompetenceMutationRejection(disposition, competenceCase.Mutation); err != nil {
				return CompetenceExecution{}, fmt.Errorf("competence %d %s: %w", competenceCase.Ordinal, competenceCase.Kind, err)
			}
			outcome.CorruptionRejected = true
		}
		execution.Outcomes = append(execution.Outcomes, outcome)
	}
	return execution, nil
}

func requireCompetenceMutationRejection(source Disposition, mutation nogoodfixture.CompetenceMutation) error {
	mutations := []func(*Disposition){competenceMutation(mutation)}
	if mutation == nogoodfixture.DuplicateCompletion {
		mutations = append(mutations, deleteCompletion, corruptCompletion)
	}
	for _, mutate := range mutations {
		copyDisposition := source
		copyDisposition.Store = cloneStore(source.Store)
		mutate(&copyDisposition)
		if err := auditProposalOccurrence(copyDisposition); err == nil {
			return fmt.Errorf("mutation %q passed occurrence audit", mutation)
		}
	}
	return nil
}

func competenceMutation(mutation nogoodfixture.CompetenceMutation) func(*Disposition) {
	return func(disposition *Disposition) {
		switch mutation {
		case nogoodfixture.DuplicateCompletion:
			original := disposition.Store.Get(disposition.Completion)
			duplicate := unit.New(original.Name + ".duplicate")
			for slot, value := range original.Slots {
				duplicate.Set(slot, cloneSlotValue(value))
			}
			disposition.Store.Put(duplicate)
		case nogoodfixture.CrossDecision:
			disposition.Store.Get(disposition.Completion).Set("binding", disposition.Request)
		case nogoodfixture.StaleTarget:
			disposition.Store.Get(disposition.Request).Set("targetDigest", digestBytes([]byte("stale")))
		}
	}
}

func deleteCompletion(disposition *Disposition) { disposition.Store.Delete(disposition.Completion) }

func corruptCompletion(disposition *Disposition) {
	completion := disposition.Store.Get(disposition.Completion)
	completion.Set("conflict", !completion.GetBool("conflict"))
}
