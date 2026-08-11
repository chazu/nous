package actionrelationexp

import (
	"encoding/json"
	"testing"
)

func TestFrozenManifest(t *testing.T) {
	if err := ValidateManifest(); err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal([]byte(PreregisteredManifestJSON), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["experiment_version"] != ExperimentVersion || manifest["seed_authority"] != SeedAuthority || manifest["training_observations_per_curriculum"] != float64(16) || manifest["maximum_normalized_guards"] != float64(512) || PlanCommit != "15808faae785fe22b025b6de3a6751ed6d365c00" {
		t.Fatalf("manifest identity drift: %#v", manifest)
	}
	if manifest["mutation_enabled"] != false || manifest["tie_policy"] != "maximum-positive-coverage-then-minimum-literals-retain-all-ties-unanimous-use" {
		t.Fatalf("manifest policy drift: %#v", manifest)
	}
}
