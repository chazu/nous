package actionrelationcompetence

import "testing"

func TestSafeCompetenceUniverse(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Sequences != 40320 || report.Steps != 322560 {
		t.Fatalf("report=%+v", report)
	}
}
