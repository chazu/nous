package actionrelationcompetence

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationwire"
)

const (
	caseHeader         = "ARCC1\n"
	resultHeader       = "ARCR1\n"
	maximumRowBytes    = 512
	maximumCases       = 65536
	maximumPackBytes   = 16 * 1024 * 1024
	competenceRootPath = ".nous/actionrelations-v1-competence-evidence"
)

type CaseRow struct {
	Suite    string
	CaseID   string
	Input    string
	Expected string
}

func (r CaseRow) wire() []any { return []any{r.Suite, r.CaseID, r.Input, r.Expected} }

type ResultRow struct {
	Suite      string
	CaseID     string
	Production string
	Oracle     string
}

func (r ResultRow) wire() []any { return []any{r.Suite, r.CaseID, r.Production, r.Oracle, "passed"} }

type EvidenceFile struct {
	Path string
	Mode string
	Data []byte
}

type RowShard struct {
	Ordinal    int
	Path       string
	FirstSuite string
	FirstCase  string
	LastSuite  string
	LastCase   string
	Rows       int
	Bytes      int
	Digest     string
}

type RowManifest struct {
	Class     string
	TotalRows int
	RowRoot   string
	Shards    []RowShard
	Canonical []byte
	Digest    string
}

type Evidence struct {
	Cases          []CaseRow
	Results        []ResultRow
	CaseFiles      []EvidenceFile
	ResultFiles    []EvidenceFile
	CaseManifest   RowManifest
	ResultManifest RowManifest
}

func ParseEvidence(caseManifestBytes, resultManifestBytes []byte, files map[string][]byte) (Evidence, error) {
	caseManifest, err := parseRowManifest(caseManifestBytes, "cases")
	if err != nil {
		return Evidence{}, err
	}
	resultManifest, err := parseRowManifest(resultManifestBytes, "results")
	if err != nil {
		return Evidence{}, err
	}
	if caseManifest.TotalRows != resultManifest.TotalRows || len(files) != len(caseManifest.Shards)+len(resultManifest.Shards) {
		return Evidence{}, fmt.Errorf("competence retained file set is not closed")
	}
	var cases []CaseRow
	var results []ResultRow
	caseFiles := make([]EvidenceFile, len(caseManifest.Shards))
	for index, shard := range caseManifest.Shards {
		data, present := files[shard.Path]
		if !present || len(data) != shard.Bytes || shaHex(data) != shard.Digest {
			return Evidence{}, fmt.Errorf("invalid retained case shard %d", index)
		}
		decoded, decodeErr := decodeCasePack(data)
		if decodeErr != nil || len(decoded) != shard.Rows {
			return Evidence{}, fmt.Errorf("decode retained case shard %d: %w", index, decodeErr)
		}
		cases = append(cases, decoded...)
		caseFiles[index] = EvidenceFile{Path: shard.Path, Mode: "100644", Data: bytes.Clone(data)}
	}
	resultFiles := make([]EvidenceFile, len(resultManifest.Shards))
	for index, shard := range resultManifest.Shards {
		data, present := files[shard.Path]
		if !present || len(data) != shard.Bytes || shaHex(data) != shard.Digest {
			return Evidence{}, fmt.Errorf("invalid retained result shard %d", index)
		}
		decoded, decodeErr := decodeResultPack(data)
		if decodeErr != nil || len(decoded) != shard.Rows {
			return Evidence{}, fmt.Errorf("decode retained result shard %d: %w", index, decodeErr)
		}
		results = append(results, decoded...)
		resultFiles[index] = EvidenceFile{Path: shard.Path, Mode: "100644", Data: bytes.Clone(data)}
	}
	value := Evidence{Cases: cases, Results: results, CaseFiles: caseFiles, ResultFiles: resultFiles, CaseManifest: caseManifest, ResultManifest: resultManifest}
	if err := VerifyEvidence(value); err != nil {
		return Evidence{}, err
	}
	return value, nil
}

func BuildEvidence(cases []CaseRow, results []ResultRow) (Evidence, error) {
	if len(cases) < 1 || len(cases) > maximumCases || len(cases) != len(results) {
		return Evidence{}, fmt.Errorf("invalid competence evidence cardinality")
	}
	caseWires := make([]any, len(cases))
	resultWires := make([]any, len(results))
	for index := range cases {
		if err := verifyRows(index, cases, results); err != nil {
			return Evidence{}, err
		}
		caseWires[index], resultWires[index] = cases[index].wire(), results[index].wire()
	}
	caseRoot, _ := actionrelationwire.RootDigest("competence-cases", caseWires)
	resultRoot, _ := actionrelationwire.RootDigest("competence-results", resultWires)
	caseManifest, caseFiles, err := packRows("cases", caseRoot, cases, func(index int) []any { return cases[index].wire() })
	if err != nil {
		return Evidence{}, err
	}
	resultManifest, resultFiles, err := packRows("results", resultRoot, cases, func(index int) []any { return results[index].wire() })
	if err != nil {
		return Evidence{}, err
	}
	value := Evidence{Cases: slices.Clone(cases), Results: slices.Clone(results), CaseFiles: caseFiles, ResultFiles: resultFiles, CaseManifest: caseManifest, ResultManifest: resultManifest}
	if err := VerifyEvidence(value); err != nil {
		return Evidence{}, err
	}
	return value, nil
}

func VerifyEvidence(value Evidence) error {
	rebuilt, err := BuildEvidenceUnchecked(value.Cases, value.Results)
	if err != nil || !equalManifest(rebuilt.CaseManifest, value.CaseManifest) || !equalManifest(rebuilt.ResultManifest, value.ResultManifest) || !equalFiles(rebuilt.CaseFiles, value.CaseFiles) || !equalFiles(rebuilt.ResultFiles, value.ResultFiles) {
		return fmt.Errorf("competence evidence does not reconstruct")
	}
	return nil
}

func BuildEvidenceUnchecked(cases []CaseRow, results []ResultRow) (Evidence, error) {
	if len(cases) < 1 || len(cases) > maximumCases || len(cases) != len(results) {
		return Evidence{}, fmt.Errorf("invalid competence evidence cardinality")
	}
	caseWires := make([]any, len(cases))
	resultWires := make([]any, len(results))
	for index := range cases {
		if err := verifyRows(index, cases, results); err != nil {
			return Evidence{}, err
		}
		caseWires[index], resultWires[index] = cases[index].wire(), results[index].wire()
	}
	caseRoot, _ := actionrelationwire.RootDigest("competence-cases", caseWires)
	resultRoot, _ := actionrelationwire.RootDigest("competence-results", resultWires)
	caseManifest, caseFiles, err := packRows("cases", caseRoot, cases, func(index int) []any { return cases[index].wire() })
	if err != nil {
		return Evidence{}, err
	}
	resultManifest, resultFiles, err := packRows("results", resultRoot, cases, func(index int) []any { return results[index].wire() })
	if err != nil {
		return Evidence{}, err
	}
	return Evidence{Cases: slices.Clone(cases), Results: slices.Clone(results), CaseFiles: caseFiles, ResultFiles: resultFiles, CaseManifest: caseManifest, ResultManifest: resultManifest}, nil
}

func verifyRows(index int, cases []CaseRow, results []ResultRow) error {
	caseRow, result := cases[index], results[index]
	if !validKey(caseRow.Suite) || !validKey(caseRow.CaseID) || !digestText(caseRow.Input) || !digestText(caseRow.Expected) || result.Suite != caseRow.Suite || result.CaseID != caseRow.CaseID || result.Production != caseRow.Expected || result.Oracle != caseRow.Expected {
		return fmt.Errorf("invalid competence row %d", index)
	}
	if index > 0 {
		previous := cases[index-1]
		if caseRow.Suite < previous.Suite || caseRow.Suite == previous.Suite && caseRow.CaseID <= previous.CaseID {
			return fmt.Errorf("competence rows are not unique key ordered")
		}
	}
	return nil
}

func validKey(value string) bool { return value != "" && len(value) <= 64 && utf8.ValidString(value) }

func digestText(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == hex.EncodeToString(decoded)
}

func packRows(class, root string, keys []CaseRow, wire func(int) []any) (RowManifest, []EvidenceFile, error) {
	header, extension := caseHeader, "arcc"
	if class == "results" {
		header, extension = resultHeader, "arcr"
	} else if class != "cases" {
		return RowManifest{}, nil, fmt.Errorf("invalid competence row class")
	}
	encoded := make([][]byte, len(keys))
	for index := range keys {
		encoded[index], _ = json.Marshal(wire(index))
		if len(encoded[index]) > maximumRowBytes {
			return RowManifest{}, nil, fmt.Errorf("competence row %d exceeds cap", index)
		}
	}
	var files []EvidenceFile
	var shards []RowShard
	for first := 0; first < len(encoded); {
		last, size := first, len(header)
		for last < len(encoded) && size+4+len(encoded[last]) <= maximumPackBytes {
			size += 4 + len(encoded[last])
			last++
		}
		if last == first {
			return RowManifest{}, nil, fmt.Errorf("competence frame exceeds pack cap")
		}
		data := make([]byte, size)
		copy(data, header)
		offset := len(header)
		for index := first; index < last; index++ {
			binary.BigEndian.PutUint32(data[offset:offset+4], uint32(len(encoded[index])))
			offset += 4
			copy(data[offset:], encoded[index])
			offset += len(encoded[index])
		}
		ordinal := len(files)
		path := fmt.Sprintf("%s/%s-%04d.%s", competenceRootPath, class, ordinal, extension)
		files = append(files, EvidenceFile{Path: path, Mode: "100644", Data: data})
		shards = append(shards, RowShard{Ordinal: ordinal, Path: path, FirstSuite: keys[first].Suite, FirstCase: keys[first].CaseID, LastSuite: keys[last-1].Suite, LastCase: keys[last-1].CaseID, Rows: last - first, Bytes: len(data), Digest: shaHex(data)})
		first = last
	}
	rows := make([]any, len(shards))
	for index, shard := range shards {
		rows[index] = []any{shard.Ordinal, shard.Path, shard.FirstSuite, shard.FirstCase, shard.LastSuite, shard.LastCase, shard.Rows, shard.Bytes, shard.Digest}
	}
	canonical, _ := json.Marshal([]any{"actionrelation-competence-row-manifest/v1", class, len(keys), root, rows})
	if len(canonical) > 4096 {
		return RowManifest{}, nil, fmt.Errorf("competence manifest exceeds cap")
	}
	return RowManifest{Class: class, TotalRows: len(keys), RowRoot: root, Shards: shards, Canonical: canonical, Digest: shaHex(canonical)}, files, nil
}

func equalManifest(left, right RowManifest) bool {
	return left.Class == right.Class && left.TotalRows == right.TotalRows && left.RowRoot == right.RowRoot && slices.Equal(left.Shards, right.Shards) && bytes.Equal(left.Canonical, right.Canonical) && left.Digest == right.Digest
}

func equalFiles(left, right []EvidenceFile) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Path != right[index].Path || left[index].Mode != right[index].Mode || !bytes.Equal(left[index].Data, right[index].Data) {
			return false
		}
	}
	return true
}

func shaHex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

type Root struct {
	SourceRoot        string
	BinaryDigest      string
	BuildAuthority    actionrelationexp.AuthorityRef
	CommandArgv       []string
	Environment       []actionrelationexp.EnvironmentRow
	Evidence          Evidence
	CaseManifestRef   actionrelationexp.AuthorityRef
	ResultManifestRef actionrelationexp.AuthorityRef
	Canonical         []byte
	Digest            string
}

func ParseRoot(data []byte, evidence Evidence) (Root, error) {
	var fields []json.RawMessage
	if len(data) > 4096 || json.Unmarshal(data, &fields) != nil || len(fields) != 12 {
		return Root{}, fmt.Errorf("invalid competence root wire")
	}
	var version, caseRoot, resultRoot, status string
	var total int
	value := Root{Evidence: evidence, Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-competence-root/v2" || json.Unmarshal(fields[1], &value.SourceRoot) != nil || json.Unmarshal(fields[2], &value.BinaryDigest) != nil || json.Unmarshal(fields[4], &value.CommandArgv) != nil || json.Unmarshal(fields[8], &caseRoot) != nil || json.Unmarshal(fields[9], &resultRoot) != nil || json.Unmarshal(fields[10], &total) != nil || json.Unmarshal(fields[11], &status) != nil || status != "passed" {
		return Root{}, fmt.Errorf("invalid competence root fields")
	}
	var err error
	if value.BuildAuthority, err = parseRootAuthorityRef(fields[3]); err != nil {
		return Root{}, err
	}
	if value.Environment, err = parseRootEnvironment(fields[5]); err != nil {
		return Root{}, err
	}
	if value.CaseManifestRef, err = parseRootAuthorityRef(fields[6]); err != nil {
		return Root{}, err
	}
	if value.ResultManifestRef, err = parseRootAuthorityRef(fields[7]); err != nil {
		return Root{}, err
	}
	if caseRoot != evidence.CaseManifest.RowRoot || resultRoot != evidence.ResultManifest.RowRoot || total != len(evidence.Cases) || VerifyRoot(value) != nil {
		return Root{}, fmt.Errorf("competence root does not close retained evidence")
	}
	return value, nil
}

func BuildRoot(value Root) (Root, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := competenceRootCanonical(value)
	if err != nil {
		return Root{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	if err := VerifyRoot(value); err != nil {
		return Root{}, err
	}
	return value, nil
}

func VerifyRoot(value Root) error {
	canonical, err := competenceRootCanonical(value)
	if err != nil || len(value.Canonical) > 4096 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid competence root")
	}
	return nil
}

func competenceRootCanonical(value Root) ([]byte, error) {
	if !digestText(value.SourceRoot) || !digestText(value.BinaryDigest) || value.BuildAuthority.Verify() != nil || len(value.CommandArgv) == 0 || VerifyEvidence(value.Evidence) != nil || value.CaseManifestRef.Verify() != nil || value.ResultManifestRef.Verify() != nil || value.CaseManifestRef.Digest != value.Evidence.CaseManifest.Digest || value.ResultManifestRef.Digest != value.Evidence.ResultManifest.Digest {
		return nil, fmt.Errorf("invalid competence root authority")
	}
	if value.BuildAuthority.Path != "docs/actionrelations-build-authority.json" || value.CaseManifestRef.Path != competenceRootPath+"/cases-root.json" || value.ResultManifestRef.Path != competenceRootPath+"/results-root.json" {
		return nil, fmt.Errorf("noncanonical competence authority path")
	}
	environment := make([]any, len(value.Environment))
	previous := ""
	for index, row := range value.Environment {
		if row.Key == "" || index > 0 && row.Key <= previous {
			return nil, fmt.Errorf("invalid competence environment")
		}
		for _, character := range row.Key {
			if !(character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_') {
				return nil, fmt.Errorf("invalid competence environment key")
			}
		}
		for _, character := range row.Value {
			if character > 127 {
				return nil, fmt.Errorf("invalid competence environment value")
			}
		}
		environment[index] = []any{row.Key, row.Value}
		previous = row.Key
	}
	for _, argument := range value.CommandArgv {
		if argument == "" || !utf8.ValidString(argument) {
			return nil, fmt.Errorf("invalid competence argv")
		}
	}
	return json.Marshal([]any{"actionrelation-competence-root/v2", value.SourceRoot, value.BinaryDigest, value.BuildAuthority.Wire(), value.CommandArgv, environment, value.CaseManifestRef.Wire(), value.ResultManifestRef.Wire(), value.Evidence.CaseManifest.RowRoot, value.Evidence.ResultManifest.RowRoot, len(value.Evidence.Cases), "passed"})
}

func parseRowManifest(data []byte, class string) (RowManifest, error) {
	var fields []json.RawMessage
	if len(data) > 4096 || json.Unmarshal(data, &fields) != nil || len(fields) != 5 {
		return RowManifest{}, fmt.Errorf("invalid competence manifest wire")
	}
	var version string
	value := RowManifest{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	var shards [][]json.RawMessage
	if json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-competence-row-manifest/v1" || json.Unmarshal(fields[1], &value.Class) != nil || value.Class != class || json.Unmarshal(fields[2], &value.TotalRows) != nil || json.Unmarshal(fields[3], &value.RowRoot) != nil || json.Unmarshal(fields[4], &shards) != nil || !digestText(value.RowRoot) {
		return RowManifest{}, fmt.Errorf("invalid competence manifest fields")
	}
	value.Shards = make([]RowShard, len(shards))
	for index, wire := range shards {
		row := &value.Shards[index]
		if len(wire) != 9 || json.Unmarshal(wire[0], &row.Ordinal) != nil || json.Unmarshal(wire[1], &row.Path) != nil || json.Unmarshal(wire[2], &row.FirstSuite) != nil || json.Unmarshal(wire[3], &row.FirstCase) != nil || json.Unmarshal(wire[4], &row.LastSuite) != nil || json.Unmarshal(wire[5], &row.LastCase) != nil || json.Unmarshal(wire[6], &row.Rows) != nil || json.Unmarshal(wire[7], &row.Bytes) != nil || json.Unmarshal(wire[8], &row.Digest) != nil || row.Ordinal != index || !validKey(row.FirstSuite) || !validKey(row.FirstCase) || !validKey(row.LastSuite) || !validKey(row.LastCase) || row.Rows < 1 || row.Bytes < len(caseHeader) || row.Bytes > maximumPackBytes || !digestText(row.Digest) {
			return RowManifest{}, fmt.Errorf("invalid competence manifest shard %d", index)
		}
	}
	return value, nil
}

func decodeCasePack(data []byte) ([]CaseRow, error) {
	var result []CaseRow
	err := decodeFrames(data, caseHeader, func(frame []byte) error {
		var fields []json.RawMessage
		var row CaseRow
		if json.Unmarshal(frame, &fields) != nil || len(fields) != 4 || json.Unmarshal(fields[0], &row.Suite) != nil || json.Unmarshal(fields[1], &row.CaseID) != nil || json.Unmarshal(fields[2], &row.Input) != nil || json.Unmarshal(fields[3], &row.Expected) != nil {
			return fmt.Errorf("invalid competence case frame")
		}
		canonical, _ := json.Marshal(row.wire())
		if !bytes.Equal(canonical, frame) {
			return fmt.Errorf("noncanonical competence case frame")
		}
		result = append(result, row)
		return nil
	})
	return result, err
}

func decodeResultPack(data []byte) ([]ResultRow, error) {
	var result []ResultRow
	err := decodeFrames(data, resultHeader, func(frame []byte) error {
		var fields []json.RawMessage
		var row ResultRow
		var status string
		if json.Unmarshal(frame, &fields) != nil || len(fields) != 5 || json.Unmarshal(fields[0], &row.Suite) != nil || json.Unmarshal(fields[1], &row.CaseID) != nil || json.Unmarshal(fields[2], &row.Production) != nil || json.Unmarshal(fields[3], &row.Oracle) != nil || json.Unmarshal(fields[4], &status) != nil || status != "passed" {
			return fmt.Errorf("invalid competence result frame")
		}
		canonical, _ := json.Marshal(row.wire())
		if !bytes.Equal(canonical, frame) {
			return fmt.Errorf("noncanonical competence result frame")
		}
		result = append(result, row)
		return nil
	})
	return result, err
}

func decodeFrames(data []byte, header string, decode func([]byte) error) error {
	if !bytes.HasPrefix(data, []byte(header)) {
		return fmt.Errorf("invalid competence pack header")
	}
	for offset := len(header); offset < len(data); {
		if len(data)-offset < 4 {
			return fmt.Errorf("truncated competence frame length")
		}
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if length < 1 || length > maximumRowBytes || len(data)-offset < length {
			return fmt.Errorf("invalid competence frame length")
		}
		if err := decode(data[offset : offset+length]); err != nil {
			return err
		}
		offset += length
	}
	return nil
}

func parseRootAuthorityRef(data json.RawMessage) (actionrelationexp.AuthorityRef, error) {
	var fields []json.RawMessage
	var value actionrelationexp.AuthorityRef
	if json.Unmarshal(data, &fields) != nil || len(fields) != 3 || json.Unmarshal(fields[0], &value.Path) != nil || json.Unmarshal(fields[1], &value.Digest) != nil || json.Unmarshal(fields[2], &value.Mode) != nil || value.Verify() != nil {
		return actionrelationexp.AuthorityRef{}, fmt.Errorf("invalid competence authority reference")
	}
	return value, nil
}

func parseRootEnvironment(data json.RawMessage) ([]actionrelationexp.EnvironmentRow, error) {
	var fields [][]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return nil, fmt.Errorf("invalid competence environment wire")
	}
	rows := make([]actionrelationexp.EnvironmentRow, len(fields))
	for index, field := range fields {
		if len(field) != 2 || json.Unmarshal(field[0], &rows[index].Key) != nil || json.Unmarshal(field[1], &rows[index].Value) != nil {
			return nil, fmt.Errorf("invalid competence environment row")
		}
	}
	return rows, nil
}
