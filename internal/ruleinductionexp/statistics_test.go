package ruleinductionexp

import "testing"

func TestPairedStatisticsAreDeterministic(t *testing.T) {
	treatment := []float64{5, 6, 7, 8, 9, 10, 11, 12}
	control := []float64{10, 12, 14, 16, 18, 20, 22, 24}
	first := pairedContrast("direct", "work-ratio-of-means", treatment, control, .15)
	second := pairedContrast("direct", "work-ratio-of-means", treatment, control, .15)
	if first != second || first.RelativeReduction != .5 || !first.Passed {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}
