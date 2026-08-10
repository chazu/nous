package actionrelationfixturecore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type View struct {
	Bank                int
	Canonical           []byte
	Digest              string
	Proof               []byte
	ProofDigest         string
	SemanticWorldDigest string
	OriginalStateDigest string
	OriginalActionsRoot string
	OccurrenceMapRoot   string
	CellCount           int
	ActionCount         int
}

func Views(testCase Case) ([]View, error) {
	state, err := actionrelations.ParseState(testCase.State)
	if err != nil {
		return nil, err
	}
	a, err := actionrelations.ParseOccurrence(testCase.AOccurrence)
	if err != nil {
		return nil, err
	}
	b, err := actionrelations.ParseOccurrence(testCase.BOccurrence)
	if err != nil {
		return nil, err
	}
	result := make([]View, 2)
	for bank, names := range [][]string{{"xa", "xb", "xc"}, {"red", "green", "blue"}} {
		actionNames := [][]string{{"aa", "ab"}, {"joba", "jobb"}}[bank]
		roleToName := map[string]string{}
		for index := range state.Cells {
			roleToName[fmt.Sprintf("c%d", index)] = names[index]
		}
		originalState := actionrelations.State{Events: slices.Clone(state.Events)}
		for _, cell := range state.Cells {
			originalState.Cells = append(originalState.Cells, actionrelations.Cell{Name: roleToName[cell.Name], Value: cell.Value})
		}
		slices.SortFunc(originalState.Cells, func(left, right actionrelations.Cell) int {
			return bytes.Compare([]byte(left.Name), []byte(right.Name))
		})
		occurrences := []actionrelations.Occurrence{a, b}
		actions := make([]actionrelations.Action, 2)
		mapping := make([]any, 2)
		for index, occurrence := range occurrences {
			semantic := occurrence.Action
			actions[index] = actionrelations.Action{Name: actionNames[index], Kind: semantic.Kind, X: roleToName[semantic.XRole], Y: roleToName[semantic.YRole], N: semantic.N, Symbol: semantic.Symbol}
			occurrenceDigest, _ := occurrence.Digest()
			mapping[index] = []any{actionNames[index], occurrenceDigest}
		}
		world := actionrelations.World{State: originalState, Actions: actions}
		normalized, err := world.Normalize()
		if err != nil {
			return nil, err
		}
		core, _ := normalized.CanonicalJSON()
		semanticWorldDigest, _ := normalized.Digest()
		expected := actionrelations.NormalizedWorld{State: state, Actions: []actionrelations.SemanticAction{a.Action, b.Action}}
		expectedCore, _ := expected.CanonicalJSON()
		if !bytes.Equal(core, expectedCore) {
			return nil, fmt.Errorf("presentation wrapper did not normalize to semantic core")
		}
		stateJSON, _ := originalState.CanonicalJSON()
		actionRows := make([]any, 2)
		for index, action := range actions {
			actionJSON, _ := action.CanonicalJSON()
			actionRows[index] = json.RawMessage(actionJSON)
		}
		viewWire := []any{"action-presentation-view/v1", json.RawMessage(stateJSON), actionRows, semanticWorldDigest, mapping}
		viewJSON, _ := json.Marshal(viewWire)
		viewDigest := digestView(viewJSON)
		mapRows := make([]any, 0, len(roleToName))
		for original, role := range invertMap(roleToName) {
			mapRows = append(mapRows, []any{original, role})
		}
		slices.SortFunc(mapRows, func(left, right any) int {
			a, _ := json.Marshal(left)
			b, _ := json.Marshal(right)
			return bytes.Compare(a, b)
		})
		proofJSON, _ := json.Marshal([]any{"action-normalization-proof/v1", viewDigest, mapRows, semanticWorldDigest})
		actionDigests := make([]string, len(actions))
		for index, action := range actions {
			actionJSON, _ := action.CanonicalJSON()
			actionDigests[index] = digestView(actionJSON)
		}
		originalActionsRoot, err := actionrelationwire.RootDigest("original-actions", actionDigests)
		if err != nil {
			return nil, err
		}
		occurrenceMapRoot, err := actionrelationwire.RootDigest("occurrence-map", mapping)
		if err != nil {
			return nil, err
		}
		result[bank] = View{
			Bank: bank, Canonical: viewJSON, Digest: viewDigest, Proof: proofJSON, ProofDigest: digestView(proofJSON), SemanticWorldDigest: semanticWorldDigest,
			OriginalStateDigest: digestView(stateJSON), OriginalActionsRoot: originalActionsRoot, OccurrenceMapRoot: occurrenceMapRoot, CellCount: len(state.Cells), ActionCount: len(actions),
		}
	}
	return result, nil
}

func invertMap(roleToName map[string]string) map[string]string {
	result := map[string]string{}
	for role, name := range roleToName {
		result[name] = role
	}
	return result
}

func digestView(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
