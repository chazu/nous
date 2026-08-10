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
