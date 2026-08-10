package actionrelationutility

import (
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
	closed             bool
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
