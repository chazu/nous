package ruleinduction

import "testing"

func TestProfileKeysCommitToQueuesAndStageBoundary(t *testing.T) {
	codes, _ := CanonicalCodes(EnumerateDefinitions())
	p := ExperimentProfile{
		ProfileVersion: "rule-induction-profile/v1", ExperimentVersion: ExperimentVersion, GeneratorVersion: GeneratorVersion, GrammarVersion: GrammarVersion, CostVersion: CostVersion, OracleVersion: "independent-fixed-point/v1", ReportVersion: "rule-induction-report/v1", BaselineVersion: "factored-direct-lff/v1", StatisticsVersion: "paired-resampling/v1", QueueVersion: "policy-queues/v1", CacheVersion: "semantic-definition-stage-cache/v1", IntegrityContract: "budgeted-transcript",
		Panel: "development", Seed: 11001, Policy: "shared-library",
		Categories:       CategoryBindings{Partial: "p", Refinement: "j", Candidate: "c", Result: "r", Observation: "o", Evidence: "e", Constraint: "f", Comparison: "m", Prune: "u", Library: "l", Provenance: "v", Projection: "y", Transcript: "x", Boundary: "b", Corpus: "z", Selection: "s", Terminal: "t"},
		Tasks:            TaskBindings{Start: "start", Refine: "refine", Evaluate: "evaluate", Continue: "continue"},
		ConstantBindings: []string{"ka", "kb", "kc", "kd", "ke", "kf", "kg", "kh"}, PredicateBindings: []string{"q0", "q1", "q2"}, Metarules: []string{"identity", "tailrec", "invented-projection"}, Stage1Queue: codes, Stage2Queue: codes,
		CandidateCap: 15, EvaluationCap: 31, FixedPointStepCap: 1000000, SemanticWorkCap: 500000, EngineCycleCap: 1000, AttributedUnitCap: 100000, ReportByteCap: 8388608, InitialPriority: 950, RefinePriority: 900, EvaluatePriority: 800, InitialReason: "Start staged rule induction",
	}
	key, err := p.Key()
	if err != nil || len(key) != 71 {
		t.Fatalf("key = %q, err = %v", key, err)
	}
	p.Stage2Queue[0], p.Stage2Queue[1] = p.Stage2Queue[1], p.Stage2Queue[0]
	changed, err := p.Key()
	if err != nil || changed == key {
		t.Fatalf("queue change did not change profile: %q", changed)
	}
	p.InitialPriority++
	priorityChanged, err := p.Key()
	if err != nil || priorityChanged == changed {
		t.Fatalf("priority change did not change profile: %q", priorityChanged)
	}
	stage1 := StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: changed, Stage: 1, FactDigest: SemanticDigest([]string{"0:0:1"}), ExampleDigest: SemanticDigest([]string{"0:1:1"})}
	stage1Key, err := stage1.Key()
	if err != nil {
		t.Fatal(err)
	}
	stage2 := StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: changed, Stage: 2, FactDigest: stage1.FactDigest, ExampleDigest: SemanticDigest([]string{"0:2:0"}), PriorBoundaryDigest: SemanticDigest(stage1Key)}
	if _, err := stage2.Key(); err != nil {
		t.Fatal(err)
	}
}

func TestProfileRejectsUnknownPolicyAndNoncanonicalCaps(t *testing.T) {
	codes, _ := CanonicalCodes(EnumerateDefinitions())
	profile := ExperimentProfile{
		ProfileVersion: "rule-induction-profile/v1", ExperimentVersion: ExperimentVersion, GeneratorVersion: GeneratorVersion, GrammarVersion: GrammarVersion, CostVersion: CostVersion, OracleVersion: "independent-fixed-point/v1", ReportVersion: "rule-induction-report/v1", BaselineVersion: "factored-direct-lff/v1", StatisticsVersion: "paired-resampling/v1", QueueVersion: "policy-queues/v1", CacheVersion: "semantic-definition-stage-cache/v1", IntegrityContract: "budgeted-transcript",
		Panel: "development", Seed: 11001, Policy: "shared-library",
		Categories: CategoryBindings{Partial: "p", Refinement: "j", Candidate: "c", Result: "r", Observation: "o", Evidence: "e", Constraint: "f", Comparison: "m", Prune: "u", Library: "l", Provenance: "v", Projection: "y", Transcript: "x", Boundary: "b", Corpus: "z", Selection: "s", Terminal: "t"},
		Tasks:      TaskBindings{Start: "start", Refine: "refine", Evaluate: "evaluate", Continue: "continue"}, ConstantBindings: []string{"ka", "kb", "kc", "kd", "ke", "kf", "kg", "kh"}, PredicateBindings: []string{"q0", "q1", "q2"}, Metarules: []string{"identity", "tailrec", "invented-projection"}, Stage1Queue: codes, Stage2Queue: codes,
		CandidateCap: CandidateCap, EvaluationCap: EvaluationCap, FixedPointStepCap: FixedPointStepCap, SemanticWorkCap: SemanticWorkCap, EngineCycleCap: EngineCycleCap, AttributedUnitCap: AttributedUnitCap, ReportByteCap: ReportByteCap, InitialPriority: 950, RefinePriority: 900, EvaluatePriority: 800, InitialReason: "Start staged rule induction",
	}
	if !profile.Valid() {
		t.Fatal("canonical profile rejected")
	}
	profile.Policy = "unknown-policy"
	if profile.Valid() {
		t.Fatal("unknown policy accepted")
	}
	profile.Policy = "shared-library"
	for name, mutate := range map[string]func(*ExperimentProfile){
		"evaluation":      func(p *ExperimentProfile) { p.EvaluationCap-- },
		"fixed point":     func(p *ExperimentProfile) { p.FixedPointStepCap-- },
		"semantic work":   func(p *ExperimentProfile) { p.SemanticWorkCap-- },
		"engine cycle":    func(p *ExperimentProfile) { p.EngineCycleCap-- },
		"attributed unit": func(p *ExperimentProfile) { p.AttributedUnitCap-- },
		"report byte":     func(p *ExperimentProfile) { p.ReportByteCap-- },
	} {
		t.Run(name, func(t *testing.T) {
			changed := profile
			mutate(&changed)
			if changed.Valid() {
				t.Fatal("noncanonical cap accepted")
			}
		})
	}
}
