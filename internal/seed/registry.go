package seed

import (
	"fmt"
	"path/filepath"

	"github.com/chazu/nous/internal/cueload"
	"github.com/chazu/nous/internal/unit"
)

// DomainsDir is set by the CLI. Empty means use embedded.
var DomainsDir string

// LoadDomain loads common types then the named domain from CUE files.
func LoadDomain(s *unit.Store, name string) error {
	// Load common types first
	commonDefs, err := loadFromDir("common")
	if err != nil {
		return fmt.Errorf("loading common: %w", err)
	}
	populateStore(s, commonDefs)

	// Load the requested domain
	domainDefs, err := loadFromDir(name)
	if err != nil {
		return fmt.Errorf("loading domain %s: %w", name, err)
	}
	populateStore(s, domainDefs)

	return nil
}

func loadFromDir(name string) ([]cueload.UnitDef, error) {
	if DomainsDir != "" {
		return cueload.LoadDir(filepath.Join(DomainsDir, name))
	}
	// TODO: embedded fallback will be added later
	return nil, fmt.Errorf("no domains-dir specified and embedded loading not yet implemented")
}

func populateStore(s *unit.Store, defs []cueload.UnitDef) {
	for _, def := range defs {
		u := unit.New(def.Name)
		u.SetWorth(def.Worth)
		if len(def.IsA) > 0 {
			u.Set("isA", def.IsA)
		}
		for k, v := range def.Slots {
			u.Set(k, v)
		}
		s.Put(u)
	}
}

// Available returns the list of known domain names.
func Available() string {
	return "math, observations"
}
