package causalv2

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ControlCertificateDomain   = "causal-control-certificate/v2"
	ControlBundleDomain        = "causal-control-bundle/v2"
	ControlEvidenceDomain      = "causal-control-evidence/v2"
	CorruptionEnumeratorDomain = "causal-corruption-enumerator/v2"
)

var ControlNames = []string{
	"hidden-twin", "wrong-context", "static-rule", "recomputed-rule", "opaque-alias", "presentation-order", "proposal-order", "cost-perturbation", "occupied-name", "alternate-descriptor", "mutation-inert", "child-vm", "stale-response", "duplicate-response", "corruption-suite", "deterministic-json", "no-credit", "dependency",
}

type ControlResult struct {
	ProfileDigest    string   `json:"profile_digest"`
	Actions          []string `json:"actions"`
	Outcomes         []string `json:"outcomes"`
	PosteriorDigests []string `json:"posterior_digests"`
	Costs            []int    `json:"costs"`
	Terminal         string   `json:"terminal"`
	Score            int      `json:"score"`
	FailureCode      string   `json:"failure_code"`
	TranscriptDigest string   `json:"transcript_digest"`
}

func (result ControlResult) Validate() error {
	if err := requireDigest("profile_digest", result.ProfileDigest, true); err != nil {
		return err
	}
	if result.Actions == nil || result.Outcomes == nil || result.PosteriorDigests == nil || result.Costs == nil {
		return errors.New("control result arrays must encode as arrays, not null")
	}
	if len(result.Actions) != len(result.Outcomes) || len(result.Actions) != len(result.Costs) {
		return errors.New("control action, outcome, and cost arrays differ in length")
	}
	if len(result.PosteriorDigests) != 0 && len(result.PosteriorDigests) != len(result.Actions)+1 {
		return errors.New("control posterior digests must be empty or contain the initial plus every transition")
	}
	for _, action := range result.Actions {
		if _, err := causal.ParseAction(action); err != nil {
			return err
		}
	}
	for _, outcome := range result.Outcomes {
		if err := validateOutcome(outcome); err != nil {
			return err
		}
	}
	for _, cost := range result.Costs {
		if cost < 0 || cost > 100 {
			return errors.New("invalid control cost")
		}
	}
	for _, digest := range result.PosteriorDigests {
		if err := requireDigest("posterior_digest", digest, false); err != nil {
			return err
		}
	}
	if result.Terminal != "" && !slices.Contains([]string{"identified", "equivalence", "budget-exhausted"}, result.Terminal) {
		return errors.New("invalid control terminal")
	}
	return requireDigest("transcript_digest", result.TranscriptDigest, true)
}

type ControlCertificate struct {
	ControlVersion    string        `json:"control_version"`
	Name              string        `json:"name"`
	FixtureDigest     string        `json:"fixture_digest"`
	TreatmentEvidence ControlResult `json:"treatment_evidence"`
	ControlEvidence   ControlResult `json:"control_evidence"`
	Observed          string        `json:"observed"`
	Passed            bool          `json:"passed"`
	MeterCounts       [15]int64     `json:"meter_counts"`
	Work              int64         `json:"work"`
	CertificateDigest string        `json:"certificate_digest"`
}

func validateControlCertificate(certificate ControlCertificate) error {
	if certificate.ControlVersion != ControlCertificateDomain {
		return errors.New("invalid control certificate version")
	}
	if !slices.Contains(ControlNames, certificate.Name) {
		return fmt.Errorf("invalid control name %q", certificate.Name)
	}
	if err := requireDigest("fixture_digest", certificate.FixtureDigest, true); err != nil {
		return err
	}
	if err := certificate.TreatmentEvidence.Validate(); err != nil {
		return fmt.Errorf("treatment evidence: %w", err)
	}
	if err := certificate.ControlEvidence.Validate(); err != nil {
		return fmt.Errorf("control evidence: %w", err)
	}
	counter := CounterFromCounts(certificate.MeterCounts)
	if err := counter.Validate(); err != nil {
		return err
	}
	if certificate.Work != counter.TotalWork {
		return errors.New("control work does not equal meter total_work")
	}
	return nil
}

func controlCertificateDigest(certificate ControlCertificate) (string, error) {
	certificate.CertificateDigest = ""
	return Digest(ControlCertificateDomain, certificate)
}

func SignControlCertificate(certificate *ControlCertificate) error {
	if certificate == nil {
		return errors.New("nil control certificate")
	}
	normalizeControlResult(&certificate.TreatmentEvidence)
	normalizeControlResult(&certificate.ControlEvidence)
	if err := validateControlCertificate(*certificate); err != nil {
		return err
	}
	digest, err := controlCertificateDigest(*certificate)
	if err != nil {
		return err
	}
	certificate.CertificateDigest = digest
	encoded, err := CanonicalJSON(certificate)
	if err != nil {
		return err
	}
	return CheckByteCap(encoded, PreregisteredManifest().ControlCertificateByteCap)
}

func normalizeControlResult(result *ControlResult) {
	if result.Actions == nil {
		result.Actions = []string{}
	}
	if result.Outcomes == nil {
		result.Outcomes = []string{}
	}
	if result.PosteriorDigests == nil {
		result.PosteriorDigests = []string{}
	}
	if result.Costs == nil {
		result.Costs = []int{}
	}
}

func VerifyControlCertificate(data []byte) (ControlCertificate, error) {
	certificate, err := StrictDecode[ControlCertificate](data)
	if err != nil {
		return certificate, err
	}
	if err := validateControlCertificate(certificate); err != nil {
		return certificate, err
	}
	if err := requireDigest("certificate_digest", certificate.CertificateDigest, false); err != nil {
		return certificate, err
	}
	want, err := controlCertificateDigest(certificate)
	if err != nil {
		return certificate, err
	}
	if certificate.CertificateDigest != want {
		return certificate, errors.New("control certificate digest mismatch")
	}
	if err := CheckByteCap(data, PreregisteredManifest().ControlCertificateByteCap); err != nil {
		return certificate, err
	}
	return certificate, nil
}

type ControlBundle struct {
	ControlBundleVersion string               `json:"control_bundle_version"`
	Certificates         []ControlCertificate `json:"certificates"`
	ControlBundleDigest  string               `json:"control_bundle_digest"`
}

type CacheTrace struct {
	Statuses []string `json:"statuses"`
	Hits     int      `json:"hits"`
	Misses   int      `json:"misses"`
}

type MatrixControlRow struct {
	Seed                       int64         `json:"seed"`
	FixtureDigest              string        `json:"fixture_digest"`
	TreatmentEpisodeDigest     string        `json:"treatment_episode_digest"`
	TreatmentCertificateDigest string        `json:"treatment_certificate_digest"`
	ControlEpisodeDigest       string        `json:"control_episode_digest"`
	ControlCertificateDigest   string        `json:"control_certificate_digest"`
	Treatment                  ControlResult `json:"treatment"`
	Control                    ControlResult `json:"control"`
	TreatmentMeterCounts       [15]int64     `json:"treatment_meter_counts"`
	ControlMeterCounts         [15]int64     `json:"control_meter_counts"`
	TreatmentCache             CacheTrace    `json:"treatment_cache"`
	ControlCache               CacheTrace    `json:"control_cache"`
}

type MutationConfig struct {
	Enabled           bool    `json:"enabled"`
	Interval          int     `json:"interval"`
	MaxMutants        int     `json:"max_mutants"`
	MutantWorth       int     `json:"mutant_worth"`
	ValidateOnly      bool    `json:"validate_only"`
	MinApplics        int     `json:"min_applics"`
	MutationThreshold float64 `json:"mutation_threshold"`
}

type MutantRecord struct {
	Name          string `json:"name"`
	MutantOf      string `json:"mutant_of"`
	SourceSlot    string `json:"source_slot"`
	Operation     string `json:"operation"`
	ProgramDigest string `json:"program_digest"`
	Worth         int    `json:"worth"`
}

type MutationProof struct {
	FixtureDigest  string         `json:"fixture_digest"`
	OffConfig      MutationConfig `json:"off_config"`
	OnConfig       MutationConfig `json:"on_config"`
	OffResult      ControlResult  `json:"off_result"`
	OnResult       ControlResult  `json:"on_result"`
	OffMutants     []MutantRecord `json:"off_mutants"`
	OnMutants      []MutantRecord `json:"on_mutants"`
	OffMeterCounts [15]int64      `json:"off_meter_counts"`
	OnMeterCounts  [15]int64      `json:"on_meter_counts"`
}

type ChildVMProof struct {
	FixtureDigest      string    `json:"fixture_digest"`
	ProfileDigest      string    `json:"profile_digest"`
	Operation          string    `json:"operation"`
	ArtifactsBefore    int       `json:"artifacts_before"`
	ArtifactsAfter     int       `json:"artifacts_after"`
	MeterCountsBefore  [15]int64 `json:"meter_counts_before"`
	MeterCountsAfter   [15]int64 `json:"meter_counts_after"`
	TeacherCallsBefore int       `json:"teacher_calls_before"`
	TeacherCallsAfter  int       `json:"teacher_calls_after"`
	FailureCode        string    `json:"failure_code"`
}

type Base64URL string

func EncodeBase64URL(data []byte) Base64URL {
	return Base64URL(base64.RawURLEncoding.EncodeToString(data))
}

func (value Base64URL) Bytes() ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(string(value))
	if err != nil || EncodeBase64URL(decoded) != value {
		return nil, errors.New("noncanonical unpadded base64url")
	}
	return decoded, nil
}

type CorruptionCase struct {
	Name               string    `json:"name"`
	MutationDescriptor string    `json:"mutation_descriptor"`
	MutatedBytesDigest string    `json:"mutated_bytes_digest"`
	RejectionCode      string    `json:"rejection_code"`
	MeterCounts        [15]int64 `json:"meter_counts"`
}

type NoCreditProof struct {
	CentralProfileBytes      Base64URL                `json:"central_profile_bytes"`
	CertificateDigests       []string                 `json:"certificate_digests"`
	ArtifactBytes            []Base64URL              `json:"artifact_bytes"`
	Aggregates               []RuleAggregatePayload   `json:"aggregates"`
	CentralTranscript        []CentralTranscriptEvent `json:"central_transcript"`
	TaskMeterItems           []TaskMeterItem          `json:"task_meter_items"`
	Counts                   [15]int64                `json:"counts"`
	Resolution               string                   `json:"resolution"`
	WinnerTies               []string                 `json:"winner_ties"`
	SelectedRule             string                   `json:"selected_rule"`
	TerminalTranscriptDigest string                   `json:"terminal_transcript_digest"`
}

type CorruptionProof struct {
	EnumeratorVersion string           `json:"enumerator_version"`
	FixtureBytes      Base64URL        `json:"fixture_bytes"`
	ProfileBytes      Base64URL        `json:"profile_bytes"`
	BaselineArtifacts []Base64URL      `json:"baseline_artifacts"`
	CaseCount         int              `json:"case_count"`
	CaseSetDigest     string           `json:"case_set_digest"`
	Cases             []CorruptionCase `json:"cases"`
}

type DependencyParameter struct {
	Function       string `json:"function"`
	ParameterIndex int    `json:"parameter_index"`
	Type           string `json:"type"`
}

type DependencyFile struct {
	Path                       string                `json:"path"`
	SourceSHA256               string                `json:"source_sha256"`
	Imports                    []string              `json:"imports"`
	ExportedFunctionParameters []DependencyParameter `json:"exported_function_parameters"`
}

type RunnerField struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	HiddenBearing bool   `json:"hidden_bearing"`
}

type DependencyProof struct {
	AuditedCommit  string           `json:"audited_commit"`
	AuditedRoots   []string         `json:"audited_roots"`
	Files          []DependencyFile `json:"files"`
	RunnerMethods  []string         `json:"runner_methods"`
	RunnerFields   []RunnerField    `json:"runner_fields"`
	TeacherMethods []string         `json:"teacher_methods"`
	Lookups        int64            `json:"lookups"`
	Forbidden      []string         `json:"forbidden"`
}

type ControlEvidence struct {
	ControlEvidenceVersion string             `json:"control_evidence_version"`
	SelectedRule           string             `json:"selected_rule"`
	StaticRule             string             `json:"static_rule"`
	StaticMatrix           []MatrixControlRow `json:"static_matrix"`
	RecomputedMatrix       []MatrixControlRow `json:"recomputed_matrix"`
	Mutation               MutationProof      `json:"mutation"`
	ChildVM                ChildVMProof       `json:"child_vm"`
	Corruption             CorruptionProof    `json:"corruption"`
	NoCredit               NoCreditProof      `json:"no_credit"`
	Dependency             DependencyProof    `json:"dependency"`
	ControlEvidenceDigest  string             `json:"control_evidence_digest"`
}

func validateCacheTrace(trace CacheTrace, actionCount int, recomputed bool) error {
	if trace.Statuses == nil || len(trace.Statuses) != 6*actionCount || trace.Hits+trace.Misses != len(trace.Statuses) {
		return errors.New("cache trace cardinality mismatch")
	}
	hits, misses := 0, 0
	for _, status := range trace.Statuses {
		switch status {
		case "hit":
			hits++
		case "miss":
			misses++
		default:
			return errors.New("invalid cache status")
		}
	}
	if hits != trace.Hits || misses != trace.Misses || (recomputed && hits != 0) {
		return errors.New("cache trace counts mismatch")
	}
	return nil
}

func validateControlEvidence(evidence ControlEvidence) error {
	if evidence.ControlEvidenceVersion != ControlEvidenceDomain {
		return errors.New("invalid control evidence version")
	}
	if _, err := causal.ParseRule(evidence.SelectedRule); err != nil {
		return fmt.Errorf("invalid selected control rule: %w", err)
	}
	rules := causal.Rules()
	if len(rules) == 0 || evidence.StaticRule != rules[0].Code() {
		return errors.New("static control rule is not semantic-first grammar order")
	}
	for name, rows := range map[string][]MatrixControlRow{"static": evidence.StaticMatrix, "recomputed": evidence.RecomputedMatrix} {
		if len(rows) != PreregisteredManifest().TrainingSeeds.Count {
			return fmt.Errorf("%s matrix has %d rows, want 12", name, len(rows))
		}
		for index, row := range rows {
			wantSeed := PreregisteredManifest().TrainingSeeds.Start + int64(index)*PreregisteredManifest().TrainingSeeds.Step
			if row.Seed != wantSeed || requireDigest("matrix fixture digest", row.FixtureDigest, false) != nil || requireDigest("treatment episode digest", row.TreatmentEpisodeDigest, false) != nil || requireDigest("treatment certificate digest", row.TreatmentCertificateDigest, false) != nil {
				return fmt.Errorf("%s matrix row %d has invalid identity", name, index)
			}
			if name == "static" {
				if requireDigest("control episode digest", row.ControlEpisodeDigest, false) != nil || requireDigest("control certificate digest", row.ControlCertificateDigest, false) != nil {
					return errors.New("static matrix control digests are empty")
				}
			} else if row.ControlEpisodeDigest != "" || row.ControlCertificateDigest != "" {
				return errors.New("recomputed matrix control digests must be empty")
			}
			if err := row.Treatment.Validate(); err != nil {
				return fmt.Errorf("%s treatment row %d: %w", name, index, err)
			}
			if err := row.Control.Validate(); err != nil {
				return fmt.Errorf("%s control row %d: %w", name, index, err)
			}
			if err := CounterFromCounts(row.TreatmentMeterCounts).Validate(); err != nil {
				return err
			}
			if err := CounterFromCounts(row.ControlMeterCounts).Validate(); err != nil {
				return err
			}
			if err := validateCacheTrace(row.TreatmentCache, len(row.Treatment.Actions), false); err != nil {
				return err
			}
			if err := validateCacheTrace(row.ControlCache, len(row.Control.Actions), name == "recomputed"); err != nil {
				return err
			}
		}
	}
	if err := validateMutationProof(evidence.Mutation); err != nil {
		return err
	}
	if err := validateChildVMProof(evidence.ChildVM); err != nil {
		return err
	}
	if evidence.Corruption.EnumeratorVersion != CorruptionEnumeratorDomain || evidence.Corruption.FixtureBytes == "" || evidence.Corruption.ProfileBytes == "" || len(evidence.Corruption.BaselineArtifacts) == 0 || len(evidence.Corruption.Cases) == 0 || evidence.Corruption.CaseCount != len(evidence.Corruption.Cases) || evidence.Corruption.CaseCount > 486 {
		return errors.New("corruption proof is incomplete")
	}
	if _, err := evidence.Corruption.FixtureBytes.Bytes(); err != nil {
		return err
	}
	if _, err := evidence.Corruption.ProfileBytes.Bytes(); err != nil {
		return err
	}
	caseNames := make([]string, len(evidence.Corruption.Cases))
	for i, artifact := range evidence.Corruption.BaselineArtifacts {
		if _, err := artifact.Bytes(); err != nil {
			return fmt.Errorf("baseline artifact %d: %w", i, err)
		}
	}
	for i, item := range evidence.Corruption.Cases {
		caseNames[i] = item.Name
		if item.Name == "" || item.MutationDescriptor != item.Name || requireDigest("mutated bytes digest", item.MutatedBytesDigest, false) != nil || item.RejectionCode == "" {
			return fmt.Errorf("invalid corruption case %d", i)
		}
		if err := CounterFromCounts(item.MeterCounts).Validate(); err != nil {
			return err
		}
		encoded, _ := CanonicalJSON(item)
		if len(encoded) > 2048 {
			return errors.New("corruption case exceeds byte cap")
		}
	}
	wantCaseDigest, _ := Digest(CorruptionEnumeratorDomain, caseNames)
	if evidence.Corruption.CaseSetDigest != wantCaseDigest {
		return errors.New("corruption case-set digest mismatch")
	}
	if len(evidence.NoCredit.CertificateDigests) != 480 || len(evidence.NoCredit.Aggregates) != 40 || len(evidence.NoCredit.ArtifactBytes) != 1041 || len(evidence.NoCredit.CentralTranscript) != 0 || len(evidence.NoCredit.TaskMeterItems) != 525 || len(evidence.NoCredit.WinnerTies) != 0 || evidence.NoCredit.Resolution != "unresolved" || evidence.NoCredit.SelectedRule != "" || evidence.NoCredit.TerminalTranscriptDigest != ZeroDigest {
		return errors.New("no-credit proof is incomplete or resolved")
	}
	if _, err := evidence.NoCredit.CentralProfileBytes.Bytes(); err != nil {
		return err
	}
	for _, digest := range evidence.NoCredit.CertificateDigests {
		if err := requireDigest("no-credit certificate digest", digest, false); err != nil {
			return err
		}
	}
	for i, artifact := range evidence.NoCredit.ArtifactBytes {
		if _, err := artifact.Bytes(); err != nil {
			return fmt.Errorf("no-credit artifact %d: %w", i, err)
		}
	}
	if err := CounterFromCounts(evidence.NoCredit.Counts).Validate(); err != nil {
		return fmt.Errorf("no-credit counts: %w", err)
	}
	for index, item := range evidence.NoCredit.TaskMeterItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("no-credit task %d: %w", index, err)
		}
	}
	if err := validateDependencyProof(evidence.Dependency); err != nil {
		return err
	}
	return nil
}

func validateMutationProof(proof MutationProof) error {
	if requireDigest("mutation fixture digest", proof.FixtureDigest, false) != nil || proof.OffConfig.Enabled || !proof.OnConfig.Enabled {
		return errors.New("invalid mutation proof identity")
	}
	want := MutationConfig{Interval: 1, MaxMutants: 1, MutantWorth: 400, ValidateOnly: false, MinApplics: 1, MutationThreshold: 2}
	off, on := proof.OffConfig, proof.OnConfig
	off.Enabled, on.Enabled = false, false
	if off != want || on != want || len(proof.OffMutants) != 0 || len(proof.OnMutants) < 1 {
		return errors.New("mutation proof does not execute frozen off/on trial")
	}
	if err := proof.OffResult.Validate(); err != nil {
		return err
	}
	if err := proof.OnResult.Validate(); err != nil {
		return err
	}
	if proof.OffResult.FailureCode != "" || proof.OnResult.FailureCode != "" || !slices.Equal(proof.OffResult.Actions, proof.OnResult.Actions) || !slices.Equal(proof.OffResult.Outcomes, proof.OnResult.Outcomes) || !slices.Equal(proof.OffResult.PosteriorDigests, proof.OnResult.PosteriorDigests) || !slices.Equal(proof.OffResult.Costs, proof.OnResult.Costs) || proof.OffResult.Terminal != proof.OnResult.Terminal || proof.OffResult.Score != proof.OnResult.Score {
		return errors.New("mutation changed the semantic projection")
	}
	if err := CounterFromCounts(proof.OffMeterCounts).Validate(); err != nil {
		return err
	}
	if err := CounterFromCounts(proof.OnMeterCounts).Validate(); err != nil {
		return err
	}
	for index, mutant := range proof.OnMutants {
		if mutant.Name == "" || mutant.MutantOf == "" || mutant.SourceSlot == "" || mutant.Operation == "" || requireDigest("mutant program digest", mutant.ProgramDigest, false) != nil {
			return errors.New("invalid mutant record")
		}
		if index > 0 && proof.OnMutants[index-1].Name >= mutant.Name {
			return errors.New("mutant records are not sorted")
		}
	}
	return nil
}

func validateChildVMProof(proof ChildVMProof) error {
	if requireDigest("child fixture digest", proof.FixtureDigest, false) != nil || requireDigest("child profile digest", proof.ProfileDigest, false) != nil || proof.Operation != "causal-v2-task-valid?" || proof.FailureCode != "child-vm-unauthorized" || proof.ArtifactsBefore != proof.ArtifactsAfter || proof.MeterCountsBefore != proof.MeterCountsAfter || proof.TeacherCallsBefore != proof.TeacherCallsAfter {
		return errors.New("child VM did not fail before evidence")
	}
	return CounterFromCounts(proof.MeterCountsAfter).Validate()
}

func validateDependencyProof(proof DependencyProof) error {
	if !validCommit(proof.AuditedCommit) || proof.AuditedRoots == nil || !slices.Equal(proof.AuditedRoots, []string{"."}) || len(proof.Files) == 0 || proof.Lookups <= 0 || len(proof.Forbidden) != 0 {
		return errors.New("dependency proof is incomplete or forbidden")
	}
	last := ""
	for _, file := range proof.Files {
		if file.Path <= last || strings.HasPrefix(file.Path, "./") || strings.Contains("/"+file.Path+"/", "/../") || requireDigest("dependency source digest", file.SourceSHA256, false) != nil || file.Imports == nil || file.ExportedFunctionParameters == nil || !sort.StringsAreSorted(file.Imports) {
			return errors.New("invalid dependency file")
		}
		last = file.Path
		for index, parameter := range file.ExportedFunctionParameters {
			if parameter.Function == "" || parameter.ParameterIndex < 0 || parameter.Type == "" || index > 0 && (file.ExportedFunctionParameters[index-1].Function > parameter.Function || file.ExportedFunctionParameters[index-1].Function == parameter.Function && file.ExportedFunctionParameters[index-1].ParameterIndex >= parameter.ParameterIndex) {
				return errors.New("dependency parameters are not canonical")
			}
		}
	}
	if proof.RunnerMethods == nil || proof.RunnerFields == nil || proof.TeacherMethods == nil || proof.Forbidden == nil || !sort.StringsAreSorted(proof.RunnerMethods) || !sort.StringsAreSorted(proof.TeacherMethods) {
		return errors.New("dependency arrays are not canonical")
	}
	for index, field := range proof.RunnerFields {
		if field.Name == "" || field.Type == "" || field.HiddenBearing || index > 0 && proof.RunnerFields[index-1].Name >= field.Name {
			return errors.New("runner fields are not canonical or hidden-free")
		}
	}
	return nil
}

func controlEvidenceDigest(evidence ControlEvidence) (string, error) {
	evidence.ControlEvidenceDigest = ""
	return Digest(ControlEvidenceDomain, evidence)
}

func controlEvidenceArrayBytes(values ...any) (int, error) {
	total := 0
	for _, value := range values {
		encoded, err := CanonicalJSON(value)
		if err != nil {
			return 0, err
		}
		total += len(encoded)
	}
	return total, nil
}

func checkControlEvidenceCaps(evidence ControlEvidence) error {
	m := PreregisteredManifest()
	encoded, err := CanonicalJSON(evidence)
	if err != nil {
		return err
	}
	if err := CheckByteCap(encoded, m.ControlEvidenceByteCap); err != nil {
		return err
	}
	groups := []struct {
		name   string
		cap    int
		values []any
	}{
		{"no_credit_artifacts", m.ControlEvidenceNoCreditArtifactsByteCap, []any{evidence.NoCredit.ArtifactBytes}},
		{"corruption_baseline", m.ControlEvidenceCorruptionBaselineByteCap, []any{evidence.Corruption.BaselineArtifacts}},
		{"corruption_cases", m.ControlEvidenceCorruptionCasesByteCap, []any{evidence.Corruption.Cases}},
		{"dependency_files", m.ControlEvidenceDependencyFilesByteCap, []any{evidence.Dependency.Files}},
		{"other_records", m.ControlEvidenceOtherRecordsByteCap, []any{evidence.StaticMatrix, evidence.RecomputedMatrix, evidence.Mutation.OffMutants, evidence.Mutation.OnMutants, evidence.NoCredit.CertificateDigests, evidence.NoCredit.Aggregates, evidence.NoCredit.CentralTranscript, evidence.NoCredit.TaskMeterItems, evidence.Dependency.AuditedRoots, evidence.Dependency.RunnerMethods, evidence.Dependency.RunnerFields, evidence.Dependency.TeacherMethods, evidence.Dependency.Forbidden}},
	}
	for _, group := range groups {
		length, err := controlEvidenceArrayBytes(group.values...)
		if err != nil {
			return err
		}
		if length > group.cap {
			return fmt.Errorf("control evidence %s bytes=%d exceed cap=%d", group.name, length, group.cap)
		}
	}
	shell := evidence
	shell.ControlEvidenceDigest = ""
	shell.StaticMatrix = []MatrixControlRow{}
	shell.RecomputedMatrix = []MatrixControlRow{}
	shell.Mutation.OffMutants = []MutantRecord{}
	shell.Mutation.OnMutants = []MutantRecord{}
	shell.Corruption.BaselineArtifacts = []Base64URL{}
	shell.Corruption.Cases = []CorruptionCase{}
	shell.NoCredit.CertificateDigests = []string{}
	shell.NoCredit.ArtifactBytes = []Base64URL{}
	shell.NoCredit.Aggregates = []RuleAggregatePayload{}
	shell.NoCredit.CentralTranscript = []CentralTranscriptEvent{}
	shell.NoCredit.TaskMeterItems = []TaskMeterItem{}
	shell.Dependency.AuditedRoots = []string{}
	shell.Dependency.Files = []DependencyFile{}
	shell.Dependency.RunnerMethods = []string{}
	shell.Dependency.RunnerFields = []RunnerField{}
	shell.Dependency.TeacherMethods = []string{}
	shell.Dependency.Forbidden = []string{}
	shellBytes, err := CanonicalJSON(shell)
	if err != nil {
		return err
	}
	return CheckByteCap(shellBytes, 262144)
}

func SignControlEvidence(evidence *ControlEvidence) error {
	if evidence == nil {
		return errors.New("nil control evidence")
	}
	if err := validateControlEvidence(*evidence); err != nil {
		return err
	}
	digest, err := controlEvidenceDigest(*evidence)
	if err != nil {
		return err
	}
	evidence.ControlEvidenceDigest = digest
	return checkControlEvidenceCaps(*evidence)
}

func VerifyControlEvidence(data []byte) (ControlEvidence, error) {
	evidence, err := StrictDecode[ControlEvidence](data)
	if err != nil {
		return evidence, err
	}
	if err := validateControlEvidence(evidence); err != nil {
		return evidence, err
	}
	if err := requireDigest("control_evidence_digest", evidence.ControlEvidenceDigest, false); err != nil {
		return evidence, err
	}
	want, err := controlEvidenceDigest(evidence)
	if err != nil {
		return evidence, err
	}
	if evidence.ControlEvidenceDigest != want {
		return evidence, errors.New("control evidence digest mismatch")
	}
	if err := checkControlEvidenceCaps(evidence); err != nil {
		return evidence, err
	}
	return evidence, nil
}

func validateControlBundle(bundle ControlBundle) error {
	if bundle.ControlBundleVersion != ControlBundleDomain {
		return errors.New("invalid control bundle version")
	}
	if len(bundle.Certificates) != len(ControlNames) {
		return fmt.Errorf("control certificate count=%d, want %d", len(bundle.Certificates), len(ControlNames))
	}
	for i, certificate := range bundle.Certificates {
		if certificate.Name != ControlNames[i] {
			return errors.New("control certificates are out of fixed order")
		}
		encoded, err := CanonicalJSON(certificate)
		if err != nil {
			return err
		}
		if _, err := VerifyControlCertificate(encoded); err != nil {
			return fmt.Errorf("control certificate %q: %w", certificate.Name, err)
		}
	}
	return nil
}

func controlBundleDigest(bundle ControlBundle) (string, error) {
	bundle.ControlBundleDigest = ""
	return Digest(ControlBundleDomain, bundle)
}

func SignControlBundle(bundle *ControlBundle) error {
	if bundle == nil {
		return errors.New("nil control bundle")
	}
	if err := validateControlBundle(*bundle); err != nil {
		return err
	}
	digest, err := controlBundleDigest(*bundle)
	if err != nil {
		return err
	}
	bundle.ControlBundleDigest = digest
	encoded, err := CanonicalJSON(bundle)
	if err != nil {
		return err
	}
	return CheckByteCap(encoded, PreregisteredManifest().ControlBundleByteCap)
}

func VerifyControlBundle(data []byte) (ControlBundle, error) {
	bundle, err := StrictDecode[ControlBundle](data)
	if err != nil {
		return bundle, err
	}
	if err := validateControlBundle(bundle); err != nil {
		return bundle, err
	}
	if err := requireDigest("control_bundle_digest", bundle.ControlBundleDigest, false); err != nil {
		return bundle, err
	}
	want, err := controlBundleDigest(bundle)
	if err != nil {
		return bundle, err
	}
	if bundle.ControlBundleDigest != want {
		return bundle, errors.New("control bundle digest mismatch")
	}
	if err := CheckByteCap(data, PreregisteredManifest().ControlBundleByteCap); err != nil {
		return bundle, err
	}
	return bundle, nil
}
