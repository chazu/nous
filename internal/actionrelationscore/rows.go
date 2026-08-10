// Package actionrelationscore executes policy/curriculum scoring after fixture
// truth has been sealed and constructs the canonical scorer rows.
package actionrelationscore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
)

const (
	LifecycleCap = 2_000_000
	HistoryCap   = 65_536
	zeroDigest   = "0000000000000000000000000000000000000000000000000000000000000000"
)

var Policies = []actionrelationsearch.Policy{
	actionrelationsearch.Complete,
	actionrelationsearch.Lexical,
	actionrelationsearch.StaticSleep,
	actionrelationsearch.DynamicSleep,
	actionrelationsearch.NousSleep,
	actionrelationsearch.NoGuardSleep,
	actionrelationsearch.LearnedNoUse,
}

// MatchCounts is the closed scorer diagnostic vector. The first five fields
// reconstruct training precision/recall; the last three reconstruct utility
// matched-pair and false-match gates for the row's named stratum.
type MatchCounts struct {
	TrainingPositive      int
	TrainingNegative      int
	TrainingTruePositive  int
	TrainingFalsePositive int
	TrainingFalseNegative int
	UtilityAttempts       int
	UtilityMatches        int
	UtilityFalseMatches   int
}

func (m MatchCounts) wire() []int {
	return []int{m.TrainingPositive, m.TrainingNegative, m.TrainingTruePositive, m.TrainingFalsePositive, m.TrainingFalseNegative, m.UtilityAttempts, m.UtilityMatches, m.UtilityFalseMatches}
}

// CertificateCounts is [attempted,successful,cached-success,cached-failure].
type CertificateCounts struct {
	Attempted     int
	Successful    int
	CachedSuccess int
	CachedFailure int
}

func (c CertificateCounts) wire() []int {
	return []int{c.Attempted, c.Successful, c.CachedSuccess, c.CachedFailure}
}

type WorldPolicyRow struct {
	Panel                    string
	Curriculum               int
	Family                   int
	WorldOrdinal             int
	Stratum                  string
	WorldDigest              string
	Policy                   actionrelationsearch.Policy
	SearchTerminal           string
	UtilityWorkVector        [12]int
	UtilityTotal             int
	MatchCounts              MatchCounts
	CertificateCounts        CertificateCounts
	SleepCount               int
	HistoryCount             int
	TerminalSetDigest        string
	WorkTerminalDigestOrZero string
	BehaviorEqual            bool
	BudgetRemaining          int
	OperationRoot            actionrelationexp.OperationRoot
	Canonical                []byte
	Digest                   string
}

func BuildWorldPolicyRow(value WorldPolicyRow) (WorldPolicyRow, error) {
	value.Canonical, _ = json.Marshal([]any{
		"actionrelation-world-policy-row/v2", value.Panel, value.Curriculum, value.Family,
		value.WorldOrdinal, value.Stratum, value.WorldDigest, string(value.Policy), value.SearchTerminal,
		value.UtilityWorkVector, value.UtilityTotal, value.MatchCounts.wire(), value.CertificateCounts.wire(),
		value.SleepCount, value.HistoryCount, value.TerminalSetDigest, value.WorkTerminalDigestOrZero,
		value.BehaviorEqual, value.BudgetRemaining, value.OperationRoot.Digest,
	})
	value.Digest = digest(value.Canonical)
	if err := VerifyWorldPolicyRow(value); err != nil {
		return WorldPolicyRow{}, err
	}
	return value, nil
}

func VerifyWorldPolicyRow(value WorldPolicyRow) error {
	if value.Panel != "development" && value.Panel != "validation" && value.Panel != "locked" || value.Curriculum < 0 || value.Family < 0 || value.Family > 7 || value.WorldOrdinal < 0 || value.WorldOrdinal > 5 || !oneStratum(value.Stratum) || !digestText(value.WorldDigest) || !slices.Contains(Policies, value.Policy) {
		return fmt.Errorf("invalid world-policy identity")
	}
	if value.SearchTerminal != "completed" && value.SearchTerminal != "budget-exhausted" || !nonnegativeVector(value.UtilityWorkVector) || sum(value.UtilityWorkVector) != value.UtilityTotal || value.UtilityTotal < 1 || !nonnegative(value.MatchCounts.wire()) || !nonnegative(value.CertificateCounts.wire()) || value.SleepCount < 0 || value.HistoryCount < 0 || value.HistoryCount > HistoryCap || !digestText(value.TerminalSetDigest) || !digestText(value.WorkTerminalDigestOrZero) || value.BudgetRemaining < 0 || value.BudgetRemaining > LifecycleCap {
		return fmt.Errorf("invalid world-policy result")
	}
	if value.SearchTerminal == "completed" && value.WorkTerminalDigestOrZero != zeroDigest || value.SearchTerminal == "budget-exhausted" && (value.WorkTerminalDigestOrZero == zeroDigest || value.BudgetRemaining != 0) || value.OperationRoot.Digest != digest(value.OperationRoot.Canonical) || actionrelationexp.ValidateObject(46, value.OperationRoot.Canonical) != nil {
		return fmt.Errorf("invalid world-policy terminal authority")
	}
	want := value
	want.Canonical, want.Digest = nil, ""
	rebuilt, _ := json.Marshal([]any{
		"actionrelation-world-policy-row/v2", want.Panel, want.Curriculum, want.Family,
		want.WorldOrdinal, want.Stratum, want.WorldDigest, string(want.Policy), want.SearchTerminal,
		want.UtilityWorkVector, want.UtilityTotal, want.MatchCounts.wire(), want.CertificateCounts.wire(),
		want.SleepCount, want.HistoryCount, want.TerminalSetDigest, want.WorkTerminalDigestOrZero,
		want.BehaviorEqual, want.BudgetRemaining, want.OperationRoot.Digest,
	})
	if !bytes.Equal(rebuilt, value.Canonical) || value.Digest != digest(value.Canonical) || actionrelationexp.ValidateObject(32, value.Canonical) != nil {
		return fmt.Errorf("invalid world-policy wire")
	}
	return nil
}

type CurriculumPolicyRow struct {
	Panel                               string
	Curriculum                          int
	Family                              int
	Policy                              actionrelationsearch.Policy
	AcquisitionTerminal                 string
	ArtifactDigest                      string
	AcquisitionWorkVector               [12]int
	AcquisitionWorkTerminalDigestOrZero string
	WorldRowDigests                     []string
	AggregateTerminal                   string
	SearchWorkVector                    [12]int
	SearchTotal                         int
	LifecycleWorkVector                 [12]int
	LifecycleTotal                      int
	BehaviorEqual                       bool
	BudgetRemaining                     int
	OperationRoot                       actionrelationexp.OperationRoot
	Canonical                           []byte
	Digest                              string
}

func BuildCurriculumPolicyRow(value CurriculumPolicyRow) (CurriculumPolicyRow, error) {
	value.Canonical, _ = json.Marshal([]any{
		"actionrelation-curriculum-policy-row/v2", value.Panel, value.Curriculum, value.Family, string(value.Policy),
		value.AcquisitionTerminal, value.ArtifactDigest, value.AcquisitionWorkVector, value.AcquisitionWorkTerminalDigestOrZero,
		value.WorldRowDigests, value.AggregateTerminal, value.SearchWorkVector, value.SearchTotal,
		value.LifecycleWorkVector, value.LifecycleTotal, value.BehaviorEqual, value.BudgetRemaining, value.OperationRoot.Digest,
	})
	value.Digest = digest(value.Canonical)
	if err := VerifyCurriculumPolicyRow(value); err != nil {
		return CurriculumPolicyRow{}, err
	}
	return value, nil
}

func VerifyCurriculumPolicyRow(value CurriculumPolicyRow) error {
	if value.Panel != "development" && value.Panel != "validation" && value.Panel != "locked" || value.Curriculum < 0 || value.Family < 0 || value.Family > 7 || !slices.Contains(Policies, value.Policy) || !oneAcquisitionTerminal(value.AcquisitionTerminal) || !digestText(value.ArtifactDigest) || !digestText(value.AcquisitionWorkTerminalDigestOrZero) || len(value.WorldRowDigests) != 6 || !uniqueDigests(value.WorldRowDigests) {
		return fmt.Errorf("invalid curriculum-policy identity")
	}
	if !nonnegativeVector(value.AcquisitionWorkVector) || !nonnegativeVector(value.SearchWorkVector) || !nonnegativeVector(value.LifecycleWorkVector) || value.SearchTotal != sum(value.SearchWorkVector) || value.LifecycleTotal != sum(value.LifecycleWorkVector) || value.LifecycleTotal != sum(value.AcquisitionWorkVector)+value.SearchTotal || value.LifecycleTotal > LifecycleCap || value.BudgetRemaining < 0 || value.BudgetRemaining > LifecycleCap {
		return fmt.Errorf("invalid curriculum-policy work")
	}
	for index := range value.LifecycleWorkVector {
		if value.LifecycleWorkVector[index] != value.AcquisitionWorkVector[index]+value.SearchWorkVector[index] {
			return fmt.Errorf("curriculum-policy counter %d does not conserve", index+1)
		}
	}
	if value.AggregateTerminal != "completed" && value.AggregateTerminal != "budget-exhausted" || value.AggregateTerminal == "completed" && value.BudgetRemaining != LifecycleCap-value.LifecycleTotal || value.AggregateTerminal == "budget-exhausted" && value.BudgetRemaining != 0 || value.AcquisitionTerminal == "not-applicable" && (value.ArtifactDigest != zeroDigest || sum(value.AcquisitionWorkVector) != 0 || value.AcquisitionWorkTerminalDigestOrZero != zeroDigest) {
		return fmt.Errorf("invalid curriculum-policy terminal")
	}
	if value.OperationRoot.Digest != digest(value.OperationRoot.Canonical) || actionrelationexp.ValidateObject(46, value.OperationRoot.Canonical) != nil {
		return fmt.Errorf("invalid curriculum-policy operation root")
	}
	want, _ := json.Marshal([]any{
		"actionrelation-curriculum-policy-row/v2", value.Panel, value.Curriculum, value.Family, string(value.Policy),
		value.AcquisitionTerminal, value.ArtifactDigest, value.AcquisitionWorkVector, value.AcquisitionWorkTerminalDigestOrZero,
		value.WorldRowDigests, value.AggregateTerminal, value.SearchWorkVector, value.SearchTotal,
		value.LifecycleWorkVector, value.LifecycleTotal, value.BehaviorEqual, value.BudgetRemaining, value.OperationRoot.Digest,
	})
	if !bytes.Equal(want, value.Canonical) || value.Digest != digest(value.Canonical) || actionrelationexp.ValidateObject(33, value.Canonical) != nil {
		return fmt.Errorf("invalid curriculum-policy wire")
	}
	return nil
}

func digest(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func digestText(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == hex.EncodeToString(decoded)
}

func nonnegative(values []int) bool {
	for _, value := range values {
		if value < 0 {
			return false
		}
	}
	return true
}

func nonnegativeVector(value [12]int) bool { return nonnegative(value[:]) }

func sum(value [12]int) int {
	total := 0
	for _, item := range value {
		total += item
	}
	return total
}

func oneStratum(value string) bool {
	return value == "positive-effect" || value == "neutral" || value == "adverse"
}

func oneAcquisitionTerminal(value string) bool {
	return value == "not-applicable" || value == "completed" || value == "no-discovery" || value == "budget-exhausted"
}

func uniqueDigests(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if !digestText(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
