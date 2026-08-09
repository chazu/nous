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
	b, err := json.Marshal([]any{"transform-events/v1", e.PanelOrdinal, e.Policy, e.TaskToken, e.Sequence, e.Phase, e.Category, e.Operation, e.Subject, e.Object, e.Outcome, e.Previous})
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
	if err != nil || len(data) > cap {
		return "", errors.New("invalid transformation object")
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
	Ordinal      int
	Policy       string
	Token        string
	Manifest     string
	Previous     string
	Events       []TransformEvent
	Objects      TransformObjectTable
	Vector       [12]int64
	Work         int64
	Applications int
	terminal     bool
	lastObject   string
	lastOutput   string
	lastAttach   bool
}

func newTransformTranscriptSink(ordinal int, policy, token, manifestDigest string) (*TransformTranscriptSink, error) {
	if !digestString(manifestDigest) {
		return nil, errors.New("invalid policy manifest digest")
	}
	initial, _ := json.Marshal([]any{"transform-chain/v1", manifestDigest, token})
	return &TransformTranscriptSink{Ordinal: ordinal, Policy: policy, Token: token, Manifest: manifestDigest, Previous: digestBytes(initial), Objects: newTransformObjectTable()}, nil
}

func (s *TransformTranscriptSink) Admit(data []byte) (string, error) { return s.Objects.admit(data) }

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
	operationDigest, err := s.Admit(operationBytes)
	if err != nil {
		return err
	}
	charge := lifecycleCharges[o.Category]
	if len(s.Events) >= EventCountCap || s.Work+charge > LifecycleWorkCap || o.Phase != "terminal" && s.Work+charge >= LifecycleWorkCap {
		return errors.New("transformation lifecycle cap")
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
	if o.Operation == "evidence-link" {
		s.lastAttach = false
	} else {
		s.lastObject = operationDigest
		s.lastOutput = ""
		if len(o.Outputs) == 1 {
			s.lastOutput = o.Outputs[0]
		}
		s.lastAttach = oneOfString(o.Operation, "node", "parent", "target", "compare", "candidate-allocate", "refine", "edit-validate", "edit-apply", "schema-application", "output-compare", "verify")
	}
	if o.Phase == "terminal" && o.Operation == "terminal" {
		s.terminal = true
	}
	return nil
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
	for _, digest := range append(slices.Clone(o.Inputs), o.Outputs...) {
		if digest != "" {
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
		s.Applications++
	}
	if o.Operation == "terminal" {
		if len(o.Inputs) != 1 || len(o.Outputs) != 1 || !oneOfString(o.Outcome, "completed", "no-discovery", "budget-exhausted") {
			return errors.New("invalid terminal operation")
		}
	}
	return nil
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
	Raw    []byte
	Gzip   []byte
	Vector [12]int64
	Work   int64
}

func (s *TransformTranscriptSink) Bundle() (TransformTranscriptBundle, error) {
	if !s.terminal {
		return TransformTranscriptBundle{}, errors.New("missing terminal event")
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
	return TransformTranscriptBundle{raw.Bytes(), compressed.Bytes(), s.Vector, s.Work}, nil
}

func reduceTransformTranscript(raw []byte, manifestDigest string) (TransformTranscriptBundle, error) {
	if len(raw) == 0 || len(raw) > RawChunkByteCap || raw[len(raw)-1] != '\n' || !digestString(manifestDigest) {
		return TransformTranscriptBundle{}, errors.New("invalid transcript framing")
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, EventByteCap+1), EventByteCap+1)
	var events []TransformEvent
	var vector [12]int64
	var work int64
	previous := ""
	terminal := false
	for scanner.Scan() {
		line := bytes.Clone(scanner.Bytes())
		event, err := parseTransformEvent(line)
		if err != nil || event.Sequence != len(events) || terminal {
			return TransformTranscriptBundle{}, errors.New("invalid transcript event")
		}
		if len(events) == 0 {
			initial, _ := json.Marshal([]any{"transform-chain/v1", manifestDigest, event.TaskToken})
			previous = digestBytes(initial)
		}
		if event.Previous != previous {
			return TransformTranscriptBundle{}, errors.New("transcript chain mismatch")
		}
		step, _ := json.Marshal([]any{"transform-chain-step/v1", json.RawMessage(line)})
		previous = digestBytes(step)
		vector[event.Category]++
		work += lifecycleCharges[event.Category]
		if work > LifecycleWorkCap || len(events) >= EventCountCap {
			return TransformTranscriptBundle{}, errors.New("transcript cap")
		}
		terminal = event.Phase == "terminal" && event.Operation == "terminal"
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil || !terminal {
		return TransformTranscriptBundle{}, errors.New("invalid transcript termination")
	}
	return TransformTranscriptBundle{Raw: bytes.Clone(raw), Vector: vector, Work: work}, nil
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
	if err := decoder.Decode(&row); err != nil || len(row) != 12 || row[0] != "transform-events/v1" {
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
