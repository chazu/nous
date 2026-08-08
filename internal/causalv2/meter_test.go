package causalv2

import "testing"

func workCounter(values ...int64) Counter {
	var counts [15]int64
	for i, value := range values {
		counts[i] = value
	}
	counter := CounterFromCounts(counts)
	counter.TotalWork = counter.ComputedTotalWork()
	return counter
}

func TestMeterItemsAndTotalWork(t *testing.T) {
	counter := Counter{SCMEvaluations: 1, MemoStates: 2, MemoLookups: 3, QEvaluations: 4, TableLookups: 5}
	counter.TotalWork = 15
	if err := counter.Validate(); err != nil {
		t.Fatal(err)
	}
	items := make([]MeterItem, len(MeterNames))
	for i, name := range MeterNames {
		items[i] = MeterItem{Name: name}
	}
	items[0].Active = true
	items[0].Counts = counter.Counts()
	if err := ValidateMeterItems(items); err != nil {
		t.Fatal(err)
	}
	items[1].Counts[0] = 1
	items[1].Counts[14] = 1
	if err := ValidateMeterItems(items); err == nil {
		t.Fatal("accepted nonzero inactive item")
	}
}

func TestAggregateMeterCapsAndEmptyMaxima(t *testing.T) {
	empty, err := NewAggregateMeter("curriculum", "training", nil)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Episodes != 0 || empty.Totals != (Counter{}) || empty.Maxima != (Counter{}) || !empty.Valid {
		t.Fatalf("bad N=0 aggregate: %+v", empty)
	}
	over := Counter{SCMEvaluations: 4097, TotalWork: 4097}
	meter, err := NewAggregateMeter("production", "evaluation", []Counter{over})
	if err != nil {
		t.Fatal(err)
	}
	if meter.Valid {
		t.Fatal("production overrun marked valid")
	}
	if err := VerifyAggregateMeter(meter, "evaluation", []Counter{over}); err != nil {
		t.Fatal(err)
	}
}

func TestTaskMeterDigestAndCap(t *testing.T) {
	counter := Counter{QEvaluations: 2, TableLookups: 3, TotalWork: 5}
	items := []TaskMeterItem{
		{Name: "certificate-replay", Subject: "z-certificate", Counts: counter.Counts()},
		{Name: "certificate-replay", Subject: "a-certificate", Counts: counter.Counts()},
		{Name: "post-selection-replay", Subject: "z-certificate", Counts: counter.Counts()},
		{Name: "curriculum", Subject: "000001:admission", Counts: counter.Counts()},
	}
	if _, err := TaskMeterItemsDigest(items); err != nil {
		t.Fatal(err)
	}
	items[3].Name = "certificate-replay"
	if _, err := TaskMeterItemsDigest(items); err == nil {
		t.Fatal("accepted category order reversal")
	}
}
