package transformexp

import "testing"

func TestExhaustiveTransformationCompetence(t *testing.T) {
	report, err := runTransformCompetence()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Forests != 351 || report.SchemaApplications != 25272 || report.ProgramApplications != 7020 || report.Microcases != 14 {
		t.Fatalf("competence=%+v", report)
	}
}
