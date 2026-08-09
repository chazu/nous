package transformexp

import (
	"bytes"
	"testing"
)

func TestSafePanelReportIsDeterministicAndCannotNameProtectedPanel(t *testing.T) {
	curricula := make([]curriculum, 9)
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 841500+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		curricula[family] = c
	}
	first, err := runSafePanel("../../domains", "safe", curricula, 841001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runSafePanel("../../domains", "safe", curricula, 841001)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := first.JSON()
	right, _ := second.JSON()
	if !bytes.Equal(left, right) {
		t.Fatal("safe report is nondeterministic")
	}
	if !bytes.Contains(left, []byte(`"point_denominator": 9`)) {
		t.Fatalf("safe report omitted inference: %s", left)
	}
	if first.MechanicallyValid || len(first.Rows) != 9*len(empiricalPolicies) || first.Competence.Passed != true {
		t.Fatalf("safe report shape=%+v", first)
	}
	if _, err := runSafePanel("../../domains", "development", curricula, 841001); err == nil {
		t.Fatal("safe runner accepted protected panel name")
	}
}
