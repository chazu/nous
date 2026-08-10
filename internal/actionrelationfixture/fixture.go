package actionrelationfixture

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

var cellNameBanks = [][]string{{"xa", "xb", "xc"}, {"p", "q", "r"}, {"red", "green", "blue"}, {"u", "v", "w"}}
var actionNameBanks = [][]string{
	{"aa", "ab", "ac", "ad", "ae", "af", "ag", "ah"},
	{"opa", "opb", "opc", "opd", "ope", "opf", "opg", "oph"},
	{"stepa", "stepb", "stepc", "stepd", "stepe", "stepf", "stepg", "steph"},
	{"joba", "jobb", "jobc", "jobd", "jobe", "jobf", "jobg", "jobh"},
}
var preoccupationNames = []string{"AR.Candidate", "AR.Edge", "AR.Observation", "AR.Relation"}

type UtilityView struct {
	Slot                int
	Stratum             string
	Core                actionrelationfixturecore.UtilityCore
	State               actionrelations.State
	CellDeclarations    []actionrelations.Cell
	Actions             []actionrelations.Action
	CellPermutation     []int
	ActionPermutation   []int
	StorePreoccupations []string
	CellBank            int
	ActionBank          int
}

type Curriculum struct {
	Family              int
	WithinFamilyOrdinal int
	Draws               DrawBlock
	Worlds              []UtilityView
}

func BuildCurriculum(context DrawContext) (Curriculum, error) {
	draws, err := PrecommitDraws(context)
	if err != nil {
		return Curriculum{}, err
	}
	family := context.Curriculum % 8
	result := Curriculum{Family: family, WithinFamilyOrdinal: context.Curriculum / 8, Draws: draws}
	catalogs := map[string][]actionrelationfixturecore.UtilityCore{}
	for _, stratum := range []string{actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.Neutral, actionrelationfixturecore.Adverse} {
		catalogs[stratum], err = actionrelationfixturecore.SkeletonCatalog(family, stratum)
		if err != nil || len(catalogs[stratum]) < 2 {
			return Curriculum{}, fmt.Errorf("family %d %s catalog: %w", family, stratum, err)
		}
	}
	for slot := 0; slot < 6; slot++ {
		stratum := []string{actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.Neutral, actionrelationfixturecore.Neutral, actionrelationfixturecore.Adverse, actionrelationfixturecore.Adverse}[slot]
		catalog := catalogs[stratum]
		localSlot := slot % 2
		firstDraw := draws.Draws[(slot-localSlot)*11]
		first, _ := Pick(firstDraw.U64, len(catalog))
		selected := first
		if localSlot == 1 {
			remaining := append(slices.Clone(catalog[:first]), catalog[first+1:]...)
			second, _ := Pick(draws.Draws[slot*11].U64, len(remaining))
			core := remaining[second]
			for index := range catalog {
				if catalog[index].Digest == core.Digest {
					selected = index
					break
				}
			}
		}
		view, err := present(slot, stratum, catalog[selected], draws.Draws[slot*11:slot*11+11])
		if err != nil {
			return Curriculum{}, err
		}
		result.Worlds = append(result.Worlds, view)
	}
	return result, nil
}

func present(slot int, stratum string, core actionrelationfixturecore.UtilityCore, draws []Draw) (UtilityView, error) {
	cellBank, _ := Pick(draws[1].U64, len(cellNameBanks))
	actionBank, _ := Pick(draws[2].U64, len(actionNameBanks))
	cellPermutation := []int{0, 1, 2}
	for offset, k := range []int{2, 1} {
		swap, _ := Pick(draws[3+offset].U64, k+1)
		cellPermutation[k], cellPermutation[swap] = cellPermutation[swap], cellPermutation[k]
	}
	actionPermutation := []int{0, 1, 2, 3, 4, 5}
	for offset, k := range []int{5, 4, 3, 2, 1} {
		swap, _ := Pick(draws[5+offset].U64, k+1)
		actionPermutation[k], actionPermutation[swap] = actionPermutation[swap], actionPermutation[k]
	}
	roleNames := map[string]string{}
	for role := 0; role < 3; role++ {
		roleNames[fmt.Sprintf("c%d", role)] = cellNameBanks[cellBank][role]
	}
	state := actionrelations.State{Events: slices.Clone(core.World.State.Events)}
	cells := make([]actionrelations.Cell, 3)
	for index, cell := range core.World.State.Cells {
		cells[index] = actionrelations.Cell{Name: roleNames[cell.Name], Value: cell.Value}
	}
	cellDeclarations := make([]actionrelations.Cell, 3)
	for index, original := range cellPermutation {
		cellDeclarations[index] = cells[original]
	}
	state.Cells = slices.Clone(cells)
	slices.SortFunc(state.Cells, func(a, b actionrelations.Cell) int { return bytes.Compare([]byte(a.Name), []byte(b.Name)) })
	orderedActions := make([]actionrelations.Action, len(core.World.Occurrences))
	for index, occurrence := range core.World.Occurrences {
		semantic := occurrence.Action
		orderedActions[index] = actionrelations.Action{
			Name: actionNameBanks[actionBank][index], Kind: semantic.Kind, X: roleNames[semantic.XRole],
			Y: roleNames[semantic.YRole], N: semantic.N, Symbol: semantic.Symbol,
		}
	}
	actions := make([]actionrelations.Action, len(orderedActions))
	for index, original := range actionPermutation {
		actions[index] = orderedActions[original]
	}
	world := actionrelations.World{State: state, Actions: actions}
	normalized, err := world.Normalize()
	if err != nil {
		return UtilityView{}, err
	}
	got, _ := normalized.CanonicalJSON()
	if !bytes.Equal(got, core.Canonical) {
		return UtilityView{}, fmt.Errorf("presentation did not preserve semantic core")
	}
	preoccupationCount, _ := Pick(draws[10].U64, 5)
	return UtilityView{
		Slot: slot, Stratum: stratum, Core: core, State: state, CellDeclarations: cellDeclarations, Actions: actions,
		CellPermutation: cellPermutation, ActionPermutation: actionPermutation,
		StorePreoccupations: slices.Clone(preoccupationNames[:preoccupationCount]), CellBank: cellBank, ActionBank: actionBank,
	}, nil
}
