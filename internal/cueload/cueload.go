// Package cueload compiles CUE domain files against the embedded #Unit schema
// and returns structured UnitDef values.
package cueload

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// UnitDef is the Go representation of a single unit parsed from CUE.
type UnitDef struct {
	Name  string
	Worth int
	IsA   []string
	Slots map[string]any
}

// schemaSource returns the embedded schema CUE text.
// embeddedSchema is defined in embed.go via //go:embed.
func getSchema() string { return embeddedSchema }

var pkgLine = regexp.MustCompile(`(?m)^\s*package\s+\S+\s*\n`)
var importBlock = regexp.MustCompile(`(?ms)^\s*import\s*\(.*?\)\s*\n`)
var importSingle = regexp.MustCompile(`(?m)^\s*import\s+"[^"]*"\s*\n`)

// stripHeader removes package and import declarations from CUE source.
func stripHeader(src string) string {
	src = pkgLine.ReplaceAllString(src, "")
	src = importBlock.ReplaceAllString(src, "")
	src = importSingle.ReplaceAllString(src, "")
	return src
}

// LoadDir reads all .cue files from dir, compiles each with the schema, and
// returns the combined list of units.
func LoadDir(dir string) ([]UnitDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cueload: read dir: %w", err)
	}
	return loadEntries(entries, os.DirFS(dir))
}

// LoadFS loads .cue files from an fs.FS (e.g. an embed.FS or os.DirFS).
// This is the general-purpose loader for any filesystem source.
func LoadFS(fsys fs.FS) ([]UnitDef, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("cueload: read fs: %w", err)
	}
	return loadEntries(entries, fsys)
}

func loadEntries(entries []fs.DirEntry, fsys fs.FS) ([]UnitDef, error) {
	schema := stripHeader(getSchema())
	var all []UnitDef

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cue") || e.Name() == "schema.cue" {
			continue
		}
		raw, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("cueload: read %s: %w", e.Name(), err)
		}
		units, err := compileAndExtract(schema, string(raw), e.Name())
		if err != nil {
			return nil, err
		}
		all = append(all, units...)
	}
	return all, nil
}

// CompileSource compiles a raw CUE source string (no package line needed)
// against the schema and extracts units. Exported for testing.
func CompileSource(src string) ([]UnitDef, error) {
	schema := stripHeader(getSchema())
	return compileAndExtract(schema, src, "<inline>")
}

func compileAndExtract(schema, domainSrc, filename string) ([]UnitDef, error) {
	combined := schema + "\n" + stripHeader(domainSrc)

	ctx := cuecontext.New()
	val := ctx.CompileString(combined, cue.Filename(filename))
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("cueload: compile %s: %w", filename, err)
	}

	unitsList := val.LookupPath(cue.ParsePath("units"))
	if err := unitsList.Err(); err != nil {
		return nil, fmt.Errorf("cueload: lookup units in %s: %w", filename, err)
	}

	iter, err := unitsList.List()
	if err != nil {
		return nil, fmt.Errorf("cueload: iterate units in %s: %w", filename, err)
	}

	var defs []UnitDef
	for iter.Next() {
		v := iter.Value()
		ud, err := decodeUnit(v)
		if err != nil {
			return nil, fmt.Errorf("cueload: decode unit in %s: %w", filename, err)
		}
		defs = append(defs, ud)
	}
	return defs, nil
}

// typed fields that go into struct fields, not Slots.
var typedFields = map[string]bool{"name": true, "worth": true, "isA": true}

func decodeUnit(v cue.Value) (UnitDef, error) {
	var ud UnitDef
	ud.Slots = make(map[string]any)

	nameVal := v.LookupPath(cue.ParsePath("name"))
	if err := nameVal.Err(); err != nil {
		return ud, fmt.Errorf("missing name: %w", err)
	}
	ud.Name, _ = nameVal.String()

	worthVal := v.LookupPath(cue.ParsePath("worth"))
	if err := worthVal.Err(); err != nil {
		return ud, fmt.Errorf("missing worth: %w", err)
	}
	w, _ := worthVal.Int64()
	ud.Worth = int(w)

	isaVal := v.LookupPath(cue.ParsePath("isA"))
	if err := isaVal.Err(); err == nil {
		iter, err := isaVal.List()
		if err == nil {
			for iter.Next() {
				s, _ := iter.Value().String()
				ud.IsA = append(ud.IsA, s)
			}
		}
	}
	if ud.IsA == nil {
		ud.IsA = []string{}
	}

	// Walk all fields and put non-typed ones into Slots.
	fields, _ := v.Fields(cue.Optional(true))
	for fields.Next() {
		label := fields.Selector().String()
		if typedFields[label] {
			continue
		}
		ud.Slots[label] = goValue(fields.Value())
	}
	return ud, nil
}

func goValue(v cue.Value) any {
	switch v.IncompleteKind() {
	case cue.StringKind:
		s, _ := v.String()
		return s
	case cue.IntKind:
		n, _ := v.Int64()
		return int(n)
	case cue.FloatKind:
		f, _ := v.Float64()
		return f
	case cue.BoolKind:
		b, _ := v.Bool()
		return b
	case cue.ListKind:
		return goList(v)
	case cue.StructKind:
		return goStruct(v)
	default:
		// Fallback: try to decode as string.
		s, err := v.String()
		if err == nil {
			return s
		}
		s2, _ := v.Eval().String()
		return s2
	}
}

func goList(v cue.Value) any {
	iter, err := v.List()
	if err != nil {
		return []any{}
	}

	var ints []int
	var strs []string
	var mixed []any
	allInt, allStr := true, true

	for iter.Next() {
		g := goValue(iter.Value())
		mixed = append(mixed, g)
		if _, ok := g.(int); !ok {
			allInt = false
		}
		if _, ok := g.(string); !ok {
			allStr = false
		}
	}

	if len(mixed) == 0 {
		return []any{}
	}
	if allInt {
		for _, x := range mixed {
			ints = append(ints, x.(int))
		}
		return ints
	}
	if allStr {
		for _, x := range mixed {
			strs = append(strs, x.(string))
		}
		return strs
	}
	return mixed
}

func goStruct(v cue.Value) any {
	m := make(map[string]any)
	fields, _ := v.Fields(cue.Optional(true))
	for fields.Next() {
		m[fields.Selector().String()] = goValue(fields.Value())
	}
	return m
}

// LoadFile loads units from a single CUE file on disk.
func LoadFile(path string) ([]UnitDef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cueload: read file: %w", err)
	}
	schema := stripHeader(getSchema())
	return compileAndExtract(schema, string(raw), filepath.Base(path))
}
