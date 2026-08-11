// Package actionrelationexp owns the guarded-action-relations experiment
// contract. Panel constructors remain absent until implementation review.
package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	PlanCommit        = "95860673664799ea1e18c7cb1f7e433238830216"
	ExperimentVersion = "actionrelations/v1"
	SeedAuthority     = "part3/actionrelations/v1"
)

const PreregisteredManifestJSON = `{"experiment_version":"actionrelations/v1","seed_authority":"part3/actionrelations/v1","state_version":"finite-action-state/v1","action_version":"finite-action/v1","observation_version":"action-pair-observation/v1","guard_version":"action-guard/v1","relation_version":"guarded-action-relation/v1","certificate_version":"local-diamond-certificate/v1","search_version":"certified-sleep-search/v1","cost_version":"actionrelation-lifecycle/v1","statistics_version":"paired-stratified-search-ratio/v2","report_version":"actionrelation-trials/v1","evidence_version":"actionrelation-packed-evidence/v1","maximum_cells":3,"minimum_cell_value":0,"maximum_cell_value":3,"maximum_event_count":8,"maximum_actions":8,"maximum_reachable_states":64,"maximum_history_length":8,"maximum_competence_sequences":40320,"maximum_competence_cases":65536,"maximum_utility_histories":65536,"maximum_normalized_guards":512,"maximum_training_observations":256,"training_observations_per_curriculum":16,"utility_worlds_per_curriculum":6,"development_seeds":{"start":851001,"count":16,"step":1},"validation_seeds":{"start":852001,"count":24,"step":1},"locked_curricula":32,"maximum_generator_attempts":32,"generator_work_cap_per_attempt":1000000,"generator_work_cap_per_curriculum":32000000,"development_power_outer_replicates":2000,"development_power_inner_replicates":2000,"bootstrap_replicates":10000,"randomization_replicates":10000,"minimum_locked_search_work_reduction":0.15,"minimum_locked_saving_coverage":0.8,"minimum_locked_power":0.8,"alpha":0.05,"randomness_version":"sha256-counter-index/v1","maximum_logical_record_bytes":65536,"maximum_pack_bytes":16777216,"maximum_pack_index_rows":4096,"maximum_pack_index_bytes":1048576,"maximum_object_index_rows_per_curriculum":65248,"maximum_object_index_shards_per_curriculum":19,"maximum_structural_output_map_rows_per_curriculum":4192,"maximum_panel_run_rows":1408,"maximum_panel_authority_bytes":3145728,"maximum_competence_evidence_bytes":68157440,"maximum_development_evidence_bytes":2718957568,"maximum_validation_evidence_bytes":4069523456,"maximum_locked_evidence_bytes":5420089344,"maximum_report_bytes":14680064,"tie_policy":"maximum-positive-coverage-then-minimum-literals-retain-all-ties-unanimous-use","mutation_enabled":false}`

func EvidenceRoot(panel string) (string, error) {
	if !panelNames[panel] {
		return "", fmt.Errorf("invalid evidence panel")
	}
	return ".nous/actionrelations-v1-" + panel + "-evidence", nil
}

func ValidateManifest() error {
	var value any
	if err := json.Unmarshal([]byte(PreregisteredManifestJSON), &value); err != nil {
		return err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(PreregisteredManifestJSON)); err != nil {
		return err
	}
	if !bytes.Equal(compact.Bytes(), []byte(PreregisteredManifestJSON)) {
		return fmt.Errorf("manifest is not compact canonical JSON")
	}
	return nil
}
