package actionrelationfixture

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"

	"github.com/chazu/nous/internal/actionrelationwire"
)

type DrawContext struct {
	Panel          string
	Authority      string
	Curriculum     int
	CurriculumSeed any
	Attempt        int
}

type Draw struct {
	Ordinal   int
	Namespace string
	Index     int
	U64       uint64
	U64Hex    string
	Canonical []byte
	Digest    string
}

type DrawBlock struct {
	Context DrawContext
	Draws   []Draw
	Root    string
}

func PrecommitDraws(context DrawContext) (DrawBlock, error) {
	if err := validateDrawContext(context); err != nil {
		return DrawBlock{}, err
	}
	var specs []struct {
		namespace string
		index     int
	}
	for slot := 0; slot < 6; slot++ {
		specs = append(specs,
			struct {
				namespace string
				index     int
			}{"skeleton-variant", slot},
			struct {
				namespace string
				index     int
			}{"cell-name-bank", slot},
			struct {
				namespace string
				index     int
			}{"action-name-bank", slot},
		)
		for k := 2; k >= 1; k-- {
			specs = append(specs, struct {
				namespace string
				index     int
			}{"cell-permutation", 8*slot + k})
		}
		for k := 5; k >= 1; k-- {
			specs = append(specs, struct {
				namespace string
				index     int
			}{"action-permutation", 8*slot + k})
		}
		specs = append(specs, struct {
			namespace string
			index     int
		}{"store-preoccupation-count", slot})
	}
	if len(specs) != 66 {
		panic("frozen draw schedule cardinality")
	}
	block := DrawBlock{Context: context, Draws: make([]Draw, len(specs))}
	digests := make([]string, len(specs))
	for ordinal, spec := range specs {
		preimage, _ := json.Marshal([]any{"actionrelation-fixture-draw/v1", context.Panel, context.Authority, context.Curriculum, context.CurriculumSeed, context.Attempt, spec.namespace, spec.index})
		hash := sha256.Sum256(preimage)
		value := binary.BigEndian.Uint64(hash[:8])
		u64Hex := hex.EncodeToString(hash[:8])
		row, _ := json.Marshal([]any{"action-generator-draw/v1", context.Panel, context.Authority, context.Curriculum, context.CurriculumSeed, context.Attempt, ordinal, spec.namespace, spec.index, u64Hex})
		digest := sha256.Sum256(row)
		digests[ordinal] = hex.EncodeToString(digest[:])
		block.Draws[ordinal] = Draw{Ordinal: ordinal, Namespace: spec.namespace, Index: spec.index, U64: value, U64Hex: u64Hex, Canonical: row, Digest: digests[ordinal]}
	}
	block.Root, _ = actionrelationwire.RootDigest("generator-draws", digests)
	return block, nil
}

func Pick(value uint64, n int) (int, error) {
	if n < 1 {
		return 0, fmt.Errorf("invalid pick range")
	}
	high, _ := bits.Mul64(value, uint64(n))
	return int(high), nil
}

func validateDrawContext(context DrawContext) error {
	if context.Curriculum < 0 || context.Attempt < 0 || context.Attempt > 31 {
		return fmt.Errorf("invalid draw ordinal")
	}
	switch context.Panel {
	case "development":
		seed, ok := context.CurriculumSeed.(int)
		if context.Authority != "development-public-v1" || !ok || seed != 851001+context.Curriculum {
			return fmt.Errorf("invalid development draw authority")
		}
	case "validation":
		seed, ok := context.CurriculumSeed.(int)
		if context.Authority != "validation-public-v1" || !ok || seed != 852001+context.Curriculum {
			return fmt.Errorf("invalid validation draw authority")
		}
	case "locked":
		seed, ok := context.CurriculumSeed.(string)
		wantSeed, err := LockedCurriculumSeed(context.Authority, context.Curriculum)
		if err != nil || !ok || seed != wantSeed {
			return fmt.Errorf("invalid locked draw authority")
		}
	default:
		return fmt.Errorf("invalid draw panel")
	}
	return nil
}

func LockedCurriculumSeed(authority string, curriculum int) (string, error) {
	key, err := hex.DecodeString(authority)
	if err != nil || len(key) != 32 || hex.EncodeToString(key) != authority || curriculum < 0 {
		return "", fmt.Errorf("invalid locked root authority")
	}
	preimage, _ := json.Marshal([]any{"actionrelation-locked-curriculum/v1", curriculum})
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(preimage)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func digest(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}
