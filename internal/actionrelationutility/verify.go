package actionrelationutility

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/unit"
)

const utilityZeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

func VerifySearchRun(run SearchRun) error {
	if run.Store == nil || run.RunID == "" || run.WorldDigest == "" || run.Policy == "" || run.Terminal != "completed" && run.Terminal != "budget-exhausted" {
		return fmt.Errorf("invalid utility run identity")
	}
	if run.Terminal == "completed" {
		if run.WorkTerminal.Digest != "" || actionrelationsearch.VerifyResultEvidence(run.Search) != nil {
			return fmt.Errorf("invalid completed utility evidence")
		}
	} else if err := verifyWorkTerminal(run); err != nil {
		return err
	}
	rebuiltTranscript, err := BuildTranscript(run.Store, run.RunID, run.Records)
	if err != nil || actionrelationexp.VerifyTranscript(run.Transcript) != nil || !reflect.DeepEqual(rebuiltTranscript, run.Transcript) {
		return fmt.Errorf("utility transcript does not rebuild")
	}
	wantRunRoot, err := actionrelationexp.BuildOperationRange(run.RunID, 2, 0, run.Transcript.CallIDs)
	if err != nil || wantRunRoot.Digest != run.RunRoot.Digest || !bytes.Equal(wantRunRoot.Canonical, run.RunRoot.Canonical) || actionrelationexp.VerifyOperationRange(run.RunRoot, run.Transcript) != nil {
		return fmt.Errorf("utility run operation root mismatch")
	}
	vector, err := MeterWorkVector(run.Records)
	if err != nil {
		return err
	}
	for index := range vector {
		vector[index] += run.InitialWork[index]
	}
	if vector != run.WorkVector || sumWorkVector(vector) != run.WorkTotal {
		return fmt.Errorf("utility work vector does not conserve")
	}
	return verifyCertificateAuthority(run)
}

func verifyWorkTerminal(run SearchRun) error {
	if run.WorkTerminal.Digest != digestBytesText(run.WorkTerminal.Canonical) {
		return fmt.Errorf("invalid work-terminal digest")
	}
	row, err := utilityCanonicalRow(run.WorkTerminal.Canonical, 8, "action-work-terminal/v1")
	if err != nil {
		return err
	}
	var runID, rejected, terminal string
	var phase, total, remaining int
	var vector [12]int
	if json.Unmarshal(row[1], &runID) != nil || json.Unmarshal(row[2], &phase) != nil || json.Unmarshal(row[3], &rejected) != nil || json.Unmarshal(row[4], &terminal) != nil || json.Unmarshal(row[5], &vector) != nil || json.Unmarshal(row[6], &total) != nil || json.Unmarshal(row[7], &remaining) != nil || runID != run.RunID || phase != 2 || terminal != "budget-exhausted" || !digestTextUtility(rejected) || vector != run.WorkVector || total != run.WorkTotal || remaining != 0 {
		return fmt.Errorf("invalid work-terminal authority")
	}
	if len(run.Records) == 0 || run.Records[len(run.Records)-1].Code != 19 || len(run.Records[len(run.Records)-1].Outputs) != 1 || digestBytesText(run.Records[len(run.Records)-1].Outputs[0]) != run.WorkTerminal.Digest {
		return fmt.Errorf("work terminal is not the final charged output")
	}
	return nil
}

type verifiedObject struct {
	unit      *unit.Unit
	canonical []byte
	row       []json.RawMessage
}

func verifyCertificateAuthority(run SearchRun) error {
	objects := map[string]verifiedObject{}
	for _, name := range run.Store.All() {
		u := run.Store.Get(name)
		canonical := []byte(u.GetString("canonicalObject"))
		digest := u.GetString("objectDigest")
		if len(canonical) == 0 || !digestTextUtility(digest) {
			continue
		}
		if digestBytesText(canonical) != digest {
			return fmt.Errorf("Store object %s changed digest", name)
		}
		var row []json.RawMessage
		if json.Unmarshal(canonical, &row) != nil {
			return fmt.Errorf("Store object %s is not canonical JSON", name)
		}
		reencoded, _ := json.Marshal(row)
		if !bytes.Equal(reencoded, canonical) {
			return fmt.Errorf("Store object %s is not canonical", name)
		}
		objects[digest] = verifiedObject{unit: u, canonical: canonical, row: row}
	}
	roots := map[string]actionrelationexp.OperationRoot{}
	for _, root := range run.ProofRoots {
		if roots[root.Digest].Digest != "" || actionrelationexp.VerifyOperationRange(root, run.Transcript) != nil {
			return fmt.Errorf("invalid or duplicate certificate operation root")
		}
		roots[root.Digest] = root
	}
	callSequence := map[string]int{}
	for sequence, callID := range run.Transcript.CallIDs {
		callSequence[callID] = sequence
	}
	usedRoots := map[string]bool{}
	cacheRows := map[string]verifiedObject{}
	certificates := map[string]verifiedObject{}
	for digest, object := range objects {
		switch utilityRowTag(object.row) {
		case "certificate-cache-row/v3":
			cacheRows[digest] = object
		case "local-diamond-certificate/v1":
			certificates[digest] = object
		}
	}
	for cacheDigest, cache := range cacheRows {
		if len(cache.row) != 12 {
			return fmt.Errorf("invalid cache row length")
		}
		world, policy, state := utilityString(cache.row[1]), utilityString(cache.row[2]), utilityString(cache.row[3])
		minimum, maximum := utilityString(cache.row[4]), utilityString(cache.row[5])
		missCall, attemptDigest, rootDigest := utilityString(cache.row[6]), utilityString(cache.row[7]), utilityString(cache.row[8])
		result, certificateDigest, status := utilityString(cache.row[9]), utilityString(cache.row[10]), utilityString(cache.row[11])
		if world != run.WorldDigest || policy != string(run.Policy) || !digestTextUtility(state) || !digestTextUtility(minimum) || !digestTextUtility(maximum) || minimum >= maximum || status != "valid" {
			return fmt.Errorf("cache row changed pair authority")
		}
		attempt := objects[attemptDigest]
		if len(attempt.row) != 10 || utilityRowTag(attempt.row) != "local-diamond-certificate-attempt/v2" || utilityString(attempt.row[1]) != state || utilityString(attempt.row[6]) != rootDigest || utilityString(attempt.row[7]) != result || utilityString(attempt.row[8]) != certificateDigest || utilityString(attempt.row[9]) != "valid" {
			return fmt.Errorf("cache row does not resolve its certificate attempt")
		}
		aDigest, bDigest := utilityString(attempt.row[2]), utilityString(attempt.row[3])
		if slices.Min([]string{aDigest, bDigest}) != minimum || slices.Max([]string{aDigest, bDigest}) != maximum {
			return fmt.Errorf("cache min/max pair does not match oriented attempt")
		}
		root, ok := roots[rootDigest]
		if !ok {
			return fmt.Errorf("cache row has unresolved operation root")
		}
		start, count, err := operationRangeBounds(root)
		missSequence, callOK := callSequence[missCall]
		if err != nil || !callOK || missSequence < start || missSequence >= start+count || start+count >= len(run.Records) || run.Records[missSequence].Code != 18 || run.Records[start+count].Code != 25 || digestBytesText(firstOutput(run.Records[start+count])) != cacheDigest {
			return fmt.Errorf("cache row does not close its exact miss/proof range")
		}
		misses := 0
		var operationRows []string
		for sequence := start; sequence < start+count; sequence++ {
			record := run.Records[sequence]
			if record.Code == 18 {
				misses++
				if sequence != missSequence || record.Status != 1 || len(record.Outputs) != 0 {
					return fmt.Errorf("operation range contains wrong cache miss")
				}
			}
			if record.Code == 25 {
				return fmt.Errorf("operation range includes cache finalization")
			}
			if record.Code == 12 || record.Code == 13 || record.Code == 14 {
				if len(record.Outputs) == 0 {
					return fmt.Errorf("certificate proof call lacks output")
				}
				operationRows = append(operationRows, digestBytesText(record.Outputs[0]))
			}
		}
		var attemptRows []string
		if json.Unmarshal(attempt.row[5], &attemptRows) != nil || misses != 1 || !slices.Equal(operationRows, attemptRows) {
			return fmt.Errorf("certificate attempt operation rows do not reconstruct: range=%v attempt=%v misses=%d", operationRows, attemptRows, misses)
		}
		if result == "certified" {
			certificate := certificates[certificateDigest]
			if err := verifyCertificateObject(certificate, state, aDigest, bDigest, utilityString(attempt.row[4]), rootDigest); err != nil {
				return err
			}
		} else if result != "not-certified" || certificateDigest != utilityZeroDigest {
			return fmt.Errorf("invalid non-certified cache result")
		}
		usedRoots[rootDigest] = true
	}
	if len(usedRoots) != len(roots) {
		return fmt.Errorf("unreferenced certificate operation root")
	}
	for _, propagation := range run.Search.Propagations {
		row, err := utilityCanonicalRow(propagation.Canonical, 9, "sleep-propagation-core/v1")
		if err != nil || certificates[utilityString(row[6])].unit == nil {
			return fmt.Errorf("sleep propagation has unresolved certificate")
		}
	}
	for _, record := range run.Records {
		if record.Code == 18 && record.Status == 3 {
			if len(record.Outputs) != 1 || cacheRows[digestBytesText(record.Outputs[0])].unit == nil {
				return fmt.Errorf("cache-hit call has unresolved retained row")
			}
		}
	}
	return nil
}

func verifyCertificateObject(certificate verifiedObject, state, aDigest, bDigest, witnessDigest, rootDigest string) error {
	if len(certificate.row) != 10 || utilityRowTag(certificate.row) != "local-diamond-certificate/v1" || utilityString(certificate.row[1]) != state || utilityString(certificate.row[2]) != aDigest || utilityString(certificate.row[3]) != bDigest || utilityString(certificate.row[4]) != witnessDigest || utilityString(certificate.row[5]) != utilityString(certificate.row[6]) || utilityString(certificate.row[8]) != aDigest || utilityString(certificate.row[9]) != rootDigest {
		return fmt.Errorf("certificate object does not match its valid attempt")
	}
	var equal bool
	if json.Unmarshal(certificate.row[7], &equal) != nil || !equal {
		return fmt.Errorf("certificate object lacks equality authority")
	}
	return nil
}

func operationRangeBounds(root actionrelationexp.OperationRoot) (int, int, error) {
	row, err := utilityCanonicalRow(root.Canonical, 7, "actionrelation-operation-root/v1")
	if err != nil || utilityString(row[1]) != "range" {
		return 0, 0, fmt.Errorf("invalid operation range")
	}
	var start, count int
	if json.Unmarshal(row[4], &start) != nil || json.Unmarshal(row[5], &count) != nil || start < 0 || count < 1 {
		return 0, 0, fmt.Errorf("invalid operation range bounds")
	}
	return start, count, nil
}

func utilityCanonicalRow(canonical []byte, length int, tag string) ([]json.RawMessage, error) {
	var row []json.RawMessage
	if json.Unmarshal(canonical, &row) != nil || len(row) != length || utilityRowTag(row) != tag {
		return nil, fmt.Errorf("invalid %s row", tag)
	}
	reencoded, _ := json.Marshal(row)
	if !bytes.Equal(reencoded, canonical) {
		return nil, fmt.Errorf("noncanonical %s row", tag)
	}
	return row, nil
}

func utilityRowTag(row []json.RawMessage) string {
	if len(row) == 0 {
		return ""
	}
	return utilityString(row[0])
}

func utilityString(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func digestTextUtility(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 32 && hex.EncodeToString(raw) == value
}

func digestBytesText(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
