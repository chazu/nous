# Self-Modification Loop Design

**Date:** 2026-04-14
**Status:** Approved

## Overview

Wire the four disconnected feedback loops in nous's credit assignment and mutation system. The self-modification loop is the core EURISKO differentiator -- without it, nous is a fixed heuristic interpreter. With it, heuristics improve over time by learning from their own successes and failures.

## Decisions

- **Approach:** Bottom-up wiring (A). Build each loop in dependency order, validate each layer before building the next.
- **Failure semantics:** Track firing-but-bad-outcome (B), not non-firing. Two failure signals: immediate (no output produced) and deferred (created unit dies).
- **Reward semantics:** Worth-growth-based (C). Units whose worth increases above creation baseline trigger rewards to their creditors. Future consideration: usage-based reward (B) where units referenced by other heuristics' computations also trigger reward.
- **Applics pattern analysis:** Implemented as a seed meta-heuristic (B), not a Go function. H-AnalyzeApplics is a heuristic subject to its own worth, credit, and mutation -- the recursive property that makes EURISKO interesting.

## The Four Loops

### Loop 1: Accurate Failure Recording + Performance-Based Mutation

#### trackApplics Fixes

Current state: `trackApplics(heuristic, target, succeeded)` is called with `succeeded=true` every time a heuristic fires. No failure data exists.

Two failure signals:

**Immediate failure (no output):** After executing a heuristic's ThenParts in `fireTaskRule`/`fireUnitRule`, check whether anything was produced. "Produced" means: at least one new unit was created OR at least one agenda item was added. Slot modifications alone (e.g., worth adjustments) do not count as output, since many heuristics adjust worth as a side effect.

Detection: snapshot store size and agenda size before ThenParts execution, compare after. If both unchanged, it's a no-op firing.

**Deferred failure (created unit dies):** When `HandleDeletedUnit` fires and traces creditors, record a failure in each creditor heuristic's applics via `trackApplics(creditorHeuristic, deadUnit, false)`. This runs alongside the existing `punishCreators` call. A heuristic that creates units that later die accumulates failure records proportional to its kill rate.

Successful firings that produce output continue to record `succeeded=true`. The `overallRecord` map and rolling 50-item `applics` window are unchanged structurally.

#### Performance-Based Mutation Trigger

Replace the time-based trigger with a performance-based one.

Every `MutConfig.Interval` cycles, scan all heuristics with at least `MutConfig.MinApplics` (default 10) total firings. Compute success ratio from `overallRecord`. Heuristics with success ratio below `MutConfig.MutationThreshold` (default 0.3) are mutation candidates.

If candidates exist, pick the one with the lowest success ratio. On ties, prefer lower worth. If no candidates, skip mutation this interval.

The mutated heuristic is created as a new unit (the original survives). The mutant inherits the original's creditors plus the original itself as a creditor. If the mutant succeeds, the original gets credit for being a good mutation source.

New MutConfig fields:
- `MinApplics int` (default 10)
- `MutationThreshold float64` (default 0.3)

Removed: random heuristic selection weighted by worth. Exploration moves to Loop 3 (H-AnalyzeApplics).

### Loop 2: Worth-Growth Reward

Current state: `rewardCreators(unitName, amount)` exists but is never called.

After each cycle (or every 10 cycles), scan non-heuristic units with creditors for worth growth since creation.

Mechanics:
- Add `creationWorth` slot to units, set when created by a heuristic.
- Add `lastRewardedWorth` slot to track the high-water mark for reward.
- Check: if `worth > lastRewardedWorth`, compute `delta = worth - lastRewardedWorth`, call `rewardCreators(unitName, delta / 2)`, update `lastRewardedWorth = worth`.
- Halving the delta keeps rewards moderate.
- Skip units without creditors (seed units). Skip heuristics (evaluated by applics, not worth growth).

Frequency: every 10 cycles. Worth changes accumulate from domain heuristics before rewarding.

Edge case: if worth drops below `lastRewardedWorth` and rises again, creators only get rewarded for growth above the previous high-water mark. No double-dipping.

**Future consideration:** Usage-based reward, where units that are actively referenced by other heuristics during their firing also trigger reward to their creators. This would capture "this unit is contributing to the system's reasoning" as a distinct positive signal. Deferred to a follow-up design.

### Loop 3: Meta-Heuristic H-AnalyzeApplics

A seed meta-heuristic that fires during normal engine operation, inspects other heuristics' applics records, detects patterns in where they succeed and fail, and emits targeted specializations.

When it fires (via unit-focus on a Heuristic unit), it reads that heuristic's applics rolling window. Each applics entry records the target unit name. It groups successes and failures by the target unit's `isA` types. If a clear skew emerges -- e.g., 8/10 successes on `Set` units and 5/6 failures on `Number` units -- it creates a specialized copy of the heuristic with an added `isA` check in `ifPotentiallyRelevant` restricting it to the successful type.

Structure:
```
name: H-AnalyzeApplics
isA: ["Heuristic"]
worth: 600
english: "Inspect a heuristic's applics for type-skewed success patterns and propose specializations"
```

- `ifPotentiallyRelevant`: target unit is a Heuristic with at least MinApplics firings
- `ifTrulyRelevant`: success ratio between 0.3 and 0.7 (below 0.3 = Loop 1 handles it, above 0.7 = performing well, middle = specialization opportunity)
- `thenCompute`: analyze applics by target type, detect skew, create specialized copy

Credit: H-AnalyzeApplics lists itself as creditor on any specialized heuristic it creates. If the specialization performs better, H-AnalyzeApplics gets rewarded via Loop 2. If the specialization dies, H-AnalyzeApplics gets punished.

Registered in both math and observations domains.

### Loop 4: HindSight Validation

Current state: `createAvoidanceRule` generates `HAvoid-X` heuristics when units die, but they're never validated or evaluated.

**Validation on creation:** When `createAvoidanceRule` generates a new HAvoid rule, dry-run parse the generated DSL program through the tokenizer. If it doesn't tokenize cleanly, discard it and log the failure.

**Effectiveness tracking:**
- When an HAvoid rule fires and aborts: record `trackApplics(havoid, target, true)`.
- HAvoid rules start at worth 300 (unproven).
- After 3+ successful firings without the system creating a bad unit of the same type: boost to 600 (proven useful).
- After 200 cycles with zero firings: demote to 100 (too narrow or failure pattern doesn't recur).
- HAvoid rules that reach worth 0 get killed like any other unit, triggering HindSight -- which could create an `HAvoid-HAvoid` rule. The system can learn that a particular avoidance strategy was itself bad.

The recursive case (HAvoid-HAvoid) is the key property: meta-meta-reasoning through the same mechanism as base-level reasoning.

## Code Changes

### New DSL Builtins (`internal/dsl/builtins.go`)

| Builtin | Stack effect | Purpose |
|---|---|---|
| `get-applics` | `(unitName -- applicsList)` | Push unit's applics rolling window |
| `applics-success-ratio` | `(unitName -- float)` | Compute successes / total from overallRecord |
| `applics-by-type` | `(unitName -- map)` | Group applics by target unit's isA, return `{type: {s: N, f: N}}` |
| `abort` | `( -- )` | Signal engine to stop processing this task (needs to be a DSL word so HAvoid rules can emit it) |

### Engine Changes (`internal/engine/`)

- `fire.go`: After ThenParts execute, detect no-op firings (snapshot store size + agenda size before/after; slot modifications alone don't count). Call `trackApplics(h, target, false)` on no-op.
- `credit.go`: In `HandleDeletedUnit`, add `trackApplics(creditor, deadUnit, false)` alongside `punishCreators`. Add `rewardForWorthGrowth()` method. Validate HAvoid DSL programs on creation via tokenizer dry-run.
- `mutation.go`: Replace time-based trigger with performance-based scan. Add `MinApplics` and `MutationThreshold` to `MutationConfig`. Remove random-weighted selection.
- `engine.go`: Call `rewardForWorthGrowth()` every 10 cycles. HAvoid demotion check every 50 cycles.

### New Seed Heuristic (`internal/seed/heuristics.go`)

- `H-AnalyzeApplics`: meta-heuristic for Loop 3.

### Unit Changes

No structural changes to `internal/unit/unit.go`. `creationWorth` and `lastRewardedWorth` are regular slots set via `Set()`.

## Test Plan

- trackApplics failure recording: no-op firing produces failure record, unit death traces back failure to creditor heuristics
- rewardCreators on worth growth: delta calculation correct, high-water mark prevents double-dipping, skips seed units and heuristics
- Performance-based mutation trigger: fires on low success ratio, skips when all heuristics adequate, picks worst performer
- H-AnalyzeApplics: detects type skew in applics, creates specialized copy with correct isA check, credits itself
- HAvoid validation: generated programs parse cleanly, HAvoid fires and records success, promotion after 3+ firings, demotion after 200 idle cycles
- End-to-end: heuristic creates bad unit -> unit dies -> creditor punished + failure recorded -> mutation triggered -> mutant replaces original
