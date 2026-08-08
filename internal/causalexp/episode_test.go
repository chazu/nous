package causalexp

import (
	"fmt"
	"testing"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

func TestEpisodeUsesCUEAndOracle(t *testing.T) {
	fixture, e := Generate("development", 12001, 0)
	if e != nil {
		t.Fatal(e)
	}
	report, e := runEpisode("../../domains", "development", fixture, InformationGain, "")
	if e != nil {
		t.Fatal(e)
	}
	if !report.Correct {
		t.Fatalf("terminal=%s score=%d posterior=%d actions=%v", report.Terminal, report.Score, report.FinalPosterior, report.Actions)
	}
	if report.OracleDisagreements != 0 || report.OracleAgreements == 0 {
		t.Fatalf("oracle=%d/%d", report.OracleAgreements, report.OracleDisagreements)
	}
}
func TestCurriculumRequiresCredit(t *testing.T) {
	var apps []ApplicationCertificate
	for _, rule := range causal.Rules() {
		for i := 0; i < 12; i++ {
			apps = append(apps, ApplicationCertificate{RuleCode: rule.Code(), Score: 100 + i, Terminal: "identified", CertificateDigest: fmt.Sprintf("%064x", len(apps)+1)})
		}
	}
	selected, _, _, _, e := runCurriculum("../../domains", apps, true)
	if e != nil {
		t.Fatal(e)
	}
	if selected == "" {
		t.Fatal("credit curriculum did not select")
	}
	none, _, _, _, e := runCurriculum("../../domains", apps, false)
	if e != nil {
		t.Fatal(e)
	}
	if none != "" {
		t.Fatalf("no-credit selected %q", none)
	}
}
