package seed

import (
	"fmt"

	"github.com/chazu/nous/internal/unit"
)

// DomainLoader populates a store with domain-specific concepts.
type DomainLoader func(s *unit.Store)

var domains = map[string]DomainLoader{
	"math": LoadMath,
}

// LoadDomain loads a named domain into the store. Returns an error if unknown.
func LoadDomain(s *unit.Store, name string) error {
	loader, ok := domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, Available())
	}
	loader(s)
	return nil
}

// Available returns a comma-separated list of registered domain names.
func Available() string {
	var names []string
	for name := range domains {
		names = append(names, name)
	}
	return fmt.Sprintf("%v", names)
}

// Register adds a domain loader. Intended for use in init() or test setup.
func Register(name string, loader DomainLoader) {
	domains[name] = loader
}
