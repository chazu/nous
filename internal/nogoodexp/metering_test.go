package nogoodexp

import (
	"os"
	"strings"
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
	tasks, err := nogoodfixture.DevelopmentPanel()
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

func TestMeterRejectsTrainingAndResumeOmissionsAndTupleRetargeting(t *testing.T) {
	training, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	for index, record := range training.MeterRecords {
		if record.Operation == "problem-read" {
			training.MeterRecords = append(training.MeterRecords[:index:index], training.MeterRecords[index+1:]...)
			break
		}
	}
	if _, err := acquisitionTranscript(training, nil); err == nil {
		t.Fatal("training meter accepted an omitted problem read")
	}
	training, err = RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	for index := range training.MeterRecords {
		if training.MeterRecords[index].Operation == "problem-read" {
			training.MeterRecords[index].Subject = "NG.Training.Example.3"
			break
		}
	}
	if _, err := acquisitionTranscript(training, nil); err == nil {
		t.Fatal("training meter accepted a retargeted problem read")
	}
	training, err = RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	training.MeterRecords[0].Category = 11
	if _, err := acquisitionTranscript(training, nil); err == nil {
		t.Fatal("training meter accepted category corruption")
	}

	artifact, authority := learnedArtifact(t)
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	resume, err := ConsiderPrune("../../domains", tasks[56].ProblemJSON, tasks[56].Decision, &artifact, &authority)
	if err != nil {
		t.Fatal(err)
	}
	for index, record := range resume.MeterRecords {
		if record.Operation == "domain-read" {
			resume.MeterRecords = append(resume.MeterRecords[:index:index], resume.MeterRecords[index+1:]...)
			break
		}
	}
	if _, err := bridgeTranscript(56, resume); err == nil {
		t.Fatal("resume meter accepted an omitted domain read")
	}

	proposal, err := ConsiderPrune("../../domains", tasks[0].ProblemJSON, tasks[0].Decision, &artifact, &authority)
	if err != nil {
		t.Fatal(err)
	}
	for index := range proposal.MeterRecords {
		if proposal.MeterRecords[index].Operation == "completion-domain-read" {
			proposal.MeterRecords[index].Object = "arbitrary"
			break
		}
	}
	if _, err := bridgeTranscript(0, proposal); err == nil {
		t.Fatal("proposal meter accepted a retargeted completion-domain read")
	}
}

func TestAcquisitionTranscriptUsesOnlyClosedPhaseOrdinals(t *testing.T) {
	training, err := RunTraining("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := NewBridgeExecution("../../domains", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := acquisitionTranscript(training, bridge.preflight)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint32]bool{}
	for _, event := range events {
		if !legalAcquisitionOrdinal(event.TaskOrdinal) {
			t.Fatalf("illegal acquisition ordinal %#x", event.TaskOrdinal)
		}
		seen[event.TaskOrdinal] = true
	}
	for ordinal := uint32(0x80000000); ordinal <= 0x80000004; ordinal++ {
		if !seen[ordinal] {
			t.Fatalf("missing training phase ordinal %#x", ordinal)
		}
	}
	for ordinal := uint32(0x90000000); ordinal <= 0x90000018; ordinal++ {
		if !seen[ordinal] {
			t.Fatalf("missing promotion/freeze phase ordinal %#x", ordinal)
		}
	}
	if !seen[0xffffffff] {
		t.Fatal("missing bridge-profile phase ordinal")
	}
	for _, event := range markRecomputedAcquisition(events, 7) {
		named := false
		for _, operand := range event.Operands {
			named = named || strings.HasPrefix(operand.Text, "recomputed-acquisition/")
		}
		if !named || event.TaskOrdinal != 7 {
			t.Fatalf("recomputed acquisition event lacks closed phase identity: %#v", event)
		}
	}
}
