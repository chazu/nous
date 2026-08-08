package causalv2

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

func TestControlEvidenceMaximumWidthCapGolden(t *testing.T) {
	manifest := PreregisteredManifest()
	const (
		maximumCases       = 486
		maximumCaseBytes   = 2048
		fixedShellBytes    = 262144
		maximumCasesArray  = maximumCases*maximumCaseBytes + (maximumCases - 1) + 2
		maximumRecordBytes = 1572864 + 524288 + 1048576 + 262144 + 524288
	)
	if maximumCasesArray != 995815 || maximumCasesArray > manifest.ControlEvidenceCorruptionCasesByteCap {
		t.Fatalf("corruption cases array bound=%d cap=%d", maximumCasesArray, manifest.ControlEvidenceCorruptionCasesByteCap)
	}
	if maximumRecordBytes != 3932160 || maximumRecordBytes+fixedShellBytes != manifest.ControlEvidenceByteCap {
		t.Fatalf("control evidence budget=%d+%d, cap=%d", maximumRecordBytes, fixedShellBytes, manifest.ControlEvidenceByteCap)
	}
	groups := []int{manifest.ControlEvidenceNoCreditArtifactsByteCap, manifest.ControlEvidenceCorruptionBaselineByteCap, manifest.ControlEvidenceCorruptionCasesByteCap, manifest.ControlEvidenceDependencyFilesByteCap, manifest.ControlEvidenceOtherRecordsByteCap}
	total := 0
	for _, group := range groups {
		total += group
	}
	if total != maximumRecordBytes {
		t.Fatalf("manifest evidence group total=%d, want %d", total, maximumRecordBytes)
	}
	maximumWidth := exactWidthCorruptionCase(t, maximumCaseBytes)
	cases := make([]CorruptionCase, maximumCases)
	for index := range cases {
		cases[index] = maximumWidth
	}
	casesBytes, err := CanonicalJSON(cases)
	if err != nil || len(casesBytes) != maximumCasesArray {
		t.Fatalf("materialized corruption cases bytes=%d want=%d err=%v", len(casesBytes), maximumCasesArray, err)
	}

	evidence := validControlEvidenceForTest(t)
	evidence.StaticMatrix = []MatrixControlRow{}
	evidence.RecomputedMatrix = []MatrixControlRow{}
	evidence.Mutation.OffMutants = []MutantRecord{}
	evidence.Mutation.OnMutants = []MutantRecord{}
	evidence.NoCredit.CertificateDigests = []string{}
	evidence.NoCredit.ArtifactBytes = []Base64URL{Base64URL(strings.Repeat("A", manifest.ControlEvidenceNoCreditArtifactsByteCap-4))}
	evidence.NoCredit.Aggregates = []RuleAggregatePayload{}
	evidence.NoCredit.CentralTranscript = []CentralTranscriptEvent{}
	evidence.NoCredit.TaskMeterItems = []TaskMeterItem{}
	evidence.Corruption.BaselineArtifacts = []Base64URL{Base64URL(strings.Repeat("A", manifest.ControlEvidenceCorruptionBaselineByteCap-4))}
	evidence.Corruption.Cases = cases
	evidence.Dependency.AuditedRoots = []string{}
	evidence.Dependency.Files = []DependencyFile{exactWidthDependencyFile(t, manifest.ControlEvidenceDependencyFilesByteCap)}
	evidence.Dependency.RunnerMethods = []string{strings.Repeat("m", manifest.ControlEvidenceOtherRecordsByteCap-24-4)}
	evidence.Dependency.RunnerFields = []RunnerField{}
	evidence.Dependency.TeacherMethods = []string{}
	evidence.Dependency.Forbidden = []string{}
	if err := checkControlEvidenceCaps(evidence); err != nil {
		t.Fatalf("maximum-width grouped evidence rejected: %v", err)
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
	if err != nil || len(shellBytes) > fixedShellBytes {
		t.Fatalf("maximum-width fixed shell bytes=%d cap=%d err=%v", len(shellBytes), fixedShellBytes, err)
	}
	over := evidence
	over.NoCredit.ArtifactBytes = append([]Base64URL(nil), evidence.NoCredit.ArtifactBytes...)
	over.NoCredit.ArtifactBytes[0] += "A"
	if err := checkControlEvidenceCaps(over); err == nil {
		t.Fatal("no-credit group one byte over its boundary was accepted")
	}
}

func exactWidthCorruptionCase(t *testing.T, target int) CorruptionCase {
	t.Helper()
	item := CorruptionCase{Name: "n", MutationDescriptor: "n", MutatedBytesDigest: strings.Repeat("a", 64), RejectionCode: "", MeterCounts: [15]int64{4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 4000000, 531441, 4000000, 4000000, 4000000, 5000, 1000, 4000000}}
	base, err := CanonicalJSON(item)
	if err != nil || len(base) > target {
		t.Fatalf("corruption case base bytes=%d target=%d err=%v", len(base), target, err)
	}
	item.RejectionCode = strings.Repeat("r", target-len(base))
	encoded, err := CanonicalJSON(item)
	if err != nil || len(encoded) != target {
		t.Fatalf("corruption case bytes=%d target=%d err=%v", len(encoded), target, err)
	}
	return item
}

func exactWidthDependencyFile(t *testing.T, target int) DependencyFile {
	t.Helper()
	item := DependencyFile{Path: "", SourceSHA256: strings.Repeat("a", 64), Imports: []string{}, ExportedFunctionParameters: []DependencyParameter{}}
	base, err := CanonicalJSON([]DependencyFile{item})
	if err != nil || len(base) > target {
		t.Fatalf("dependency array base bytes=%d target=%d err=%v", len(base), target, err)
	}
	item.Path = strings.Repeat("p", target-len(base))
	encoded, err := CanonicalJSON([]DependencyFile{item})
	if err != nil || len(encoded) != target {
		t.Fatalf("dependency array bytes=%d target=%d err=%v", len(encoded), target, err)
	}
	return item
}

func validControlEvidenceForTest(t *testing.T) ControlEvidence {
	t.Helper()
	digest := strings.Repeat("a", 64)
	empty := ControlResult{Actions: []string{}, Outcomes: []string{}, PosteriorDigests: []string{}, Costs: []int{}}
	evidence := ControlEvidence{ControlEvidenceVersion: ControlEvidenceDomain, SelectedRule: causal.Rules()[0].Code(), StaticRule: causal.Rules()[0].Code(), StaticMatrix: []MatrixControlRow{}, RecomputedMatrix: []MatrixControlRow{}}
	for index := 0; index < 12; index++ {
		seed := PreregisteredManifest().TrainingSeeds.Start + int64(index)
		base := MatrixControlRow{Seed: seed, FixtureDigest: digest, TreatmentEpisodeDigest: digest, TreatmentCertificateDigest: digest, Treatment: empty, Control: empty, TreatmentCache: CacheTrace{Statuses: []string{}}, ControlCache: CacheTrace{Statuses: []string{}}}
		static := base
		static.ControlEpisodeDigest = digest
		static.ControlCertificateDigest = digest
		evidence.StaticMatrix = append(evidence.StaticMatrix, static)
		evidence.RecomputedMatrix = append(evidence.RecomputedMatrix, base)
	}
	off := MutationConfig{Interval: 1, MaxMutants: 1, MutantWorth: 400, MinApplics: 1, MutationThreshold: 2}
	on := off
	on.Enabled = true
	evidence.Mutation = MutationProof{FixtureDigest: digest, OffConfig: off, OnConfig: on, OffResult: empty, OnResult: empty, OffMutants: []MutantRecord{}, OnMutants: []MutantRecord{{Name: "M-H-1", MutantOf: "H", SourceSlot: "thenCompute", Operation: "delete-token", ProgramDigest: digest, Worth: 400}}}
	evidence.ChildVM = ChildVMProof{FixtureDigest: digest, ProfileDigest: digest, Operation: "causal-v2-task-valid?", FailureCode: "child-vm-unauthorized"}
	caseNames := []string{"delete-kind-cache"}
	caseDigest, _ := Digest(CorruptionEnumeratorDomain, caseNames)
	evidence.Corruption = CorruptionProof{EnumeratorVersion: CorruptionEnumeratorDomain, FixtureBytes: EncodeBase64URL([]byte("fixture")), ProfileBytes: EncodeBase64URL([]byte("profile")), BaselineArtifacts: []Base64URL{EncodeBase64URL([]byte("artifact"))}, CaseCount: 1, CaseSetDigest: caseDigest, Cases: []CorruptionCase{{Name: caseNames[0], MutationDescriptor: caseNames[0], MutatedBytesDigest: digest, RejectionCode: "corruption-rejected"}}}
	noCredit := NoCreditProof{CentralProfileBytes: EncodeBase64URL([]byte("profile")), CertificateDigests: make([]string, 480), ArtifactBytes: make([]Base64URL, 1041), Aggregates: make([]RuleAggregatePayload, 40), CentralTranscript: []CentralTranscriptEvent{}, TaskMeterItems: []TaskMeterItem{}, Resolution: "unresolved", WinnerTies: []string{}, TerminalTranscriptDigest: ZeroDigest}
	for index := range noCredit.CertificateDigests {
		noCredit.CertificateDigests[index] = digest
	}
	for index := range noCredit.ArtifactBytes {
		noCredit.ArtifactBytes[index] = EncodeBase64URL([]byte("a"))
	}
	for index := 0; index < 525; index++ {
		noCredit.TaskMeterItems = append(noCredit.TaskMeterItems, TaskMeterItem{Name: "curriculum", Subject: fmt.Sprintf("%06d:test", index+1)})
	}
	evidence.NoCredit = noCredit
	evidence.Dependency = DependencyProof{AuditedCommit: strings.Repeat("b", 40), AuditedRoots: []string{"."}, Files: []DependencyFile{{Path: "internal/example.go", SourceSHA256: digest, Imports: []string{}, ExportedFunctionParameters: []DependencyParameter{}}}, RunnerMethods: []string{}, RunnerFields: []RunnerField{}, TeacherMethods: []string{}, Lookups: 1, Forbidden: []string{}}
	if err := SignControlEvidence(&evidence); err != nil {
		t.Fatal(err)
	}
	return evidence
}

func TestControlEvidenceRoundTripAndAttacks(t *testing.T) {
	evidence := validControlEvidenceForTest(t)
	encoded, _ := CanonicalJSON(evidence)
	if _, err := VerifyControlEvidence(encoded); err != nil {
		t.Fatal(err)
	}
	tampered := evidence
	tampered.StaticMatrix = append([]MatrixControlRow(nil), evidence.StaticMatrix...)
	tampered.StaticMatrix[0].Treatment.Score++
	bad, _ := CanonicalJSON(tampered)
	if _, err := VerifyControlEvidence(bad); err == nil {
		t.Fatal("accepted digest-stale matrix tamper")
	}
	unknown := append([]byte(nil), encoded...)
	unknown = bytes.Replace(unknown, []byte(`"selected_rule":`), []byte(`"unexpected":true,"selected_rule":`), 1)
	if _, err := VerifyControlEvidence(unknown); err == nil {
		t.Fatal("accepted unknown evidence field")
	}
	base64Tamper := evidence
	base64Tamper.Corruption.FixtureBytes = Base64URL("Zml4dHVyZQ==")
	base64Tamper.ControlEvidenceDigest = ""
	if err := SignControlEvidence(&base64Tamper); err == nil {
		t.Fatal("accepted padded base64")
	}
}
