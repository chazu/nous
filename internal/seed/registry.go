package seed

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chazu/nous/internal/cueload"
	"github.com/chazu/nous/internal/unit"
)

// DomainsDir is set by the CLI. Empty means use embedded.
var DomainsDir string

// LoadDomain loads common types then the named domain from CUE files.
// After all units are loaded, registers inverse slot pairs and computes
// the initial inverse index.
func LoadDomain(s *unit.Store, name string) error {
	// Load common types first (includes slot ontology)
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

	// Register inverse pairs from slot definition units
	registerInverses(s)

	// Compute initial inverse index from existing slot values
	computeInverseIndex(s)

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

// registerInverses scans for slot definition units with an "inverse" field
// and registers each pair with the store.
func registerInverses(s *unit.Store) {
	for _, name := range s.All() {
		if !s.IsA(name, "Slot") {
			continue
		}
		u := s.Get(name)
		if u == nil {
			continue
		}
		inv := u.GetString("inverse")
		if inv != "" {
			// Register using lowercase first char to match actual slot keys
			// Slot def units are named "Domain" but slot keys are "domain"
			slotKey := lcFirst(name)
			invKey := lcFirst(inv)
			s.RegisterInverse(slotKey, invKey)
		}
	}
}

// computeInverseIndex scans all units and builds inverse slot values
// from existing data. E.g., if SetUnion has range=["Set"], then
// Set gets isRangeOf=["SetUnion"].
func computeInverseIndex(s *unit.Store) {
	for _, name := range s.All() {
		u := s.Get(name)
		if u == nil {
			continue
		}
		for k, v := range u.Slots {
			// Use SetSlot to trigger inverse maintenance
			// But only for slots that have registered inverses
			s.SetSlot(name, k, v)
		}
	}
}

func lcFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// Available returns the list of known domain names.
func Available() string {
	return "math, observations"
}
