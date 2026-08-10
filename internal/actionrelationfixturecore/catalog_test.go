package actionrelationfixturecore

import "testing"

func TestAllFrozenSkeletonCatalogsAreNonemptyAndExact(t *testing.T) {
	for family := range FamilyNames {
		for _, stratum := range []string{PositiveEffect, Neutral, Adverse} {
			catalog, err := SkeletonCatalog(family, stratum)
			if err != nil {
				t.Fatalf("family %d %s: %v", family, stratum, err)
			}
			if len(catalog) < 2 {
				t.Fatalf("family %d %s catalog has %d entries, need two without replacement", family, stratum, len(catalog))
			}
			previous := ""
			for _, core := range catalog {
				if len(core.World.Occurrences) != 6 || core.ReachableStates > 64 {
					t.Fatalf("family %d %s occurrence/state bounds=%d/%d", family, stratum, len(core.World.Occurrences), core.ReachableStates)
				}
				if previous != "" && string(core.Canonical) <= previous {
					t.Fatalf("family %d %s catalog is not unique canonical-byte order", family, stratum)
				}
				previous = string(core.Canonical)
				switch stratum {
				case PositiveEffect:
					if core.LatentCommutes < 4 {
						t.Fatalf("positive family %d has only %d latent commutations", family, core.LatentCommutes)
					}
				case Neutral:
					if !core.neutral || !core.noLatentMatch {
						t.Fatalf("neutral family %d violates closed predicate", family)
					}
				case Adverse:
					if core.OutsideCommutes < 4 || !core.noLatentMatch {
						t.Fatalf("adverse family %d violates closed predicate", family)
					}
				}
			}
		}
	}
}
