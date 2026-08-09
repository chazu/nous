package nogoodexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	PlanCommit               = "23ba4ff097c1c6ded9f488eabf4d96d4eccfbea3"
	ReportVersion            = "nogood-trials/v2"
	ReportByteCap            = 16777216
	FixtureBundleByteCap     = 16777216
	NoMatchBridgeOverheadCap = 83
	DevelopmentTaskCount     = 96
	ValidationTaskCount      = 192
	LockedTaskCount          = 384
	PolicyCount              = 13
	TranscriptBundleChunks   = 26
)

// PreregisteredManifestJSON is the exact source authority reproduced in every
// report. Keep this canonical one-line representation synchronized with the
// accepted plan at PlanCommit.
const PreregisteredManifestJSON = `{"experiment_version":"nogoods/v2","seed_authority":"part3/nogoods/v1","generator_version":"blocked-pair-csp/v1","grammar_version":"three-role-three-edge-mask/v1","semantics_version":"finite-neq-csp/v1","oracle_version":"independent-exhaustive-coloring/v1","baseline_version":"mac-cbj-mrv-degree/v1","cost_version":"nogood-lifecycle-events/v2","statistics_version":"paired-stratified-bootstrap/v1","report_version":"nogood-trials/v2","integrity_contract":"budgeted-transcript","training_seeds":{"start":831001,"count":4,"step":1},"competence_development_seeds":{"start":831101,"count":8,"step":1},"competence_validation_seeds":{"start":831201,"count":16,"step":1},"development_seeds":{"start":832001,"count":96,"step":1},"validation_seeds":{"start":833001,"count":192,"step":1},"locked_tasks":384,"value_count":4,"minimum_variables":3,"maximum_variables":8,"maximum_edges":18,"schema_roles":["anchor","pair-0","pair-1"],"candidate_edge_masks":8,"training_examples":4,"maximum_training_completions":2,"target_certificate_completions":1,"training_work_cap":2000,"target_prune_work_cap":128,"no_match_bridge_overhead_cap":83,"policy_work_cap_per_task":2000000,"bridge_task_pop_cap":2000,"attributed_unit_cap":200000,"report_byte_cap":16777216,"fixture_bundle_byte_cap":16777216,"transcript_event_cap_per_execution":8000000,"transcript_event_cap_per_bundle":16000000,"transcript_raw_byte_cap_per_execution":1073741824,"transcript_raw_byte_cap_per_bundle":2147483648,"transcript_gzip_byte_cap_per_execution":1074000000,"transcript_gzip_byte_cap_per_bundle":2148000000,"minimum_primary_reduction":0.00,"maximum_nonreusable_harm":0.10,"alpha":0.05,"confidence_interval":"paired-stratified-bootstrap-two-sided-95","paired_test":"paired-stratified-randomization-two-sided","bootstrap_replicates":10000,"randomization_replicates":10000,"bootstrap_indices_zero_based":[249,9749],"development_power_outer_replicates":2000,"development_power_inner_replicates":2000,"development_power_randomization_replicates":2000,"power_bootstrap_indices_zero_based":[49,1949],"minimum_locked_power":0.80,"tie_policy":"canonical-mask-then-semantic-key","mutation_enabled":false}`

func preregisteredManifest() json.RawMessage {
	return json.RawMessage(PreregisteredManifestJSON)
}

func validatePreregisteredManifest() error {
	var decoded any
	if err := json.Unmarshal([]byte(PreregisteredManifestJSON), &decoded); err != nil {
		return fmt.Errorf("decode preregistered manifest: %w", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(PreregisteredManifestJSON)); err != nil {
		return err
	}
	if !bytes.Equal(compact.Bytes(), []byte(PreregisteredManifestJSON)) {
		return fmt.Errorf("preregistered manifest is not canonical JSON")
	}
	return nil
}
