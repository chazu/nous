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
	if err != nil || work <= 0 || vector[0] == 0 || vector[1] == 0 || vector[2] == 0 || vector[4] != 13 || vector[5] != 12 || vector[11] != 14 {
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
