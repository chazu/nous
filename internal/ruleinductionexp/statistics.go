package ruleinductionexp

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sort"
)

func contrastRNG(name, operation string) *rand.Rand {
	material := "rule-induction/v1|locked|" + name + "|" + operation
	sum := sha256.Sum256([]byte(material))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16])))
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func pairedContrast(name, statistic string, treatment, control []float64, minimum float64) ContrastReport {
	controlName := map[string]string{"direct": string(LFFDirect), "task-local": string(TaskLocal), "recomputed": string(SharedRecomputed), "inlined": string(SharedInlined), "beneficial-candidates": string(LFFDirect)}[name]
	result := ContrastReport{Treatment: string(SharedLibrary), Control: controlName, Statistic: statistic, RandomizationReplicates: 10000, BootstrapReplicates: 10000, MinimumEffect: minimum}
	if len(treatment) == 0 || len(treatment) != len(control) || mean(control) == 0 {
		return result
	}
	result.RelativeReduction = 1 - mean(treatment)/mean(control)
	result.MeanDifference = mean(control) - mean(treatment)
	differences := make([]float64, len(treatment))
	for index := range treatment {
		differences[index] = control[index] - treatment[index]
	}
	observed := math.Abs(mean(differences))
	rng := contrastRNG(name, "randomization")
	extreme := 0
	for replicate := 0; replicate < result.RandomizationReplicates; replicate++ {
		total := 0.0
		for _, difference := range differences {
			if rng.IntN(2) == 0 {
				total -= difference
			} else {
				total += difference
			}
		}
		if math.Abs(total/float64(len(differences))) >= observed {
			extreme++
		}
	}
	result.PValue = float64(1+extreme) / float64(1+result.RandomizationReplicates)
	rng = contrastRNG(name, "bootstrap")
	bootstrap := make([]float64, result.BootstrapReplicates)
	for replicate := range bootstrap {
		treatmentTotal, controlTotal := 0.0, 0.0
		for range treatment {
			index := rng.IntN(len(treatment))
			treatmentTotal += treatment[index]
			controlTotal += control[index]
		}
		if controlTotal == 0 {
			return result
		}
		bootstrap[replicate] = 1 - treatmentTotal/controlTotal
	}
	sort.Float64s(bootstrap)
	result.CI95 = [2]float64{bootstrap[249], bootstrap[9749]}
	result.Passed = result.RelativeReduction >= minimum && result.PValue < 0.05 && result.CI95[0] > 0
	return result
}

func policyByName(report *Report, name Policy) *PolicyReport {
	for index := range report.Policies {
		if report.Policies[index].Name == name {
			return &report.Policies[index]
		}
	}
	return nil
}

func workValues(policy *PolicyReport, cohort Cohort) []float64 {
	var result []float64
	for _, fixture := range policy.Fixtures {
		if cohort == "" || fixture.Cohort == cohort {
			result = append(result, float64(fixture.Work.Total))
		}
	}
	return result
}

func stage2Values(policy *PolicyReport, cohort Cohort) []float64 {
	var result []float64
	for _, fixture := range policy.Fixtures {
		if cohort == "" || fixture.Cohort == cohort {
			result = append(result, float64(fixture.Stage2Candidates))
		}
	}
	return result
}

func inlinedAccuracyAndScheduleEqual(shared, inlined *PolicyReport) (bool, bool) {
	if len(shared.Fixtures) != len(inlined.Fixtures) {
		return false, false
	}
	accuracyEqual, scheduleEqual := true, true
	for index := range shared.Fixtures {
		a, b := shared.Fixtures[index], inlined.Fixtures[index]
		if a.Seed != b.Seed || a.Cohort != b.Cohort || a.HeldOutCorrect != b.HeldOutCorrect || a.HeldOutTotal != b.HeldOutTotal || a.Accuracy != b.Accuracy {
			accuracyEqual = false
		}
		if a.Seed != b.Seed || a.Cohort != b.Cohort || a.Terminal != b.Terminal || a.Stage1Definition != b.Stage1Definition || a.Stage2Definition != b.Stage2Definition || a.CandidatesConsumed != b.CandidatesConsumed || a.CandidatesExecuted != b.CandidatesExecuted || a.CandidatesPruned != b.CandidatesPruned || a.Stage1Candidates != b.Stage1Candidates || a.Stage2Candidates != b.Stage2Candidates {
			scheduleEqual = false
		}
	}
	return accuracyEqual, scheduleEqual
}

func computeContrasts(report *Report) {
	shared, direct, taskLocal, recomputed, inlined := policyByName(report, SharedLibrary), policyByName(report, LFFDirect), policyByName(report, TaskLocal), policyByName(report, SharedRecomputed), policyByName(report, SharedInlined)
	report.Contrasts = []ContrastReport{}
	if shared == nil || direct == nil || taskLocal == nil || recomputed == nil || inlined == nil {
		return
	}
	report.Contrasts = append(report.Contrasts,
		pairedContrast("direct", "work-ratio-of-means", workValues(shared, ""), workValues(direct, ""), report.Manifest.MinimumPrimaryReduction),
		pairedContrast("task-local", "work-ratio-of-means", workValues(shared, ""), workValues(taskLocal, ""), report.Manifest.MinimumPrimaryReduction),
		pairedContrast("recomputed", "work-ratio-of-means", workValues(shared, ""), workValues(recomputed, ""), report.Manifest.MinimumRecomputedReduction),
		pairedContrast("inlined", "work-ratio-of-means", workValues(shared, ""), workValues(inlined, ""), report.Manifest.MinimumInlinedReduction),
		pairedContrast("beneficial-candidates", "candidate-ratio-of-means", stage2Values(shared, Beneficial), stage2Values(direct, Beneficial), report.Manifest.MinimumBeneficialCandidateReduction),
	)
	directContrast, taskLocalContrast, recomputedContrast, inlinedContrast, beneficialContrast := report.Contrasts[0], report.Contrasts[1], report.Contrasts[2], report.Contrasts[3], report.Contrasts[4]
	report.Gates.Accuracy = shared.Overall.Accuracy == report.Manifest.LockedAccuracyGate
	report.Gates.DirectReduction, report.Gates.DirectPValue, report.Gates.DirectCI = directContrast.RelativeReduction >= report.Manifest.MinimumPrimaryReduction, directContrast.PValue < report.Manifest.Alpha, directContrast.CI95[0] > 0
	report.Gates.TaskLocalReduction, report.Gates.TaskLocalPValue, report.Gates.TaskLocalCI = taskLocalContrast.RelativeReduction >= report.Manifest.MinimumPrimaryReduction, taskLocalContrast.PValue < report.Manifest.Alpha, taskLocalContrast.CI95[0] > 0
	report.Gates.RecomputedReduction, report.Gates.RecomputedPValue, report.Gates.RecomputedCI = recomputedContrast.RelativeReduction >= report.Manifest.MinimumRecomputedReduction, recomputedContrast.PValue < report.Manifest.Alpha, recomputedContrast.CI95[0] > 0
	report.Gates.InlinedAccuracyEqual, report.Gates.InlinedCandidateScheduleEqual = inlinedAccuracyAndScheduleEqual(shared, inlined)
	report.Gates.InlinedReduction, report.Gates.InlinedPValue, report.Gates.InlinedCI = inlinedContrast.RelativeReduction >= report.Manifest.MinimumInlinedReduction, inlinedContrast.PValue < report.Manifest.Alpha, inlinedContrast.CI95[0] > 0
	report.Gates.BeneficialSearch, report.Gates.BeneficialSearchPValue, report.Gates.BeneficialSearchCI = beneficialContrast.RelativeReduction >= report.Manifest.MinimumBeneficialCandidateReduction, beneficialContrast.PValue < report.Manifest.Alpha, beneficialContrast.CI95[0] > 0
	sharedHarmful, directHarmful := workValues(shared, Harmful), workValues(direct, Harmful)
	report.Gates.HarmfulRatio = len(sharedHarmful) > 0 && mean(directHarmful) > 0 && mean(sharedHarmful)/mean(directHarmful) <= report.Manifest.MaximumHarmfulRatio
}
