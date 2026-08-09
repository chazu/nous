package transformexp

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

const (
	LifecycleWorkCap = 12000
	EventCountCap    = 50000
	EventByteCap     = 384
	RawChunkByteCap  = 19200000
	GzipChunkByteCap = 19250000
	ObjectByteCap    = 67108864
	ObjectCountCap   = 24002
)

var lifecycleCharges = [12]int64{1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1}

type TransformEvent struct {
	PanelOrdinal int
	Policy       string
	TaskToken    string
	Sequence     int
	Phase        string
	Category     int
	Operation    string
	Subject      string
	Object       string
	Outcome      string
	Previous     string
}

func (e TransformEvent) canonicalJSON() ([]byte, error) {
	if e.PanelOrdinal < 0 || e.PanelOrdinal > 999 || len(e.Policy) == 0 || len(e.Policy) > 32 || !ascii(e.Policy) || len(e.TaskToken) != 16 || !ascii(e.TaskToken) || e.Sequence < 0 || e.Sequence > 99999 || !oneOfString(e.Phase, "acquire", "target", "anchor", "scope", "old-guard", "locality", "training-validate", "freeze", "heldout", "terminal") || e.Category < 0 || e.Category >= len(lifecycleCharges) || len(e.Operation) == 0 || len(e.Operation) > 24 || !ascii(e.Operation) || len(e.Outcome) == 0 || len(e.Outcome) > 32 || !ascii(e.Outcome) || !digestString(e.Subject) || !digestString(e.Object) || !digestString(e.Previous) {
		return nil, errors.New("invalid transformation event")
	}
	b, err := json.Marshal([]any{"transform-events/v2", e.PanelOrdinal, e.Policy, e.TaskToken, e.Sequence, e.Phase, e.Category, e.Operation, e.Subject, e.Object, e.Outcome, e.Previous})
	if err != nil || len(b) > EventByteCap {
		return nil, errors.New("transformation event exceeds canonical cap")
	}
	return b, nil
}

type TransformOperation struct {
	Operation string
	Phase     string
	Inputs    []string
	Outputs   []string
	Outcome   string
	Category  int
}

func (o TransformOperation) canonicalJSON() ([]byte, error) {
	if len(o.Operation) == 0 || len(o.Operation) > 24 || !ascii(o.Operation) || !oneOfString(o.Phase, "acquire", "target", "anchor", "scope", "old-guard", "locality", "training-validate", "freeze", "heldout", "terminal") || len(o.Inputs) > 8 || len(o.Outputs) > 8 || len(o.Outcome) == 0 || len(o.Outcome) > 32 || !ascii(o.Outcome) || o.Category < 0 || o.Category >= len(lifecycleCharges) {
		return nil, errors.New("invalid transformation operation")
	}
	for _, digest := range append(slices.Clone(o.Inputs), o.Outputs...) {
		if digest != "" && !digestString(digest) {
			return nil, errors.New("invalid operation digest")
		}
	}
	b, err := json.Marshal([]any{"transform-operation/v1", o.Operation, o.Phase, o.Inputs, o.Outputs, o.Outcome, o.Category, lifecycleCharges[o.Category]})
	if err != nil || len(b) > 1536 {
		return nil, errors.New("operation exceeds canonical cap")
	}
	return b, nil
}

type TransformObjectTable struct {
	Objects map[string][]byte
	Bytes   int
}

func newTransformObjectTable() TransformObjectTable {
	return TransformObjectTable{Objects: map[string][]byte{}}
}

func (t *TransformObjectTable) admit(data []byte) (string, error) {
	cap, err := transformObjectCap(data)
	if err != nil {
		return "", fmt.Errorf("invalid transformation object: %w", err)
	}
	if len(data) > cap {
		return "", fmt.Errorf("invalid transformation object: size %d exceeds cap %d", len(data), cap)
	}
	digest := digestBytes(data)
	if prior, ok := t.Objects[digest]; ok {
		if !bytes.Equal(prior, data) {
			return "", errors.New("object digest collision")
		}
		return digest, nil
	}
	if len(t.Objects) >= ObjectCountCap || t.Bytes+len(data) > ObjectByteCap {
		return "", errors.New("transformation object table cap")
	}
	t.Objects[digest] = bytes.Clone(data)
	t.Bytes += len(data)
	return digest, nil
}

func transformObjectCap(data []byte) (int, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return 0, errors.New("object trailing bytes")
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(canonical, data) {
		return 0, errors.New("noncanonical object")
	}
	row, ok := value.([]any)
	if !ok || len(row) == 0 {
		return 0, errors.New("object is not a versioned array")
	}
	version, ok := row[0].(string)
	if !ok {
		return 0, errors.New("object version")
	}
	caps := map[string]int{
		"transform-atom/v1": 128, "transform-node-facts/v1": 192, "transform-parent-facts/v1": 128,
		"typed-reference-forest/v1": 2048, "set-value/v1": 128, "transform-edit-status/v1": 256,
		"concrete-program/v1": 640, "transform-program-batch/v1": 1152, "transform-partial/v1": 256,
		"transform-schema/v1": 256, "transform-result/v1": 256, "transform-closure/v1": 1024,
		"transform-certificate/v1": 2048, "transform-schema-application/v1": 2560,
		"transform-evidence-attempt/v1": 2560, "transform-terminal/v1": 256,
		"transform-store-boundary/v1": 256, "transform-operation/v1": 1536,
	}
	cap, ok := caps[version]
	if !ok {
		return 0, errors.New("unknown transformation object kind")
	}
	return cap, nil
}

type TransformTranscriptSink struct {
	Ordinal              int
	Policy               string
	Token                string
	Manifest             string
	Previous             string
	Events               []TransformEvent
	Objects              TransformObjectTable
	Vector               [12]int64
	Work                 int64
	Applications         int
	terminal             bool
	lastObject           string
	lastOutput           string
	lastAttach           bool
	reserved             bool
	reservedPhase        string
	reservedStartWork    int64
	reservedMaximumWork  int64
	applicationCommitted bool
	waitingForComparison bool
}

var errTransformApplicationBudget = errors.New("transformation application budget exhausted")

func newTransformTranscriptSink(ordinal int, policy, token, manifestDigest string) (*TransformTranscriptSink, error) {
	if !digestString(manifestDigest) {
		return nil, errors.New("invalid policy manifest digest")
	}
	initial, _ := json.Marshal([]any{"transform-chain/v1", manifestDigest, token})
	return &TransformTranscriptSink{Ordinal: ordinal, Policy: policy, Token: token, Manifest: manifestDigest, Previous: digestBytes(initial), Objects: newTransformObjectTable()}, nil
}

func (s *TransformTranscriptSink) Admit(data []byte) (string, error) { return s.Objects.admit(data) }

func (s *TransformTranscriptSink) BeginApplication(phase string, maximumWork int64) error {
	applicationCap := ApplicationsPerPolicy
	if phase != "heldout" {
		applicationCap -= 8
	}
	if s.reserved {
		return errors.New("nested transformation application reservation")
	}
	if maximumWork <= 0 || s.Applications >= applicationCap || s.Work+maximumWork >= LifecycleWorkCap {
		return errTransformApplicationBudget
	}
	s.reserved = true
	s.reservedPhase = phase
	s.reservedStartWork = s.Work
	s.reservedMaximumWork = maximumWork
	s.applicationCommitted = false
	return nil
}

func (s *TransformTranscriptSink) EmitValues(operation, phase, outcome string, category int, inputs, outputs [][]byte) error {
	before := make(map[string]struct{}, len(s.Objects.Objects))
	for digest := range s.Objects.Objects {
		before[digest] = struct{}{}
	}
	beforeBytes := s.Objects.Bytes
	committed := false
	defer func() {
		if committed {
			return
		}
		for digest := range s.Objects.Objects {
			if _, existed := before[digest]; !existed {
				delete(s.Objects.Objects, digest)
			}
		}
		s.Objects.Bytes = beforeBytes
	}()
	inputDigests := make([]string, len(inputs))
	for index, value := range inputs {
		digest, err := s.Admit(value)
		if err != nil {
			return fmt.Errorf("input %d: %w", index, err)
		}
		inputDigests[index] = digest
	}
	outputDigests := make([]string, len(outputs))
	for index, value := range outputs {
		digest, err := s.Admit(value)
		if err != nil {
			return fmt.Errorf("output %d: %w", index, err)
		}
		outputDigests[index] = digest
	}
	if err := s.Emit(TransformOperation{operation, phase, inputDigests, outputDigests, outcome, category}); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *TransformTranscriptSink) EmitEvidenceLink(phase string, attemptedBytes []byte) error {
	if !s.lastAttach || s.lastOutput == "" || s.lastObject == "" {
		return errors.New("invalid evidence boundary")
	}
	before := make(map[string]struct{}, len(s.Objects.Objects))
	for digest := range s.Objects.Objects {
		before[digest] = struct{}{}
	}
	beforeBytes := s.Objects.Bytes
	committed := false
	defer func() {
		if committed {
			return
		}
		for digest := range s.Objects.Objects {
			if _, existed := before[digest]; !existed {
				delete(s.Objects.Objects, digest)
			}
		}
		s.Objects.Bytes = beforeBytes
	}()
	attemptedDigest := digestBytes(attemptedBytes)
	var attempted any
	if json.Unmarshal(attemptedBytes, &attempted) != nil {
		return errors.New("evidence value is not JSON")
	}
	attemptedKind, err := transformSemanticKind(attemptedBytes)
	if err != nil {
		return err
	}
	attemptBytes, _ := json.Marshal([]any{"transform-evidence-attempt/v1", "attached", attemptedKind, attempted, attemptedDigest, s.lastOutput, s.lastObject})
	attemptDigest, err := s.Admit(attemptBytes)
	if err != nil {
		return err
	}
	if err := s.Emit(TransformOperation{"evidence-link", phase, []string{attemptedDigest, s.lastOutput, s.lastObject}, []string{attemptDigest}, "attached", 10}); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *TransformTranscriptSink) Emit(o TransformOperation) error {
	if s.terminal {
		return errors.New("event after terminal")
	}
	if err := s.validateOperation(o); err != nil {
		return err
	}
	operationBytes, err := o.canonicalJSON()
	if err != nil {
		return err
	}
	operationDigest := digestBytes(operationBytes)
	_, operationExisted := s.Objects.Objects[operationDigest]
	operationDigest, err = s.Admit(operationBytes)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed && !operationExisted {
			delete(s.Objects.Objects, operationDigest)
			s.Objects.Bytes -= len(operationBytes)
		}
	}()
	charge := lifecycleCharges[o.Category]
	if len(s.Events) >= EventCountCap || s.Work+charge > LifecycleWorkCap || o.Phase != "terminal" && s.Work+charge >= LifecycleWorkCap {
		return errors.New("transformation lifecycle cap")
	}
	if s.reserved && s.Work+charge > s.reservedStartWork+s.reservedMaximumWork {
		return errors.New("transformation application exceeded reserved work")
	}
	if o.Operation == "schema-application" || o.Operation == "replay-application" {
		if !s.reserved || s.reservedPhase != o.Phase || s.applicationCommitted {
			return errors.New("unreserved transformation application")
		}
	}
	subject := digestBytes([]byte("transform-empty-subject/v1"))
	if len(o.Inputs) > 0 && o.Inputs[0] != "" {
		subject = o.Inputs[0]
	}
	event := TransformEvent{s.Ordinal, s.Policy, s.Token, len(s.Events), o.Phase, o.Category, o.Operation, subject, operationDigest, o.Outcome, s.Previous}
	eventBytes, err := event.canonicalJSON()
	if err != nil {
		return err
	}
	s.Events = append(s.Events, event)
	s.Vector[o.Category]++
	s.Work += charge
	step, _ := json.Marshal([]any{"transform-chain-step/v1", json.RawMessage(eventBytes)})
	s.Previous = digestBytes(step)
	if o.Operation == "schema-application" || o.Operation == "replay-application" {
		s.Applications++
		s.applicationCommitted = true
		s.waitingForComparison = o.Operation == "schema-application" && o.Phase == "training-validate" && s.reservedMaximumWork == 80 && o.Outcome == "applied"
		if o.Operation == "replay-application" {
			s.finishApplicationReservation()
		}
	}
	if o.Operation == "evidence-link" {
		s.lastAttach = false
		if s.reserved && s.applicationCommitted && !s.waitingForComparison {
			s.finishApplicationReservation()
		}
	} else {
		s.lastObject = operationDigest
		s.lastOutput = ""
		if len(o.Outputs) == 1 {
			s.lastOutput = o.Outputs[0]
		}
		s.lastAttach = oneOfString(o.Operation, "node", "parent", "target", "compare", "candidate-allocate", "refine", "edit-validate", "edit-apply", "schema-application", "output-compare", "verify")
	}
	if o.Operation == "output-compare" && s.waitingForComparison && s.outputComparisonIsEndpoint(o) {
		s.finishApplicationReservation()
	}
	if o.Phase == "terminal" && o.Operation == "terminal" {
		s.terminal = true
	}
	committed = true
	return nil
}

func (s *TransformTranscriptSink) finishApplicationReservation() {
	s.reserved = false
	s.reservedPhase = ""
	s.reservedStartWork = 0
	s.reservedMaximumWork = 0
	s.applicationCommitted = false
	s.waitingForComparison = false
}

func (s *TransformTranscriptSink) outputComparisonIsEndpoint(operation TransformOperation) bool {
	if len(operation.Inputs) != 3 {
		return false
	}
	forest, err := transformschema.ParseForest(s.Objects.Objects[operation.Inputs[1]])
	if err != nil || len(forest.Nodes) == 0 {
		return false
	}
	kind, value, err := decodeTransformAtom(s.Objects.Objects[operation.Inputs[2]])
	id, ok := jsonInteger(value)
	if err != nil || kind != "id" || !ok {
		return false
	}
	return id == len(forest.Nodes)-1
}

func (s *TransformTranscriptSink) validateOperation(o TransformOperation) error {
	categories := map[string]int{
		"node": 0, "parent": 1, "target": 2, "compare": 3, "candidate-allocate": 4, "refine": 5,
		"edit-validate": 6, "edit-apply": 7, "schema-predicate": 8, "output-compare": 9,
		"evidence-link": 10, "canonicalize": 11, "hash": 11, "verify": 11,
		"schema-application": 11, "replay-application": 11, "terminal": 11,
	}
	category, ok := categories[o.Operation]
	if !ok || o.Category != category {
		return errors.New("operation category mismatch")
	}
	if o.Operation == "terminal" && o.Phase != "terminal" || o.Phase == "terminal" && o.Operation != "terminal" {
		return errors.New("terminal phase mismatch")
	}
	if o.Operation == "evidence-link" {
		if !s.lastAttach || len(o.Inputs) != 3 || o.Inputs[1] != s.lastOutput || o.Inputs[2] != s.lastObject || len(o.Outputs) != 1 || !oneOfString(o.Outcome, "attached", "rejected") {
			return errors.New("invalid immediate evidence attachment")
		}
		if o.Inputs[0] != s.lastOutput && s.lastOutput != "" && !s.applicationResultProjection(o.Inputs[0]) {
			return errors.New("evidence attempted digest mismatch")
		}
	}
	for index, digest := range append(slices.Clone(o.Inputs), o.Outputs...) {
		if digest != "" {
			if o.Operation == "evidence-link" && index == 0 {
				continue
			}
			if _, exists := s.Objects.Objects[digest]; !exists {
				return errors.New("operation references absent object")
			}
		}
	}
	if o.Operation == "schema-application" || o.Operation == "replay-application" {
		if o.Phase != "training-validate" && o.Phase != "heldout" || o.Phase == "training-validate" && s.Applications >= 40 || s.Applications >= 48 {
			return errors.New("application credit exhausted")
		}
		if len(o.Inputs) != 2 || len(o.Outputs) != 1 {
			return errors.New("invalid application arity")
		}
	}
	if o.Operation == "terminal" {
		if len(o.Inputs) != 1 || len(o.Outputs) != 1 || !oneOfString(o.Outcome, "completed", "no-discovery", "budget-exhausted") {
			return errors.New("invalid terminal operation")
		}
	}
	return validateReducedOperation(o, s.Applications, s.lastAttach, s.lastOutput, s.lastObject)
}

func (s *TransformTranscriptSink) applicationResultProjection(attempted string) bool {
	data := s.Objects.Objects[s.lastOutput]
	if len(data) == 0 {
		return false
	}
	var row []any
	if json.Unmarshal(data, &row) != nil || len(row) != 3 || row[0] != "transform-schema-application/v1" {
		return false
	}
	result, err := json.Marshal(row[1])
	return err == nil && digestBytes(result) == attempted
}

type TransformTranscriptBundle struct {
	Raw          []byte
	Gzip         []byte
	Vector       [12]int64
	Work         int64
	Objects      map[string][]byte
	Terminal     string
	Applications int
}

func (s *TransformTranscriptSink) Bundle() (TransformTranscriptBundle, error) {
	if !s.terminal {
		return TransformTranscriptBundle{}, errors.New("missing terminal event")
	}
	if s.reserved {
		return TransformTranscriptBundle{}, errors.New("unfinished transformation application reservation")
	}
	var raw bytes.Buffer
	for _, event := range s.Events {
		encoded, err := event.canonicalJSON()
		if err != nil {
			return TransformTranscriptBundle{}, err
		}
		raw.Write(encoded)
		raw.WriteByte('\n')
	}
	if raw.Len() > RawChunkByteCap {
		return TransformTranscriptBundle{}, errors.New("raw transcript cap")
	}
	var compressed bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	zw.Header.ModTime = time.Unix(0, 0)
	zw.Header.OS = 255
	if _, err := zw.Write(raw.Bytes()); err != nil {
		return TransformTranscriptBundle{}, err
	}
	if err := zw.Close(); err != nil {
		return TransformTranscriptBundle{}, err
	}
	if compressed.Len() > GzipChunkByteCap {
		return TransformTranscriptBundle{}, errors.New("gzip transcript cap")
	}
	objects := make(map[string][]byte, len(s.Objects.Objects))
	for digest, value := range s.Objects.Objects {
		objects[digest] = bytes.Clone(value)
	}
	return TransformTranscriptBundle{Raw: raw.Bytes(), Gzip: compressed.Bytes(), Vector: s.Vector, Work: s.Work, Objects: objects, Terminal: s.Events[len(s.Events)-1].Outcome, Applications: s.Applications}, nil
}

type transformApplicationSpan struct {
	first, application, last int
	phase                    string
	maximumWork              int64
	operation                string
}

func transformApplicationSpans(raw []byte, objects map[string][]byte) (map[int]transformApplicationSpan, error) {
	type decodedEvent struct {
		operation TransformOperation
	}
	var decoded []decodedEvent
	preScanner := bufio.NewScanner(bytes.NewReader(raw))
	preScanner.Buffer(make([]byte, EventByteCap+1), EventByteCap+1)
	for preScanner.Scan() {
		event, err := parseTransformEvent(preScanner.Bytes())
		if err != nil || event.Sequence != len(decoded) {
			return nil, errors.New("cannot predecode application reservations")
		}
		operation, err := parseTransformOperation(objects[event.Object])
		if err != nil {
			return nil, errors.New("cannot predecode application operation")
		}
		decoded = append(decoded, decodedEvent{operation})
	}
	if err := preScanner.Err(); err != nil {
		return nil, err
	}
	spans := map[int]transformApplicationSpan{}
	covered := map[int]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	sequence := 0
	for scanner.Scan() {
		event, err := parseTransformEvent(scanner.Bytes())
		if err != nil || event.Sequence != sequence {
			return nil, errors.New("cannot reconstruct application reservations")
		}
		operation, err := parseTransformOperation(objects[event.Object])
		if err != nil {
			return nil, errors.New("cannot reconstruct application operation")
		}
		span := transformApplicationSpan{}
		switch operation.Operation {
		case "replay-application":
			span = transformApplicationSpan{sequence, sequence, sequence, operation.Phase, 1, operation.Operation}
		case "schema-application":
			if len(operation.Outputs) != 1 {
				return nil, errors.New("schema application reservation lacks output")
			}
			var application []json.RawMessage
			var certificate []json.RawMessage
			if json.Unmarshal(objects[operation.Outputs[0]], &application) != nil || len(application) != 3 || json.Unmarshal(application[2], &certificate) != nil || len(certificate) != 12 {
				return nil, errors.New("schema application reservation certificate wire")
			}
			var first, last int
			if json.Unmarshal(certificate[10], &first) != nil || json.Unmarshal(certificate[11], &last) != nil || first < 0 || first > sequence || last != sequence+1 {
				return nil, errors.New("schema application reservation range")
			}
			maximumWork := int64(68)
			endpoint := last
			if operation.Phase == "training-validate" && operation.Outcome == "applied" {
				for next := last + 1; next < len(decoded) && decoded[next].operation.Operation == "output-compare" && decoded[next].operation.Phase == operation.Phase; next++ {
					endpoint = next
				}
				if endpoint > last {
					maximumWork = 80
				}
			}
			span = transformApplicationSpan{first, sequence, endpoint, operation.Phase, maximumWork, operation.Operation}
		default:
			sequence++
			continue
		}
		if _, exists := spans[span.first]; exists {
			return nil, errors.New("duplicate application reservation start")
		}
		for index := span.first; index <= span.last; index++ {
			if covered[index] {
				return nil, errors.New("overlapping application reservation")
			}
			covered[index] = true
		}
		spans[span.first] = span
		sequence++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return spans, nil
}

func reduceTransformTranscript(raw []byte, objects map[string][]byte, manifestDigest string) (TransformTranscriptBundle, error) {
	return reduceTransformTranscriptWithTraining(raw, objects, manifestDigest, nil)
}

func reduceTransformTranscriptWithTraining(raw []byte, objects map[string][]byte, manifestDigest string, training []byte) (TransformTranscriptBundle, error) {
	if len(raw) == 0 || len(raw) > RawChunkByteCap || raw[len(raw)-1] != '\n' || !digestString(manifestDigest) {
		return TransformTranscriptBundle{}, errors.New("invalid transcript framing")
	}
	reservationSpans, err := transformApplicationSpans(raw, objects)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, EventByteCap+1), EventByteCap+1)
	var events []TransformEvent
	var vector [12]int64
	var work int64
	previous := ""
	terminal := false
	terminalOutcome := ""
	applications := 0
	usedObjects := map[string]bool{}
	lastOperation, lastOutput := "", ""
	lastAttach := false
	panelOrdinal, policy, taskToken := -1, "", ""
	var lifecycle *transformLifecycleState
	var activeReservation *transformApplicationSpan
	reservationStartWork := int64(0)
	for scanner.Scan() {
		line := bytes.Clone(scanner.Bytes())
		event, err := parseTransformEvent(line)
		if err != nil || event.Sequence != len(events) || terminal {
			return TransformTranscriptBundle{}, errors.New("invalid transcript event")
		}
		if len(events) == 0 {
			panelOrdinal, policy, taskToken = event.PanelOrdinal, event.Policy, event.TaskToken
			lifecycle, err = newTransformLifecycleState(policy, training)
			if err != nil {
				return TransformTranscriptBundle{}, errors.New("invalid committed training fixture")
			}
			initial, _ := json.Marshal([]any{"transform-chain/v1", manifestDigest, event.TaskToken})
			previous = digestBytes(initial)
		} else if event.PanelOrdinal != panelOrdinal || event.Policy != policy || event.TaskToken != taskToken {
			return TransformTranscriptBundle{}, errors.New("transcript identity changed")
		}
		if event.Previous != previous {
			return TransformTranscriptBundle{}, errors.New("transcript chain mismatch")
		}
		operationBytes, exists := objects[event.Object]
		if !exists || digestBytes(operationBytes) != event.Object {
			return TransformTranscriptBundle{}, errors.New("event operation object is absent")
		}
		operation, err := parseTransformOperation(operationBytes)
		if err != nil || operation.Operation != event.Operation || operation.Phase != event.Phase || operation.Outcome != event.Outcome || operation.Category != event.Category {
			return TransformTranscriptBundle{}, errors.New("event operation object mismatch")
		}
		subject := digestBytes([]byte("transform-empty-subject/v1"))
		if len(operation.Inputs) != 0 && operation.Inputs[0] != "" {
			subject = operation.Inputs[0]
		}
		if event.Subject != subject {
			return TransformTranscriptBundle{}, errors.New("event subject mismatch")
		}
		if span, starts := reservationSpans[event.Sequence]; starts {
			applicationCap := ApplicationsPerPolicy
			if span.phase != "heldout" {
				applicationCap -= 8
			}
			if activeReservation != nil || applications >= applicationCap || work+span.maximumWork >= LifecycleWorkCap {
				return TransformTranscriptBundle{}, errors.New("application reservation exceeds live budget")
			}
			spanCopy := span
			activeReservation = &spanCopy
			reservationStartWork = work
		}
		if activeReservation != nil && (event.Sequence < activeReservation.first || event.Sequence > activeReservation.last || operation.Phase != activeReservation.phase) {
			return TransformTranscriptBundle{}, errors.New("operation escaped application reservation")
		}
		if oneOfString(operation.Operation, "schema-application", "replay-application") && (activeReservation == nil || event.Sequence != activeReservation.application || operation.Operation != activeReservation.operation) {
			return TransformTranscriptBundle{}, errors.New("unreserved application operation")
		}
		for index, digest := range append(slices.Clone(operation.Inputs), operation.Outputs...) {
			if digest == "" {
				continue
			}
			if operation.Operation == "evidence-link" && index == 0 {
				continue
			}
			value, ok := objects[digest]
			if !ok || digestBytes(value) != digest {
				return TransformTranscriptBundle{}, errors.New("operation references absent object")
			}
			if _, err := transformObjectCap(value); err != nil {
				return TransformTranscriptBundle{}, errors.New("operation references invalid object")
			}
			usedObjects[digest] = true
		}
		if err := validateReducedOperation(operation, applications, lastAttach, lastOutput, lastOperation); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("operation %d %s/%s/%s: %w", event.Sequence, operation.Phase, operation.Operation, operation.Outcome, err)
		}
		if err := validateTransformSemantics(operation, objects); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("operation %d %s semantic mismatch: %w", event.Sequence, operation.Operation, err)
		}
		if err := lifecycle.observe(operation, objects); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("operation %d %s lifecycle mismatch: %w", event.Sequence, operation.Operation, err)
		}
		usedObjects[event.Object] = true
		if operation.Operation == "schema-application" || operation.Operation == "replay-application" {
			applications++
		}
		if operation.Operation == "evidence-link" {
			lastAttach = false
		} else {
			lastOperation = event.Object
			lastOutput = ""
			if len(operation.Outputs) == 1 {
				lastOutput = operation.Outputs[0]
			}
			lastAttach = oneOfString(operation.Operation, "node", "parent", "target", "compare", "candidate-allocate", "refine", "edit-validate", "edit-apply", "schema-application", "output-compare", "verify")
		}
		step, _ := json.Marshal([]any{"transform-chain-step/v1", json.RawMessage(line)})
		previous = digestBytes(step)
		vector[event.Category]++
		work += lifecycleCharges[event.Category]
		if activeReservation != nil && work-reservationStartWork > activeReservation.maximumWork {
			return TransformTranscriptBundle{}, errors.New("application exceeded reserved work")
		}
		if activeReservation != nil && event.Sequence == activeReservation.last {
			wantEndpoint := "evidence-link"
			if activeReservation.operation == "schema-application" && activeReservation.maximumWork == 80 {
				wantEndpoint = "output-compare"
			}
			if activeReservation.operation == "schema-application" && operation.Operation != wantEndpoint || activeReservation.operation == "replay-application" && operation.Operation != "replay-application" {
				return TransformTranscriptBundle{}, errors.New("application reservation has invalid final event")
			}
			activeReservation = nil
		}
		if work > LifecycleWorkCap || len(events) >= EventCountCap || event.Phase != "terminal" && work >= LifecycleWorkCap {
			return TransformTranscriptBundle{}, errors.New("transcript cap")
		}
		terminal = event.Phase == "terminal" && event.Operation == "terminal"
		if terminal {
			if err := validateTerminalObject(objects[operation.Outputs[0]], event.Outcome, work, applications, event.Sequence); err != nil {
				return TransformTranscriptBundle{}, err
			}
			terminalOutcome = event.Outcome
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil || !terminal {
		return TransformTranscriptBundle{}, errors.New("invalid transcript termination")
	}
	if activeReservation != nil {
		return TransformTranscriptBundle{}, errors.New("unfinished application reservation")
	}
	if len(usedObjects) != len(objects) {
		return TransformTranscriptBundle{}, errors.New("object table contains unreferenced values")
	}
	cloned := make(map[string][]byte, len(objects))
	for digest, value := range objects {
		if digestBytes(value) != digest {
			return TransformTranscriptBundle{}, errors.New("object table digest mismatch")
		}
		cloned[digest] = bytes.Clone(value)
	}
	return TransformTranscriptBundle{Raw: bytes.Clone(raw), Vector: vector, Work: work, Objects: cloned, Terminal: terminalOutcome, Applications: applications}, nil
}

func parseTransformOperation(data []byte) (TransformOperation, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row []any
	if decoder.Decode(&row) != nil || len(row) != 8 || row[0] != "transform-operation/v1" {
		return TransformOperation{}, errors.New("operation wire")
	}
	operation, a := row[1].(string)
	phase, b := row[2].(string)
	inputRows, c := row[3].([]any)
	outputRows, d := row[4].([]any)
	outcome, e := row[5].(string)
	categoryNumber, f := row[6].(json.Number)
	chargeNumber, g := row[7].(json.Number)
	category64, categoryErr := categoryNumber.Int64()
	charge64, chargeErr := chargeNumber.Int64()
	if !(a && b && c && d && e && f && g) || categoryErr != nil || chargeErr != nil || category64 < 0 || category64 >= int64(len(lifecycleCharges)) || charge64 != lifecycleCharges[category64] {
		return TransformOperation{}, errors.New("operation fields")
	}
	inputs := make([]string, len(inputRows))
	outputs := make([]string, len(outputRows))
	for index, value := range inputRows {
		var ok bool
		inputs[index], ok = value.(string)
		if !ok {
			return TransformOperation{}, errors.New("operation input")
		}
	}
	for index, value := range outputRows {
		var ok bool
		outputs[index], ok = value.(string)
		if !ok {
			return TransformOperation{}, errors.New("operation output")
		}
	}
	operationValue := TransformOperation{operation, phase, inputs, outputs, outcome, int(category64)}
	canonical, err := operationValue.canonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return TransformOperation{}, errors.New("noncanonical operation")
	}
	return operationValue, nil
}

func validateReducedOperation(operation TransformOperation, applications int, lastAttach bool, lastOutput, lastOperation string) error {
	categories := map[string]int{"node": 0, "parent": 1, "target": 2, "compare": 3, "candidate-allocate": 4, "refine": 5, "edit-validate": 6, "edit-apply": 7, "schema-predicate": 8, "output-compare": 9, "evidence-link": 10, "canonicalize": 11, "hash": 11, "verify": 11, "schema-application": 11, "replay-application": 11, "terminal": 11}
	if category, ok := categories[operation.Operation]; !ok || category != operation.Category {
		return errors.New("operation category mismatch")
	}
	if operation.Operation == "terminal" && operation.Phase != "terminal" || operation.Phase == "terminal" && operation.Operation != "terminal" {
		return errors.New("terminal phase mismatch")
	}
	if operation.Operation == "evidence-link" && (!lastAttach || len(operation.Inputs) != 3 || operation.Inputs[1] != lastOutput || operation.Inputs[2] != lastOperation || len(operation.Outputs) != 1 || !oneOfString(operation.Outcome, "attached", "rejected")) {
		return errors.New("invalid immediate evidence attachment")
	}
	if operation.Operation == "schema-application" || operation.Operation == "replay-application" {
		if operation.Phase != "training-validate" && operation.Phase != "heldout" || operation.Phase == "training-validate" && applications >= 40 || applications >= 48 || len(operation.Inputs) != 2 || len(operation.Outputs) != 1 {
			return errors.New("invalid application operation")
		}
	}
	if operation.Operation == "terminal" && (len(operation.Inputs) != 1 || len(operation.Outputs) != 1 || !oneOfString(operation.Outcome, "completed", "no-discovery", "budget-exhausted")) {
		return errors.New("invalid terminal operation")
	}
	type rule struct {
		phases, outcomes                             []string
		inputsMin, inputsMax, outputsMin, outputsMax int
	}
	nonterminal := []string{"acquire", "target", "anchor", "scope", "old-guard", "locality", "training-validate", "freeze", "heldout"}
	nodePhases := []string{"acquire", "target", "anchor", "scope", "old-guard", "locality", "training-validate", "heldout"}
	rules := map[string]rule{
		"node":               {nodePhases, []string{"ok", "invalid-input"}, 2, 2, 0, 1},
		"parent":             {nodePhases, []string{"ok", "absent", "invalid-input"}, 2, 2, 0, 1},
		"target":             {nodePhases, []string{"ok", "absent", "invalid-input"}, 2, 2, 0, 1},
		"compare":            {[]string{"acquire", "target", "anchor", "scope", "old-guard", "locality"}, []string{"true", "false", "invalid-input"}, 2, 2, 0, 1},
		"candidate-allocate": {[]string{"target", "anchor", "scope", "old-guard", "locality", "freeze"}, []string{"allocated", "duplicate", "rejected"}, 1, 1, 0, 1},
		"refine":             {[]string{"target", "anchor", "scope", "old-guard", "locality"}, []string{"refined", "rejected", "invalid-input"}, 2, 2, 0, 1},
		"edit-validate":      {[]string{"acquire", "training-validate", "heldout"}, []string{"valid", "no-op", "invalid-input"}, 2, 2, 1, 1},
		"edit-apply":         {[]string{"acquire", "training-validate", "heldout"}, []string{"applied", "invalid-input"}, 2, 2, 1, 1},
		"schema-predicate":   {[]string{"training-validate", "heldout"}, []string{"true", "false", "invalid-input"}, 4, 4, 0, 1},
		"output-compare":     {[]string{"acquire", "training-validate"}, []string{"equal", "different", "invalid-input"}, 3, 3, 0, 1},
		"evidence-link":      {[]string{"acquire", "target", "anchor", "scope", "old-guard", "locality", "training-validate", "freeze", "heldout"}, []string{"attached", "rejected"}, 3, 3, 1, 1},
		"canonicalize":       {nonterminal, []string{"canonical", "invalid-input"}, 1, 1, 1, 1},
		"hash":               {nonterminal, []string{"hashed", "invalid-input"}, 1, 1, 1, 1},
		"verify":             {[]string{"acquire", "freeze"}, []string{"verified", "rejected"}, 1, 1, 1, 1},
		"schema-application": {[]string{"training-validate", "heldout"}, []string{"applied", "abstain/request-count", "abstain/anchor", "abstain/locality", "abstain/expansion", "abstain/no-op", "invalid-input"}, 2, 2, 1, 1},
		"replay-application": {[]string{"training-validate", "heldout"}, []string{"applied", "abstain/replay-miss", "invalid-input"}, 2, 2, 1, 1},
		"terminal":           {[]string{"terminal"}, []string{"completed", "no-discovery", "budget-exhausted"}, 1, 1, 1, 1},
	}
	r := rules[operation.Operation]
	if !slices.Contains(r.phases, operation.Phase) || !slices.Contains(r.outcomes, operation.Outcome) || len(operation.Inputs) < r.inputsMin || len(operation.Inputs) > r.inputsMax || len(operation.Outputs) < r.outputsMin || len(operation.Outputs) > r.outputsMax {
		return fmt.Errorf("operation %q phase %q outcome %q violates normative matrix: inputs=%d outputs=%d", operation.Operation, operation.Phase, operation.Outcome, len(operation.Inputs), len(operation.Outputs))
	}
	if err := validateTransformOutcomeArity(operation); err != nil {
		return err
	}
	return nil
}

func validateTransformOutcomeArity(operation TransformOperation) error {
	want := 1
	switch operation.Operation {
	case "node", "parent", "target":
		if operation.Outcome != "ok" {
			want = 0
		}
	case "compare":
		if operation.Outcome == "invalid-input" {
			want = 0
		}
	case "candidate-allocate":
		if operation.Outcome != "allocated" {
			want = 0
		}
	case "refine":
		if operation.Outcome != "refined" {
			want = 0
		}
	case "edit-apply":
		if operation.Outcome != "applied" {
			want = 0
		}
	case "output-compare", "canonicalize", "hash":
		if operation.Outcome == "invalid-input" {
			want = 0
		}
	case "schema-predicate":
		if operation.Outcome == "invalid-input" {
			want = 0
		}
	}
	if len(operation.Outputs) != want {
		return fmt.Errorf("operation outcome requires %d outputs, got %d", want, len(operation.Outputs))
	}
	return nil
}

func transformSemanticKind(data []byte) (string, error) {
	var wire []json.RawMessage
	var version string
	if json.Unmarshal(data, &wire) != nil || len(wire) == 0 || json.Unmarshal(wire[0], &version) != nil {
		return "", errors.New("evidence value has no semantic kind")
	}
	kinds := map[string]string{
		"transform-node-facts/v1":   "node-facts",
		"transform-parent-facts/v1": "parent-facts",
		"transform-atom/v1":         "atom",
		"transform-partial/v1":      "partial",
		"transform-schema/v1":       "schema",
		"transform-edit-status/v1":  "edit-status",
		"typed-reference-forest/v1": "forest",
		"transform-result/v1":       "result",
		"transform-closure/v1":      "closure",
	}
	kind, ok := kinds[version]
	if !ok {
		return "", errors.New("value is not attachable evidence")
	}
	return kind, nil
}

func validateTerminalObject(data []byte, outcome string, work int64, applications, priorEvents int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row []any
	if decoder.Decode(&row) != nil || len(row) != 5 || row[0] != "transform-terminal/v1" || row[1] != outcome {
		return errors.New("terminal object wire")
	}
	values := []int64{work, int64(applications), int64(priorEvents)}
	for index, expected := range values {
		number, ok := row[index+2].(json.Number)
		actual, err := number.Int64()
		if !ok || err != nil || actual != expected {
			return errors.New("terminal object totals mismatch")
		}
	}
	return nil
}

func decodeTransformGzip(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > GzipChunkByteCap {
		return nil, errors.New("gzip transcript cap")
	}
	reader := bytes.NewReader(data)
	zr, err := gzip.NewReader(reader)
	if err != nil || zr.OS != 255 {
		return nil, errors.New("invalid gzip transcript header")
	}
	zr.Multistream(false)
	raw, err := io.ReadAll(io.LimitReader(zr, RawChunkByteCap+1))
	if err != nil || len(raw) > RawChunkByteCap {
		return nil, errors.New("invalid gzip transcript body")
	}
	if err := zr.Close(); err != nil || reader.Len() != 0 {
		return nil, errors.New("concatenated or trailing gzip data")
	}
	return raw, nil
}

func parseTransformEvent(data []byte) (TransformEvent, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row []any
	if err := decoder.Decode(&row); err != nil || len(row) != 12 || row[0] != "transform-events/v2" {
		return TransformEvent{}, errors.New("event wire")
	}
	integer := func(value any) (int, bool) {
		n, ok := value.(json.Number)
		if !ok {
			return 0, false
		}
		i, err := n.Int64()
		return int(i), err == nil && int64(int(i)) == i
	}
	ordinal, a := integer(row[1])
	policy, b := row[2].(string)
	token, c := row[3].(string)
	sequence, d := integer(row[4])
	phase, e := row[5].(string)
	category, f := integer(row[6])
	operation, g := row[7].(string)
	subject, h := row[8].(string)
	object, i := row[9].(string)
	outcome, j := row[10].(string)
	previous, k := row[11].(string)
	if !(a && b && c && d && e && f && g && h && i && j && k) {
		return TransformEvent{}, errors.New("event fields")
	}
	event := TransformEvent{ordinal, policy, token, sequence, phase, category, operation, subject, object, outcome, previous}
	canonical, err := event.canonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return TransformEvent{}, errors.New("noncanonical event")
	}
	return event, nil
}

func digestBytes(data []byte) string {
	d := sha256.Sum256(data)
	return hex.EncodeToString(d[:])
}

func digestString(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == string(bytes.ToLower([]byte(value)))
}

func ascii(value string) bool {
	for _, b := range []byte(value) {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func oneOfString(value string, options ...string) bool { return slices.Contains(options, value) }

func workForVector(vector [12]int64) (int64, error) {
	var work int64
	for i, count := range vector {
		if count < 0 || count > (1<<63-1)/lifecycleCharges[i] || work > (1<<63-1)-count*lifecycleCharges[i] {
			return 0, fmt.Errorf("lifecycle work overflow")
		}
		work += count * lifecycleCharges[i]
	}
	return work, nil
}
