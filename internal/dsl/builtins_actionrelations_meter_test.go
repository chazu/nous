package dsl

import (
	"strings"
	"testing"
)

func TestActionRelationMeterIsCapabilityScoped(t *testing.T) {
	if err := RegisterActionRelationMeter("ar-meter-test"); err != nil {
		t.Fatal(err)
	}
	defer UnregisterActionRelationMeter("ar-meter-test")
	if err := ChargeActionRelationMeter("ar-meter-test", 1, 1, "guard-root", [][]byte{[]byte("in")}, [][]byte{[]byte("out")}); err != nil {
		t.Fatal(err)
	}
	if err := ChargeActionRelationMeter("ar-meter-test", 25, 11, "cache-finalize", nil, nil); err != nil {
		t.Fatal(err)
	}
	records, err := ActionRelationMeterSnapshot("ar-meter-test")
	if err != nil || len(records) != 2 || records[0].Code != 1 || records[1].Counter != 11 {
		t.Fatalf("records=%#v err=%v", records, err)
	}
	records[0].Inputs[0][0] = 'x'
	again, _ := ActionRelationMeterSnapshot("ar-meter-test")
	if string(again[0].Inputs[0]) != "in" {
		t.Fatal("meter snapshot aliased owned evidence")
	}
}

func TestActionRelationMeterEnforcesReservedPlan(t *testing.T) {
	plan := []ActionRelationMeterPlanEntry{
		{Code: 1, SourceTaskDigest: strings.Repeat("1", 64)},
		{Code: 25, SourceTaskDigest: strings.Repeat("2", 64)},
	}
	if err := RegisterActionRelationMeterPlan("ar-meter-plan-test", plan); err != nil {
		t.Fatal(err)
	}
	defer UnregisterActionRelationMeter("ar-meter-plan-test")
	if err := ChargeActionRelationMeter("ar-meter-plan-test", 25, 11, "out-of-order", nil, nil); err == nil {
		t.Fatal("meter accepted an operation outside reserved order")
	}
	if err := ChargeActionRelationMeter("ar-meter-plan-test", 1, 1, "guard-root", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ActionRelationMeterPlanComplete("ar-meter-plan-test"); err == nil {
		t.Fatal("meter accepted an incomplete reserved plan")
	}
	if err := ChargeActionRelationMeter("ar-meter-plan-test", 25, 11, "cache-finalize", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ActionRelationMeterPlanComplete("ar-meter-plan-test"); err != nil {
		t.Fatal(err)
	}
	records, err := ActionRelationMeterSnapshot("ar-meter-plan-test")
	if err != nil || len(records) != 2 || records[0].SourceTaskDigest != strings.Repeat("1", 64) || records[1].SourceTaskDigest != strings.Repeat("2", 64) {
		t.Fatalf("records=%#v err=%v", records, err)
	}
}
