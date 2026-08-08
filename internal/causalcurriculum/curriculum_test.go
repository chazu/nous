package causalcurriculum

import (
	"context"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/causalv2"
)

func TestCurriculumExecutesExactCreditedLedgerAndRejectsTamper(t *testing.T) {
	profileBytes, episodes, certificates := curriculumFixture(t, true)
	result, err := Run(context.Background(), profileBytes, episodes, certificates)
	if err != nil {
		t.Fatal(err)
	}
	rules := sortedRuleCodes()
	if result.SelectedRule != rules[0] || result.Unresolved {
		t.Fatalf("selection=(%q,%v), want (%q,false)", result.SelectedRule, result.Unresolved, rules[0])
	}
	if len(result.Applications) != 480 || len(result.Aggregates) != 40 || len(result.WinnerTies) != 1 || len(result.Transcript) != 521 || len(result.TaskMeterItems) != 525 {
		t.Fatalf("cardinalities apps=%d aggs=%d ties=%d transcript=%d tasks=%d", len(result.Applications), len(result.Aggregates), len(result.WinnerTies), len(result.Transcript), len(result.TaskMeterItems))
	}
	if len(result.ArtifactBytes) != 2044 || result.Counts.AttributedUnits != 2044 {
		t.Fatalf("credited ledger=(%d,%d), want 2044", len(result.ArtifactBytes), result.Counts.AttributedUnits)
	}
	wantKinds := map[int]string{0: "central-descriptor", 1: "certificate", 480: "certificate", 481: "central-rule", 520: "central-rule", 521: "application", 522: "credit", 1480: "credit", 1481: "aggregate", 1520: "aggregate", 1521: "central-tie", 1522: "central-selection", 1523: "transcript", 2043: "transcript"}
	for index, kind := range wantKinds {
		artifact, verifyErr := causalv2.VerifyArtifact(result.ArtifactBytes[index])
		if verifyErr != nil || artifact.Kind != kind || artifact.ChargeIndex != index {
			t.Fatalf("artifact %d=(%q,%d,%v), want (%q,%d,nil)", index, artifact.Kind, artifact.ChargeIndex, verifyErr, kind, index)
		}
	}
	if result.TaskMeterItems[0].Subject != "000001:initialization" || result.TaskMeterItems[1].Subject != "000002:admission" || result.TaskMeterItems[481].Subject != "000482:matrix-barrier" || result.TaskMeterItems[522].Subject != "000523:aggregate-barrier" || result.TaskMeterItems[524].Subject != "000525:transcript-barrier" {
		t.Fatalf("noncanonical task ordering: first=%q last=%q", result.TaskMeterItems[0].Subject, result.TaskMeterItems[524].Subject)
	}
	if err := Verify(profileBytes, episodes, certificates, result); err != nil {
		t.Fatalf("verify valid result: %v", err)
	}
	tampered := result
	tampered.SelectedRule = rules[1]
	if err := Verify(profileBytes, episodes, certificates, tampered); err == nil {
		t.Fatal("selected-rule tamper was accepted")
	}
	tampered = result
	tampered.ArtifactBytes = cloneBytes(result.ArtifactBytes)
	tampered.ArtifactBytes[len(tampered.ArtifactBytes)-1][20] ^= 1
	if err := Verify(profileBytes, episodes, certificates, tampered); err == nil {
		t.Fatal("artifact-byte tamper was accepted")
	}
	tampered = result
	tampered.TaskMeterItems = append([]causalv2.TaskMeterItem(nil), result.TaskMeterItems...)
	tampered.TaskMeterItems[0].Counts[12]++
	if err := Verify(profileBytes, episodes, certificates, tampered); err == nil {
		t.Fatal("task-meter tamper was accepted")
	}
}

func TestCurriculumNoCreditPreservesEvidenceAndSuppressesSelection(t *testing.T) {
	profileBytes, episodes, certificates := curriculumFixture(t, false)
	result, err := Run(context.Background(), profileBytes, episodes, certificates)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedRule != "" || !result.Unresolved || len(result.WinnerTies) != 0 || len(result.Transcript) != 0 {
		t.Fatalf("no-credit resolution=(%q,%v,%d,%d)", result.SelectedRule, result.Unresolved, len(result.WinnerTies), len(result.Transcript))
	}
	if len(result.ArtifactBytes) != 1041 || len(result.Applications) != 480 || len(result.Aggregates) != 40 {
		t.Fatalf("no-credit evidence ledger=%d apps=%d aggs=%d", len(result.ArtifactBytes), len(result.Applications), len(result.Aggregates))
	}
	for _, encoded := range result.ArtifactBytes {
		artifact, verifyErr := causalv2.VerifyArtifact(encoded)
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
		if artifact.Kind == "credit" || artifact.Kind == "central-tie" || artifact.Kind == "central-selection" || artifact.Kind == "transcript" {
			t.Fatalf("no-credit control emitted %q", artifact.Kind)
		}
	}
	if err := Verify(profileBytes, episodes, certificates, result); err != nil {
		t.Fatalf("verify no-credit result: %v", err)
	}
}

func TestCreditModeIsSignedIntoCentralProfile(t *testing.T) {
	creditedProfile, episodes, certificates := curriculumFixture(t, true)
	noCreditProfile, _, _ := curriculumFixture(t, false)
	if string(creditedProfile) == string(noCreditProfile) {
		t.Fatal("credited and no-credit profiles have identical signed bytes")
	}
	credited, err := Run(context.Background(), creditedProfile, episodes, certificates)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(noCreditProfile, episodes, certificates, credited); err == nil {
		t.Fatal("credited result verified under separately signed no-credit profile")
	}
}

func TestCurriculumRejectsForgedIncompleteAndDuplicateMatrices(t *testing.T) {
	profileBytes, episodes, certificates := curriculumFixture(t, true)
	t.Run("forged certificate", func(t *testing.T) {
		forged := cloneBytes(certificates)
		forged[0][len(forged[0])-2] ^= 1
		if _, err := Run(context.Background(), profileBytes, episodes, forged); err == nil {
			t.Fatal("forged certificate was accepted")
		}
	})
	t.Run("re-signed certificate not derived from episode", func(t *testing.T) {
		forged := cloneBytes(certificates)
		certificate, err := causalv2.VerifyApplicationCertificate(forged[0])
		if err != nil {
			t.Fatal(err)
		}
		certificate.Score++
		if err := causalv2.SignApplicationCertificate(&certificate); err != nil {
			t.Fatal(err)
		}
		forged[0], _ = causalv2.CanonicalJSON(certificate)
		if _, err := Run(context.Background(), profileBytes, episodes, forged); err == nil {
			t.Fatal("re-signed certificate differing from its episode was accepted")
		}
	})
	t.Run("re-signed episode not summarized by certificate", func(t *testing.T) {
		forged := cloneBytes(episodes)
		episode, err := causalv2.VerifyTrainingEpisodeEvidence(forged[0])
		if err != nil {
			t.Fatal(err)
		}
		episode.Score++
		if err := causalv2.SignTrainingEpisodeEvidence(&episode); err != nil {
			t.Fatal(err)
		}
		forged[0], _ = causalv2.CanonicalJSON(episode)
		if _, err := Run(context.Background(), profileBytes, forged, certificates); err == nil {
			t.Fatal("re-signed episode differing from its certificate was accepted")
		}
	})
	t.Run("incomplete matrix", func(t *testing.T) {
		if _, err := Run(context.Background(), profileBytes, episodes[:479], certificates[:479]); err == nil {
			t.Fatal("incomplete matrix was accepted")
		}
	})
	t.Run("duplicate matrix entry", func(t *testing.T) {
		duplicate := cloneBytes(certificates)
		duplicate[1] = append([]byte(nil), duplicate[0]...)
		if _, err := Run(context.Background(), profileBytes, episodes, duplicate); err == nil {
			t.Fatal("duplicate matrix entry was accepted")
		}
	})
}

func curriculumFixture(t *testing.T, creditEnabled bool) ([]byte, [][]byte, [][]byte) {
	t.Helper()
	profile := causalv2.CentralProfile{CentralProfileVersion: causalv2.CentralProfileDomain, Manifest: causalv2.PreregisteredManifest(), PlanCommit: strings.Repeat("a", 40), PretrainingCommit: strings.Repeat("b", 40), CreditEnabled: creditEnabled}
	if err := causalv2.SignCentralProfile(&profile); err != nil {
		t.Fatal(err)
	}
	profileBytes, err := causalv2.CanonicalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	rules, seeds := sortedRuleCodes(), trainingSeeds(profile.Manifest)
	episodes := make([][]byte, 0, certificateCount)
	certificates := make([][]byte, 0, certificateCount)
	for _, seed := range seeds {
		for ruleIndex, rule := range rules {
			meterItems := make([]causalv2.MeterItem, len(causalv2.MeterNames))
			for index, name := range causalv2.MeterNames {
				meterItems[index] = causalv2.MeterItem{Name: name}
			}
			meterItems[0].Active = true
			meterItems[1].Active = true
			meterItems[4].Active = true
			meterItems[5].Active = ruleIndex == 0
			episode := causalv2.TrainingEpisodeEvidence{
				EpisodeReportVersion: causalv2.TrainingEpisodeDomain, Seed: seed, ProfileDigest: strings.Repeat("1", 64), FixtureDigest: strings.Repeat("2", 64), RuleCode: rule,
				Actions: []string{}, TeacherOutcomes: []string{}, Score: ruleIndex, Terminal: "identified", Cost: 0, FinalPosterior: []string{},
				PosteriorDigest: strings.Repeat("3", 64), TranscriptDigest: strings.Repeat("4", 64), OracleAgreements: 1, OracleDisagreements: 0,
				MeterItems: meterItems, AllCapsValid: true,
			}
			if err := causalv2.SignTrainingEpisodeEvidence(&episode); err != nil {
				t.Fatal(err)
			}
			episodeEncoded, err := causalv2.CanonicalJSON(episode)
			if err != nil {
				t.Fatal(err)
			}
			episodes = append(episodes, episodeEncoded)
			certificate := causalv2.ApplicationCertificate{
				Seed: seed, ProfileDigest: strings.Repeat("1", 64), FixtureDigest: strings.Repeat("2", 64), RuleCode: rule,
				Score: ruleIndex, Terminal: "identified", Cost: 0,
				PosteriorDigest: strings.Repeat("3", 64), TranscriptDigest: strings.Repeat("4", 64), OracleAgreements: 1,
				OracleDisagreements: 0, AllCapsValid: true, EpisodeReportDigest: episode.EpisodeReportDigest,
			}
			if err := causalv2.SignApplicationCertificate(&certificate); err != nil {
				t.Fatal(err)
			}
			encoded, err := causalv2.CanonicalJSON(certificate)
			if err != nil {
				t.Fatal(err)
			}
			certificates = append(certificates, encoded)
		}
	}
	return profileBytes, episodes, certificates
}

func cloneBytes(source [][]byte) [][]byte {
	result := make([][]byte, len(source))
	for index := range source {
		result[index] = append([]byte(nil), source[index]...)
	}
	return result
}
