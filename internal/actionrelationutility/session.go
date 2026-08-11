package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

type Session struct {
	Store              *unit.Store
	RunID              string
	MeterToken         string
	Cap                int
	Total              int
	Sequence           int
	ReservationOrdinal int
	Reservations       []string
	InitialWork        [12]int
	closed             bool
}

type WorkTerminal struct {
	Canonical           []byte
	Digest              string
	RejectedReservation actionrelationledger.Reservation
}

func BeginSession(store *unit.Store, runID, token string, initialTotal, cap int) (*Session, error) {
	if store == nil || token == "" || initialTotal < 0 || cap < 1 || initialTotal >= cap {
		return nil, fmt.Errorf("invalid utility session")
	}
	if err := dsl.RegisterActionRelationMeterPlan(token, nil); err != nil {
		return nil, err
	}
	return &Session{Store: store, RunID: runID, MeterToken: token, Cap: cap, Total: initialTotal}, nil
}

func (s *Session) SetInitialWorkVector(vector [12]int) error {
	if s == nil || s.closed || s.Sequence != 0 || sumWorkVector(vector) != s.Total {
		return fmt.Errorf("invalid initial utility work vector")
	}
	s.InitialWork = vector
	return nil
}

func (s *Session) Reserve(taskDigest string, operationCodes []uint8) (actionrelationledger.Reservation, error) {
	if s == nil || s.closed || len(operationCodes) == 0 {
		return actionrelationledger.Reservation{}, fmt.Errorf("closed utility session")
	}
	reservation, err := actionrelationledger.BuildReservation(s.RunID, taskDigest, operationCodes, s.Total, s.Cap)
	if err != nil {
		return actionrelationledger.Reservation{}, err
	}
	name := fmt.Sprintf("AR.Reservation.%s.%05d", s.RunID, s.ReservationOrdinal)
	u := unit.New(name)
	u.Set("isA", []string{"CompoundWorkReservation", "Anything"})
	u.Set("canonicalObject", string(reservation.Canonical))
	u.Set("objectDigest", reservation.Digest)
	s.Store.Put(u)
	s.Reservations = append(s.Reservations, name)
	s.ReservationOrdinal++
	if reservation.Status != "reserved" {
		return reservation, nil
	}
	plan := make([]dsl.ActionRelationMeterPlanEntry, len(operationCodes))
	for index, code := range operationCodes {
		plan[index] = dsl.ActionRelationMeterPlanEntry{Code: uint16(code), SourceTaskDigest: reservation.Digest}
	}
	if err := dsl.ExtendActionRelationMeterPlan(s.MeterToken, plan); err != nil {
		return actionrelationledger.Reservation{}, err
	}
	s.Total = reservation.TotalAfter
	s.Sequence += len(operationCodes)
	return reservation, nil
}

func (s *Session) Snapshot() ([]dsl.ActionRelationMeterRecord, error) {
	if s == nil || s.closed {
		return nil, fmt.Errorf("closed utility session")
	}
	return dsl.ActionRelationMeterSnapshot(s.MeterToken)
}

func (s *Session) Close() ([]dsl.ActionRelationMeterRecord, error) {
	if s == nil || s.closed {
		return nil, fmt.Errorf("closed utility session")
	}
	if err := dsl.ActionRelationMeterPlanComplete(s.MeterToken); err != nil {
		return nil, err
	}
	records, err := dsl.ActionRelationMeterSnapshot(s.MeterToken)
	s.closed = true
	dsl.UnregisterActionRelationMeter(s.MeterToken)
	return records, err
}

func (s *Session) Abort() {
	if s != nil && !s.closed {
		s.closed = true
		dsl.UnregisterActionRelationMeter(s.MeterToken)
	}
}

func MeterWorkVector(records []dsl.ActionRelationMeterRecord) ([12]int, error) {
	var result [12]int
	for _, record := range records {
		if record.Counter < 1 || record.Counter > 12 {
			return [12]int{}, fmt.Errorf("invalid action-relation work counter")
		}
		result[record.Counter-1]++
	}
	return result, nil
}

func (s *Session) WorkVector() ([12]int, error) {
	records, err := s.Snapshot()
	if err != nil {
		return [12]int{}, err
	}
	vector, err := MeterWorkVector(records)
	if err != nil {
		return [12]int{}, err
	}
	for index := range vector {
		vector[index] += s.InitialWork[index]
	}
	if sumWorkVector(vector) != s.Total {
		return [12]int{}, fmt.Errorf("utility work vector does not conserve reservations")
	}
	return vector, nil
}

func (s *Session) TerminateBudget(rejected actionrelationledger.Reservation) (WorkTerminal, error) {
	if s == nil || s.closed || rejected.Status != "rejected-cap" || rejected.RunID != s.RunID || rejected.TotalBefore != s.Total {
		return WorkTerminal{}, fmt.Errorf("invalid rejected utility reservation")
	}
	taskWire, _ := json.Marshal([]any{"actionrelation-budget-terminal-task/v1", s.RunID, rejected.Digest})
	taskHash := sha256.Sum256(taskWire)
	terminalReservation, err := actionrelationledger.BuildTerminalReservation(s.RunID, hex.EncodeToString(taskHash[:]), s.Total, s.Cap)
	if err != nil {
		return WorkTerminal{}, err
	}
	name := fmt.Sprintf("AR.Reservation.%s.%05d", s.RunID, s.ReservationOrdinal)
	u := unit.New(name)
	u.Set("isA", []string{"CompoundWorkReservation", "Anything"})
	u.Set("canonicalObject", string(terminalReservation.Canonical))
	u.Set("objectDigest", terminalReservation.Digest)
	s.Store.Put(u)
	s.Reservations = append(s.Reservations, name)
	s.ReservationOrdinal++
	if err := dsl.ExtendActionRelationMeterPlan(s.MeterToken, []dsl.ActionRelationMeterPlanEntry{{Code: 19, SourceTaskDigest: terminalReservation.Digest}}); err != nil {
		return WorkTerminal{}, err
	}
	vector, err := s.WorkVector()
	if err != nil {
		return WorkTerminal{}, err
	}
	vector[11]++
	total := sumWorkVector(vector)
	wire, _ := json.Marshal([]any{"action-work-terminal/v1", s.RunID, 2, rejected.Digest, "budget-exhausted", vector, total, 0})
	digestBytes := sha256.Sum256(wire)
	terminal := WorkTerminal{Canonical: wire, Digest: hex.EncodeToString(digestBytes[:]), RejectedReservation: rejected}
	if err := dsl.ChargeActionRelationMeter(s.MeterToken, 19, 12, "budget-terminal", [][]byte{[]byte(rejected.Digest)}, [][]byte{wire}); err != nil {
		return WorkTerminal{}, err
	}
	s.Total = terminalReservation.TotalAfter
	s.Sequence++
	return terminal, nil
}

func sumWorkVector(vector [12]int) int {
	total := 0
	for _, value := range vector {
		total += value
	}
	return total
}
