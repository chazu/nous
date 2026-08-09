package nogoodexp

import (
	"os"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func TestEngineDispatchSurchargePinsAuditedSource(t *testing.T) {
	want := map[string]string{
		"../engine/engine.go": "74decf624a94088639b25f73a273a261914fc3020f4d218fbfa613721bca0022",
		"../engine/fire.go":   "336f97c0b271a9c386056e4e2784b849facf91805494fec7a345a9708947303f",
	}
	for path, digest := range want {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := digestBytes(data); got != digest {
			t.Fatalf("audited dispatch source %s drifted: %s", path, got)
		}
	}
	if len(engineDispatchOperations) != 22 {
		t.Fatalf("dispatch surcharge operations = %d", len(engineDispatchOperations))
	}
}

func TestMeterRejectsRecordsThatDoNotReconcileWithOccurrenceStore(t *testing.T) {
	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	disposition, err := ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &artifact, &authority)
	if err != nil {
		t.Fatal(err)
	}
	disposition.MeterRecords = disposition.MeterRecords[1:]
	if _, err := bridgeTranscript(0, disposition); err == nil {
		t.Fatal("meter accepted an omitted operation")
	}
}
