package transformexp

import "testing"

func TestTransformationPowerIsDeterministic(t *testing.T) {
	rows := make([]pairedTransformRow, 45)
	for i := range rows {
		rows[i] = pairedTransformRow{Ordinal: i, Family: i % 9, NousSuccess: true, PBESuccess: i%9 < 5, NonmatchingNousWork: 4, NonmatchingPBEWork: 5}
	}
	first, err := estimateTransformPower(rows, 20, 100)
	if err != nil {
		t.Fatal(err)
	}
	second, err := estimateTransformPower(rows, 20, 100)
	if err != nil || first != second {
		t.Fatalf("nondeterministic power first=%+v second=%+v err=%v", first, second, err)
	}
	if first.Replicates != 20 || first.Passing < 0 || first.Passing > first.Replicates || first.Authorized != (first.Passing*100 >= 80*first.Replicates) {
		t.Fatalf("power=%+v", first)
	}
}
