package ruleinductionexp

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/ruleinductionoracle"
	"github.com/chazu/nous/internal/unit"
)

type oracleAuditReport struct {
	Agreements    int
	Disagreements int
	Work          int
	Stage1Ties    []string
	Stage2Ties    []string
	Constraints   int
	Comparisons   int
	Prunes        int
}

func oracleDefinition(code string) (ruleinductionoracle.Definition, bool) {
	for _, definition := range ruleinductionoracle.Definitions() {
		if ruleinductionoracle.Code(definition) == code {
			return definition, true
		}
	}
	return ruleinductionoracle.Definition{}, false
}

func firstExact(queue, exact []string) string {
	for _, queued := range queue {
		if slices.Contains(exact, queued) {
			return queued
		}
	}
	return ""
}

// auditProductionRun is deliberately post-run and reads production only as
// artifacts. All semantic judgments are recomputed by ruleinductionoracle.
func auditProductionRun(store *unit.Store, experiment *unit.Unit, fixture Fixture, policy Policy) (oracleAuditReport, error) {
	audit := oracleAuditReport{Stage1Ties: ruleinductionoracle.ExactCodes(fixture.Background, fixture.Stage1), Stage2Ties: ruleinductionoracle.ExactCodes(fixture.Background, fixture.Stage2)}
	joint := ruleinductionoracle.JointTheories()
	if len(ruleinductionoracle.Definitions()) != 15 || len(joint) != 240 {
		return audit, fmt.Errorf("oracle grammar cardinality definitions=%d joint=%d", len(ruleinductionoracle.Definitions()), len(joint))
	}
	for _, theory := range joint {
		_ = ruleinductionoracle.Evaluate(theory.Stage1, fixture.Background)
		_ = ruleinductionoracle.Evaluate(theory.Stage2, fixture.Background)
		audit.Work += 2
	}

	candidateCategory := experiment.GetString("candidateCategory")
	projectionCount := 0
	var projection *unit.Unit
	for _, name := range store.All() {
		candidate := store.Get(name)
		if candidate.GetString("experiment") != experiment.Name || !directCategory(candidate, candidateCategory) {
			continue
		}
		if candidate.GetBool("projection") {
			projectionCount++
			projection = candidate
		}
		if !candidate.GetBool("riEvaluated") {
			continue
		}
		definition, ok := oracleDefinition(candidate.GetString("definitionCode"))
		if !ok {
			return audit, fmt.Errorf("oracle: candidate %s has illegal definition", name)
		}
		relation := ruleinductionoracle.Evaluate(definition, fixture.Background)
		examples := fixture.Stage1
		if candidate.GetString("stage") == "stage2" {
			examples = fixture.Stage2
		}
		support, failures, falsePositive, falseNegative := 0, 0, 0, 0
		for _, example := range examples {
			actual := relation.Has(example.X, example.Y)
			if actual == example.Positive {
				support++
			} else {
				failures++
				if actual {
					falsePositive++
				} else {
					falseNegative++
				}
			}
		}
		audit.Work++
		if candidate.GetString("signature") != relation.Signature() || candidate.GetInt("exampleCount") != len(examples) || candidate.GetInt("supportCount") != support || candidate.GetInt("failureCount") != failures || candidate.GetInt("falsePositiveCount") != falsePositive || candidate.GetInt("falseNegativeCount") != falseNegative {
			audit.Disagreements++
			return audit, fmt.Errorf("oracle: candidate %s evidence disagrees", name)
		}
		audit.Agreements++
	}
	wantProjection := 0
	if policy == SharedLibrary || policy == SharedInlined {
		wantProjection = 1
	}
	if projectionCount != wantProjection {
		return audit, fmt.Errorf("oracle: shared projection count=%d want=%d", projectionCount, wantProjection)
	}
	wantFallback, fallbackActions := false, 0
	if projection != nil {
		wantFallback = projection.GetInt("failureCount") > 0
	}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "transcript" && u.GetString("action") == "fallback" {
			fallbackActions++
			if projection == nil || u.GetString("artifactLink") != projection.Name {
				return audit, fmt.Errorf("oracle: fallback transcript has wrong projection link")
			}
		}
	}
	wantUse := projection != nil && !wantFallback
	wantFallbackActions := 0
	if wantFallback {
		wantFallbackActions = 1
	}
	if experiment.GetBool("fellBack") != wantFallback || experiment.GetBool("usedFrozenLibrary") != wantUse || fallbackActions != wantFallbackActions {
		audit.Disagreements++
		return audit, fmt.Errorf("oracle: fallback conservation got fellBack=%t used=%t actions=%d want=%t/%t/%d", experiment.GetBool("fellBack"), experiment.GetBool("usedFrozenLibrary"), fallbackActions, wantFallback, wantUse, wantFallbackActions)
	}
	audit.Agreements++
	audit.Work++

	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") != experiment.Name {
			continue
		}
		if directCategory(u, experiment.GetString("constraintCategory")) {
			audit.Constraints++
			failed, failedOK := oracleDefinition(u.GetString("failedCode"))
			matchingFailure := false
			for _, candidateName := range store.All() {
				candidate := store.Get(candidateName)
				if candidate.GetString("experiment") == experiment.Name && directCategory(candidate, candidateCategory) && candidate.GetString("stage") == u.GetString("stage") && candidate.GetString("definitionCode") == u.GetString("failedCode") {
					matchingFailure = u.GetString("direction") == "too-general" && candidate.GetInt("falsePositiveCount") > 0 || u.GetString("direction") == "too-specific" && candidate.GetInt("falseNegativeCount") > 0
				}
			}
			if !failedOK || !matchingFailure || !ruleinductionoracle.ConstraintSound(u.GetString("direction"), failed, failed, fixture.Background) {
				return audit, fmt.Errorf("oracle: unsound constraint %s", name)
			}
			audit.Agreements++
			audit.Work++
		}
		if directCategory(u, experiment.GetString("comparisonCategory")) {
			audit.Comparisons++
			constraint := store.Get(u.GetString("constraint"))
			candidate := store.Get(u.GetString("candidate"))
			if constraint == nil || candidate == nil {
				return audit, fmt.Errorf("oracle: malformed comparison %s", name)
			}
			failed, failedOK := oracleDefinition(constraint.GetString("failedCode"))
			compared, comparedOK := oracleDefinition(candidate.GetString("definitionCode"))
			if !failedOK || !comparedOK {
				return audit, fmt.Errorf("oracle: malformed comparison %s", name)
			}
			general, specific := compared, failed
			if constraint.GetString("direction") == "too-specific" {
				general, specific = failed, compared
			}
			if u.GetBool("subsumes") != ruleinductionoracle.StructurallySubsumes(general, specific) {
				return audit, fmt.Errorf("oracle: comparison %s disagrees", name)
			}
			audit.Agreements++
			audit.Work++
		}
		if directCategory(u, experiment.GetString("pruneCategory")) {
			audit.Prunes++
			comparison := store.Get(u.GetString("comparison"))
			if comparison == nil || !comparison.GetBool("subsumes") {
				return audit, fmt.Errorf("oracle: unsound prune %s", name)
			}
			audit.Agreements++
			audit.Work++
		}
	}

	wantStage1 := firstExact(experiment.GetStrings("stage1Queue"), audit.Stage1Ties)
	wantStage2 := firstExact(experiment.GetStrings("stage2Queue"), audit.Stage2Ties)
	if (policy == SharedLibrary || policy == SharedInlined) && slices.Contains(audit.Stage2Ties, experiment.GetString("frozenCode")) {
		wantStage2 = experiment.GetString("frozenCode")
	}
	wantTerminal := "identified"
	if wantStage2 == "" {
		wantTerminal = "no-solution"
	}
	if experiment.GetString("frozenCode") != wantStage1 || experiment.GetString("selectedCode") != wantStage2 || experiment.GetString("terminal") != wantTerminal {
		audit.Disagreements++
		return audit, fmt.Errorf("oracle terminal disagreement got=%s/%s/%s want=%s/%s/%s", experiment.GetString("frozenCode"), experiment.GetString("selectedCode"), experiment.GetString("terminal"), wantStage1, wantStage2, wantTerminal)
	}
	audit.Agreements += 3
	audit.Work += 3
	return audit, nil
}
