// Package causalcurriculum executes and independently reconstructs the
// accepted active-causal-diagnosis/v2 central credit curriculum.
package causalcurriculum

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/cueload"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ruleCount        = 40
	seedCount        = 12
	certificateCount = ruleCount * seedCount
	taskCount        = 525
	runtimeName      = "Causal.Curriculum.Cursor"
)

//go:embed curriculum.cue
var curriculumFS embed.FS

// Result is the complete unprotected result of one central curriculum run.
// ArtifactBytes are in charge-index order and permit independent reconstruction.
type Result struct {
	ProfileDigest            string                            `json:"profile_digest"`
	TrainingKey              string                            `json:"training_key"`
	CreditEnabled            bool                              `json:"credit_enabled"`
	Applications             []causalv2.ApplicationCertificate `json:"applications"`
	Aggregates               []causalv2.RuleAggregatePayload   `json:"aggregates"`
	WinnerTies               []string                          `json:"winner_ties"`
	SelectedRule             string                            `json:"selected_rule"`
	Unresolved               bool                              `json:"unresolved"`
	ArtifactBytes            [][]byte                          `json:"artifact_bytes"`
	Transcript               []causalv2.CentralTranscriptEvent `json:"transcript"`
	Counts                   causalv2.Counter                  `json:"counts"`
	TaskMeterItems           []causalv2.TaskMeterItem          `json:"task_meter_items"`
	TaskMeterItemsDigest     string                            `json:"task_meter_items_digest"`
	TerminalTranscriptDigest string                            `json:"terminal_transcript_digest"`
}

// Run executes the CUE-owned curriculum from a signed central profile and the
// exact seed-major/rule-major episode/certificate evidence matrix.
func Run(ctx context.Context, centralProfileBytes []byte, episodeBytes, certificateBytes [][]byte) (Result, error) {
	return run(ctx, centralProfileBytes, episodeBytes, certificateBytes, true)
}

// Verify reconstructs a supplied result from its sealed inputs in a fresh
// store, then requires exact canonical equality.
func Verify(centralProfileBytes []byte, episodeBytes, certificateBytes [][]byte, result Result) error {
	if err := verifyResult(centralProfileBytes, episodeBytes, certificateBytes, result); err != nil {
		return fmt.Errorf("reconstruct curriculum result: %w", err)
	}
	want, err := run(context.Background(), centralProfileBytes, episodeBytes, certificateBytes, false)
	if err != nil {
		return fmt.Errorf("fresh curriculum execution: %w", err)
	}
	wantBytes, err := causalv2.CanonicalJSON(want)
	if err != nil {
		return err
	}
	gotBytes, err := causalv2.CanonicalJSON(result)
	if err != nil {
		return err
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		return errors.New("result differs from fresh central reconstruction")
	}
	return nil
}

func run(ctx context.Context, centralProfileBytes []byte, episodeBytes, certificateBytes [][]byte, verify bool) (Result, error) {
	profile, err := causalv2.VerifyCentralProfile(centralProfileBytes)
	if err != nil {
		return Result{}, fmt.Errorf("central profile: %w", err)
	}
	if len(episodeBytes) != certificateCount || len(certificateBytes) != certificateCount {
		return Result{}, fmt.Errorf("episode/certificate matrix has %d/%d entries, want %d each", len(episodeBytes), len(certificateBytes), certificateCount)
	}
	rules := sortedRuleCodes()
	seeds := trainingSeeds(profile.Manifest)
	certificates := make([]causalv2.ApplicationCertificate, certificateCount)
	for index, encoded := range certificateBytes {
		certificate, episode, verifyErr := causalv2.VerifyApplicationCertificateForEpisode(encoded, episodeBytes[index])
		if verifyErr != nil {
			return Result{}, fmt.Errorf("certificate %d: %w", index, verifyErr)
		}
		if certificate.Seed != seeds[index/ruleCount] || certificate.RuleCode != rules[index%ruleCount] || certificate.ProfileDigest != episode.ProfileDigest || !certificate.AllCapsValid || certificate.OracleDisagreements != 0 {
			return Result{}, fmt.Errorf("certificate %d is outside the exact valid matrix", index)
		}
		certificates[index] = certificate
	}

	store := unit.NewStore()
	if err := loadCurriculumDomain(store); err != nil {
		return Result{}, err
	}
	installVocabulary(store)
	meterToken, err := newMeterToken()
	if err != nil {
		return Result{}, err
	}
	if err := dsl.RegisterCausalCurriculumMeter(meterToken); err != nil {
		return Result{}, err
	}
	defer dsl.UnregisterCausalCurriculumMeter(meterToken)
	runtime := newRuntime(profile, centralProfileBytes, meterToken, store)
	if err := seedArtifacts(store, runtime, profile, seeds, certificates, episodeBytes, certificateBytes); err != nil {
		return Result{}, err
	}

	ag := agenda.New()
	ag.Push(&agenda.Task{Priority: 900, UnitName: runtimeName, SlotName: "ccInitialize", Reasons: []string{"initialize exact central curriculum"}})
	eng := engine.New(store, ag)
	eng.MaxCycles = 1
	eng.Verbosity = 0
	eng.Out = io.Discard
	eng.MutConfig.Enabled = false

	items := make([]causalv2.TaskMeterItem, 0, taskCount)
	for ag.Len() != 0 {
		task := ag.Peek()
		if task == nil {
			return Result{}, errors.New("curriculum agenda lost its next task")
		}
		before, err := dsl.CausalCurriculumMeterSnapshot(meterToken)
		if err != nil {
			return Result{}, err
		}
		if err := eng.Run(ctx); err != nil {
			return Result{}, err
		}
		if failure := runtime.GetString("failure"); failure != "" {
			return Result{}, errors.New(failure)
		}
		after, err := dsl.CausalCurriculumMeterSnapshot(meterToken)
		if err != nil {
			return Result{}, err
		}
		delta, err := subtractCounter(after, before)
		if err != nil {
			return Result{}, err
		}
		item := causalv2.TaskMeterItem{Name: "curriculum", Subject: fmt.Sprintf("%06d:%s", len(items)+1, taskKind(task.SlotName)), Counts: delta.Counts()}
		if err := item.Validate(); err != nil {
			return Result{}, err
		}
		items = append(items, item)
		if len(items) > taskCount {
			return Result{}, errors.New("curriculum exceeded 525 tasks")
		}
	}
	if len(items) != taskCount || eng.TaskNum != taskCount || runtime.GetString("phase") != "terminal" {
		return Result{}, fmt.Errorf("curriculum stopped at phase %q after %d tasks", runtime.GetString("phase"), len(items))
	}

	result, err := collectResult(store, runtime, profile, certificates, items, meterToken)
	if err != nil {
		return Result{}, err
	}
	if verify {
		if err := verifyResult(centralProfileBytes, episodeBytes, certificateBytes, result); err != nil {
			return Result{}, fmt.Errorf("verify emitted curriculum: %w", err)
		}
	}
	return result, nil
}

func loadCurriculumDomain(store *unit.Store) error {
	defs, err := cueload.LoadFS(curriculumFS)
	if err != nil {
		return err
	}
	for _, def := range defs {
		u := unit.New(def.Name)
		u.SetWorth(def.Worth)
		u.Set("isA", def.IsA)
		for slot, value := range def.Slots {
			u.Set(slot, value)
		}
		store.Put(u)
	}
	return nil
}

func installVocabulary(store *unit.Store) {
	vocabulary := unit.New("Vocabulary")
	vocabulary.Set("isA", []string{"Anything"})
	store.Put(vocabulary)
	marker := unit.New("CausalCurriculumVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "causalcurriculum")
	store.Put(marker)
}

func newRuntime(profile causalv2.CentralProfile, profileBytes []byte, meterToken string, store *unit.Store) *unit.Unit {
	runtime := unit.New(runtimeName)
	runtime.Set("isA", []string{"CausalCurriculumCursor", "Anything"})
	runtime.Set("phase", "initializing")
	runtime.Set("centralProfileDigest", profile.ProfileDigest)
	runtime.Set("trainingKey", profile.TrainingKey)
	runtime.Set("creditEnabled", profile.CreditEnabled)
	runtime.Set("meterToken", meterToken)
	runtime.Set("centralProfileBytes", string(profileBytes))
	runtime.Set("nextChargeIndex", 481)
	runtime.Set("lastTranscriptDigest", causalv2.ZeroDigest)
	store.Put(runtime)
	return runtime
}

func seedArtifacts(store *unit.Store, runtime *unit.Unit, profile causalv2.CentralProfile, seeds []int64, certificates []causalv2.ApplicationCertificate, episodeBytes, certificateBytes [][]byte) error {
	descriptor := causalv2.CentralDescriptorPayload{CentralProfileDigest: profile.ProfileDigest, ExpectedRules: ruleCount, ExpectedSeeds: seeds, ExpectedCertificates: certificateCount, CreditEnabled: profile.CreditEnabled}
	descriptorUnit, err := putSeedArtifact(store, runtime, profile, 0, "central-descriptor", "CausalCentralDescriptorArtifact", descriptor)
	if err != nil {
		return err
	}
	runtime.Set("descriptorUnit", descriptorUnit.Name)
	units := make([]string, certificateCount)
	for index, certificate := range certificates {
		payload := causalv2.CertificatePayload{CertificateBytes: base64.RawURLEncoding.EncodeToString(certificateBytes[index]), CertificateDigest: certificate.CertificateDigest}
		u, err := putSeedArtifact(store, runtime, profile, index+1, "certificate", "CausalCentralCertificateArtifact", payload)
		if err != nil {
			return err
		}
		u.Set("matrixIndex", index)
		u.Set("episodeBytes", string(episodeBytes[index]))
		units[index] = u.Name
	}
	runtime.Set("certificateUnits", units)
	return nil
}

func putSeedArtifact(store *unit.Store, runtime *unit.Unit, profile causalv2.CentralProfile, chargeIndex int, kind, category string, payload any) (*unit.Unit, error) {
	artifact, err := causalv2.NewArtifact(profile.ProfileDigest, profile.TrainingKey, 0, kind, payload, chargeIndex)
	if err != nil {
		return nil, err
	}
	encoded, err := causalv2.CanonicalJSON(artifact)
	if err != nil {
		return nil, err
	}
	if store.Has(artifact.Name()) {
		return nil, fmt.Errorf("occupied seeded artifact name %q", artifact.Name())
	}
	u := unit.New(artifact.Name())
	u.Set("isA", []string{category, "CausalCurriculumArtifact", "CausalArtifact", "Anything"})
	u.Set("sealed", true)
	u.Set("artifactBytes", string(encoded))
	u.Set("artifactDigest", artifact.ArtifactDigest)
	u.Set("kind", artifact.Kind)
	u.Set("chargeIndex", artifact.ChargeIndex)
	u.Set("runtime", runtime.Name)
	store.Put(u)
	return u, nil
}

func subtractCounter(after, before causalv2.Counter) (causalv2.Counter, error) {
	a, b := after.Counts(), before.Counts()
	var values [15]int64
	for index := range values {
		values[index] = a[index] - b[index]
		if values[index] < 0 {
			return causalv2.Counter{}, errors.New("curriculum counter decreased")
		}
	}
	result := causalv2.CounterFromCounts(values)
	return result, result.Validate()
}

func taskKind(slot string) string {
	return map[string]string{
		"ccInitialize": "initialization", "ccAdmit": "admission", "ccMatrix": "matrix-barrier",
		"ccAggregate": "aggregate", "ccAggregates": "aggregate-barrier", "ccSelect": "selection-barrier", "ccTranscript": "transcript-barrier",
	}[slot]
}

func collectResult(store *unit.Store, runtime *unit.Unit, profile causalv2.CentralProfile, certificates []causalv2.ApplicationCertificate, items []causalv2.TaskMeterItem, meterToken string) (Result, error) {
	names := store.Examples("CausalCurriculumArtifact")
	sort.Slice(names, func(i, j int) bool {
		return store.Get(names[i]).GetInt("chargeIndex") < store.Get(names[j]).GetInt("chargeIndex")
	})
	artifactBytes := make([][]byte, len(names))
	for index, name := range names {
		u := store.Get(name)
		if u.GetInt("chargeIndex") != index {
			return Result{}, errors.New("non-contiguous central artifact charge order")
		}
		artifactBytes[index] = []byte(u.GetString("artifactBytes"))
	}
	aggregates := make([]causalv2.RuleAggregatePayload, 0, ruleCount)
	for _, name := range runtime.GetStrings("aggregateUnits") {
		artifact, err := causalv2.VerifyArtifact([]byte(store.Get(name).GetString("artifactBytes")))
		if err != nil {
			return Result{}, err
		}
		payload, err := causalv2.StrictDecode[causalv2.RuleAggregatePayload](artifact.Payload)
		if err != nil {
			return Result{}, err
		}
		aggregates = append(aggregates, payload)
	}
	ties := make([]string, 0, len(runtime.GetStrings("tieUnits")))
	for _, name := range runtime.GetStrings("tieUnits") {
		ties = append(ties, store.Get(name).GetString("ruleCode"))
	}
	transcript := make([]causalv2.CentralTranscriptEvent, 0, len(runtime.GetStrings("transcriptUnits")))
	for _, name := range runtime.GetStrings("transcriptUnits") {
		artifact, err := causalv2.VerifyArtifact([]byte(store.Get(name).GetString("artifactBytes")))
		if err != nil {
			return Result{}, err
		}
		event, err := causalv2.StrictDecode[causalv2.CentralTranscriptEvent](artifact.Payload)
		if err != nil {
			return Result{}, err
		}
		transcript = append(transcript, event)
	}
	taskDigest, err := causalv2.TaskMeterItemsDigest(items)
	if err != nil {
		return Result{}, err
	}
	counts, err := dsl.CausalCurriculumMeterSnapshot(meterToken)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ProfileDigest: profile.ProfileDigest, TrainingKey: profile.TrainingKey, CreditEnabled: profile.CreditEnabled,
		Applications: certificates, Aggregates: aggregates, WinnerTies: ties, SelectedRule: runtime.GetString("selectedRule"), Unresolved: runtime.GetBool("unresolved"),
		ArtifactBytes: artifactBytes, Transcript: transcript, Counts: counts, TaskMeterItems: items, TaskMeterItemsDigest: taskDigest,
		TerminalTranscriptDigest: runtime.GetString("lastTranscriptDigest"),
	}, nil
}

func newMeterToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("create curriculum meter capability: %w", err)
	}
	return hex.EncodeToString(token[:]), nil
}

func sortedRuleCodes() []string {
	rules := causal.Rules()
	codes := make([]string, len(rules))
	for index, rule := range rules {
		codes[index] = rule.Code()
	}
	sort.Strings(codes)
	return codes
}

func trainingSeeds(manifest causalv2.Manifest) []int64 {
	seeds := make([]int64, manifest.TrainingSeeds.Count)
	for index := range seeds {
		seeds[index] = manifest.TrainingSeeds.Start + int64(index)*manifest.TrainingSeeds.Step
	}
	return seeds
}
