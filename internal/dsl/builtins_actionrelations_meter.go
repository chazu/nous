package dsl

import (
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

type ActionRelationMeterRecord struct {
	Code             uint16
	Counter          uint8
	Status           uint8
	Operation        string
	SourceTaskDigest string
	Inputs           [][]byte
	Outputs          [][]byte
}

type ActionRelationMeterPlanEntry struct {
	Code             uint16
	SourceTaskDigest string
}

type actionRelationMeter struct {
	records []ActionRelationMeterRecord
	plan    []ActionRelationMeterPlanEntry
}

var actionRelationMeters = struct {
	sync.Mutex
	items map[string]*actionRelationMeter
}{items: map[string]*actionRelationMeter{}}

func RegisterActionRelationMeter(token string) error {
	return RegisterActionRelationMeterPlan(token, nil)
}

func RegisterActionRelationMeterPlan(token string, plan []ActionRelationMeterPlanEntry) error {
	if token == "" {
		return errors.New("empty action-relation meter token")
	}
	actionRelationMeters.Lock()
	defer actionRelationMeters.Unlock()
	if _, exists := actionRelationMeters.items[token]; exists {
		return errors.New("duplicate action-relation meter token")
	}
	for _, entry := range plan {
		raw, err := hex.DecodeString(entry.SourceTaskDigest)
		if entry.Code < 1 || entry.Code > 25 || err != nil || len(raw) != 32 || entry.SourceTaskDigest != strings.ToLower(entry.SourceTaskDigest) {
			return errors.New("invalid action-relation meter plan")
		}
	}
	actionRelationMeters.items[token] = &actionRelationMeter{plan: append([]ActionRelationMeterPlanEntry(nil), plan...)}
	return nil
}

// ExtendActionRelationMeterPlan publishes one complete next compound block.
// No extension is legal while an earlier reserved block remains unconsumed.
func ExtendActionRelationMeterPlan(token string, plan []ActionRelationMeterPlanEntry) error {
	if len(plan) == 0 {
		return errors.New("empty action-relation meter plan extension")
	}
	actionRelationMeters.Lock()
	defer actionRelationMeters.Unlock()
	meter := actionRelationMeters.items[token]
	if meter == nil || len(meter.records) != len(meter.plan) {
		return errors.New("action-relation meter has outstanding reserved work")
	}
	for _, entry := range plan {
		raw, err := hex.DecodeString(entry.SourceTaskDigest)
		if entry.Code < 1 || entry.Code > 25 || err != nil || len(raw) != 32 || entry.SourceTaskDigest != strings.ToLower(entry.SourceTaskDigest) {
			return errors.New("invalid action-relation meter plan extension")
		}
	}
	meter.plan = append(meter.plan, plan...)
	return nil
}

func UnregisterActionRelationMeter(token string) {
	actionRelationMeters.Lock()
	delete(actionRelationMeters.items, token)
	actionRelationMeters.Unlock()
}

func ActionRelationMeterSnapshot(token string) ([]ActionRelationMeterRecord, error) {
	actionRelationMeters.Lock()
	defer actionRelationMeters.Unlock()
	meter := actionRelationMeters.items[token]
	if meter == nil {
		return nil, errors.New("unknown action-relation meter capability")
	}
	result := make([]ActionRelationMeterRecord, len(meter.records))
	for index, record := range meter.records {
		result[index] = record
		result[index].Inputs = cloneByteRows(record.Inputs)
		result[index].Outputs = cloneByteRows(record.Outputs)
	}
	return result, nil
}

func ChargeActionRelationMeter(token string, code uint16, counter uint8, operation string, inputs, outputs [][]byte) error {
	return ChargeActionRelationMeterStatus(token, code, counter, 1, operation, inputs, outputs)
}

func ChargeActionRelationMeterStatus(token string, code uint16, counter, status uint8, operation string, inputs, outputs [][]byte) error {
	if code < 1 || code > 25 || counter < 1 || counter > 12 || operation == "" {
		return errors.New("invalid action-relation meter event")
	}
	if status != 1 && (status != 3 || code != 16 && code != 18) {
		return errors.New("invalid action-relation meter status")
	}
	actionRelationMeters.Lock()
	defer actionRelationMeters.Unlock()
	meter := actionRelationMeters.items[token]
	if meter == nil {
		return errors.New("unknown action-relation meter capability")
	}
	source := ""
	if len(meter.plan) > 0 {
		sequence := len(meter.records)
		if sequence >= len(meter.plan) || meter.plan[sequence].Code != code {
			return errors.New("action-relation operation violates reserved plan")
		}
		source = meter.plan[sequence].SourceTaskDigest
	}
	meter.records = append(meter.records, ActionRelationMeterRecord{Code: code, Counter: counter, Status: status, Operation: operation, SourceTaskDigest: source, Inputs: cloneByteRows(inputs), Outputs: cloneByteRows(outputs)})
	return nil
}

func ActionRelationMeterPlanComplete(token string) error {
	actionRelationMeters.Lock()
	defer actionRelationMeters.Unlock()
	meter := actionRelationMeters.items[token]
	if meter == nil || len(meter.plan) > 0 && len(meter.records) != len(meter.plan) {
		return errors.New("incomplete action-relation meter plan")
	}
	return nil
}

func recordActionRelation(vm *VM, code uint16, counter uint8, operation string, inputs, outputs [][]byte) error {
	token := actionRelationMeterToken(vm)
	if token == "" {
		return nil
	}
	return ChargeActionRelationMeter(token, code, counter, operation, inputs, outputs)
}

func actionRelationMeterToken(vm *VM) string {
	if vm == nil || vm.Store == nil || vm.CurrentTask == nil {
		return ""
	}
	u := vm.Store.Get(vm.CurrentTask.UnitName)
	if u == nil {
		return ""
	}
	if token := u.GetString("meterToken"); token != "" {
		return token
	}
	if experiment := vm.Store.Get(u.GetString("experiment")); experiment != nil {
		return experiment.GetString("meterToken")
	}
	return ""
}

func bARMeter(vm *VM) error {
	token := actionRelationMeterToken(vm)
	records, err := ActionRelationMeterSnapshot(token)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	counts := make([]int, 12)
	for _, record := range records {
		counts[int(record.Counter)-1]++
	}
	values := make([]Value, len(counts))
	for index, count := range counts {
		values[index] = IntVal(count)
	}
	vm.push(ListVal(values))
	return nil
}

func cloneByteRows(rows [][]byte) [][]byte {
	result := make([][]byte, len(rows))
	for index, row := range rows {
		result[index] = append([]byte(nil), row...)
	}
	return result
}

func actionRelationPhaseCode(vm *VM, training, certificate, learned, fallback uint16) uint16 {
	if vm != nil && vm.CurrentTask != nil {
		switch vm.CurrentTask.SlotName {
		case "arObserve", "arEvaluate", "arFinalize":
			return training
		case "arCertify", "arCertifyInitial", "arCertifyCross", "arCertifyEquality", "arCertifyAssemble":
			return certificate
		case "arMatch":
			return learned
		}
	}
	return fallback
}
