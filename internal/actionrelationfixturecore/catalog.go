package actionrelationfixturecore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const (
	PositiveEffect = "positive-effect"
	Neutral        = "neutral"
	Adverse        = "adverse"
)

var FamilyNames = []string{
	"disjoint-adds", "bounded-shared-adds", "equal-sets", "emit-independent-add",
	"repeated-swap", "emit-independent-transfer", "identical-emits", "embedded-bounded-adds",
}

type UtilityCore struct {
	Family          int
	Stratum         string
	World           actionrelations.NormalizedWorld
	Canonical       []byte
	Digest          string
	ReachableStates int
	ReachableNodes  int
	LatentCommutes  int
	OutsideCommutes int
	StaticFailures  int
	neutral         bool
	noLatentMatch   bool
}

func SkeletonCatalog(family int, stratum string) ([]UtilityCore, error) {
	return SkeletonCatalogMeasured(family, stratum, nil)
}

func SkeletonCatalogMeasured(family int, stratum string, reserve WorkReservation) ([]UtilityCore, error) {
	if family < 0 || family >= len(FamilyNames) || stratum != PositiveEffect && stratum != Neutral && stratum != Adverse {
		return nil, fmt.Errorf("invalid family/stratum")
	}
	actions := skeletonActions(family, stratum)
	occurrences, err := actionrelations.AssignOccurrences(actions)
	if err != nil {
		return nil, fmt.Errorf("assign skeleton occurrences %+v: %w", actions, err)
	}
	seen := map[string]bool{}
	var result []UtilityCore
	for encoded := 0; encoded < 64; encoded++ {
		if err := reserveWork(reserve); err != nil {
			return nil, err
		}
		values := []int{encoded / 16, encoded / 4 % 4, encoded % 4}
		state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: values[0]}, {Name: "c1", Value: values[1]}, {Name: "c2", Value: values[2]}}, Events: []string{}}
		if hasApplicableInertCheck(state, occurrences) {
			continue
		}
		presentationActions := make([]actionrelations.Action, len(actions))
		for index, semantic := range actions {
			presentationActions[index] = actionrelations.Action{Name: fmt.Sprintf("a%d", index), Kind: semantic.Kind, X: semantic.XRole, Y: semantic.YRole, N: semantic.N, Symbol: semantic.Symbol}
		}
		world, err := (actionrelations.World{State: state, Actions: presentationActions}).Normalize()
		if err != nil {
			return nil, fmt.Errorf("normalize skeleton state %v: %w", values, err)
		}
		canonical, err := world.CanonicalJSON()
		if err != nil {
			return nil, fmt.Errorf("canonical skeleton state %v: %w", values, err)
		}
		digestBytes := sha256.Sum256(canonical)
		digest := hex.EncodeToString(digestBytes[:])
		if seen[digest] {
			continue
		}
		seen[digest] = true
		assessment, err := assessUtilityCoreMeasured(family, world, reserve)
		if err != nil {
			return nil, fmt.Errorf("state %v: %w", values, err)
		}
		assessment.Family, assessment.Stratum, assessment.World, assessment.Canonical, assessment.Digest = family, stratum, world, canonical, digest
		if err := reserveWork(reserve); err != nil {
			return nil, err
		}
		if stratumAccepts(family, stratum, assessment) {
			result = append(result, assessment)
		}
	}
	slices.SortFunc(result, func(a, b UtilityCore) int { return bytes.Compare(a.Canonical, b.Canonical) })
	return result, nil
}

func skeletonActions(family int, stratum string) []actionrelations.SemanticAction {
	if stratum == Neutral {
		return []actionrelations.SemanticAction{
			action("claim", "c0", "", 0, ""),
			action("check", "c1", "", 3, ""), action("check", "c1", "", 3, ""), action("check", "c1", "", 3, ""),
			action("check", "c1", "", 3, ""), action("check", "c1", "", 3, ""),
		}
	}
	if stratum == PositiveEffect && family == 7 {
		return []actionrelations.SemanticAction{
			action("set", "c0", "", 0, ""),
			action("add", "c0", "", 1, ""), action("add", "c0", "", 1, ""), action("add", "c0", "", 1, ""), action("add", "c0", "", 1, ""),
			action("check", "c2", "", 3, ""),
		}
	}
	pairFamily := family
	if stratum == Adverse {
		pairFamily = []int{4, 2, 6, 4, 6, 2, 2, 4}[family]
	}
	left, right := motif(pairFamily)
	result := []actionrelations.SemanticAction{left, left, right, right}
	if stratum == Adverse && slices.Contains([]int{0, 3, 5}, family) {
		latentLeft, latentRight := motif(family)
		result = append(result, latentLeft, latentRight)
	} else {
		result = append(result, action("check", "c2", "", 3, ""), action("check", "c2", "", 3, ""))
	}
	return result
}

func motif(family int) (actionrelations.SemanticAction, actionrelations.SemanticAction) {
	switch family {
	case 0:
		return action("add", "c0", "", 1, ""), action("add", "c1", "", 1, "")
	case 1, 7:
		return action("add", "c0", "", 1, ""), action("add", "c0", "", 1, "")
	case 2:
		return action("set", "c0", "", 1, ""), action("set", "c0", "", 1, "")
	case 3:
		return action("emit", "", "", 0, "e1"), action("add", "c0", "", 1, "")
	case 4:
		return action("swap", "c0", "c1", 0, ""), action("swap", "c0", "c1", 0, "")
	case 5:
		return action("emit", "", "", 0, "e2"), action("transfer", "c0", "c1", 1, "")
	case 6:
		return action("emit", "", "", 0, "e0"), action("emit", "", "", 0, "e0")
	default:
		panic("invalid family")
	}
}

func action(kind, x, y string, n int, symbol string) actionrelations.SemanticAction {
	return actionrelations.SemanticAction{Kind: kind, XRole: x, YRole: y, N: n, Symbol: symbol}
}

func hasApplicableInertCheck(state actionrelations.State, occurrences []actionrelations.Occurrence) bool {
	stateJSON, _ := state.CanonicalJSON()
	for _, occurrence := range occurrences {
		if occurrence.Action.Kind != "check" {
			continue
		}
		actionJSON, _ := occurrence.Action.CanonicalJSON()
		transition, _ := actionrelationoracle.Apply(stateJSON, actionJSON)
		if transition.Applicable {
			return true
		}
	}
	return false
}

type reachableNode struct {
	state     actionrelations.State
	remaining []actionrelations.Occurrence
}

func assessUtilityCore(family int, world actionrelations.NormalizedWorld) (UtilityCore, error) {
	return assessUtilityCoreMeasured(family, world, nil)
}

func assessUtilityCoreMeasured(family int, world actionrelations.NormalizedWorld, reserve WorkReservation) (UtilityCore, error) {
	queue := []reachableNode{{state: world.State, remaining: world.Occurrences}}
	seenNodes := map[string]bool{}
	seenStates := map[string]bool{}
	latentRows, outsideRows, staticFailureRows := map[string]bool{}, map[string]bool{}, map[string]bool{}
	neutral := true
	noLatentMatch := true
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if err := reserveWork(reserve); err != nil {
			return UtilityCore{}, err
		}
		key := nodeIdentity(node)
		if seenNodes[key] {
			continue
		}
		seenNodes[key] = true
		stateJSON, _ := node.state.CanonicalJSON()
		seenStates[string(stateJSON)] = true
		type enabledStep struct {
			occurrence actionrelations.Occurrence
			next       actionrelations.State
		}
		var enabled []enabledStep
		for _, occurrence := range node.remaining {
			actionJSON, _ := occurrence.Action.CanonicalJSON()
			transition, err := actionrelationoracle.Apply(stateJSON, actionJSON)
			if err != nil {
				return UtilityCore{}, err
			}
			if transition.Applicable {
				next, err := actionrelations.ParseState(transition.State)
				if err != nil {
					return UtilityCore{}, err
				}
				enabled = append(enabled, enabledStep{occurrence, next})
			}
		}
		if len(enabled) > 1 {
			neutral = false
		}
		for leftIndex := 0; leftIndex < len(node.remaining); leftIndex++ {
			for rightIndex := leftIndex + 1; rightIndex < len(node.remaining); rightIndex++ {
				if err := reserveWork(reserve); err != nil {
					return UtilityCore{}, err
				}
				left, right, err := actionrelations.CanonicalPair(node.remaining[leftIndex], node.remaining[rightIndex])
				if err != nil {
					return UtilityCore{}, fmt.Errorf("pair %d/%d: %w", leftIndex, rightIndex, err)
				}
				leftJSON, _ := left.Action.CanonicalJSON()
				rightJSON, _ := right.Action.CanonicalJSON()
				observation, err := actionrelationoracle.Observe(stateJSON, leftJSON, rightJSON)
				if err != nil {
					return UtilityCore{}, err
				}
				latent, err := latentMatch(family, node.state, left, right)
				if err != nil {
					return UtilityCore{}, fmt.Errorf("latent pair %d/%d: %w", leftIndex, rightIndex, err)
				}
				if latent {
					noLatentMatch = false
				}
				if observation.Label != "commutes" {
					continue
				}
				pairKey := localPairIdentity(node.state, left, right)
				if latent {
					latentRows[pairKey] = true
					leftFacts, _ := actionrelations.Facts(node.state, left)
					rightFacts, _ := actionrelations.Facts(node.state, right)
					static, _ := actionrelations.EvaluateAtom("read-write-disjoint", leftFacts, rightFacts)
					if !static {
						staticFailureRows[pairKey] = true
					}
				} else {
					outsideRows[pairKey] = true
				}
			}
		}
		for _, step := range enabled {
			digest, _ := step.occurrence.Digest()
			queue = append(queue, reachableNode{state: step.next, remaining: removeCoreOccurrence(node.remaining, digest)})
		}
		if len(seenNodes)+len(queue) > 4096 {
			return UtilityCore{}, fmt.Errorf("fixture reachability runaway")
		}
	}
	return UtilityCore{ReachableStates: len(seenStates), ReachableNodes: len(seenNodes), LatentCommutes: len(latentRows), OutsideCommutes: len(outsideRows), StaticFailures: len(staticFailureRows), neutral: neutral, noLatentMatch: noLatentMatch}, nil
}

func stratumAccepts(family int, stratum string, core UtilityCore) bool {
	switch stratum {
	case PositiveEffect:
		overlap := slices.Contains([]int{1, 2, 4, 6, 7}, family)
		return core.LatentCommutes >= 4 && (!overlap || core.StaticFailures >= 4)
	case Neutral:
		return core.neutral && core.noLatentMatch
	case Adverse:
		return core.OutsideCommutes >= 4 && core.noLatentMatch
	default:
		return false
	}
}

func latentMatch(family int, state actionrelations.State, left, right actionrelations.Occurrence) (bool, error) {
	stateJSON, _ := state.CanonicalJSON()
	for _, occurrence := range []actionrelations.Occurrence{left, right} {
		actionJSON, _ := occurrence.Action.CanonicalJSON()
		transition, err := actionrelationoracle.Apply(stateJSON, actionJSON)
		if err != nil || !transition.Applicable {
			return false, err
		}
	}
	a, b := left.Action, right.Action
	samePrimary := a.XRole != "" && a.XRole == b.XRole
	switch family {
	case 0:
		return a.Kind == "add" && b.Kind == "add" && a.XRole != b.XRole, nil
	case 1, 7:
		if a.Kind != "add" || b.Kind != "add" || !samePrimary {
			return false, nil
		}
		value, ok := state.Value(a.XRole)
		return ok && value+a.N+b.N >= 0 && value+a.N+b.N <= 3, nil
	case 2:
		return a.Kind == "set" && b.Kind == "set" && samePrimary && a.N == b.N, nil
	case 3:
		return a.Kind == "add" && b.Kind == "emit" || a.Kind == "emit" && b.Kind == "add", nil
	case 4:
		return a.Kind == "swap" && b.Kind == "swap" && a.XRole == b.XRole && a.YRole == b.YRole, nil
	case 5:
		return a.Kind == "emit" && b.Kind == "transfer" || a.Kind == "transfer" && b.Kind == "emit", nil
	case 6:
		return a.Kind == "emit" && b.Kind == "emit" && a.Symbol == b.Symbol, nil
	default:
		return false, fmt.Errorf("invalid family")
	}
}

func nodeIdentity(node reachableNode) string {
	stateJSON, _ := node.state.CanonicalJSON()
	digests := make([]string, len(node.remaining))
	for index, occurrence := range node.remaining {
		digests[index], _ = occurrence.Digest()
	}
	slices.Sort(digests)
	wire, _ := json.Marshal([]any{json.RawMessage(stateJSON), digests})
	return string(wire)
}

func localPairIdentity(state actionrelations.State, left, right actionrelations.Occurrence) string {
	stateDigest, _ := state.Digest()
	leftDigest, _ := left.Digest()
	rightDigest, _ := right.Digest()
	return stateDigest + leftDigest + rightDigest
}

func removeCoreOccurrence(values []actionrelations.Occurrence, digest string) []actionrelations.Occurrence {
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
