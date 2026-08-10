package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type CertificateCache struct {
	rows map[string]string
}

type CacheDecision struct {
	Certified         bool
	CertificateDigest string
	CacheRow          string
	CacheHit          bool
	OperationRoot     actionrelationexp.OperationRoot
}

func NewCertificateCache() *CertificateCache {
	return &CertificateCache{rows: map[string]string{}}
}

func CertifyCached(session *Session, cache *CertificateCache, worldDigest, policy string, state actionrelations.State, a, b actionrelations.Occurrence, witness []byte, operationStart int, token string) (CacheDecision, error) {
	if session == nil || cache == nil || session.closed {
		return CacheDecision{}, fmt.Errorf("invalid utility certificate cache")
	}
	a, b, err := actionrelations.CanonicalPair(a, b)
	if err != nil || a == b {
		return CacheDecision{}, actionrelations.ErrInvalid
	}
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	key := strings.Join([]string{worldDigest, policy, stateDigest, aDigest, bDigest}, ":")
	lookupSequence := session.Sequence
	lookupTask, _ := json.Marshal([]any{"actionrelation-cache-lookup-task/v1", session.RunID, worldDigest, policy, stateDigest, aDigest, bDigest})
	lookupHash := sha256.Sum256(lookupTask)
	reservation, err := session.Reserve(hex.EncodeToString(lookupHash[:]), []uint8{18})
	if err != nil || reservation.Status != "reserved" {
		return CacheDecision{}, fmt.Errorf("reserve certificate cache lookup: %w", err)
	}
	inputs := [][]byte{[]byte(worldDigest), []byte(policy), []byte(stateDigest), []byte(aDigest), []byte(bDigest)}
	if rowName := cache.rows[key]; rowName != "" {
		row := session.Store.Get(rowName)
		if row == nil || !session.Store.IsA(row.Name, "ActionCertificateCacheRow") || row.GetString("worldDigest") != worldDigest || row.GetString("policy") != policy {
			return CacheDecision{}, fmt.Errorf("invalid certificate cache entry")
		}
		if err := dsl.ChargeActionRelationMeterStatus(session.MeterToken, 18, 11, 3, "certificate-cache-lookup", inputs, [][]byte{[]byte(row.GetString("canonicalObject"))}); err != nil {
			return CacheDecision{}, err
		}
		return CacheDecision{Certified: row.GetString("result") == "certified", CertificateDigest: nonzeroCertificate(row), CacheRow: rowName, CacheHit: true}, nil
	}
	if err := dsl.ChargeActionRelationMeter(session.MeterToken, 18, 11, "certificate-cache-lookup", inputs, nil); err != nil {
		return CacheDecision{}, err
	}
	transcript, err := BuildTranscript(session.Store, session.RunID, mustSnapshot(session))
	if err != nil {
		return CacheDecision{}, err
	}
	missCallID := transcript.CallIDs[lookupSequence]
	if operationStart < 0 {
		operationStart = lookupSequence
	}
	certificate, err := Certify(session, state, a, b, witness, operationStart, token)
	if err != nil {
		return CacheDecision{}, err
	}
	attempt := session.Store.Get(certificate.Attempt)
	if attempt == nil || attempt.GetString("status") != "valid" {
		return CacheDecision{}, fmt.Errorf("invalid certificate attempt is not cacheable")
	}
	finalizeTask, _ := json.Marshal([]any{"actionrelation-cache-finalize-task/v1", session.RunID, worldDigest, policy, stateDigest, aDigest, bDigest, missCallID, attempt.GetString("objectDigest"), certificate.OperationRoot.Digest})
	finalizeHash := sha256.Sum256(finalizeTask)
	reservation, err = session.Reserve(hex.EncodeToString(finalizeHash[:]), []uint8{25})
	if err != nil || reservation.Status != "reserved" {
		return CacheDecision{}, fmt.Errorf("reserve certificate cache finalization: %w", err)
	}
	requestName := "AR.CacheRequest." + token
	request := unit.New(requestName)
	request.Set("isA", []string{"ActionCertificateCacheRequest", "Anything"})
	request.Set("meterToken", session.MeterToken)
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	request.Set("worldDigest", worldDigest)
	request.Set("policy", policy)
	request.Set("state", string(stateJSON))
	request.Set("aOccurrence", string(aJSON))
	request.Set("bOccurrence", string(bJSON))
	request.Set("missLookupCallID", missCallID)
	request.Set("attemptUnit", attempt.Name)
	request.Set("operationRoot", certificate.OperationRoot.Digest)
	if session.Store.Has(requestName) {
		return CacheDecision{}, fmt.Errorf("certificate cache request name occupied")
	}
	session.Store.Put(request)
	if err := runCacheFinalize(session.Store, requestName); err != nil {
		return CacheDecision{}, err
	}
	rowName := request.GetString("resultRow")
	row := session.Store.Get(rowName)
	if row == nil {
		return CacheDecision{}, fmt.Errorf("certificate cache finalization lacks row")
	}
	cache.rows[key] = rowName
	return CacheDecision{Certified: row.GetString("result") == "certified", CertificateDigest: nonzeroCertificate(row), CacheRow: rowName, OperationRoot: certificate.OperationRoot}, nil
}

func runCacheFinalize(store *unit.Store, request string) error {
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: request, SlotName: "arCacheFinalize"})
	if eng.LastError != nil {
		return eng.LastError
	}
	if store.Get(request).GetString("terminal") != "completed" {
		return fmt.Errorf("certificate cache finalization did not complete")
	}
	return nil
}

func mustSnapshot(session *Session) []dsl.ActionRelationMeterRecord {
	records, _ := session.Snapshot()
	return records
}

func nonzeroCertificate(row *unit.Unit) string {
	if row.GetString("result") == "certified" {
		return row.GetString("certificateDigest")
	}
	return ""
}
