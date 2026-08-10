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

func TestActionRelationMeterExtendsOnlyAtCompoundBoundary(t *testing.T) {
	token := "ar-meter-extension-test"
	if err := RegisterActionRelationMeterPlan(token, nil); err != nil {
		t.Fatal(err)
	}
	defer UnregisterActionRelationMeter(token)
	first := ActionRelationMeterPlanEntry{Code: 23, SourceTaskDigest: strings.Repeat("a", 64)}
	second := ActionRelationMeterPlanEntry{Code: 19, SourceTaskDigest: strings.Repeat("b", 64)}
	if err := ExtendActionRelationMeterPlan(token, []ActionRelationMeterPlanEntry{first}); err != nil {
		t.Fatal(err)
	}
	if err := ExtendActionRelationMeterPlan(token, []ActionRelationMeterPlanEntry{second}); err == nil {
		t.Fatal("meter extended with outstanding reserved work")
	}
	if err := ChargeActionRelationMeter(token, 23, 10, "search-applicable", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ExtendActionRelationMeterPlan(token, []ActionRelationMeterPlanEntry{second}); err != nil {
		t.Fatal(err)
	}
	if err := ChargeActionRelationMeter(token, 19, 12, "terminal", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ActionRelationMeterPlanComplete(token); err != nil {
		t.Fatal(err)
	}
}

func TestActionRelationMeterRestrictsCacheHitStatus(t *testing.T) {
	if err := RegisterActionRelationMeterPlan("ar-meter-status", []ActionRelationMeterPlanEntry{{Code: 18, SourceTaskDigest: strings.Repeat("c", 64)}}); err != nil {
		t.Fatal(err)
	}
	defer UnregisterActionRelationMeter("ar-meter-status")
	if err := ChargeActionRelationMeterStatus("ar-meter-status", 18, 11, 3, "certificate-cache-lookup", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := RegisterActionRelationMeterPlan("ar-meter-status-invalid", []ActionRelationMeterPlanEntry{{Code: 13, SourceTaskDigest: strings.Repeat("d", 64)}}); err != nil {
		t.Fatal(err)
	}
	defer UnregisterActionRelationMeter("ar-meter-status-invalid")
	if err := ChargeActionRelationMeterStatus("ar-meter-status-invalid", 13, 10, 3, "not-a-cache", nil, nil); err == nil {
		t.Fatal("non-lookup operation accepted cache-hit status")
	}
}
