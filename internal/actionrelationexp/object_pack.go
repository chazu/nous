package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/chazu/nous/internal/actionrelationwire"
)

const (
	ObjectHeader        = "AROP1\n"
	IndexHeader         = "ARIX1\n"
	MaximumPackBytes    = 16 * 1024 * 1024
	MaximumIndexRows    = 4096
	MaximumIndexBytes   = 1024 * 1024
	ObjectIndexRowBytes = 96
)

const (
	MaximumCurriculumObjects = 65_248
	MaximumCurriculumIndexes = 19
)

var objectKinds = map[uint16]string{
	1: "finite-action-state/v1", 2: "finite-action-semantic/v1", 3: "action-occurrence/v1", 4: "finite-action-world-core/v1",
	5: "remaining-occurrences/v1", 6: "action-relation-pattern/v1", 7: "action-guard/v1", 8: "action-local-facts/v1",
	9: "guarded-action-relation/v1", 10: "guarded-action-artifact/v1", 11: "action-training-evidence/v1", 12: "action-presentation-view/v1",
	13: "action-normalization-proof/v1", 14: "learned-witness/v1", 15: "static-witness/v1", 16: "dynamic-witness/v1",
	17: "local-diamond-certificate/v1", 18: "sleep-propagation-core/v1", 19: "sleep-proof-map/v1", 20: "sleep-search-node/v1",
	21: "sleep-search-edge/v1", 22: "completed-subtree/v2", 23: "action-terminal/v1", 24: "sleep-terminal-set/v1",
	25: "sleep-subtree-root/v1", 26: "certificate-cache-row/v3", 27: "compound-work-reservation/v1", 28: "action-guard-search-barrier/v1",
	29: "action-scorer-truth-shard/v1", 32: "actionrelation-world-policy-row/v2", 33: "actionrelation-curriculum-policy-row/v2",
	35: "action-store-boundary/v3", 36: "action-generator-attempt-ledger/v2", 37: "action-validity-row/v1", 38: "action-applicability-row/v1",
	39: "action-transition-row/v1", 40: "action-state-equality-row/v1", 41: "action-literal-evaluation-row/v1", 42: "action-relation-match-row/v1",
	43: "action-unanimous-use/v1", 44: "local-diamond-certificate-attempt/v3", 45: "action-raw-input/v1", 46: "actionrelation-operation-root/v1",
	47: "actionrelation-curriculum-fixture/v1", 48: "action-static-footprint-row/v1", 49: "action-work-terminal/v1",
}

type ObjectScope struct {
	Curriculum int
	Class      string
}

func (s ObjectScope) value() []any { return []any{"curriculum", s.Curriculum, s.Class} }

func (s ObjectScope) validate() error {
	if s.Curriculum < 0 {
		return fmt.Errorf("invalid curriculum")
	}
	switch s.Class {
	case "acquisition-nous-preboundary", "acquisition-no-guard-preboundary", "utility", "authority":
		return nil
	default:
		return fmt.Errorf("invalid object scope class %q", s.Class)
	}
}

type ObjectRecord struct {
	Kind  uint16
	Bytes []byte
}

type EvidenceFile struct {
	Path string
	Mode string
	Data []byte
}

type ObjectPackShard struct {
	PackOrdinal int
	Path        string
	FirstDigest string
	LastDigest  string
	RecordCount int
	ByteLength  int
	PackDigest  string
}

type ObjectPackRoot struct {
	Scope        ObjectScope
	TotalRecords int
	Shards       []ObjectPackShard
}

func (r ObjectPackRoot) CanonicalJSON() ([]byte, error) {
	if err := r.Scope.validate(); err != nil || r.TotalRecords < 1 || r.TotalRecords > MaximumCurriculumObjects || len(r.Shards) < 1 {
		return nil, fmt.Errorf("invalid object pack root")
	}
	rows := make([]any, len(r.Shards))
	total := 0
	previous := ""
	for index, shard := range r.Shards {
		if shard.PackOrdinal != index || !safeEvidencePath(shard.Path) || !digestText(shard.FirstDigest) || !digestText(shard.LastDigest) || shard.FirstDigest > shard.LastDigest || shard.RecordCount < 1 || shard.ByteLength < len(ObjectHeader)+4 || shard.ByteLength > MaximumPackBytes || !digestText(shard.PackDigest) || previous != "" && shard.FirstDigest <= previous {
			return nil, fmt.Errorf("invalid object shard %d", index)
		}
		rows[index] = []any{shard.PackOrdinal, shard.Path, shard.FirstDigest, shard.LastDigest, shard.RecordCount, shard.ByteLength, shard.PackDigest}
		total += shard.RecordCount
		previous = shard.LastDigest
	}
	if total != r.TotalRecords {
		return nil, fmt.Errorf("object root count mismatch")
	}
	return json.Marshal([]any{"actionrelation-pack-root/v1", "object", r.Scope.value(), 0, r.TotalRecords, rows})
}

func (r ObjectPackRoot) Digest() (string, error) { return canonicalDigest(r.CanonicalJSON()) }

type IndexShard struct {
	ShardOrdinal int
	Path         string
	FirstDigest  string
	LastDigest   string
	RowCount     int
	ByteLength   int
	PackDigest   string
}

type IndexRoot struct {
	Scope                ObjectScope
	ObjectPackRootDigest string
	TotalRows            int
	ObjectSetRoot        string
	Shards               []IndexShard
}

func (r IndexRoot) CanonicalJSON() ([]byte, error) {
	if err := r.Scope.validate(); err != nil || !digestText(r.ObjectPackRootDigest) || !digestText(r.ObjectSetRoot) || r.TotalRows < 1 || r.TotalRows > MaximumCurriculumObjects || len(r.Shards) < 1 || len(r.Shards) > MaximumCurriculumIndexes {
		return nil, fmt.Errorf("invalid index root")
	}
	rows := make([]any, len(r.Shards))
	total := 0
	previous := ""
	for index, shard := range r.Shards {
		if shard.ShardOrdinal != index || !safeEvidencePath(shard.Path) || !digestText(shard.FirstDigest) || !digestText(shard.LastDigest) || shard.FirstDigest > shard.LastDigest || shard.RowCount < 1 || shard.RowCount > MaximumIndexRows || shard.ByteLength != len(IndexHeader)+shard.RowCount*ObjectIndexRowBytes || shard.ByteLength > MaximumIndexBytes || !digestText(shard.PackDigest) || previous != "" && shard.FirstDigest <= previous {
			return nil, fmt.Errorf("invalid index shard %d", index)
		}
		rows[index] = []any{shard.ShardOrdinal, shard.Path, shard.FirstDigest, shard.LastDigest, shard.RowCount, shard.ByteLength, shard.PackDigest}
		total += shard.RowCount
		previous = shard.LastDigest
	}
	if total != r.TotalRows {
		return nil, fmt.Errorf("index root count mismatch")
	}
	return json.Marshal([]any{"actionrelation-index-root/v1", r.Scope.value(), r.ObjectPackRootDigest, r.TotalRows, r.ObjectSetRoot, rows})
}

func (r IndexRoot) Digest() (string, error) { return canonicalDigest(r.CanonicalJSON()) }

type ObjectBundle struct {
	Scope       ObjectScope
	ObjectFiles []EvidenceFile
	IndexFiles  []EvidenceFile
	ObjectRoot  ObjectPackRoot
	IndexRoot   IndexRoot
}

type packedObject struct {
	kind       uint16
	digest     string
	digestRaw  [32]byte
	bytes      []byte
	pack       uint16
	offset     uint64
	frameBytes int
}

func BuildObjectBundle(scope ObjectScope, records []ObjectRecord) (ObjectBundle, error) {
	root, _ := EvidenceRoot("development")
	return BuildObjectBundleAt(root, scope, records)
}

func BuildObjectBundleAt(evidenceRoot string, scope ObjectScope, records []ObjectRecord) (ObjectBundle, error) {
	return buildObjectBundle(evidenceRoot, scope, records, MaximumPackBytes, MaximumIndexRows, MaximumIndexBytes)
}

func buildObjectBundle(evidenceRoot string, scope ObjectScope, records []ObjectRecord, maxPackBytes, maxIndexRows, maxIndexBytes int) (ObjectBundle, error) {
	if !validEvidenceRoot(evidenceRoot) || scope.validate() != nil || len(records) == 0 || len(records) > MaximumCurriculumObjects || maxPackBytes <= len(ObjectHeader)+4 || maxIndexRows < 1 || maxIndexBytes < len(IndexHeader)+ObjectIndexRowBytes {
		return ObjectBundle{}, fmt.Errorf("invalid object bundle shape")
	}
	objects := make([]packedObject, len(records))
	seen := map[string]bool{}
	for index, record := range records {
		if err := ValidateObject(record.Kind, record.Bytes); err != nil {
			return ObjectBundle{}, fmt.Errorf("object %d: %w", index, err)
		}
		digestRaw := sha256.Sum256(record.Bytes)
		digest := hex.EncodeToString(digestRaw[:])
		if seen[digest] {
			return ObjectBundle{}, fmt.Errorf("duplicate object digest %s", digest)
		}
		seen[digest] = true
		objects[index] = packedObject{kind: record.Kind, digest: digest, digestRaw: digestRaw, bytes: bytes.Clone(record.Bytes), frameBytes: 4 + len(record.Bytes)}
	}
	slices.SortFunc(objects, func(a, b packedObject) int { return bytes.Compare(a.digestRaw[:], b.digestRaw[:]) })

	var objectFiles []EvidenceFile
	var objectShards []ObjectPackShard
	for first := 0; first < len(objects); {
		last, size := first, len(ObjectHeader)
		for last < len(objects) && size+objects[last].frameBytes <= maxPackBytes {
			size += objects[last].frameBytes
			last++
		}
		if last == first {
			return ObjectBundle{}, fmt.Errorf("object frame exceeds pack cap")
		}
		packOrdinal := len(objectFiles)
		if packOrdinal > int(^uint16(0)) {
			return ObjectBundle{}, fmt.Errorf("too many object packs")
		}
		data := make([]byte, size)
		copy(data, ObjectHeader)
		offset := len(ObjectHeader)
		for index := first; index < last; index++ {
			binary.BigEndian.PutUint32(data[offset:offset+4], uint32(len(objects[index].bytes)))
			offset += 4
			objects[index].pack = uint16(packOrdinal)
			objects[index].offset = uint64(offset)
			copy(data[offset:], objects[index].bytes)
			offset += len(objects[index].bytes)
		}
		path := fmt.Sprintf("%s/packs/curriculum-%04d/%s/object-%04d.arop", evidenceRoot, scope.Curriculum, scope.Class, packOrdinal)
		digest := sha256.Sum256(data)
		objectFiles = append(objectFiles, EvidenceFile{Path: path, Mode: "100644", Data: data})
		objectShards = append(objectShards, ObjectPackShard{PackOrdinal: packOrdinal, Path: path, FirstDigest: objects[first].digest, LastDigest: objects[last-1].digest, RecordCount: last - first, ByteLength: len(data), PackDigest: hex.EncodeToString(digest[:])})
		first = last
	}
	objectRoot := ObjectPackRoot{Scope: scope, TotalRecords: len(objects), Shards: objectShards}
	objectRootDigest, err := objectRoot.Digest()
	if err != nil {
		return ObjectBundle{}, err
	}

	objectSetRows := make([]any, len(objects))
	byKind := slices.Clone(objects)
	slices.SortFunc(byKind, func(a, b packedObject) int {
		if a.kind != b.kind {
			return int(a.kind) - int(b.kind)
		}
		return bytes.Compare(a.digestRaw[:], b.digestRaw[:])
	})
	for index, object := range byKind {
		objectSetRows[index] = []any{object.kind, object.digest}
	}
	objectSetRoot, err := actionrelationwire.RootDigest("indexed-object-set", []any{scope.value(), objectSetRows})
	if err != nil {
		return ObjectBundle{}, err
	}

	var indexFiles []EvidenceFile
	var indexShards []IndexShard
	rowLimit := min(maxIndexRows, (maxIndexBytes-len(IndexHeader))/ObjectIndexRowBytes)
	for first := 0; first < len(objects); {
		last := min(len(objects), first+rowLimit)
		data := make([]byte, len(IndexHeader)+(last-first)*ObjectIndexRowBytes)
		copy(data, IndexHeader)
		for index := first; index < last; index++ {
			row := data[len(IndexHeader)+(index-first)*ObjectIndexRowBytes:]
			copy(row[:32], objects[index].digestRaw[:])
			binary.BigEndian.PutUint64(row[32:40], objects[index].offset)
			binary.BigEndian.PutUint32(row[40:44], uint32(len(objects[index].bytes)))
			binary.BigEndian.PutUint16(row[44:46], objects[index].kind)
			binary.BigEndian.PutUint16(row[46:48], objects[index].pack)
		}
		ordinal := len(indexFiles)
		path := fmt.Sprintf("%s/packs/curriculum-%04d/%s/index-%04d.arix", evidenceRoot, scope.Curriculum, scope.Class, ordinal)
		digest := sha256.Sum256(data)
		indexFiles = append(indexFiles, EvidenceFile{Path: path, Mode: "100644", Data: data})
		indexShards = append(indexShards, IndexShard{ShardOrdinal: ordinal, Path: path, FirstDigest: objects[first].digest, LastDigest: objects[last-1].digest, RowCount: last - first, ByteLength: len(data), PackDigest: hex.EncodeToString(digest[:])})
		first = last
	}
	indexRoot := IndexRoot{Scope: scope, ObjectPackRootDigest: objectRootDigest, TotalRows: len(objects), ObjectSetRoot: objectSetRoot, Shards: indexShards}
	if len(indexShards) > MaximumCurriculumIndexes {
		return ObjectBundle{}, fmt.Errorf("object index exceeds frozen shard cap")
	}
	if _, err := indexRoot.CanonicalJSON(); err != nil {
		return ObjectBundle{}, err
	}
	return ObjectBundle{Scope: scope, ObjectFiles: objectFiles, IndexFiles: indexFiles, ObjectRoot: objectRoot, IndexRoot: indexRoot}, nil
}

func validEvidenceRoot(root string) bool {
	for _, panel := range []string{"development", "validation", "locked"} {
		want, _ := EvidenceRoot(panel)
		if root == want {
			return true
		}
	}
	return false
}

func ValidateObject(kind uint16, data []byte) error {
	want, ok := objectKinds[kind]
	limit := 1024
	if kind == 9 || kind == 35 || kind == 47 {
		limit = 4096
	}
	if kind == 10 || kind == 28 || kind == 29 || kind == 36 {
		limit = 65536
	}
	if !ok || len(data) == 0 || len(data) > limit || !utf8.Valid(data) {
		return fmt.Errorf("invalid object kind or size")
	}
	var row []json.RawMessage
	if err := json.Unmarshal(data, &row); err != nil || len(row) == 0 {
		return fmt.Errorf("invalid object JSON")
	}
	canonical, err := json.Marshal(row)
	if err != nil || !bytes.Equal(canonical, data) {
		return fmt.Errorf("noncanonical object JSON")
	}
	var got string
	if err := json.Unmarshal(row[0], &got); err != nil || got != want {
		return fmt.Errorf("kind %d decodes as %q want %q", kind, got, want)
	}
	if err := validateTypedObject(kind, data, row); err != nil {
		return fmt.Errorf("kind %d typed decode: %w", kind, err)
	}
	return nil
}

func VerifyObjectBundle(bundle ObjectBundle) error {
	return verifyObjectBundle(bundle, MaximumPackBytes)
}

func verifyObjectBundle(bundle ObjectBundle, maxPackBytes int) error {
	if maxPackBytes <= len(ObjectHeader)+4 || maxPackBytes > MaximumPackBytes {
		return fmt.Errorf("invalid object pack cap")
	}
	if err := bundle.Scope.validate(); err != nil || bundle.Scope != bundle.ObjectRoot.Scope || bundle.Scope != bundle.IndexRoot.Scope || len(bundle.ObjectFiles) != len(bundle.ObjectRoot.Shards) || len(bundle.IndexFiles) != len(bundle.IndexRoot.Shards) {
		return fmt.Errorf("object bundle authority mismatch")
	}
	objectRootDigest, err := bundle.ObjectRoot.Digest()
	if err != nil || objectRootDigest != bundle.IndexRoot.ObjectPackRootDigest {
		return fmt.Errorf("object root digest mismatch")
	}
	if _, err := bundle.IndexRoot.CanonicalJSON(); err != nil {
		return err
	}
	objects := map[string]packedObject{}
	orderedDigests := make([]string, 0, bundle.ObjectRoot.TotalRecords)
	for packOrdinal, file := range bundle.ObjectFiles {
		shard := bundle.ObjectRoot.Shards[packOrdinal]
		if file.Path != shard.Path || file.Mode != "100644" || !safeEvidencePath(file.Path) || len(file.Data) != shard.ByteLength || len(file.Data) < len(ObjectHeader) || string(file.Data[:len(ObjectHeader)]) != ObjectHeader || shaHex(file.Data) != shard.PackDigest {
			return fmt.Errorf("invalid object pack %d", packOrdinal)
		}
		offset, count := len(ObjectHeader), 0
		for offset < len(file.Data) {
			if offset+4 > len(file.Data) {
				return fmt.Errorf("truncated object prefix")
			}
			length := int(binary.BigEndian.Uint32(file.Data[offset : offset+4]))
			offset += 4
			if length < 1 || offset+length > len(file.Data) {
				return fmt.Errorf("invalid object frame")
			}
			payload := file.Data[offset : offset+length]
			digestRaw := sha256.Sum256(payload)
			digest := hex.EncodeToString(digestRaw[:])
			if _, duplicate := objects[digest]; duplicate || len(orderedDigests) > 0 && digest <= orderedDigests[len(orderedDigests)-1] {
				return fmt.Errorf("duplicate or unordered object")
			}
			objects[digest] = packedObject{digest: digest, digestRaw: digestRaw, bytes: payload, pack: uint16(packOrdinal), offset: uint64(offset)}
			orderedDigests = append(orderedDigests, digest)
			offset += length
			count++
		}
		if count != shard.RecordCount || orderedDigests[len(orderedDigests)-count] != shard.FirstDigest || orderedDigests[len(orderedDigests)-1] != shard.LastDigest {
			return fmt.Errorf("object shard range mismatch")
		}
		if packOrdinal+1 < len(bundle.ObjectFiles) {
			next := bundle.ObjectFiles[packOrdinal+1]
			if len(next.Data) < len(ObjectHeader)+4 {
				return fmt.Errorf("invalid next object pack")
			}
			nextLength := int(binary.BigEndian.Uint32(next.Data[len(ObjectHeader) : len(ObjectHeader)+4]))
			if len(file.Data)+4+nextLength <= maxPackBytes {
				return fmt.Errorf("non-greedy object shard")
			}
		}
	}
	if len(objects) != bundle.ObjectRoot.TotalRecords {
		return fmt.Errorf("object total mismatch")
	}

	var objectSetRows []any
	var typed []packedObject
	for shardOrdinal, file := range bundle.IndexFiles {
		shard := bundle.IndexRoot.Shards[shardOrdinal]
		if file.Path != shard.Path || file.Mode != "100644" || len(file.Data) != shard.ByteLength || len(file.Data) < len(IndexHeader) || string(file.Data[:len(IndexHeader)]) != IndexHeader || shaHex(file.Data) != shard.PackDigest || (len(file.Data)-len(IndexHeader))%ObjectIndexRowBytes != 0 {
			return fmt.Errorf("invalid index pack %d", shardOrdinal)
		}
		count := (len(file.Data) - len(IndexHeader)) / ObjectIndexRowBytes
		if count != shard.RowCount || count > MaximumIndexRows || len(file.Data) > MaximumIndexBytes {
			return fmt.Errorf("invalid index shard count")
		}
		for index := 0; index < count; index++ {
			row := file.Data[len(IndexHeader)+index*ObjectIndexRowBytes:][:ObjectIndexRowBytes]
			if !allZero(row[48:]) {
				return fmt.Errorf("nonzero index padding")
			}
			digest := hex.EncodeToString(row[:32])
			if len(typed) > 0 && digest <= typed[len(typed)-1].digest {
				return fmt.Errorf("unordered index")
			}
			object, ok := objects[digest]
			kind := binary.BigEndian.Uint16(row[44:46])
			if !ok || binary.BigEndian.Uint64(row[32:40]) != object.offset || int(binary.BigEndian.Uint32(row[40:44])) != len(object.bytes) || binary.BigEndian.Uint16(row[46:48]) != object.pack || ValidateObject(kind, object.bytes) != nil {
				return fmt.Errorf("index row does not resolve exact object")
			}
			object.kind = kind
			typed = append(typed, object)
		}
		if typed[len(typed)-count].digest != shard.FirstDigest || typed[len(typed)-1].digest != shard.LastDigest {
			return fmt.Errorf("index shard range mismatch")
		}
	}
	if len(typed) != len(objects) {
		return fmt.Errorf("index coverage mismatch")
	}
	slices.SortFunc(typed, func(a, b packedObject) int {
		if a.kind != b.kind {
			return int(a.kind) - int(b.kind)
		}
		return bytes.Compare(a.digestRaw[:], b.digestRaw[:])
	})
	for _, object := range typed {
		objectSetRows = append(objectSetRows, []any{object.kind, object.digest})
	}
	objectSetRoot, _ := actionrelationwire.RootDigest("indexed-object-set", []any{bundle.Scope.value(), objectSetRows})
	if objectSetRoot != bundle.IndexRoot.ObjectSetRoot {
		return fmt.Errorf("object set root mismatch")
	}
	return nil
}

func safeEvidencePath(path string) bool {
	if path == "" || len(path) > 192 || filepath.IsAbs(path) || strings.Contains(path, "\\") || !utf8.ValidString(path) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, character := range part {
			if character > 127 || !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("._-", character)) {
				return false
			}
		}
	}
	return true
}

func canonicalDigest(data []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func shaHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}
