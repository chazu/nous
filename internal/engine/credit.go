package engine

import (
	"fmt"
	"strings"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

// punishCreators halves the Worth of every unit listed in the target's creditors slot.
// Uses the snapshot since the unit may already be deleted.
func (e *Engine) punishCreators(unitName string, snapshot map[string]any) {
	creditors := extractStringList(snapshot["creditors"])
	for _, creditor := range creditors {
		c := e.Store.Get(creditor)
		if c == nil {
			continue
		}
		oldWorth := c.Worth()
		c.SetWorth(oldWorth / 2)
		e.log(1, "  Credit: halved Worth of %s from %d to %d (created failed unit %s)",
			creditor, oldWorth, c.Worth(), unitName)
	}
}

// rewardCreators boosts the Worth of creditors by the given amount.
func (e *Engine) rewardCreators(unitName string, amount int) {
	u := e.Store.Get(unitName)
	if u == nil {
		return
	}
	for _, creditor := range u.GetStrings("creditors") {
		c := e.Store.Get(creditor)
		if c == nil {
			continue
		}
		oldWorth := c.Worth()
		c.SetWorth(oldWorth + amount)
		e.log(2, "  Credit: boosted Worth of %s from %d to %d (created useful unit %s)",
			creditor, oldWorth, c.Worth(), unitName)
	}
}

// rewardForWorthGrowth rewards creditors when a unit's worth grows above its last rewarded baseline.
// Heuristics are skipped — they are evaluated by applics, not worth growth.
func (e *Engine) rewardForWorthGrowth() {
	for _, name := range e.Store.All() {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		if e.Store.IsA(name, "Heuristic") {
			continue
		}
		creditors := u.GetStrings("creditors")
		if len(creditors) == 0 {
			continue
		}
		if !u.Has("creationWorth") {
			continue
		}
		lastRewarded := u.GetInt("lastRewardedWorth")
		currentWorth := u.Worth()
		if currentWorth > lastRewarded {
			delta := currentWorth - lastRewarded
			reward := delta / 2
			if reward > 0 {
				e.rewardCreators(name, reward)
				u.Set("lastRewardedWorth", currentWorth)
				e.log(2, "  Reward: %s grew %d->%d, rewarding creditors +%d",
					name, lastRewarded, currentWorth, reward)
			}
		}
	}
}

// trackApplics records that a heuristic fired on a unit.
func (e *Engine) trackApplics(heuristicName, targetUnit string, succeeded bool) {
	h := e.Store.Get(heuristicName)
	if h == nil {
		return
	}

	record := h.GetMap("overallRecord")
	if record == nil {
		record = map[string]any{"successes": 0, "failures": 0}
	}
	if succeeded {
		record["successes"] = toInt(record["successes"]) + 1
	} else {
		record["failures"] = toInt(record["failures"]) + 1
	}
	h.Set("overallRecord", record)

	applic := map[string]any{
		"taskNum": e.TaskNum,
		"target":  targetUnit,
		"result":  succeeded,
	}
	applics, _ := h.Get("applics").([]map[string]any)
	applics = append(applics, applic)
	if len(applics) > 50 {
		applics = applics[len(applics)-50:]
	}
	h.Set("applics", applics)
}

// HandleDeletedUnit processes credit assignment and HindSight when a unit is killed.
func (e *Engine) HandleDeletedUnit(unitName string) {
	snapshot := e.VM.DeletedSnapshots[unitName]
	if snapshot == nil {
		return
	}

	// Record in graveyard
	creditors := extractStringList(snapshot["creditors"])
	isA := extractStringList(snapshot["isA"])
	worth := toInt(snapshot["worth"])

	grave := GraveRecord{
		Name:      unitName,
		IsA:       isA,
		Creditors: creditors,
		Worth:     worth,
		Slots:     snapshot,
		Cycle:     e.cycle,
	}
	e.Graveyard = append(e.Graveyard, grave)

	e.log(1, "  Unit %s killed (was worth %d, creditors: %v)", unitName, worth, creditors)

	// Drop any pending tasks targeting this unit — they'd fire heuristics on
	// a non-existent unit, whose get-slot returns nil for every slot and
	// retriggers nil-matching guards in a tight loop.
	if n := e.Agenda.PurgeUnit(unitName); n > 0 {
		e.log(2, "  Purged %d orphan tasks for %s", n, unitName)
	}

	// Punish creditors
	e.punishCreators(unitName, snapshot)

	// Record deferred failure in each creditor's applics
	for _, creditor := range creditors {
		if e.Store.Get(creditor) != nil {
			e.trackApplics(creditor, unitName, false)
		}
	}

	// HindSight: create an avoidance heuristic
	e.createAvoidanceRule(grave)
}

// createAvoidanceRule generates an HAvoid heuristic that prevents the same
// class of failure. This is the nous equivalent of EURISKO's H-HindSight.
//
// The avoidance rule examines what created the failed unit and generates a
// condition that blocks similar creations in the future.
func (e *Engine) createAvoidanceRule(grave GraveRecord) {
	if len(grave.Creditors) == 0 {
		return // can't learn from units without provenance
	}

	// Build the avoidance heuristic name
	avoidName := "HAvoid-" + sanitizeName(grave.Name)
	if e.Store.Has(avoidName) {
		return // already learned this lesson
	}

	// What type was the failed unit?
	failedType := "Anything"
	if len(grave.IsA) > 0 {
		failedType = grave.IsA[0]
	}

	// The avoidance condition: if the creditor heuristic is about to fire
	// and would create something of the same type, check if it's similar
	// to what failed before.
	//
	// For now, the avoidance rule simply prevents the creditor from creating
	// units of the same type. This is crude but functional — EURISKO started
	// here too and refined avoidance rules through mutation.
	creditor := grave.Creditors[0]

	// Build the condition: "If we're working on a task that the creditor
	// would normally fire on, and the target is the same type as what
	// failed, abort."
	ifProg := fmt.Sprintf(`
		"ArgU" @ "%s" isa?
		"ArgU" @ "creditors" get-slot nil !=
		and
		if
			"ArgU" @ "creditors" get-slot
			each
				it "cred" !
				"cred" @ "%s" =
				if
					# Same creditor that made the failed unit — abort
					abort
				then
			end
		then
		false
	`, failedType, creditor)

	tokens := dsl.Tokenize(ifProg)
	if len(tokens) == 0 {
		e.log(1, "  HindSight: discarded invalid HAvoid program for %s", grave.Name)
		return
	}

	avoid := unit.New(avoidName)
	avoid.SetWorth(300) // avoidance rules start unproven
	avoid.Set("isA", []string{"Heuristic", "HAvoidRule", "Anything"})
	avoid.Set("english", fmt.Sprintf("Avoid: %s creating %s-type units (learned from %s dying)",
		creditor, failedType, grave.Name))
	avoid.Set("creditors", []string{"H-HindSight"})
	avoid.Set("ifPotentiallyRelevant", ifProg)
	avoid.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	avoid.Set("creationCycle", e.cycle)
	// Record the provenance so we can trace why this rule exists
	avoid.Set("avoidance_of", grave.Name)
	avoid.Set("avoidance_creditor", creditor)
	avoid.Set("avoidance_type", failedType)

	e.Store.Put(avoid)
	e.log(1, "  HindSight: created %s (blocks %s from making %s-type units)",
		avoidName, creditor, failedType)
}

// promoteOrDemoteHAvoidRules evaluates HAvoid rules periodically,
// promoting proven ones and demoting idle ones.
func (e *Engine) promoteOrDemoteHAvoidRules() {
	for _, name := range e.Store.Examples("HAvoidRule") {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		record := u.GetMap("overallRecord")
		if record == nil {
			continue
		}
		successes := toInt(record["successes"])
		creationCycle := u.GetInt("creationCycle")
		age := e.cycle - creationCycle

		// Promote: 3+ successful firings
		if successes >= 3 && u.Worth() < 600 {
			u.SetWorth(600)
			e.log(1, "  HAvoid: promoted %s to 600 (proven useful, %d firings)", name, successes)
			continue
		}

		// Demote: 200+ cycles with zero firings
		total := successes + toInt(record["failures"])
		if age >= 200 && total == 0 && u.Worth() > 100 {
			u.SetWorth(100)
			e.log(1, "  HAvoid: demoted %s to 100 (idle for %d cycles)", name, age)
		}
	}
}

func sanitizeName(name string) string {
	// Replace characters that would be problematic in unit names
	r := strings.NewReplacer(
		" ", "-",
		"/", "-",
		".", "-",
	)
	return r.Replace(name)
}

// extractStringList handles both []string and single string values.
func extractStringList(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	default:
		return nil
	}
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
}

// DumpWorths prints all units and their worths, sorted by worth descending.
func (e *Engine) DumpWorths() {
	type entry struct {
		name  string
		worth int
	}
	var entries []entry
	for _, name := range e.Store.All() {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		entries = append(entries, entry{name, u.Worth()})
	}
	// Sort by worth descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].worth > entries[i].worth {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	fmt.Fprintf(e.Out, "\n--- Unit Worths ---\n")
	for _, ent := range entries {
		fmt.Fprintf(e.Out, "  %4d  %s\n", ent.worth, ent.name)
	}
}
