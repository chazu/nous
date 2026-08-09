package transformexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sort"
)

type transformPower struct {
	Passing    int
	Replicates int
	Authorized bool
}

func estimateTransformPower(rows []pairedTransformRow, outerReplicates, innerReplicates int) (transformPower, error) {
	if outerReplicates < 1 || innerReplicates < 40 {
		return transformPower{}, fmt.Errorf("invalid power replicate counts")
	}
	byFamily := make([][]pairedTransformRow, 9)
	for _, row := range rows {
		if row.Family < 0 || row.Family >= len(byFamily) {
			return transformPower{}, fmt.Errorf("invalid power row family")
		}
		byFamily[row.Family] = append(byFamily[row.Family], row)
	}
	for family, members := range byFamily {
		if len(members) == 0 {
			return transformPower{}, fmt.Errorf("empty power family %d", family)
		}
	}
	counts := []int{15, 15, 14, 14, 14, 14, 14, 14, 14}
	result := transformPower{Replicates: outerReplicates}
	for outer := range outerReplicates {
		rng := transformPowerRNG(outer, -1, "panel")
		panel := make([]pairedTransformRow, 0, 128)
		for family, count := range counts {
			for range count {
				row := byFamily[family][rng.Uint64N(uint64(len(byFamily[family])))]
				row.Ordinal = len(panel)
				panel = append(panel, row)
			}
		}
		passed, err := transformSyntheticPanelPasses(panel, outer, innerReplicates)
		if err != nil {
			return transformPower{}, err
		}
		if passed {
			result.Passing++
		}
	}
	result.Authorized = result.Passing*100 >= 80*result.Replicates
	return result, nil
}

func transformSyntheticPanelPasses(rows []pairedTransformRow, outer, replicates int) (bool, error) {
	if len(rows) != 128 {
		return false, fmt.Errorf("synthetic panel size %d", len(rows))
	}
	byFamily := make([][]pairedTransformRow, 9)
	var observed int
	nousSuccesses, falseApplications := 0, 0
	var nousWork, pbeWork int64
	for _, row := range rows {
		byFamily[row.Family] = append(byFamily[row.Family], row)
		observed += boolInt(row.NousSuccess) - boolInt(row.PBESuccess)
		nousSuccesses += boolInt(row.NousSuccess)
		falseApplications += row.FalseApplications
		nousWork += row.NonmatchingNousWork
		pbeWork += row.NonmatchingPBEWork
	}
	if nousSuccesses*100 < 80*len(rows) || observed*10 < len(rows) || falseApplications != 0 || pbeWork == 0 || nousWork*4 > pbeWork*5 {
		return false, nil
	}
	bootstrap := make([]int, replicates)
	for replicate := range replicates {
		rng := transformPowerRNG(outer, replicate, "bootstrap")
		for family := range byFamily {
			members := byFamily[family]
			for range members {
				row := members[rng.Uint64N(uint64(len(members)))]
				bootstrap[replicate] += boolInt(row.NousSuccess) - boolInt(row.PBESuccess)
			}
		}
	}
	sort.Ints(bootstrap)
	lowerIndex := replicates * 25 / 1000
	if replicates == 2000 {
		lowerIndex = 49
	}
	if bootstrap[lowerIndex] <= 0 {
		return false, nil
	}
	extreme := 0
	for replicate := range replicates {
		rng := transformPowerRNG(outer, replicate, "randomization")
		sample := 0
		for _, row := range rows {
			difference := boolInt(row.NousSuccess) - boolInt(row.PBESuccess)
			if rng.Uint64N(2) == 1 {
				difference = -difference
			}
			sample += difference
		}
		if absInt(sample) >= absInt(observed) {
			extreme++
		}
	}
	return (1+extreme)*100 < 5*(1+replicates), nil
}

func transformPowerRNG(outer, inner int, purpose string) *rand.Rand {
	var preimage []byte
	if purpose == "panel" {
		preimage, _ = json.Marshal([]any{"part3/transform-schema/v1", "power", 841001, outer, purpose})
	} else {
		preimage, _ = json.Marshal([]any{"part3/transform-schema/v1", "power", 841001, outer, inner, purpose})
	}
	digest := sha256.Sum256(preimage)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
