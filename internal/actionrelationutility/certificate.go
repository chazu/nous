package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationcertify"
	"github.com/chazu/nous/internal/actionrelationexp"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type CertificateResult struct {
	actionrelationcertify.Result
	OperationRoot actionrelationexp.OperationRoot
}

func Certify(session *Session, state actionrelations.State, a, b actionrelations.Occurrence, witness []byte, operationStart int, token string) (CertificateResult, error) {
	if session == nil || session.closed {
		return CertificateResult{}, fmt.Errorf("closed utility session")
	}
	request, err := actionrelationcertify.Begin(session.Store, state, a, b, witness, token, session.MeterToken)
	if err != nil {
		return CertificateResult{}, err
	}
	reserveStage := func(stage string, codes []uint8) error {
		if len(codes) == 0 {
			return nil
		}
		taskWire, _ := json.Marshal([]any{"actionrelation-certificate-stage/v1", session.RunID, request, stage, codes})
		taskHash := sha256.Sum256(taskWire)
		reservation, err := session.Reserve(hex.EncodeToString(taskHash[:]), codes)
		if err != nil {
			return fmt.Errorf("reserve certificate %s: %w", stage, err)
		}
		if reservation.Status == "rejected-cap" {
			return &budgetExhaustedError{Reservation: reservation}
		}
		if reservation.Status != "reserved" {
			return fmt.Errorf("reserve certificate %s returned %s", stage, reservation.Status)
		}
		return nil
	}
	if err := reserveStage("initial", []uint8{13, 13, 12, 12}); err != nil {
		return CertificateResult{}, err
	}
	if err := actionrelationcertify.RunInitial(session.Store, request); err != nil {
		return CertificateResult{}, err
	}
	crossCodes := actionrelationcertify.CrossOperationCodes(session.Store, request)
	if err := reserveStage("cross", crossCodes); err != nil {
		return CertificateResult{}, err
	}
	if err := actionrelationcertify.RunCross(session.Store, request); err != nil {
		return CertificateResult{}, err
	}
	equalityCodes := actionrelationcertify.EqualityOperationCodes(session.Store, request)
	if err := reserveStage("equality", equalityCodes); err != nil {
		return CertificateResult{}, err
	}
	if err := actionrelationcertify.RunEquality(session.Store, request); err != nil {
		return CertificateResult{}, err
	}
	root, err := session.OperationRoot(operationStart)
	if err != nil {
		return CertificateResult{}, err
	}
	result, err := actionrelationcertify.Complete(session.Store, request, root.Digest)
	if err != nil {
		return CertificateResult{}, err
	}
	return CertificateResult{Result: result, OperationRoot: root}, nil
}
