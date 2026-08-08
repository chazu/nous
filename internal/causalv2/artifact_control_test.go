package causalv2

import (
	"encoding/base64"
	"strings"
	"testing"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

func TestAuthorizationAndArtifactRoundTrip(t *testing.T) {
	digestA, digestB := strings.Repeat("a", 64), strings.Repeat("b", 64)
	authorization := Authorization{ProfileDigest: digestA, Episode: "episode-1", Step: 0, Action: "do:0=1", SelectionArtifactDigest: digestB, OpaqueToken: strings.Repeat("c", 64)}
	if err := SignAuthorization(&authorization); err != nil {
		t.Fatal(err)
	}
	authorizationBytes, _ := CanonicalJSON(authorization)
	if _, err := VerifyAuthorization(authorizationBytes); err != nil {
		t.Fatal(err)
	}
	artifact, err := NewArtifact(digestA, "episode-1", 0, "authorization", authorization, 7)
	if err != nil {
		t.Fatal(err)
	}
	artifactBytes, _ := CanonicalJSON(artifact)
	verified, err := VerifyArtifact(artifactBytes)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Name() != "Causal.authorization."+artifact.ArtifactDigest {
		t.Fatalf("artifact name=%q", verified.Name())
	}
	artifact.SemanticKey = strings.Repeat("0", 64)
	bad, _ := CanonicalJSON(artifact)
	if _, err := VerifyArtifact(bad); err == nil {
		t.Fatal("accepted corrupt semantic key")
	}
}

func TestCertificateArtifactVerifiesDecodedCertificate(t *testing.T) {
	digest := strings.Repeat("d", 64)
	certificate := ApplicationCertificate{
		Seed: 122001, ProfileDigest: digest, FixtureDigest: digest, RuleCode: causal.Rules()[0].Code(),
		Score: 10, Terminal: "identified", Cost: 10, PosteriorDigest: digest, TranscriptDigest: digest,
		OracleAgreements: 1, AllCapsValid: true, EpisodeReportDigest: digest,
	}
	if err := SignApplicationCertificate(&certificate); err != nil {
		t.Fatal(err)
	}
	certificateBytes, _ := CanonicalJSON(certificate)
	payload := CertificatePayload{CertificateBytes: base64.RawURLEncoding.EncodeToString(certificateBytes), CertificateDigest: certificate.CertificateDigest}
	artifact, err := NewArtifact(digest, "training", 0, "certificate", payload, 1)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := CanonicalJSON(artifact)
	if _, err := VerifyArtifact(encoded); err != nil {
		t.Fatal(err)
	}
	payload.CertificateDigest = strings.Repeat("e", 64)
	if _, err := NewArtifact(digest, "training", 0, "certificate", payload, 1); err == nil {
		t.Fatal("accepted certificate payload digest mismatch")
	}
}

func emptyControlResult() ControlResult {
	return ControlResult{Actions: []string{}, Outcomes: []string{}, PosteriorDigests: []string{}, Costs: []int{}}
}

func TestControlBundleRoundTrip(t *testing.T) {
	counter := Counter{SCMEvaluations: 1, TotalWork: 1}
	bundle := ControlBundle{ControlBundleVersion: ControlBundleDomain}
	for _, name := range ControlNames {
		certificate := ControlCertificate{ControlVersion: ControlCertificateDomain, Name: name, TreatmentEvidence: emptyControlResult(), ControlEvidence: emptyControlResult(), Observed: "executed", Passed: true, MeterCounts: counter.Counts(), Work: 1}
		if err := SignControlCertificate(&certificate); err != nil {
			t.Fatalf("sign %s: %v", name, err)
		}
		bundle.Certificates = append(bundle.Certificates, certificate)
	}
	if err := SignControlBundle(&bundle); err != nil {
		t.Fatal(err)
	}
	encoded, _ := CanonicalJSON(bundle)
	if _, err := VerifyControlBundle(encoded); err != nil {
		t.Fatal(err)
	}
	bundle.Certificates[0], bundle.Certificates[1] = bundle.Certificates[1], bundle.Certificates[0]
	if err := SignControlBundle(&bundle); err == nil {
		t.Fatal("accepted reordered controls")
	}
}
