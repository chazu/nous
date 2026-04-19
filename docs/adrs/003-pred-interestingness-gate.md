# ADR-003: Predicate interestingness gate = worth ≥ 600 OR isAInt OR rarity[0] < 0.3

**Date:** 2026-04-18
**Status:** Accepted
**Phase:** 5.11, 5.2
**Commits:** 1a3186d, d4772bd, 40f9872

## Context

H25/H26/H27/H28 all fire only on "interesting" predicates. EURISKO's gate is `HasHighWorth OR IsAInt`. We had to translate this into our slot system and decide whether to extend it.

Our existing machinery:
- `worth` slot on every unit (numeric)
- `isAInt` slot populated by H22/H23 when a unit's examples pass an interestingness predicate
- `rarity` tuple `[freqTrue, numT, numF]` populated by `apply-pred` on Pred units (Phase 5.10)

H24 already uses the rarity limb to discover rare predicates. If H25–H28's gate also consults rarity, H24's discoveries feed directly into H27/H28 as triggers on the next focus.

## Decision

Three-limb OR gate: `worth ≥ 600 OR isAInt != nil OR rarity[0] < 0.3`.

- **worth ≥ 600**: direct analog of `HasHighWorth`. Our default worth is 500, so 600 is "above average" — consistent with how other heuristics use the threshold.
- **isAInt != nil**: direct analog of `IsAInt`.
- **rarity[0] < 0.3**: a predicate that's almost-always-false (or has no true outcomes yet) is inherently interesting.

## Alternatives considered

- **Just `worth ≥ 600 OR isAInt`**: EURISKO-faithful but leaves H24's rarity discoveries with no downstream consumer. Rejected because we built H24 specifically to find rare predicates; letting H27/H28 see them closes a feedback loop.
- **Learnable gate**: some heuristic tunes the thresholds. Out of scope; our worth-economy tuning is already complex enough.
- **High-frequency (>0.7) as "interesting"**: symmetric-extreme case — an almost-always-true pred is surprising too. Not added; our current predicates are set-typed and mostly fail on non-set data, so the "true side" is never extreme. Reconsider if we add predicates that typically hold.

## Consequences

- The gate is truthy earlier and on more preds than EURISKO's, which risks more firings. Mitigated by one-shot dedupe on the output unit (ADR-004).
- `rarity[0]` is `nil` (untouched) until the pred is invoked. The guard `"rarity" get-slot nil != AND first 0.3 <` handles this. Without the guard, we'd compare `nil < 0.3` which would either error or spuriously match.
- Consistency across H25/H26/H27/H28 means one change to the gate shape will need to land in four places. Accepted cost — the alternative (extract a shared "interesting-pred?" builtin) was not warranted for a three-clause expression.
