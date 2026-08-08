package causalexp

import "testing"

func TestPanelsGenerateDeterministically(t *testing.T) {
	manifest := PreregisteredManifest()
	ranges := []struct {
		panel string
		r     SeedRange
	}{{"development", manifest.DevelopmentSeeds}, {"training", manifest.TrainingSeeds}, {"validation", manifest.ValidationSeeds}, {"locked", manifest.LockedSeeds}}
	for _, item := range ranges {
		for i := 0; i < item.r.Count; i++ {
			seed := item.r.Start + int64(i)*item.r.Step
			a, e := Generate(item.panel, seed, i)
			if e != nil {
				t.Fatalf("%s seed %d: %v", item.panel, seed, e)
			}
			b, e := Generate(item.panel, seed, i)
			if e != nil || a.FixtureDigest != b.FixtureDigest {
				t.Fatalf("nondeterministic %s seed %d", item.panel, seed)
			}
		}
	}
}
