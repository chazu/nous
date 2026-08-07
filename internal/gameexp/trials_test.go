package gameexp

import (
	"reflect"
	"testing"
)

func TestPreregisteredGameTrial(t *testing.T) {
	report, err := Run("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	if !report.ExperimentComplete || !report.AgendaDrained || !report.TrainingStoreUnchanged {
		t.Fatalf("completion/drain/immutability = %v/%v/%v", report.ExperimentComplete, report.AgendaDrained, report.TrainingStoreUnchanged)
	}
	if report.Candidates != 32 || report.EvaluationCases != 6 || report.Results != 192 || report.Observations != 192 || report.Evidence != 32 || report.Schemas != 14 || report.Conjectures != 14 {
		t.Fatalf("artifact counts = %#v", report)
	}
	wantFrontier := []string{"DDDCC", "DCDDC", "DCDCC", "DCCDD", "DCCDC", "DCCCD", "DCCCC", "CDDDC", "CDDCD", "CCDDD", "CCDCD", "CCDCC", "CCCCD", "CCCCC"}
	wantScalar := []string{"DDDDD", "DDCDD", "DCDDD", "DCCDD"}
	if !reflect.DeepEqual(report.Frontier, wantFrontier) || !reflect.DeepEqual(report.ScalarLeaders, wantScalar) {
		t.Fatalf("frontier/scalar = %v/%v", report.Frontier, report.ScalarLeaders)
	}
	if report.TrainingBehaviorClasses != 26 || report.FrontierBehaviorClasses != 13 || report.SplitTrainingClasses != 2 {
		t.Fatalf("behavior classes frontier/splits = %d/%d/%d", report.TrainingBehaviorClasses, report.FrontierBehaviorClasses, report.SplitTrainingClasses)
	}
	if report.Oracle.EnumerationAgreements != 32 || report.Oracle.MatchAgreements != 192 || report.Oracle.ObjectiveAgreements != 32 || report.Oracle.DominanceAgreements != 1024 || !report.Oracle.FrontierAgreement || !report.Oracle.ScalarLeaderAgreement || report.Oracle.BehaviorPartitionAgreements != 1024 {
		t.Fatalf("oracle agreement = %#v", report.Oracle)
	}
}

func TestGameTrialReportIsDeterministic(t *testing.T) {
	first, err := Run("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := first.JSON()
	secondJSON, _ := second.JSON()
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("fixed trial reports differ")
	}
}

func TestClassSplitsSyntheticWitness(t *testing.T) {
	training := map[int]string{}
	heldOut := map[int]string{}
	for code := 0; code < 32; code++ {
		training[code] = "unique-" + actionsForCode(code)
		heldOut[code] = training[code]
	}
	training[0], training[1] = "same-training", "same-training"
	heldOut[0], heldOut[1] = "held-a", "held-b"
	classes, pairs := classSplits(training, heldOut)
	if classes != 1 || pairs != 1 {
		t.Fatalf("synthetic split classes/pairs = %d/%d", classes, pairs)
	}
}
