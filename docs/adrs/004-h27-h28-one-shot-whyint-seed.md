# ADR-004: H27/H28/H25/H26 are one-shot + auto-seed whyInt on ≥4 members

**Date:** 2026-04-18
**Status:** Accepted
**Phase:** 5.11, 5.2
**Commits:** 1a3186d, d4772bd, 40f9872

## Context

When H25/H26/H27/H28 fire, they create a new category (`SatisfyingSetFor<pred>` / `FailingSetFor<pred>`) with examples derived from running the predicate. Two open questions:

1. **Re-firing**: EURISKO's H27/H28 have no explicit guard — re-firings are suppressed only by agenda/credit economics. Should we replicate that, or add an explicit one-shot?
2. **Downstream wiring**: Should the new category auto-schedule a `whyInt` task so H24 can then discover interesting predicates on it?

## Decision

- **One-shot** via `unit-exists?` dedupe on the output name (`SatisfyingSetFor-<pred>` etc.). If the unit already exists, the heuristic does nothing further.
- **Auto-seed whyInt** when the filtered set has ≥ 4 members. The threshold mirrors H24's minimum evidence bar — below it, H24 has too little to work with anyway.

## Alternatives considered

- **EURISKO-faithful re-firing**: let the heuristic fire on every focus, relying on worth/credit to suppress waste. Rejected because our credit system is less battle-tested than EURISKO's — a runaway duplicate creation could confuse the economy before it punishes the heuristic. One-shot is a safer ceiling.
- **No whyInt auto-seed**: let the next focus cycle naturally scan the new category. Works, but relies on worth-economics to bring the category into focus, and our agenda is often saturated. Explicit seeding guarantees the pipeline closes within the same run.
- **Seed whyInt regardless of member count**: simpler code, but fires H24 against under-evidenced categories that will never find a cross-matching predicate. Wasted cycles.

## Consequences

- The full pred → category → pred loop closes within a single run. Verified in a 300-cycle math run: `H28: created FailingSetFor-IsEmpty` at ~cycle 15, then `FailingSetFor-IsEmpty.whyInt` runs at cycle 221 as H24 scans it.
- One-shot dedupe means re-derived interestingness (a predicate that becomes *more* interesting later) doesn't re-trigger category creation. If this matters in practice we'd add a heuristic that bumps `supportCount` on the existing category instead.
- The 4-member threshold is hardcoded. Could have made it a per-heuristic slot (see ADR-008); chose not to because the value is meaningful only in relation to H24's internal threshold and tracking both would invite drift.
