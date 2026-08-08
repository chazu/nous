package causalcurriculum

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"

	"github.com/chazu/nous/internal/causalv2"
)

func verifyResult(profileBytes []byte, episodeBytes, certificateBytes [][]byte, result Result) error {
	profile, err := causalv2.VerifyCentralProfile(profileBytes)
	if err != nil {
		return err
	}
	if result.ProfileDigest != profile.ProfileDigest || result.TrainingKey != profile.TrainingKey || result.CreditEnabled != profile.CreditEnabled {
		return errors.New("result is not bound to the central profile")
	}
	if len(episodeBytes) != certificateCount || len(certificateBytes) != certificateCount || len(result.Applications) != certificateCount {
		return errors.New("result lacks the exact 480-episode/certificate matrix")
	}
	rules, seeds := sortedRuleCodes(), trainingSeeds(profile.Manifest)
	certificates := make([]causalv2.ApplicationCertificate, certificateCount)
	for index, encoded := range certificateBytes {
		certificate, _, verifyErr := causalv2.VerifyApplicationCertificateForEpisode(encoded, episodeBytes[index])
		if verifyErr != nil {
			return verifyErr
		}
		if certificate.Seed != seeds[index/ruleCount] || certificate.RuleCode != rules[index%ruleCount] || !certificate.AllCapsValid || certificate.OracleDisagreements != 0 {
			return fmt.Errorf("certificate %d violates matrix admission", index)
		}
		certificates[index] = certificate
	}
	if !reflect.DeepEqual(result.Applications, certificates) {
		return errors.New("application result differs from admitted certificates")
	}

	artifacts := make([]causalv2.Artifact, len(result.ArtifactBytes))
	for index, encoded := range result.ArtifactBytes {
		artifact, verifyErr := causalv2.VerifyArtifact(encoded)
		if verifyErr != nil {
			return fmt.Errorf("artifact %d: %w", index, verifyErr)
		}
		if artifact.ChargeIndex != index || artifact.ProfileDigest != profile.ProfileDigest || artifact.Scope != profile.TrainingKey || artifact.Step != 0 {
			return fmt.Errorf("artifact %d has a noncanonical envelope", index)
		}
		artifacts[index] = artifact
	}
	position := 0
	next := func(kind string) (causalv2.Artifact, error) {
		if position >= len(artifacts) {
			return causalv2.Artifact{}, fmt.Errorf("missing %s artifact", kind)
		}
		artifact := artifacts[position]
		position++
		if artifact.Kind != kind {
			return artifact, fmt.Errorf("artifact %d kind=%q, want %q", position-1, artifact.Kind, kind)
		}
		return artifact, nil
	}
	descriptorArtifact, err := next("central-descriptor")
	if err != nil {
		return err
	}
	wantDescriptor := causalv2.CentralDescriptorPayload{CentralProfileDigest: profile.ProfileDigest, ExpectedRules: ruleCount, ExpectedSeeds: seeds, ExpectedCertificates: certificateCount, CreditEnabled: profile.CreditEnabled}
	if err := requirePayload(descriptorArtifact, wantDescriptor); err != nil {
		return err
	}
	for index, certificate := range certificates {
		artifact, nextErr := next("certificate")
		if nextErr != nil {
			return nextErr
		}
		want := causalv2.CertificatePayload{CertificateBytes: base64.RawURLEncoding.EncodeToString(certificateBytes[index]), CertificateDigest: certificate.CertificateDigest}
		if err := requirePayload(artifact, want); err != nil {
			return fmt.Errorf("certificate artifact %d: %w", index, err)
		}
	}
	for _, code := range rules {
		artifact, nextErr := next("central-rule")
		if nextErr != nil {
			return nextErr
		}
		if err := requirePayload(artifact, causalv2.CentralRulePayload{RuleCode: code}); err != nil {
			return err
		}
	}

	applicationArtifacts := make([]causalv2.Artifact, certificateCount)
	creditArtifacts := make([]causalv2.Artifact, 0, certificateCount)
	for index, certificate := range certificates {
		application, nextErr := next("application")
		if nextErr != nil {
			return nextErr
		}
		wantApplication := causalv2.ApplicationPayload{Seed: certificate.Seed, RuleCode: certificate.RuleCode, CertificateDigest: certificate.CertificateDigest, Score: certificate.Score, Terminal: certificate.Terminal, Cost: certificate.Cost}
		if err := requirePayload(application, wantApplication); err != nil {
			return fmt.Errorf("application %d: %w", index, err)
		}
		applicationArtifacts[index] = application
		if result.CreditEnabled {
			credit, creditErr := next("credit")
			if creditErr != nil {
				return creditErr
			}
			wantCredit := causalv2.CreditPayload{ApplicationArtifactDigest: application.ArtifactDigest, Delta: profile.Manifest.InvalidOrExhaustedScore - certificate.Score}
			if err := requirePayload(credit, wantCredit); err != nil {
				return fmt.Errorf("credit %d: %w", index, err)
			}
			creditArtifacts = append(creditArtifacts, credit)
		}
	}

	wantAggregates := make([]causalv2.RuleAggregatePayload, ruleCount)
	aggregateArtifacts := make([]causalv2.Artifact, ruleCount)
	for ruleIndex, code := range rules {
		byRule := make([]causalv2.ApplicationCertificate, seedCount)
		aggregate := causalv2.RuleAggregatePayload{Code: code}
		for seedIndex := 0; seedIndex < seedCount; seedIndex++ {
			certificate := certificates[seedIndex*ruleCount+ruleIndex]
			byRule[seedIndex] = certificate
			aggregate.Applications++
			aggregate.TotalScore += certificate.Score
			aggregate.TotalCost += certificate.Cost
			switch certificate.Terminal {
			case "identified":
				aggregate.Identified++
			case "equivalence":
				aggregate.Equivalence++
			case "budget-exhausted":
				aggregate.BudgetExhausted++
			}
			if result.CreditEnabled {
				aggregate.Worth += profile.Manifest.InvalidOrExhaustedScore - certificate.Score
			}
		}
		aggregate.ApplicationDigest, err = causalv2.Digest(causalv2.RuleApplicationsDomain, byRule)
		if err != nil {
			return err
		}
		artifact, nextErr := next("aggregate")
		if nextErr != nil {
			return nextErr
		}
		if err := requirePayload(artifact, aggregate); err != nil {
			return fmt.Errorf("aggregate %s: %w", code, err)
		}
		wantAggregates[ruleIndex], aggregateArtifacts[ruleIndex] = aggregate, artifact
	}
	if !reflect.DeepEqual(result.Aggregates, wantAggregates) {
		return errors.New("reported aggregates differ from certificate reconstruction")
	}

	tieArtifacts := make([]causalv2.Artifact, 0, ruleCount)
	selectionArtifact := causalv2.Artifact{}
	if result.CreditEnabled {
		best := 0
		for index := 1; index < len(wantAggregates); index++ {
			candidate, incumbent := wantAggregates[index], wantAggregates[best]
			if candidate.Worth > incumbent.Worth || (candidate.Worth == incumbent.Worth && (candidate.BudgetExhausted < incumbent.BudgetExhausted || (candidate.BudgetExhausted == incumbent.BudgetExhausted && candidate.Code < incumbent.Code))) {
				best = index
			}
		}
		winner := wantAggregates[best]
		wantTies := make([]string, 0, ruleCount)
		for index, aggregate := range wantAggregates {
			if aggregate.Worth != winner.Worth || aggregate.BudgetExhausted != winner.BudgetExhausted {
				continue
			}
			artifact, nextErr := next("central-tie")
			if nextErr != nil {
				return nextErr
			}
			want := causalv2.CentralTiePayload{RuleCode: aggregate.Code, AggregateArtifactDigest: aggregateArtifacts[index].ArtifactDigest}
			if err := requirePayload(artifact, want); err != nil {
				return err
			}
			wantTies = append(wantTies, aggregate.Code)
			tieArtifacts = append(tieArtifacts, artifact)
		}
		if !reflect.DeepEqual(result.WinnerTies, wantTies) || result.SelectedRule != winner.Code || result.Unresolved {
			return errors.New("reported winner ties or selected rule are invalid")
		}
		selectionArtifact, err = next("central-selection")
		if err != nil {
			return err
		}
		tieDigests := make([]string, len(tieArtifacts))
		for index, artifact := range tieArtifacts {
			tieDigests[index] = artifact.ArtifactDigest
		}
		if err := requirePayload(selectionArtifact, causalv2.CentralSelectionPayload{SelectedRule: winner.Code, TieArtifactDigests: tieDigests}); err != nil {
			return err
		}
	} else if len(result.WinnerTies) != 0 || result.SelectedRule != "" || !result.Unresolved {
		return errors.New("no-credit result did not remain unresolved")
	}

	wantTranscript := make([]causalv2.CentralTranscriptEvent, 0, 521)
	if result.CreditEnabled {
		subjects := append(append([]causalv2.Artifact(nil), creditArtifacts...), aggregateArtifacts...)
		subjects = append(subjects, selectionArtifact)
		previous := causalv2.ZeroDigest
		work := result.Counts.TotalWork - int64(len(subjects)*17)
		if work < 0 {
			return errors.New("transcript work exceeds total work")
		}
		for index, subject := range subjects {
			artifact, nextErr := next("transcript")
			if nextErr != nil {
				return nextErr
			}
			event, decodeErr := causalv2.StrictDecode[causalv2.CentralTranscriptEvent](artifact.Payload)
			if decodeErr != nil {
				return decodeErr
			}
			kind := "admission"
			if index >= certificateCount && index < certificateCount+ruleCount {
				kind = "aggregate"
			} else if index == certificateCount+ruleCount {
				kind = "selection"
			}
			if event.Index != index || event.PreviousDigest != previous || event.Kind != kind || event.SubjectArtifactDigest != subject.ArtifactDigest || event.WorkBefore != work || event.WorkAfter != work+17 {
				return fmt.Errorf("transcript event %d differs from reconstruction", index)
			}
			wantTranscript = append(wantTranscript, event)
			previous, work = event.EventDigest, event.WorkAfter
		}
		if result.TerminalTranscriptDigest != previous {
			return errors.New("terminal transcript digest mismatch")
		}
	} else if result.TerminalTranscriptDigest != causalv2.ZeroDigest {
		return errors.New("no-credit terminal transcript digest is not the empty chain")
	}
	if position != len(artifacts) {
		return fmt.Errorf("%d trailing central artifacts", len(artifacts)-position)
	}
	if len(artifacts) > 2083 || (!result.CreditEnabled && len(artifacts) != 1041) {
		return errors.New("central artifact ledger cardinality is invalid")
	}
	if !reflect.DeepEqual(result.Transcript, wantTranscript) {
		return errors.New("reported transcript differs from artifact reconstruction")
	}
	return verifyMeters(result)
}

func requirePayload(artifact causalv2.Artifact, want any) error {
	encoded, err := causalv2.CanonicalJSON(want)
	if err != nil {
		return err
	}
	if !bytes.Equal(artifact.Payload, encoded) {
		return errors.New("artifact payload differs from reconstruction")
	}
	return nil
}

func verifyMeters(result Result) error {
	if len(result.TaskMeterItems) != taskCount {
		return fmt.Errorf("curriculum task meter items=%d, want %d", len(result.TaskMeterItems), taskCount)
	}
	digest, err := causalv2.TaskMeterItemsDigest(result.TaskMeterItems)
	if err != nil {
		return err
	}
	if result.TaskMeterItemsDigest != digest {
		return errors.New("curriculum task meter digest mismatch")
	}
	var totals [15]int64
	for index, item := range result.TaskMeterItems {
		wantSubject := fmt.Sprintf("%06d:", index+1)
		if item.Name != "curriculum" || !bytes.HasPrefix([]byte(item.Subject), []byte(wantSubject)) {
			return fmt.Errorf("curriculum task meter item %d has invalid identity", index)
		}
		for index, count := range item.Counts {
			totals[index] += count
		}
	}
	wantCounts := causalv2.CounterFromCounts(totals)
	if result.Counts != wantCounts {
		return errors.New("curriculum total meter differs from task reconstruction")
	}
	manifest := causalv2.PreregisteredManifest()
	if result.Counts.TotalWork > int64(manifest.CurriculumSemanticWorkCap) || result.Counts.AttributedUnits > int64(manifest.CurriculumAttributedUnitCap) || result.Counts.EngineCycles > int64(manifest.CurriculumEngineCycleCap) {
		return errors.New("curriculum meter exceeds a preregistered cap")
	}
	return nil
}
