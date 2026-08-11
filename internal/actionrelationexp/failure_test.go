package actionrelationexp

import (
	"strings"
	"testing"
)

func TestInvalidAuthorityRoundTripsOnlyClosedFailureKinds(t *testing.T) {
	value, err := BuildInvalidAuthority(InvalidAuthority{
		Panel: "validation", Kind: "report", SourceRoot: testAuthorityDigest("source"),
		AttemptCommitment: testAuthorityDigest("attempt"), Reason: "isolated audit failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseInvalidAuthority(value.Canonical)
	if err != nil || parsed.Digest != value.Digest || parsed.Kind != "report" {
		t.Fatalf("failure authority did not round trip: %+v %v", parsed, err)
	}
	value.Kind = "publication"
	if _, err := BuildInvalidAuthority(value); err == nil {
		t.Fatal("accepted invalid-authority publication placeholder")
	}
}

func TestDevelopmentInvalidAuthorityRequiresZeroAttemptCommitment(t *testing.T) {
	_, err := BuildInvalidAuthority(InvalidAuthority{
		Panel: "development", Kind: "fixture-root", SourceRoot: testAuthorityDigest("source"),
		AttemptCommitment: testAuthorityDigest("protected"), Reason: "failed",
	})
	if err == nil {
		t.Fatal("development failure accepted protected attempt authority")
	}
}

func TestInvalidAuthorityReasonRequiresPrintableASCII(t *testing.T) {
	base := InvalidAuthority{
		Panel: "validation", Kind: "report", SourceRoot: testAuthorityDigest("source"),
		AttemptCommitment: testAuthorityDigest("attempt"), Reason: "failed",
	}
	for _, reason := range []string{"", "line\nbreak", "tab\tbreak", "non-ascii \u00e9"} {
		base.Reason = reason
		if _, err := BuildInvalidAuthority(base); err == nil {
			t.Fatalf("accepted non-printable reason %q", reason)
		}
	}
	base.Reason = strings.Repeat("x", 1025)
	if _, err := BuildInvalidAuthority(base); err == nil {
		t.Fatal("accepted oversized reason")
	}
}
