package causalexpv2

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sort"
)

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

func contrastRNG(name, operation string) *rand.Rand {
	sum := sha256.Sum256([]byte("active-causal-diagnosis/v2|locked|" + name + "|" + operation))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16])))
}

func pairedContrast(name string, treatment, control []float64) Contrast {
	result := Contrast{Name: name, Treatment: "learned", Control: "information-gain-per-cost", Statistic: "score-ratio-of-means", RandomizationReplicates: 10000, BootstrapReplicates: 10000, MinimumEffect: .10}
	if len(treatment) == 0 || len(treatment) != len(control) || mean(control) == 0 {
		return result
	}
	result.RelativeReduction = 1 - mean(treatment)/mean(control)
	result.MeanDifference = mean(control) - mean(treatment)
	differences := make([]float64, len(treatment))
	for i := range treatment {
		differences[i] = control[i] - treatment[i]
	}
	observed := math.Abs(mean(differences))
	random := contrastRNG(name, "randomization")
	extreme := 0
	for replicate := 0; replicate < 10000; replicate++ {
		total := 0.0
		for _, difference := range differences {
			if random.IntN(2) == 0 {
				total -= difference
			} else {
				total += difference
			}
		}
		if math.Abs(total/float64(len(differences))) >= observed {
			extreme++
		}
	}
	result.PValue = float64(1+extreme) / 10001
	random = contrastRNG(name, "bootstrap")
	bootstrap := make([]float64, 10000)
	for replicate := range bootstrap {
		a, b := 0.0, 0.0
		for range treatment {
			index := random.IntN(len(treatment))
			a += treatment[index]
			b += control[index]
		}
		if b == 0 {
			return result
		}
		bootstrap[replicate] = 1 - a/b
	}
	sort.Float64s(bootstrap)
	result.CI95 = [2]float64{bootstrap[249], bootstrap[9749]}
	result.Passed = result.RelativeReduction >= .10 && result.PValue < .05 && result.CI95[0] > 0
	return result
}
