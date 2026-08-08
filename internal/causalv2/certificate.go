package causalv2

import (
	"errors"
	"fmt"
	"slices"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

// ApplicationCertificate is the exact inherited application object with v2
// digest semantics. It is the decoded content of certificate artifact payloads.
type ApplicationCertificate struct {
	Seed                int64  `json:"seed"`
	ProfileDigest       string `json:"profile_digest"`
	FixtureDigest       string `json:"fixture_digest"`
	RuleCode            string `json:"rule_code"`
	Score               int    `json:"score"`
	Terminal            string `json:"terminal"`
	Cost                int    `json:"cost"`
	PosteriorDigest     string `json:"posterior_digest"`
	TranscriptDigest    string `json:"transcript_digest"`
	OracleAgreements    int    `json:"oracle_agreements"`
	OracleDisagreements int    `json:"oracle_disagreements"`
	AllCapsValid        bool   `json:"all_caps_valid"`
	EpisodeReportDigest string `json:"episode_report_digest"`
	CertificateDigest   string `json:"certificate_digest"`
}

func validateApplicationCertificate(certificate ApplicationCertificate) error {
	for field, digest := range map[string]string{
		"profile_digest": certificate.ProfileDigest, "fixture_digest": certificate.FixtureDigest,
		"posterior_digest": certificate.PosteriorDigest, "transcript_digest": certificate.TranscriptDigest,
		"episode_report_digest": certificate.EpisodeReportDigest,
	} {
		if err := requireDigest(field, digest, false); err != nil {
			return err
		}
	}
	if _, err := causal.ParseRule(certificate.RuleCode); err != nil {
		return err
	}
	if !slices.Contains([]string{"identified", "equivalence", "budget-exhausted"}, certificate.Terminal) {
		return errors.New("invalid application terminal")
	}
	if certificate.Score < 0 || certificate.Score > PreregisteredManifest().InvalidOrExhaustedScore || certificate.Cost < 0 || certificate.Cost > PreregisteredManifest().EpisodeCostCeiling {
		return errors.New("application score or cost outside manifest bounds")
	}
	if certificate.OracleAgreements < 0 || certificate.OracleDisagreements < 0 {
		return errors.New("negative oracle count")
	}
	return nil
}

func applicationCertificateDigest(certificate ApplicationCertificate) (string, error) {
	certificate.CertificateDigest = ""
	return Digest(ApplicationCertificateDomain, certificate)
}

func SignApplicationCertificate(certificate *ApplicationCertificate) error {
	if certificate == nil {
		return errors.New("nil application certificate")
	}
	if err := validateApplicationCertificate(*certificate); err != nil {
		return err
	}
	digest, err := applicationCertificateDigest(*certificate)
	if err != nil {
		return err
	}
	certificate.CertificateDigest = digest
	encoded, err := CanonicalJSON(certificate)
	if err != nil {
		return err
	}
	if err := CheckByteCap(encoded, PreregisteredManifest().ApplicationCertificateByteCap); err != nil {
		return fmt.Errorf("application certificate: %w", err)
	}
	return nil
}

func VerifyApplicationCertificate(data []byte) (ApplicationCertificate, error) {
	certificate, err := StrictDecode[ApplicationCertificate](data)
	if err != nil {
		return certificate, err
	}
	if err := validateApplicationCertificate(certificate); err != nil {
		return certificate, err
	}
	if err := requireDigest("certificate_digest", certificate.CertificateDigest, false); err != nil {
		return certificate, err
	}
	want, err := applicationCertificateDigest(certificate)
	if err != nil {
		return certificate, err
	}
	if certificate.CertificateDigest != want {
		return certificate, errors.New("application certificate digest mismatch")
	}
	if err := CheckByteCap(data, PreregisteredManifest().ApplicationCertificateByteCap); err != nil {
		return certificate, err
	}
	return certificate, nil
}
