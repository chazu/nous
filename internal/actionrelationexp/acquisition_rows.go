package actionrelationexp

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationwire"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func BuildAcquisitionTables(run actionrelationacquire.Run) (map[uint16]TablePack, error) {
	experiment := run.Store.Get(run.Experiment)
	if experiment == nil {
		return nil, fmt.Errorf("missing experiment")
	}
	inputs := map[uint16][]string{
		101: experiment.GetStrings("literalRowUnits"),
		102: experiment.GetStrings("guardResultUnits"),
		103: experiment.GetStrings("candidateUnits"),
		104: experiment.GetStrings("edgeUnits"),
		105: experiment.GetStrings("observationUnits"),
		106: experiment.GetStrings("viewEvidenceUnits"),
		108: experiment.GetStrings("candidateResultUnits"),
	}
	scope := experiment.GetString("scope")
	if scope != "nous" && scope != "no-guard" {
		return nil, fmt.Errorf("invalid acquisition scope")
	}
	operationRecords, err := acquisitionOperationRecords(run)
	if err != nil {
		return nil, err
	}
	tables := map[uint16]TablePack{}
	kinds := []uint16{101, 102, 103, 104, 105, 106, 107, 108}
	if scope == "no-guard" {
		kinds = []uint16{102, 103, 105, 106, 107, 108}
	}
	for _, kind := range kinds {
		names := inputs[kind]
		records := make([][]byte, len(names))
		if kind == 107 {
			records = operationRecords
		}
		for ordinal, name := range names {
			u := run.Store.Get(name)
			if u == nil {
				return nil, fmt.Errorf("missing kind %d unit %q", kind, name)
			}
			var err error
			switch kind {
			case 101:
				records[ordinal], err = encodeLiteralRow(u)
			case 102:
				records[ordinal], err = encodeGuardResult(u)
			case 103:
				records[ordinal], err = encodeCandidate(run.Store, u)
			case 104:
				records[ordinal], err = encodeRefinement(u)
			case 105:
				records[ordinal], err = encodeObservation(u)
			case 106:
				records[ordinal], err = encodeViewEvidence(u)
			case 108:
				records[ordinal], err = encodeCandidateResult(run.Store, u)
			}
			if err != nil {
				return nil, fmt.Errorf("kind %d ordinal %d: %w", kind, ordinal, err)
			}
		}
		pack, err := BuildTablePack(kind, 0, records)
		if err != nil {
			return nil, err
		}
		tables[kind] = pack
	}
	return tables, nil
}

func BuildAcquisitionTableBundles(run actionrelationacquire.Run, curriculum int) (map[uint16]TableBundle, error) {
	root, _ := EvidenceRoot("development")
	return BuildAcquisitionTableBundlesAt(root, run, curriculum)
}

func BuildAcquisitionTableBundlesAt(evidenceRoot string, run actionrelationacquire.Run, curriculum int) (map[uint16]TableBundle, error) {
	tables, err := BuildAcquisitionTables(run)
	if err != nil {
		return nil, err
	}
	experiment := run.Store.Get(run.Experiment)
	if experiment == nil {
		return nil, fmt.Errorf("missing acquisition experiment")
	}
	scope := experiment.GetString("scope")
	bundles := make(map[uint16]TableBundle, len(tables))
	for kind, table := range tables {
		recordSize := tableRecordSizes[kind]
		count := int(table.LastOrdinal-table.FirstOrdinal) + 1
		records := make([][]byte, count)
		for ordinal := range records {
			start := len(TableHeader) + ordinal*recordSize
			records[ordinal] = table.Bytes[start : start+recordSize]
		}
		bundle, err := BuildTableBundleAt(evidenceRoot, curriculum, scope, kind, records)
		if err != nil {
			return nil, fmt.Errorf("kind %d: %w", kind, err)
		}
		bundles[kind] = bundle
	}
	return bundles, nil
}

type AcquisitionEvidence struct {
	Run        actionrelationacquire.Run
	Tables     map[uint16]TableBundle
	Transcript AcquisitionTranscript
}

func CompleteAcquisition(session *actionrelationacquire.Session, curriculum int) (AcquisitionEvidence, error) {
	return CompleteAcquisitionFor(session, curriculum, "development", PlanCommit)
}

func CompleteNoGuardAcquisition(session *actionrelationacquire.Session, curriculum int) (AcquisitionEvidence, error) {
	return CompleteNoGuardAcquisitionFor(session, curriculum, "development", PlanCommit)
}

func CompleteAcquisitionFor(session *actionrelationacquire.Session, curriculum int, panel, authority string) (AcquisitionEvidence, error) {
	return completeAcquisitionFor(session, curriculum, panel, authority, "nous")
}

func CompleteNoGuardAcquisitionFor(session *actionrelationacquire.Session, curriculum int, panel, authority string) (AcquisitionEvidence, error) {
	return completeAcquisitionFor(session, curriculum, panel, authority, "no-guard")
}

func completeAcquisitionFor(session *actionrelationacquire.Session, curriculum int, panel, authority, scope string) (AcquisitionEvidence, error) {
	if session == nil || session.Scope != scope {
		return AcquisitionEvidence{}, fmt.Errorf("acquisition session scope mismatch")
	}
	partial, err := session.Snapshot()
	if err != nil {
		return AcquisitionEvidence{}, err
	}
	evidenceRoot, rootErr := EvidenceRoot(panel)
	if rootErr != nil {
		session.Abort()
		return AcquisitionEvidence{}, rootErr
	}
	tables, err := BuildAcquisitionTableBundlesAt(evidenceRoot, partial, curriculum)
	if err != nil {
		session.Abort()
		return AcquisitionEvidence{}, err
	}
	manifestDigest := func(kind uint16) (string, error) {
		bundle, ok := tables[kind]
		if !ok {
			return "", fmt.Errorf("missing acquisition table %d", kind)
		}
		return canonicalDigest(bundle.Manifest.CanonicalJSON())
	}
	edgeRoot := zeroIfEmpty("")
	var evaluationRoots []string
	if scope == "nous" {
		edgeRoot, err = manifestDigest(104)
		if err != nil {
			session.Abort()
			return AcquisitionEvidence{}, err
		}
		evaluationOne, oneErr := manifestDigest(101)
		if oneErr != nil {
			session.Abort()
			return AcquisitionEvidence{}, oneErr
		}
		evaluationRoots = append(evaluationRoots, evaluationOne)
	}
	evaluationTwo, err := manifestDigest(102)
	if err != nil {
		session.Abort()
		return AcquisitionEvidence{}, err
	}
	evaluationRoots = append(evaluationRoots, evaluationTwo)
	experiment := partial.Store.Get(partial.Experiment)
	candidateLeaves := tables[103].LeafDigests
	resultLeaves := tables[108].LeafDigests
	wantCandidates := 451
	if scope == "no-guard" {
		wantCandidates = 1
	}
	if experiment == nil || len(candidateLeaves) != wantCandidates || len(resultLeaves) != wantCandidates {
		session.Abort()
		return AcquisitionEvidence{}, fmt.Errorf("acquisition leaf cardinality mismatch")
	}
	for ordinal, name := range experiment.GetStrings("candidateUnits") {
		partial.Store.Get(name).Set("tableLeafDigest", candidateLeaves[ordinal])
	}
	for ordinal, name := range experiment.GetStrings("candidateResultUnits") {
		partial.Store.Get(name).Set("tableLeafDigest", resultLeaves[ordinal])
	}
	winnerLeaves := make([]string, len(experiment.GetStrings("winnerResultUnits")))
	for index, name := range experiment.GetStrings("winnerResultUnits") {
		result := partial.Store.Get(name)
		if result == nil || result.GetInt("ordinal") < 0 || result.GetInt("ordinal") >= len(resultLeaves) {
			session.Abort()
			return AcquisitionEvidence{}, fmt.Errorf("invalid acquisition winner")
		}
		winnerLeaves[index] = resultLeaves[result.GetInt("ordinal")]
	}
	run, err := session.BindEvidence(actionrelationacquire.EvidenceRoots{
		CandidateLeaves: candidateLeaves, EdgeTableRoot: edgeRoot,
		EvaluationTableRoots: evaluationRoots, WinnerLeaves: winnerLeaves,
	})
	if err != nil {
		return AcquisitionEvidence{}, err
	}
	runID, err := AcquisitionRunID(panel, authority, curriculum, scope)
	if err != nil {
		return AcquisitionEvidence{}, err
	}
	if runID != session.RunID {
		return AcquisitionEvidence{}, fmt.Errorf("acquisition run authority changed after reservation")
	}
	transcript, err := BuildAcquisitionTranscriptAt(evidenceRoot, run, tables, runID)
	if err != nil {
		return AcquisitionEvidence{}, err
	}
	observationNames := run.Store.Get(run.Experiment).GetStrings("observationUnits")
	observationRecords := make([][]byte, len(observationNames))
	for ordinal, name := range observationNames {
		observationRecords[ordinal], err = encodeObservation(run.Store.Get(name))
		if err != nil {
			return AcquisitionEvidence{}, fmt.Errorf("rebuild observation table %d: %w", ordinal, err)
		}
	}
	tables[105], err = BuildTableBundleAt(evidenceRoot, curriculum, scope, 105, observationRecords)
	if err != nil {
		return AcquisitionEvidence{}, err
	}
	if err := retainAcquisitionTranscriptAuthority(run.Store, run.Experiment, transcript); err != nil {
		return AcquisitionEvidence{}, err
	}
	return AcquisitionEvidence{Run: run, Tables: tables, Transcript: transcript}, nil
}

func retainAcquisitionTranscriptAuthority(store *unit.Store, experimentName string, transcript AcquisitionTranscript) error {
	experiment := store.Get(experimentName)
	if experiment == nil {
		return fmt.Errorf("missing acquisition experiment")
	}
	reservationNames := experiment.GetStrings("reservationUnits")
	if experiment.GetString("runID") != transcript.RunID || len(reservationNames) != len(transcript.Reservations) {
		return fmt.Errorf("missing pre-execution acquisition reservations")
	}
	for index, reservation := range transcript.Reservations {
		if err := VerifyWorkReservation(reservation, acquisitionLifecycleCap); err != nil {
			return err
		}
		name := fmt.Sprintf("AR.Reservation.%s.%05d", transcript.RunID, index)
		u := store.Get(name)
		if reservationNames[index] != name || u == nil || u.GetString("canonicalObject") != string(reservation.Canonical) || u.GetString("objectDigest") != reservation.Digest {
			return fmt.Errorf("pre-execution acquisition reservation %d changed", index)
		}
	}
	rootNames := make([]string, 0, len(transcript.ObservationRoots)+1)
	for index, root := range append(append([]OperationRoot{}, transcript.ObservationRoots...), transcript.RunRoot) {
		if root.Digest != shaHex(root.Canonical) || ValidateObject(46, root.Canonical) != nil {
			return fmt.Errorf("invalid retained operation root %d", index)
		}
		name := fmt.Sprintf("AR.OperationRoot.%s.%03d", transcript.RunID, index)
		u := unit.New(name)
		u.Set("isA", []string{"ActionRelationOperationRoot", "Anything"})
		u.Set("canonicalObject", string(root.Canonical))
		u.Set("objectDigest", root.Digest)
		store.Put(u)
		rootNames = append(rootNames, name)
	}
	journalRoot, _ := transcript.Transcript.JournalRoot.Digest()
	inputRoot, _ := transcript.Transcript.InputRoot.Digest()
	detailRoot, _ := transcript.Transcript.DetailRoot.Digest()
	experiment.Set("runID", transcript.RunID)
	experiment.Set("operationRootUnits", rootNames)
	experiment.Set("runOperationRoot", transcript.RunRoot.Digest)
	experiment.Set("journalRoot", journalRoot)
	experiment.Set("inputRoot", inputRoot)
	experiment.Set("detailRoot", detailRoot)
	return nil
}

func encodeViewEvidence(u *unit.Unit) ([]byte, error) {
	record := make([]byte, 512)
	for _, field := range []struct {
		start int
		name  string
	}{
		{0, "viewDigest"},
		{32, "observationDigest"},
		{64, "semanticWorldDigest"},
		{96, "normalizationProofDigest"},
		{128, "originalStateDigest"},
		{160, "originalActionsRoot"},
		{192, "occurrenceMapRoot"},
	} {
		if !putDigest(record[field.start:field.start+32], u.GetString(field.name)) {
			return nil, fmt.Errorf("invalid view-evidence %s", field.name)
		}
	}
	bank, cellCount, actionCount := u.GetInt("bank"), u.GetInt("cellCount"), u.GetInt("actionCount")
	if bank < 0 || bank > 1 || cellCount < 1 || cellCount > 3 || actionCount < 1 || actionCount > 8 {
		return nil, fmt.Errorf("invalid view-evidence dimensions")
	}
	record[224], record[225], record[226], record[227] = byte(bank), byte(cellCount), byte(actionCount), 1
	return record, nil
}

func encodeObservation(u *unit.Unit) ([]byte, error) {
	record := make([]byte, 512)
	var row []any
	if json.Unmarshal([]byte(u.GetString("canonicalObject")), &row) != nil || len(row) != 11 || row[0] != "action-pair-observation/v1" {
		return nil, fmt.Errorf("invalid observation wire")
	}
	label, ok := row[10].(string)
	if !ok {
		return nil, fmt.Errorf("invalid observation label")
	}
	record[0], record[1], record[2] = 1, observationLabelCode(label), 1
	if record[1] == 0 {
		return nil, fmt.Errorf("unknown observation label")
	}
	positions := [6][2]int{{4, 100}, {5, 132}, {6, 164}, {7, 196}, {8, 228}, {9, 260}}
	if !putDigest(record[4:36], anyString(row[1])) || !putDigest(record[36:68], anyString(row[2])) || !putDigest(record[68:100], anyString(row[3])) {
		return nil, fmt.Errorf("invalid observation identity")
	}
	for bit, position := range positions {
		if row[position[0]] == nil {
			record[3] |= 1 << bit
			continue
		}
		if !putDigest(record[position[1]:position[1]+32], anyString(row[position[0]])) {
			return nil, fmt.Errorf("invalid observation component %d", bit)
		}
	}
	if record[3]&3 != 0 || !putDigest(record[292:324], u.GetString("operationRoot")) {
		return nil, fmt.Errorf("invalid observation nulls/root")
	}
	return record, nil
}

func acquisitionOperationRecords(run actionrelationacquire.Run) ([][]byte, error) {
	var records [][]byte
	for index, meter := range run.MeterRecords {
		if meter.Code != 4 && meter.Code != 5 && meter.Code != 6 || len(meter.Outputs) == 0 {
			continue
		}
		record, err := encodeOperationRow(meter.Outputs[0])
		if err != nil {
			return nil, fmt.Errorf("meter record %d: %w", index, err)
		}
		records = append(records, record)
	}
	return records, nil
}

func encodeOperationRow(wire []byte) ([]byte, error) {
	record := make([]byte, 256)
	var row []any
	if json.Unmarshal(wire, &row) != nil || len(row) < 4 {
		return nil, fmt.Errorf("invalid operation wire")
	}
	record[0], record[2] = 1, 1
	switch row[0] {
	case "action-applicability-row/v1":
		if len(row) != 5 || anyString(row[4]) != "valid" || !putDigest(record[4:36], anyString(row[1])) || !putDigest(record[36:68], anyString(row[2])) {
			return nil, fmt.Errorf("invalid applicability wire")
		}
		record[1] = 1
		result, ok := row[3].(bool)
		if !ok {
			return nil, fmt.Errorf("invalid applicability result")
		}
		record[3] = boolByte(result)
	case "action-transition-row/v1":
		if len(row) != 6 || !putDigest(record[4:36], anyString(row[1])) || !putDigest(record[36:68], anyString(row[2])) {
			return nil, fmt.Errorf("invalid transition wire")
		}
		record[1] = 2
		outcome := anyString(row[5])
		if outcome == "applied" {
			record[3] = 1
			if !putDigest(record[68:100], anyString(row[4])) {
				return nil, fmt.Errorf("invalid applied output")
			}
		} else if outcome == "inapplicable" && anyString(row[4]) == zeroIfEmpty("") {
			record[3] = 2
		} else {
			return nil, fmt.Errorf("invalid transition outcome")
		}
	case "action-state-equality-row/v1":
		if len(row) != 5 || anyString(row[4]) != "valid" || !putDigest(record[4:36], anyString(row[1])) || !putDigest(record[36:68], anyString(row[2])) {
			return nil, fmt.Errorf("invalid equality wire")
		}
		record[1] = 3
		result, ok := row[3].(bool)
		if !ok {
			return nil, fmt.Errorf("invalid equality result")
		}
		record[3] = boolByte(result)
	default:
		return nil, fmt.Errorf("unknown operation wire")
	}
	return record, nil
}

func observationLabelCode(label string) byte {
	labels := []string{"", "commutes", "a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable", "conflicts", "invalid"}
	for index, candidate := range labels {
		if label == candidate {
			return byte(index)
		}
	}
	return 0
}

func anyString(value any) string {
	text, _ := value.(string)
	return text
}

func encodeLiteralRow(u *unit.Unit) ([]byte, error) {
	record := make([]byte, 128)
	if !putDigest(record[0:32], u.GetString("guardDigest")) || !putDigest(record[32:64], u.GetString("observationDigest")) || !putDigest(record[96:128], factPairRoot(u.GetString("aFactsDigest"), u.GetString("bFactsDigest"))) {
		return nil, fmt.Errorf("invalid literal digest")
	}
	atom := actionrelationsAtomCode(u.GetString("atom"))
	if atom == 0 {
		return nil, fmt.Errorf("invalid atom")
	}
	binary.BigEndian.PutUint16(record[64:66], atom)
	record[66] = boolByte(u.GetBool("polarity"))
	record[67] = boolByte(u.GetBool("result"))
	record[68] = 1
	return record, nil
}

func encodeGuardResult(u *unit.Unit) ([]byte, error) {
	record := make([]byte, 96)
	if !putDigest(record[0:32], u.GetString("guardDigest")) || !putDigest(record[32:64], u.GetString("observationDigest")) {
		return nil, fmt.Errorf("invalid guard-result digest")
	}
	record[64], record[65] = boolByte(u.GetBool("result")), 1
	return record, nil
}

func encodeCandidate(store *unit.Store, u *unit.Unit) ([]byte, error) {
	record := make([]byte, 128)
	guard, guardErr := actionrelations.ParseGuard([]byte(u.GetString("guard")))
	pattern, patternErr := actionrelations.ParsePattern([]byte(u.GetString("pattern")))
	if guardErr != nil || patternErr != nil {
		return nil, fmt.Errorf("invalid candidate semantics")
	}
	guardDigest, _ := guard.Digest()
	patternDigest, _ := pattern.Digest()
	if !putDigest(record[0:32], guardDigest) || !putDigest(record[64:96], patternDigest) {
		return nil, fmt.Errorf("invalid candidate digest")
	}
	if parent := u.GetString("parentCandidate"); parent != "" {
		parentUnit := store.Get(parent)
		if parentUnit == nil || !putDigest(record[32:64], parentUnit.GetString("objectDigest")) {
			return nil, fmt.Errorf("invalid candidate parent")
		}
	}
	ordinal, literals := u.GetInt("ordinal"), u.GetInt("literalCount")
	if ordinal < 0 || ordinal > 450 || literals < 0 || literals > 2 {
		return nil, fmt.Errorf("invalid candidate ordinal/literals")
	}
	binary.BigEndian.PutUint16(record[96:98], uint16(ordinal))
	record[98], record[99] = byte(literals), 1
	return record, nil
}

func encodeRefinement(u *unit.Unit) ([]byte, error) {
	record := make([]byte, 96)
	parent, parentErr := actionrelations.ParseGuard([]byte(u.GetString("parentGuard")))
	child, childErr := actionrelations.ParseGuard([]byte(u.GetString("childGuard")))
	if parentErr != nil || childErr != nil {
		return nil, fmt.Errorf("invalid refinement guards")
	}
	parentDigest, _ := parent.Digest()
	childDigest, _ := child.Digest()
	if !putDigest(record[0:32], parentDigest) || !putDigest(record[32:64], childDigest) {
		return nil, fmt.Errorf("invalid refinement digest")
	}
	atom := actionrelationsAtomCode(u.GetString("atom"))
	ordinal := u.GetInt("ordinal")
	if atom == 0 || ordinal < 0 || ordinal > 449 {
		return nil, fmt.Errorf("invalid refinement atom/ordinal")
	}
	binary.BigEndian.PutUint16(record[64:66], atom)
	record[66], record[67] = boolByte(u.GetBool("polarity")), 1
	binary.BigEndian.PutUint32(record[68:72], uint32(ordinal))
	return record, nil
}

func encodeCandidateResult(store *unit.Store, u *unit.Unit) ([]byte, error) {
	record := make([]byte, 128)
	candidate := store.Get(u.GetString("candidate"))
	if candidate == nil || !putDigest(record[0:32], candidate.GetString("objectDigest")) {
		return nil, fmt.Errorf("invalid result candidate")
	}
	guardResultDigests := make([]string, 0, len(u.GetStrings("guardResults")))
	for _, name := range u.GetStrings("guardResults") {
		row := store.Get(name)
		if row == nil {
			return nil, fmt.Errorf("missing guard result")
		}
		guardResultDigests = append(guardResultDigests, row.GetString("objectDigest"))
	}
	vectorRoot, err := actionrelationwire.RootDigest("guard-result-vector", guardResultDigests)
	if err != nil || !putDigest(record[32:64], vectorRoot) {
		return nil, fmt.Errorf("invalid guard-result vector")
	}
	positive, negative := u.GetInt("positiveCoverage"), u.GetInt("negativeCoverage")
	if positive < 0 || positive > 16 || negative < 0 || negative > 16 {
		return nil, fmt.Errorf("invalid result coverage")
	}
	record[64], record[65] = byte(positive), byte(negative)
	record[66], record[67], record[68] = boolByte(u.GetBool("wrapperCoverageComplete")), boolByte(u.GetBool("eligible")), 1
	return record, nil
}

func actionrelationsAtomCode(atom string) uint16 {
	for index, candidate := range actionrelations.Atoms {
		if atom == candidate {
			return uint16(index + 1)
		}
	}
	return 0
}

func factPairRoot(left, right string) string {
	root, _ := actionrelationwire.RootDigest("local-fact-pair", []string{left, right})
	return root
}

func putDigest(target []byte, value string) bool {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 || len(target) != 32 {
		return false
	}
	copy(target, decoded)
	return true
}

func boolByte(value bool) byte {
	if value {
		return 1
	}
	return 0
}
