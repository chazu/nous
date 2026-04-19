# ADR-006: Wake up H27/H28 via seed-worth bumps + H-ExercisePreds heuristic

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 5.11 follow-up, 5.2 follow-up
**Commits:** ff667fb, 40f9872

## Context

After landing H27/H28 (Phase 5.11), a 300-cycle math-domain run showed **zero firings**. The discovery-layer feature worked under tests but was inert in production.

Three causes, stacked:

1. **Seed predicates were too boring**: `IsEmpty`, `IsSingleton`, etc. started at worth 500 (default). The interestingness gate (ADR-003) requires worth ≥ 600.
2. **Rarity never populated**: `apply-pred` is the only path that bumps `rarity` counters, and nothing in the math-domain seed invoked unary preds directly. H-Conjecture uses the raw `set-equal?` / `set-subset?` builtins, not `apply-pred` on the Pred units.
3. **Preds never focused**: even if the gate passed, `fireTaskRule` only evaluates `ifPotentiallyRelevant` with `ArgU = task.UnitName`. No task ever targeted `IsEmpty`/`IsSingleton`/etc., so the gate was never checked against a Pred.

A faithful EURISKO simulation would have predicates accumulating interestingness through use. We don't have the density of use to make that work organically yet.

## Decision

Three coordinated changes:

1. **Bump seed-predicate worths to 700** for `IsEmpty`, `IsSingleton`, `SetEqual`, `SubsetOf`. Analog of EURISKO's hand-seeded "interesting" predicates.
2. **Add `H-ExercisePreds` heuristic** that runs every domain-matching unary predicate against a focused category's examples. `apply-pred` populates rarity as a side effect. One-shot per category via `predsExercised` flag. Extended to also handle binary preds (see point 3).
3. **`H-ExercisePreds` schedules a whyInt task on each exercised pred** (one-shot via `predFocusScheduled`). This guarantees the pred gets focused later, giving H27/H28 a task with `ArgU = pred` to run against.

## Alternatives considered

- **Just bump the worths** (skip points 2–3): worth alone passes the gate, but preds still never get focused in task-focus mode when the agenda is saturated. In practice H27/H28 still wouldn't fire in short runs.
- **Modify H-Conjecture to use `apply-pred`**: would populate rarity naturally through normal use. Rejected because the rarity side effect is incidental to H-Conjecture's job, and the DSL builtin version is measurably faster per firing.
- **Seed an initial whyInt task for each pred at startup** (engine-level in `SeedInitialAgenda`): works but buries the wiring in Go. The heuristic approach keeps the knowledge in CUE where other mutations can see and alter it.
- **Run binary preds Cartesian-style in H-ExercisePreds too**: originally tried, but N² per-category × N preds made TestSpecializationPipeline and TestH19CriterialSparesSpecializations starve for cycles. Reduced to *schedule-only* for binary preds — H25/H26 do their own Cartesian.

## Consequences

- 300-cycle run post-change: H27/H28 fire (on IsEmpty), H25/H26 fire twice each (on SetEqual, SubsetOf), 115 OPair units materialize, downstream whyInt tasks queue. Discovery layer is alive.
- The worth bumps change the agenda's focus-order in short runs. `TestSpecializationPipeline` and `TestH19CriterialSparesSpecializations` broke until they were updated to call `SeedInitialAgenda` (see ADR-009). Production was never affected.
- H-ExercisePreds is domain-agnostic (lives in `domains/common/`) but assumes predicates have a `domain` slot whose first element names a type whose examples have `data`. Fair for the math domain; will need revisiting for non-mathematical domains.
