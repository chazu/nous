package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	PlanArchiveDigest  = "b0f41759d9335c74f200f3d288820f690ad5592f0b5878ea3b2971bccf0691ca"
	BuildAuthorityPath = "docs/actionrelations-build-authority.json"
	PanelBinaryPath    = ".nous/bin/actionrelation-nous-v1"
)

type SourceRow struct {
	Path       string
	GitMode    string
	GitBlobOID string
	ByteLength int64
	Digest     string
	Role       string
}

func (r SourceRow) wire() []any {
	return []any{r.Path, r.GitMode, r.GitBlobOID, r.ByteLength, r.Digest, r.Role}
}

type NonInputRow struct {
	Path   string
	Status string
}

func (r NonInputRow) wire() []any { return []any{r.Path, r.Status} }

type BuildAuthority struct {
	PlanCommit                  string
	PlanArchiveDigest           string
	PlanReview                  AuthorityRef
	ImplementationCommit        string
	ImplementationArchiveDigest string
	ImplementationReview        AuthorityRef
	BuildHead                   string
	SourceRoot                  string
	SourceRows                  []SourceRow
	GitVersion                  string
	GoVersion                   string
	GoExecutablePath            string
	GoExecutableDigest          string
	MiseTomlDigest              string
	BuildArgv                   []string
	BuildEnvironment            []EnvironmentRow
	GOOS                        string
	GOARCH                      string
	CGOEnabled                  string
	BinaryPath                  string
	BinaryDigest                string
	GoVersionMDigest            string
	NonInputRows                []NonInputRow
	Canonical                   []byte
	Digest                      string
}

func ParseBuildAuthority(data []byte) (BuildAuthority, error) {
	if len(data) > 1<<20 {
		return BuildAuthority{}, fmt.Errorf("build authority exceeds cap")
	}
	var fields []json.RawMessage
	if json.Unmarshal(data, &fields) != nil || len(fields) != 24 {
		return BuildAuthority{}, fmt.Errorf("invalid build authority wire")
	}
	var version string
	if json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-build-authority/v1" {
		return BuildAuthority{}, fmt.Errorf("invalid build authority version")
	}
	value := BuildAuthority{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	stringsOut := []*string{
		&value.PlanCommit, &value.PlanArchiveDigest, nil,
		&value.ImplementationCommit, &value.ImplementationArchiveDigest, nil,
		&value.BuildHead, &value.SourceRoot, nil, &value.GitVersion,
		&value.GoVersion, &value.GoExecutablePath, &value.GoExecutableDigest,
		&value.MiseTomlDigest, nil, nil, &value.GOOS, &value.GOARCH,
		&value.CGOEnabled, &value.BinaryPath, &value.BinaryDigest,
		&value.GoVersionMDigest, nil,
	}
	for offset, target := range stringsOut {
		if target != nil && json.Unmarshal(fields[offset+1], target) != nil {
			return BuildAuthority{}, fmt.Errorf("invalid build authority field %d", offset+1)
		}
	}
	var err error
	if value.PlanReview, err = parseAuthorityRef(fields[3]); err != nil {
		return BuildAuthority{}, err
	}
	if value.ImplementationReview, err = parseAuthorityRef(fields[6]); err != nil {
		return BuildAuthority{}, err
	}
	if value.SourceRows, err = parseSourceRows(fields[9]); err != nil {
		return BuildAuthority{}, err
	}
	if json.Unmarshal(fields[15], &value.BuildArgv) != nil {
		return BuildAuthority{}, fmt.Errorf("invalid build argv wire")
	}
	if value.BuildEnvironment, err = parseEnvironmentRows(fields[16]); err != nil {
		return BuildAuthority{}, err
	}
	if value.NonInputRows, err = parseNonInputRows(fields[23]); err != nil {
		return BuildAuthority{}, err
	}
	if err := VerifyBuildAuthority(value); err != nil {
		return BuildAuthority{}, err
	}
	return value, nil
}

func BuildSourceRoot(implementationCommit string, rows []SourceRow) (string, error) {
	wires, err := sourceRowWires(rows)
	if err != nil || !commitText(implementationCommit) {
		return "", fmt.Errorf("invalid source-tree authority")
	}
	canonical, _ := json.Marshal([]any{"actionrelation-source-tree/v1", implementationCommit, wires})
	return shaHex(canonical), nil
}

func BuildBuildAuthority(value BuildAuthority) (BuildAuthority, error) {
	value.Canonical = nil
	value.Digest = ""
	canonical, err := buildAuthorityCanonical(value)
	if err != nil {
		return BuildAuthority{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	if err := VerifyBuildAuthority(value); err != nil {
		return BuildAuthority{}, err
	}
	return value, nil
}

func VerifyBuildAuthority(value BuildAuthority) error {
	canonical, err := buildAuthorityCanonical(value)
	if err != nil || len(value.Canonical) > 1<<20 || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid build authority")
	}
	return nil
}

func buildAuthorityCanonical(value BuildAuthority) ([]byte, error) {
	if value.PlanCommit != PlanCommit || value.PlanArchiveDigest != PlanArchiveDigest || !commitText(value.ImplementationCommit) || !digestText(value.ImplementationArchiveDigest) || !commitText(value.BuildHead) || !digestText(value.SourceRoot) {
		return nil, fmt.Errorf("invalid build commit authority")
	}
	if value.PlanReview.Verify() != nil || value.PlanReview.Path != ReviewManifestPath("plan") || value.ImplementationReview.Verify() != nil || value.ImplementationReview.Path != ReviewManifestPath("implementation") {
		return nil, fmt.Errorf("invalid build review authority")
	}
	sourceRows, err := sourceRowWires(value.SourceRows)
	if err != nil {
		return nil, err
	}
	sourceRoot, err := BuildSourceRoot(value.ImplementationCommit, value.SourceRows)
	if err != nil || sourceRoot != value.SourceRoot {
		return nil, fmt.Errorf("build source root mismatch")
	}
	if !boundedASCII(value.GitVersion, 256) || !boundedASCII(value.GoVersion, 256) || !canonicalAbsolutePath(value.GoExecutablePath) || !digestText(value.GoExecutableDigest) || !digestText(value.MiseTomlDigest) || !safeToken(value.GOOS) || !safeToken(value.GOARCH) || value.CGOEnabled != "0" && value.CGOEnabled != "1" || value.BinaryPath != PanelBinaryPath || !digestText(value.BinaryDigest) || !digestText(value.GoVersionMDigest) {
		return nil, fmt.Errorf("invalid build tool or binary authority")
	}
	if len(value.BuildArgv) < 2 || len(value.BuildArgv) > 64 {
		return nil, fmt.Errorf("invalid build argv")
	}
	for _, argument := range value.BuildArgv {
		if argument == "" || len(argument) > 4096 || !utf8.ValidString(argument) || strings.HasPrefix(argument, "-overlay") || strings.HasPrefix(argument, "-tags") {
			return nil, fmt.Errorf("invalid build argument")
		}
	}
	environment, err := environmentWires(value.BuildEnvironment)
	if err != nil {
		return nil, err
	}
	if environmentValue(value.BuildEnvironment, "GOFLAGS") != "" || environmentValue(value.BuildEnvironment, "GOWORK") != "off" || !environmentHas(value.BuildEnvironment, "GOFLAGS") || !environmentHas(value.BuildEnvironment, "GOWORK") {
		return nil, fmt.Errorf("noncanonical Go build environment")
	}
	nonInputs, err := nonInputRowWires(value.NonInputRows)
	if err != nil {
		return nil, err
	}
	for _, required := range []string{".git/hooks", "go.work", "go.work.sum"} {
		if !slices.ContainsFunc(value.NonInputRows, func(row NonInputRow) bool { return row.Path == required }) {
			return nil, fmt.Errorf("missing non-input authority for %s", required)
		}
	}
	return json.Marshal([]any{
		"actionrelation-build-authority/v1",
		value.PlanCommit, value.PlanArchiveDigest, value.PlanReview.Wire(),
		value.ImplementationCommit, value.ImplementationArchiveDigest,
		value.ImplementationReview.Wire(), value.BuildHead, value.SourceRoot,
		sourceRows, value.GitVersion, value.GoVersion, value.GoExecutablePath,
		value.GoExecutableDigest, value.MiseTomlDigest, value.BuildArgv,
		environment, value.GOOS, value.GOARCH, value.CGOEnabled,
		value.BinaryPath, value.BinaryDigest, value.GoVersionMDigest, nonInputs,
	})
}

func sourceRowWires(rows []SourceRow) ([]any, error) {
	if len(rows) == 0 || len(rows) > 8192 {
		return nil, fmt.Errorf("invalid source-row cardinality")
	}
	wires := make([]any, len(rows))
	previous := ""
	for index, row := range rows {
		if !canonicalRelativePath(row.Path) || index > 0 && row.Path <= previous || row.GitMode != "100644" && row.GitMode != "100755" || !objectIDText(row.GitBlobOID) || row.ByteLength < 0 || !digestText(row.Digest) || !sourceRoles[row.Role] {
			return nil, fmt.Errorf("invalid source row %d", index)
		}
		wires[index] = row.wire()
		previous = row.Path
	}
	return wires, nil
}

func nonInputRowWires(rows []NonInputRow) ([]any, error) {
	if len(rows) == 0 || len(rows) > 8192 {
		return nil, fmt.Errorf("invalid non-input row cardinality")
	}
	wires := make([]any, len(rows))
	previous := ""
	for index, row := range rows {
		if !canonicalRelativePath(row.Path) || index > 0 && row.Path <= previous || row.Status != "absent" && row.Status != "present-not-read" {
			return nil, fmt.Errorf("invalid non-input row %d", index)
		}
		wires[index] = row.wire()
		previous = row.Path
	}
	return wires, nil
}

var sourceRoles = map[string]bool{
	"compiler-input": true,
	"domain":         true,
	"test":           true,
	"toolchain":      true,
	"plan":           true,
	"umbrella":       true,
}

func canonicalRelativePath(value string) bool {
	if value == "" || len(value) > 4096 || !utf8.ValidString(value) || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || filepath.Clean(value) != value {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.IndexByte(part, 0) >= 0 {
			return false
		}
	}
	return true
}

func canonicalAbsolutePath(value string) bool {
	return value != "" && len(value) <= 4096 && utf8.ValidString(value) && filepath.IsAbs(value) && filepath.Clean(value) == value
}

func objectIDText(value string) bool {
	return len(value) == 40 && lowerHex(value)
}

func lowerHex(value string) bool {
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func boundedASCII(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && isASCII(value)
}

func safeToken(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '_' || character == '-') {
			return false
		}
	}
	return true
}

func environmentHas(rows []EnvironmentRow, key string) bool {
	return slices.ContainsFunc(rows, func(row EnvironmentRow) bool { return row.Key == key })
}

func environmentValue(rows []EnvironmentRow, key string) string {
	for _, row := range rows {
		if row.Key == key {
			return row.Value
		}
	}
	return ""
}

func parseAuthorityRef(data json.RawMessage) (AuthorityRef, error) {
	var wire []json.RawMessage
	var value AuthorityRef
	if json.Unmarshal(data, &wire) != nil || len(wire) != 3 || json.Unmarshal(wire[0], &value.Path) != nil || json.Unmarshal(wire[1], &value.Digest) != nil || json.Unmarshal(wire[2], &value.Mode) != nil || value.Verify() != nil {
		return AuthorityRef{}, fmt.Errorf("invalid authority-reference wire")
	}
	return value, nil
}

func parseSourceRows(data json.RawMessage) ([]SourceRow, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid source-row wire")
	}
	rows := make([]SourceRow, len(wires))
	for index, wire := range wires {
		if len(wire) != 6 || json.Unmarshal(wire[0], &rows[index].Path) != nil || json.Unmarshal(wire[1], &rows[index].GitMode) != nil || json.Unmarshal(wire[2], &rows[index].GitBlobOID) != nil || json.Unmarshal(wire[3], &rows[index].ByteLength) != nil || json.Unmarshal(wire[4], &rows[index].Digest) != nil || json.Unmarshal(wire[5], &rows[index].Role) != nil {
			return nil, fmt.Errorf("invalid source row %d wire", index)
		}
	}
	return rows, nil
}

func parseEnvironmentRows(data json.RawMessage) ([]EnvironmentRow, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid environment-row wire")
	}
	rows := make([]EnvironmentRow, len(wires))
	for index, wire := range wires {
		if len(wire) != 2 || json.Unmarshal(wire[0], &rows[index].Key) != nil || json.Unmarshal(wire[1], &rows[index].Value) != nil {
			return nil, fmt.Errorf("invalid environment row %d wire", index)
		}
	}
	return rows, nil
}

func parseNonInputRows(data json.RawMessage) ([]NonInputRow, error) {
	var wires [][]json.RawMessage
	if json.Unmarshal(data, &wires) != nil {
		return nil, fmt.Errorf("invalid non-input-row wire")
	}
	rows := make([]NonInputRow, len(wires))
	for index, wire := range wires {
		if len(wire) != 2 || json.Unmarshal(wire[0], &rows[index].Path) != nil || json.Unmarshal(wire[1], &rows[index].Status) != nil {
			return nil, fmt.Errorf("invalid non-input row %d wire", index)
		}
	}
	return rows, nil
}
