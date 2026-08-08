package causalv2

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ArtifactDomain      = "causal-artifact/v2"
	SemanticKeyDomain   = "causal-semantic-key/v2"
	AuthorizationDomain = "causal-authorization/v2"
)

var ArtifactKinds = []string{
	"descriptor-snapshot", "observation", "posterior", "cache", "partition", "proposal", "score", "tie", "selection", "authorization", "result", "elimination", "consumption", "transcript", "terminal",
	"central-descriptor", "central-rule", "certificate", "application", "credit", "aggregate", "central-tie", "central-selection",
}

type PartitionCell struct {
	Outcome    string   `json:"outcome"`
	Hypotheses []string `json:"hypotheses"`
}

type DescriptorSnapshotPayload struct {
	PreviousSnapshotArtifactDigest string   `json:"previous_snapshot_artifact_digest"`
	State                          string   `json:"state"`
	Aliases                        []string `json:"aliases"`
	Costs                          []int    `json:"costs"`
	Presentation                   []int    `json:"presentation"`
	InitialPosteriorArtifactDigest string   `json:"initial_posterior_artifact_digest"`
	PosteriorDigest                string   `json:"posterior_digest"`
	AcquisitionCode                string   `json:"acquisition_code"`
	TotalCost                      int      `json:"total_cost"`
	ActionCount                    int      `json:"action_count"`
	RemainingEvaluations           int64    `json:"remaining_evaluations"`
	RemainingWork                  int64    `json:"remaining_work"`
	RemainingCycles                int64    `json:"remaining_cycles"`
	RemainingUnits                 int64    `json:"remaining_units"`
}

type ObservationPayload struct {
	Outcome string `json:"outcome"`
}
type PosteriorPayload struct {
	Hypotheses        []string `json:"hypotheses"`
	SemanticSetDigest string   `json:"semantic_set_digest"`
}
type CachePayload struct {
	Action                  string          `json:"action"`
	PosteriorArtifactDigest string          `json:"posterior_artifact_digest"`
	Cells                   []PartitionCell `json:"cells"`
	E                       int             `json:"E"`
	W                       int             `json:"W"`
	H                       string          `json:"H"`
	C                       int             `json:"C"`
	R                       int             `json:"R"`
}
type PartitionPayload struct {
	Action                  string          `json:"action"`
	PosteriorArtifactDigest string          `json:"posterior_artifact_digest"`
	Cells                   []PartitionCell `json:"cells"`
}
type ProposalPayload struct {
	Action              string `json:"action"`
	CacheArtifactDigest string `json:"cache_artifact_digest"`
}

type ScorePayload struct {
	Action              string `json:"action"`
	RuleCode            string `json:"rule_code"`
	CacheArtifactDigest string `json:"cache_artifact_digest"`
}
type TiePayload struct {
	Action              string `json:"action"`
	ScoreArtifactDigest string `json:"score_artifact_digest"`
}

type SelectionPayload struct {
	Action             string   `json:"action"`
	TieArtifactDigests []string `json:"tie_artifact_digests"`
}
type ResultPayload struct {
	AuthorizationArtifactDigest string `json:"authorization_artifact_digest"`
	Action                      string `json:"action"`
	Outcome                     string `json:"outcome"`
}
type EliminationPayload struct {
	Hypothesis           string `json:"hypothesis"`
	ResultArtifactDigest string `json:"result_artifact_digest"`
}
type ConsumptionPayload struct {
	ResultArtifactDigest    string `json:"result_artifact_digest"`
	PosteriorArtifactDigest string `json:"posterior_artifact_digest"`
}
type TerminalPayload struct {
	Terminal         string `json:"terminal"`
	PosteriorDigest  string `json:"posterior_digest"`
	TotalCost        int    `json:"total_cost"`
	ActionCount      int    `json:"action_count"`
	TranscriptDigest string `json:"transcript_digest"`
}
type CentralDescriptorPayload struct {
	CentralProfileDigest string  `json:"central_profile_digest"`
	ExpectedRules        int     `json:"expected_rules"`
	ExpectedSeeds        []int64 `json:"expected_seeds"`
	ExpectedCertificates int     `json:"expected_certificates"`
	CreditEnabled        bool    `json:"credit_enabled"`
}
type CentralRulePayload struct {
	RuleCode string `json:"rule_code"`
}
type CertificatePayload struct {
	CertificateBytes  string `json:"certificate_bytes"`
	CertificateDigest string `json:"certificate_digest"`
}
type ApplicationPayload struct {
	Seed              int64  `json:"seed"`
	RuleCode          string `json:"rule_code"`
	CertificateDigest string `json:"certificate_digest"`
	Score             int    `json:"score"`
	Terminal          string `json:"terminal"`
	Cost              int    `json:"cost"`
}
type CreditPayload struct {
	ApplicationArtifactDigest string `json:"application_artifact_digest"`
	Delta                     int    `json:"delta"`
}
type RuleAggregatePayload struct {
	Code              string `json:"code"`
	Applications      int    `json:"applications"`
	TotalScore        int    `json:"total_score"`
	TotalCost         int    `json:"total_cost"`
	Identified        int    `json:"identified"`
	Equivalence       int    `json:"equivalence"`
	BudgetExhausted   int    `json:"budget_exhausted"`
	Worth             int    `json:"worth"`
	ApplicationDigest string `json:"application_digest"`
}
type CentralTiePayload struct {
	RuleCode                string `json:"rule_code"`
	AggregateArtifactDigest string `json:"aggregate_artifact_digest"`
}
type CentralSelectionPayload struct {
	SelectedRule       string   `json:"selected_rule"`
	TieArtifactDigests []string `json:"tie_artifact_digests"`
}

type TranscriptEntry struct {
	TranscriptVersion              string `json:"transcript_version"`
	Episode                        string `json:"episode"`
	Step                           int    `json:"step"`
	PreviousDigest                 string `json:"previous_digest"`
	RuleCode                       string `json:"rule_code"`
	Action                         string `json:"action"`
	PosteriorBeforeDigest          string `json:"posterior_before_digest"`
	PartitionDigest                string `json:"partition_digest"`
	TeacherOutcome                 string `json:"teacher_outcome"`
	PosteriorAfterDigest           string `json:"posterior_after_digest"`
	EliminatedDigest               string `json:"eliminated_digest"`
	CostBefore                     int    `json:"cost_before"`
	CostAfter                      int    `json:"cost_after"`
	ActionCount                    int    `json:"action_count"`
	CacheStatus                    string `json:"cache_status"`
	AttributedUnitPrefix           int    `json:"attributed_unit_prefix"`
	RemainingHypothesisEvaluations int64  `json:"remaining_hypothesis_evaluations"`
	RemainingSemanticWork          int64  `json:"remaining_semantic_work"`
	RemainingEngineCycles          int64  `json:"remaining_engine_cycles"`
	RemainingAttributedUnits       int64  `json:"remaining_attributed_units"`
	TranscriptDigest               string `json:"transcript_digest"`
}

type CentralTranscriptEvent struct {
	EventVersion          string `json:"event_version"`
	Index                 int    `json:"index"`
	PreviousDigest        string `json:"previous_digest"`
	Kind                  string `json:"kind"`
	SubjectArtifactDigest string `json:"subject_artifact_digest"`
	WorkBefore            int64  `json:"work_before"`
	WorkAfter             int64  `json:"work_after"`
	EventDigest           string `json:"event_digest"`
}

type Authorization struct {
	ProfileDigest           string `json:"profile_digest"`
	Episode                 string `json:"episode"`
	Step                    int    `json:"step"`
	Action                  string `json:"action"`
	SelectionArtifactDigest string `json:"selection_artifact_digest"`
	OpaqueToken             string `json:"opaque_token"`
	AuthorizationDigest     string `json:"authorization_digest"`
}

func authorizationDigest(authorization Authorization) (string, error) {
	authorization.AuthorizationDigest = ""
	return Digest(AuthorizationDomain, authorization)
}
func validateAuthorization(authorization Authorization) error {
	if err := requireDigest("profile_digest", authorization.ProfileDigest, false); err != nil {
		return err
	}
	if authorization.Episode == "" || authorization.Step < 0 {
		return errors.New("invalid authorization episode or step")
	}
	if _, err := causal.ParseAction(authorization.Action); err != nil {
		return err
	}
	if err := requireDigest("selection_artifact_digest", authorization.SelectionArtifactDigest, false); err != nil {
		return err
	}
	return requireDigest("opaque_token", authorization.OpaqueToken, false)
}
func SignAuthorization(authorization *Authorization) error {
	if authorization == nil {
		return errors.New("nil authorization")
	}
	if err := validateAuthorization(*authorization); err != nil {
		return err
	}
	digest, err := authorizationDigest(*authorization)
	if err == nil {
		authorization.AuthorizationDigest = digest
	}
	return err
}
func VerifyAuthorization(data []byte) (Authorization, error) {
	a, err := StrictDecode[Authorization](data)
	if err != nil {
		return a, err
	}
	if err = validateAuthorization(a); err != nil {
		return a, err
	}
	if err = requireDigest("authorization_digest", a.AuthorizationDigest, false); err != nil {
		return a, err
	}
	want, err := authorizationDigest(a)
	if err != nil {
		return a, err
	}
	if a.AuthorizationDigest != want {
		return a, errors.New("authorization digest mismatch")
	}
	return a, nil
}

type Artifact struct {
	ArtifactVersion string          `json:"artifact_version"`
	ProfileDigest   string          `json:"profile_digest"`
	Scope           string          `json:"scope"`
	Step            int             `json:"step"`
	Kind            string          `json:"kind"`
	SemanticKey     string          `json:"semantic_key"`
	Payload         json.RawMessage `json:"payload"`
	ChargeIndex     int             `json:"charge_index"`
	ArtifactDigest  string          `json:"artifact_digest"`
}

type semanticKeyInput struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func SemanticKey(kind string, payload any) (string, error) {
	raw, err := CanonicalJSON(payload)
	if err != nil {
		return "", err
	}
	return semanticKeyRaw(kind, raw)
}
func semanticKeyRaw(kind string, raw json.RawMessage) (string, error) {
	return Digest(SemanticKeyDomain, semanticKeyInput{kind, raw})
}
func artifactDigest(artifact Artifact) (string, error) {
	artifact.ArtifactDigest = ""
	return Digest(ArtifactDomain, artifact)
}
func (artifact Artifact) Name() string {
	return "Causal." + artifact.Kind + "." + artifact.ArtifactDigest
}

func NewArtifact(profileDigest, scope string, step int, kind string, payload any, chargeIndex int) (Artifact, error) {
	raw, err := CanonicalJSON(payload)
	if err != nil {
		return Artifact{}, err
	}
	key, err := semanticKeyRaw(kind, raw)
	if err != nil {
		return Artifact{}, err
	}
	a := Artifact{ArtifactVersion: ArtifactDomain, ProfileDigest: profileDigest, Scope: scope, Step: step, Kind: kind, SemanticKey: key, Payload: raw, ChargeIndex: chargeIndex}
	if err = validateArtifact(a, false); err != nil {
		return Artifact{}, err
	}
	a.ArtifactDigest, err = artifactDigest(a)
	return a, err
}

func validateArtifact(artifact Artifact, requireOuterDigest bool) error {
	if artifact.ArtifactVersion != ArtifactDomain {
		return errors.New("invalid artifact version")
	}
	if err := requireDigest("profile_digest", artifact.ProfileDigest, false); err != nil {
		return err
	}
	if artifact.Scope == "" || artifact.Step < 0 || artifact.ChargeIndex < 0 {
		return errors.New("invalid artifact scope, step, or charge index")
	}
	if !slices.Contains(ArtifactKinds, artifact.Kind) {
		return fmt.Errorf("invalid artifact kind %q", artifact.Kind)
	}
	if err := validatePayload(artifact.Kind, artifact.Payload); err != nil {
		return fmt.Errorf("%s payload: %w", artifact.Kind, err)
	}
	if artifact.Kind == "authorization" {
		authorization, err := StrictDecode[Authorization](artifact.Payload)
		if err != nil {
			return err
		}
		if authorization.ProfileDigest != artifact.ProfileDigest || authorization.Episode != artifact.Scope || authorization.Step != artifact.Step {
			return errors.New("authorization payload is not bound to artifact envelope")
		}
	}
	want, err := semanticKeyRaw(artifact.Kind, artifact.Payload)
	if err != nil {
		return err
	}
	if artifact.SemanticKey != want {
		return errors.New("artifact semantic key mismatch")
	}
	if requireOuterDigest {
		return requireDigest("artifact_digest", artifact.ArtifactDigest, false)
	}
	return nil
}

func VerifyArtifact(data []byte) (Artifact, error) {
	artifact, err := StrictDecode[Artifact](data)
	if err != nil {
		return artifact, err
	}
	if err = validateArtifact(artifact, true); err != nil {
		return artifact, err
	}
	want, err := artifactDigest(artifact)
	if err != nil {
		return artifact, err
	}
	if artifact.ArtifactDigest != want {
		return artifact, errors.New("artifact digest mismatch")
	}
	return artifact, nil
}

func decodePayload[T any](raw json.RawMessage) (T, error) { return StrictDecode[T](raw) }

func validatePayload(kind string, raw json.RawMessage) error {
	switch kind {
	case "descriptor-snapshot":
		p, e := decodePayload[DescriptorSnapshotPayload](raw)
		if e != nil {
			return e
		}
		if !slices.Contains([]string{"ready", "awaiting-teacher", "terminal"}, p.State) {
			return errors.New("invalid snapshot state")
		}
		return nil
	case "observation":
		p, e := decodePayload[ObservationPayload](raw)
		if e != nil {
			return e
		}
		return validateOutcome(p.Outcome)
	case "posterior":
		p, e := decodePayload[PosteriorPayload](raw)
		if e != nil {
			return e
		}
		if len(p.Hypotheses) == 0 || !sort.StringsAreSorted(p.Hypotheses) {
			return errors.New("posterior not sorted")
		}
		return requireDigest("semantic_set_digest", p.SemanticSetDigest, false)
	case "cache":
		p, e := decodePayload[CachePayload](raw)
		if e != nil {
			return e
		}
		if e = requireDigest("posterior_artifact_digest", p.PosteriorArtifactDigest, false); e != nil {
			return e
		}
		if p.R != 0 && p.R != 1 {
			return errors.New("invalid cache repeat feature")
		}
		return validateActionAndCells(p.Action, p.Cells)
	case "partition":
		p, e := decodePayload[PartitionPayload](raw)
		if e != nil {
			return e
		}
		return validateActionAndCells(p.Action, p.Cells)
	case "proposal":
		p, e := decodePayload[ProposalPayload](raw)
		if e != nil {
			return e
		}
		return validateActionDigest(p.Action, "cache_artifact_digest", p.CacheArtifactDigest)
	case "score":
		p, e := decodePayload[ScorePayload](raw)
		if e != nil {
			return e
		}
		if !validAcquisition(p.RuleCode) {
			return errors.New("invalid score rule")
		}
		return validateActionDigest(p.Action, "cache_artifact_digest", p.CacheArtifactDigest)
	case "tie":
		p, e := decodePayload[TiePayload](raw)
		if e != nil {
			return e
		}
		return validateActionDigest(p.Action, "score_artifact_digest", p.ScoreArtifactDigest)
	case "selection":
		p, e := decodePayload[SelectionPayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.ParseAction(p.Action); e != nil {
			return e
		}
		return validateDigests(p.TieArtifactDigests)
	case "authorization":
		_, e := VerifyAuthorization(raw)
		return e
	case "result":
		p, e := decodePayload[ResultPayload](raw)
		if e != nil {
			return e
		}
		if e = validateActionDigest(p.Action, "authorization_artifact_digest", p.AuthorizationArtifactDigest); e != nil {
			return e
		}
		return validateOutcome(p.Outcome)
	case "elimination":
		p, e := decodePayload[EliminationPayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.Parse(p.Hypothesis); e != nil {
			return e
		}
		return requireDigest("result_artifact_digest", p.ResultArtifactDigest, false)
	case "consumption":
		p, e := decodePayload[ConsumptionPayload](raw)
		if e != nil {
			return e
		}
		if e = requireDigest("result_artifact_digest", p.ResultArtifactDigest, false); e != nil {
			return e
		}
		return requireDigest("posterior_artifact_digest", p.PosteriorArtifactDigest, false)
	case "terminal":
		p, e := decodePayload[TerminalPayload](raw)
		if e != nil {
			return e
		}
		if !slices.Contains([]string{"identified", "equivalence", "budget-exhausted"}, p.Terminal) {
			return errors.New("invalid terminal")
		}
		if e = requireDigest("posterior_digest", p.PosteriorDigest, false); e != nil {
			return e
		}
		return requireDigest("transcript_digest", p.TranscriptDigest, false)
	case "central-descriptor":
		p, e := decodePayload[CentralDescriptorPayload](raw)
		if e != nil {
			return e
		}
		if p.ExpectedRules != 40 || len(p.ExpectedSeeds) != 12 || p.ExpectedCertificates != 480 {
			return errors.New("invalid central descriptor cardinality")
		}
		return requireDigest("central_profile_digest", p.CentralProfileDigest, false)
	case "central-rule":
		p, e := decodePayload[CentralRulePayload](raw)
		if e != nil {
			return e
		}
		_, e = causal.ParseRule(p.RuleCode)
		return e
	case "certificate":
		p, e := decodePayload[CertificatePayload](raw)
		if e != nil {
			return e
		}
		decoded, e := validateCertificateBytes(p.CertificateBytes)
		if e != nil {
			return e
		}
		certificate, e := VerifyApplicationCertificate(decoded)
		if e != nil {
			return e
		}
		if p.CertificateDigest != certificate.CertificateDigest {
			return errors.New("certificate payload digest does not match decoded certificate")
		}
		return nil
	case "application":
		p, e := decodePayload[ApplicationPayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.ParseRule(p.RuleCode); e != nil {
			return e
		}
		return requireDigest("certificate_digest", p.CertificateDigest, false)
	case "credit":
		p, e := decodePayload[CreditPayload](raw)
		if e != nil {
			return e
		}
		return requireDigest("application_artifact_digest", p.ApplicationArtifactDigest, false)
	case "aggregate":
		p, e := decodePayload[RuleAggregatePayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.ParseRule(p.Code); e != nil {
			return e
		}
		return requireDigest("application_digest", p.ApplicationDigest, false)
	case "central-tie":
		p, e := decodePayload[CentralTiePayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.ParseRule(p.RuleCode); e != nil {
			return e
		}
		return requireDigest("aggregate_artifact_digest", p.AggregateArtifactDigest, false)
	case "central-selection":
		p, e := decodePayload[CentralSelectionPayload](raw)
		if e != nil {
			return e
		}
		if _, e = causal.ParseRule(p.SelectedRule); e != nil {
			return e
		}
		return validateDigests(p.TieArtifactDigests)
	case "transcript":
		if event, e := decodePayload[CentralTranscriptEvent](raw); e == nil {
			if event.EventVersion != "causal-central-transcript/v2" || event.Index < 0 || !slices.Contains([]string{"admission", "aggregate", "selection"}, event.Kind) {
				return errors.New("invalid central transcript event")
			}
			if e = requireDigest("previous_digest", event.PreviousDigest, false); e != nil {
				return e
			}
			if e = requireDigest("subject_artifact_digest", event.SubjectArtifactDigest, false); e != nil {
				return e
			}
			if event.WorkBefore < 0 || event.WorkAfter < event.WorkBefore {
				return errors.New("invalid central transcript work interval")
			}
			if e = requireDigest("event_digest", event.EventDigest, false); e != nil {
				return e
			}
			preimage := event
			preimage.EventDigest = ""
			want, e := Digest(CentralTranscriptEventDomain, preimage)
			if e != nil {
				return e
			}
			if event.EventDigest != want {
				return errors.New("central transcript event digest mismatch")
			}
			return nil
		}
		entry, e := decodePayload[TranscriptEntry](raw)
		if e != nil {
			return e
		}
		if entry.TranscriptVersion != "causal-transcript/v2" || entry.Step < 0 {
			return errors.New("invalid episode transcript")
		}
		if _, e = causal.ParseAction(entry.Action); e != nil {
			return e
		}
		if e = validateOutcome(entry.TeacherOutcome); e != nil {
			return e
		}
		for field, digest := range map[string]string{"previous_digest": entry.PreviousDigest, "posterior_before_digest": entry.PosteriorBeforeDigest, "partition_digest": entry.PartitionDigest, "posterior_after_digest": entry.PosteriorAfterDigest, "eliminated_digest": entry.EliminatedDigest, "transcript_digest": entry.TranscriptDigest} {
			if e = requireDigest(field, digest, false); e != nil {
				return e
			}
		}
		preimage := entry
		preimage.TranscriptDigest = ""
		want, e := Digest(TranscriptEntryDomain, preimage)
		if e != nil {
			return e
		}
		if entry.TranscriptDigest != want {
			return errors.New("episode transcript digest mismatch")
		}
		return nil
	default:
		return errors.New("unsupported artifact kind")
	}
}

func validateActionAndCells(action string, cells []PartitionCell) error {
	if _, e := causal.ParseAction(action); e != nil {
		return e
	}
	if len(cells) == 0 {
		return errors.New("empty partition")
	}
	previous := ""
	for _, cell := range cells {
		if e := validateOutcome(cell.Outcome); e != nil {
			return e
		}
		if previous != "" && cell.Outcome <= previous {
			return errors.New("partition cells out of order")
		}
		previous = cell.Outcome
		if !sort.StringsAreSorted(cell.Hypotheses) {
			return errors.New("cell hypotheses not sorted")
		}
	}
	return nil
}
func validateActionDigest(action, field, digest string) error {
	if _, e := causal.ParseAction(action); e != nil {
		return e
	}
	return requireDigest(field, digest, false)
}
func validateDigests(values []string) error {
	if len(values) == 0 {
		return errors.New("empty digest array")
	}
	for _, value := range values {
		if e := requireDigest("artifact digest", value, false); e != nil {
			return e
		}
	}
	return nil
}
func validateCertificateBytes(encoded string) ([]byte, error) {
	if len(encoded) == 0 || slices.Contains([]byte(encoded), '=') {
		return nil, errors.New("invalid padded or empty certificate_bytes")
	}
	decoded, e := base64.RawURLEncoding.DecodeString(encoded)
	if e != nil {
		return nil, e
	}
	if base64.RawURLEncoding.EncodeToString(decoded) != encoded {
		return nil, errors.New("noncanonical certificate_bytes")
	}
	return decoded, nil
}
