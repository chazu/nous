package actionrelationscore

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationsearch"
)

// DerivePanelMechanicalGates reconstructs the six scorer-owned gates from the
// canonical leaf rows. AuthorityClosure and PrimaryAuditEqual are deliberately
// left false: the supervisor owns those checks because they depend on retained
// repository files and two isolated executions.
func DerivePanelMechanicalGates(value PanelSummary) (MechanicalGates, error) {
	if err := VerifyPanelSummary(value); err != nil {
		return MechanicalGates{}, err
	}
	wantCurricula := map[string]int{"development": 16, "validation": 24, "locked": 32}[value.Panel]
	worlds := make(map[[3]int]WorldPolicyRow, len(value.WorldRows))
	for _, row := range value.WorldRows {
		key := [3]int{row.Curriculum, row.WorldOrdinal, policyIndex(row.Policy)}
		if row.Panel != value.Panel || row.Curriculum >= wantCurricula || row.Family != row.Curriculum%8 || worlds[key].Digest != "" {
			return MechanicalGates{}, fmt.Errorf("duplicate or out-of-authority world row")
		}
		worlds[key] = row
	}

	gates := MechanicalGates{
		SemanticAgreement:      true,
		WorkConservation:       true,
		ArtifactsImmutable:     true,
		NousZeroFalseMatches:   true,
		RequiredBehaviorEqual:  true,
		FreshCertificatesValid: true,
	}
	byCurriculum := make([]map[actionrelationsearch.Policy]CurriculumPolicyRow, wantCurricula)
	for index := range byCurriculum {
		byCurriculum[index] = map[actionrelationsearch.Policy]CurriculumPolicyRow{}
	}
	for _, row := range value.CurriculumRows {
		if row.Panel != value.Panel || row.Curriculum >= wantCurricula || row.Family != row.Curriculum%8 || byCurriculum[row.Curriculum][row.Policy].Digest != "" {
			return MechanicalGates{}, fmt.Errorf("duplicate or out-of-authority curriculum row")
		}
		byCurriculum[row.Curriculum][row.Policy] = row
	}

	required := map[actionrelationsearch.Policy]bool{
		actionrelationsearch.Complete:     true,
		actionrelationsearch.StaticSleep:  true,
		actionrelationsearch.DynamicSleep: true,
		actionrelationsearch.NousSleep:    true,
		actionrelationsearch.NoGuardSleep: true,
		actionrelationsearch.LearnedNoUse: true,
	}
	for curriculum, policies := range byCurriculum {
		if len(policies) != len(Policies) {
			return MechanicalGates{}, fmt.Errorf("curriculum %d lacks policy rows", curriculum)
		}
		for policyOrdinal, policy := range Policies {
			row := policies[policy]
			var search [12]int
			behavior := true
			aggregate := row.AcquisitionTerminal
			if aggregate == "not-applicable" || aggregate == "completed" || aggregate == "no-discovery" {
				aggregate = "completed"
			}
			worldDigests := make([]string, 6)
			for worldOrdinal := 0; worldOrdinal < 6; worldOrdinal++ {
				world, ok := worlds[[3]int{curriculum, worldOrdinal, policyOrdinal}]
				if !ok {
					return MechanicalGates{}, fmt.Errorf("curriculum %d policy %s lacks world %d", curriculum, policy, worldOrdinal)
				}
				worldDigests[worldOrdinal] = world.Digest
				addVector(&search, world.UtilityWorkVector)
				behavior = behavior && world.BehaviorEqual
				if world.SearchTerminal == "budget-exhausted" {
					aggregate = "budget-exhausted"
				}
				if policy == actionrelationsearch.NousSleep && world.MatchCounts.UtilityFalseMatches != 0 {
					gates.NousZeroFalseMatches = false
				}
				if world.SearchTerminal == "completed" && !world.BehaviorEqual {
					gates.SemanticAgreement = false
				}
				if required[policy] && world.SearchTerminal == "completed" && !world.BehaviorEqual {
					gates.RequiredBehaviorEqual = false
				}
				if !freshCertificateCounts(policy, world) {
					gates.FreshCertificatesValid = false
				}
			}
			if search != row.SearchWorkVector || sum(search) != row.SearchTotal || !slices.Equal(worldDigests, row.WorldRowDigests) || behavior != row.BehaviorEqual || aggregate != row.AggregateTerminal {
				gates.WorkConservation = false
			}
		}

		complete := policies[actionrelationsearch.Complete]
		lexical := policies[actionrelationsearch.Lexical]
		for worldOrdinal := 0; worldOrdinal < 6; worldOrdinal++ {
			base := worlds[[3]int{curriculum, worldOrdinal, policyIndex(actionrelationsearch.Complete)}]
			lex := worlds[[3]int{curriculum, worldOrdinal, policyIndex(actionrelationsearch.Lexical)}]
			if base.SearchTerminal == "completed" && lex.SearchTerminal == "completed" && base.TerminalSetDigest != lex.TerminalSetDigest {
				gates.SemanticAgreement = false
			}
			for policy := range required {
				candidate := worlds[[3]int{curriculum, worldOrdinal, policyIndex(policy)}]
				if base.SearchTerminal == "completed" && candidate.SearchTerminal == "completed" && candidate.TerminalSetDigest != base.TerminalSetDigest {
					gates.RequiredBehaviorEqual = false
				}
			}
		}
		if complete.AggregateTerminal == "completed" && lexical.AggregateTerminal == "completed" && complete.BehaviorEqual != lexical.BehaviorEqual {
			gates.SemanticAgreement = false
		}
		if !immutableAcquisitionRows(policies) {
			gates.ArtifactsImmutable = false
		}
	}
	return gates, nil
}

func immutableAcquisitionRows(rows map[actionrelationsearch.Policy]CurriculumPolicyRow) bool {
	zero := [12]int{}
	for _, policy := range []actionrelationsearch.Policy{actionrelationsearch.Complete, actionrelationsearch.Lexical, actionrelationsearch.StaticSleep, actionrelationsearch.DynamicSleep} {
		row := rows[policy]
		if row.AcquisitionTerminal != "not-applicable" || row.ArtifactDigest != zeroDigest || row.AcquisitionWorkVector != zero || row.AcquisitionWorkTerminalDigestOrZero != zeroDigest {
			return false
		}
	}
	nous, control := rows[actionrelationsearch.NousSleep], rows[actionrelationsearch.LearnedNoUse]
	if nous.AcquisitionTerminal != "completed" || control.AcquisitionTerminal != nous.AcquisitionTerminal || control.ArtifactDigest != nous.ArtifactDigest || control.AcquisitionWorkVector != nous.AcquisitionWorkVector || control.AcquisitionWorkTerminalDigestOrZero != nous.AcquisitionWorkTerminalDigestOrZero {
		return false
	}
	noGuard := rows[actionrelationsearch.NoGuardSleep]
	return noGuard.AcquisitionTerminal == "completed" && noGuard.ArtifactDigest != zeroDigest
}

func freshCertificateCounts(policy actionrelationsearch.Policy, row WorldPolicyRow) bool {
	counts := row.CertificateCounts
	if counts.Successful > counts.Attempted || counts.CachedSuccess+counts.CachedFailure > row.UtilityWorkVector[10] {
		return false
	}
	switch policy {
	case actionrelationsearch.Complete, actionrelationsearch.Lexical, actionrelationsearch.LearnedNoUse:
		return row.SleepCount == 0 && counts == (CertificateCounts{})
	case actionrelationsearch.StaticSleep, actionrelationsearch.DynamicSleep, actionrelationsearch.NousSleep, actionrelationsearch.NoGuardSleep:
		return row.SleepCount <= counts.Successful+counts.CachedSuccess
	default:
		return false
	}
}
