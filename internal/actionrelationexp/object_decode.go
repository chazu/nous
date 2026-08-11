package actionrelationexp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationledger"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const zeroObjectDigest = "0000000000000000000000000000000000000000000000000000000000000000"

type decodedTruthPair struct {
	State string
	A     string
	B     string
	Label string
}

type decodedTruthShard struct {
	World     string
	Ordinal   int
	Count     int
	Terminals []string
	Pairs     []decodedTruthPair
}

type decodedWorldPolicyRow struct {
	Panel         string
	Curriculum    int
	Family        int
	WorldOrdinal  int
	Stratum       string
	World         string
	Policy        string
	Terminal      string
	Work          [12]int
	WorkTotal     int
	Matches       [8]int
	Certificates  [4]int
	SleepCount    int
	HistoryCount  int
	TerminalSet   string
	WorkTerminal  string
	BehaviorEqual bool
	Remaining     int
	OperationRoot string
}

type decodedCurriculumPolicyRow struct {
	Panel               string
	Curriculum          int
	Family              int
	Policy              string
	Acquisition         string
	Artifact            string
	AcquisitionWork     [12]int
	AcquisitionTerminal string
	WorldRows           []string
	Terminal            string
	SearchWork          [12]int
	SearchTotal         int
	LifecycleWork       [12]int
	LifecycleTotal      int
	BehaviorEqual       bool
	Remaining           int
	OperationRoot       string
}

type decodedStoreBoundary struct {
	Curriculum int
	Scope      string
	Tables     []string
	ObjectSet  string
	IndexRoot  string
}

type decodedFixture struct {
	Panel          string
	Curriculum     int
	Family         int
	Accepted       int
	Training       []string
	Views          []string
	Worlds         []string
	AttemptLedgers []string
}

type decodedAttemptLedger struct {
	Panel      string
	Authority  string
	Curriculum int
	Seed       json.RawMessage
	Attempt    int
	Draws      []json.RawMessage
	DrawRoot   string
	Phases     []json.RawMessage
	TotalWork  int
	Terminal   string
}

type decodedOperationRoot struct {
	Variant  string
	RunID    string
	Phase    uint8
	Start    uint32
	Count    int
	CallRoot string
	Context  string
	Children []string
}

type decodedCertificateAttempt struct {
	State       string
	A           string
	B           string
	Witness     string
	Operation   string
	Result      string
	Certificate string
	Status      string
}

type decodedWorkTerminal struct {
	RunID     string
	Phase     int
	Rejected  string
	Terminal  string
	Work      [12]int
	Total     int
	Remaining int
}

func validateTypedObject(kind uint16, data []byte, row []json.RawMessage) error {
	switch kind {
	case 1:
		_, err := actionrelations.ParseState(data)
		return err
	case 2:
		_, err := actionrelations.ParseSemanticAction(data)
		return err
	case 3:
		_, err := actionrelations.ParseOccurrence(data)
		return err
	case 4:
		return validateWorldCore(data, row)
	case 5:
		return validateRemaining(row)
	case 6:
		_, err := actionrelations.ParsePattern(data)
		return err
	case 7:
		_, err := actionrelations.ParseGuard(data)
		return err
	case 8:
		_, err := actionrelations.ParseLocalFacts(data)
		return err
	case 9:
		_, err := actionrelations.ParseRelation(data)
		return err
	case 10:
		_, err := actionrelations.ParseArtifact(data)
		return err
	case 11:
		return validateDigestTuple(row, 3)
	case 12:
		return validatePresentationView(row)
	case 13:
		return validateNormalizationProof(row)
	case 14:
		return validateWitness(row, 2, "")
	case 15:
		return validateWitness(row, 2, "")
	case 16:
		return validateWitness(row, 3, "all-pairs")
	case 17:
		return validateCertificate(row)
	case 18:
		return validatePropagation(row)
	case 19:
		return validateProofMap(row)
	case 20:
		return validateDigestTuple(row, 4)
	case 21:
		return validateSearchEdge(row)
	case 22:
		return validateCompletedSubtree(row)
	case 23:
		return validateTerminal(row)
	case 24:
		return validateTerminalSet(row)
	case 25:
		return validateSubtreeRoot(row)
	case 26:
		return validateCacheRow(row)
	case 27:
		_, err := decodeReservation(data)
		return err
	case 28:
		return validateSearchBarrier(row)
	case 29:
		_, err := decodeTruthShard(row)
		return err
	case 32:
		_, err := decodeWorldPolicyRow(row)
		return err
	case 33:
		_, err := decodeCurriculumPolicyRow(row)
		return err
	case 35:
		_, err := decodeStoreBoundary(row)
		return err
	case 36:
		_, err := decodeAttemptLedger(row)
		return err
	case 37:
		return validateValidityRow(row)
	case 38:
		return validateApplicabilityRow(row)
	case 39:
		return validateTransitionRow(row)
	case 40:
		return validateEqualityRow(row)
	case 41:
		return validateLiteralRow(row)
	case 42:
		return validateMatchRow(row)
	case 43:
		return validateUnanimousUse(row)
	case 44:
		_, err := decodeCertificateAttempt(row)
		return err
	case 45:
		return validateRawInput(row)
	case 46:
		_, err := decodeOperationRoot(row)
		return err
	case 47:
		_, err := decodeFixture(row)
		return err
	case 48:
		return validateStaticFootprint(row)
	case 49:
		_, err := decodeWorkTerminal(row)
		return err
	default:
		return nil
	}
}

func validateCompletedSubtree(row []json.RawMessage) error {
	var tag, parent, taken, edge, subtree, terminalSet, status string
	if len(row) != 7 || decode(row[0], &tag) || tag != objectKinds[22] || decode(row[1], &parent) || decode(row[2], &taken) || decode(row[3], &edge) || decode(row[4], &subtree) || decode(row[5], &terminalSet) || decode(row[6], &status) || !digestText(parent) || !digestText(taken) || !digestText(edge) || !digestText(subtree) || !digestText(terminalSet) || status != "completed" {
		return fmt.Errorf("invalid completed subtree")
	}
	return nil
}

func validateSubtreeRoot(row []json.RawMessage) error {
	var tag, node string
	var completions []string
	if len(row) != 3 || decode(row[0], &tag) || tag != objectKinds[25] || decode(row[1], &node) || decode(row[2], &completions) || !digestText(node) || len(completions) > actionrelations.MaxActions || !uniqueDigestList(completions) {
		return fmt.Errorf("invalid subtree root")
	}
	return nil
}

func decodeCertificateAttempt(row []json.RawMessage) (decodedCertificateAttempt, error) {
	v := decodedCertificateAttempt{}
	var tag string
	if len(row) != 9 || decode(row[0], &tag) || tag != objectKinds[44] || decode(row[1], &v.State) || decode(row[2], &v.A) || decode(row[3], &v.B) || decode(row[4], &v.Witness) || decode(row[5], &v.Operation) || decode(row[6], &v.Result) || decode(row[7], &v.Certificate) || decode(row[8], &v.Status) || !digestText(v.State) || !digestText(v.A) || !digestText(v.B) || v.A == v.B || !digestText(v.Witness) || !digestText(v.Operation) || !digestText(v.Certificate) {
		return decodedCertificateAttempt{}, fmt.Errorf("invalid certificate attempt")
	}
	if v.Result == "certified" {
		if v.Status != "valid" || v.Certificate == zeroObjectDigest {
			return decodedCertificateAttempt{}, fmt.Errorf("invalid certified attempt")
		}
	} else if v.Result == "not-certified" {
		if v.Status != "valid" || v.Certificate != zeroObjectDigest {
			return decodedCertificateAttempt{}, fmt.Errorf("invalid negative attempt")
		}
	} else if v.Result == "invalid" {
		if v.Status != "invalid-input" || v.Certificate != zeroObjectDigest {
			return decodedCertificateAttempt{}, fmt.Errorf("invalid invalid-input attempt")
		}
	} else {
		return decodedCertificateAttempt{}, fmt.Errorf("unknown certificate attempt result")
	}
	return v, nil
}

func validateWorldCore(data []byte, row []json.RawMessage) error {
	if len(row) != 3 {
		return fmt.Errorf("invalid world-core shape")
	}
	state, err := actionrelations.ParseState(row[1])
	if err != nil {
		return fmt.Errorf("invalid world-core state")
	}
	var actions []json.RawMessage
	if json.Unmarshal(row[2], &actions) != nil || len(actions) < 1 || len(actions) > actionrelations.MaxActions {
		return fmt.Errorf("invalid world-core actions")
	}
	semantic := make([]actionrelations.SemanticAction, len(actions))
	for index, raw := range actions {
		semantic[index], err = actionrelations.ParseSemanticAction(raw)
		if err != nil {
			return fmt.Errorf("invalid world-core action %d", index)
		}
		if index > 0 && bytes.Compare(actions[index-1], raw) > 0 {
			return fmt.Errorf("unordered world-core actions")
		}
	}
	want, err := (actionrelations.NormalizedWorld{State: state, Actions: semantic}).CanonicalJSON()
	if err != nil || !bytes.Equal(want, data) {
		return fmt.Errorf("world core does not reconstruct")
	}
	return nil
}

func validateRemaining(row []json.RawMessage) error {
	var values []string
	if len(row) != 2 || decode(row[1], &values) || len(values) > actionrelations.MaxActions || !sortedUniqueDigestList(values) {
		return fmt.Errorf("invalid remaining occurrence set")
	}
	return nil
}

func validateDigestTuple(row []json.RawMessage, length int) error {
	if len(row) != length {
		return fmt.Errorf("invalid digest tuple length")
	}
	for index := 1; index < len(row); index++ {
		var value string
		if decode(row[index], &value) || !digestText(value) {
			return fmt.Errorf("invalid digest tuple field %d", index)
		}
	}
	return nil
}

func validatePresentationView(row []json.RawMessage) error {
	if len(row) != 5 {
		return fmt.Errorf("invalid presentation view length")
	}
	state, err := actionrelations.ParseState(row[1])
	if err != nil {
		return fmt.Errorf("invalid presentation state")
	}
	var actions []json.RawMessage
	var world string
	var mapping [][]json.RawMessage
	if json.Unmarshal(row[2], &actions) != nil || len(actions) < 1 || len(actions) > actionrelations.MaxActions || decode(row[3], &world) || !digestText(world) || json.Unmarshal(row[4], &mapping) != nil || len(mapping) != len(actions) {
		return fmt.Errorf("invalid presentation view fields")
	}
	concrete := make([]actionrelations.Action, len(actions))
	for index, raw := range actions {
		concrete[index], err = actionrelations.ParseAction(raw)
		if err != nil || len(mapping[index]) != 2 {
			return fmt.Errorf("invalid presentation action %d", index)
		}
		var name, occurrence string
		if decode(mapping[index][0], &name) || name != concrete[index].Name || decode(mapping[index][1], &occurrence) || !digestText(occurrence) {
			return fmt.Errorf("invalid presentation mapping %d", index)
		}
	}
	normalized, err := (actionrelations.World{State: state, Actions: concrete}).Normalize()
	if err != nil {
		return fmt.Errorf("presentation world does not normalize")
	}
	digest, _ := normalized.Digest()
	if digest != world {
		return fmt.Errorf("presentation semantic world changed")
	}
	return nil
}

func validateNormalizationProof(row []json.RawMessage) error {
	var view, world string
	var mappings [][]json.RawMessage
	if len(row) != 4 || decode(row[1], &view) || !digestText(view) || json.Unmarshal(row[2], &mappings) != nil || len(mappings) < 1 || len(mappings) > actionrelations.MaxCells || decode(row[3], &world) || !digestText(world) {
		return fmt.Errorf("invalid normalization proof")
	}
	previous := ""
	seenRoles := map[string]bool{}
	for index, mapping := range mappings {
		var original, role string
		if len(mapping) != 2 || decode(mapping[0], &original) || decode(mapping[1], &role) || original == "" || len(role) != 2 || role[0] != 'c' || role[1] < '0' || role[1] > '2' || index > 0 && original <= previous || seenRoles[role] {
			return fmt.Errorf("invalid normalization mapping %d", index)
		}
		seenRoles[role], previous = true, original
	}
	return nil
}

func validateWitness(row []json.RawMessage, length int, fixed string) error {
	if len(row) != length {
		return fmt.Errorf("invalid witness length")
	}
	if fixed != "" {
		var value string
		if decode(row[1], &value) || value != fixed {
			return fmt.Errorf("invalid witness variant")
		}
	}
	var digest string
	if decode(row[length-1], &digest) || !digestText(digest) {
		return fmt.Errorf("invalid witness digest")
	}
	return nil
}

func validateCertificate(row []json.RawMessage) error {
	if len(row) != 10 {
		return fmt.Errorf("invalid certificate length")
	}
	for _, index := range []int{1, 2, 3, 4, 5, 6, 8, 9} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid certificate digest %d", index)
		}
	}
	var result bool
	var a, representative string
	if decode(row[7], &result) || !result || decode(row[2], &a) || decode(row[8], &representative) || representative != a {
		return fmt.Errorf("invalid certificate result")
	}
	return nil
}

func validatePropagation(row []json.RawMessage) error {
	if len(row) != 9 {
		return fmt.Errorf("invalid propagation length")
	}
	for _, index := range []int{1, 2, 3, 5, 6, 7, 8} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid propagation digest %d", index)
		}
	}
	var taken, sleeper, source string
	if decode(row[2], &taken) || decode(row[3], &sleeper) || taken == sleeper || decode(row[4], &source) || source != "earlier-sibling" && source != "prior-sleep" {
		return fmt.Errorf("invalid propagation orientation")
	}
	return nil
}

func validateProofMap(row []json.RawMessage) error {
	var pairs [][]json.RawMessage
	if len(row) != 2 || json.Unmarshal(row[1], &pairs) != nil || len(pairs) > actionrelations.MaxActions {
		return fmt.Errorf("invalid proof map")
	}
	previous := ""
	for index, pair := range pairs {
		var sleeper, proof string
		if len(pair) != 2 || decode(pair[0], &sleeper) || !digestText(sleeper) || decode(pair[1], &proof) || !digestText(proof) || index > 0 && sleeper <= previous {
			return fmt.Errorf("invalid proof map row %d", index)
		}
		previous = sleeper
	}
	return nil
}

func validateSearchEdge(row []json.RawMessage) error {
	var parent, taken, child string
	var propagations []string
	if len(row) != 5 || decode(row[1], &parent) || !digestText(parent) || decode(row[2], &taken) || !digestText(taken) || decode(row[3], &propagations) || len(propagations) > actionrelations.MaxActions-1 || !uniqueDigestList(propagations) || decode(row[4], &child) || !digestText(child) {
		return fmt.Errorf("invalid search edge")
	}
	return nil
}

func validateTerminal(row []json.RawMessage) error {
	if len(row) != 4 {
		return fmt.Errorf("invalid terminal length")
	}
	if _, err := actionrelations.ParseState(row[1]); err != nil {
		return fmt.Errorf("invalid terminal state")
	}
	var remaining []string
	var terminal string
	if decode(row[2], &remaining) || len(remaining) > actionrelations.MaxActions || !sortedUniqueDigestList(remaining) || decode(row[3], &terminal) || terminal != "complete" && terminal != "deadlock" || (len(remaining) == 0) != (terminal == "complete") {
		return fmt.Errorf("invalid terminal semantics")
	}
	return nil
}

func validateTerminalSet(row []json.RawMessage) error {
	var terminals []string
	if len(row) != 2 || decode(row[1], &terminals) || !sortedUniqueDigestList(terminals) {
		return fmt.Errorf("invalid terminal set")
	}
	return nil
}

func validateCacheRow(row []json.RawMessage) error {
	if len(row) != 12 {
		return fmt.Errorf("invalid cache row length")
	}
	var policy, minimum, maximum, result, certificate, status string
	for _, index := range []int{1, 3, 4, 5, 6, 7, 8, 10} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid cache digest %d", index)
		}
	}
	if decode(row[2], &policy) || !onePolicy(policy) || decode(row[4], &minimum) || decode(row[5], &maximum) || minimum >= maximum || decode(row[9], &result) || decode(row[10], &certificate) || decode(row[11], &status) || status != "valid" || result != "certified" && result != "not-certified" || (result == "certified") != (certificate != zeroObjectDigest) {
		return fmt.Errorf("invalid cache result")
	}
	return nil
}

func validateSearchBarrier(row []json.RawMessage) error {
	var candidates, evaluations, winners []string
	var edgeRoot, status string
	if len(row) != 6 || decode(row[1], &candidates) || len(candidates) < 1 || len(candidates) > 512 || !uniqueDigestList(candidates) || decode(row[2], &edgeRoot) || !digestText(edgeRoot) || decode(row[3], &evaluations) || !uniqueDigestList(evaluations) || decode(row[4], &winners) || !uniqueDigestList(winners) || decode(row[5], &status) || status != "completed" {
		return fmt.Errorf("invalid guard search barrier")
	}
	return nil
}

func validateValidityRow(row []json.RawMessage) error {
	var class, source, status string
	var result bool
	if len(row) != 5 || decode(row[1], &class) || class != "state" && class != "action" || decode(row[2], &source) || !digestText(source) || decode(row[3], &result) || decode(row[4], &status) || status != "valid" && status != "invalid-input" || (status == "valid") != result {
		return fmt.Errorf("invalid validity row")
	}
	return nil
}

func validateApplicabilityRow(row []json.RawMessage) error {
	var state, occurrence, status string
	var result bool
	if len(row) != 5 || decode(row[1], &state) || !digestText(state) || decode(row[2], &occurrence) || !digestText(occurrence) || decode(row[3], &result) || decode(row[4], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid applicability row")
	}
	return nil
}

func validateTransitionRow(row []json.RawMessage) error {
	var state, occurrence, applicability, resultState, status string
	if len(row) != 6 || decode(row[1], &state) || !digestText(state) || decode(row[2], &occurrence) || !digestText(occurrence) || decode(row[3], &applicability) || !digestText(applicability) || decode(row[4], &resultState) || !digestText(resultState) || decode(row[5], &status) || !slices.Contains([]string{"applied", "inapplicable", "invalid-input"}, status) || (status == "applied") != (resultState != zeroObjectDigest) {
		return fmt.Errorf("invalid transition row")
	}
	return nil
}

func validateEqualityRow(row []json.RawMessage) error {
	var left, right, status string
	var result bool
	if len(row) != 5 || decode(row[1], &left) || !digestText(left) || decode(row[2], &right) || !digestText(right) || decode(row[3], &result) || decode(row[4], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid equality row")
	}
	return nil
}

func validateLiteralRow(row []json.RawMessage) error {
	var state, aFacts, bFacts, atom, status string
	var polarity, result bool
	if len(row) != 8 || decode(row[1], &state) || !digestText(state) || decode(row[2], &aFacts) || !digestText(aFacts) || decode(row[3], &bFacts) || !digestText(bFacts) || decode(row[4], &atom) || !slices.Contains(actionrelations.Atoms, atom) || decode(row[5], &polarity) || decode(row[6], &result) || decode(row[7], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid literal row")
	}
	return nil
}

func validateMatchRow(row []json.RawMessage) error {
	if len(row) != 12 {
		return fmt.Errorf("invalid match row length")
	}
	for _, index := range []int{1, 2, 3, 4, 5, 6} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid match digest %d", index)
		}
	}
	var trace, pattern, result bool
	var literals []string
	var status string
	if decode(row[7], &trace) || decode(row[8], &pattern) || decode(row[9], &literals) || len(literals) > 2 || !allDigestList(literals) || decode(row[10], &result) || decode(row[11], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid match result")
	}
	return nil
}

func validateUnanimousUse(row []json.RawMessage) error {
	if len(row) != 8 {
		return fmt.Errorf("invalid unanimous-use length")
	}
	for _, index := range []int{1, 2, 3, 4, 5} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid unanimous-use digest %d", index)
		}
	}
	var result bool
	var status string
	if decode(row[6], &result) || decode(row[7], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid unanimous-use result")
	}
	return nil
}

func validateRawInput(row []json.RawMessage) error {
	var class, encoded string
	if len(row) != 3 || decode(row[1], &class) || class != "state" && class != "action" || decode(row[2], &encoded) || encoded == "" {
		return fmt.Errorf("invalid raw input")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > 768 || base64.RawURLEncoding.EncodeToString(raw) != encoded {
		return fmt.Errorf("invalid raw input encoding")
	}
	return nil
}

func validateStaticFootprint(row []json.RawMessage) error {
	if len(row) != 10 {
		return fmt.Errorf("invalid static footprint length")
	}
	for _, index := range []int{1, 2, 3, 4, 5, 6, 7} {
		var digest string
		if decode(row[index], &digest) || !digestText(digest) {
			return fmt.Errorf("invalid static footprint digest %d", index)
		}
	}
	var a, b, status string
	var result bool
	if decode(row[4], &a) || decode(row[5], &b) || a == b || decode(row[8], &result) || decode(row[9], &status) || status != "valid" && status != "invalid-input" || status == "invalid-input" && result {
		return fmt.Errorf("invalid static footprint result")
	}
	return nil
}

func decodeReservation(data []byte) (actionrelationledger.Reservation, error) {
	value, err := actionrelationledger.ParseReservation(data)
	if err != nil || !runIDText(value.RunID) || !digestText(value.TaskDigest) || len(value.OperationCodes) < 1 || value.TotalBefore < 0 {
		return actionrelationledger.Reservation{}, fmt.Errorf("invalid reservation")
	}
	for _, code := range value.OperationCodes {
		if code < 1 || code > 25 {
			return actionrelationledger.Reservation{}, fmt.Errorf("invalid reservation operation")
		}
	}
	if value.Status == "reserved" {
		if value.TotalAfter != value.TotalBefore+len(value.OperationCodes) {
			return actionrelationledger.Reservation{}, fmt.Errorf("invalid reserved total")
		}
	} else if value.Status == "rejected-cap" {
		if value.TotalAfter != value.TotalBefore {
			return actionrelationledger.Reservation{}, fmt.Errorf("invalid rejected total")
		}
	} else {
		return actionrelationledger.Reservation{}, fmt.Errorf("invalid reservation status")
	}
	return value, nil
}

func decodeTruthShard(row []json.RawMessage) (decodedTruthShard, error) {
	value := decodedTruthShard{}
	var tag string
	var pairRows [][]json.RawMessage
	if len(row) != 6 || decode(row[0], &tag) || tag != objectKinds[29] || decode(row[1], &value.World) || decode(row[2], &value.Ordinal) || decode(row[3], &value.Count) || decode(row[4], &value.Terminals) || json.Unmarshal(row[5], &pairRows) != nil || !digestText(value.World) || value.Ordinal < 0 || value.Count < 1 || value.Ordinal >= value.Count || len(value.Terminals) < 1 || len(pairRows) < 1 || !sortedUniqueDigestList(value.Terminals) {
		return decodedTruthShard{}, fmt.Errorf("invalid scorer truth shard")
	}
	value.Pairs = make([]decodedTruthPair, len(pairRows))
	labels := []string{"commutes", "conflicts", "a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable"}
	for index, fields := range pairRows {
		pair := &value.Pairs[index]
		if len(fields) != 4 || decode(fields[0], &pair.State) || decode(fields[1], &pair.A) || decode(fields[2], &pair.B) || decode(fields[3], &pair.Label) || !digestText(pair.State) || !digestText(pair.A) || !digestText(pair.B) || pair.A >= pair.B || !slices.Contains(labels, pair.Label) || index > 0 && compareTruthPair(value.Pairs[index-1], *pair) >= 0 {
			return decodedTruthShard{}, fmt.Errorf("invalid scorer truth row %d", index)
		}
	}
	return value, nil
}

func compareTruthPair(a, b decodedTruthPair) int {
	if a.State != b.State {
		return compareString(a.State, b.State)
	}
	if a.A != b.A {
		return compareString(a.A, b.A)
	}
	return compareString(a.B, b.B)
}

func decodeWorldPolicyRow(row []json.RawMessage) (decodedWorldPolicyRow, error) {
	v := decodedWorldPolicyRow{}
	var tag string
	var work, matches, certificates []int
	if len(row) != 20 || decode(row[0], &tag) || tag != objectKinds[32] || decode(row[1], &v.Panel) || decode(row[2], &v.Curriculum) || decode(row[3], &v.Family) || decode(row[4], &v.WorldOrdinal) || decode(row[5], &v.Stratum) || decode(row[6], &v.World) || decode(row[7], &v.Policy) || decode(row[8], &v.Terminal) || decode(row[9], &work) || decode(row[10], &v.WorkTotal) || decode(row[11], &matches) || decode(row[12], &certificates) || decode(row[13], &v.SleepCount) || decode(row[14], &v.HistoryCount) || decode(row[15], &v.TerminalSet) || decode(row[16], &v.WorkTerminal) || decode(row[17], &v.BehaviorEqual) || decode(row[18], &v.Remaining) || decode(row[19], &v.OperationRoot) {
		return decodedWorldPolicyRow{}, fmt.Errorf("invalid world-policy fields")
	}
	if !copyFixed(v.Work[:], work) || !copyFixed(v.Matches[:], matches) || !copyFixed(v.Certificates[:], certificates) || !onePanel(v.Panel) || v.Curriculum < 0 || v.Family != v.Curriculum%8 || v.WorldOrdinal < 0 || v.WorldOrdinal > 5 || !slices.Contains([]string{"positive-effect", "neutral", "adverse"}, v.Stratum) || !digestText(v.World) || !onePolicy(v.Policy) || !slices.Contains([]string{"completed", "budget-exhausted"}, v.Terminal) || !nonnegativeInts(work) || sumInts(work) != v.WorkTotal || v.WorkTotal < 1 || !nonnegativeInts(matches) || !nonnegativeInts(certificates) || v.SleepCount < 0 || v.HistoryCount < 0 || v.HistoryCount > 65_536 || !digestText(v.TerminalSet) || !digestText(v.WorkTerminal) || v.Remaining < 0 || v.Remaining > 2_000_000 || !digestText(v.OperationRoot) {
		return decodedWorldPolicyRow{}, fmt.Errorf("invalid world-policy semantics")
	}
	if v.Terminal == "completed" && v.WorkTerminal != zeroObjectDigest || v.Terminal == "budget-exhausted" && (v.WorkTerminal == zeroObjectDigest || v.Remaining != 0) {
		return decodedWorldPolicyRow{}, fmt.Errorf("invalid world-policy terminal")
	}
	return v, nil
}

func decodeCurriculumPolicyRow(row []json.RawMessage) (decodedCurriculumPolicyRow, error) {
	v := decodedCurriculumPolicyRow{}
	var tag string
	var acquisition, search, lifecycle []int
	if len(row) != 18 || decode(row[0], &tag) || tag != objectKinds[33] || decode(row[1], &v.Panel) || decode(row[2], &v.Curriculum) || decode(row[3], &v.Family) || decode(row[4], &v.Policy) || decode(row[5], &v.Acquisition) || decode(row[6], &v.Artifact) || decode(row[7], &acquisition) || decode(row[8], &v.AcquisitionTerminal) || decode(row[9], &v.WorldRows) || decode(row[10], &v.Terminal) || decode(row[11], &search) || decode(row[12], &v.SearchTotal) || decode(row[13], &lifecycle) || decode(row[14], &v.LifecycleTotal) || decode(row[15], &v.BehaviorEqual) || decode(row[16], &v.Remaining) || decode(row[17], &v.OperationRoot) {
		return decodedCurriculumPolicyRow{}, fmt.Errorf("invalid curriculum-policy fields")
	}
	if !copyFixed(v.AcquisitionWork[:], acquisition) || !copyFixed(v.SearchWork[:], search) || !copyFixed(v.LifecycleWork[:], lifecycle) || !onePanel(v.Panel) || v.Curriculum < 0 || v.Family != v.Curriculum%8 || !onePolicy(v.Policy) || !slices.Contains([]string{"not-applicable", "completed", "no-discovery", "budget-exhausted"}, v.Acquisition) || !digestText(v.Artifact) || !digestText(v.AcquisitionTerminal) || len(v.WorldRows) != 6 || !uniqueDigestList(v.WorldRows) || !slices.Contains([]string{"completed", "budget-exhausted"}, v.Terminal) || !nonnegativeInts(acquisition) || !nonnegativeInts(search) || !nonnegativeInts(lifecycle) || sumInts(search) != v.SearchTotal || sumInts(lifecycle) != v.LifecycleTotal || v.LifecycleTotal > 2_000_000 || v.Remaining < 0 || v.Remaining > 2_000_000 || !digestText(v.OperationRoot) {
		return decodedCurriculumPolicyRow{}, fmt.Errorf("invalid curriculum-policy semantics")
	}
	for index := range lifecycle {
		if lifecycle[index] != acquisition[index]+search[index] {
			return decodedCurriculumPolicyRow{}, fmt.Errorf("curriculum-policy work does not conserve")
		}
	}
	if v.Terminal == "completed" && v.Remaining != 2_000_000-v.LifecycleTotal || v.Terminal == "budget-exhausted" && v.Remaining != 0 || v.Acquisition == "not-applicable" && (v.Artifact != zeroObjectDigest || sumInts(acquisition) != 0 || v.AcquisitionTerminal != zeroObjectDigest) {
		return decodedCurriculumPolicyRow{}, fmt.Errorf("invalid curriculum-policy terminal")
	}
	return v, nil
}

func decodeStoreBoundary(row []json.RawMessage) (decodedStoreBoundary, error) {
	v := decodedStoreBoundary{}
	var tag string
	if len(row) != 6 || decode(row[0], &tag) || tag != objectKinds[35] || decode(row[1], &v.Curriculum) || decode(row[2], &v.Scope) || decode(row[3], &v.Tables) || decode(row[4], &v.ObjectSet) || decode(row[5], &v.IndexRoot) || v.Curriculum < 0 || v.Scope != "nous" && v.Scope != "no-guard" || len(v.Tables) != map[string]int{"nous": 8, "no-guard": 6}[v.Scope] || !allDigestList(v.Tables) || !digestText(v.ObjectSet) || !digestText(v.IndexRoot) {
		return decodedStoreBoundary{}, fmt.Errorf("invalid store boundary")
	}
	return v, nil
}

func decodeAttemptLedger(row []json.RawMessage) (decodedAttemptLedger, error) {
	v := decodedAttemptLedger{}
	var tag string
	if len(row) != 11 || decode(row[0], &tag) || tag != objectKinds[36] || decode(row[1], &v.Panel) || decode(row[2], &v.Authority) || decode(row[3], &v.Curriculum) || len(row[4]) == 0 || decode(row[5], &v.Attempt) || json.Unmarshal(row[6], &v.Draws) != nil || decode(row[7], &v.DrawRoot) || json.Unmarshal(row[8], &v.Phases) != nil || decode(row[9], &v.TotalWork) || decode(row[10], &v.Terminal) {
		return decodedAttemptLedger{}, fmt.Errorf("invalid attempt ledger fields")
	}
	v.Seed = slices.Clone(row[4])
	if !onePanel(v.Panel) || !validPanelAuthority(v.Panel, v.Authority) || v.Curriculum < 0 || v.Attempt < 0 || v.Attempt > 31 || len(v.Draws) != 66 || !digestText(v.DrawRoot) || len(v.Phases) < 1 || len(v.Phases) > 8 || v.TotalWork < 66 || v.TotalWork > 1_000_000 || !slices.Contains([]string{"accepted", "rejected"}, v.Terminal) {
		return decodedAttemptLedger{}, fmt.Errorf("invalid attempt ledger semantics")
	}
	return v, nil
}

func decodeOperationRoot(row []json.RawMessage) (decodedOperationRoot, error) {
	v := decodedOperationRoot{}
	var tag string
	if len(row) < 2 || decode(row[0], &tag) || tag != objectKinds[46] || decode(row[1], &v.Variant) {
		return decodedOperationRoot{}, fmt.Errorf("invalid operation root")
	}
	switch v.Variant {
	case "range":
		if len(row) != 7 || decode(row[2], &v.RunID) || decode(row[3], &v.Phase) || decode(row[4], &v.Start) || decode(row[5], &v.Count) || decode(row[6], &v.CallRoot) || !runIDText(v.RunID) || v.Phase < 1 || v.Phase > 2 || v.Count < 0 || !digestText(v.CallRoot) {
			return decodedOperationRoot{}, fmt.Errorf("invalid operation range")
		}
	case "concat":
		if len(row) != 4 || decode(row[2], &v.Context) || decode(row[3], &v.Children) || !digestText(v.Context) || len(v.Children) < 1 || !allDigestList(v.Children) {
			return decodedOperationRoot{}, fmt.Errorf("invalid operation concat")
		}
	default:
		return decodedOperationRoot{}, fmt.Errorf("unknown operation root variant")
	}
	return v, nil
}

func decodeFixture(row []json.RawMessage) (decodedFixture, error) {
	v := decodedFixture{}
	var tag string
	if len(row) != 9 || decode(row[0], &tag) || tag != objectKinds[47] || decode(row[1], &v.Panel) || decode(row[2], &v.Curriculum) || decode(row[3], &v.Family) || decode(row[4], &v.Accepted) || decode(row[5], &v.Training) || decode(row[6], &v.Views) || decode(row[7], &v.Worlds) || decode(row[8], &v.AttemptLedgers) || !onePanel(v.Panel) || v.Curriculum < 0 || v.Family != v.Curriculum%8 || v.Accepted < 0 || v.Accepted > 31 || len(v.Training) != 16 || len(v.Views) != 32 || len(v.Worlds) != 6 || len(v.AttemptLedgers) != v.Accepted+1 || !uniqueDigestList(v.Training) || !uniqueDigestList(v.Views) || !uniqueDigestList(v.Worlds) || !uniqueDigestList(v.AttemptLedgers) {
		return decodedFixture{}, fmt.Errorf("invalid curriculum fixture")
	}
	return v, nil
}

func decodeWorkTerminal(row []json.RawMessage) (decodedWorkTerminal, error) {
	v := decodedWorkTerminal{}
	var tag string
	var work []int
	if len(row) != 8 || decode(row[0], &tag) || tag != objectKinds[49] || decode(row[1], &v.RunID) || decode(row[2], &v.Phase) || decode(row[3], &v.Rejected) || decode(row[4], &v.Terminal) || decode(row[5], &work) || decode(row[6], &v.Total) || decode(row[7], &v.Remaining) || !copyFixed(v.Work[:], work) || !runIDText(v.RunID) || v.Phase != 2 || !digestText(v.Rejected) || v.Terminal != "budget-exhausted" || !nonnegativeInts(work) || sumInts(work) != v.Total || v.Total < 1 || v.Total > 2_000_000 || v.Remaining != 0 {
		return decodedWorkTerminal{}, fmt.Errorf("invalid work terminal")
	}
	return v, nil
}

func decode(raw json.RawMessage, target any) bool { return json.Unmarshal(raw, target) != nil }

func onePanel(value string) bool {
	return value == "development" || value == "validation" || value == "locked"
}

func onePolicy(value string) bool {
	return slices.Contains([]string{"complete", "lexical-order", "static-rw-sleep", "dynamic-diamond-sleep", "nous-guarded-sleep", "no-guard-sleep", "learned-no-use"}, value)
}

func copyFixed(target, source []int) bool {
	if len(target) != len(source) {
		return false
	}
	copy(target, source)
	return true
}

func nonnegativeInts(values []int) bool {
	for _, value := range values {
		if value < 0 {
			return false
		}
	}
	return true
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func allDigestList(values []string) bool {
	for _, value := range values {
		if !digestText(value) {
			return false
		}
	}
	return true
}

func uniqueDigestList(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if !digestText(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func sortedUniqueDigestList(values []string) bool {
	for index, value := range values {
		if !digestText(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func compareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
