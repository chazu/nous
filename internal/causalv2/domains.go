package causalv2

const (
	HypothesisSetDomain          = "causal-hypothesis-set/v2"
	PartitionDomain              = "causal-partition/v2"
	TranscriptEntryDomain        = "causal-transcript-entry/v2"
	EmptyTranscriptDomain        = "causal-empty-transcript/v2"
	ApplicationCertificateDomain = "causal-application-certificate/v2"
	RuleApplicationsDomain       = "causal-rule-applications/v2"
	TrainingEpisodeDomain        = "causal-training-episode/v2"
	TrainingEpisodeBundleDomain  = "causal-training-episode-bundle/v2"
	TrainingDigestInputDomain    = "causal-training-digest-input/v2"
	CentralTranscriptEventDomain = "causal-central-transcript-event/v2"
	DiagnosisReportDomain        = "causal-diagnosis-report/v2"
)

var V2DigestDomains = []string{
	ProfileDomain, CentralProfileDomain, PublicTokenDomain, PublicFixtureDomain, PrivateFixtureDomain,
	ArtifactDomain, AuthorizationDomain, HypothesisSetDomain, PartitionDomain, TranscriptEntryDomain,
	EmptyTranscriptDomain, ApplicationCertificateDomain, RuleApplicationsDomain, TrainingEpisodeDomain,
	TrainingEpisodeBundleDomain, ControlCertificateDomain, ControlBundleDomain, ControlEvidenceDomain, TrainingDigestInputDomain,
	SemanticKeyDomain, MeterArrayDomain, TaskMeterItemsDomain, CentralTranscriptEventDomain, DiagnosisReportDomain,
}
