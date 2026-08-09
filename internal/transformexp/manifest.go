package transformexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	PlanCommit            = "baff1990798846b9314c9b42745198098c8087f1"
	DevelopmentCount      = 48
	ValidationCount       = 96
	LockedCount           = 128
	ApplicationsPerPolicy = 48
)

const PreregisteredManifestJSON = `{"experiment_version":"transform-schema/v1","seed_authority":"part3/transform-schema/v1","generator_version":"request-reference-forest/v1","term_version":"typed-reference-forest/v1","edit_grammar_version":"set-scalar-from-request/v1","schema_grammar_version":"anchor-target-scope-old-guard-locality/v1","oracle_version":"independent-forest-transform/v1","baseline_version":"lgg-and-bounded-pbe/v1","cache_version":"disabled/v1","cost_version":"transform-lifecycle-events/v1","statistics_version":"paired-stratified-resampling/v1","report_version":"transform-schema-trials/v1","policy_fixture_version":"transform-policy-curriculum/v1","scorer_fixture_version":"transform-scorer-curriculum/v1","transcript_version":"transform-events/v1","integrity_contract":"budgeted-transcript","training_examples_per_curriculum":8,"training_positive_examples":4,"training_negative_examples":4,"heldout_cases_per_curriculum":8,"heldout_positive_cases":4,"heldout_abstention_cases":4,"development_seeds":{"start":841001,"count":48,"step":1},"validation_seeds":{"start":842001,"count":96,"step":1},"locked_curricula":128,"maximum_nodes":12,"maximum_groups":2,"maximum_requests":2,"maximum_definitions":2,"maximum_references":6,"maximum_concrete_edits":4,"anchor_modes":["request-target","from-value","first-local"],"target_masks":["definition","references","definition+references"],"reference_scopes":["local","global"],"old_value_guards":["equals-from","any"],"anchor_locality_guards":["required","none"],"schema_candidates":72,"semantic_families":9,"candidate_refinement_edges":138,"nous_refinement_edges_per_curriculum":12,"nous_candidate_allocations_per_curriculum":13,"schema_application_cap_per_curriculum":48,"competence_schema_application_cap":26000,"competence_program_application_cap":8000,"competence_work_cap":5000000,"generator_schema_application_cap_per_attempt":1200,"generator_schema_application_cap_per_curriculum":120000,"generator_work_cap_per_attempt":200000,"generator_work_cap_per_curriculum":20000000,"oracle_work_cap_per_curriculum":250000,"engine_cycle_cap_per_curriculum":2000,"attributed_unit_cap_per_curriculum":20000,"lifecycle_work_cap_per_curriculum":12000,"report_byte_cap":16777216,"fixture_bundle_byte_cap":16777216,"transcript_event_cap_per_policy_curriculum":50000,"transcript_event_cap_per_policy_locked_panel":6400000,"transcript_raw_byte_cap_per_policy_curriculum":19200000,"transcript_gzip_byte_cap_per_policy_curriculum":19250000,"object_byte_cap_per_policy_curriculum":67108864,"object_leaf_byte_cap":2560,"object_leaf_count_cap_per_policy_curriculum":24002,"object_root_byte_cap_per_policy_curriculum":4194304,"transcript_raw_byte_cap_per_policy_locked_panel":2457600000,"transcript_gzip_byte_cap_per_policy_locked_panel":2464000000,"object_byte_cap_per_policy_locked_panel":8589934592,"minimum_locked_success_advantage":0.1,"minimum_locked_success_rate":0.8,"maximum_false_application_rate":0.0,"maximum_nonmatching_work_ratio":1.25,"alpha":0.05,"confidence_interval":"paired-stratified-bootstrap-two-sided-95","paired_test":"paired-randomization-two-sided","bootstrap_replicates":10000,"randomization_replicates":10000,"bootstrap_indices_zero_based":[249,9749],"development_power_outer_replicates":2000,"development_power_inner_replicates":2000,"minimum_locked_power":0.8,"tie_policy":"minimum-description-length-then-canonical-code-retain-all-ties","mutation_enabled":false}`

func validateManifest() error {
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
