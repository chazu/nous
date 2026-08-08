package causalrun

import causal "github.com/chazu/nous/internal/vocab/causal"

type observationPayload struct {
	Outcome string `json:"outcome"`
}

type posteriorPayload struct {
	Hypotheses        []string `json:"hypotheses"`
	SemanticSetDigest string   `json:"semantic_set_digest"`
}

type descriptorSnapshotPayload struct {
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
	RemainingEvaluations           int      `json:"remaining_evaluations"`
	RemainingWork                  int      `json:"remaining_work"`
	RemainingCycles                int      `json:"remaining_cycles"`
	RemainingUnits                 int      `json:"remaining_units"`
}

type cachePayload struct {
	Action                  string        `json:"action"`
	PosteriorArtifactDigest string        `json:"posterior_artifact_digest"`
	Cells                   []causal.Cell `json:"cells"`
	E                       int           `json:"E"`
	W                       int           `json:"W"`
	H                       string        `json:"H"`
	C                       int           `json:"C"`
	R                       int           `json:"R"`
}

type proposalPayload struct {
	Action              string `json:"action"`
	CacheArtifactDigest string `json:"cache_artifact_digest"`
}

type partitionPayload struct {
	Action                  string        `json:"action"`
	PosteriorArtifactDigest string        `json:"posterior_artifact_digest"`
	Cells                   []causal.Cell `json:"cells"`
}

type scorePayload struct {
	Action              string `json:"action"`
	RuleCode            string `json:"rule_code"`
	CacheArtifactDigest string `json:"cache_artifact_digest"`
}

type tiePayload struct {
	Action              string `json:"action"`
	ScoreArtifactDigest string `json:"score_artifact_digest"`
}

type selectionPayload struct {
	Action             string   `json:"action"`
	TieArtifactDigests []string `json:"tie_artifact_digests"`
}

type resultPayload struct {
	AuthorizationArtifactDigest string `json:"authorization_artifact_digest"`
	Action                      string `json:"action"`
	Outcome                     string `json:"outcome"`
}

type eliminationPayload struct {
	Hypothesis           string `json:"hypothesis"`
	ResultArtifactDigest string `json:"result_artifact_digest"`
}

type consumptionPayload struct {
	ResultArtifactDigest    string `json:"result_artifact_digest"`
	PosteriorArtifactDigest string `json:"posterior_artifact_digest"`
}

// TranscriptEntry retains the inherited exact transcript field order while
// using the v2 digest domains.
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
	RemainingHypothesisEvaluations int    `json:"remaining_hypothesis_evaluations"`
	RemainingSemanticWork          int    `json:"remaining_semantic_work"`
	RemainingEngineCycles          int    `json:"remaining_engine_cycles"`
	RemainingAttributedUnits       int    `json:"remaining_attributed_units"`
	TranscriptDigest               string `json:"transcript_digest"`
}

type terminalPayload struct {
	Terminal         string `json:"terminal"`
	PosteriorDigest  string `json:"posterior_digest"`
	TotalCost        int    `json:"total_cost"`
	ActionCount      int    `json:"action_count"`
	TranscriptDigest string `json:"transcript_digest"`
}

type artifactRef struct {
	Kind        string
	Digest      string
	SemanticKey string
	Canonical   []byte
	ChargeIndex int
	MeterAfter  Counts
}

type candidateArtifacts struct {
	candidate candidate
	cache     artifactRef
	proposal  artifactRef
	partition artifactRef
	score     artifactRef
	cacheHit  bool
}

type cachedCandidate struct {
	cells    []causal.Cell
	features causal.Features
}
