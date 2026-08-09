package transformexp

import "testing"

func TestOrdinaryHeuristicsAcquireAndAllocate(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runAcquisition("../../domains", c.Training, "safe-test")
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "awaiting-factor-evidence" || len(run.Programs) != 4 || len(run.Candidates) != 12 {
		t.Fatalf("terminal=%s programs=%d candidates=%d tasks=%d", run.Terminal, len(run.Programs), len(run.Candidates), run.TasksPopped)
	}
	if len(run.MeterRecords) != 96 {
		t.Fatalf("meter records=%d", len(run.MeterRecords))
	}
}
