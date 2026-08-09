package transformexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"slices"
	"sort"
)

type pairedTransformRow struct {
	Ordinal             int
	Family              int
	NousSuccess         bool
	PBESuccess          bool
	FalseApplications   int
	NonmatchingNousWork int64
	NonmatchingPBEWork  int64
}

type rationalPoint struct {
	Numerator   int64
	Denominator int64
}

type transformInference struct {
	Point                rationalPoint
	Lower                rationalPoint
	Upper                rationalPoint
	RandomizationExtreme int
	PValue               rationalPoint
	NousSuccesses        int
	PBESuccesses         int
	FalseApplications    int
	NonmatchingNous      int64
	NonmatchingPBE       int64
}

func (i transformInference) MarshalJSON() ([]byte, error) {
	type wire struct {
		PointNumerator       int64 `json:"point_numerator"`
		PointDenominator     int64 `json:"point_denominator"`
		LowerNumerator       int64 `json:"lower_numerator"`
		LowerDenominator     int64 `json:"lower_denominator"`
		UpperNumerator       int64 `json:"upper_numerator"`
		UpperDenominator     int64 `json:"upper_denominator"`
		RandomizationExtreme int   `json:"randomization_extreme"`
		PNumerator           int64 `json:"p_numerator"`
		PDenominator         int64 `json:"p_denominator"`
		NousSuccesses        int   `json:"nous_successes"`
		PBESuccesses         int   `json:"pbe_successes"`
		FalseApplications    int   `json:"false_applications"`
		NonmatchingNous      int64 `json:"nonmatching_nous"`
		NonmatchingPBE       int64 `json:"nonmatching_pbe"`
	}
	return json.Marshal(wire{i.Point.Numerator, i.Point.Denominator, i.Lower.Numerator, i.Lower.Denominator, i.Upper.Numerator, i.Upper.Denominator, i.RandomizationExtreme, i.PValue.Numerator, i.PValue.Denominator, i.NousSuccesses, i.PBESuccesses, i.FalseApplications, i.NonmatchingNous, i.NonmatchingPBE})
}

type indexedPoint struct {
	Point   rationalPoint
	Ordinal int
}

func computeTransformInference(rows []pairedTransformRow, panel string, authority uint64, bootstrapReplicates, randomizationReplicates int) (transformInference, error) {
	return computeTransformInferenceWithPairs(rows, panel, authority, nil, bootstrapReplicates, randomizationReplicates)
}

func computeTransformInferenceWithPairs(rows []pairedTransformRow, panel string, authority uint64, lockedPairs [][2]uint64, bootstrapReplicates, randomizationReplicates int) (transformInference, error) {
	if len(rows) == 0 || bootstrapReplicates < 1 || randomizationReplicates < 1 {
		return transformInference{}, fmt.Errorf("invalid inference dimensions")
	}
	if lockedPairs != nil && len(lockedPairs) != bootstrapReplicates+randomizationReplicates {
		return transformInference{}, fmt.Errorf("locked inference pair count mismatch")
	}
	rows = slices.Clone(rows)
	slices.SortFunc(rows, func(a, b pairedTransformRow) int { return a.Ordinal - b.Ordinal })
	byFamily := make([][]pairedTransformRow, 9)
	result := transformInference{}
	var total int64
	for i, row := range rows {
		if row.Ordinal != i || row.Family < 0 || row.Family >= len(byFamily) || row.FalseApplications < 0 || row.NonmatchingNousWork < 0 || row.NonmatchingPBEWork < 0 {
			return transformInference{}, fmt.Errorf("invalid paired row")
		}
		byFamily[row.Family] = append(byFamily[row.Family], row)
		difference := int64(boolInt(row.NousSuccess) - boolInt(row.PBESuccess))
		total += difference
		result.NousSuccesses += boolInt(row.NousSuccess)
		result.PBESuccesses += boolInt(row.PBESuccess)
		result.FalseApplications += row.FalseApplications
		result.NonmatchingNous += row.NonmatchingNousWork
		result.NonmatchingPBE += row.NonmatchingPBEWork
	}
	for family, members := range byFamily {
		if len(members) == 0 {
			return transformInference{}, fmt.Errorf("empty family stratum %d", family)
		}
	}
	result.Point = rationalPoint{total, int64(len(rows))}
	bootstrap := make([]indexedPoint, bootstrapReplicates)
	for replicate := range bootstrapReplicates {
		rng := transformStatisticsRNGFor(panel, authority, lockedPairs, replicate, "bootstrap/nous-vs-pbe")
		var sample int64
		for family := range byFamily {
			members := byFamily[family]
			for range members {
				row := members[rng.Uint64N(uint64(len(members)))]
				sample += int64(boolInt(row.NousSuccess) - boolInt(row.PBESuccess))
			}
		}
		bootstrap[replicate] = indexedPoint{rationalPoint{sample, int64(len(rows))}, replicate}
	}
	sort.Slice(bootstrap, func(i, j int) bool {
		comparison := compareRational(bootstrap[i].Point, bootstrap[j].Point)
		return comparison < 0 || comparison == 0 && bootstrap[i].Ordinal < bootstrap[j].Ordinal
	})
	lowerIndex, upperIndex := bootstrapIntervalIndices(bootstrapReplicates)
	result.Lower, result.Upper = bootstrap[lowerIndex].Point, bootstrap[upperIndex].Point
	observed := abs64(total)
	for replicate := range randomizationReplicates {
		rng := transformStatisticsRNGFor(panel, authority, lockedPairs, bootstrapReplicates+replicate, "randomization/nous-vs-pbe")
		var sample int64
		for _, row := range rows {
			difference := int64(boolInt(row.NousSuccess) - boolInt(row.PBESuccess))
			if rng.Uint64N(2) == 1 {
				difference = -difference
			}
			sample += difference
		}
		if abs64(sample) >= observed {
			result.RandomizationExtreme++
		}
	}
	result.PValue = rationalPoint{int64(1 + result.RandomizationExtreme), int64(1 + randomizationReplicates)}
	return result, nil
}

func transformStatisticsRNGFor(panel string, authority uint64, lockedPairs [][2]uint64, replicate int, purpose string) *rand.Rand {
	if lockedPairs != nil {
		pair := lockedPairs[replicate]
		return rand.New(rand.NewPCG(pair[0], pair[1]))
	}
	return transformStatisticsRNG(panel, authority, replicate, purpose)
}

func transformStatisticsRNG(panel string, authority uint64, replicate int, purpose string) *rand.Rand {
	preimage, _ := json.Marshal([]any{"part3/transform-schema/v2", "statistics", panel, authority, replicate, purpose})
	digest := sha256.Sum256(preimage)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func lockedStatisticsPairs(rootCommitment string) ([][2]uint64, error) {
	if !isLowerHex(rootCommitment, 64) {
		return nil, fmt.Errorf("invalid locked statistics root commitment")
	}
	pairs := make([][2]uint64, 20000)
	for replicate := 0; replicate < 10000; replicate++ {
		for purposeIndex, purpose := range []string{"bootstrap/nous-vs-pbe", "randomization/nous-vs-pbe"} {
			preimage, _ := json.Marshal([]any{"part3/transform-schema/v2", "statistics", "locked", rootCommitment, replicate, purpose})
			digest := sha256.Sum256(preimage)
			index := replicate + purposeIndex*10000
			pairs[index] = [2]uint64{binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])}
		}
	}
	return pairs, nil
}

func bootstrapIntervalIndices(replicates int) (int, int) {
	if replicates == 10000 {
		return 249, 9749
	}
	lower := replicates * 25 / 1000
	upper := replicates*975/1000 - 1
	if upper < lower {
		upper = lower
	}
	return lower, upper
}

func compareRational(a, b rationalPoint) int {
	left := new(big.Int).Mul(big.NewInt(a.Numerator), big.NewInt(b.Denominator))
	right := new(big.Int).Mul(big.NewInt(b.Numerator), big.NewInt(a.Denominator))
	return left.Cmp(right)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func abs64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
