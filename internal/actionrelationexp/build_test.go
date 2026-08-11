package actionrelationexp

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBuildPlanAuthorityMatchesCommittedReview(t *testing.T) {
	canonical, err := os.ReadFile("../../docs/actionrelations-plan-reviews.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ParseReviewManifest(canonical, map[string][]byte{
		"docs/actionrelations-reviews/plan/round-15/architecture.txt":          []byte("ACCEPTED"),
		"docs/actionrelations-reviews/plan/round-15/action-semantics.txt":      []byte("ACCEPTED"),
		"docs/actionrelations-reviews/plan/round-15/experimental-validity.txt": []byte("ACCEPTED"),
	})
	if err != nil || manifest.ReviewedCommit != PlanCommit || manifest.ArchiveDigest != PlanArchiveDigest {
		t.Fatalf("build plan authority differs from committed review: %v", err)
	}
}

func TestBuildAuthorityClosesSourceToolAndNonInputRows(t *testing.T) {
	row := SourceRow{Path: "cmd/nous/main.go", GitMode: "100644", GitBlobOID: strings.Repeat("a", 40), ByteLength: 12, Digest: shaHex([]byte("source")), Role: "compiler-input"}
	implementation := strings.Repeat("b", 40)
	sourceRoot, err := BuildSourceRoot(implementation, []SourceRow{row})
	if err != nil {
		t.Fatal(err)
	}
	ref := func(path string) AuthorityRef {
		value, refErr := Reference(path, []byte(path))
		if refErr != nil {
			t.Fatal(refErr)
		}
		return value
	}
	value, err := BuildBuildAuthority(BuildAuthority{
		PlanCommit: PlanCommit, PlanArchiveDigest: PlanArchiveDigest,
		PlanReview: ref(ReviewManifestPath("plan")), ImplementationCommit: implementation,
		ImplementationArchiveDigest: shaHex([]byte("implementation archive")), ImplementationReview: ref(ReviewManifestPath("implementation")),
		BuildHead: strings.Repeat("c", 40), SourceRoot: sourceRoot, SourceRows: []SourceRow{row},
		GitVersion: "git version 2.52.0", GoVersion: "go version go1.25.12 darwin/arm64",
		GoExecutablePath: "/opt/go/bin/go", GoExecutableDigest: shaHex([]byte("go")), MiseTomlDigest: shaHex([]byte("mise")),
		BuildArgv: []string{"/opt/go/bin/go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-o", PanelBinaryPath, "./cmd/nous"},
		BuildEnvironment: []EnvironmentRow{
			{Key: "CGO_ENABLED", Value: "0"}, {Key: "GOARCH", Value: "arm64"}, {Key: "GOCACHE", Value: "/tmp/cache"},
			{Key: "GOENV", Value: "off"}, {Key: "GOFLAGS", Value: ""}, {Key: "GOMODCACHE", Value: "/tmp/mod"},
			{Key: "GOOS", Value: "darwin"}, {Key: "GOPATH", Value: "/tmp/gopath"}, {Key: "GOPROXY", Value: "off"},
			{Key: "GOSUMDB", Value: "off"}, {Key: "GOTOOLCHAIN", Value: "local"}, {Key: "GOWORK", Value: "off"},
			{Key: "HOME", Value: "/tmp/home"}, {Key: "LC_ALL", Value: "C"}, {Key: "TMPDIR", Value: "/tmp/build"}, {Key: "TZ", Value: "UTC"},
		},
		GOOS: "darwin", GOARCH: "arm64", CGOEnabled: "0", BinaryPath: PanelBinaryPath,
		BinaryDigest: shaHex([]byte("binary")), GoVersionMDigest: shaHex([]byte("version-m")),
		NonInputRows: []NonInputRow{{Path: ".git/hooks", Status: "present-not-read"}, {Path: "go.work", Status: "present-not-read"}, {Path: "go.work.sum", Status: "present-not-read"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyBuildAuthority(value); err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseBuildAuthority(value.Canonical)
	if err != nil || !bytes.Equal(parsed.Canonical, value.Canonical) {
		t.Fatalf("parse: %v", err)
	}
	corrupted := value
	corrupted.SourceRows = append([]SourceRow(nil), value.SourceRows...)
	corrupted.SourceRows[0].Digest = strings.Repeat("0", 64)
	if VerifyBuildAuthority(corrupted) == nil {
		t.Fatal("accepted corrupted source row")
	}
	corrupted = value
	corrupted.BuildArgv = append([]string(nil), value.BuildArgv...)
	corrupted.BuildArgv = append(corrupted.BuildArgv, "-tags=answer")
	if VerifyBuildAuthority(corrupted) == nil {
		t.Fatal("accepted build tags")
	}
	corrupted = value
	corrupted.Canonical = bytes.Replace(value.Canonical, []byte("present-not-read"), []byte("absent----------"), 1)
	if VerifyBuildAuthority(corrupted) == nil {
		t.Fatal("accepted changed canonical authority")
	}
}

func TestSourceRowsRequireRawPathOrderAndAllowedRoles(t *testing.T) {
	base := SourceRow{GitMode: "100644", GitBlobOID: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64), Role: "test"}
	first, second := base, base
	first.Path, second.Path = "z_test.go", "a_test.go"
	if _, err := BuildSourceRoot(strings.Repeat("c", 40), []SourceRow{first, second}); err == nil {
		t.Fatal("accepted reordered source rows")
	}
	first.Path, first.Role = "a_test.go", "fixture"
	if _, err := BuildSourceRoot(strings.Repeat("c", 40), []SourceRow{first}); err == nil {
		t.Fatal("accepted unknown source role")
	}
}
