package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

// RetainedPackRefs is the complete physical-pack authority reachable from a
// panel payload. Read must return the exact bytes retained at a logical path.
type RetainedPackRefs struct {
	Panel           string
	Authority       string
	RunEvidence     AuthorityRef
	ObjectRoots     []ObjectManifestRef
	IndexRoots      []ObjectManifestRef
	JournalRoots    []TranscriptManifestRef
	InputRoots      []TranscriptManifestRef
	DetailRoots     []TranscriptManifestRef
	Tables          []TableManifestRef
	StructuralMaps  []AuthorityRef
	StoreBoundaries []StoreBoundaryRow
}

// VerifyRetainedPacks reopens every manifest, follows every physical shard,
// validates its complete binary representation, and closes the per-run roots
// back to the run-evidence pack. The returned paths are the exact reachable
// manifest and pack set (the fixture root is intentionally outside this DAG).
func VerifyRetainedPacks(value RetainedPackRefs, read func(string) ([]byte, error)) ([]string, error) {
	curricula := panelRunCounts[value.Panel] / 44
	if !validPanelAuthority(value.Panel, value.Authority) || curricula == 0 || read == nil || value.RunEvidence.Verify() != nil ||
		len(value.ObjectRoots) != curricula*4 || len(value.IndexRoots) != curricula*4 ||
		len(value.JournalRoots) != curricula*44 || len(value.InputRoots) != curricula*44 || len(value.DetailRoots) != curricula*44 ||
		len(value.Tables) != curricula*14 || len(value.StructuralMaps) != curricula || len(value.StoreBoundaries) != curricula*2 {
		return nil, fmt.Errorf("invalid retained pack authority")
	}
	reachable := map[string]bool{}
	retainedLeaves := map[string]bool{}
	type retainedObject struct {
		kind      uint16
		canonical []byte
	}
	retainedObjects := map[string]retainedObject{}
	readPath := func(path, digest string) ([]byte, error) {
		if !safeEvidencePath(path) || !digestText(digest) || reachable[path] {
			return nil, fmt.Errorf("duplicate or invalid retained path %q", path)
		}
		data, err := read(path)
		if err != nil || shaHex(data) != digest {
			return nil, fmt.Errorf("retained digest mismatch: %s", path)
		}
		reachable[path] = true
		return data, nil
	}
	readFile := func(path string, length int, digest string) (EvidenceFile, error) {
		data, err := readPath(path, digest)
		if err != nil || len(data) != length {
			return EvidenceFile{}, fmt.Errorf("retained physical length mismatch: %s", path)
		}
		return EvidenceFile{Path: path, Mode: "100644", Data: data}, nil
	}

	type scopeKey struct {
		curriculum int
		class      string
	}
	type indexAuthority struct{ digest, objectSet string }
	indexAuthorities := map[scopeKey]indexAuthority{}
	indexes := make(map[scopeKey]ObjectManifestRef, len(value.IndexRoots))
	for _, ref := range value.IndexRoots {
		key := scopeKey{ref.Scope.Curriculum, ref.Scope.Class}
		if ref.Scope.validate() != nil || indexes[key].Path != "" {
			return nil, fmt.Errorf("duplicate or invalid retained index scope")
		}
		indexes[key] = ref
	}
	for _, objectRef := range value.ObjectRoots {
		key := scopeKey{objectRef.Scope.Curriculum, objectRef.Scope.Class}
		indexRef, ok := indexes[key]
		if !ok || objectRef.Scope.validate() != nil {
			return nil, fmt.Errorf("object root lacks matching index root")
		}
		objectBytes, err := readPath(objectRef.Path, objectRef.Digest)
		if err != nil {
			return nil, err
		}
		objectRoot, err := ParseObjectPackRoot(objectBytes)
		if err != nil || objectRoot.Scope != objectRef.Scope {
			return nil, fmt.Errorf("object root does not reconstruct: %s", objectRef.Path)
		}
		indexBytes, err := readPath(indexRef.Path, indexRef.Digest)
		if err != nil {
			return nil, err
		}
		indexRoot, err := ParseIndexRoot(indexBytes)
		if err != nil || indexRoot.Scope != indexRef.Scope {
			return nil, fmt.Errorf("index root does not reconstruct: %s", indexRef.Path)
		}
		bundle := ObjectBundle{Scope: objectRef.Scope, ObjectRoot: objectRoot, IndexRoot: indexRoot}
		objectPayloads := map[string][]byte{}
		for _, shard := range objectRoot.Shards {
			file, err := readFile(shard.Path, shard.ByteLength, shard.PackDigest)
			if err != nil {
				return nil, err
			}
			bundle.ObjectFiles = append(bundle.ObjectFiles, file)
			for offset := len(ObjectHeader); offset < len(file.Data); {
				length := int(binary.BigEndian.Uint32(file.Data[offset : offset+4]))
				offset += 4
				objectDigest := shaHex(file.Data[offset : offset+length])
				retainedLeaves[objectDigest] = true
				objectPayloads[objectDigest] = bytes.Clone(file.Data[offset : offset+length])
				offset += length
			}
		}
		for _, shard := range indexRoot.Shards {
			file, err := readFile(shard.Path, shard.ByteLength, shard.PackDigest)
			if err != nil {
				return nil, err
			}
			bundle.IndexFiles = append(bundle.IndexFiles, file)
		}
		if err := VerifyObjectBundle(bundle); err != nil {
			return nil, fmt.Errorf("retained object bundle %v: %w", key, err)
		}
		for _, file := range bundle.IndexFiles {
			for offset := len(IndexHeader); offset < len(file.Data); offset += ObjectIndexRowBytes {
				row := file.Data[offset:][:ObjectIndexRowBytes]
				digest := hex.EncodeToString(row[:32])
				kind := binary.BigEndian.Uint16(row[44:46])
				if prior, ok := retainedObjects[digest]; ok && (prior.kind != kind || !bytes.Equal(prior.canonical, objectPayloads[digest])) {
					return nil, fmt.Errorf("cross-kind retained object collision")
				}
				retainedObjects[digest] = retainedObject{kind: kind, canonical: objectPayloads[digest]}
			}
		}
		indexDigest, _ := indexRoot.Digest()
		indexAuthorities[key] = indexAuthority{digest: indexDigest, objectSet: indexRoot.ObjectSetRoot}
		delete(indexes, key)
	}
	if len(indexes) != 0 {
		return nil, fmt.Errorf("orphan retained index root")
	}

	type transcriptRoots struct {
		journal TranscriptManifestRef
		input   TranscriptManifestRef
		detail  TranscriptManifestRef
	}
	transcripts := map[string]transcriptRoots{}
	addTranscript := func(class string, refs []TranscriptManifestRef) error {
		for _, ref := range refs {
			roots := transcripts[ref.RunID]
			if !runIDText(ref.RunID) {
				return fmt.Errorf("invalid retained transcript run")
			}
			switch class {
			case "journal":
				if roots.journal.Path != "" {
					return fmt.Errorf("duplicate retained journal")
				}
				roots.journal = ref
			case "input":
				if roots.input.Path != "" {
					return fmt.Errorf("duplicate retained input")
				}
				roots.input = ref
			case "detail":
				if roots.detail.Path != "" {
					return fmt.Errorf("duplicate retained detail")
				}
				roots.detail = ref
			}
			transcripts[ref.RunID] = roots
		}
		return nil
	}
	if err := addTranscript("journal", value.JournalRoots); err != nil {
		return nil, err
	}
	if err := addTranscript("input", value.InputRoots); err != nil {
		return nil, err
	}
	if err := addTranscript("detail", value.DetailRoots); err != nil {
		return nil, err
	}
	journalDigests, inputDigests, detailDigests := map[string]string{}, map[string]string{}, map[string]string{}
	chargedRoots := map[string]string{}
	chargedOutputs := map[string][]string{}
	for runID, refs := range transcripts {
		if refs.journal.Path == "" || refs.input.Path == "" || refs.detail.Path == "" {
			return nil, fmt.Errorf("incomplete retained transcript %s", runID)
		}
		bundle := TranscriptBundle{RunID: runID}
		for _, item := range []struct {
			ref   TranscriptManifestRef
			class string
			root  *TranscriptRoot
			files *[]EvidenceFile
		}{{refs.journal, "journal", &bundle.JournalRoot, &bundle.JournalFiles}, {refs.input, "input", &bundle.InputRoot, &bundle.InputFiles}, {refs.detail, "detail", &bundle.DetailRoot, &bundle.DetailFiles}} {
			manifest, err := readPath(item.ref.Path, item.ref.Digest)
			if err != nil {
				return nil, err
			}
			root, err := ParseTranscriptRoot(manifest)
			if err != nil || root.Class != item.class || root.RunID != runID {
				return nil, fmt.Errorf("retained transcript root does not reconstruct: %s", item.ref.Path)
			}
			*item.root = root
			for _, shard := range root.Shards {
				file, err := readFile(shard.Path, shard.ByteLength, shard.PackDigest)
				if err != nil {
					return nil, err
				}
				*item.files = append(*item.files, file)
			}
		}
		if err := VerifyTranscript(bundle); err != nil {
			return nil, fmt.Errorf("retained transcript %s: %w", runID, err)
		}
		journalDigests[runID], _ = bundle.JournalRoot.Digest()
		inputDigests[runID], _ = bundle.InputRoot.Digest()
		detailDigests[runID], _ = bundle.DetailRoot.Digest()
		chargedRoots[runID], _ = ChargedOutputsRoot(bundle)
		for shardOrdinal, file := range bundle.DetailFiles {
			shard := bundle.DetailRoot.Shards[shardOrdinal]
			for local := 0; local < shard.RecordCount; local++ {
				row := file.Data[len(DetailHeader)+local*DetailRowBytes:][:DetailRowBytes]
				for output := 0; output < int(row[75]); output++ {
					chargedOutputs[runID] = append(chargedOutputs[runID], hex.EncodeToString(row[128+output*32:160+output*32]))
				}
			}
		}
	}

	tableDigests := map[string][]string{}
	for _, ref := range value.Tables {
		manifestBytes, err := readPath(ref.Path, ref.Digest)
		if err != nil {
			return nil, err
		}
		manifest, err := ParseTableManifest(manifestBytes)
		if err != nil || manifest.Curriculum != ref.Curriculum || manifest.Scope != ref.Scope || manifest.Kind != ref.Kind {
			return nil, fmt.Errorf("retained table manifest does not reconstruct: %s", ref.Path)
		}
		bundle := TableBundle{Manifest: manifest}
		for _, shard := range manifest.Shards {
			file, err := readFile(shard.RelativePath, shard.ByteLength, shard.PackDigest)
			if err != nil {
				return nil, err
			}
			bundle.Files = append(bundle.Files, file)
			for ordinal := shard.FirstOrdinal; ordinal <= shard.LastOrdinal; ordinal++ {
				local := int(ordinal - shard.FirstOrdinal)
				start := len(TableHeader) + local*manifest.RecordSize
				leaf := TableLeafDigest(manifest.Kind, ordinal, file.Data[start:start+manifest.RecordSize])
				leafDigest := hex.EncodeToString(leaf[:])
				bundle.LeafDigests = append(bundle.LeafDigests, leafDigest)
				retainedLeaves[leafDigest] = true
			}
		}
		if err := VerifyTableBundle(bundle); err != nil {
			return nil, fmt.Errorf("retained table bundle %s: %w", ref.Path, err)
		}
		tableKey := fmt.Sprintf("%d:%s", ref.Curriculum, ref.Scope)
		tableDigests[tableKey] = append(tableDigests[tableKey], ref.Digest)
	}
	for index, boundary := range value.StoreBoundaries {
		wantCurriculum, wantScope := index/2, "nous"
		if index%2 == 1 {
			wantScope = "no-guard"
		}
		key := scopeKey{curriculum: boundary.Curriculum, class: "acquisition-" + boundary.Scope + "-preboundary"}
		indexAuthority, ok := indexAuthorities[key]
		object, objectOK := retainedObjects[boundary.BoundaryDigest]
		if boundary.Curriculum != wantCurriculum || boundary.Scope != wantScope || !ok || boundary.PreboundaryIndexRoot != indexAuthority.digest || !objectOK || object.kind != 35 {
			return nil, fmt.Errorf("store boundary authority mismatch: %d/%s", boundary.Curriculum, boundary.Scope)
		}
		var row []json.RawMessage
		var tag, scope, objectSet, indexDigest string
		var curriculum int
		var tables []string
		if json.Unmarshal(object.canonical, &row) != nil || len(row) != 6 || json.Unmarshal(row[0], &tag) != nil || tag != "action-store-boundary/v3" || json.Unmarshal(row[1], &curriculum) != nil || json.Unmarshal(row[2], &scope) != nil || json.Unmarshal(row[3], &tables) != nil || json.Unmarshal(row[4], &objectSet) != nil || json.Unmarshal(row[5], &indexDigest) != nil || curriculum != boundary.Curriculum || scope != boundary.Scope || objectSet != indexAuthority.objectSet || indexDigest != indexAuthority.digest || !slices.Equal(tables, tableDigests[fmt.Sprintf("%d:%s", curriculum, scope)]) {
			return nil, fmt.Errorf("store boundary leaf does not reconstruct: %d/%s", boundary.Curriculum, boundary.Scope)
		}
	}

	structuralRoots := map[string]string{}
	structuralObjects := map[string]map[string]bool{}
	for curriculum, ref := range value.StructuralMaps {
		manifest, err := readPath(ref.Path, ref.Digest)
		if err != nil {
			return nil, err
		}
		root, packPath, packLength, packDigest, err := ParseStructuralOutputMap(value.Panel, manifest)
		if err != nil || root.Curriculum != curriculum {
			return nil, fmt.Errorf("retained structural map does not reconstruct: %s", ref.Path)
		}
		if packPath != "" {
			file, err := readFile(packPath, packLength, packDigest)
			if err != nil {
				return nil, err
			}
			root.File = &file
		}
		attributions, err := decodeStructuralAttributions(root)
		if err != nil {
			return nil, fmt.Errorf("retained structural rows %d: %w", curriculum, err)
		}
		rebuilt, err := BuildStructuralOutputMapAt(root.EvidenceRoot, root.Curriculum, root.RunIDs, attributions)
		if err != nil {
			return nil, fmt.Errorf("retained structural rebuild %d: %w", curriculum, err)
		}
		root.RunRoots = rebuilt.RunRoots
		if err := VerifyStructuralOutputMap(root); err != nil {
			return nil, fmt.Errorf("retained structural map %d: %w", curriculum, err)
		}
		for runID, digest := range root.RunRoots {
			if structuralRoots[runID] != "" {
				return nil, fmt.Errorf("duplicate structural run root")
			}
			structuralRoots[runID] = digest
		}
		for _, attribution := range attributions {
			key := fmt.Sprintf("%d:%s", attribution.Kind, attribution.Digest)
			for _, runID := range attribution.RunIDs {
				if structuralObjects[runID] == nil {
					structuralObjects[runID] = map[string]bool{}
				}
				structuralObjects[runID][key] = true
			}
		}
	}

	runManifest, err := readPath(value.RunEvidence.Path, value.RunEvidence.Digest)
	if err != nil {
		return nil, err
	}
	evidenceRoot, _ := EvidenceRoot(value.Panel)
	runPackPath := evidenceRoot + "/packs/run-evidence-0000.arrv"
	runPackBytes, err := read(runPackPath)
	if err != nil {
		return nil, err
	}
	runPack, err := ParseRunEvidencePack(value.Panel, value.Authority, runManifest, runPackBytes)
	if err != nil {
		return nil, err
	}
	if reachable[runPackPath] {
		return nil, fmt.Errorf("duplicate run-evidence pack")
	}
	reachable[runPackPath] = true
	for _, record := range runPack.Records {
		if journalDigests[record.RunID] != record.JournalRoot || inputDigests[record.RunID] != record.InputRoot || detailDigests[record.RunID] != record.DetailRoot || chargedRoots[record.RunID] != record.ChargedRoot || structuralRoots[record.RunID] != record.StructuralRoot {
			return nil, fmt.Errorf("run-evidence row does not close retained run %s", record.RunID)
		}
		if !structuralObjects[record.RunID][fmt.Sprintf("46:%s", record.OperationRoot)] {
			return nil, fmt.Errorf("run-evidence operation root is not structurally retained: %s", record.RunID)
		}
		if record.WorkTerminal != "" && !retainedLeaves[record.WorkTerminal] {
			return nil, fmt.Errorf("run-evidence work terminal is not retained: %s", record.RunID)
		}
		for _, output := range chargedOutputs[record.RunID] {
			if !retainedLeaves[output] {
				return nil, fmt.Errorf("charged output lacks retained typed leaf: %s/%s", record.RunID, output)
			}
		}
	}
	if len(runPack.Records) != len(transcripts) || len(runPack.Records) != len(structuralRoots) {
		return nil, fmt.Errorf("retained run-evidence coverage mismatch")
	}
	paths := make([]string, 0, len(reachable))
	for path := range reachable {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths, nil
}

func ParseObjectPackRoot(data []byte) (ObjectPackRoot, error) {
	var fields []json.RawMessage
	var version, class string
	var reserved int
	var scope []json.RawMessage
	value := ObjectPackRoot{}
	if json.Unmarshal(data, &fields) != nil || len(fields) != 6 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-pack-root/v1" || json.Unmarshal(fields[1], &class) != nil || class != "object" || json.Unmarshal(fields[2], &scope) != nil || len(scope) != 3 || json.Unmarshal(scope[0], &class) != nil || class != "curriculum" || json.Unmarshal(scope[1], &value.Scope.Curriculum) != nil || json.Unmarshal(scope[2], &value.Scope.Class) != nil || json.Unmarshal(fields[3], &reserved) != nil || reserved != 0 || json.Unmarshal(fields[4], &value.TotalRecords) != nil {
		return ObjectPackRoot{}, fmt.Errorf("invalid object root wire")
	}
	var rows [][]json.RawMessage
	if json.Unmarshal(fields[5], &rows) != nil {
		return ObjectPackRoot{}, fmt.Errorf("invalid object shard rows")
	}
	value.Shards = make([]ObjectPackShard, len(rows))
	for index, row := range rows {
		shard := &value.Shards[index]
		if len(row) != 7 || json.Unmarshal(row[0], &shard.PackOrdinal) != nil || json.Unmarshal(row[1], &shard.Path) != nil || json.Unmarshal(row[2], &shard.FirstDigest) != nil || json.Unmarshal(row[3], &shard.LastDigest) != nil || json.Unmarshal(row[4], &shard.RecordCount) != nil || json.Unmarshal(row[5], &shard.ByteLength) != nil || json.Unmarshal(row[6], &shard.PackDigest) != nil {
			return ObjectPackRoot{}, fmt.Errorf("invalid object shard row")
		}
	}
	canonical, err := value.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return ObjectPackRoot{}, fmt.Errorf("object root does not reconstruct")
	}
	return value, nil
}

func ParseIndexRoot(data []byte) (IndexRoot, error) {
	var fields []json.RawMessage
	var version, scopeKind string
	var scope []json.RawMessage
	value := IndexRoot{}
	if json.Unmarshal(data, &fields) != nil || len(fields) != 6 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-index-root/v1" || json.Unmarshal(fields[1], &scope) != nil || len(scope) != 3 || json.Unmarshal(scope[0], &scopeKind) != nil || scopeKind != "curriculum" || json.Unmarshal(scope[1], &value.Scope.Curriculum) != nil || json.Unmarshal(scope[2], &value.Scope.Class) != nil || json.Unmarshal(fields[2], &value.ObjectPackRootDigest) != nil || json.Unmarshal(fields[3], &value.TotalRows) != nil || json.Unmarshal(fields[4], &value.ObjectSetRoot) != nil {
		return IndexRoot{}, fmt.Errorf("invalid index root wire")
	}
	var rows [][]json.RawMessage
	if json.Unmarshal(fields[5], &rows) != nil {
		return IndexRoot{}, fmt.Errorf("invalid index shard rows")
	}
	value.Shards = make([]IndexShard, len(rows))
	for index, row := range rows {
		shard := &value.Shards[index]
		if len(row) != 7 || json.Unmarshal(row[0], &shard.ShardOrdinal) != nil || json.Unmarshal(row[1], &shard.Path) != nil || json.Unmarshal(row[2], &shard.FirstDigest) != nil || json.Unmarshal(row[3], &shard.LastDigest) != nil || json.Unmarshal(row[4], &shard.RowCount) != nil || json.Unmarshal(row[5], &shard.ByteLength) != nil || json.Unmarshal(row[6], &shard.PackDigest) != nil {
			return IndexRoot{}, fmt.Errorf("invalid index shard row")
		}
	}
	canonical, err := value.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return IndexRoot{}, fmt.Errorf("index root does not reconstruct")
	}
	return value, nil
}

func ParseTranscriptRoot(data []byte) (TranscriptRoot, error) {
	var fields []json.RawMessage
	var version string
	value := TranscriptRoot{}
	if json.Unmarshal(data, &fields) != nil || len(fields) != 6 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-pack-root/v1" || json.Unmarshal(fields[1], &value.Class) != nil || json.Unmarshal(fields[2], &value.RunID) != nil || json.Unmarshal(fields[3], &value.RecordSize) != nil || json.Unmarshal(fields[4], &value.TotalRecords) != nil {
		return TranscriptRoot{}, fmt.Errorf("invalid transcript root wire")
	}
	var rows [][]json.RawMessage
	if json.Unmarshal(fields[5], &rows) != nil {
		return TranscriptRoot{}, fmt.Errorf("invalid transcript shard rows")
	}
	value.Shards = make([]TranscriptShard, len(rows))
	for index, row := range rows {
		shard := &value.Shards[index]
		if len(row) != 7 || json.Unmarshal(row[0], &shard.PackOrdinal) != nil || json.Unmarshal(row[1], &shard.Path) != nil || json.Unmarshal(row[2], &shard.FirstSequence) != nil || json.Unmarshal(row[3], &shard.LastSequence) != nil || json.Unmarshal(row[4], &shard.RecordCount) != nil || json.Unmarshal(row[5], &shard.ByteLength) != nil || json.Unmarshal(row[6], &shard.PackDigest) != nil {
			return TranscriptRoot{}, fmt.Errorf("invalid transcript shard row")
		}
	}
	canonical, err := value.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return TranscriptRoot{}, fmt.Errorf("transcript root does not reconstruct")
	}
	return value, nil
}

func ParseTableManifest(data []byte) (TableManifest, error) {
	var fields []json.RawMessage
	var version string
	value := TableManifest{}
	if json.Unmarshal(data, &fields) != nil || len(fields) != 8 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-table-manifest/v3" || json.Unmarshal(fields[1], &value.Curriculum) != nil || json.Unmarshal(fields[2], &value.Scope) != nil || json.Unmarshal(fields[3], &value.Kind) != nil || json.Unmarshal(fields[4], &value.RecordSize) != nil || json.Unmarshal(fields[5], &value.Count) != nil || json.Unmarshal(fields[7], &value.MerkleRoot) != nil {
		return TableManifest{}, fmt.Errorf("invalid table manifest wire")
	}
	var rows [][]json.RawMessage
	if json.Unmarshal(fields[6], &rows) != nil {
		return TableManifest{}, fmt.Errorf("invalid table shard rows")
	}
	value.Shards = make([]TableShard, len(rows))
	for index, row := range rows {
		shard := &value.Shards[index]
		if len(row) != 6 || json.Unmarshal(row[0], &shard.PackOrdinal) != nil || json.Unmarshal(row[1], &shard.RelativePath) != nil || json.Unmarshal(row[2], &shard.FirstOrdinal) != nil || json.Unmarshal(row[3], &shard.LastOrdinal) != nil || json.Unmarshal(row[4], &shard.ByteLength) != nil || json.Unmarshal(row[5], &shard.PackDigest) != nil {
			return TableManifest{}, fmt.Errorf("invalid table shard row")
		}
	}
	canonical, err := value.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return TableManifest{}, fmt.Errorf("table manifest does not reconstruct")
	}
	return value, nil
}

func ParseStructuralOutputMap(panel string, data []byte) (StructuralOutputMap, string, int, string, error) {
	var fields []json.RawMessage
	var version, ignoredRoot string
	var rowSize, count int
	value := StructuralOutputMap{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if json.Unmarshal(data, &fields) != nil || len(fields) != 7 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-structural-output-map/v1" || json.Unmarshal(fields[1], &value.Curriculum) != nil || json.Unmarshal(fields[2], &value.RunIDs) != nil || json.Unmarshal(fields[3], &rowSize) != nil || rowSize != StructuralMapRowSize || json.Unmarshal(fields[4], &count) != nil || json.Unmarshal(fields[5], &ignoredRoot) != nil || !digestText(ignoredRoot) {
		return StructuralOutputMap{}, "", 0, "", fmt.Errorf("invalid structural map wire")
	}
	value.EvidenceRoot, _ = EvidenceRoot(panel)
	value.RunRoots = map[string]string{}
	var rows [][]json.RawMessage
	if json.Unmarshal(fields[6], &rows) != nil || len(rows) > 1 {
		return StructuralOutputMap{}, "", 0, "", fmt.Errorf("invalid structural map shard rows")
	}
	if len(rows) == 0 {
		if count != 0 {
			return StructuralOutputMap{}, "", 0, "", fmt.Errorf("missing structural map shard")
		}
		return value, "", 0, "", nil
	}
	row := rows[0]
	var ordinal, firstKind, lastKind, records int
	var path, firstDigest, lastDigest, digest string
	var length int
	if len(row) != 9 || json.Unmarshal(row[0], &ordinal) != nil || ordinal != 0 || json.Unmarshal(row[1], &path) != nil || json.Unmarshal(row[2], &firstKind) != nil || json.Unmarshal(row[3], &firstDigest) != nil || json.Unmarshal(row[4], &lastKind) != nil || json.Unmarshal(row[5], &lastDigest) != nil || json.Unmarshal(row[6], &records) != nil || json.Unmarshal(row[7], &length) != nil || json.Unmarshal(row[8], &digest) != nil || records != count || length != len(StructuralMapHeader)+count*StructuralMapRowSize || !digestText(firstDigest) || !digestText(lastDigest) || firstKind < 1 || lastKind < firstKind || !safeEvidencePath(path) || !digestText(digest) {
		return StructuralOutputMap{}, "", 0, "", fmt.Errorf("invalid structural map shard")
	}
	return value, path, length, digest, nil
}
