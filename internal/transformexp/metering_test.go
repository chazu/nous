package transformexp

import "testing"

func TestAcquisitionSemanticRecordsReduceToTranscript(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runAcquisition("../../domains", c.Training, "metered-test")
	if err != nil {
		t.Fatal(err)
	}
	work, vector, err := transformMeterWork(run.MeterRecords)
	wantVector := [12]int64{1053, 184, 231, 2062, 13, 12, 30, 30, 98, 66, 20, 14}
	if err != nil || work != 3843 || vector != wantVector || len(run.MeterRecords) != 3813 {
		t.Fatalf("work=%d vector=%v err=%v", work, vector, err)
	}
	manifest := digestBytes([]byte("acquisition manifest"))
	bundle, err := transcriptFromAcquisition(run, 0, NousRefine, "0123456789abcdef", manifest)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Work != work+1 || bundle.Vector[11] != vector[11]+1 {
		t.Fatalf("bundle work=%d vector=%v", bundle.Work, bundle.Vector)
	}
	if _, err := reduceTransformTranscript(bundle.Raw, bundle.Objects, manifest); err != nil {
		t.Fatal(err)
	}
}
