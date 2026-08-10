package actionrelationfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const maximumTruthShardBytes = 65536

type PairLabelRow struct {
	StateDigest string
	ADigest     string
	BDigest     string
	Label       string
}

type TruthShard struct {
	WorldDigest string
	Ordinal     int
	Count       int
	Terminals   []string
	PairRows    []PairLabelRow
	Canonical   []byte
	Digest      string
}

type WorldTruth struct {
	WorldDigest string
	Terminals   []string
	PairRows    []PairLabelRow
	Shards      []TruthShard
}

type CurriculumTruth struct {
	Worlds []WorldTruth
	Root   string
}

func SealCurriculumTruth(curriculum Curriculum) (CurriculumTruth, error) {
	if len(curriculum.Worlds) != 6 {
		return CurriculumTruth{}, fmt.Errorf("curriculum truth requires six worlds")
	}
	result := CurriculumTruth{Worlds: make([]WorldTruth, 6)}
	type shardReference struct {
		worldDigest string
		ordinal     int
		digest      string
	}
	var references []shardReference
	for slot, view := range curriculum.Worlds {
		world, err := (actionrelations.World{State: view.State, Actions: view.Actions}).Normalize()
		if err != nil || worldDigest(world) != view.Core.Digest {
			return CurriculumTruth{}, fmt.Errorf("truth world %d changed semantic core", slot)
		}
		truth, err := SealWorldTruth(world)
		if err != nil {
			return CurriculumTruth{}, fmt.Errorf("truth world %d: %w", slot, err)
		}
		result.Worlds[slot] = truth
		for _, shard := range truth.Shards {
			references = append(references, shardReference{worldDigest: truth.WorldDigest, ordinal: shard.Ordinal, digest: shard.Digest})
		}
	}
	slices.SortFunc(references, func(a, b shardReference) int {
		if value := bytes.Compare(mustDigest(a.worldDigest), mustDigest(b.worldDigest)); value != 0 {
			return value
		}
		return a.ordinal - b.ordinal
	})
	rootRows := make([]any, len(references))
	for index, reference := range references {
		rootRows[index] = []any{reference.worldDigest, reference.ordinal, reference.digest}
	}
	root, err := actionrelationwire.RootDigest("scorer-shards", rootRows)
	if err != nil {
		return CurriculumTruth{}, err
	}
	result.Root = root
	return result, nil
}

func SealWorldTruth(world actionrelations.NormalizedWorld) (WorldTruth, error) {
	canonical, err := world.CanonicalJSON()
	if err != nil {
		return WorldTruth{}, err
	}
	digestBytes := sha256.Sum256(canonical)
	worldDigest := hex.EncodeToString(digestBytes[:])
	complete, err := actionrelationsearch.Search(actionrelations.World{State: world.State, Actions: presentationActions(world.Actions)}, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil {
		return WorldTruth{}, err
	}
	terminals := slices.Clone(complete.TerminalDigests)
	slices.Sort(terminals)
	terminals = slices.Compact(terminals)
	rows, err := enumeratePairLabels(world)
	if err != nil {
		return WorldTruth{}, err
	}
	shards, err := buildTruthShards(worldDigest, terminals, rows)
	if err != nil {
		return WorldTruth{}, err
	}
	result := WorldTruth{WorldDigest: worldDigest, Terminals: terminals, PairRows: rows, Shards: shards}
	if err := VerifyWorldTruth(result); err != nil {
		return WorldTruth{}, err
	}
	return result, nil
}

func VerifyWorldTruth(truth WorldTruth) error {
	if !digestText(truth.WorldDigest) || !sortedUniqueDigests(truth.Terminals) || len(truth.PairRows) == 0 || len(truth.Shards) == 0 {
		return fmt.Errorf("invalid world truth authority")
	}
	for index, row := range truth.PairRows {
		if !validPairLabel(row) || index > 0 && comparePairRows(truth.PairRows[index-1], row) >= 0 {
			return fmt.Errorf("invalid pair-label row %d", index)
		}
	}
	var rebuilt []PairLabelRow
	for ordinal, shard := range truth.Shards {
		if shard.Ordinal != ordinal || shard.Count != len(truth.Shards) || shard.WorldDigest != truth.WorldDigest || !slices.Equal(shard.Terminals, truth.Terminals) || len(shard.PairRows) == 0 || len(shard.Canonical) > maximumTruthShardBytes || shard.Digest != shaHex(shard.Canonical) {
			return fmt.Errorf("invalid truth shard %d", ordinal)
		}
		want, _ := truthShardWire(shard.WorldDigest, shard.Ordinal, shard.Count, shard.Terminals, shard.PairRows)
		if !bytes.Equal(want, shard.Canonical) {
			return fmt.Errorf("truth shard %d changed wire", ordinal)
		}
		rebuilt = append(rebuilt, shard.PairRows...)
	}
	if !slices.Equal(rebuilt, truth.PairRows) {
		return fmt.Errorf("truth shards do not partition pair rows")
	}
	expected, err := buildTruthShards(truth.WorldDigest, truth.Terminals, truth.PairRows)
	if err != nil || len(expected) != len(truth.Shards) {
		return fmt.Errorf("truth shards are not greedily maximal")
	}
	for ordinal := range expected {
		if !bytes.Equal(expected[ordinal].Canonical, truth.Shards[ordinal].Canonical) {
			return fmt.Errorf("truth shard %d is not greedily maximal", ordinal)
		}
	}
	return nil
}

type truthNode struct {
	state     actionrelations.State
	remaining []actionrelations.Occurrence
}

func enumeratePairLabels(world actionrelations.NormalizedWorld) ([]PairLabelRow, error) {
	queue := []truthNode{{state: world.State, remaining: world.Occurrences}}
	seenNodes := map[string]bool{}
	rows := map[string]PairLabelRow{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		identity := truthNodeIdentity(node)
		if seenNodes[identity] {
			continue
		}
		seenNodes[identity] = true
		stateJSON, _ := node.state.CanonicalJSON()
		stateDigest, _ := node.state.Digest()
		for leftIndex := 0; leftIndex < len(node.remaining); leftIndex++ {
			for rightIndex := leftIndex + 1; rightIndex < len(node.remaining); rightIndex++ {
				left, right, err := actionrelations.CanonicalPair(node.remaining[leftIndex], node.remaining[rightIndex])
				if err != nil {
					return nil, err
				}
				leftJSON, _ := left.Action.CanonicalJSON()
				rightJSON, _ := right.Action.CanonicalJSON()
				observation, err := actionrelationoracle.Observe(stateJSON, leftJSON, rightJSON)
				if err != nil {
					return nil, err
				}
				aDigest, _ := left.Digest()
				bDigest, _ := right.Digest()
				label := observation.Label
				if aDigest > bDigest {
					aDigest, bDigest = bDigest, aDigest
					label = reversePairLabel(label)
				}
				row := PairLabelRow{StateDigest: stateDigest, ADigest: aDigest, BDigest: bDigest, Label: label}
				rows[stateDigest+aDigest+bDigest] = row
			}
		}
		for _, occurrence := range node.remaining {
			next, outcome, err := actionrelations.Apply(node.state, occurrence.Action)
			if err != nil {
				return nil, err
			}
			if outcome != "applied" {
				continue
			}
			digest, _ := occurrence.Digest()
			queue = append(queue, truthNode{state: next, remaining: removeTruthOccurrence(node.remaining, digest)})
		}
		if len(seenNodes)+len(queue) > 65536 {
			return nil, fmt.Errorf("truth reachability exceeds frozen cap")
		}
	}
	result := make([]PairLabelRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	slices.SortFunc(result, comparePairRows)
	return result, nil
}

func reversePairLabel(label string) string {
	switch label {
	case "a-enables-b":
		return "b-enables-a"
	case "b-enables-a":
		return "a-enables-b"
	case "a-disables-b":
		return "b-disables-a"
	case "b-disables-a":
		return "a-disables-b"
	default:
		return label
	}
}

func buildTruthShards(worldDigest string, terminals []string, rows []PairLabelRow) ([]TruthShard, error) {
	count := 1
	for attempts := 0; attempts < 8; attempts++ {
		var shards []TruthShard
		for first := 0; first < len(rows); {
			last := first + 1
			for last <= len(rows) {
				wire, _ := truthShardWire(worldDigest, len(shards), count, terminals, rows[first:last])
				if len(wire) > maximumTruthShardBytes {
					last--
					break
				}
				last++
			}
			if last > len(rows) {
				last = len(rows)
			}
			if last <= first {
				return nil, fmt.Errorf("one scorer row exceeds shard cap")
			}
			wire, _ := truthShardWire(worldDigest, len(shards), count, terminals, rows[first:last])
			shards = append(shards, TruthShard{WorldDigest: worldDigest, Ordinal: len(shards), Count: count, Terminals: slices.Clone(terminals), PairRows: slices.Clone(rows[first:last]), Canonical: wire, Digest: shaHex(wire)})
			first = last
		}
		if len(shards) == count {
			return shards, nil
		}
		count = len(shards)
	}
	return nil, fmt.Errorf("truth shard count did not stabilize")
}

func truthShardWire(worldDigest string, ordinal, count int, terminals []string, rows []PairLabelRow) ([]byte, error) {
	pairRows := make([]any, len(rows))
	for index, row := range rows {
		pairRows[index] = []any{row.StateDigest, row.ADigest, row.BDigest, row.Label}
	}
	return json.Marshal([]any{"action-scorer-truth-shard/v1", worldDigest, ordinal, count, terminals, pairRows})
}

func comparePairRows(a, b PairLabelRow) int {
	if value := bytes.Compare(mustDigest(a.StateDigest), mustDigest(b.StateDigest)); value != 0 {
		return value
	}
	if value := bytes.Compare(mustDigest(a.ADigest), mustDigest(b.ADigest)); value != 0 {
		return value
	}
	return bytes.Compare(mustDigest(a.BDigest), mustDigest(b.BDigest))
}

func validPairLabel(row PairLabelRow) bool {
	return digestText(row.StateDigest) && digestText(row.ADigest) && digestText(row.BDigest) && row.ADigest < row.BDigest && slices.Contains([]string{"commutes", "conflicts", "a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable"}, row.Label)
}

func truthNodeIdentity(node truthNode) string {
	stateJSON, _ := node.state.CanonicalJSON()
	digests := make([]string, len(node.remaining))
	for index, occurrence := range node.remaining {
		digests[index], _ = occurrence.Digest()
	}
	slices.Sort(digests)
	wire, _ := json.Marshal([]any{json.RawMessage(stateJSON), digests})
	return string(wire)
}

func removeTruthOccurrence(values []actionrelations.Occurrence, digest string) []actionrelations.Occurrence {
	result := make([]actionrelations.Occurrence, 0, len(values)-1)
	removed := false
	for _, value := range values {
		current, _ := value.Digest()
		if !removed && current == digest {
			removed = true
			continue
		}
		result = append(result, value)
	}
	return result
}

func presentationActions(actions []actionrelations.SemanticAction) []actionrelations.Action {
	result := make([]actionrelations.Action, len(actions))
	for index, action := range actions {
		result[index] = actionrelations.Action{Name: fmt.Sprintf("a%d", index), Kind: action.Kind, X: action.XRole, Y: action.YRole, N: action.N, Symbol: action.Symbol}
	}
	return result
}

func worldDigest(world actionrelations.NormalizedWorld) string {
	value, _ := world.Digest()
	return value
}

func sortedUniqueDigests(values []string) bool {
	for index, value := range values {
		if !digestText(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func digestText(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 32 && hex.EncodeToString(raw) == value
}

func mustDigest(value string) []byte {
	raw, _ := hex.DecodeString(value)
	return raw
}

func shaHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
