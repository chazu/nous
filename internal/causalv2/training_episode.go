package causalv2

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

// TrainingEpisodeEvidence is the canonical episode record that an application
// certificate summarizes. Keeping this schema at the proof boundary lets a
// consumer bind a certificate to the exact evidence bytes without importing
// the experiment package that produced them.
type TrainingEpisodeEvidence struct {
	EpisodeReportVersion  string      `json:"episode_report_version"`
	Seed                  int64       `json:"seed"`
	ProfileDigest         string      `json:"profile_digest"`
	FixtureDigest         string      `json:"fixture_digest"`
	RuleCode              string      `json:"rule_code"`
	Actions               []string    `json:"actions"`
	TeacherOutcomes       []string    `json:"teacher_outcomes"`
	Terminal              string      `json:"terminal"`
	Score                 int         `json:"score"`
	Cost                  int         `json:"cost"`
	FinalPosterior        []string    `json:"final_posterior"`
	PosteriorDigest       string      `json:"posterior_digest"`
	TranscriptDigest      string      `json:"transcript_digest"`
	HypothesisEvaluations int         `json:"hypothesis_evaluations"`
	SemanticWork          int         `json:"semantic_work"`
	AttributedUnits       int         `json:"attributed_units"`
	EngineCycles          int         `json:"engine_cycles"`
	OracleAgreements      int         `json:"oracle_agreements"`
	OracleDisagreements   int         `json:"oracle_disagreements"`
	MeterItems            []MeterItem `json:"meter_items"`
	AllCapsValid          bool        `json:"all_caps_valid"`
	EpisodeReportDigest   string      `json:"episode_report_digest"`
}

func validateTrainingEpisode(e TrainingEpisodeEvidence) error {
	if e.EpisodeReportVersion != TrainingEpisodeDomain {
		return errors.New("invalid training episode version")
	}
	for field, digest := range map[string]string{
		"profile_digest": e.ProfileDigest, "fixture_digest": e.FixtureDigest,
		"posterior_digest": e.PosteriorDigest, "transcript_digest": e.TranscriptDigest,
	} {
		if err := requireDigest(field, digest, false); err != nil {
			return err
		}
	}
	if _, err := causal.ParseRule(e.RuleCode); err != nil {
		return err
	}
	if !slices.Contains([]string{"identified", "equivalence", "budget-exhausted"}, e.Terminal) {
		return errors.New("invalid training episode terminal")
	}
	m := PreregisteredManifest()
	if e.Score < 0 || e.Score > m.InvalidOrExhaustedScore || e.Cost < 0 || e.Cost > m.EpisodeCostCeiling {
		return errors.New("training episode score or cost outside manifest bounds")
	}
	if len(e.Actions) != len(e.TeacherOutcomes) {
		return errors.New("training episode action and outcome cardinalities differ")
	}
	if e.HypothesisEvaluations < 0 || e.SemanticWork < 0 || e.AttributedUnits < 0 || e.EngineCycles < 0 || e.OracleAgreements < 0 || e.OracleDisagreements < 0 {
		return errors.New("negative training episode count")
	}
	if err := ValidateMeterItems(e.MeterItems); err != nil {
		return err
	}
	production := e.MeterItems[0]
	if production.Name != "production" || !production.Active {
		return errors.New("training episode production meter is inactive")
	}
	if !e.MeterItems[1].Active || !e.MeterItems[4].Active {
		return errors.New("required training episode meter is inactive")
	}
	if e.MeterItems[5].Active != (e.RuleCode == "P=C;M=gain;S=E") {
		return errors.New("training episode DP ownership is not lexical-first")
	}
	counts := production.Counter()
	if e.HypothesisEvaluations != int(counts.SCMEvaluations) || e.SemanticWork != int(counts.TotalWork) || e.AttributedUnits != int(counts.AttributedUnits) || e.EngineCycles != int(counts.EngineCycles) {
		return errors.New("training episode summary differs from production meter")
	}
	capsValid := true
	for _, item := range e.MeterItems {
		if !item.Active {
			continue
		}
		meter, err := NewAggregateMeter(item.Name, "training", []Counter{item.Counter()})
		if err != nil {
			return err
		}
		capsValid = capsValid && meter.Valid
	}
	if e.AllCapsValid != capsValid {
		return errors.New("training episode all_caps_valid differs from meter reconstruction")
	}
	return nil
}

func trainingEpisodeDigest(e TrainingEpisodeEvidence) (string, error) {
	e.EpisodeReportDigest = ""
	return Digest(TrainingEpisodeDomain, e)
}

func SignTrainingEpisodeEvidence(e *TrainingEpisodeEvidence) error {
	if e == nil {
		return errors.New("nil training episode evidence")
	}
	if err := validateTrainingEpisode(*e); err != nil {
		return err
	}
	digest, err := trainingEpisodeDigest(*e)
	if err != nil {
		return err
	}
	e.EpisodeReportDigest = digest
	encoded, err := CanonicalJSON(*e)
	if err != nil {
		return err
	}
	return verifyTrainingEpisodeCaps(encoded, *e)
}

func verifyTrainingEpisodeCaps(encoded []byte, e TrainingEpisodeEvidence) error {
	base := e
	base.MeterItems = []MeterItem{}
	baseBytes, err := CanonicalJSON(base)
	if err != nil {
		return err
	}
	meterBytes, err := CanonicalJSON(e.MeterItems)
	if err != nil {
		return err
	}
	m := PreregisteredManifest()
	if len(baseBytes) > m.TrainingEpisodeBaseByteCap || len(meterBytes)-2 > m.EpisodeMeterItemsByteCap || len(encoded) > m.TrainingEpisodeReportByteCap {
		return errors.New("training episode encoding cap exceeded")
	}
	return nil
}

func VerifyTrainingEpisodeEvidence(data []byte) (TrainingEpisodeEvidence, error) {
	e, err := StrictDecode[TrainingEpisodeEvidence](data)
	if err != nil {
		return e, err
	}
	if err := validateTrainingEpisode(e); err != nil {
		return e, err
	}
	if err := requireDigest("episode_report_digest", e.EpisodeReportDigest, false); err != nil {
		return e, err
	}
	want, err := trainingEpisodeDigest(e)
	if err != nil {
		return e, err
	}
	if e.EpisodeReportDigest != want {
		return e, errors.New("training episode digest mismatch")
	}
	canonical, err := CanonicalJSON(e)
	if err != nil {
		return e, err
	}
	if !bytes.Equal(data, canonical) {
		return e, errors.New("training episode bytes are not canonical")
	}
	if err := verifyTrainingEpisodeCaps(data, e); err != nil {
		return e, err
	}
	return e, nil
}

// VerifyApplicationCertificateForEpisode admits a certificate only when every
// summary field is derived from the exact canonical episode evidence bytes.
func VerifyApplicationCertificateForEpisode(certificateBytes, episodeBytes []byte) (ApplicationCertificate, TrainingEpisodeEvidence, error) {
	certificate, err := VerifyApplicationCertificate(certificateBytes)
	if err != nil {
		return certificate, TrainingEpisodeEvidence{}, err
	}
	episode, err := VerifyTrainingEpisodeEvidence(episodeBytes)
	if err != nil {
		return certificate, episode, err
	}
	if certificate.Seed != episode.Seed || certificate.ProfileDigest != episode.ProfileDigest || certificate.FixtureDigest != episode.FixtureDigest || certificate.RuleCode != episode.RuleCode || certificate.Score != episode.Score || certificate.Terminal != episode.Terminal || certificate.Cost != episode.Cost || certificate.PosteriorDigest != episode.PosteriorDigest || certificate.TranscriptDigest != episode.TranscriptDigest || certificate.OracleAgreements != episode.OracleAgreements || certificate.OracleDisagreements != episode.OracleDisagreements || certificate.AllCapsValid != episode.AllCapsValid || certificate.EpisodeReportDigest != episode.EpisodeReportDigest {
		return certificate, episode, fmt.Errorf("application certificate is not derived from matching episode %s", episode.EpisodeReportDigest)
	}
	return certificate, episode, nil
}
