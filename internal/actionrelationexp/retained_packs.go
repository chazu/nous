package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type retainedObjectValue struct {
	kind      uint16
	canonical []byte
	classes   map[string]bool
}

type retainedPhysicalObject struct {
	curriculum int
	class      string
	digest     string
	kind       uint16
	bytes      int
}

type retainedRunAuthority struct {
	curriculum         int
	operationRoot      string
	phase              uint8
	workTerminal       string
	work               [12]int
	initialWork        [12]int
	terminal           string
	world              string
	policy             string
	terminalSet        string
	historyCount       int
	matches            [8]int
	certificates       [4]int
	sleepCount         int
	truthPairs         map[string]string
	truthTerminals     []string
	trainingMatches    [5]int
	worldOrdinal       int
	acquisitionTotal   int
	remaining          int
	artifact           string
	initialState       string
	initialOccurrences []string
}

type retainedCurriculumAuthority struct {
	runs            map[string]retainedRunAuthority
	fixture         decodedFixture
	fixtureDigest   string
	truthReferences []retainedTruthReference
}

type retainedTruthReference struct {
	world   string
	ordinal int
	digest  string
}

type retainedPanelFixture struct {
	Panel           string
	Authority       string
	CurriculumRoots []string
	ScorerRoot      string
}

func parseRetainedPanelFixture(data []byte) (retainedPanelFixture, error) {
	var row []json.RawMessage
	var tag string
	value := retainedPanelFixture{}
	if len(data) < 1 || len(data) > 4096 || json.Unmarshal(data, &row) != nil || len(row) != 5 ||
		json.Unmarshal(row[0], &tag) != nil || tag != "actionrelation-fixture-root/v2" ||
		json.Unmarshal(row[1], &value.Panel) != nil || json.Unmarshal(row[2], &value.Authority) != nil ||
		json.Unmarshal(row[3], &value.CurriculumRoots) != nil || json.Unmarshal(row[4], &value.ScorerRoot) != nil ||
		!validPanelAuthority(value.Panel, value.Authority) || len(value.CurriculumRoots) != panelRunCounts[value.Panel]/44 ||
		!uniqueDigestList(value.CurriculumRoots) || !digestText(value.ScorerRoot) {
		return retainedPanelFixture{}, fmt.Errorf("invalid retained panel fixture")
	}
	want, _ := json.Marshal([]any{"actionrelation-fixture-root/v2", value.Panel, value.Authority, value.CurriculumRoots, value.ScorerRoot})
	if !bytes.Equal(want, data) {
		return retainedPanelFixture{}, fmt.Errorf("noncanonical retained panel fixture")
	}
	return value, nil
}

// RetainedPackRefs is the complete physical-pack authority reachable from a
// panel payload. Read must return the exact bytes retained at a logical path.
type RetainedPackRefs struct {
	Panel           string
	Authority       string
	Fixture         AuthorityRef
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
// manifest and pack set, including the fixture root that commits every
// curriculum fixture and scorer-truth shard.
func VerifyRetainedPacks(value RetainedPackRefs, read func(string) ([]byte, error)) ([]string, error) {
	curricula := panelRunCounts[value.Panel] / 44
	if !validPanelAuthority(value.Panel, value.Authority) || curricula == 0 || read == nil || value.RunEvidence.Verify() != nil ||
		value.Fixture.Verify() != nil || value.Fixture.Path != ExpectedAuthorityPath(value.Panel, "fixture-root") ||
		len(value.ObjectRoots) != curricula*4 || len(value.IndexRoots) != curricula*4 ||
		len(value.JournalRoots) != curricula*44 || len(value.InputRoots) != curricula*44 || len(value.DetailRoots) != curricula*44 ||
		len(value.Tables) != curricula*14 || len(value.StructuralMaps) != curricula || len(value.StoreBoundaries) != curricula*2 {
		return nil, fmt.Errorf("invalid retained pack authority")
	}
	reachable := map[string]bool{}
	retainedLeaves := map[string]bool{}
	retainedTableLeaves := map[string][]retainedTableLeaf{}
	retainedObjects := map[string]retainedObjectValue{}
	var physicalObjects []retainedPhysicalObject
	retainedByCurriculum := make([]map[string]retainedObjectValue, curricula)
	retainedByScope := make([]map[string]map[string]retainedObjectValue, curricula)
	curriculumAuthorities := make([]retainedCurriculumAuthority, curricula)
	trainingCores := make([]map[string]map[string]bool, curricula)
	viewEvidence := make([]map[string]map[string]bool, curricula)
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
	fixtureBytes, err := readPath(value.Fixture.Path, value.Fixture.Digest)
	if err != nil {
		return nil, err
	}
	panelFixture, err := parseRetainedPanelFixture(fixtureBytes)
	if err != nil || panelFixture.Panel != value.Panel || panelFixture.Authority != value.Authority || len(panelFixture.CurriculumRoots) != curricula {
		return nil, fmt.Errorf("retained panel fixture does not reconstruct")
	}

	type scopeKey struct {
		curriculum int
		class      string
	}
	type indexAuthority struct{ digest, objectSet string }
	indexAuthorities := map[scopeKey]indexAuthority{}
	indexes := make(map[scopeKey]ObjectManifestRef, len(value.IndexRoots))
	objectCounts := make([]int, curricula)
	indexShards := make([]int, curricula)
	largeCounts := make([][4]int, curricula)
	mediumCounts := make([]int, curricula)
	curriculumManifestBytes := make([]int, curricula)
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
		if key.curriculum < 0 || key.curriculum >= curricula {
			return nil, fmt.Errorf("object scope curriculum is outside panel")
		}
		objectCounts[key.curriculum] += objectRoot.TotalRecords
		indexShards[key.curriculum] += len(indexRoot.Shards)
		curriculumManifestBytes[key.curriculum] += len(objectBytes) + len(indexBytes)
		if len(objectBytes) > 4_096 || len(indexBytes) > 16_384 {
			return nil, fmt.Errorf("curriculum object/index root manifest exceeds frozen cap")
		}
		if objectCounts[key.curriculum] > MaximumCurriculumObjects || indexShards[key.curriculum] > MaximumCurriculumIndexes {
			return nil, fmt.Errorf("curriculum object/index capacity exceeded")
		}
		objectPayloads := map[string][]byte{}
		for _, shard := range objectRoot.Shards {
			file, err := readFile(shard.Path, shard.ByteLength, shard.PackDigest)
			if err != nil {
				return nil, err
			}
			bundle.ObjectFiles = append(bundle.ObjectFiles, file)
			for offset := len(ObjectHeader); offset < len(file.Data); {
				if len(file.Data)-offset < 4 {
					return nil, fmt.Errorf("truncated retained object frame")
				}
				length := int(binary.BigEndian.Uint32(file.Data[offset : offset+4]))
				offset += 4
				if length < 1 || length > len(file.Data)-offset {
					return nil, fmt.Errorf("invalid retained object frame")
				}
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
				if !retainedKindAllowedInScope(key.class, kind) {
					return nil, fmt.Errorf("retained kind %d is not allowed in scope %s", kind, key.class)
				}
				physicalObjects = append(physicalObjects, retainedPhysicalObject{curriculum: key.curriculum, class: key.class, digest: digest, kind: kind, bytes: len(objectPayloads[digest])})
				switch kind {
				case 10, 28:
					largeCounts[key.curriculum][0]++
				case 29:
					largeCounts[key.curriculum][1]++
				case 36:
					largeCounts[key.curriculum][2]++
				case 9, 35, 47:
					mediumCounts[key.curriculum]++
				}
				if prior, ok := retainedObjects[digest]; ok && (prior.kind != kind || !bytes.Equal(prior.canonical, objectPayloads[digest])) {
					return nil, fmt.Errorf("cross-kind retained object collision")
				}
				object := retainedObjectValue{kind: kind, canonical: objectPayloads[digest]}
				retainedObjects[digest] = object
				if retainedByCurriculum[key.curriculum] == nil {
					retainedByCurriculum[key.curriculum] = map[string]retainedObjectValue{}
				}
				local := retainedByCurriculum[key.curriculum][digest]
				if local.classes == nil {
					local = retainedObjectValue{kind: kind, canonical: objectPayloads[digest], classes: map[string]bool{}}
				}
				local.classes[key.class] = true
				retainedByCurriculum[key.curriculum][digest] = local
				if retainedByScope[key.curriculum] == nil {
					retainedByScope[key.curriculum] = map[string]map[string]retainedObjectValue{}
				}
				if retainedByScope[key.curriculum][key.class] == nil {
					retainedByScope[key.curriculum][key.class] = map[string]retainedObjectValue{}
				}
				retainedByScope[key.curriculum][key.class][digest] = object
			}
		}
		indexDigest, _ := indexRoot.Digest()
		indexAuthorities[key] = indexAuthority{digest: indexDigest, objectSet: indexRoot.ObjectSetRoot}
		delete(indexes, key)
	}
	for curriculum := range curricula {
		largeCounts[curriculum][3] = largeCounts[curriculum][0] + largeCounts[curriculum][1] + largeCounts[curriculum][2]
		if largeCounts[curriculum][0] > 32 || largeCounts[curriculum][1] > 32 || largeCounts[curriculum][2] > 32 || largeCounts[curriculum][3] > 96 || mediumCounts[curriculum] > 512 {
			return nil, fmt.Errorf("curriculum per-class object capacity exceeded: %d", curriculum)
		}
	}
	if len(indexes) != 0 {
		return nil, fmt.Errorf("orphan retained index root")
	}
	for curriculum, objects := range retainedByCurriculum {
		if err := verifyCurriculumAuthorityObjects(value.Panel, value.Authority, curriculum, objects, &curriculumAuthorities[curriculum]); err != nil {
			return nil, fmt.Errorf("retained curriculum authority %d: %w", curriculum, err)
		}
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
	transcriptCalls := map[string][]retainedCall{}
	transcriptBundles := map[string]TranscriptBundle{}
	transcriptManifestBytes := map[string]int{}
	transcriptClassBytes := map[string][3]int{}
	for runID, refs := range transcripts {
		if refs.journal.Path == "" || refs.input.Path == "" || refs.detail.Path == "" {
			return nil, fmt.Errorf("incomplete retained transcript %s", runID)
		}
		bundle := TranscriptBundle{RunID: runID}
		for classIndex, item := range []struct {
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
			if err != nil || root.Class != item.class || root.RunID != runID || len(manifest) > 4_096 {
				return nil, fmt.Errorf("retained transcript root does not reconstruct: %s", item.ref.Path)
			}
			transcriptManifestBytes[runID] += len(manifest)
			*item.root = root
			for _, shard := range root.Shards {
				file, err := readFile(shard.Path, shard.ByteLength, shard.PackDigest)
				if err != nil {
					return nil, err
				}
				*item.files = append(*item.files, file)
				classBytes := transcriptClassBytes[runID]
				classBytes[classIndex] += len(file.Data)
				transcriptClassBytes[runID] = classBytes
			}
		}
		calls, err := decodeRetainedCalls(bundle)
		if err != nil {
			return nil, fmt.Errorf("retained transcript %s calls: %w", runID, err)
		}
		bundle.CallIDs = make([]string, len(calls))
		bundle.EnvelopeDigests = make([]string, len(calls))
		for index, call := range calls {
			bundle.CallIDs[index] = call.callID
			bundle.EnvelopeDigests[index] = call.envelopeDigest
		}
		if err := VerifyTranscript(bundle); err != nil {
			return nil, fmt.Errorf("retained transcript %s: %w", runID, err)
		}
		transcriptBundles[runID] = bundle
		transcriptCalls[runID] = calls
		journalDigests[runID], _ = bundle.JournalRoot.Digest()
		inputDigests[runID], _ = bundle.InputRoot.Digest()
		detailDigests[runID], _ = bundle.DetailRoot.Digest()
		chargedRoots[runID], _ = ChargedOutputsRoot(bundle)
	}

	tableDigests := map[string][]string{}
	for _, ref := range value.Tables {
		manifestBytes, err := readPath(ref.Path, ref.Digest)
		if err != nil {
			return nil, err
		}
		manifest, err := ParseTableManifest(manifestBytes)
		if err != nil || manifest.Curriculum != ref.Curriculum || manifest.Scope != ref.Scope || manifest.Kind != ref.Kind || len(manifestBytes) > 4_096 {
			return nil, fmt.Errorf("retained table manifest does not reconstruct: %s", ref.Path)
		}
		curriculumManifestBytes[ref.Curriculum] += len(manifestBytes)
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
				record := bytes.Clone(file.Data[start : start+manifest.RecordSize])
				leaf := TableLeafDigest(manifest.Kind, ordinal, record)
				leafDigest := hex.EncodeToString(leaf[:])
				bundle.LeafDigests = append(bundle.LeafDigests, leafDigest)
				retainedLeaves[leafDigest] = true
				retainedTableLeaves[leafDigest] = append(retainedTableLeaves[leafDigest], retainedTableLeaf{kind: manifest.Kind, curriculum: manifest.Curriculum, scope: manifest.Scope, manifest: ref.Digest, ordinal: ordinal, record: record})
				if manifest.Kind == 105 {
					canonical, canonicalErr := observationCanonical(record)
					if canonicalErr != nil {
						return nil, canonicalErr
					}
					if trainingCores[manifest.Curriculum] == nil {
						trainingCores[manifest.Curriculum] = map[string]map[string]bool{}
					}
					if trainingCores[manifest.Curriculum][manifest.Scope] == nil {
						trainingCores[manifest.Curriculum][manifest.Scope] = map[string]bool{}
					}
					trainingCores[manifest.Curriculum][manifest.Scope][shaHex(canonical)] = true
				}
				if manifest.Kind == 106 {
					wire, _ := json.Marshal([]any{"action-view-evidence/v1", digestAt(record, 32), digestAt(record, 0), digestAt(record, 96)})
					if viewEvidence[manifest.Curriculum] == nil {
						viewEvidence[manifest.Curriculum] = map[string]map[string]bool{}
					}
					if viewEvidence[manifest.Curriculum][manifest.Scope] == nil {
						viewEvidence[manifest.Curriculum][manifest.Scope] = map[string]bool{}
					}
					viewEvidence[manifest.Curriculum][manifest.Scope][shaHex(wire)] = true
				}
			}
		}
		if err := VerifyTableBundle(bundle); err != nil {
			return nil, fmt.Errorf("retained table bundle %s: %w", ref.Path, err)
		}
		tableKey := fmt.Sprintf("%d:%s", ref.Curriculum, ref.Scope)
		tableDigests[tableKey] = append(tableDigests[tableKey], ref.Digest)
	}
	var scorerReferences []retainedTruthReference
	for curriculum := range curriculumAuthorities {
		fixture := curriculumAuthorities[curriculum].fixture
		if curriculumAuthorities[curriculum].fixtureDigest != panelFixture.CurriculumRoots[curriculum] {
			return nil, fmt.Errorf("curriculum fixture %d differs from panel fixture root", curriculum)
		}
		for _, scope := range []string{"nous", "no-guard"} {
			for _, digest := range fixture.Training {
				if !trainingCores[curriculum][scope][digest] {
					return nil, fmt.Errorf("curriculum fixture %d/%s training core does not resolve", curriculum, scope)
				}
			}
			for _, digest := range fixture.Views {
				if !viewEvidence[curriculum][scope][digest] {
					return nil, fmt.Errorf("curriculum fixture %d/%s presentation view does not resolve", curriculum, scope)
				}
			}
		}
		scorerReferences = append(scorerReferences, curriculumAuthorities[curriculum].truthReferences...)
	}
	slices.SortFunc(scorerReferences, func(a, b retainedTruthReference) int {
		if a.world != b.world {
			return compareString(a.world, b.world)
		}
		if a.ordinal != b.ordinal {
			return a.ordinal - b.ordinal
		}
		return compareString(a.digest, b.digest)
	})
	scorerRows := make([]any, len(scorerReferences))
	for index, reference := range scorerReferences {
		scorerRows[index] = []any{reference.world, reference.ordinal, reference.digest}
	}
	scorerRoot, err := actionrelationwire.RootDigest("scorer-shards", scorerRows)
	if err != nil || scorerRoot != panelFixture.ScorerRoot {
		return nil, fmt.Errorf("retained scorer truth differs from panel fixture root")
	}
	for index, boundary := range value.StoreBoundaries {
		wantCurriculum, wantScope := index/2, "nous"
		if index%2 == 1 {
			wantScope = "no-guard"
		}
		key := scopeKey{curriculum: boundary.Curriculum, class: "acquisition-" + boundary.Scope + "-preboundary"}
		indexAuthority, ok := indexAuthorities[key]
		object, objectOK := retainedByCurriculum[boundary.Curriculum][boundary.BoundaryDigest]
		if boundary.Curriculum != wantCurriculum || boundary.Scope != wantScope || !ok || boundary.PreboundaryIndexRoot != indexAuthority.digest || !objectOK || object.kind != 35 || len(object.classes) != 1 || !object.classes["authority"] {
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
		if len(manifest) > 8_192 {
			return nil, fmt.Errorf("retained structural map manifest exceeds frozen cap")
		}
		curriculumManifestBytes[curriculum] += len(manifest)
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
	expectedRuns := map[string]retainedRunAuthority{}
	for _, curriculum := range curriculumAuthorities {
		for runID, authority := range curriculum.runs {
			if _, duplicate := expectedRuns[runID]; duplicate {
				return nil, fmt.Errorf("duplicate expected run identity")
			}
			expectedRuns[runID] = authority
		}
	}
	curriculumCalls := make([]int, curricula)
	curriculumTranscriptBytes := make([][3]int, curricula)
	chargedOutputs := make([]map[string]bool, curricula)
	usedUtility := make([]map[string]bool, curricula)
	usedAcquisition := make([]map[string]map[string]bool, curricula)
	for curriculum := range curricula {
		chargedOutputs[curriculum] = map[string]bool{}
		usedUtility[curriculum] = map[string]bool{}
		usedAcquisition[curriculum] = map[string]map[string]bool{"nous": {}, "no-guard": {}}
	}
	for _, record := range runPack.Records {
		if journalDigests[record.RunID] != record.JournalRoot || inputDigests[record.RunID] != record.InputRoot || detailDigests[record.RunID] != record.DetailRoot || chargedRoots[record.RunID] != record.ChargedRoot || structuralRoots[record.RunID] != record.StructuralRoot {
			return nil, fmt.Errorf("run-evidence row does not close retained run %s", record.RunID)
		}
		if !structuralObjects[record.RunID][fmt.Sprintf("46:%s", record.OperationRoot)] {
			return nil, fmt.Errorf("run-evidence operation root is not structurally retained: %s", record.RunID)
		}
		authority, ok := expectedRuns[record.RunID]
		operationObject, operationOK := retainedObjects[authority.operationRoot]
		if !ok || !operationOK || operationObject.kind != 46 || record.OperationRoot != authority.operationRoot {
			return nil, fmt.Errorf("run-evidence identity or operation root is not expected: %s", record.RunID)
		}
		curriculumCalls[authority.curriculum] += len(transcriptCalls[record.RunID])
		curriculumManifestBytes[authority.curriculum] += transcriptManifestBytes[record.RunID]
		classBytes := transcriptClassBytes[record.RunID]
		for index := range classBytes {
			curriculumTranscriptBytes[authority.curriculum][index] += classBytes[index]
		}
		markUsed := usedUtility[authority.curriculum]
		objectClass := "utility"
		if authority.phase == 1 {
			markUsed = usedAcquisition[authority.curriculum][authority.policy]
			objectClass = "acquisition-" + authority.policy + "-preboundary"
		}
		ownObjects := retainedByScope[authority.curriculum][objectClass]
		runObjects, allowedCrossScope, scopeErr := retainedObjectsForRun(authority, retainedByScope[authority.curriculum])
		if scopeErr != nil {
			return nil, fmt.Errorf("retained run %s physical scope: %w", record.RunID, scopeErr)
		}
		for _, call := range transcriptCalls[record.RunID] {
			if _, exists := ownObjects[call.source]; !exists {
				return nil, fmt.Errorf("charged reservation is outside run scope: %s/%s", record.RunID, call.source)
			}
			markUsed[call.source] = true
			for outputIndex, output := range call.outputs {
				object, own := ownObjects[output]
				if !own {
					if retainedAcquisitionTableOutput(authority, call, outputIndex, output, retainedTableLeaves) {
						continue
					}
					if !allowedCrossScope[output] || call.operation != 10 || output != authority.artifact {
						return nil, fmt.Errorf("charged output is outside run scope: %s/%s", record.RunID, output)
					}
					continue
				}
				if object.canonical != nil {
					markUsed[output] = true
					chargedOutputs[authority.curriculum][objectClass+":"+output] = true
				}
			}
			payloadDigests := map[string]bool{}
			markRetainedPayloadDigests(call.payload, payloadDigests)
			for digest := range payloadDigests {
				if _, retained := retainedObjects[digest]; !retained {
					continue
				}
				if _, allowed := runObjects[digest]; !allowed {
					return nil, fmt.Errorf("charged payload object is outside run scope: %s/%s", record.RunID, digest)
				}
				if _, own := ownObjects[digest]; own {
					markUsed[digest] = true
				}
			}
		}
		for key := range structuralObjects[record.RunID] {
			if len(key) > 6 {
				digest := key[strings.IndexByte(key, ':')+1:]
				if _, own := ownObjects[digest]; !own {
					return nil, fmt.Errorf("structural object is outside run scope: %s/%s", record.RunID, digest)
				}
				markUsed[digest] = true
			}
		}
		operation := OperationRoot{Canonical: operationObject.canonical, Digest: authority.operationRoot}
		decoded, decodeErr := decodeOperationRoot(mustObjectRow(operationObject.canonical))
		if decodeErr != nil || decoded.Phase != authority.phase || VerifyOperationRange(operation, transcriptBundles[record.RunID]) != nil {
			return nil, fmt.Errorf("run-evidence operation range does not replay: %s", record.RunID)
		}
		wantWorkTerminal := authority.workTerminal
		if wantWorkTerminal == zeroObjectDigest {
			wantWorkTerminal = ""
		}
		if record.WorkTerminal != wantWorkTerminal {
			return nil, fmt.Errorf("run-evidence work terminal differs from score row: %s", record.RunID)
		}
		delete(expectedRuns, record.RunID)
		if record.WorkTerminal != "" && !retainedLeaves[record.WorkTerminal] {
			return nil, fmt.Errorf("run-evidence work terminal is not retained: %s", record.RunID)
		}
		for _, call := range transcriptCalls[record.RunID] {
			for _, output := range call.outputs {
				if !retainedLeaves[output] {
					return nil, fmt.Errorf("charged output lacks retained typed leaf: %s/%s", record.RunID, output)
				}
			}
		}
		if err := verifyRetainedRunReplay(record, authority, transcriptCalls[record.RunID], runObjects, retainedTableLeaves, structuralObjects[record.RunID]); err != nil {
			return nil, fmt.Errorf("retained run %s replay: %w", record.RunID, err)
		}
	}
	if len(expectedRuns) != 0 || len(runPack.Records) != len(transcripts) || len(runPack.Records) != len(structuralRoots) {
		return nil, fmt.Errorf("retained run-evidence coverage mismatch")
	}
	for curriculum := range curricula {
		if curriculumCalls[curriculum] > 61_056 || curriculumTranscriptBytes[curriculum][0] > 7_815_438 || curriculumTranscriptBytes[curriculum][1] > 63_357_710 || curriculumTranscriptBytes[curriculum][2] > 11_723_022 || curriculumManifestBytes[curriculum] > 1_048_576 {
			return nil, fmt.Errorf("curriculum %d exceeds frozen transcript or manifest capacity", curriculum)
		}
	}
	chargedCount, supportingSmallCount, supportingMediumCount, largeCount := make([]int, curricula), make([]int, curricula), make([]int, curricula), make([]int, curricula)
	chargedBytes, supportingSmallBytes, supportingMediumBytes, largeBytes := make([]int, curricula), make([]int, curricula), make([]int, curricula), make([]int, curricula)
	supportingKinds := make([]map[uint16]int, curricula)
	for curriculum := range curricula {
		supportingKinds[curriculum] = map[uint16]int{}
	}
	for _, object := range physicalObjects {
		if object.class == "utility" && !usedUtility[object.curriculum][object.digest] {
			return nil, fmt.Errorf("utility object is neither charged nor structurally attributed: %d/%s", object.curriculum, object.digest)
		}
		if strings.HasPrefix(object.class, "acquisition-") {
			scope := strings.TrimSuffix(strings.TrimPrefix(object.class, "acquisition-"), "-preboundary")
			if !usedAcquisition[object.curriculum][scope][object.digest] {
				return nil, fmt.Errorf("preboundary object is unreachable from its acquisition: %d/%s/%s", object.curriculum, scope, object.digest)
			}
		}
		switch {
		case object.class != "authority" && chargedOutputs[object.curriculum][object.class+":"+object.digest]:
			chargedCount[object.curriculum]++
			chargedBytes[object.curriculum] += object.bytes
			if object.bytes > 1_024 {
				return nil, fmt.Errorf("charged result exceeds small-object cap")
			}
		case slices.Contains([]uint16{10, 28, 29, 36}, object.kind):
			largeCount[object.curriculum]++
			largeBytes[object.curriculum] += object.bytes
			if object.bytes > 65_536 {
				return nil, fmt.Errorf("large supporting object exceeds cap")
			}
		case slices.Contains([]uint16{9, 35, 47}, object.kind):
			supportingMediumCount[object.curriculum]++
			supportingMediumBytes[object.curriculum] += object.bytes
			supportingKinds[object.curriculum][object.kind]++
			if object.bytes > 4_096 {
				return nil, fmt.Errorf("medium supporting object exceeds cap")
			}
		default:
			supportingSmallCount[object.curriculum]++
			supportingSmallBytes[object.curriculum] += object.bytes
			supportingKinds[object.curriculum][object.kind]++
			if object.bytes > 1_024 {
				return nil, fmt.Errorf("small supporting object exceeds cap")
			}
		}
	}
	for curriculum := range curricula {
		if chargedCount[curriculum] > 61_056 || chargedBytes[curriculum] > 61_056*1_024 || supportingSmallCount[curriculum] > 3_584 || supportingSmallBytes[curriculum] > 3_584*1_024 || supportingMediumCount[curriculum] > 512 || supportingMediumBytes[curriculum] > 512*4_096 || largeCount[curriculum] > 96 || largeBytes[curriculum] > 96*65_536 {
			return nil, fmt.Errorf("curriculum %d exceeds frozen object-class capacity", curriculum)
		}
		for kind, count := range supportingKinds[curriculum] {
			if count > 4_096 {
				return nil, fmt.Errorf("curriculum %d supporting kind %d exceeds cap", curriculum, kind)
			}
		}
	}
	paths := make([]string, 0, len(reachable))
	for path := range reachable {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths, nil
}

func retainedAcquisitionTableOutput(authority retainedRunAuthority, call retainedCall, outputIndex int, digest string, tables map[string][]retainedTableLeaf) bool {
	if authority.phase != 1 {
		return false
	}
	var kind uint16
	switch {
	case call.operation == 1 && outputIndex == 1:
		kind = 103
	case call.operation == 2 && outputIndex == 0:
		kind = 103
	case call.operation == 3 && outputIndex == 1:
		kind = 104
	case (call.operation == 4 || call.operation == 5 || call.operation == 6) && outputIndex == 0:
		kind = 107
	case call.operation == 7 && outputIndex == 0:
		kind = 101
	case call.operation == 20 && outputIndex == 0:
		kind = 108
	case call.operation == 22 && outputIndex == 0:
		kind = 102
	default:
		return false
	}
	matches := 0
	for _, leaf := range tables[digest] {
		leafDigest := TableLeafDigest(leaf.kind, leaf.ordinal, leaf.record)
		if hex.EncodeToString(leafDigest[:]) == digest && leaf.curriculum == authority.curriculum && leaf.scope == authority.policy && leaf.kind == kind {
			matches++
		}
	}
	return matches == 1
}

// retainedObjectsForRun preserves the physical object capability boundary during
// semantic replay. Acquisition runs can resolve only their own preboundary.
// Utility runs can additionally resolve public fixture preimages and the exact
// acquisition boundary/artifact closure that their charged artifact-load names.
func retainedObjectsForRun(authority retainedRunAuthority, scopes map[string]map[string]retainedObjectValue) (map[string]retainedObjectValue, map[string]bool, error) {
	class := "utility"
	if authority.phase == 1 {
		class = "acquisition-" + authority.policy + "-preboundary"
	}
	result := map[string]retainedObjectValue{}
	for digest, object := range scopes[class] {
		result[digest] = object
	}
	cross := map[string]bool{}
	if authority.phase == 1 {
		return result, cross, nil
	}
	artifactScope := ""
	switch authority.policy {
	case "nous-guarded-sleep", "learned-no-use":
		artifactScope = "nous"
	case "no-guard-sleep":
		artifactScope = "no-guard"
	}
	for digest, object := range scopes["authority"] {
		switch object.kind {
		case 1, 2, 3, 4:
			result[digest] = object
		case 35:
			if artifactScope == "" {
				continue
			}
			boundary, err := decodeStoreBoundary(mustObjectRow(object.canonical))
			if err != nil {
				return nil, nil, err
			}
			if boundary.Curriculum == authority.curriculum && boundary.Scope == artifactScope {
				result[digest], cross[digest] = object, true
			}
		}
	}
	if artifactScope == "" {
		return result, cross, nil
	}
	preboundary := scopes["acquisition-"+artifactScope+"-preboundary"]
	if artifact, ok := preboundary[authority.artifact]; !ok || artifact.kind != 10 {
		return nil, nil, fmt.Errorf("utility artifact is absent from its acquisition scope")
	}
	queue := []string{authority.artifact}
	seen := map[string]bool{}
	for len(queue) != 0 {
		digest := queue[0]
		queue = queue[1:]
		if seen[digest] {
			continue
		}
		object, ok := preboundary[digest]
		if !ok {
			return nil, nil, fmt.Errorf("artifact closure leaves its acquisition scope: %s", digest)
		}
		seen[digest], cross[digest], result[digest] = true, true, object
		references := map[string]bool{}
		var row []json.RawMessage
		if json.Unmarshal(object.canonical, &row) == nil {
			markRetainedPayloadDigests(row, references)
		}
		for reference := range references {
			if _, inScope := preboundary[reference]; inScope && !seen[reference] {
				queue = append(queue, reference)
			}
		}
	}
	return result, cross, nil
}

func verifyCurriculumAuthorityObjects(panel, authority string, curriculum int, objects map[string]retainedObjectValue, result *retainedCurriculumAuthority) error {
	if len(objects) == 0 || result == nil {
		return fmt.Errorf("empty curriculum object set")
	}
	fixtures := map[string]decodedFixture{}
	ledgers := map[string]decodedAttemptLedger{}
	truth := map[string][]struct {
		digest string
		value  decodedTruthShard
	}{}
	worldRows := map[string]decodedWorldPolicyRow{}
	curriculumRows := map[string]decodedCurriculumPolicyRow{}
	operationRoots := map[string]decodedOperationRoot{}
	workTerminals := map[string]decodedWorkTerminal{}
	reservations := map[string]struct{}{}
	worldObjects := map[string]bool{}
	usedAuthority := map[string]bool{}
	boundaryCount := 0
	type retainedWorldTruth struct {
		terminals   []string
		pairs       map[string]string
		state       string
		occurrences []string
	}
	truthWorlds := map[string]retainedWorldTruth{}
	for digest, object := range objects {
		var row []json.RawMessage
		if json.Unmarshal(object.canonical, &row) != nil {
			return fmt.Errorf("object %s no longer decodes", digest)
		}
		switch object.kind {
		case 4:
			if object.classes["authority"] {
				worldObjects[digest] = true
			}
		case 27:
			if _, err := decodeReservation(object.canonical); err != nil {
				return err
			}
			reservations[digest] = struct{}{}
		case 29:
			value, err := decodeTruthShard(row)
			if err != nil || !object.classes["authority"] {
				return fmt.Errorf("truth shard outside authority scope")
			}
			truth[value.World] = append(truth[value.World], struct {
				digest string
				value  decodedTruthShard
			}{digest, value})
			result.truthReferences = append(result.truthReferences, retainedTruthReference{world: value.World, ordinal: value.Ordinal, digest: digest})
			usedAuthority[digest] = true
		case 32:
			value, err := decodeWorldPolicyRow(row)
			if err != nil || !object.classes["authority"] || value.Panel != panel || value.Curriculum != curriculum {
				return fmt.Errorf("world-policy row outside curriculum authority")
			}
			worldRows[digest] = value
			usedAuthority[digest] = true
		case 33:
			value, err := decodeCurriculumPolicyRow(row)
			if err != nil || !object.classes["authority"] || value.Panel != panel || value.Curriculum != curriculum {
				return fmt.Errorf("curriculum-policy row outside curriculum authority")
			}
			curriculumRows[digest] = value
			usedAuthority[digest] = true
		case 35:
			if !object.classes["authority"] {
				return fmt.Errorf("store boundary outside authority scope")
			}
			boundaryCount++
			usedAuthority[digest] = true
		case 36:
			value, err := decodeAttemptLedger(row)
			if err != nil || !object.classes["authority"] || value.Panel != panel || value.Curriculum != curriculum {
				return fmt.Errorf("attempt ledger outside curriculum authority")
			}
			ledgers[digest] = value
			usedAuthority[digest] = true
		case 46:
			value, err := decodeOperationRoot(row)
			if err != nil {
				return err
			}
			operationRoots[digest] = value
		case 47:
			value, err := decodeFixture(row)
			if err != nil || !object.classes["authority"] || value.Panel != panel || value.Curriculum != curriculum {
				return fmt.Errorf("fixture outside curriculum authority")
			}
			fixtures[digest] = value
			usedAuthority[digest] = true
		case 49:
			value, err := decodeWorkTerminal(row)
			if err != nil {
				return err
			}
			workTerminals[digest] = value
		}
	}
	if len(fixtures) != 1 || len(worldRows) != 42 || len(curriculumRows) != 7 || len(worldObjects) != 6 || boundaryCount != 2 {
		return fmt.Errorf("authority cardinality fixtures=%d worlds=%d curricula=%d world-cores=%d boundaries=%d", len(fixtures), len(worldRows), len(curriculumRows), len(worldObjects), boundaryCount)
	}
	var fixture decodedFixture
	for digest, value := range fixtures {
		fixture = value
		result.fixtureDigest = digest
	}
	result.fixture = fixture
	if fixture.Family != curriculum%8 {
		return fmt.Errorf("fixture family changed")
	}
	if len(ledgers) != fixture.Accepted+1 || len(fixture.AttemptLedgers) != len(ledgers) {
		return fmt.Errorf("fixture ledger count does not close")
	}
	for attempt, digest := range fixture.AttemptLedgers {
		ledger, ok := ledgers[digest]
		if !ok || ledger.Attempt != attempt || ledger.Authority == "" || attempt < fixture.Accepted && ledger.Terminal != "rejected" || attempt == fixture.Accepted && ledger.Terminal != "accepted" {
			return fmt.Errorf("fixture ledger %d does not close", attempt)
		}
	}
	for ordinal, digest := range fixture.Worlds {
		if !worldObjects[digest] {
			return fmt.Errorf("fixture world %d lacks retained semantic core", ordinal)
		}
		shards := truth[digest]
		if len(shards) == 0 {
			return fmt.Errorf("fixture world %d lacks scorer truth", ordinal)
		}
		slices.SortFunc(shards, func(a, b struct {
			digest string
			value  decodedTruthShard
		}) int {
			return a.value.Ordinal - b.value.Ordinal
		})
		for index, shard := range shards {
			if shard.value.Ordinal != index || shard.value.Count != len(shards) || !slices.Equal(shard.value.Terminals, shards[0].value.Terminals) {
				return fmt.Errorf("truth shards for world %d do not partition", ordinal)
			}
			if index > 0 && compareTruthPair(shards[index-1].value.Pairs[len(shards[index-1].value.Pairs)-1], shard.value.Pairs[0]) >= 0 {
				return fmt.Errorf("truth rows for world %d overlap", ordinal)
			}
		}
		worldTruth := retainedWorldTruth{terminals: slices.Clone(shards[0].value.Terminals), pairs: map[string]string{}}
		for _, shard := range shards {
			for _, pair := range shard.value.Pairs {
				key := pair.State + pair.A + pair.B
				if _, duplicate := worldTruth.pairs[key]; duplicate {
					return fmt.Errorf("truth rows for world %d duplicate a pair", ordinal)
				}
				worldTruth.pairs[key] = pair.Label
			}
		}
		usedAuthority[digest] = true
		var worldRow []json.RawMessage
		if json.Unmarshal(objects[digest].canonical, &worldRow) != nil || len(worldRow) != 3 {
			return fmt.Errorf("fixture world %d does not decode", ordinal)
		}
		stateCanonical, _ := json.Marshal(worldRow[1])
		stateDigest := shaHex(stateCanonical)
		if objects[stateDigest].kind != 1 || !objects[stateDigest].classes["authority"] || !bytes.Equal(objects[stateDigest].canonical, stateCanonical) {
			return fmt.Errorf("fixture world %d state preimage is absent", ordinal)
		}
		usedAuthority[stateDigest] = true
		worldTruth.state = stateDigest
		var actionRows []json.RawMessage
		if json.Unmarshal(worldRow[2], &actionRows) != nil {
			return fmt.Errorf("fixture world %d actions do not decode", ordinal)
		}
		actions := make([]actionrelations.SemanticAction, len(actionRows))
		for index, raw := range actionRows {
			actionCanonical, _ := json.Marshal(raw)
			actionDigest := shaHex(actionCanonical)
			if objects[actionDigest].kind != 2 || !objects[actionDigest].classes["authority"] || !bytes.Equal(objects[actionDigest].canonical, actionCanonical) {
				return fmt.Errorf("fixture world %d action %d preimage is absent", ordinal, index)
			}
			usedAuthority[actionDigest] = true
			var parseErr error
			actions[index], parseErr = actionrelations.ParseSemanticAction(actionCanonical)
			if parseErr != nil {
				return parseErr
			}
		}
		occurrences, occurrenceErr := actionrelations.AssignOccurrences(actions)
		if occurrenceErr != nil {
			return occurrenceErr
		}
		worldTruth.occurrences = make([]string, len(occurrences))
		for index, occurrence := range occurrences {
			canonical, _ := occurrence.CanonicalJSON()
			occurrenceDigest := shaHex(canonical)
			if objects[occurrenceDigest].kind != 3 || !objects[occurrenceDigest].classes["authority"] || !bytes.Equal(objects[occurrenceDigest].canonical, canonical) {
				return fmt.Errorf("fixture world %d occurrence %d preimage is absent", ordinal, index)
			}
			usedAuthority[occurrenceDigest] = true
			worldTruth.occurrences[index] = occurrenceDigest
		}
		slices.Sort(worldTruth.occurrences)
		truthWorlds[digest] = worldTruth
		delete(truth, digest)
	}
	if len(truth) != 0 {
		return fmt.Errorf("unreachable scorer truth world")
	}
	type worldKey struct {
		policy  string
		ordinal int
	}
	worldGrid := map[worldKey]string{}
	trainingByScope := map[string][5]int{}
	trainingSeen := map[string]bool{}
	for digest, row := range worldRows {
		if row.Family != fixture.Family || row.World != fixture.Worlds[row.WorldOrdinal] {
			return fmt.Errorf("world-policy row changed fixture identity")
		}
		key := worldKey{row.Policy, row.WorldOrdinal}
		if worldGrid[key] != "" {
			return fmt.Errorf("duplicate world-policy grid cell")
		}
		worldGrid[key] = digest
		root, ok := operationRoots[row.OperationRoot]
		if !ok || root.Variant != "range" || root.Phase != 2 {
			return fmt.Errorf("world-policy operation root does not resolve")
		}
		usedAuthority[row.OperationRoot] = true
		if row.Terminal == "budget-exhausted" {
			terminal, ok := workTerminals[row.WorkTerminal]
			if !ok || terminal.RunID != root.RunID {
				return fmt.Errorf("world-policy work terminal does not resolve")
			}
			if _, ok := reservations[terminal.Rejected]; !ok {
				return fmt.Errorf("work terminal rejected reservation is absent")
			}
		}
		worldTruth, truthOK := truthWorlds[row.World]
		behaviorEqual := false
		if row.Terminal == "completed" && truthOK {
			terminalWire, _ := json.Marshal([]any{"sleep-terminal-set/v1", worldTruth.terminals})
			terminalObject := objects[row.TerminalSet]
			behaviorEqual = terminalObject.kind == 24 && bytes.Equal(terminalObject.canonical, terminalWire) && shaHex(terminalWire) == row.TerminalSet
		} else if row.TerminalSet != zeroObjectDigest {
			return fmt.Errorf("budget-exhausted world carries a terminal set")
		}
		if row.BehaviorEqual != behaviorEqual {
			return fmt.Errorf("world-policy behavior equality differs from sealed truth")
		}
		scope := ""
		if row.Policy == "nous-guarded-sleep" || row.Policy == "learned-no-use" {
			scope = "nous"
		} else if row.Policy == "no-guard-sleep" {
			scope = "no-guard"
		}
		var training [5]int
		copy(training[:], row.Matches[:5])
		if scope == "" {
			if training != [5]int{} {
				return fmt.Errorf("baseline world-policy row carries acquisition match counts")
			}
		} else if trainingSeen[scope] && trainingByScope[scope] != training {
			return fmt.Errorf("shared acquisition match counts differ across world rows")
		} else {
			trainingSeen[scope], trainingByScope[scope] = true, training
		}
	}
	for _, policy := range []string{"complete", "lexical-order", "static-rw-sleep", "dynamic-diamond-sleep", "nous-guarded-sleep", "no-guard-sleep", "learned-no-use"} {
		for ordinal := 0; ordinal < 6; ordinal++ {
			if worldGrid[worldKey{policy, ordinal}] == "" {
				return fmt.Errorf("missing world-policy grid cell %s/%d", policy, ordinal)
			}
		}
	}
	curriculumPolicies := map[string]bool{}
	expectedRuns := map[string]retainedRunAuthority{}
	acquisitionRoots := map[string]string{}
	addExpected := func(runID string, value retainedRunAuthority) error {
		if prior, ok := expectedRuns[runID]; ok && !reflect.DeepEqual(prior, value) {
			return fmt.Errorf("run authority changed across policy rows")
		}
		expectedRuns[runID] = value
		return nil
	}
	for _, row := range curriculumRows {
		if row.Family != fixture.Family || curriculumPolicies[row.Policy] {
			return fmt.Errorf("duplicate or cross-family curriculum policy")
		}
		curriculumPolicies[row.Policy] = true
		for ordinal, digest := range row.WorldRows {
			if digest != worldGrid[worldKey{row.Policy, ordinal}] {
				return fmt.Errorf("curriculum policy world row %d does not close", ordinal)
			}
		}
		root, ok := operationRoots[row.OperationRoot]
		if !ok || root.Variant != "concat" {
			return fmt.Errorf("curriculum operation concat does not resolve")
		}
		usedAuthority[row.OperationRoot] = true
		worldChildren := make([]string, 6)
		initialWork := row.AcquisitionWork
		searchWork := [12]int{}
		behaviorEqual := true
		aggregateTerminal := "completed"
		for ordinal := 0; ordinal < 6; ordinal++ {
			worldRow := worldRows[worldGrid[worldKey{row.Policy, ordinal}]]
			worldChildren[ordinal] = worldRow.OperationRoot
			runID, err := actionrelationledger.UtilityRunID(panel, authority, curriculum, row.Policy, ordinal, fixture.Worlds[ordinal])
			operation := operationRoots[worldRow.OperationRoot]
			if err != nil || operation.Variant != "range" || operation.Phase != 2 || operation.RunID != runID {
				return fmt.Errorf("utility run identity does not reconstruct: %s/%d", row.Policy, ordinal)
			}
			authority := retainedRunAuthority{
				curriculum: curriculum, operationRoot: worldRow.OperationRoot, phase: 2,
				workTerminal: worldRow.WorkTerminal, work: worldRow.Work, initialWork: initialWork,
				terminal: worldRow.Terminal, world: worldRow.World, policy: worldRow.Policy,
				terminalSet: worldRow.TerminalSet, historyCount: worldRow.HistoryCount,
				matches: worldRow.Matches, certificates: worldRow.Certificates, sleepCount: worldRow.SleepCount,
				truthPairs: truthWorlds[worldRow.World].pairs, truthTerminals: slices.Clone(truthWorlds[worldRow.World].terminals),
				worldOrdinal: ordinal, acquisitionTotal: sumInts(row.AcquisitionWork[:]), remaining: worldRow.Remaining, artifact: row.Artifact,
				initialState: truthWorlds[worldRow.World].state, initialOccurrences: slices.Clone(truthWorlds[worldRow.World].occurrences),
			}
			if err := addExpected(runID, authority); err != nil {
				return fmt.Errorf("utility run authority %s/%d: %w", row.Policy, ordinal, err)
			}
			for index := range searchWork {
				searchWork[index] += worldRow.Work[index]
			}
			behaviorEqual = behaviorEqual && worldRow.BehaviorEqual
			if worldRow.Terminal == "budget-exhausted" {
				aggregateTerminal = "budget-exhausted"
			}
			for index := range initialWork {
				initialWork[index] += worldRow.Work[index]
			}
		}
		wantChildren := slices.Clone(worldChildren)
		acquisitionScope := ""
		switch row.Policy {
		case "nous-guarded-sleep", "learned-no-use":
			acquisitionScope = "nous"
		case "no-guard-sleep":
			acquisitionScope = "no-guard"
		}
		if acquisitionScope != "" {
			if len(root.Children) != 7 {
				return fmt.Errorf("curriculum operation concat lacks acquisition child")
			}
			acquisitionRoot := root.Children[0]
			operation, ok := operationRoots[acquisitionRoot]
			runID, err := actionrelationledger.AcquisitionRunID(panel, authority, curriculum, acquisitionScope)
			if !ok || err != nil || operation.Variant != "range" || operation.Phase != 1 || operation.RunID != runID || acquisitionRoots[acquisitionScope] != "" && acquisitionRoots[acquisitionScope] != acquisitionRoot {
				return fmt.Errorf("acquisition run identity does not reconstruct: %s", acquisitionScope)
			}
			acquisitionRoots[acquisitionScope] = acquisitionRoot
			if err := addExpected(runID, retainedRunAuthority{
				curriculum: curriculum, operationRoot: acquisitionRoot, phase: 1,
				workTerminal: row.AcquisitionTerminal, work: row.AcquisitionWork,
				terminal: row.Acquisition, policy: acquisitionScope, trainingMatches: trainingByScope[acquisitionScope], artifact: row.Artifact,
			}); err != nil {
				return fmt.Errorf("acquisition run authority %s: %w", acquisitionScope, err)
			}
			wantChildren = append([]string{acquisitionRoot}, wantChildren...)
		} else if row.Acquisition != "not-applicable" || len(root.Children) != 6 {
			return fmt.Errorf("baseline curriculum row carries acquisition authority")
		}
		if !slices.Equal(root.Children, wantChildren) {
			return fmt.Errorf("curriculum operation concat children do not reconstruct")
		}
		for _, child := range root.Children {
			if _, ok := operationRoots[child]; !ok {
				return fmt.Errorf("curriculum operation child is absent")
			}
		}
		lifecycleWork := row.AcquisitionWork
		for index := range lifecycleWork {
			lifecycleWork[index] += searchWork[index]
		}
		remaining := 2_000_000 - sumInts(lifecycleWork[:])
		if aggregateTerminal == "budget-exhausted" {
			remaining = 0
		}
		if searchWork != row.SearchWork || lifecycleWork != row.LifecycleWork || row.SearchTotal != sumInts(searchWork[:]) || row.LifecycleTotal != sumInts(lifecycleWork[:]) || row.BehaviorEqual != behaviorEqual || row.Terminal != aggregateTerminal || row.Remaining != remaining {
			return fmt.Errorf("curriculum-policy aggregates differ from exact world rows")
		}
	}
	if len(expectedRuns) != 44 || len(acquisitionRoots) != 2 {
		return fmt.Errorf("expected run authority count=%d acquisitions=%d", len(expectedRuns), len(acquisitionRoots))
	}
	for digest, object := range objects {
		if object.classes["authority"] && !usedAuthority[digest] {
			return fmt.Errorf("unreachable object in curriculum authority scope: kind %d", object.kind)
		}
	}
	result.runs = expectedRuns
	return nil
}

func mustObjectRow(data []byte) []json.RawMessage {
	var row []json.RawMessage
	_ = json.Unmarshal(data, &row)
	return row
}

func retainedKindAllowedInScope(class string, kind uint16) bool {
	switch class {
	case "authority":
		return slices.Contains([]uint16{1, 2, 3, 4, 29, 32, 33, 35, 36, 46, 47}, kind)
	case "utility":
		return !slices.Contains([]uint16{29, 32, 33, 35, 36, 47}, kind)
	case "acquisition-nous-preboundary", "acquisition-no-guard-preboundary":
		return !slices.Contains([]uint16{18, 19, 20, 21, 22, 23, 24, 25, 26, 29, 32, 33, 35, 36, 43, 44, 47, 48, 49}, kind)
	default:
		return false
	}
}

func markRetainedPayloadDigests(payload []json.RawMessage, used map[string]bool) {
	var walk func(any)
	walk = func(value any) {
		switch current := value.(type) {
		case string:
			if digestText(current) {
				used[current] = true
			}
		case []any:
			for _, item := range current {
				walk(item)
			}
		case map[string]any:
			for _, item := range current {
				walk(item)
			}
		}
	}
	for _, raw := range payload {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			walk(value)
		}
	}
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
