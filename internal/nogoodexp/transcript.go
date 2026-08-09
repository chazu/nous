package nogoodexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"sort"
	"unicode/utf8"
)

const (
	transcriptHeaderSize = 4096
	transcriptRecordSize = 96
	transcriptDictCap    = 1 << 20
)

type TranscriptOperand struct {
	Text    string
	Number  int32
	Numeric bool
	Absent  bool
}

func ID(value string) TranscriptOperand { return TranscriptOperand{Text: value} }
func OptionalID(value string) TranscriptOperand {
	if value == "" {
		return TranscriptOperand{Absent: true}
	}
	return ID(value)
}
func Number(value int32) TranscriptOperand { return TranscriptOperand{Number: value, Numeric: true} }
func Omitted() TranscriptOperand           { return TranscriptOperand{Absent: true} }

type TranscriptEvent struct {
	Category    uint8
	Code        uint8
	TaskOrdinal uint32
	Operands    [8]TranscriptOperand
}

type TranscriptBundle struct {
	Raw        []byte
	Gzip       []byte
	Dictionary []string
	Events     []TranscriptEvent
	Vector     [12]int64
}

type operandKind uint8

const (
	omit operandKind = iota
	requiredID
	optionalID
	numeric
)

type transcriptSpec struct {
	category uint8
	kinds    [8]operandKind
}

var transcriptSpecs = map[uint8]transcriptSpec{
	1:  {1, [8]operandKind{requiredID, optionalID, numeric, numeric, requiredID, requiredID}},
	2:  {2, [8]operandKind{requiredID, requiredID, requiredID, requiredID}},
	3:  {3, [8]operandKind{requiredID, requiredID, numeric, requiredID, requiredID}},
	4:  {4, [8]operandKind{requiredID, requiredID, requiredID, requiredID, requiredID}},
	5:  {5, [8]operandKind{requiredID, requiredID, requiredID, optionalID, requiredID}},
	6:  {6, [8]operandKind{requiredID, requiredID, optionalID, numeric, requiredID, requiredID}},
	7:  {7, [8]operandKind{requiredID, optionalID, optionalID, numeric, requiredID, requiredID}},
	8:  {8, [8]operandKind{optionalID, optionalID, numeric, optionalID, optionalID, requiredID, requiredID}},
	9:  {9, [8]operandKind{requiredID, requiredID, optionalID, optionalID, optionalID, requiredID, requiredID}},
	10: {10, [8]operandKind{requiredID, optionalID, requiredID, optionalID, optionalID, requiredID, requiredID}},
	11: {11, [8]operandKind{requiredID, numeric}},
	12: {12, [8]operandKind{requiredID, optionalID, optionalID, requiredID, requiredID}},
	13: {12, [8]operandKind{requiredID, requiredID, requiredID, numeric, requiredID, requiredID}},
	14: {12, [8]operandKind{requiredID, requiredID, requiredID, requiredID, requiredID, requiredID}},
	15: {12, [8]operandKind{requiredID, requiredID, optionalID, optionalID, requiredID, requiredID}},
	16: {12, [8]operandKind{requiredID, optionalID, numeric, requiredID, requiredID}},
	17: {12, [8]operandKind{requiredID, optionalID, requiredID, requiredID}},
	18: {12, [8]operandKind{requiredID, optionalID, numeric, requiredID, requiredID}},
}

func EncodeTranscript(events []TranscriptEvent) (TranscriptBundle, error) {
	dictionarySet := map[string]bool{}
	var vector [12]int64
	for _, event := range events {
		spec, ok := transcriptSpecs[event.Code]
		if !ok || event.Category != spec.category {
			return TranscriptBundle{}, fmt.Errorf("invalid transition code/category %d/%d", event.Code, event.Category)
		}
		vector[event.Category-1]++
		for index, kind := range spec.kinds {
			operand := event.Operands[index]
			switch kind {
			case omit:
				if !operand.Absent && (operand.Numeric || operand.Text != "" || operand.Number != 0) {
					return TranscriptBundle{}, fmt.Errorf("code %d operand %d must be omitted", event.Code, index)
				}
			case numeric:
				if !operand.Numeric || operand.Absent || operand.Text != "" {
					return TranscriptBundle{}, fmt.Errorf("code %d operand %d must be numeric", event.Code, index)
				}
			case requiredID:
				if operand.Numeric || operand.Absent || !validDictionaryEntry(operand.Text) {
					return TranscriptBundle{}, fmt.Errorf("code %d operand %d must be a required id", event.Code, index)
				}
				dictionarySet[operand.Text] = true
			case optionalID:
				if operand.Absent {
					continue
				}
				if operand.Numeric || !validDictionaryEntry(operand.Text) {
					return TranscriptBundle{}, fmt.Errorf("code %d operand %d must be an optional id", event.Code, index)
				}
				dictionarySet[operand.Text] = true
			}
		}
	}
	dictionary := make([]string, 0, len(dictionarySet))
	for entry := range dictionarySet {
		dictionary = append(dictionary, entry)
	}
	sort.Slice(dictionary, func(i, j int) bool {
		left, right := sha256.Sum256([]byte(dictionary[i])), sha256.Sum256([]byte(dictionary[j]))
		if comparison := bytes.Compare(left[:], right[:]); comparison != 0 {
			return comparison < 0
		}
		return dictionary[i] < dictionary[j]
	})
	dictionaryBytes := encodeDictionary(dictionary)
	if len(dictionaryBytes) > transcriptDictCap {
		return TranscriptBundle{}, fmt.Errorf("transcript dictionary exceeds cap")
	}
	dictionaryDigest := sha256.Sum256(dictionaryBytes)
	header := make([]byte, transcriptHeaderSize)
	copy(header[0:4], "NGT1")
	binary.BigEndian.PutUint16(header[4:6], 1)
	binary.BigEndian.PutUint16(header[6:8], transcriptRecordSize)
	binary.BigEndian.PutUint64(header[8:16], uint64(len(dictionaryBytes)))
	binary.BigEndian.PutUint64(header[16:24], uint64(len(events)))
	copy(header[24:56], dictionaryDigest[:])
	ids := make(map[string]int32, len(dictionary))
	for index, entry := range dictionary {
		ids[entry] = int32(index + 1)
	}
	raw := make([]byte, 0, len(header)+len(dictionaryBytes)+len(events)*transcriptRecordSize)
	raw = append(raw, header...)
	raw = append(raw, dictionaryBytes...)
	for sequence, event := range events {
		record := make([]byte, transcriptRecordSize)
		record[0], record[1], record[2] = 1, event.Category, event.Code
		binary.BigEndian.PutUint32(record[4:8], event.TaskOrdinal)
		binary.BigEndian.PutUint64(record[8:16], uint64(sequence))
		spec := transcriptSpecs[event.Code]
		for index, kind := range spec.kinds {
			value := int32(0)
			if kind == numeric {
				value = event.Operands[index].Number
			} else if (kind == requiredID || kind == optionalID) && !event.Operands[index].Absent {
				value = ids[event.Operands[index].Text]
			}
			binary.BigEndian.PutUint32(record[16+index*4:20+index*4], uint32(value))
		}
		tupleDigest := sha256.New()
		tupleDigest.Write([]byte("ngt-tuple/v1\x00"))
		tupleDigest.Write(record[:48])
		copy(record[48:80], tupleDigest.Sum(nil))
		raw = append(raw, record...)
	}
	compressed, err := canonicalGzip(raw)
	if err != nil {
		return TranscriptBundle{}, err
	}
	return TranscriptBundle{Raw: raw, Gzip: compressed, Dictionary: dictionary, Vector: vector}, nil
}

func validDictionaryEntry(value string) bool {
	return value != "" && len(value) <= 128 && utf8.ValidString(value) && !bytes.ContainsRune([]byte(value), '\x00')
}

func encodeDictionary(entries []string) []byte {
	var output bytes.Buffer
	_ = binary.Write(&output, binary.BigEndian, uint32(len(entries)))
	for _, entry := range entries {
		_ = binary.Write(&output, binary.BigEndian, uint16(len(entry)))
		output.WriteString(entry)
	}
	return output.Bytes()
}

func DecodeTranscript(raw []byte) (TranscriptBundle, error) {
	if len(raw) < transcriptHeaderSize || string(raw[:4]) != "NGT1" || binary.BigEndian.Uint16(raw[4:6]) != 1 || binary.BigEndian.Uint16(raw[6:8]) != transcriptRecordSize {
		return TranscriptBundle{}, fmt.Errorf("invalid transcript header")
	}
	for _, value := range raw[56:transcriptHeaderSize] {
		if value != 0 {
			return TranscriptBundle{}, fmt.Errorf("nonzero header padding")
		}
	}
	dictionaryLength := binary.BigEndian.Uint64(raw[8:16])
	eventCount := binary.BigEndian.Uint64(raw[16:24])
	if dictionaryLength > transcriptDictCap || dictionaryLength > uint64(len(raw)-transcriptHeaderSize) || eventCount > (uint64(len(raw))-transcriptHeaderSize-dictionaryLength)/transcriptRecordSize || uint64(len(raw)) != transcriptHeaderSize+dictionaryLength+eventCount*transcriptRecordSize {
		return TranscriptBundle{}, fmt.Errorf("invalid transcript size")
	}
	dictionaryBytes := raw[transcriptHeaderSize : transcriptHeaderSize+int(dictionaryLength)]
	digest := sha256.Sum256(dictionaryBytes)
	if !bytes.Equal(digest[:], raw[24:56]) {
		return TranscriptBundle{}, fmt.Errorf("dictionary digest mismatch")
	}
	dictionary, err := decodeDictionary(dictionaryBytes)
	if err != nil {
		return TranscriptBundle{}, err
	}
	if !slices.Equal(encodeDictionary(dictionary), dictionaryBytes) {
		return TranscriptBundle{}, fmt.Errorf("noncanonical dictionary")
	}
	var vector [12]int64
	events := make([]TranscriptEvent, 0, int(eventCount))
	offset := transcriptHeaderSize + int(dictionaryLength)
	for sequence := uint64(0); sequence < eventCount; sequence++ {
		record := raw[offset : offset+transcriptRecordSize]
		offset += transcriptRecordSize
		if record[0] != 1 || record[3] != 0 || binary.BigEndian.Uint64(record[8:16]) != sequence {
			return TranscriptBundle{}, fmt.Errorf("invalid record envelope at sequence %d", sequence)
		}
		spec, ok := transcriptSpecs[record[2]]
		if !ok || record[1] != spec.category {
			return TranscriptBundle{}, fmt.Errorf("invalid record code/category")
		}
		want := sha256.New()
		want.Write([]byte("ngt-tuple/v1\x00"))
		want.Write(record[:48])
		if !bytes.Equal(want.Sum(nil), record[48:80]) || !bytes.Equal(record[80:96], make([]byte, 16)) {
			return TranscriptBundle{}, fmt.Errorf("record digest/padding mismatch")
		}
		for index, kind := range spec.kinds {
			value := int32(binary.BigEndian.Uint32(record[16+index*4 : 20+index*4]))
			switch kind {
			case omit:
				if value != 0 {
					return TranscriptBundle{}, fmt.Errorf("nonzero omitted operand")
				}
			case requiredID:
				if value <= 0 || int(value) > len(dictionary) {
					return TranscriptBundle{}, fmt.Errorf("invalid required dictionary id")
				}
			case optionalID:
				if value < 0 || int(value) > len(dictionary) {
					return TranscriptBundle{}, fmt.Errorf("invalid optional dictionary id")
				}
			}
		}
		event := TranscriptEvent{Category: record[1], Code: record[2], TaskOrdinal: binary.BigEndian.Uint32(record[4:8])}
		for index, kind := range spec.kinds {
			value := int32(binary.BigEndian.Uint32(record[16+index*4 : 20+index*4]))
			switch kind {
			case omit:
				event.Operands[index] = Omitted()
			case numeric:
				event.Operands[index] = Number(value)
			case requiredID, optionalID:
				if value == 0 {
					event.Operands[index] = Omitted()
				} else {
					event.Operands[index] = ID(dictionary[value-1])
				}
			}
		}
		events = append(events, event)
		vector[record[1]-1]++
	}
	return TranscriptBundle{Raw: slices.Clone(raw), Dictionary: dictionary, Events: events, Vector: vector}, nil
}

func decodeDictionary(data []byte) ([]string, error) {
	reader := bytes.NewReader(data)
	var count uint32
	if err := binary.Read(reader, binary.BigEndian, &count); err != nil {
		return nil, err
	}
	entries := make([]string, 0, count)
	seen := map[string]bool{}
	for index := uint32(0); index < count; index++ {
		var length uint16
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil || length == 0 || length > 128 {
			return nil, fmt.Errorf("invalid dictionary length")
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		entry := string(payload)
		if !validDictionaryEntry(entry) || seen[entry] {
			return nil, fmt.Errorf("invalid dictionary entry")
		}
		seen[entry] = true
		entries = append(entries, entry)
	}
	if reader.Len() != 0 {
		return nil, fmt.Errorf("trailing dictionary bytes")
	}
	return entries, nil
}
